package tandem

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/SSSOC-CAN/laniakea/intercept"
	bg "github.com/SSSOCPaulCote/blunderguard"
	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/db"
	"github.com/TheRebelOfBabylon/tandem/nostr"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

const (
	ErrHijackUnsupported = bg.Error("hijack not supported")
)

var (
	upgrader                    = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	HTTPgracefulShutdownTimeout = 10 * time.Second
)

// loggingHandler is an HTTP handler to log HTTP method, URL, status code and time to perform the wrapped handler
func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request, log zerolog.Logger, dbClient *db.MongoDB, p *nostr.SafeParser) {
		o := &ResponseObserver{ResponseWriter: w}
		next.ServeHTTP(o, r)
		log.Info().Msgf("[%s] %q %d", r.Method, r.URL.String(), o.status)
	}
	ch, _ := next.(*CustomHandler)
	return &CustomHandler{Logger: ch.Logger, handlerFunc: fn, Client: ch.Client, Parser: ch.Parser}

}

// newConnectionHandler handles the incoming HTTP 1.0 request and upgrades it to establish the WebSocket connection
func newConnectionHandler(w http.ResponseWriter, r *http.Request, log zerolog.Logger, dbClient *db.MongoDB, p *nostr.SafeParser) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Msgf("Error when upgrading to WebSocket connection: %v", err)
		return
	}
	defer conn.Close()
	log.Debug().Msgf("Client IP: %v", conn.RemoteAddr().String())
loop:
	for {
		// TODO - Add rate limiting
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Error().Msgf("Error during message reading: %v", err)
			break
		}
		switch messageType {
		case websocket.TextMessage:
			// let's validate it's a Nostr message
			msg, err := nostr.ParseAndValidateNostr(message, p)
			// TODO - Add an error counter and end connection with clients sending broken messages
			switch m := msg.(type) {
			case nostr.Event:
				// Handle parsing error
				if err != nil {
					log.Error().Msgf("error parsing nostr message: %v", err)
					errMsg := fmt.Sprintf("[\"NOTICE\", \"%v\"]", err)
					if err == nostr.ErrNoContent || err == nostr.ErrInvalidSig || err == nostr.ErrMissingField { // TODO - Add checking for invalid created_at field
						errMsg = fmt.Sprintf("[\"OK\", \"%s\", %v, \"invalid: %v\"]", m.EventId, false, err)
					}
					err = conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
					if err != nil {
						log.Error().Msgf("error sending command result: %v", err)
						break loop
					}
					continue loop
				}
				log.Debug().Msgf("Parsed Nostr message: %v", m)
				// Write to db
				err = dbClient.InsertEvent(context.TODO(), m)
				if err != nil {
					log.Error().Msgf("error writing to database: %v", err)
					errMsg := fmt.Sprintf("[\"OK\", \"%s\", %v, \"error: %v\"]", m.EventId, false, err)
					if err == db.ErrEventDuplicate {
						errMsg = fmt.Sprintf("[\"OK\", \"%s\", %v, \"duplicate: %v\"]", m.EventId, true, err)
					}
					err = conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
					if err != nil {
						log.Error().Msgf("error sending command result: %v", err)
						break loop
					}
					continue loop
				}
				// Send OK response
				err = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("[\"OK\", \"%s\", %v, \"\"]", m.EventId, true)))
				if err != nil {
					log.Error().Msgf("error sending command result: %v", err)
					break loop
				}
				//case nostr.Request:
				//case nostr.Close:
			}
		case websocket.CloseMessage:
			log.Debug().Msgf("closing connection to %v...", conn.RemoteAddr().String())
			break loop
		case websocket.PingMessage:
			log.Debug().Msg("received ping, sending pong...")
			err = conn.WriteMessage(websocket.PongMessage, []byte(""))
			if err != nil {
				log.Error().Msgf("error sending pong: %v", err)
				break loop
			}
		case websocket.PongMessage:
			// TODO - Add Pong timeout and resets
			continue
		}
	}
}

// Main is the true entrypoint of tandem
func Main(cfg *config.Config, log zerolog.Logger, interceptor *intercept.Interceptor) error {
	var wg sync.WaitGroup
	ctx := context.Background()

	// Connect to the database
	conn, err := db.NewDBConnection(ctx, &cfg.Database)
	if err != nil {
		log.Error().Msgf("could not connect to db: %v", err)
		return err
	}

	// Create goroutine safe, fastjson parser
	var p nostr.SafeParser

	// We need to start the WebSocket server
	// Define the gorilla mux router
	router := mux.NewRouter()
	withLogger := CustomHandlerFactory(log.With().Str("subsystem", "HTTP").Logger(), conn, &p)
	router.Handle("/", withLogger(newConnectionHandler))
	router.Use(loggingHandler)
	httpSrv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("%s:%v", cfg.Network.BindAddress, cfg.Network.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := httpSrv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Error().Msgf("Unexpected error when stopping HTTP server: %v", err)
		}
	}()
	<-interceptor.ShutdownChannel()
	// Graceful shutdown of HTTP server
	httpSrv.Shutdown(ctx)
	wg.Wait()
	return nil
}
