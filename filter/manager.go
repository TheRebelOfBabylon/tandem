package filter

import (
	"context"
	"errors"
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

type Filters map[string][]*nostr.ReqEnvelope

// Contains checks if the given connectionId has any filters with the given subscriptionId
func (f Filters) Contains(connectionId, subscriptionId string) bool {
	filters, ok := f[connectionId]
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

// EndSubscription is called when we receive a CLOSE message from a client for a given subscription
func (f Filters) EndSubscription(connectionId, subscriptionId string) {
	filters, ok := f[connectionId]
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
	f[connectionId] = newFilters
	return
}

// EndConnection removes a given connectionId from the map
func (f Filters) EndConnection(connectionId string) {
	delete(f, connectionId)
}

// AddSubscription appends a filter to the given list of filters for a given connectionId
func (f Filters) AddSubscription(connectionId string, subscription *nostr.ReqEnvelope) {
	filters, ok := f[connectionId]
	if !ok {
		f[connectionId] = []*nostr.ReqEnvelope{subscription}
		return
	}
	filters = append(filters, subscription)
	f[connectionId] = filters
}

type FilterManager struct {
	recvFromIngester chan msg.ParsedMsg
	sendToWSHandler  chan msg.Msg
	quit             chan struct{}
	filters          Filters
	dbConn           storage.StorageBackend
	logger           zerolog.Logger
	sync.WaitGroup
}

// NewFilterManager instantiates a new filter manager
func NewFilterManager(recvFromIngester chan msg.ParsedMsg, dbConn storage.StorageBackend, logger zerolog.Logger) *FilterManager {
	return &FilterManager{
		recvFromIngester: recvFromIngester,
		sendToWSHandler:  make(chan msg.Msg),
		quit:             make(chan struct{}),
		dbConn:           dbConn,
		logger:           logger,
	}
}

// Start will start the filter manager
func (f *FilterManager) Start() error {
	f.Add(1)
	go f.manage()
	return nil
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
				// TODO - maybe this should be handled by a filterWorker. But then we need to make this map go-routine safe (i.e. RWMutex)
				// iterate over our map of connections
				for connectionId, filters := range f.filters {
					// iterate over filters
				innerLoop:
					for _, filter := range filters {
						if filter.Match(&envelope.Event) {
							f.sendToWSHandler <- msg.Msg{ConnectionId: connectionId, Data: envelope.Serialize()}
							break innerLoop // We want to break as soon as we have a match to not accidentally send the same client the same event multiple times
						}
					}
				}

			case *nostr.ReqEnvelope:
				// perform db query
				// TODO - This might also need to be a worker go-routine
				for _, filter := range envelope.Filters {
					rcvChan, err := f.dbConn.QueryEvents(context.TODO(), filter)
					if err != nil {
						f.logger.Error().Err(err).Msg("failed to query database for events") // TODO - Should this be fatal?
					}
					f.Add(1)
					go func() {
						defer f.Done()
						timeOut := time.NewTimer(15 * time.Second) // TODO - Make this timeout configurable
						for {
							select {
							case event, ok := <-rcvChan:
								if !ok {
									return
								}
								f.sendToWSHandler <- msg.Msg{ConnectionId: message.ConnectionId, Data: event.Serialize()}
							case <-timeOut.C:
								f.logger.Warn().Str("connectionId", message.ConnectionId).Msgf("timeout reading all messages queried for this filter: %s", filter.String())
								return
							}
						}
					}()
				}
				// check if we have some filters for this connection Id
				if !f.filters.Contains(message.ConnectionId, envelope.SubscriptionID) {
					// add it to our map
					f.filters.AddSubscription(message.ConnectionId, envelope)
				}
			case *nostr.CloseEnvelope:
				if f.filters.Contains(message.ConnectionId, string(*envelope)) {
					// remove it from our map
					f.filters.EndSubscription(message.ConnectionId, string(*envelope))
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
	close(f.sendToWSHandler)
	close(f.quit)
	f.Wait()
	return nil
}
