package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
	"path/filepath"
	"strings"
)

// Action for syncing
type Action struct {
	Type     int
	Src      *FileURI
	Dst      *FileURI
	Size     int64
	Checksum string
}

const (
	NUM_COPY     = 4
	NUM_CHECKSUM = 1
	QUEUE_SIZE   = 1000000 // 1M
)

// One command to sync files/directories -- it's always recursive when directories are present
//
// Useful options
//    --verbose
//    --dry-run
//    --no-check-md5
//
// Notes:  Sync is one of three forms
//
//    file -> file
//    file(s) -> directory
//    directory -> directory
//
//  -- If src ends in a '/' then a directory isn't created on the destination
//       s3-cli sync foo/bar s3://bucket/data/  -- yields s3://bucket/data/bar/...
//       s3-cli sync foo/bar/ s3://bucket/data/  -- yields s3://bucket/data/...
//
func CmdSync(config *Config, c *cli.Context) error {
	const (
		ACT_COPY     = iota
		ACT_REMOVE   = iota
		ACT_CHECKSUM = iota
	)

	args := c.Args()
	if len(args) < 2 {
		return fmt.Errorf("Not enough arguments")
	}

	dst, args := args[len(args)-1], args[:len(args)-1]

	dst_uri, err := FileURINew(dst)
	if err != nil {
		return fmt.Errorf("Invalid destination argument %s", dst)
	}
	if dst_uri.Scheme == "" {
		dst_uri.Scheme = "file"
	}
	if dst_uri.Path == "" {
		dst_uri.Path = "/"
	}

	// src_map := make(map[FileURI]map[string]FileObject, 0)
	srcs := make([]FileURI, 0)

	for _, path := range args {
		u, err := FileURINew(path)
		if err != nil {
			return fmt.Errorf("Invalid source argument %s", path)
		}
		if u.Scheme == "" {
			u.Scheme = "file"
		}
		srcs = append(srcs, *u)
	}

	// Not sure this is 100% right way to do this, but couldn't come up with a better idea
	fileCount := 0
	dirCount := 0
	for _, u := range srcs {
		if u.Path[len(u.Path)-1] != '/' {
			fileCount += 1
		} else {
			dirCount += 1
		}
	}
	if fileCount != 0 && dirCount != 0 {
		return fmt.Errorf("Can't mix files and directories in sources")
	}

	// Handle the inputs
	src_is_directory := len(srcs) == 1 && strings.HasSuffix(srcs[0].Path, "/")
	if !src_is_directory && len(srcs) == 1 {
		src := srcs[0]
		if src.Scheme == "file" {
			if info, err := os.Stat(src.Path); err == nil {
				src_is_directory = info.IsDir()
			}
		} else {
			bsvc, err := SessionForBucket(SessionNew(config), src.Bucket)
			if err != nil {
				return err
			}
			params := &s3.HeadObjectInput{
				Bucket: aws.String(src.Bucket),
				Key:    src.Key(),
			}
			if _, err := bsvc.HeadObject(params); err != nil {
				src_is_directory = true
			}
		}
	}

	// Figure out what we're coping too if it's a file->file copy or to a directory
	dst_is_directory := src_is_directory || len(srcs) > 1 || strings.HasSuffix(dst_uri.Path, "/")
	if !dst_is_directory && dst_uri.Scheme == "file" {
		if info, err := os.Stat(dst_uri.Path); err == nil {
			dst_is_directory = info.IsDir()
		}
	}
	// Fix the output path if it is a directory
	if dst_is_directory && !strings.HasSuffix(dst_uri.Path, "/") {
		dst_uri.Path += "/"
	}

	///==================
	// Note: General improvement here that's pending is to make this a channel based system
	//       where we're dispatching commands at goroutines channels to get acted on.
	var (
		estimated_bytes int64
		file_count      int64
		wg              sync.WaitGroup
	)

	chanCopy := make(chan Action, QUEUE_SIZE)
	chanChecksum := make(chan Action, QUEUE_SIZE)
	chanRemove := make(chan Action, QUEUE_SIZE)
	chanProgress := make(chan int64)

	go workerProgress(chanProgress)

	wg.Add(1)
	go workerRemove(config, &wg, chanRemove, chanProgress)

	wg.Add(NUM_CHECKSUM)
	for i := 0; i < NUM_CHECKSUM; i++ {
		go workerChecksum(config, &wg, chanChecksum, chanProgress)
	}

	wg.Add(NUM_COPY)
	for i := 0; i < NUM_COPY; i++ {
		go workerCopy(config, &wg, chanCopy, chanProgress)
	}

	addWork := func(src *FileURI, src_info *FileObject, dst *FileURI, dst_info *FileObject) {
		/*
		   if src == nil {
		       fmt.Println("NIL", " -> ", dst.String())
		   } else {
		       fmt.Println(src.String(), " -> ", dst.String())
		   }
		*/
		file_count += 1

		if src_info == nil {
			chanRemove <- Action{
				Type: ACT_REMOVE,
				Src:  src,
				Dst:  dst,
			}
		} else if dst_info == nil {
			chanCopy <- Action{
				Type: ACT_COPY,
				Src:  src,
				Dst:  dst,
				Size: src_info.Size,
			}
			estimated_bytes += src_info.Size
			chanProgress <- src_info.Size
		} else if src_info.Size != dst_info.Size {
			chanCopy <- Action{
				Type: ACT_COPY,
				Src:  src,
				Dst:  dst,
				Size: src_info.Size,
			}
			estimated_bytes += src_info.Size
			chanProgress <- src_info.Size
		} else if config.CheckMD5 {
			if src_info.Checksum != "" && dst_info.Checksum != "" && src_info.Checksum != dst_info.Checksum {
				chanCopy <- Action{
					Type: ACT_COPY,
					Src:  src,
					Dst:  dst,
					Size: src_info.Size,
				}
				estimated_bytes += src_info.Size
				chanProgress <- src_info.Size
			} else {
				check := src_info.Checksum
				if check == "" {
					check = dst_info.Checksum
				}
				chanChecksum <- Action{
					Type:     ACT_CHECKSUM,
					Src:      src,
					Dst:      dst,
					Checksum: check,
					Size:     src_info.Size,
				}
				estimated_bytes += src_info.Size
			}
		}
	}

	///==================

	if !src_is_directory {
		// file -> file  (potential rename, etc.)
		// file(s) -> directory

		uri_list := make([]*FileURI, 0)
		dst_list := make([]*FileURI, 0)

		if dst_is_directory {
			// Destination is a directory, create real names for the results
			for idx, src := range srcs {
				d := dst_uri.Join(filepath.Base(src.Path))
				uri_list = append(uri_list, &srcs[idx], d)
				dst_list = append(dst_list, d)
			}
		} else {
			uri_list = append(uri_list, &srcs[0], dst_uri)
			dst_list = append(dst_list, dst_uri)
		}

		finfo := getFileInfo(config, uri_list)

		for idx := range srcs {
			src_info, exists := finfo[srcs[idx]]
			if !exists {
				return fmt.Errorf("Unable to stat the source file %s", srcs[idx].String())
			}
			dst_info, _ := finfo[*dst_list[idx]]
			// fmt.Println(*dst_list[idx], dst_info)

			addWork(&srcs[idx], src_info, dst_list[idx], dst_info)
		}
	} else {
		// directory -> directory
		// If the path doesn't end in a "/" then we prefix the resulting paths with it

		dropLen := len(srcs[0].Path)
		prefix := dst_uri.Path
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		if !strings.HasSuffix(srcs[0].Path, "/") {
			prefix += filepath.Base(srcs[0].Path) + "/"
			dropLen += 1
		}

		// fmt.Println("DROP len=", dropLen, " prefix=", prefix, " dst_uri=", dst_uri.String())

		src_files, err := buildFileInfo(config, &srcs[0], dropLen, prefix)
		if err != nil {
			return err
		}

		dst_files, err := buildFileInfo(config, dst_uri, 0, "")
		if err != nil {
			return err
		}

		// This loop will add COPIES
		for file, _ := range src_files {
			// fmt.Println(" FILE = ", file)
			src_info := src_files[file]
			addWork(srcs[0].SetPath(src_info.Name), src_info, dst_uri.SetPath(file), dst_files[file])
		}
		// This loop will add REMOVES from DST
		for file, _ := range dst_files {
			// fmt.Println("Remove Check", file)
			if src_info := src_files[file]; src_info == nil {
				addWork(nil, nil, dst_uri.Join(file), dst_files[file])
			}
		}
	}

	if config.Verbose {
		fmt.Printf("%d files to consider - %d bytes\n", file_count, estimated_bytes)
	}

	close(chanCopy)
	close(chanChecksum)
	close(chanRemove)
	wg.Wait()

	chanProgress <- 0
	close(chanProgress)
	os.Stdout.Write([]byte{'\n'})

	return nil
}

