package main

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// DefaultRegion to use for S3 credential creation
const defaultRegion = "us-east-1"

func buildSessionConfig(config *Config) aws.Config {
	// By default make sure a region is specified, this is required for S3 operations
	sessionConfig := aws.Config{Region: aws.String(defaultRegion)}

	if config.AccessKey != "" && config.SecretKey != "" {
		sessionConfig.Credentials = credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, "")
	}

	return sessionConfig
}

func buildEndpointResolver(hostname string) endpoints.Resolver {
	defaultResolver := endpoints.DefaultResolver()

	fixedHost := hostname
	if !strings.HasPrefix(hostname, "http") {
		fixedHost = "https://" + hostname
	}

	return endpoints.ResolverFunc(func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		if service == endpoints.S3ServiceID {
			return endpoints.ResolvedEndpoint{
				URL: fixedHost,
			}, nil
		}

		return defaultResolver.EndpointFor(service, region, optFns...)
	})
}

// SessionNew - Read the config for default credentials, if not provided use environment based variables
func SessionNew(config *Config) *s3.S3 {
	sessionConfig := buildSessionConfig(config)

	if config.HostBase != "" && config.HostBase != "s3.amazon.com" {
		sessionConfig.EndpointResolver = buildEndpointResolver(config.HostBase)
	}

	return s3.New(session.Must(session.NewSessionWithOptions(session.Options{
		Config:            sessionConfig,
		SharedConfigState: session.SharedConfigEnable,
	})))
}

// SessionForBucket - For a given S3 bucket, create an approprate session that references the region
// that this bucket is located in
func SessionForBucket(config *Config, bucket string) (*s3.S3, error) {
	sessionConfig := buildSessionConfig(config)

	if config.HostBucket == "" || config.HostBucket == "%(bucket)s.s3.amazonaws.com" {
		svc := SessionNew(config)

		if loc, err := svc.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: &bucket}); err != nil {
			return nil, err
		} else if loc.LocationConstraint == nil {
			// Use default service
			return svc, nil
		} else {
			sessionConfig.Region = loc.LocationConstraint
		}
	} else {
		host := strings.ReplaceAll(config.HostBucket, "%(bucket)s", bucket)

		sessionConfig.EndpointResolver = buildEndpointResolver(host)
	}

	return s3.New(session.Must(session.NewSessionWithOptions(session.Options{
		Config:            sessionConfig,
		SharedConfigState: session.SharedConfigEnable,
	}))), nil
}
