//go:build integration

package storage

import "github.com/TheRebelOfBabylon/tandem/config"

var TestStorageBackendConfig = config.Storage{
	Uri: "edgedb://",
}