//  Walk either S3 or the local file system gathering files
//    files_only == true -- only consider file names, not directories
//
//  dropPrefix -- number of characters to remove from the front of the filename
//
func buildFileInfo(config *Config, src *FileURI, dropPrefix int, addPrefix string) (map[string]*FileObject, error) {
	files := make(map[string]*FileObject, 0)

	if src.Scheme == "s3" {
		slen := len(*src.Key()) - 1
		if slen > 0 && (*src.Key())[slen] != '/' {
			slen += 1
		}
		objs, err := remoteList(config, nil, []string{src.String()})
		if err != nil {
			return files, err
		}
		// dropPrefix -= 1 // no leading '/'
		for idx, obj := range objs {
			if slen > 0 && len(obj.Name) > slen && obj.Name[slen] != '/' {
				// fmt.Printf("SKIP: %s %d %c %s\n", obj.Name, slen, obj.Name[slen], *src.Key())
				continue
			}
			name := addPrefix + obj.Name[dropPrefix:]
			files[name] = &objs[idx]
			// fmt.Println("s3 -- name=", name, " path=", obj.Name, " file=", files[name])
		}
	} else {
		// dropPrefix = len(src.Path)
		err := filepath.Walk(src.Path, func(path string, info os.FileInfo, _ error) error {
			if info == nil || info.IsDir() {
				return nil
			}

			name := addPrefix + path[dropPrefix:]
			files[name] = &FileObject{
				Name: path,
				Size: info.Size(),
			}
			// fmt.Println("local -- name=", name, " path=", path, " file=", files[name])

			return nil
		})

		if err != nil {
			return files, err
		}
	}
	return files, nil
}

