package main

import (
	"fmt"
	"os"
    // "log"
	"github.com/urfave/cli"
)

type CmdHandler func(*Config, *cli.Context) error

func CmdNotImplemented(*Config, *cli.Context) error {
    return fmt.Errorf("Command not implemented")
}

func main() {
    // Now the setup for the application

	cliapp := cli.NewApp()
	cliapp.Name = "s3-cli"
	// cliapp.Usage = ""
	cliapp.Version = "0.1.0"


	cliapp.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "config, c",
			Value: &cli.StringSlice{"$HOME/.s3cfg"},
			Usage: "Config `FILE` name.",
		},
		cli.StringFlag{
			Name:  "access_key",
			Usage: "AWS Access Key `ACCESS_KEY`",
		},
		cli.StringFlag{
			Name:  "secret_key",
			Usage: "AWS Secret Key `SECRET_KEY`",
		},
		cli.BoolFlag{
			Name:  "dry-run,n",
			Usage: "Only show what should be uploaded or downloaded but don't actually do it. May still perform S3 requests to get bucket listings and other information though (only for file transfer commands)",
		},
	}

    // The wrapper to launch a command -- take care of standard setup 
    //  before we get going
    launch := func(handler CmdHandler) (func(*cli.Context) error) {
        return func(c *cli.Context) error {
            err := handler(NewConfig(c), c)
            if err != nil {
                fmt.Println(err.Error())
            }
            return err
        }
    }

	cliapp.Commands = []cli.Command{
		{
			Name:  "mb",
			Usage: "Make bucket -- s3-cli mb s3://BUCKET",
			Action: launch(MakeBucket),
		},
		{
			Name:  "rb",
			Usage: "Remove bucket -- s3-cli mb s3://BUCKET",
			Action: launch(RemoveBucket),
		},
		{
			Name:  "ls",
			Usage: "List objects or buckets -- s3-cli ls [s3://BUCKET[/PREFIX]]",
			Action: launch(ListBucket),
		},
		{
			Name:  "la",
			Usage: "List all object in all buckets -- s3-cli la",
			Action: launch(CmdNotImplemented),
		},
		{
			Name:  "put",
			Usage: "Get file from bucket -- s3-cli put FILE [FILE....] s3://BUCKET/PREFIX",
			Action: launch(CmdNotImplemented),
		},
		{
			Name:  "get",
			Usage: "Get file from bucket -- s3-cli get s3://BUCKET/OBJECT LOCAL_FILE",
			Action: launch(CmdNotImplemented),
		},
		{
			Name:  "del",
			Usage: "Delete file from bucket -- s3-cli del s3://BUCKET/OBJECT",
			Action: launch(CmdNotImplemented),
		},
		{
			Name:  "rm",
			Usage: "Delete file from bucket (del synonym) -- s3-cli rm s3://BUCKET/OBJECT",
			Action: launch(CmdNotImplemented),
		},
	}

	cliapp.Run(os.Args)
}
