package main

import (
	"fmt"
	"github.com/urfave/cli"
)

func Modify(config *Config, c *cli.Context) error {
	for _, arg := range c.Args() {
		u, err := FileURINew(arg)
		if err != nil {
			return fmt.Errorf("Invalid destination argument")
		}
		if u.Scheme != "s3" {
			return fmt.Errorf("only works on S3 objects")
		}
		if err := copyFile(config, u, u, false); err != nil {
			return err
		}
	}
	return nil
}
