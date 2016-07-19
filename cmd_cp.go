package main

import (
	"os"
	"fmt"
	"strings"
	"path/filepath"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
)

// One command to do it all, since get/put/cp should be able to copy from anywhere to anywhere
//  using standard "cp" command semantics
//
func CmdCopy(config *Config, c *cli.Context) error {
	args := c.Args()

    if len(args) < 2 {
        return fmt.Errorf("Not enought arguments to the copy command")
    }

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

    for _, path := range args {
        u, err := FileURINew(path)
        if err != nil {
            return fmt.Errorf("Invalid destination argument")
        }
        if u.Scheme == "" {
            u.Scheme = "file"
        }
        if err := copyCore(config, u, dst_u); err != nil {
            return err
        }
    }

    return nil
}

// Ok, this probably could just be in CopyCmd()
func copyCore(config *Config, src, dst *FileURI) error {
	// svc := SessionNew(config)

    if src.Scheme != "file" && src.Scheme != "s3" {
        return fmt.Errorf("cp only supports local and s3 URLs")
    }
    if dst.Scheme != "file" && dst.Scheme != "s3" {
        return fmt.Errorf("cp only supports local and s3 URLs")
    }

    if config.Recursive {
        if src.Scheme == "s3" {
            // Get the remote file list and start copying
            svc, err := SessionForBucket(SessionNew(config), src.Bucket)
            if err != nil {
                return err
            }

            basePath := src.Path[1:]

            remotePager(config, svc, src.String(), false, func(page *s3.ListObjectsV2Output) {
                for _, obj := range page.Contents {
                    src_path := *obj.Key
                    src_path = src_path[len(basePath):]

                    // uri := fmt.Sprintf("/%s", src.Host, *obj.Key)
                    dst_path := dst.String()
                    if strings.HasSuffix(dst.String(), "/") {
                        dst_path += src_path
                    } else {
                        dst_path += "/" + src_path
                    }

                    dst_uri, _ := FileURINew(dst_path)
                    dst_uri.Scheme = dst.Scheme
                    src_uri, _ := FileURINew("s3://" + src.Bucket + "/" + *obj.Key)

                    copyFile(config, src_uri, dst_uri, true)
                }
            })
        } else {
            // Get the local file list and start copying
            err := filepath.Walk(src.Path, func (path string, info os.FileInfo, _ error) error {
                if info == nil || info.IsDir() {
                    return nil
                }

                dst_path := dst.String()
                if strings.HasSuffix(dst.String(), "/") {
                    dst_path += path
                } else {
                    dst_path += "/" + path
                }
                dst_uri, _ := FileURINew(dst_path)
                dst_uri.Scheme = dst.Scheme
                src_uri, _ := FileURINew("file://" + path)

                return copyFile(config, src_uri, dst_uri, true)
            })
            if err != nil {
                return err
            }
        }
    } else {
        return copyFile(config, src, dst, false)
    }
    return nil
}