// Get the file info for a simple list of files this is used in the
//    file -> file
//    file(s) -> directory
//  cases, since there is little reason to go walk huge directories trees to get information
func getFileInfo(config *Config, srcs []*FileURI) map[FileURI]*FileObject {
	result := make(map[FileURI]*FileObject)

	for _, src := range srcs {
		if src.Scheme == "file" {
			info, err := os.Stat(src.Path)
			if err != nil {
				continue
			}
			result[*src] = &FileObject{
				Name: src.Path,
				Size: info.Size(),
			}
		} else {
			bsvc, err := SessionForBucket(SessionNew(config), src.Bucket)
			if err != nil {
				continue
			}

			params := &s3.HeadObjectInput{
				Bucket: aws.String(src.Bucket),
				Key:    src.Key(),
			}
			response, err := bsvc.HeadObject(params)
			if err != nil {
				continue
			}
			result[*src] = &FileObject{
				Name:     src.Path,
				Size:     *response.ContentLength,
				Checksum: *response.ETag,
			}
		}
	}

	return result
}

// Compute the Amazon ETag hash for a given file
func amazonEtagHash(path string) (string, error) {
	const BLOCK_SIZE = 1024 * 1024 * 5    // 5MB
	const START_BLOCKS = 1024 * 1024 * 16 // 16MB

	if strings.HasPrefix(path, "file://") {
		path = path[7:]
	}
	fd, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	info, err := fd.Stat()
	if err != nil {
		return "", err
	}

	hasher := md5.New()
	count := 0

	if info.Size() > START_BLOCKS {
		for err != io.EOF {
			count += 1
			parthasher := md5.New()
			var size int64
			size, err = io.CopyN(parthasher, fd, BLOCK_SIZE)
			if err != nil && err != io.EOF {
				return "", err
			}
			if size != 0 {
				hasher.Write(parthasher.Sum(nil))
			}
		}
	} else {
		if _, err := io.Copy(hasher, fd); err != nil {
			return "", err
		}
	}

	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	if count != 0 {
		hash += fmt.Sprintf("-%d", count)
	}
	return hash, nil
}

//  GoRoutine workers -- copy from src to dst
func workerCopy(config *Config, wg *sync.WaitGroup, jobs <-chan Action, progress chan int64) {
	for item := range jobs {
		copyFile(config, item.Src, item.Dst, true)
		progress <- -item.Size
	}
	wg.Done()
}

