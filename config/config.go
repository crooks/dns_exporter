package config

import (
	"flag"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Flags are the command line Flags
type Flags struct {
	Config string
	Debug  bool
}

type ResolveItem struct {
	Nameserver string `yaml:"nameserver"`
}

// Config contains the njmon_exporter configuration data
type Config struct {
	Logging struct {
		Journal  bool   `yaml:"journal"`
		LevelStr string `yaml:"level"`
	} `yaml:"logging"`
	Exporter struct {
		Address string `yaml:"address"`
		Port    int    `yaml:"port"`
	} `yaml:"exporter"`
	DefaultNS string                 `yaml:"default_ns"`
	Resolve   map[string]ResolveItem `yaml:"resolve"`
}

func addDNSPort(hostname string) string {
	colonTest := strings.Split(hostname, ":")
	if len(colonTest) == 1 {
		return hostname + ":53"
	}
	return hostname
}

// ParseConfig imports a yaml formatted config file into a Config struct
func ParseConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	d := yaml.NewDecoder(file)
	if err := d.Decode(&config); err != nil {
		return nil, err
	}
	if config.Exporter.Address == "" {
		config.Exporter.Address = "0.0.0.0"
	}
	if config.Exporter.Port == 0 {
		config.Exporter.Port = 9117
	}
	// If no default nameserver is specified, use one of Google's
	if config.DefaultNS == "" {
		config.DefaultNS = "8.8.8.8:53"
	}
	// Ensure the defaultNS option has a port number
	config.DefaultNS = addDNSPort(config.DefaultNS)
	// Ensure all the configured resolvers contain sufficient information
	// TODO: Validate IP address syntax
	for k, items := range config.Resolve {
		if items.Nameserver == "" {
			items.Nameserver = config.DefaultNS
		}
		items.Nameserver = addDNSPort(items.Nameserver)
		config.Resolve[k] = items
	}
	return config, nil
}

// parseFlags processes arguments passed on the command line in the format
// standard format: --foo=bar
func ParseFlags() *Flags {
	f := new(Flags)
	flag.StringVar(&f.Config, "config", "njmon_exporter.yml", "Path to njmon_exporter configuration file")
	flag.BoolVar(&f.Debug, "debug", false, "Expand logging with Debug level messaging and format")
	flag.Parse()
	return f
}
