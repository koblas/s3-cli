package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
)

func ListAll(config *Config, c *cli.Context) error {
	args := c.Args()
	if len(args) != 0 {
		return fmt.Errorf("la shouldn't have arguments")
	}

	svc := SessionNew(config)

	var params *s3.ListBucketsInput
	resp, err := svc.ListBuckets(params)
	if err != nil {
		return err
	}

	for _, bucket := range resp.Buckets {
		uri := fmt.Sprintf("s3://%s", *bucket.Name)

		// Shared with "ls"
		listBucket(config, svc, []string{uri})
	}

	return nil
}
