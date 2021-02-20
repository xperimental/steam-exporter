package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type Server struct {
	Address string `yaml:"address"`
}

type Config struct {
	ConfigFile    string        `yaml:"-"`
	Verbose       bool          `yaml:"-"`
	ListenAddress string        `yaml:"listenAddress"`
	DataTimeout   time.Duration `yaml:"dataTimeout"`
	Servers       []Server      `yaml:"servers"`
}

func Get(cmd string, args []string) (*Config, error) {
	cfg := &Config{
		ConfigFile:    "steam-exporter.yml",
		Verbose:       false,
		ListenAddress: ":9791",
		DataTimeout:   1 * time.Second,
	}

	flags := pflag.NewFlagSet(cmd, pflag.ContinueOnError)
	flags.StringVarP(&cfg.ConfigFile, "config-file", "c", cfg.ConfigFile, "Path to configuration file.")
	flags.BoolVarP(&cfg.Verbose, "verbose", "v", cfg.Verbose, "Show debugging output.")

	if err := flags.Parse(args); err != nil {
		return nil, fmt.Errorf("can not parse flags: %w", err)
	}

	cfgFile, err := os.Open(cfg.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("can not open configuration file: %w", err)
	}
	defer cfgFile.Close()

	decoder := yaml.NewDecoder(cfgFile)
	decoder.SetStrict(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("can not read configuration file: %w", err)
	}

	return cfg, nil
}
