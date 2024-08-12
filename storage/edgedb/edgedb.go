package edgedb

import (
	"github.com/TheRebelOfBabylon/eventstore/edgedb"
	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/rs/zerolog"
)

var (
	queryIDsLimit     = 1000
	queryAuthorsLimit = 1000
	queryKindsLimit   = 100
	queryTagsLimit    = 100
	queryLimit        = 100
)

type EdgeDB struct {
	edgedb.EdgeDBBackend
	logger zerolog.Logger
	// TODO - may need to add RWMutex since FilterManager will read potentially at the same time as DB is writing
}

// ConnectEdgeDB establishes the connection to edgedb
func ConnectEdgeDB(cfg config.Storage, logger zerolog.Logger) (*EdgeDB, error) {
	backend := edgedb.EdgeDBBackend{
		DatabaseURI:       cfg.Uri,
		TLSSkipVerify:     cfg.SkipTlsVerify,
		QueryIDsLimit:     queryIDsLimit,
		QueryAuthorsLimit: queryAuthorsLimit,
		QueryKindsLimit:   queryKindsLimit,
		QueryTagsLimit:    queryTagsLimit,
		QueryLimit:        queryLimit,
	}
	logger.Info().Msgf("connecting to %s...", cfg.Uri)
	if err := backend.Init(); err != nil {
		return nil, err
	}
	logger.Info().Msg("connection successful")
	return &EdgeDB{
		EdgeDBBackend: backend,
		logger:        logger,
	}, nil
}

// TODO - Add a routine for receiving from ingester
