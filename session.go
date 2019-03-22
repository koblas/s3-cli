package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// DefaultRegion to use for S3 credential creation
const defaultRegion = "us-east-1"

// SessionNew - Read the config for default credentials, if not provided use environment based variables
func SessionNew(config *Config) *s3.S3 {
	// By default make sure a region is specified, this is required for S3 operations
	var sessionConfig = &aws.Config{Region: aws.String(defaultRegion)}

	if config.AccessKey != "" && config.SecretKey != "" {
		sessionConfig.Credentials = credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, "")
	}

	return s3.New(session.Must(session.NewSession(sessionConfig)))
}

// SessionForBucket - For a given S3 bucket, create an approprate session that references the region
// that this bucket is located in
func SessionForBucket(svc *s3.S3, bucket string) (*s3.S3, error) {
	if loc, err := svc.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: &bucket}); err != nil {
		return nil, err
	} else if loc.LocationConstraint != nil {
		return s3.New(session.Must(session.NewSession(&svc.Client.Config, &aws.Config{Region: loc.LocationConstraint}))), nil
	}
	return svc, nil
}
