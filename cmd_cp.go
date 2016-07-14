package main

import (
	"os"
	"fmt"
	"path"
	"strings"
	"path/filepath"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/urfave/cli"
	"net/url"
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

    dst_u, err := url.Parse(dst)
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
        u, err := url.Parse(path)
        if err != nil {
            return fmt.Errorf("Invalid destination argument")
        }
        if u.Scheme == "" {
            u.Scheme = "file"
        }
        if err := copyTo(config, u, dst_u); err != nil {
            return err
        }
    }

    return nil
}

func copyTo(config *Config, src, dst *url.URL) error {
	// svc := SessionNew(config)

    if src.Scheme != "file" && src.Scheme != "s3" {
        return fmt.Errorf("cp only supports local and s3 URLs")
    }
    if dst.Scheme != "file" && dst.Scheme != "s3" {
        return fmt.Errorf("cp only supports local and s3 URLs")
    }

    doCopy := func(src, dst *url.URL) error {
        if config.Verbose {
            fmt.Printf("Copy %s -> %s\n", src.String(), dst.String())
        }
        if config.DryRun {
            return nil
        }

        switch src.Scheme + "->" + dst.Scheme {
        case "file->file":
            return fmt.Errorf("cp should not be doing local files")
        case "s3->s3":
            return copyOnS3(config, src, dst)
        case "s3->file":
            return copyToLocal(config, src, dst)
        case "file->s3":
            return copyToS3(config, src, dst)
        }
        return nil
    }

    if config.Recursive {
        if src.Scheme == "s3" {
            svc := SessionForBucket(SessionNew(config), src.Host)
            basePath := src.Path[1:]

            remotePager(config, svc, src.String(), false, func(page *s3.ListObjectsV2Output) {
                for _, obj := range page.Contents {
                    // doCopy(src, dst)
                    src_path := *obj.Key
                    src_path = src_path[len(basePath):]

                    // uri := fmt.Sprintf("/%s", src.Host, *obj.Key)
                    dst_path := dst.String()
                    if strings.HasSuffix(dst.String(), "/") {
                        dst_path += src_path
                    } else {
                        dst_path += "/" + src_path
                    }

                    dst_uri, _ := url.Parse(dst_path)
                    dst_uri.Scheme = dst.Scheme
                    src_uri, _ := url.Parse("s3://" + src.Host + "/" + *obj.Key)

                    // fmt.Printf("Copy %s -> %s\n", src_uri.String(), dst_uri.String())
                    doCopy(src_uri, dst_uri)
                }
            })
        } else {
            // TODO: get Local file list
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
                dst_uri, _ := url.Parse(dst_path)
                dst_uri.Scheme = dst.Scheme
                src_uri, _ := url.Parse("file://" + path)

                // fmt.Printf("Copy %s -> %s\n", src_uri.String(), dst_uri.String())
                doCopy(src_uri, dst_uri)
                return nil
            })
            if err != nil {
                return err
            }
        }
    } else {
        doCopy(src, dst)
    }

    return nil
}

// Copy from S3 to local file
func copyToLocal(config *Config, src, dst *url.URL) error {
    svc := SessionForBucket(SessionNew(config), src.Host)
    downloader := s3manager.NewDownloaderWithClient(svc)

    params := &s3.GetObjectInput{
        Bucket: aws.String(src.Host),
        Key:    aws.String(src.Path[1:]),
    }

    dst_path := dst.Path

    // if the destination is a directory then copy to a file in the directory
    sinfo, err := os.Stat(dst_path)
    if err == nil && sinfo.IsDir() {
        dst_path = path.Join(dst_path, filepath.Base(src.Path))
    }

    fd, err := os.Create(dst_path)
    if err != nil {
        fmt.Println(err)
        return err
    }
    defer fd.Close()

    _, err = downloader.Download(fd, params)
    if err != nil {
        return err
    }

    return nil
}

// Copy from local file to S3
func copyToS3(config *Config, src, dst *url.URL) error {
    svc := SessionForBucket(SessionNew(config), dst.Host)
    uploader := s3manager.NewUploaderWithClient(svc)

    fd, err := os.Open(src.Path)
    if err != nil {
        return err
    }
    defer fd.Close()

    params := &s3manager.UploadInput{
        Bucket:     aws.String(dst.Host), // Required
        Key:        cleanBucketDestPath(src.Path, dst.Path),
        Body:       fd,
    }

    _, err = uploader.Upload(params)
    if err != nil {
        return err
    }

    return nil
}

// Copy from S3 to S3
//  -- if src and dst are the same it effects a "touch"
func copyOnS3(config *Config, src, dst *url.URL) error {
    svc := SessionForBucket(SessionNew(config), dst.Host)

    if strings.HasSuffix(src.Path, "/") {
        return fmt.Errorf("Invalid source for bucket to bucket copy path ends in '/'")
    }

    params := &s3.CopyObjectInput{
        Bucket:         aws.String(dst.Host),
        CopySource:     aws.String(fmt.Sprintf("/%s/%s", src.Host, src.Path[1:])),
        Key:            cleanBucketDestPath(src.Path, dst.Path),
    }

    // if this is an overwrite - note that
    if src.Host == dst.Host && *params.CopySource == fmt.Sprintf("/%s/%s", dst.Host, *params.Key) {
        params.MetadataDirective = aws.String("REPLACE")
    }

    _, err := svc.CopyObject(params)
    if err != nil {
        return err
    }

    return nil
}

// Take a src and dst and make a valid destination path for the bucket
//  if the dst ends in "/" add the basename of the source to the object
//  make sure the leading "/" is stripped off
func cleanBucketDestPath(src, dst string) *string {
    if strings.HasSuffix(dst, "/") {
        dst += filepath.Base(src)
    }
    if strings.HasPrefix(dst, "/") {
        dst = dst[1:]
    }
    return &dst
}
