package websocket

import (
	"errors"
	"net"
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
	id                 string
	conn               *websocket.Conn
	logger             zerolog.Logger
	recv               chan msg.Msg
	send               chan msg.Msg
	quit               chan struct{}
	quitWrite          chan struct{}
	quitSignalToServer chan string
}

// newWebsocketConnectionManager instantiates a new websocket connection manager
func newWebsocketConnectionManager(
	id string,
	conn *websocket.Conn,
	logger zerolog.Logger,
	send chan msg.Msg,
	recv chan msg.Msg,
	quit chan struct{},
	quitSignalToServer chan string,
) *websocketConnectionManager {
	return &websocketConnectionManager{
		id:                 id,
		conn:               conn,
		logger:             logger,
		send:               send,
		recv:               recv,
		quit:               quit,
		quitWrite:          make(chan struct{}),
		quitSignalToServer: quitSignalToServer,
	}
}

// read is the goroutine responsible for handling new incoming messages from the websocket connection and sending them to the ingester
func (m *websocketConnectionManager) read() {
	defer func() {
		m.send <- msg.Msg{ConnectionId: m.id, CloseConn: true}
		close(m.quitWrite)
	}()
	for {
		_, msgBytes, err := m.conn.ReadMessage()
		if err != nil && (websocket.IsCloseError(err, websocket.CloseAbnormalClosure) || errors.Is(err, net.ErrClosed)) {
			m.logger.Info().Msg("exiting read routine...")
			return
		} else if err != nil {
			m.logger.Error().Err(err).Msg("failed to read message from websocket connection")
			return
		}
		m.send <- msg.Msg{ConnectionId: m.id, Data: msgBytes}
	}
}

// write is the go routine responsible for sending messages over the websocket connection from the ingester
func (m *websocketConnectionManager) write() {
loop:
	for {
		select {
		case msg, ok := <-m.recv:
			if !ok {
				m.logger.Warn().Msg("receive from websocket handler channel unexpectedly closed. closing connection...")
				if err := m.conn.Close(); err != nil {
					m.logger.Error().Err(err).Msg("failed to safely close websocket connection")
				}
				m.quitSignalToServer <- m.id // if you quit without being told, you must notify
				return
			}
			if msg.ConnectionId != m.id {
				m.logger.Warn().Msg("received message destined for a different connection manager. ignoring...")
				continue loop
			}
			m.logger.Debug().Msgf("sending to client: %v", string(msg.Data))
			if err := m.conn.WriteMessage(websocket.TextMessage, msg.Data); err != nil {
				m.logger.Error().Err(err).Msg("failed to send message over websocket connection. closing connection...")
				if err := m.conn.Close(); err != nil {
					m.logger.Error().Err(err).Msg("failed to safely close websocket connection")
				}
				m.quitSignalToServer <- m.id // if you quit without being told, you must notify
				return
			}
			m.logger.Debug().Msg("message sent to client successfully")
		case <-m.quit: // if you are told to quit, no need to notify that you quit
			m.logger.Info().Msg("exiting write routine...")
			if err := m.conn.Close(); err != nil {
				m.logger.Error().Err(err).Msg("failed to safely close websocket connection")
			}
			return
		case <-m.quitWrite: // if you quit without being told, you must notify
			m.logger.Info().Msg("exiting write routine...")
			m.quitSignalToServer <- m.id
			return
		}
	}
}

// websocketHandler is the main handler for the initial request to connect via websockets
func (h *WebsocketServer) websocketHandler(w http.ResponseWriter, r *http.Request) {
	// prevent accepting new connections when shutting down
	if h.isClosing() {
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
	h.connMgrChans[id] = ConnMgrChannels{Recv: recvChan, Quit: quitChan} // TODO - may need to make this map Lockable
	connManager := newWebsocketConnectionManager(
		id,
		conn,
		h.logger.With().Str("connMgr", id).Logger(),
		h.send,
		recvChan,
		quitChan,
		h.quitSignalFromConnMgrs,
	)
	// start up the connection manager
	h.Add(2)
	go func() {
		defer h.Done()
		connManager.read()
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
			h.logger.Debug().Msgf("received from ingester: %v", msg)
			if !ok {
				h.logger.Panic().Msg("receive from ingester channel is unexpectedely closed")
			}
			if msg.Unparseable {
				// TODO - Every connectionId should get three strikes, then they're out. We should also track IP addresses and every IP address also gets three strikes. We should also track pubkeys
			}
			chans, ok := h.connMgrChans[msg.ConnectionId]
			if !ok {
				h.logger.Warn().Msgf("connection manager with id %s not found in receive from ingester routine. Ignoring...", msg.ConnectionId)
				continue loop
			}
			chans.Recv <- msg
		case msg, ok := <-h.recvFromFilterMgr:
			h.logger.Debug().Msgf("received from filter manager: %v", msg)
			if !ok {
				h.logger.Panic().Msg("receive from ingester channel is unexpectedely closed")
			}
			chans, ok := h.connMgrChans[msg.ConnectionId]
			if !ok {
				h.logger.Warn().Msgf("connection manager with id %s not found in receive from filter manager routine. Ignoring...", msg.ConnectionId)
			}
			chans.Recv <- msg
		case connId, ok := <-h.quitSignalFromConnMgrs:
			if !ok {
				h.logger.Panic().Msg("quit channel for signal from connection managers unexpectedely closed")
			}
			if _, ok := h.connMgrChans[connId]; !ok {
				h.logger.Warn().Msgf("received quit signal from connection manager with unknown id %s. ignoring...", connId)
				continue loop
			}
			delete(h.connMgrChans, connId)
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
