package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/sethvargo/go-envconfig"
)

var (
	validLogLvls = []string{
		"debug",
		"info",
		"error",
	}
	defaultLogLvl      = "info"
	ErrInvalidLogLevel = errors.New("invalid log level")
	defaultHost        = "localhost"
	defaultPort        = 5000
)

type HTTP struct {
	Host string `toml:"host" env:"HOST, overwrite"`
	Port int    `toml:"port" env:"PORT, overwrite"`
}

type Log struct {
	Level       string `toml:"level" env:"LEVEL, overwrite"`
	LogFilePath string `toml:"log_file_path" env:"FILE_PATH, overwrite"`
}

type Storage struct {
	Uri           string `toml:"uri" env:"URI, overwrite"`
	SkipTlsVerify bool   `toml:"skip_tls_verify" env:"SKIP_TLS_VERIFY, overwrite"`
}

type Config struct {
	HTTP    HTTP    `toml:"http" env:", prefix=HTTP_"`
	Log     Log     `toml:"log" env:", prefix=LOG_"`
	Storage Storage `toml:"storage" env:", prefix=STORAGE_"`
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
	// env vars override anything in the config file
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
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
	if c.HTTP.Host == "" {
		c.HTTP.Host = defaultHost
	}
	if c.HTTP.Port == 0 {
		c.HTTP.Port = defaultPort
	}
	return nil
}
