package ingester

import (
	"context"
	"errors"
	"fmt"
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
	queryFunc         func(context.Context, nostr.Filter) (chan *nostr.Event, error)
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

func (i *Ingester) SetQueryFunc(queryFunc func(context.Context, nostr.Filter) (chan *nostr.Event, error)) {
	i.queryFunc = queryFunc
}

// Start starts the ingest routine
func (i *Ingester) Start() error {
	i.logger.Info().Msg("starting up...")
	i.Add(1)
	go i.ingest()
	i.logger.Info().Msg("start up completed")
	return nil
}

// handleReplaceableEvent will query storage to see if we have an existing event with combination kind:pubkey or kind:pubkey:dTag and delete it/them if it/they exist
func (i *Ingester) handleReplaceableEvent(filter nostr.Filter, connectionId string) error {
	// check storage to see if we already have a combination of kind:pubkey or kind:pubkey:dTag
	rcvChan, err := i.queryFunc(context.Background(), filter)
	if err != nil {
		return fmt.Errorf("failed to query storage for any existing replaceable events: %w", err)
	}
	timer := time.NewTimer(5 * time.Second) // TODO - make this timeout configurable
	events := []*nostr.Event{}
queryLoop:
	for {
		select {
		case event, ok := <-rcvChan:
			if !ok || event == nil {
				break queryLoop
			}
			events = append(events, event)
		case <-timer.C:
			return errors.New("timed out while querying storage for any existing replaceable events")
		}
	}
	if len(events) > 0 {
		// delete all these events
		for _, event := range events {
			// send to db
			i.logger.Debug().Str("connectionId", connectionId).Msg("sending message to storage backend...")
			dbErrChan := make(chan error)
			i.sendToDB <- msg.ParsedMsg{ConnectionId: connectionId, Data: &nostr.EventEnvelope{Event: *event}, Callback: func(err error) { dbErrChan <- err }, DeleteEvent: true}
			timer := time.NewTimer(5 * time.Second) // TODO - make this timeout configurable

			i.logger.Debug().Str("connectionId", connectionId).Msg("awaiting signal from storage backend...")
			select {
			case err := <-dbErrChan:
				if err != nil {
					return fmt.Errorf("failed to delete stale replaceable events: %w", err)
				}
			case <-timer.C:
				return errors.New("timed out waiting for response from storage backend")
			}
		}
	}
	return nil
}

// ingestWorker is spun up as a go routine to parse, validate and verify new messages
// TODO - Add a timeout to this goroutine
func (i *Ingester) ingestWorker(message msg.Msg) {
	defer i.Done() // TODO - Can this go routine hang on channel send?
	i.logger.Debug().Str("connectionId", message.ConnectionId).Msg("starting ingest worker...")
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
		okMsg := nostr.OKEnvelope{
			EventID: envelope.ID,
			OK:      true,
			Reason:  "",
		}
		switch {
		// replaceable
		case envelope.Kind == 0 || envelope.Kind == 3 || (envelope.Kind >= 10000 && envelope.Kind < 20000):
			if err := i.handleReplaceableEvent(nostr.Filter{Kinds: []int{envelope.Kind}, Authors: []string{envelope.PubKey}}, message.ConnectionId); err != nil {
				i.logger.Error().Err(err).Msg("failed to handle replaceable event")
				okMsg.OK = false
				okMsg.Reason = fmt.Sprintf("error: %s", err.Error())
				msgBytes, err := okMsg.MarshalJSON()
				if err != nil {
					i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
				}
				i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes}
				return
			}
		// ephemeral
		case (envelope.Kind >= 20000 && envelope.Kind < 30000):
			// send OK message
			msgBytes, err := okMsg.MarshalJSON()
			if err != nil {
				i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
			}
			i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes}
			// send to filter manager
			i.sendToFilterMgr <- msg.ParsedMsg{ConnectionId: message.ConnectionId, Data: envelope}
			return
		// addressable
		case (envelope.Kind >= 30000 && envelope.Kind < 40000):
			// check if the event has a d tag, if so then handle like a parametrized replaceable event
			if dTag := envelope.Tags.GetD(); dTag != "" {
				if err := i.handleReplaceableEvent(nostr.Filter{Kinds: []int{envelope.Kind}, Authors: []string{envelope.PubKey}, Tags: nostr.TagMap{"d": []string{dTag}}}, message.ConnectionId); err != nil {
					i.logger.Error().Err(err).Msg("failed to handle replaceable event")
					okMsg.OK = false
					okMsg.Reason = fmt.Sprintf("error: %s", err.Error())
					msgBytes, err := okMsg.MarshalJSON()
					if err != nil {
						i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
					}
					i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes}
					return
				}
			}
		}
		// send to db
		i.logger.Debug().Str("connectionId", message.ConnectionId).Msg("sending message to storage backend...")
		dbErrChan := make(chan error)
		i.sendToDB <- msg.ParsedMsg{ConnectionId: message.ConnectionId, Data: envelope, Callback: func(err error) { dbErrChan <- err }}
		timer := time.NewTimer(5 * time.Second) // TODO - make this timeout configurable
		// wait for signal from storage
		i.logger.Debug().Str("connectionId", message.ConnectionId).Msg("awaiting signal from storage backend...")
		select {
		case err := <-dbErrChan:
			if err != nil {
				// send OK error message
				okMsg.OK = false
				okMsg.Reason = "error: failed to store event"
				// send OK message
				msgBytes, err := okMsg.MarshalJSON()
				if err != nil {
					i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
				}
				i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes}
				return
			}
			// send OK message
			msgBytes, err := okMsg.MarshalJSON()
			if err != nil {
				i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
			}
			i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes}
			// send to filter manager
			i.sendToFilterMgr <- msg.ParsedMsg{ConnectionId: message.ConnectionId, Data: envelope}
		case <-timer.C:
			i.logger.Error().Err(errors.New("timed out waiting for response from storage backend")).Str("connectionId", message.ConnectionId).Msg("failed to store event")
			msgBytes, err := nostr.OKEnvelope{
				EventID: envelope.ID,
				OK:      false,
				Reason:  "error: failed to store event",
			}.MarshalJSON()
			if err != nil {
				i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
			}
			i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes}
		}
	case *nostr.ReqEnvelope:
		// enforce subid being 64 characters in length
		if len(envelope.SubscriptionID) > 64 {
			i.logger.Error().Err(ErrSubIdTooLarge).Str("connectionId", message.ConnectionId).Msg("rejecting REQ")
			msgBytes, err := nostr.ClosedEnvelope{
				SubscriptionID: envelope.SubscriptionID,
				Reason:         "error: subscription id exceeds 64 character limit",
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
		if err := envelope.UnmarshalJSON(message.Data); err != nil {
			msgBytes, err := nostr.NoticeEnvelope("error: failed to parse message and continued failure to parse future messages will result in a ban").MarshalJSON()
			if err != nil {
				i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
			}
			i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes, Unparseable: true}
			i.logger.Error().Str("connectionId", message.ConnectionId).Msg("failed to parse message, skipping")
			return
		}
		// send to filter manager
		i.sendToFilterMgr <- msg.ParsedMsg{ConnectionId: message.ConnectionId, Data: envelope}
	case nil:
		i.logger.Debug().Str("connectionId", message.ConnectionId).Msgf("raw message: %s", string(message.Data))
		msgBytes, err := nostr.NoticeEnvelope("error: failed to parse message and continued failure to parse future messages will result in a ban").MarshalJSON()
		if err != nil {
			i.logger.Fatal().Err(err).Str("connectionId", message.ConnectionId).Msg("failed to JSON marshal message")
		}
		i.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: msgBytes, Unparseable: true}
		i.logger.Error().Str("connectionId", message.ConnectionId).Msg("failed to parse message, skipping")
	}
	i.logger.Debug().Str("connectionId", message.ConnectionId).Msg("ingest worker routine completed")
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
