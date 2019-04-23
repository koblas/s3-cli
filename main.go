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
	cliapp.Version = "0.2.0"

	cli.VersionFlag = cli.BoolFlag{
		Name:  "version, V",
		Usage: "print version number",
	}

	cliapp.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "config, c",
			Value: &cli.StringSlice{"$HOME/.s3cfg"},
			Usage: "Config `FILE` name.",
		},
		cli.StringFlag{
			Name:   "access-key",
			Usage:  "AWS Access Key `ACCESS_KEY`",
			EnvVar: "AWS_ACCESS_KEY_ID,AWS_ACCESS_KEY",
		},
		cli.StringFlag{
			Name:   "secret-key",
			Usage:  "AWS Secret Key `SECRET_KEY`",
			EnvVar: "AWS_SECRET_ACCESS_KEY,AWS_SECRET_KEY",
		},

		cli.BoolFlag{
			Name:  "recursive,r",
			Usage: "Recursive upload, download or removal",
		},
		cli.BoolFlag{
			Name:  "force",
			Usage: "Force overwrite and other dangerous operations.",
		},
		cli.BoolFlag{
			Name:  "skip-existing",
			Usage: "Skip over files that exist at the destination (only for [get] and [sync] commands).",
		},

		cli.BoolFlag{
			Name:  "verbose,v",
			Usage: "Verbose output (e.g. debugging)",
		},
		cli.BoolFlag{
			Name:  "dry-run,n",
			Usage: "Only show what should be uploaded or downloaded but don't actually do it. May still perform S3 requests to get bucket listings and other information though (only for file transfer commands)",
		},
		cli.BoolFlag{
			Name:  "check-md5",
			Usage: "Check MD5 sums when comparing files for [sync]. (not default)",
		},
		cli.BoolFlag{
			Name:  "no-check-md5",
			Usage: "Do not check MD5 sums when comparing files for [sync] (default).",
		},
	}

	// The wrapper to launch a command -- take care of standard setup
	//  before we get going
	launch := func(handler CmdHandler) func(*cli.Context) error {
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
			Name:   "mb",
			Usage:  "Make bucket -- s3-cli mb s3://BUCKET",
			Action: launch(MakeBucket),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "rb",
			Usage:  "Remove bucket -- s3-cli mb s3://BUCKET",
			Action: launch(RemoveBucket),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "ls",
			Usage:  "List objects or buckets -- s3-cli ls [s3://BUCKET[/PREFIX]]",
			Action: launch(ListBucket),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "la",
			Usage:  "List all object in all buckets -- s3-cli la",
			Action: launch(ListAll),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "put",
			Usage:  "Put file from bucket (really 'cp') -- s3-cli put FILE [FILE....] s3://BUCKET/PREFIX",
			Action: launch(CmdCopy),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "get",
			Usage:  "Get file from bucket (really 'cp') -- s3-cli get s3://BUCKET/OBJECT LOCAL_FILE",
			Action: launch(CmdCopy),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "del",
			Usage:  "Delete file from bucket -- s3-cli del s3://BUCKET/OBJECT",
			Action: launch(DeleteObjects),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "rm",
			Usage:  "Delete file from bucket (del synonym) -- s3-cli rm s3://BUCKET/OBJECT",
			Action: launch(DeleteObjects),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "du",
			Usage:  "Disk usage by buckets -- [s3://BUCKET[/PREFIX]]",
			Action: launch(GetUsage),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "cp",
			Usage:  "copy files and directories -- SRC [SRC...] DST",
			Action: launch(CmdCopy),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "sync",
			Usage:  "Synchronize a directory tree to S3 -- LOCAL_DIR s3://BUCKET[/PREFIX] or s3://BUCKET[/PREFIX] LOCAL_DIR",
			Action: launch(CmdSync),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "modify",
			Usage:  "Modify object metadata -- s3://BUCKET1/OBJECT",
			Action: launch(Modify),
			Flags:  cliapp.Flags,
		},
		{
			Name:   "info",
			Usage:  "Get various information about Buckets or Files -- s3://BUCKET[/OBJECT]",
			Action: launch(GetInfo),
			Flags:  cliapp.Flags,
		},
		// info
		// mv
	}

	cliapp.Run(os.Args)
}
