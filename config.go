package main

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-ini/ini"
	"github.com/urfave/cli"
)

// This is the global configuration, it's loaded from .s3cfg (by default) then with added
//  overrides from the command line
//
// Command lines are by default the snake case version of the the struct names with "-" instead of "_"
//
type Config struct {
	AccessKey    string `ini:"access_key"`
	SecretKey    string `ini:"secret_key"`
	StorageClass string `ini:"storage-class"`

	Concurrency int   `ini:"concurrency"`
	PartSize    int64 `ini:"part-size"`

	CheckMD5     bool `ini:"check_md5" cli:"check-md5"`
	DryRun       bool `ini:"dry_run"`
	Verbose      bool `ini:"verbose"`
	Recursive    bool `ini:"recursive"`
	Force        bool `ini:"force"`
	SkipExisting bool `ini:"skip_existing"`

	HostBase   string `ini:"host_base"`
	HostBucket string `ini:"host_bucket"`
}

var (
	validStorageClasses = map[string]bool{
		"":                                      true,
		s3.ObjectStorageClassStandard:           true,
		s3.ObjectStorageClassReducedRedundancy:  true,
		s3.ObjectStorageClassGlacier:            true,
		s3.ObjectStorageClassStandardIa:         true,
		s3.ObjectStorageClassOnezoneIa:          true,
		s3.ObjectStorageClassIntelligentTiering: true,
		s3.ObjectStorageClassDeepArchive:        true,
	}
)

// Read the configuration file if found, otherwise return default configuration
//  Precedence order (most important to least):
//   - Command Line options
//   - Environment Variables
//   - Config File
//   - Default Values
func NewConfig(c *cli.Context) (*Config, error) {
	var cfgPath string

	if obj := c.GlobalStringSlice("config"); len(obj) > 1 {
		cfgPath = obj[1]
	} else if obj := c.StringSlice("config"); len(obj) > 1 {
		cfgPath = obj[1]
	} else if value := GetEnv("HOME"); value != nil {
		cfgPath = path.Join(*value, ".s3cfg")
	} else {
		cfgPath = ".s3cfg"
	}

	config, err := loadConfigFile(cfgPath)
	if err != nil {
		return nil, err
	}

	parseOptions(config, c)

	if c.GlobalIsSet("no-check-md5") || c.IsSet("no-check-md5") {
		config.CheckMD5 = false
	}

	// Some additional validation
	if _, found := validStorageClasses[config.StorageClass]; !found {
		return nil, fmt.Errorf("Invalid storage class provided: %s", config.StorageClass)
	}

	return config, nil
}

// Load the config file if possible, but if there is an error return the default configuration file
func loadConfigFile(path string) (*Config, error) {
	config := &Config{CheckMD5: false}

	cfg, err := ini.Load(path)
	if err != nil {
		return config, nil
	}

	// s3cmd ini files are not Python configfiles --
	//
	// The INI file parser will use the %(bucket) and do a
	// lookup on the key, if not found it will panic
	// this puts a key in that causes the recursive lookup to limit
	// out but allows things to work.
	if _, err := cfg.Section("").NewKey("bucket", "%(bucket)s"); err != nil {
		// this shouldn't fail -- if it does the MapTo will fail
		return nil, fmt.Errorf("Unable to create bucket key")
	}

	if err := cfg.Section("default").MapTo(config); err != nil {
		return nil, err
	}

	return config, nil
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
