package websocket

import (
	"errors"
	"net/http"

	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	ErrRecvChanNotSet = errors.New("receive channel not set")
)

type ConnMgrChannels struct {
	Recv chan msg.Msg
	Quit chan struct{}
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

// websocketHandler is the main handler for the initial request to connect via websockets
func (h *WebsocketServer) websocketHandler(w http.ResponseWriter, r *http.Request) {
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

// recv is a goroutine to receive new messages from the ingester and filter manager
func (h *WebsocketServer) recv() {
	defer h.Done()
	if h.recvFromIngester == nil || h.recvFromFilterMgr == nil {
		h.logger.Error().Err(ErrRecvChanNotSet).Msg("failed to start receive routine")
		return
	}
loop:
	for {
		select {
		case msg, ok := <-h.recvFromIngester:
			if !ok {
				// TODO - handle
			}
			chans, ok := h.connMgrChans[msg.ConnectionId]
			if !ok {
				h.logger.Warn().Msgf("connection manager with id %s not found in receive from ingester routine. Ignoring...", msg.ConnectionId)
				continue loop
			}
			chans.Recv <- msg
		case msg, ok := <-h.recvFromFilterMgr:
			if !ok {
				// TODO - handle
			}
			chans, ok := h.connMgrChans[msg.ConnectionId]
			if !ok {
				h.logger.Warn().Msgf("connection manager with id %s not found in receive from filter manager routine. Ignoring...", msg.ConnectionId)
			}
			chans.Recv <- msg
		case <-h.quit:
			h.logger.Info().Msg("exiting receive from ingester routine...")
			return
		}
	}
}

// SendChannel is a getter function to get the websocket handlers send channel
func (h *WebsocketServer) SendChannel() chan msg.Msg {
	return h.send
}
