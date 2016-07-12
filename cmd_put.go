package main

import (
	"os"
	"fmt"
    "path/filepath"
	"strings"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
	"net/url"
)

func PutObject(config *Config, c *cli.Context) error {
	args := c.Args()
    if len(args) < 2 {
		return fmt.Errorf("not enough arguments for command 'put'")
    }

	svc := SessionNew(config)

	u, err := url.Parse(args[len(args)-1])
	if err != nil || u.Scheme != "s3" {
		return fmt.Errorf("Parameter problem: Destination must be S3Uri. Got: %s", args[len(args)-1])
	}

    bsvc := SessionForBucket(svc, u.Host)

    makeCopy := func(path string, f os.FileInfo, err error) error {
            if f.IsDir() {
                if config.Recursive {
                    return nil
                }
                return fmt.Errorf("Parameter problem: Use --recursive to upload a directory: %s", path)
            }

            fd, err := os.Open(path)
            if err != nil {
                return err
            }
            defer fd.Close()

            dstKey := u.Path[1:]
            if strings.HasSuffix(dstKey, "/") {
                parts := strings.Split(path, "/")
                dstKey += parts[len(parts)-1]
            }

            params := &s3.PutObjectInput{
                Bucket:             aws.String(u.Host), // Required
                Key:                aws.String(dstKey),  // Required
                Body:               fd,
            }
            if _, err := bsvc.PutObject(params); err != nil {
                return err
            }
            fmt.Printf("%s -> s3://%s/%s\n", path, u.Host, dstKey)
            return nil
    }

    for _, path := range args[0:len(args)-1] {
        filepath.Walk(path, makeCopy)
    }

	return nil
}
