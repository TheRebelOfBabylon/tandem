package websocket

import (
	"net/http"
	"sync"

	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type ConnMgrChannels struct {
	Recv chan msg.Msg
	Quit chan struct{}
}

type MainHandler struct {
	logger       zerolog.Logger
	recv         chan msg.Msg
	send         chan msg.Msg
	quit         chan struct{}
	connMgrChans map[string]ConnMgrChannels
	closing      bool
	sync.WaitGroup
}

type websocketConnectionManager struct {
	id        string
	conn      *websocket.Conn
	logger    zerolog.Logger
	recv      chan msg.Msg
	send      chan msg.Msg
	quit      chan struct{}
	quitWrite chan struct{}
}

// newWebsocketConnectionManager instantiates a new websocket connection manager
func newWebsocketConnectionManager(
	id string,
	conn *websocket.Conn,
	logger zerolog.Logger,
	send chan msg.Msg,
	recv chan msg.Msg,
	quit chan struct{},
) *websocketConnectionManager {
	return &websocketConnectionManager{
		id:        id,
		conn:      conn,
		logger:    logger,
		send:      send,
		recv:      recv,
		quit:      quit,
		quitWrite: make(chan struct{}),
	}
}

// read is the goroutine responsible for handling new incoming messages from the websocket connection and sending them to the ingester
func (m *websocketConnectionManager) read() {
	for {
		select {
		case <-m.quit:
			m.logger.Info().Msg("exiting read routine...")
			close(m.quitWrite)
			return
		default:
			_, msgBytes, err := m.conn.ReadMessage()
			if err != nil {
				m.logger.Err(err).Msg("failed to read message from websocket connection")
				break
			}
			m.send <- msg.Msg{ConnectionId: m.id, Data: msgBytes}
		}
	}
}

// TODO - Handle case when connection is dropped
// write is the go routine responsible for sending messages over the websocket connection from the ingester
func (m *websocketConnectionManager) write() {
loop:
	for {
		select {
		case msg, ok := <-m.recv:
			if !ok {
				// handle
			}
			if msg.ConnectionId != m.id {
				m.logger.Warn().Msg("received message destined for a different connection manager. Ignoring...")
				continue loop
			}
			if err := m.conn.WriteMessage(websocket.TextMessage, msg.Data); err != nil {
				m.logger.Error().Err(err).Msg("failed to send message over websocket connection")
				return
			}
		case <-m.quitWrite:
			m.logger.Info().Msg("exiting write routine...")
			return
		}
	}
}

// NewMainHandler instantiates a new main handler
func NewMainHandler(logger zerolog.Logger, recv chan msg.Msg) *MainHandler {
	return &MainHandler{
		logger:       logger,
		recv:         recv,
		send:         make(chan msg.Msg),
		quit:         make(chan struct{}),
		connMgrChans: make(map[string]ConnMgrChannels),
	}
}

// Start starts the main handler go routines
func (h *MainHandler) Start() error {
	h.closing = false // just doing this to be safe
	go h.recvFromIngester()
	return nil
}

// websocketHandler is the main handler for the initial request to connect via websockets
func (h *MainHandler) websocketHandler(w http.ResponseWriter, r *http.Request) {
	// prevent accepting new connections when shutting down
	if h.closing {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to upgrade the HTTP connection to WebSocket connection")
		return
	}
	// create a new connection manager
	id := uuid.NewString()
	h.logger.Info().Msgf("starting connection manager for new connection with id %s...", id)
	recvChan := make(chan msg.Msg)
	quitChan := make(chan struct{})
	h.connMgrChans[id] = ConnMgrChannels{Recv: recvChan, Quit: quitChan} // may need to make this map Lockable
	connManager := newWebsocketConnectionManager(
		id,
		conn,
		h.logger.With().Str("connMgr", id).Logger(),
		h.send,
		recvChan,
		quitChan,
	)
	// start up the connection manager
	h.Add(2)
	go func() {
		defer h.Done()
		connManager.read() // TODO - maybe this should return an error and we can check if we aren't shutting down, then remove these channels from the map?
	}()
	go func() {
		defer h.Done()
		connManager.write()
	}()
}

// recvFromIngester is a goroutine to receive new messages from the ingester
func (h *MainHandler) recvFromIngester() {
	defer h.Done()
loop:
	for {
		select {
		case msg, ok := <-h.recv:
			if !ok {
				// handle
			}
			chans, ok := h.connMgrChans[msg.ConnectionId]
			if !ok {
				h.logger.Warn().Msgf("connection manager with id %s not found in receive from ingester routine. Ignoring...", msg.ConnectionId)
				continue loop
			}
			chans.Recv <- msg
		case <-h.quit:
			h.logger.Info().Msg("exiting receive from ingester routine...")
			return
		}
	}
}

// SendChannel is a getter function to get the websocket handlers send channel
func (h *MainHandler) SendChannel() chan msg.Msg {
	return h.send
}

// Close safely shutsdown the main websocket handler
func (h *MainHandler) Close() error {
	h.closing = true
	close(h.quit) // shutdown receive from ingester routine first since it will send down connection manager channels
	// stop connection managers
	for _, chans := range h.connMgrChans {
		close(chans.Quit)
		close(chans.Recv)
	}
	h.Wait()
	close(h.send) // shut this down last to be safe
	return nil
}
