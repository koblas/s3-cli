package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"net/url"
)

func remotePager(config *Config, svc *s3.S3, uri string, delim bool, pager func(page *s3.ListObjectsV2Output)) error {
    u, err := url.Parse(uri)
    if err != nil || u.Scheme != "s3" {
        return fmt.Errorf("requires buckets to be prefixed with s3://")
    }

    params := &s3.ListObjectsV2Input{
        Bucket:    aws.String(u.Host), // Required
        MaxKeys:   aws.Int64(1000),
    }
    if u.Path != "" && u.Path != "/" {
        params.Prefix = aws.String(u.Path[1:])
    }
    if delim {
        params.Delimiter = aws.String("/")
    }

    wrapper := func(page *s3.ListObjectsV2Output, lastPage bool) (bool) {
        pager(page)
        return true
    }

    if err := SessionForBucket(svc, u.Host).ListObjectsV2Pages(params, wrapper); err != nil {
        return err
    }
    return nil
}

func remoteList(config *Config, svc *s3.S3, args []string) ([]FileObject, error) {
    result := make([]FileObject, 0)

    for _, arg := range args {
        pager := func(page *s3.ListObjectsV2Output) {
            for _, obj := range page.Contents {
                result = append(result, FileObject{
                        Name: *obj.Key,
                        Size: *obj.Size,
                        Checksum: *obj.ETag,
                    })
            }
        }

        remotePager(config, svc, arg, false, pager)
    }

    return result, nil
}

/*
// Get the contents of a remote URL
//  This is useful for sync, rm and get operations
func remoteList(config *Config, svc *s3.S3, args []string) ([]FileObject, error) {
    result := make([]FileObject, 0)

    for _, arg := range args {
        u, err := url.Parse(arg)
        if err != nil || u.Scheme != "s3" {
            return nil, fmt.Errorf("ls requires buckets to be prefixed with s3://")
        }

        params := &s3.ListObjectsV2Input{
            Bucket:    aws.String(u.Host), // Required
            MaxKeys:   aws.Int64(1000),
        }
        if u.Path != "" && u.Path != "/" {
            params.Prefix = aws.String(u.Path[1:])
        }

        pager := func(page *s3.ListObjectsV2Output, lastPage bool) (bool) {
            for _, obj := range page.Contents {
                result = append(result, FileObject{
                        Name: *obj.Key,
                        Size: *obj.Size,
                        Checksum: *obj.ETag,
                    })
            }
            return true
        }

        bsvc := SessionForBucket(svc, u.Host)

        if err := bsvc.ListObjectsV2Pages(params, pager); err != nil {
            fmt.Println(err)
            continue
        }
    }

	return result, nil
}
*/
