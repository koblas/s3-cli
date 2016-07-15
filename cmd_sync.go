package main

import (
	"os"
	"fmt"
	// "path"
	// "strings"
	"path/filepath"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
)


// One command to sync files/directories -- it's always recursive
// 
// Useful options
//    --verbose
//    --dry-run
//    --no-check-md5
//
func CmdSync(config *Config, c *cli.Context) error {
    const (
        ACT_COPY = iota
        ACT_REMOVE = iota
        ACT_CHECKSUM = iota
    )

	args := c.Args()

    dst, args := args[len(args)-1], args[:len(args)-1]

    dst_u, err := FileURINew(dst)
    if err != nil {
        return fmt.Errorf("Invalid destination argument")
    }
    if dst_u.Scheme == "" {
        dst_u.Scheme = "file"
    }
    if dst_u.Path == "" {
        dst_u.Path = "/"
    }

    src_map := make(map[FileURI]map[string]FileObject, 0)
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

    for _, u := range srcs {
        src_files, err := buildFileInfo(config, &u)
        if err != nil {
            return err
        }
        src_map[u] = src_files
    }

    dst_files, err := buildFileInfo(config, dst_u)
    if err != nil {
        return err
    }

    //  Here's a list of all files -- 
    //   0x01 == only in src
    //   0x02 == only in dst
    //   0x03 == in both
    all_files := make(map[string]int, 0)

    for _, files := range src_map {
        for _, f := range files {
            all_files[f.Name] = 0x01
        }
    }
    for _, f := range dst_files {
        all_files[f.Name] |= 0x02
    }

    if config.Verbose {
        fmt.Printf("%d files to consider\n", len(all_files))
    }

    findFile := func(path string) (*FileURI, *FileObject, bool) {
        var lastV *FileObject
        var lastU *FileURI
        for _, u := range srcs {
            if v, exists := src_map[u][path]; exists {
                lastV = &v
                lastU = &u
            }
        }
        return lastU, lastV, lastV != nil
    }

    //
    //  Construct to work to be done
    //
    type Action struct {
        Type        int
        SourceURL   *FileURI
        Path        string
    }

    work_queue := make([]Action, 0)

    for file, op := range all_files {
        switch op {
        case 0x01:
            src, _, _ := findFile(file)
            work_queue = append(work_queue, Action{
                Type: ACT_COPY,
                SourceURL: src,
                Path: file,
            })
        case 0x02:
            work_queue = append(work_queue, Action{
                Type: ACT_REMOVE,
                Path: file,
            })
        case 0x03:
            src, src_info, exists := findFile(file)
            if !exists {
                return fmt.Errorf("Unable to find src file")
            }
            dst_info, exists := dst_files[file]
            if !exists {
                return fmt.Errorf("Unable to find dst file")
            }

            if src_info.Size != dst_info.Size {
                work_queue = append(work_queue, Action{
                    Type: ACT_COPY,
                    SourceURL: src,
                    Path: file,
                })
            } else if config.CheckMD5 {
                if src_info.Checksum != "" && dst_info.Checksum != "" {
                    if src_info.Checksum != dst_info.Checksum {
                        work_queue = append(work_queue, Action{
                            Type: ACT_COPY,
                            SourceURL: src,
                            Path: file,
                        })
                    }
                } else {
                    work_queue = append(work_queue, Action{
                        Type: ACT_CHECKSUM,
                        SourceURL: src,
                        Path: file,
                    })
                }
            }
        }
    }

    // Build the Channels and GoRoutines
    
    // Remove from S3

    // Now do the work...
    for _, item := range work_queue {
        dst_path := fmt.Sprintf("%s/%s", dst_u.String(), item.Path)

        switch item.Type {
        case ACT_COPY:
            src_path := fmt.Sprintf("%s/%s", item.SourceURL.String(), item.Path)
            fmt.Printf("COPY %s -> %s\n", src_path, dst_path)
        case ACT_REMOVE:
            // S3 removes are handled in batch at the end
            if dst_u.Scheme == "file" {
                if config.Verbose {
                    fmt.Printf("Remove %s\n", item.Path)
                }
                if !config.DryRun {
                    os.Remove(item.Path)
                }
            }
        case ACT_CHECKSUM:
            src_path := fmt.Sprintf("%s/%s", item.SourceURL.String(), item.Path)
            if dst_u.Scheme == "s3" {
                fmt.Printf("CHECKSUM %s\n", src_path)
            } else {
                fmt.Printf("CHECKSUM %s\n", dst_path)
            }
        }
    }

    // If the destination is S3, then lets do batch removes
    if dst_u.Scheme == "s3" {
        bsvc := SessionForBucket(SessionNew(config), dst_u.Bucket)
        objects := make([]*s3.ObjectIdentifier, 0)

        // Helper to remove the actual objects
        doDelete := func() error {
            if len(objects) == 0 {
                return nil
            }
            if !config.DryRun {
                params := &s3.DeleteObjectsInput{
                    Bucket: aws.String(dst_u.Bucket), // Required
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
            if config.Verbose {
                fmt.Printf("Remove s3://%s/%s\n", dst_u.Bucket, item.Path)
            }
            objects = append(objects, &s3.ObjectIdentifier{ Key: aws.String(item.Path) })
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

// 
func buildFileInfo(config *Config, src *FileURI) (map[string]FileObject, error) {
    files := make(map[string]FileObject, 0)

    if src.Scheme == "s3" {
        svc := SessionNew(config)
        objs, err := remoteList(config, svc, []string{src.String()})
        if err != nil {
            return files, err
        }
        for _, obj := range objs {
            files[obj.Name] = obj
        }
    } else {
        err := filepath.Walk(src.Path, func (path string, info os.FileInfo, _ error) error {
            if info == nil || info.IsDir() {
                return nil
            }

            files[path] = FileObject{
                Name: path[len(src.Path):],
                Size: info.Size(),
                Checksum: "",
            }

            return nil
        })

        if err != nil {
            return files, err
        }
    }
    return files, nil
}

// Channel helpers
