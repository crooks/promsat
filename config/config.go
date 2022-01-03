package config

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// Config contains all the configuration settings for promsat
type Config struct {
	APICertFile string `yaml:"api_certfile"`
	APIPassword string `yaml:"api_password"`
	APIUser     string `yaml:"api_user"`

	BaseURLSatAPI  string `yaml:"baseurl_satellite"`
	BaseURLPromAPI string `yaml:"baseurl_prometheus"`

	ExporterJob string `yaml:"exporter_job"`
	OutJSON      string `yaml:"target_filename"`
	OutJSONTmp   string `yaml:"target_filename_tmp"`
	// Labels is a map of all labels that should be applied to auto-registered hosts.
	Labels map[string]string `yaml:"target_labels"`
	// AutoLabel is used to identify targets that have been auto-added.  The Labels map MUST include a key
	// matching AutoLabel.
	AutoLabel string `yaml:"autohosts_label"`
	AutoPort  int    `yaml:"autohosts_port"`
	// These hosts will not be added to the autohosts target file
	ExcludeHosts  []string `yaml:"exclude_hosts"`
	ExcludePrefix []string `yaml:"exclude_host_prefix"`
}

// Flags are the command line flags
type Flags struct {
	Config  string
	Debug   bool
	Version bool
}

// WriteConfig will create a YAML formatted config file from a Config struct
func (c *Config) WriteConfig(filename string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// ParseFlags transcribes command line flags into a struct
func ParseFlags() *Flags {
	f := new(Flags)
	// Config file
	flag.StringVar(&f.Config, "config", "", "Config file")
	flag.BoolVar(&f.Debug, "debug", false, "Output debugging info")
	flag.BoolVar(&f.Version, "version", false, "Print build info")
	flag.Parse()

	// If a "--config" flag hasn't been provided, try reading a YAMNCFG environment variable.
	if f.Config == "" && os.Getenv("PROMSATCFG") != "" {
		f.Config = os.Getenv("PROMSATCFG")
	}
	// If no other config is defined, boldly assume one.
	if f.Config == "" {
		f.Config = "/etc/promsat/config.yml"
	}
	return f
}

// ParseConfig expects a YAML formatted config file and populates a Config struct
func ParseConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	y := yaml.NewDecoder(file)
	config := new(Config)
	if err := y.Decode(&config); err != nil {
		return nil, err
	}
	if _, ok := config.Labels[config.AutoLabel]; !ok {
		err = fmt.Errorf("required label \"%s\" is not defined", config.AutoLabel)
		return nil, err
	}
	// Writing a JSON targets file is the entire point of this code.  Guessing a filename is probably silly but, here
	// goes.
	if config.OutJSON == "" {
		config.OutJSON = "/tmp/autoconf.json"
	}
	// If the tempfile isn't defined, make up a valid filename for it
	if config.OutJSONTmp == "" {
		config.OutJSONTmp = config.OutJSON + ".tmp"
	}
	// Set a default port for autohost targets.
	if config.AutoPort == 0 {
		config.AutoPort = 9100
	}
	// Another bold assumption if no config option is provided.
	if config.ExporterJob == "" {
		config.ExporterJob = "node_exporter"
	}
	return config, nil
}
