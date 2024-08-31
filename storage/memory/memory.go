package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/TheRebelOfBabylon/eventstore/slicestore"
	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/nbd-wtf/go-nostr"
	"github.com/rs/zerolog"
)

var (
	ErrRecvChanNotSet = errors.New("receive channel not set")
)

type Memory struct {
	*slicestore.SliceStore
	logger zerolog.Logger
	recv   chan msg.ParsedMsg
	quit   chan struct{}
	sync.WaitGroup
}

func ConnectMemory(cfg config.Storage, logger zerolog.Logger, recv chan msg.ParsedMsg) (*Memory, error) {
	sliceStore := &slicestore.SliceStore{}
	if err := sliceStore.Init(); err != nil {
		return nil, err
	}
	return &Memory{
		SliceStore: sliceStore,
		logger:     logger,
		recv:       recv,
		quit:       make(chan struct{}),
	}, nil
}

func (m *Memory) Start() error {
	m.logger.Info().Msg("starting up...")
	m.Add(1)
	go m.receiveFromIngester(m.recv)
	m.logger.Info().Msg("start up completed")
	return nil
}

func (m *Memory) receiveFromIngester(recv chan msg.ParsedMsg) {
	defer m.Done()
	if recv == nil {
		m.logger.Error().Err(ErrRecvChanNotSet).Msg("failed to start receive from ingester routine")
		return
	}
loop:
	for {
		select {
		case <-m.quit:
			m.logger.Info().Msg("exiting from receive from ingester routine")
			return
		case message, ok := <-recv:
			if !ok {
				m.logger.Fatal().Msg("receive from ingester channel unexpectedly closed")
			}
			switch envelope := message.Data.(type) {
			case *nostr.EventEnvelope:
				m.logger.Debug().Str("connectionId", message.ConnectionId).Msgf("received from ingester: %v", envelope)
				if err := m.SaveEvent(context.TODO(), &envelope.Event); err != nil {
					message.Callback(err)
					m.logger.Error().Str("connectionId", message.ConnectionId).Err(err).Msg("failed to store event")
					continue loop
				}
				message.Callback(nil)
			default:
				m.logger.Warn().Str("connectionId", message.ConnectionId).Msgf("invalid type %T for message, skipping", message.Data)
			}
		}
	}
}

func (m *Memory) Stop() error {
	m.logger.Info().Msg("shutting down...")
	close(m.quit)
	m.Wait()
	m.Close()
	m.logger.Info().Msg("shutdown completed")
	return nil
}
