package websocket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/rs/zerolog"
)

type WebsocketServer struct {
	http.Server
	logger            zerolog.Logger
	recvFromIngester  chan msg.Msg
	recvFromFilterMgr chan msg.Msg
	send              chan msg.Msg
	quit              chan struct{}
	connMgrChans      map[string]ConnMgrChannels
	closing           bool
	sync.WaitGroup
}

// NewWebsocketServer instantiates a new HTTP websocket server
func NewWebsocketServer(cfg config.HTTP, logger zerolog.Logger, recvFromIngester, recvFromFilterMgr chan msg.Msg) ConnectionHandler { // TODO - change type to the interface and create a mock for testing
	s := &WebsocketServer{
		Server: http.Server{
			Addr: fmt.Sprintf("%s:%v", cfg.Address, cfg.Port),
		},
		logger:            logger,
		recvFromIngester:  recvFromIngester,
		recvFromFilterMgr: recvFromFilterMgr,
		send:              make(chan msg.Msg),
		quit:              make(chan struct{}),
		connMgrChans:      make(map[string]ConnMgrChannels),
	}
	s.Server.Handler = http.HandlerFunc(s.websocketHandler)
	return s
}

// Start starts the HTTP server to receive websocket connections
func (s *WebsocketServer) Start() error {
	s.Add(2)
	go func() {
		defer s.Done()
		err := s.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			s.logger.Info().Msg("HTTP server safely shutdown")
		} else if err != nil {
			s.logger.Error().Err(err).Msg("failed to safely shutdown HTTP server")
		}
	}()
	go s.recv()
	return nil
}

// Stop safely shuts down the HTTP server
func (s *WebsocketServer) Stop() error {
	s.closing = true
	s.Shutdown(context.TODO())
	close(s.quit)
	for _, chans := range s.connMgrChans {
		close(chans.Quit)
		close(chans.Recv)
	}
	s.Wait()
	close(s.send)
	return nil
}
