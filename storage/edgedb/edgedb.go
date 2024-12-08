package edgedb

import (
	"errors"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/fiatjaf/eventstore/edgedb"
)

var (
	queryIDsLimit     = 1000
	queryAuthorsLimit = 1000
	queryKindsLimit   = 100
	queryTagsLimit    = 100
	queryLimit        = 100
	ErrRecvChanNotSet = errors.New("receive channel not set")
)

// ConnectEdgeDB establishes the connection to edgedb
func ConnectEdgeDB(cfg config.Storage) (*edgedb.EdgeDBBackend, error) {
	backend := edgedb.EdgeDBBackend{
		DatabaseURI:       cfg.Uri,
		TLSSkipVerify:     cfg.SkipTlsVerify,
		QueryIDsLimit:     queryIDsLimit,
		QueryAuthorsLimit: queryAuthorsLimit,
		QueryKindsLimit:   queryKindsLimit,
		QueryTagsLimit:    queryTagsLimit,
		QueryLimit:        queryLimit,
	}
	if err := backend.Init(); err != nil {
		return nil, err
	}
	return &backend, nil
}
