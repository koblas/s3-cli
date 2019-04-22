package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func remotePager(config *Config, svc *s3.S3, uri string, delim bool, pager func(page *s3.ListObjectsV2Output)) error {
	u, err := FileURINew(uri)
	if err != nil || u.Scheme != "s3" {
		return fmt.Errorf("requires buckets to be prefixed with s3://")
	}

	params := &s3.ListObjectsV2Input{
		Bucket:  aws.String(u.Bucket), // Required
		MaxKeys: aws.Int64(1000),
	}
	if u.Path != "" && u.Path != "/" {
		params.Prefix = u.Key()
	}
	if delim {
		params.Delimiter = aws.String("/")
	}

	wrapper := func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		pager(page)
		return true
	}

	if svc == nil {
		svc = SessionNew(config)
	}

	bsvc, err := SessionForBucket(config, u.Bucket)
	if err != nil {
		return err
	}
	if err := bsvc.ListObjectsV2Pages(params, wrapper); err != nil {
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
					Name:     *obj.Key,
					Size:     *obj.Size,
					Checksum: *obj.ETag,
				})
			}
		}

		remotePager(config, svc, arg, false, pager)
	}

	return result, nil
}
