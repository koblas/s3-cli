package main

import (
    "strings"
    "fmt"
	"github.com/urfave/cli"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
)

const DATE_FMT = "2006-01-02 15:04"

func ListBucket(config *Config, c *cli.Context) error {
    args := c.Args()

    svc := s3.New(session.New())

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

    if !strings.HasPrefix(args[0], "s3://") {
        return fmt.Errorf("ls requires buckets to be prefixed with s3://")
    }
    parts := strings.SplitN(args[0][5:len(args[0])], "/", 2)

    params := &s3.ListObjectsV2Input{
        Bucket:     aws.String(parts[0]), // Required
        Delimiter:  aws.String("/"),
        MaxKeys:    aws.Int64(1000),
    }
    if len(parts) > 1 {
        params.Prefix = &parts[1]
    }

    for true {
        resp, err := svc.ListObjectsV2(params)
        if err != nil {
            return err
        }

        if resp.CommonPrefixes != nil {
            for _, item := range resp.CommonPrefixes {
                fmt.Printf("%16s %9s   s3://%s/%s\n", "", "DIR", parts[0], *item.Prefix)
            }
        }
        if resp.Contents != nil {
            for _, item := range resp.Contents {
                fmt.Printf("%16s %9d   s3://%s/%s\n", item.LastModified.Format(DATE_FMT), *item.Size, parts[0], *item.Key)
            }
        }

        if resp.IsTruncated != nil && !*resp.IsTruncated {
            break
        }

        params.ContinuationToken = resp.NextContinuationToken
    }

    return nil
}
