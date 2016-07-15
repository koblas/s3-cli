package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
)

func RemoveBucket(config *Config, c *cli.Context) error {
	args := c.Args()

	svc := SessionNew(config)
	u, err := FileURINew(args[0])
	if err != nil || u.Scheme != "s3" {
		return fmt.Errorf("ls requires buckets to be prefixed with s3://")
	}
	if u.Path != "/" {
		return fmt.Errorf("Parameter problem: Expecting S3 URI with just the bucket name set instead of '%s'", args[0])
	}

	params := &s3.DeleteBucketInput{
		Bucket: aws.String(u.Bucket), // Required
	}
	if _, err := svc.DeleteBucket(params); err != nil {
		return err
	}

	fmt.Printf("Bucket 's3://%s/' removed\n", u.Bucket)
	return nil
}
