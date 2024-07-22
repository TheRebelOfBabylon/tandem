package storage

import (
	"context"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/edgedb/edgedb-go"
	"github.com/rs/zerolog"
)

type EdgeDB struct {
	*edgedb.Client
	logger zerolog.Logger
}

// ConnectEdgeDB establishes the connection to edgedb
func ConnectEdgeDB(cfg config.Storage, logger zerolog.Logger) (*EdgeDB, error) {
	opts := edgedb.Options{}
	if cfg.SkipTlsVerify {
		opts.TLSOptions = edgedb.TLSOptions{SecurityMode: edgedb.TLSModeInsecure}
	}
	logger.Info().Msgf("connecting to %s...", cfg.Uri)
	dbConn, err := edgedb.CreateClientDSN(context.TODO(), cfg.Uri, opts)
	if err != nil {
		return nil, err
	}
	logger.Info().Msg("connection successful")
	return &EdgeDB{
		Client: dbConn,
		logger: logger,
	}, nil
}
