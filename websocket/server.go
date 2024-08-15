package websocket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/rs/zerolog"
)

type WebsocketServer struct {
	http.Server
	logger zerolog.Logger
	sync.WaitGroup
}

func NewWebsocketServer(cfg config.HTTP, logger zerolog.Logger) *WebsocketServer { // TODO - change type to the interface and create a mock for testing
	return &WebsocketServer{
		Server: http.Server{
			Addr: fmt.Sprintf("%s:%v", cfg.Address, cfg.Port),
			// Handler: http, // TODO - Combine MainHandler with WebsocketServer
		},
		logger: logger,
	}
}

func (s *WebsocketServer) Start() error {
	s.Add(1)
	go func() {
		defer s.Done()
		err := s.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			s.logger.Info().Msg("HTTP server safely shutdown")
		} else if err != nil {
			s.logger.Error().Err(err).Msg("failed to safely shutdown HTTP server")
		}
	}()
	return nil
}

func (s *WebsocketServer) Stop() error {
	s.Shutdown(context.TODO())
	s.Wait()
	return nil
}
