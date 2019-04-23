package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
	"strings"
)

func GetInfo(config *Config, c *cli.Context) error {
	args := c.Args()

	// If we're not passed any args, we're going to do all S3 buckets
	if len(args) == 0 {
		return fmt.Errorf("Not enough parameters for command 'info'")
	}

	// Get the usage for the buckets
	for _, arg := range args {
		// Only do usage on S3 buckets
		u, err := FileURINew(arg)
		if err != nil || u.Scheme != "s3" {
			continue
		}

		bsvc, err := SessionForBucket(config, u.Bucket)
		if err != nil {
			return err
		}

		if u.Path == "" || u.Path == "/" {
			bucket := aws.String(u.Bucket)

			fmt.Printf("%s (bucket):\n", u.String())

			if info, err := bsvc.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: bucket}); err == nil {
				if info.LocationConstraint != nil {
					fmt.Printf("   Location: %s\n", *info.LocationConstraint)
				} else {
					fmt.Printf("   Location: %s\n", "none")
				}
			}
			if info, err := bsvc.GetBucketRequestPayment(&s3.GetBucketRequestPaymentInput{Bucket: bucket}); err == nil {
				if info.Payer != nil {
					fmt.Printf("   Payer: %s\n", *info.Payer)
				} else {
					fmt.Printf("   Payer: %s\n", "none")
				}
			}
		} else {
			params := &s3.HeadObjectInput{
				Bucket: aws.String(u.Bucket),
				Key:    u.Key(),
			}
			info, err := bsvc.HeadObject(params)
			if err != nil {
				fmt.Printf("Error fetching info for %s\n", u.String())
				continue
			}

			fmt.Printf("%s (object):\n", u.String())
			fmt.Printf("   File size: %d\n", *info.ContentLength)
			fmt.Printf("   Last mod: %s\n", info.LastModified.Format(DATE_FMT))
			fmt.Printf("   MIME type: %s\n", *info.ContentType)
			fmt.Printf("   MD5 sum: %s\n", strings.Trim(*info.ETag, "\""))
			if info.ServerSideEncryption != nil {
				fmt.Printf("   SSE: %s\n", *info.ServerSideEncryption)
			} else {
				fmt.Printf("   SSE: %s\n", "none")
			}

			// fmt.Printf("   policy: %d\n", *info.ContentLength)
			// fmt.Printf("   cors: %d\n", *info.ContentLength)
			// fmt.Printf("   ACL: %d\n", *info.)

			for k, v := range info.Metadata {
				fmt.Printf("   x-az-meta-%s: %s\n", k, *v)
			}
		}
	}

	return nil
}
