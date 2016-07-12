package main

import (
	"io"
	"os"
	"path"
	"fmt"
	"strings"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli"
	"net/url"
)

// TODO: Handle arguments
// --recursive
func GetObject(config *Config, c *cli.Context) error {
	args := c.Args()
    if len(args) != 2 {
		return fmt.Errorf("not enough arguments for command 'get'")
    }

	svc := SessionNew(config)

	u, err := url.Parse(args[0])
	if err != nil || u.Scheme != "s3" {
		return fmt.Errorf("ls requires buckets to be prefixed with s3://")
	}

    // TODO: check recursive
	if u.Path == "" || strings.HasSuffix(u.Path, "/") {
        return fmt.Errorf("Parameter problem: Expecting S3 URI with a filename or --recursive: %s", args[0])
	}

    bsvc := SessionForBucket(svc, u.Host)

    parts := strings.Split(u.Path, "/")
    filename := parts[len(parts)-1]

    params := &s3.GetObjectInput{
        Bucket:                     aws.String(u.Host), // Required
        Key:                        aws.String(u.Path[1:]),  // Required
        /*
        IfMatch:                    aws.String("IfMatch"),
        IfModifiedSince:            aws.Time(time.Now()),
        IfNoneMatch:                aws.String("IfNoneMatch"),
        IfUnmodifiedSince:          aws.Time(time.Now()),
        Range:                      aws.String("Range"),
        RequestPayer:               aws.String("RequestPayer"),
        ResponseCacheControl:       aws.String("ResponseCacheControl"),
        ResponseContentDisposition: aws.String("ResponseContentDisposition"),
        ResponseContentEncoding:    aws.String("ResponseContentEncoding"),
        ResponseContentLanguage:    aws.String("ResponseContentLanguage"),
        ResponseContentType:        aws.String("ResponseContentType"),
        ResponseExpires:            aws.Time(time.Now()),
        SSECustomerAlgorithm:       aws.String("SSECustomerAlgorithm"),
        SSECustomerKey:             aws.String("SSECustomerKey"),
        SSECustomerKeyMD5:          aws.String("SSECustomerKeyMD5"),
        VersionId:                  aws.String("ObjectVersionId"),
        */
    }

    resp, err := bsvc.GetObject(params)
    if err != nil {
        return err
    }
    // fmt.Println(resp)

    dst := args[1]
    sinfo, err := os.Stat(dst)
    if err == nil {
        if sinfo.IsDir() {
            dst = path.Join(args[1], filename)
        }
    }

    if _, err := os.Stat(dst); err == nil {
        if config.SkipExisting {
            return nil
        }
        if !config.Force {
            return fmt.Errorf("Parameter problem: File %s already exists. Use either of --force / --continue / --skip-existing or give it a new name.", dst)
        }
    }

    out, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer out.Close()

    if _, err = io.Copy(out, resp.Body); err != nil {
        return err
    }

	return nil
}
