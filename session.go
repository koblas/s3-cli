package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func SessionNew(config *Config) *s3.S3 {
	return s3.New(session.New())
}

func SessionForBucket(svc *s3.S3, bucket string) (*s3.S3, error) {
    if loc, err := svc.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: &bucket}); err != nil {
        return nil, err
    } else if (loc.LocationConstraint != nil) {
        return s3.New(session.New(&svc.Client.Config, &aws.Config{Region: loc.LocationConstraint})), nil
    }
    return svc, nil
}
