//go:build memory

package storage

import "github.com/TheRebelOfBabylon/tandem/config"

var TestStorageBackendConfig = func() config.Storage {
	return config.Storage{
		Uri: "memory://testing.db",
	}
}
