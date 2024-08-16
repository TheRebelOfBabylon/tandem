package edgedb

import (
	"context"
	"errors"
	"sync"

	"github.com/TheRebelOfBabylon/eventstore/edgedb"
	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/nbd-wtf/go-nostr"
	"github.com/rs/zerolog"
)

var (
	queryIDsLimit     = 1000
	queryAuthorsLimit = 1000
	queryKindsLimit   = 100
	queryTagsLimit    = 100
	queryLimit        = 100
	ErrRecvChanNotSet = errors.New("receive channel not set")
)

type EdgeDB struct {
	edgedb.EdgeDBBackend
	logger zerolog.Logger
	recv   chan msg.ParsedMsg
	quit   chan struct{}
	sync.WaitGroup
	// TODO - may need to add RWMutex since FilterManager will read potentially at the same time as DB is writing
}

// ConnectEdgeDB establishes the connection to edgedb
func ConnectEdgeDB(cfg config.Storage, logger zerolog.Logger, recv chan msg.ParsedMsg) (*EdgeDB, error) {
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
		recv:          recv,
		quit:          make(chan struct{}),
	}, nil
}

// TODO - Add a routine for receiving from ingester
func (e *EdgeDB) Start() error {
	e.Add(1)
	go e.receiveFromIngester(e.recv)
	return nil
}

// ReceiveFromIngester is a go-routine for receiving new events from the ingester and storing them in the db
func (e *EdgeDB) receiveFromIngester(recv chan msg.ParsedMsg) {
	defer e.Done()
	if recv == nil {
		e.logger.Error().Err(ErrRecvChanNotSet).Msg("failed to start receive from ingester routine")
		return
	}
	for {
		select {
		case <-e.quit:
			e.logger.Info().Msg("exiting from receive from ingester routine")
			return
		case message, ok := <-recv:
			if !ok {
				// TODO - handle
			}
			switch envelope := message.Data.(type) {
			case *nostr.EventEnvelope:
				if err := e.SaveEvent(context.TODO(), &envelope.Event); err != nil {
					e.logger.Fatal().Err(err).Msg("failed to store event")
				}
			default:
				e.logger.Warn().Msgf("invalid type %T for message, skipping...", message.Data)
			}
		}
	}
}

func (e *EdgeDB) Stop() error {
	close(e.quit)
	e.Wait()
	e.Close()
	return nil
}
