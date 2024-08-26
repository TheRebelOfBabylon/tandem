package filter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/TheRebelOfBabylon/tandem/storage"
	"github.com/nbd-wtf/go-nostr"
	"github.com/rs/zerolog"
)

var (
	ErrIngesterRecvNotSet = errors.New("ingester receive channel not set")
)

type FilterManager struct {
	recvFromIngester chan msg.ParsedMsg
	sendToWSHandler  chan msg.Msg
	quit             chan struct{}
	filters          map[string][]*nostr.ReqEnvelope
	dbConn           storage.StorageBackend
	logger           zerolog.Logger
	stopping         bool
	sync.WaitGroup
	sync.RWMutex
}

// NewFilterManager instantiates a new filter manager
func NewFilterManager(recvFromIngester chan msg.ParsedMsg, dbConn storage.StorageBackend, logger zerolog.Logger) *FilterManager {
	return &FilterManager{
		filters:          make(map[string][]*nostr.ReqEnvelope),
		recvFromIngester: recvFromIngester,
		sendToWSHandler:  make(chan msg.Msg),
		quit:             make(chan struct{}),
		dbConn:           dbConn,
		logger:           logger,
		stopping:         false,
	}
}

// Start will start the filter manager
func (f *FilterManager) Start() error {
	f.logger.Info().Msg("starting up...")
	f.Add(1)
	go f.manage()
	f.logger.Info().Msg("start up completed")
	return nil
}

// contains checks if the given connectionId has any filters with the given subscriptionId
func (f *FilterManager) contains(connectionId, subscriptionId string) bool {
	f.RLock()
	defer f.RUnlock()
	filters, ok := f.filters[connectionId]
	if !ok {
		return false
	}
	for _, filter := range filters {
		if filter.SubscriptionID == subscriptionId {
			return true
		}
	}
	return false
}

// endSubscription is called when we receive a CLOSE message from a client for a given subscription
func (f *FilterManager) endSubscription(connectionId, subscriptionId string) {
	f.Lock()
	defer f.Unlock()
	filters, ok := f.filters[connectionId]
	if !ok {
		return
	}
	newFilters := []*nostr.ReqEnvelope{}
	for _, filter := range filters {
		if filter.SubscriptionID == subscriptionId {
			continue
		}
		newFilters = append(newFilters, filter)
	}
	f.filters[connectionId] = newFilters
	return
}

// endConnection removes a given connectionId from the map
func (f *FilterManager) endConnection(connectionId string) {
	f.Lock()
	defer f.Unlock()
	delete(f.filters, connectionId)
}

// addSubscription appends a filter to the given list of filters for a given connectionId
func (f *FilterManager) addSubscription(connectionId string, subscription *nostr.ReqEnvelope) {
	f.Lock()
	defer f.Unlock()
	filters, ok := f.filters[connectionId]
	if !ok {
		f.filters[connectionId] = []*nostr.ReqEnvelope{subscription}
		return
	}
	// overwrite the existing filter if one with the subId exists
	for i, filter := range filters {
		if filter.SubscriptionID == subscription.SubscriptionID {
			filters[i] = subscription
			f.filters[connectionId] = filters
			return
		}
	}
	filters = append(filters, subscription)
	f.filters[connectionId] = filters
}

// matchAndSend is run as a goroutine. It will iterate over all filters for each connection Id and send the event to the Websocket handler as soon as there's a match
func (f *FilterManager) matchAndSend(event *nostr.EventEnvelope, sendChan chan msg.Msg) {
	defer f.Done()
	f.RLock()
	defer f.RUnlock()
	if f.stopping {
		f.logger.Warn().Msg("unable to send events, filterManager currently stopping")
		return
	}
	for connectionId, filters := range f.filters {
	innerLoop:
		for _, filter := range filters {
			if filter.Match(&event.Event) {
				sendChan <- msg.Msg{ConnectionId: connectionId, Data: event.Serialize()}
				break innerLoop // break as soon as we have a match to not accidentally send the same client the same event multiple times
			}
		}
	}
}

// manage is the main go routine to receive messages from the ingester
func (f *FilterManager) manage() {
	defer f.Done()
	if f.recvFromIngester == nil {
		f.logger.Error().Err(ErrIngesterRecvNotSet).Msg("failed to start filter management routine")
		return
	}
	for {
		select {
		case message, ok := <-f.recvFromIngester:
			if !ok {
				// TODO - handle
			}
			switch envelope := message.Data.(type) {
			case *nostr.EventEnvelope:
				if !f.stopping {
					f.Add(1)
					go f.matchAndSend(envelope, f.sendToWSHandler)
				}
			case *nostr.ReqEnvelope:
				// perform db query
				f.logger.Debug().Msgf("received from ingester: %v", envelope)
				for _, filter := range envelope.Filters {
					rcvChan, err := f.dbConn.QueryEvents(context.TODO(), filter)
					if err != nil {
						f.logger.Error().Err(err).Msg("failed to query database for events") // TODO - Should this be fatal?
					}
					timeOut := time.NewTimer(15 * time.Second) // TODO - Make this timeout configurable
				innerLoop:
					for {
						select {
						case event, ok := <-rcvChan:
							f.logger.Debug().Msgf("received from storage backend: %v", event)
							if !ok {
								break innerLoop
							}
							f.logger.Debug().Msgf("sending to websocket server: %v", event)
							f.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: event.Serialize()}
						case <-timeOut.C:
							f.logger.Warn().Str("connectionId", message.ConnectionId).Msgf("timeout reading all messages queried for this filter: %s", filter.String())
							break innerLoop
						}
					}
				}
				// send EOSE
				f.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: []byte(fmt.Sprintf(`["EOSE", "%s"]`, envelope.SubscriptionID))}
				f.logger.Debug().Msgf("registering new subscription with id %v...", envelope.SubscriptionID)
				f.addSubscription(message.ConnectionId, envelope)
				f.logger.Debug().Msgf("new subscription with id %v registered", envelope.SubscriptionID)
			case *nostr.CloseEnvelope:
				if f.contains(message.ConnectionId, string(*envelope)) {
					// remove it from our map
					f.endSubscription(message.ConnectionId, string(*envelope))
				}
			}
		case <-f.quit:
			f.logger.Info().Msg("stopping filter management routine...")
			return
		}
	}
}

// Stop safely stops the FilterManager instance
func (f *FilterManager) Stop() error {
	f.logger.Info().Msg("shutting down...")
	f.stopping = true
	close(f.quit)
	f.Wait()
	close(f.sendToWSHandler)
	f.logger.Info().Msg("shutdown completed")
	return nil
}

// SendChannel is wrapper around the send channel to the websocket handler
func (f *FilterManager) SendChannel() chan msg.Msg {
	return f.sendToWSHandler
}
