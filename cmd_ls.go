package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
	"net/url"
)

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

    return listBucket(config, svc, args)
}

func listBucket(config *Config, svc *s3.S3, args []string) error {
    for _, arg := range args {
        u, err := url.Parse(arg)
        if err != nil || u.Scheme != "s3" {
            return fmt.Errorf("ls requires buckets to be prefixed with s3://")
        }

        todo := []string{arg}

        params := &s3.ListObjectsV2Input{
            Bucket:    aws.String(u.Host), // Required
            Delimiter: aws.String("/"),
            MaxKeys:   aws.Int64(1000),
        }

        for len(todo) != 0 {
            var item string
            item, todo = todo[0], todo[1:]

            u2, _ := url.Parse(item)

            if u2.Path != "" && u2.Path != "/" {
                params.Prefix = aws.String(u2.Path[1:])
            }

            bsvc := SessionForBucket(svc, u.Host)

            // Iterate through everything.
            for true {
                resp, err := bsvc.ListObjectsV2(params)
                if err != nil {
                    return err
                }

                if resp.CommonPrefixes != nil {
                    for _, item := range resp.CommonPrefixes {
                        uri := fmt.Sprintf("s3://%s/%s", u.Host, *item.Prefix)

                        if config.Recursive {
                            todo = append(todo, uri)
                        } else {
                            fmt.Printf("%16s %9s   %s\n", "", "DIR", uri)
                        }
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
        }
    }

	return nil
}
