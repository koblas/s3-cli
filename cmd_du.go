package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
	"net/url"
)

func GetUsage(config *Config, c *cli.Context) error {
	args := c.Args()

    svc := SessionNew(config)

    // If we're not passed any args, we're going to do all S3 buckets
    if len(args) == 0 {
        var params *s3.ListBucketsInput
        resp, err := svc.ListBuckets(params)
        if err != nil {
            return err
        }

        for _, bucket := range resp.Buckets {
            args = append(args, fmt.Sprintf("s3://%s", *bucket.Name))
        }
    }

    // Get the usage for the buckets
    for _, arg := range args {
        u, err := url.Parse(arg)
        if err != nil || u.Scheme != "s3" {
            continue
        }

        var (
            bucketSize, bucketObjs  int64
        )

        params := &s3.ListObjectsV2Input{
            Bucket:    aws.String(u.Host), // Required
            // Delimiter: aws.String("/"),
        }
        if u.Path != "" && u.Path != "/" {
            params.Prefix = aws.String(u.Path[1:])
        }

        pager := func(page *s3.ListObjectsV2Output, lastPage bool) (bool) {
            for _, obj := range page.Contents {
                bucketSize += *obj.Size
                bucketObjs += 1
            }
            return true
        }

        bsvc := SessionForBucket(svc, u.Host)

        if err := bsvc.ListObjectsV2Pages(params, pager); err != nil {
            fmt.Println(err)
            continue
        }

        fmt.Printf("%d %d objects %s\n", bucketSize, bucketObjs, arg)
    }
    // fmt.Printf("%d bytes\n", totalSize)

	return nil
}
