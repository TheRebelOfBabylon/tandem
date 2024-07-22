package storage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/rs/zerolog"
)

var (
	ErrInvalidStorageUri = errors.New("invalid storage uri")
)

// Connect establishes the connection to the given storage backend based on a URI
func Connect(cfg config.Storage, logger zerolog.Logger) (StorageBackend, error) {
	parts := strings.Split(cfg.Uri, "://")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStorageUri, cfg.Uri)
	}
	switch parts[0] {
	case "edgedb":
		return ConnectEdgeDB(cfg, logger)
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidStorageUri, cfg.Uri)
	}
}
