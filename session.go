package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func SessionNew(config *Config) *s3.S3 {
	return s3.New(session.New())
}

func SessionRegion(svc *s3.S3, region string) *s3.S3 {
    return s3.New(session.New(&svc.Client.Config, &aws.Config{Region: &region}))
}
