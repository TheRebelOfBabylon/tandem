package ingester

import (
	"errors"
	"sync"

	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/nbd-wtf/go-nostr"
	"github.com/rs/zerolog"
)

var (
	ErrRecvChanNotSet = errors.New("receive channel not set")
)

// TODO - Ingester needs to communicate with storage backend
// TODO - create ingester workers for communicating with storage backend,
type Ingester struct {
	logger            zerolog.Logger
	recvFromWSHandler chan msg.Msg
	sendToWSHandler   chan msg.Msg
	sendToDB          chan msg.ParsedMsg
	quit              chan struct{}
	sync.WaitGroup
}

// NewIngester instantiates the ingester
func NewIngester(logger zerolog.Logger) *Ingester {
	return &Ingester{
		logger:          logger,
		sendToWSHandler: make(chan msg.Msg),
		sendToDB:        make(chan msg.ParsedMsg),
		quit:            make(chan struct{}),
	}
}

// SetRecvChannel stores the receive channel (from the Websocket Handler) in the Ingester data structure for use in the ingest go routine
func (i *Ingester) SetRecvChannel(recv chan msg.Msg) {
	i.recvFromWSHandler = recv
}

// Start starts the ingest routine
func (i *Ingester) Start() error {
	i.Add(1)
	go i.ingest()
	return nil
}

// ingestWorker is spun up as a go routine to parse, validate and verify new messages
func (i *Ingester) ingestWorker(message msg.Msg) {
	defer i.Done() // TODO - Can this go routine hang on channel send?
	switch envelope := nostr.ParseMessage(message.Data).(type) {
	case *nostr.EventEnvelope:
		i.logger.Debug().Str("connectionId", message.ConnectionId).Msgf("raw event: %v\n", envelope)
		if ok, err := envelope.CheckSignature(); err != nil || !ok {
			msgBytes, err := nostr.OKEnvelope{
				EventID: envelope.ID,
				OK:      false,
				Reason:  "error: invalid event signature or event id",
			}.MarshalJSON()
			if err != nil {
				i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
			}
			i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes}
		}
		// send to db and filter manager
	case *nostr.ReqEnvelope:
		i.logger.Debug().Str("connectionId", message.ConnectionId).Msgf("raw req: %v\n", envelope)
		// send to filter manager
	case *nostr.CloseEnvelope:
		i.logger.Debug().Str("connectionId", message.ConnectionId).Msgf("raw close: %v\n", envelope)
		// send to filter manager
	case nil:
		// TODO - Should be banning IP addresses that abuse this and/or closing websocket connection
		i.logger.Error().Str("connectionId", message.ConnectionId).Msg("failed to parse message, skipping")
	}
}

// ingest is the goroutine which will receive messages over the recv channel and start up ingest workers
func (i *Ingester) ingest() {
	defer i.Done()
	if i.recvFromWSHandler == nil {
		i.logger.Fatal().Err(ErrRecvChanNotSet).Msg("failed to start ingest routine")
		return
	}
	for {
		select {
		case message, ok := <-i.recvFromWSHandler:
			if !ok {
				// handle
			}
			// TODO - remove this code
			i.logger.Debug().Msgf("received from websocket handler: %v", message)
			// Spin up a worker go routine which will
			i.Add(1)
			go i.ingestWorker(message)
		case <-i.quit:
			i.logger.Info().Msg("stopping ingest routine...")
			return
		}
	}
}

// SendToWSHandlerChannel is a wrapper over the send to Websocket Handler channel to safely pass along the channel to those who need it
func (i *Ingester) SendToWSHandlerChannel() chan msg.Msg {
	return i.sendToWSHandler
}

// SendToDBChannel is a wrapper over the send to DB channel to safely pass along the channel to those who need it
func (i *Ingester) SendToDBChannel() chan msg.ParsedMsg {
	return i.sendToDB
}

// Stop safely shuts down the ingester
func (i *Ingester) Stop() error {
	close(i.quit)
	i.Wait()
	close(i.sendToWSHandler)
	close(i.sendToDB)
	return nil
}
