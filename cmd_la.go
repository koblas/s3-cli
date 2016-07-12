package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
)

func ListAll(config *Config, c *cli.Context) error {
	args := c.Args()
    if len(args) != 0 {
        return fmt.Errorf("la shouldn't have arguments")
    }

	svc := SessionNew(config)
    origsvc := svc

    var params *s3.ListBucketsInput
    resp, err := svc.ListBuckets(params)
    if err != nil {
        return err
    }

    for _, bucket := range resp.Buckets {
        params := &s3.ListObjectsV2Input{
            Bucket:    bucket.Name, // Required
            Delimiter: aws.String("/"),
            MaxKeys:   aws.Int64(1000),
        }

        if loc, err := svc.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: bucket.Name}); err != nil {
            return err
        } else if (loc.LocationConstraint != nil) {
            svc = SessionRegion(svc, *loc.LocationConstraint)
        } else {
            svc = origsvc
        }

        for true {
            resp, err := svc.ListObjectsV2(params)
            if err != nil {
                return err
            }

            if resp.CommonPrefixes != nil {
                for _, item := range resp.CommonPrefixes {
                    fmt.Printf("%16s %9s   s3://%s/%s\n", "", "DIR", *bucket.Name, *item.Prefix)
                }
            }
            if resp.Contents != nil {
                for _, item := range resp.Contents {
                    fmt.Printf("%16s %9d   s3://%s/%s\n", item.LastModified.Format(DATE_FMT), *item.Size, *bucket.Name, *item.Key)
                }
            }

            if resp.IsTruncated != nil && !*resp.IsTruncated {
                break
            }

            params.ContinuationToken = resp.NextContinuationToken
        }
    }

	return nil
}
