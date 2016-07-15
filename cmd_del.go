package main

import (
	"fmt"
    "strings"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
)

// TODO: Handle --recusrive
func DeleteObjects(config *Config, c *cli.Context) error {
	args := c.Args()

	svc := SessionNew(config)

    buckets := make(map[string][]*s3.ObjectIdentifier, 0)

    for _, path := range args {
        u, err := FileURINew(args[0])

        if err != nil || u.Scheme != "s3" {
            return fmt.Errorf("ls requires buckets to be prefixed with s3://")
        }

        if u.Path == "" || strings.HasSuffix(u.Path, "/") {
            return fmt.Errorf("Parameter problem: Expecting S3 URI with a filename or --recursive: %s", path)
        }

        objects := buckets[u.Bucket]
        if objects == nil {
            objects = make([]*s3.ObjectIdentifier, 0)
        }
        buckets[u.Bucket] = append(objects, &s3.ObjectIdentifier{ Key: u.Key() })
    }

    // FIXME: Limited to 1000 objects, that's that shouldn't be an issue, but ...
    for bucket, objects := range buckets {
        params := &s3.DeleteObjectsInput{
            Bucket: aws.String(bucket), // Required
            Delete: &s3.Delete{ // Required
                Objects: objects,
            },
        }

        bsvc := SessionForBucket(svc, bucket)

        _, err := bsvc.DeleteObjects(params)
        if err != nil {
            return err
        }
        for _, objs := range objects {
            fmt.Printf("delete: s3://%s/%s\n", bucket, *objs.Key)
        }
    }

	return nil
}
