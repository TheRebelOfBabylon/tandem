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
		t1 := time.Now()
		next.ServeHTTP(o, r)
		t2 := time.Now()
		log.Info().Msgf("[%s] %q %d %v", r.Method, r.URL.String(), o.status, t2.Sub(t1))
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
			if err != nil {
				log.Error().Msgf("%v", err)
				// TODO - Send error message as per NIP-20
				continue
			}
			// if no errors, store and check if active filters require sending it down the pipeline
			log.Debug().Msgf("Parsed Nostr message: %v", msg)
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

	// Run the Nostr protocol overtop the WebSocket connections
	// Listen for key Nostr events
	// Take appropriate actions
	<-interceptor.ShutdownChannel()
	// Graceful shutdown of HTTP server
	httpSrv.Shutdown(ctx)
	wg.Wait()
	return nil
}
