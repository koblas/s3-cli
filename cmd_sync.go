package main

import (
	"io"
	"os"
	"fmt"
    "crypto/md5"

	// "path"
	"strings"
	"path/filepath"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
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
        ACT_COPY = iota
        ACT_REMOVE = iota
        ACT_CHECKSUM = iota
    )

	args := c.Args()

    dst, args := args[len(args)-1], args[:len(args)-1]

    dst_uri, err := FileURINew(dst)
    if err != nil {
        return fmt.Errorf("Invalid destination argument")
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
            return fmt.Errorf("Invalid destination argument")
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
    if !src_is_directory && len(srcs) == 1 && srcs[0].Scheme == "file" {
        if info, err := os.Stat(srcs[0].Path); err == nil {
            src_is_directory = info.IsDir()
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

    type Action struct {
        Type            int
        Src             *FileURI
        Dst             *FileURI
        Checksum        string
    }

    work_queue := make([]Action, 0)

    addWork := func (src *FileURI, src_info *FileObject, dst *FileURI, dst_info *FileObject) {
        if src_info == nil {
            work_queue = append(work_queue, Action{
                Type: ACT_REMOVE,
                Src: src,
                Dst: dst,
            })
        } else if dst_info == nil {
            work_queue = append(work_queue, Action{
                Type: ACT_COPY,
                Src: src,
                Dst: dst,
            })
        } else if src_info.Size != dst_info.Size {
            work_queue = append(work_queue, Action{
                Type: ACT_COPY,
                Src: src,
                Dst: dst,
            })
        } else if config.CheckMD5 {
            if src_info.Checksum != "" && dst_info.Checksum != "" && src_info.Checksum != dst_info.Checksum {
                work_queue = append(work_queue, Action{
                    Type: ACT_COPY,
                    Src: src,
                    Dst: dst,
                })
            } else {
                check := src_info.Checksum
                if check == "" {
                    check = dst_info.Checksum
                }
                work_queue = append(work_queue, Action{
                    Type: ACT_CHECKSUM,
                    Src: src,
                    Dst: dst,
                    Checksum: check,
                })
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
        dropLen := 0
        prefix := ""
        if !strings.HasSuffix(srcs[0].Path, "/") {
            prefix = filepath.Base(srcs[0].Path)
        } else {
            dropLen = len(filepath.Base(srcs[0].Path)) + 1
        }

        src_files, err := buildFileInfo(config, &srcs[0], dropLen)
        if err != nil {
            return err
        }

        dst_uri = dst_uri.Join(prefix)

        dst_files, err := buildFileInfo(config, dst_uri, dropLen)
        if err != nil {
            return err
        }

        // This loop will add COPIES
        for file, _ := range src_files {
            addWork(srcs[0].Join(file), src_files[file], dst_uri.Join(file), dst_files[file])
        }
        // This loop will add REMOVES from DST
        for file, _ := range dst_files {
            src_info := src_files[file]
            dst_info := dst_files[file]
            if src_info != nil && dst_info != nil {
                continue
            }
            addWork(srcs[0].Join(file), src_info, dst_uri.Join(file), dst_info)
        }
    }

    if config.Verbose {
        fmt.Printf("%d files to consider\n", len(work_queue))
    }

    // Now do the work...
    for _, item := range work_queue {
        switch item.Type {
        case ACT_COPY:
            copyFile(config, item.Src, item.Dst, true)
        case ACT_REMOVE:
            // S3 removes are handled in batch at the end
            if dst_uri.Scheme == "file" {
                if config.Verbose {
                    fmt.Printf("Remove %s\n", item.Dst.String())
                }
                if !config.DryRun {
                    os.Remove(item.Dst.Path)
                }
            }
        case ACT_CHECKSUM:
            // src_path := fmt.Sprintf("%s/%s", item.SourceURL.String(), item.Path)
            var hash string
            if dst_uri.Scheme == "s3" {
                hash, err = amazonEtagHash(item.Src.Path)
                if err != nil {
                    return fmt.Errorf("Unable to get checksum of local file %s", item.Src.String())
                }
            } else {
                fmt.Printf("CHECKSUM %s\n", item.Dst.String())
                hash, err = amazonEtagHash(item.Dst.Path)
                if err != nil {
                    return fmt.Errorf("Unable to get checksum of local file %s", item.Dst.String())
                }
            }

            fmt.Printf("Got checksum %s local=%s remote=%s\n", item.Src.String(), hash, item.Checksum)
            if hash != strings.Trim(item.Checksum, "\"") {
                copyFile(config, item.Src, item.Dst, true)
            }
        }
    }

    // If the destination is S3, then lets do batch removes
    if dst_uri.Scheme == "s3" {
        bsvc := SessionForBucket(SessionNew(config), dst_uri.Bucket)
        objects := make([]*s3.ObjectIdentifier, 0)

        // Helper to remove the actual objects
        doDelete := func() error {
            if len(objects) == 0 {
                return nil
            }
            if !config.DryRun {
                params := &s3.DeleteObjectsInput{
                    Bucket: aws.String(dst_uri.Bucket), // Required
                    Delete: &s3.Delete{ // Required
                        Objects: objects,
                    },
                }

                if _, err := bsvc.DeleteObjects(params); err != nil {
                    return err
                }

            }
            objects = make([]*s3.ObjectIdentifier, 0)
            return nil
        }

        for _, item := range work_queue {
            if item.Type != ACT_REMOVE {
                continue
            }
            if config.Verbose {
                fmt.Printf("Remove %s\n", item.Dst.String())
            }
            objects = append(objects, &s3.ObjectIdentifier{ Key: item.Dst.Key() })
            if len(objects) == 500 {
                if err := doDelete(); err != nil {
                    return err
                }
            }
        }
        if err := doDelete(); err != nil {
            return err
        }
    }

    return nil
}

//  Walk either S3 or the local file system gathering files
//    files_only == true -- only consider file names, not directories
func buildFileInfo(config *Config, src *FileURI, dropPrefix int) (map[string]*FileObject, error) {
    files := make(map[string]*FileObject, 0)

    if src.Scheme == "s3" {
        objs, err := remoteList(config, nil, []string{src.String()})
        if err != nil {
            return files, err
        }
        dropPrefix = len(src.Path) - 1
        for idx, obj := range objs {
            name := obj.Name[dropPrefix:]
            files[name] = &objs[idx]
            // fmt.Println("s3 -- ", name, files[name])
        }
    } else {
        err := filepath.Walk(src.Path, func (path string, info os.FileInfo, _ error) error {
            if info == nil || info.IsDir() {
                return nil
            }

            name := path[dropPrefix:]
            files[name] = &FileObject{
                Name: path,
                Size: info.Size(),
            }
            // fmt.Println("local -- ", name, files[name])

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
            bsvc := SessionForBucket(SessionNew(config), src.Bucket)
            params := &s3.HeadObjectInput {
                Bucket: aws.String(src.Bucket),
                Key:    src.Key(),
            }
            response, err := bsvc.HeadObject(params)
            if err != nil {
                continue
            }
            result[*src] = &FileObject{
                Name: src.Path,
                Size: *response.ContentLength,
                Checksum: *response.ETag,
            }
        }
    }

    return result
}

// Compute the Amazon ETag hash for a given file
func amazonEtagHash(path string) (string, error) {
    const   BLOCK_SIZE = 1024 * 1024 * 5        // 5MB
    const   START_BLOCKS = 1024 * 1024 * 16     // 16MB

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
