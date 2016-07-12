package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
	"net/url"
)

const DATE_FMT = "2006-01-02 15:04"

func ListBucket(config *Config, c *cli.Context) error {
	args := c.Args()

	svc := SessionNew(config)

	if len(args) == 0 {
		var params *s3.ListBucketsInput
		resp, err := svc.ListBuckets(params)
		if err != nil {
			return err
		}

		for _, bucket := range resp.Buckets {
			fmt.Printf("%s  s3://%s\n", bucket.CreationDate.Format(DATE_FMT), *bucket.Name)
		}
		return nil
	}

	u, err := url.Parse(args[0])
	if err != nil || u.Scheme != "s3" {
		return fmt.Errorf("ls requires buckets to be prefixed with s3://")
	}

	params := &s3.ListObjectsV2Input{
		Bucket:    aws.String(u.Host), // Required
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int64(1000),
	}
	if u.Path != "" && u.Path != "/" {
		params.Prefix = aws.String(u.Path[1:])
	}

	for true {
		resp, err := svc.ListObjectsV2(params)
		if err != nil {
			return err
		}

		if resp.CommonPrefixes != nil {
			for _, item := range resp.CommonPrefixes {
				fmt.Printf("%16s %9s   s3://%s/%s\n", "", "DIR", u.Host, *item.Prefix)
			}
		}
		if resp.Contents != nil {
			for _, item := range resp.Contents {
				fmt.Printf("%16s %9d   s3://%s/%s\n", item.LastModified.Format(DATE_FMT), *item.Size, u.Host, *item.Key)
			}
		}

		if resp.IsTruncated != nil && !*resp.IsTruncated {
			break
		}

		params.ContinuationToken = resp.NextContinuationToken
	}

	return nil
}
