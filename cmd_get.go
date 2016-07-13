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

// Check arguments:
// if not --recursive:
//   - first N arguments must be S3Uri
//   - if the last one is S3 make current dir the destination_base
//   - if the last one is a directory:
//       - take all 'basenames' of the remote objects and
//         make the destination name be 'destination_base'+'basename'
//   - if the last one is a file or not existing:
//       - if the number of sources (N, above) == 1 treat it
//         as a filename and save the object there.
//       - if there's more sources -> Error
// if --recursive:
//   - first N arguments must be S3Uri
//       - for each Uri get a list of remote objects with that Uri as a prefix
//       - apply exclude/include rules
//       - each list item will have MD5sum, Timestamp and pointer to S3Uri
//         used as a prefix.
//   - the last arg may be '-' (stdout)
//   - the last arg may be a local directory - destination_base
//   - if the last one is S3 make current dir the destination_base
//   - if the last one doesn't exist check remote list:
//       - if there is only one item and its_prefix==its_name
//         download that item to the name given in last arg.
//       - if there are more remote items use the last arg as a destination_base
//         and try to create the directory (incl. all parents).
//
// In both cases we end up with a list mapping remote object names (keys) to local file names.

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
