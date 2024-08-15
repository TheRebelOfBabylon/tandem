package storage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/TheRebelOfBabylon/tandem/storage/edgedb"
	"github.com/TheRebelOfBabylon/tandem/storage/memory"
	"github.com/rs/zerolog"
)

var (
	ErrInvalidStorageUri  = errors.New("invalid storage uri")
	ErrUnsupportedBackend = errors.New("unsupported storage backend")
)

// Connect establishes the connection to the given storage backend based on a URI
func Connect(cfg config.Storage, logger zerolog.Logger, recv chan msg.ParsedMsg) (StorageBackend, error) {
	parts := strings.Split(cfg.Uri, "://")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStorageUri, cfg.Uri)
	}
	switch parts[0] {
	case "edgedb":
		return edgedb.ConnectEdgeDB(cfg, logger, recv)
	case "memory":
		return memory.ConnectMemory(cfg, logger, recv)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedBackend, parts[0])
	}
}
