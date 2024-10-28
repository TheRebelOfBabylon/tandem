//go:build memory

package storage

import "github.com/TheRebelOfBabylon/tandem/config"

var TestStorageBackendConfig = config.Storage{
	Uri: "memory://testing.db",
}
