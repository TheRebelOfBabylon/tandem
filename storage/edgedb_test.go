//go:build edgedb

package storage

import "github.com/TheRebelOfBabylon/tandem/config"

var TestStorageBackendConfig = func() config.Storage {
	cfg, err := config.ReadConfig("../test.toml")
	if err != nil {
		panic(err)
	}
	return cfg.Storage
}
