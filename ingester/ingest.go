package ingester

import (
	"errors"
	"sync"
	"time"

	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/nbd-wtf/go-nostr"
	"github.com/rs/zerolog"
)

var (
	ErrRecvChanNotSet = errors.New("receive channel not set")
	ErrSubIdTooLarge  = errors.New("subscription id too large")
)

type Ingester struct {
	logger            zerolog.Logger
	recvFromWSHandler chan msg.Msg
	sendToWSHandler   chan msg.Msg
	sendToDB          chan msg.ParsedMsg
	sendToFilterMgr   chan msg.ParsedMsg
	quit              chan struct{}
	stopping          bool
	sync.WaitGroup
	sync.RWMutex
}

// NewIngester instantiates the ingester
func NewIngester(logger zerolog.Logger) *Ingester {
	return &Ingester{
		logger:          logger,
		sendToWSHandler: make(chan msg.Msg),
		sendToDB:        make(chan msg.ParsedMsg),
		sendToFilterMgr: make(chan msg.ParsedMsg),
		quit:            make(chan struct{}),
		stopping:        false,
	}
}

// SetRecvChannel stores the receive channel (from the Websocket Handler) in the Ingester data structure for use in the ingest go routine
func (i *Ingester) SetRecvChannel(recv chan msg.Msg) {
	i.recvFromWSHandler = recv
}

// Start starts the ingest routine
func (i *Ingester) Start() error {
	i.logger.Info().Msg("starting up...")
	i.Add(1)
	go i.ingest()
	i.logger.Info().Msg("start up completed")
	return nil
}

// ingestWorker is spun up as a go routine to parse, validate and verify new messages
// TODO - Add a timeout to this goroutine
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
			return
		}
		// send to db
		dbErrChan := make(chan error)
		i.sendToDB <- msg.ParsedMsg{ConnectionId: message.ConnectionId, Data: envelope, Callback: func(err error) { dbErrChan <- err }}
		timer := time.NewTimer(5 * time.Second) // TODO - make this timeout configurable
	waitForDbLoop:
		for {
			select {
			case err := <-dbErrChan:
				okMsg := nostr.OKEnvelope{
					EventID: envelope.ID,
					OK:      true,
					Reason:  "",
				}
				if err != nil {
					// send OK error message
					okMsg.OK = false
					okMsg.Reason = "error: failed to store event"
					msgBytes, err := okMsg.MarshalJSON()
					if err != nil {
						i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
					}
					i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes}
					break waitForDbLoop
				}
				// send OK message
				msgBytes, err := okMsg.MarshalJSON()
				if err != nil {
					i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
				}
				i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes}
				// send to filter manager
				i.sendToFilterMgr <- msg.ParsedMsg{ConnectionId: message.ConnectionId, Data: envelope}
				break waitForDbLoop
			case <-timer.C:
				msgBytes, err := nostr.OKEnvelope{
					EventID: envelope.ID,
					OK:      false,
					Reason:  "error: timed out waiting for signal from storag",
				}.MarshalJSON()
				if err != nil {
					i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
				}
				i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes}
				break waitForDbLoop
			}
		}
	case *nostr.ReqEnvelope:
		// enforce subid being 64 characters in length
		if len(envelope.SubscriptionID) > 64 {
			i.logger.Error().Err(ErrSubIdTooLarge).Str("connectionId", message.ConnectionId).Msg("rejecting REQ")
			msgBytes, err := nostr.ClosedEnvelope{
				SubscriptionID: envelope.SubscriptionID,
				Reason:         "error: subscription id too large",
			}.MarshalJSON()
			if err != nil {
				i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
			}
			i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes}
			return
		}
		i.logger.Debug().Str("connectionId", message.ConnectionId).Msgf("raw req: %v\n", envelope)
		// send to filter manager
		i.sendToFilterMgr <- msg.ParsedMsg{ConnectionId: message.ConnectionId, Data: envelope}
	case *nostr.CloseEnvelope:
		i.logger.Debug().Str("connectionId", message.ConnectionId).Msgf("raw close: %v\n", envelope)
		// send to filter manager
		i.sendToFilterMgr <- msg.ParsedMsg{ConnectionId: message.ConnectionId, Data: envelope}
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
loop:
	for {
		select {
		case message, ok := <-i.recvFromWSHandler:
			if !ok && !i.isStopping() {
				i.logger.Warn().Msg("receive from websocket handler channel unexpectedely closed") // TODO - somehow fix this
				return
			} else if !ok && i.isStopping() {
				continue loop
			}
			if message.CloseConn {
				i.sendToFilterMgr <- msg.ParsedMsg{ConnectionId: message.ConnectionId, CloseConn: true}
				continue loop
			}
			// Spin up a worker go routine which will ingest the message
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

// SendToFilterManager is a wrapper over the send to Filter Manager channel to safely pass along the channel to those who need it
func (i *Ingester) SendToFilterManager() chan msg.ParsedMsg {
	return i.sendToFilterMgr
}

// toggleStopping sets the stopping state accordingly
func (i *Ingester) toggleStopping(state bool) {
	i.Lock()
	defer i.Unlock()
	i.stopping = state
}

// isStopping checks if the ingester is currently stopping
func (i *Ingester) isStopping() bool {
	i.RLock()
	defer i.RUnlock()
	return i.stopping
}

// Stop safely shuts down the ingester
func (i *Ingester) Stop() error {
	i.logger.Info().Msg("shutting down...")
	i.toggleStopping(true)
	close(i.quit)
	i.Wait()
	close(i.sendToWSHandler)
	close(i.sendToDB)
	close(i.sendToFilterMgr)
	i.logger.Info().Msg("shutdown completed")
	return nil
}
