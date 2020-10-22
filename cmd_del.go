package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli/v2"
	"strings"
)

// TODO: Handle --recusrive
func DeleteObjects(config *Config, c *cli.Context) error {
	args := c.Args().Slice()

	svc := SessionNew(config)

	buckets := make(map[string][]*s3.ObjectIdentifier, 0)

	for _, path := range args {
		u, err := FileURINew(args[0])

		if err != nil || u.Scheme != "s3" {
			return fmt.Errorf("rm requires buckets to be prefixed with s3://")
		}

		if (u.Path == "" || strings.HasSuffix(u.Path, "/")) && !config.Recursive {
			return fmt.Errorf("Parameter problem: Expecting S3 URI with a filename or --recursive: %s", path)
		}

		objects := buckets[u.Bucket]
		if objects == nil {
			objects = make([]*s3.ObjectIdentifier, 0)
		}
		buckets[u.Bucket] = append(objects, &s3.ObjectIdentifier{Key: u.Key()})
	}

	// FIXME: Limited to 1000 objects, that's that shouldn't be an issue, but ...
	for bucket, objects := range buckets {
		bsvc, err := SessionForBucket(config, bucket)
		if err != nil {
			return err
		}

		if config.Recursive {
			for _, obj := range objects {
				uri := fmt.Sprintf("s3://%s/%s", bucket, *obj.Key)

				remotePager(config, svc, uri, false, func(page *s3.ListObjectsV2Output) {
					olist := make([]*s3.ObjectIdentifier, 0)
					for _, item := range page.Contents {
						olist = append(olist, &s3.ObjectIdentifier{Key: item.Key})

						fmt.Printf("delete: s3://%s/%s\n", bucket, *item.Key)
					}

					if !config.DryRun {
						params := &s3.DeleteObjectsInput{
							Bucket: aws.String(bucket), // Required
							Delete: &s3.Delete{
								Objects: olist,
							},
						}

						_, err := bsvc.DeleteObjects(params)
						if err != nil {
							fmt.Println("Error removing")
						}
					}
				})
			}
		} else if !config.DryRun {
			params := &s3.DeleteObjectsInput{
				Bucket: aws.String(bucket), // Required
				Delete: &s3.Delete{ // Required
					Objects: objects,
				},
			}

			_, err := bsvc.DeleteObjects(params)
			if err != nil {
				return err
			}
		}
		for _, objs := range objects {
			fmt.Printf("delete: s3://%s/%s\n", bucket, *objs.Key)
		}
	}

	return nil
}
