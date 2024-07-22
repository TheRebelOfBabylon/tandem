package config

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/BurntSushi/toml"
)

var (
	validLogLvls = []string{
		"debug",
		"info",
		"error",
	}
	defaultLogLvl      = "info"
	ErrInvalidLogLevel = errors.New("invalid log level")
)

type Log struct {
	Level       string `toml:"level"`
	LogFilePath string `toml:"log_file_path"`
}

type Storage struct {
	Uri           string `toml:"uri"`
	SkipTlsVerify bool   `toml:"skip_tls_verify"`
}

type Config struct {
	Log     Log     `toml:"log"`
	Storage Storage `toml:"storage"`
}

// ReadConfig reads the given config file
func ReadConfig(pathToConfig string) (*Config, error) {
	var cfg Config
	cfgFileBytes, err := os.ReadFile(pathToConfig)
	if err != nil {
		return nil, err
	}

	if err := toml.Unmarshal(cfgFileBytes, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate will perform validation on the given configuration
func (c *Config) Validate() error {
	if c.Log.Level == "" {
		c.Log.Level = defaultLogLvl
	}
	if !slices.Contains(validLogLvls, c.Log.Level) {
		return fmt.Errorf("%w: %s", ErrInvalidLogLevel, c.Log.Level)
	}
	return nil
}
