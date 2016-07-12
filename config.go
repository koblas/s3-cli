package main

import (
	// "fmt"
	"github.com/go-ini/ini"
	"github.com/urfave/cli"
	"path"
)

// This is the global configuration, it's loaded from .s3cfg (by default) then with added
//  overrides from the command line
type Config struct {
	AccessKey string `ini:"access_key"`
	SecretKey string `ini:"secret_key"`

    Recursive     bool  `ini:recursive`
    Force     bool  `ini:force`
    SkipExisting     bool  `ini:skip_existing`
}

// Read the configuration file if found, otherwise return default configuration
//  Precedence order (most important to least):
//   - Command Line options
//   - Environment Variables
//   - Config File
//   - Default Values
func NewConfig(c *cli.Context) *Config {
	cfgPath := "/.s3cfg"

	if c.IsSet("config") {
		cfgPath = c.String("config")
	} else {
		if value := GetEnv("HOME"); value != nil {
			cfgPath = path.Join(*value, ".s3cfg")
		}
	}

	config := loadConfigFile(cfgPath)

	if value := GetEnv("AWS_ACCESS_KEY_ID"); value != nil {
		config.AccessKey = *value
	}
	if value := GetEnv("AWS_SECRET_ACCESS_KEY"); value != nil {
		config.SecretKey = *value
	}

	if c.GlobalIsSet("access_key") {
		config.AccessKey = c.GlobalString("access_key")
	}
	if c.GlobalIsSet("secret_key") {
		config.AccessKey = c.GlobalString("secret_key")
	}
	if c.GlobalIsSet("force") {
		config.Force = c.GlobalBool("force")
	}
	if c.GlobalIsSet("skip-existing") {
		config.SkipExisting = c.GlobalBool("skip-existing")
	}
	if c.GlobalIsSet("recursive") {
		config.Recursive = c.GlobalBool("recursive")
	}

	// fmt.Println(config)

	return config
}

// Load the config file if possible, but if there is an error return the default configuration file
func loadConfigFile(path string) *Config {
	config := Config{}

	// fmt.Println("Read config ", path)

	if err := ini.MapTo(config, path); err != nil {
		return &config
	}

	return &config
}
