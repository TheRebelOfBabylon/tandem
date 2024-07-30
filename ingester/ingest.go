package ingester

import (
	"errors"
	"sync"

	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/rs/zerolog"
)

var (
	ErrRecvChanNotSet = errors.New("receive channel not set")
)

// TODO - Ingester needs to communicate with storage backend
// TODO - create ingester workers for communicating with storage backend,
type Ingester struct {
	logger zerolog.Logger
	recv   chan msg.Msg
	send   chan msg.Msg
	quit   chan struct{}
	sync.WaitGroup
}

// NewIngester instantiates the ingester
func NewIngester(logger zerolog.Logger) *Ingester {
	return &Ingester{
		logger: logger,
		send:   make(chan msg.Msg),
		quit:   make(chan struct{}),
	}
}

// SetRecvChannel stores the receive channel (from the Websocket Handler) in the Ingester data structure for use in the ingest go routine
func (i *Ingester) SetRecvChannel(recv chan msg.Msg) {
	i.recv = recv
}

// Start starts the ingest routine
func (i *Ingester) Start() error {
	i.Add(1)
	go i.ingest()
	return nil
}

// ingest is the goroutine which will receive messages over the recv channel and start up ingest workers
func (i *Ingester) ingest() {
	defer i.Done()
	if i.recv == nil {
		i.logger.Fatal().Err(ErrRecvChanNotSet).Msg("failed to start ingest routine")
		return
	}
	for {
		select {
		case msg, ok := <-i.recv:
			if !ok {
				// handle
			}
			// TODO - remove this code
			i.logger.Debug().Msgf("received from websocket handler: %v", msg)
			i.send <- msg
			// parse
			// validate
			// verify signature
		case <-i.quit:
			i.logger.Info().Msg("stopping ingest routine...")
			return
		}
	}
}

// SendChannel is a wrapper over the send channel to safely pass along the channel to those who need it
func (i *Ingester) SendChannel() chan msg.Msg {
	return i.send
}

// Close safely shuts down the ingester
func (i *Ingester) Close() error {
	close(i.send)
	close(i.quit)
	i.Wait()
	return nil
}
