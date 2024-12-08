package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/TheRebelOfBabylon/tandem/storage/edgedb"
	"github.com/TheRebelOfBabylon/tandem/storage/memory"
	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
	"github.com/rs/zerolog"
)

var (
	ErrInvalidStorageUri  = errors.New("invalid storage uri")
	ErrUnsupportedBackend = errors.New("unsupported storage backend")
	ErrRecvChanNotSet     = errors.New("receive channel not set")
	ErrUnsupportedMsgType = errors.New("unsupported message type")
)

type StorageBackend struct {
	Store  eventstore.Store
	logger zerolog.Logger
	recv   chan msg.ParsedMsg
	quit   chan struct{}
	sync.WaitGroup
}

// Connect establishes the connection to the given storage backend based on a URI
func Connect(cfg config.Storage, logger zerolog.Logger, recv chan msg.ParsedMsg) (*StorageBackend, error) {
	parts := strings.Split(cfg.Uri, "://")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStorageUri, cfg.Uri)
	}
	switch parts[0] {
	case "edgedb":
		dbConn, err := edgedb.ConnectEdgeDB(cfg)
		if err != nil {
			logger.Error().Err(err).Msg("failed to connect to edge db")
			return nil, err
		}
		return &StorageBackend{
			Store:  dbConn,
			logger: logger,
			recv:   recv,
			quit:   make(chan struct{}),
		}, nil
	case "memory":
		dbConn, err := memory.ConnectMemory(cfg)
		if err != nil {
			logger.Error().Err(err).Msg("failed to connect to memory db")
		}
		return &StorageBackend{
			Store:  dbConn,
			logger: logger,
			recv:   recv,
			quit:   make(chan struct{}),
		}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedBackend, parts[0])
	}
}

// Start satisfies the StorageBackend interface
func (b *StorageBackend) Start() error {
	b.logger.Info().Msg("starting up...")
	b.Add(1)
	go b.receiveFromIngester(b.recv)
	b.logger.Info().Msg("start up completed")
	return nil
}

// receiveFromIngester is a goroutine made to receive events to store from the ingester
func (b *StorageBackend) receiveFromIngester(recv chan msg.ParsedMsg) {
	defer b.Done()
	if recv == nil {
		b.logger.Error().Err(ErrRecvChanNotSet).Msg("failed to start receive from ingester routine")
		return
	}
loop:
	for {
		select {
		case <-b.quit:
			b.logger.Info().Msg("exiting from receive from ingester routine")
			return
		case message, ok := <-recv:
			if !ok {
				b.logger.Fatal().Msg("receive from ingester channel unexpectedly closed")
			}
			switch envelope := message.Data.(type) {
			case *nostr.EventEnvelope:
				b.logger.Debug().Str("connectionId", message.ConnectionId).Msgf("received from ingester: %v", envelope)
				if message.DeleteEvent {
					if err := b.Store.DeleteEvent(context.TODO(), &envelope.Event); err != nil {
						message.Callback(err)
						b.logger.Error().Str("connectionId", message.ConnectionId).Err(err).Msg("failed to delete event")
						continue loop
					}
					message.Callback(nil)
					continue loop
				}
				if err := b.Store.SaveEvent(context.TODO(), &envelope.Event); err != nil {
					message.Callback(err)
					b.logger.Error().Str("connectionId", message.ConnectionId).Err(err).Msg("failed to store event")
					continue loop
				}
				message.Callback(nil)
			default:
				b.logger.Warn().Str("connectionId", message.ConnectionId).Msgf("invalid type %T for message, skipping", message.Data)
				message.Callback(ErrUnsupportedMsgType)
			}
		}
	}
}

// Stop satisfies the StorageBackend interface
func (b *StorageBackend) Stop() error {
	b.logger.Info().Msg("shutting down...")
	close(b.quit)
	b.Wait()
	b.Store.Close()
	b.logger.Info().Msg("shutdown completed")
	return nil
}
