package main

import (
    // "fmt"
    "strings"
    "reflect"
	"github.com/go-ini/ini"
	"github.com/urfave/cli"
	"path"
)

// This is the global configuration, it's loaded from .s3cfg (by default) then with added
//  overrides from the command line
//
// Command lines are by default the snake case version of the the struct names with "-" instead of "_"
//
type Config struct {
	AccessKey string `ini:"access_key"`
	SecretKey string `ini:"secret_key"`

    CheckMD5  bool  `ini:"check_md5" cli:"check-md5"`
    DryRun      bool  `ini:"dry_run"`
    Verbose     bool  `ini:"verbose"`
    Recursive     bool  `ini:"recursive"`
    Force     bool  `ini:"force"`
    SkipExisting     bool  `ini:"skip_existing"`
}

// Read the configuration file if found, otherwise return default configuration
//  Precedence order (most important to least):
//   - Command Line options
//   - Environment Variables
//   - Config File
//   - Default Values
func NewConfig(c *cli.Context) *Config {
	cfgPath := "/.s3cfg"

	if c.GlobalIsSet("config") {
		cfgPath = c.GlobalString("config")
	} else if c.IsSet("config") {
		cfgPath = c.String("config")
	} else {
		if value := GetEnv("HOME"); value != nil {
			cfgPath = path.Join(*value, ".s3cfg")
		}
	}

	config := loadConfigFile(cfgPath)

    parseOptions(config, c)

    if c.GlobalIsSet("no-check-md5") || c.IsSet("no-check-md5") {
        config.CheckMD5 = false
    }

	return config
}

// Load the config file if possible, but if there is an error return the default configuration file
func loadConfigFile(path string) *Config {
	config := &Config{CheckMD5: false}

	// fmt.Println("Read config ", path)

	cfg, err := ini.Load(path)
	if err != nil {
		return config
	}

	if err := cfg.Section("default").MapTo(config); err != nil {
		return config
	}

	return config
}

// Pull the options out of the cli.Context and save them into the configuration object
func parseOptions(config *Config, c *cli.Context) {
    rt := reflect.TypeOf(*config)
    rv := reflect.ValueOf(config)

    for i := 0; i < rt.NumField(); i++ {
        field := rt.Field(i)

        name := ""
        if field.Tag.Get("cli") != "" {
            name = field.Tag.Get("cli")
        } else {
            name = strings.Replace(CamelToSnake(field.Name), "_", "-", -1)
        }

        gset := c.GlobalIsSet(name)
        lset := c.IsSet(name)

        // fmt.Println(name, gset, lset, c.String(name))

        // FIXME: This isn't great, "IsSet()" isn't triggered for environment variables
        if !gset && !lset && c.String(name) == "" {
            continue
        }

        f := rv.Elem().FieldByName(field.Name)

        if !f.IsValid() || !f.CanSet() {
            continue
        }

        switch f.Kind() {
        case reflect.Bool:
            if lset {
                f.SetBool(c.Bool(name))
            } else {
                f.SetBool(c.GlobalBool(name))
            }
        case reflect.String:
            if lset {
                f.SetString(c.String(name))
            } else {
                f.SetString(c.GlobalString(name))
            }
        case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
            if lset {
                f.SetInt(c.Int64(name))
            } else {
                f.SetInt(c.GlobalInt64(name))
            }
        }
    }
}