//  GoRoutine workers -- remove file
func workerRemove(config *Config, wg *sync.WaitGroup, jobs <-chan Action, progress chan int64) {
	objects := make([]*s3.ObjectIdentifier, 0)

	// Helper to remove the actual objects
	doDelete := func(last *FileURI) error {
		bsvc, err := SessionForBucket(SessionNew(config), last.Bucket)
		if err != nil {
			return err
		}

		params := &s3.DeleteObjectsInput{
			Bucket: aws.String(last.Bucket), // Required
			Delete: &s3.Delete{ // Required
				Objects: objects,
			},
		}

		if _, err := bsvc.DeleteObjects(params); err != nil {
			return err
		}

		objects = make([]*s3.ObjectIdentifier, 0)
		return nil
	}

	var last *FileURI

	for item := range jobs {
		if config.Verbose {
			fmt.Printf("Remove %s\n", item.Dst.String())
		}
		if config.DryRun {
			continue
		}
		last = item.Dst

		if item.Dst.Scheme == "file" {
			if err := os.Remove(item.Dst.Path); err != nil {
				// return err
			}
		} else {
			objects = append(objects, &s3.ObjectIdentifier{Key: item.Dst.Key()})
			if len(objects) == 500 {
				if err := doDelete(last); err != nil {
					// return err
				}
			}
		}
	}

	if len(objects) != 0 {
		if err := doDelete(last); err != nil {
			// return err
		}
	}
	wg.Done()
}

//  GoRoutine workers -- check checksum and copy if needed
func workerChecksum(config *Config, wg *sync.WaitGroup, jobs <-chan Action, progress chan int64) {
	for item := range jobs {
		var (
			hash string
			err  error
		)

		if item.Dst.Scheme == "s3" {
			hash, err = amazonEtagHash(item.Src.Path)
			if err != nil {
				// return fmt.Errorf("Unable to get checksum of local file %s", item.Src.String())
				fmt.Printf("Unable to get checksum of local file %s\n", item.Src.String())
			}
		} else {
			hash, err = amazonEtagHash(item.Dst.Path)
			if err != nil {
				// return fmt.Errorf("Unable to get checksum of local file %s", item.Dst.String())
				fmt.Printf("Unable to get checksum of local file %s\n", item.Src.String())
			}
		}

		// fmt.Printf("Got checksum %s local=%s remote=%s\n", item.Src.String(), hash, item.Checksum)
		if len(item.Checksum) <= 2 || hash != item.Checksum[1:len(item.Checksum)-1] {
			progress <- item.Size
			copyFile(config, item.Src, item.Dst, true)
			progress <- -item.Size
		}
	}
	wg.Done()
}

//  output the progress to the user
func humanize(value int64) string {
	const base = 1024.0
	sizes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}

	logn := func(n, b float64) float64 {
		return math.Log(n) / math.Log(b)
	}

	if value < 10 {
		return fmt.Sprintf("%d %s", value, sizes[0])
	}
	e := math.Floor(logn(float64(value), base))
	val := math.Floor(float64(value)/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f %s"
	if val < 10 {
		f = "%.1f %s"
	}
	return fmt.Sprintf(f, val, sizes[int(e)])
}

func workerProgress(updates <-chan int64) {
	tstart := time.Now()
	var (
		lastStr               string
		totalBytes, sentBytes int64
	)

	for update := range updates {
		if update > 0 {
			totalBytes += update
		} else {
			sentBytes += -update
		}

		if totalBytes == 0 {
			continue
		}

		str := fmt.Sprintf("%s / %s (%2.1f%%)   %s/sec",
			humanize(sentBytes), humanize(totalBytes),
			100.0*float64(sentBytes)/float64(totalBytes),
			humanize(int64(float64(sentBytes)/time.Since(tstart).Seconds())))

		if str == lastStr {
			continue
		}

		for i := 0; i < len(lastStr); i++ {
			os.Stdout.Write([]byte{'\010'})
		}

		os.Stdout.Write([]byte(str))
		for i := len(str); i < len(lastStr); i++ {
			os.Stdout.Write([]byte{' '})
		}
		for i := len(str); i < len(lastStr); i++ {
			os.Stdout.Write([]byte{'\010'})
		}
		lastStr = str

		os.Stdout.Sync()
		// fmt.Println(str)
	}
}
