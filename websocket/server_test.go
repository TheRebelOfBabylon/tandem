package websocket

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"os"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/TheRebelOfBabylon/tandem/test"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

type wsConnMgrTestCase struct {
	name         string
	connId       string
	testSequence func(t *testing.T, clientConn *websocket.Conn, recvChan, sendChan chan msg.Msg, quitChan chan struct{}, quitSignalToServer chan string)
}

var (
	connIdOne = "57be7b62-e93d-4826-9bbf-4fcf91b9bde2"
	connIdTwo = "c34077d4-4e43-490c-aec2-0c9cf11d362d"
	randMsg   = func() []byte {
		input := make([]byte, 32)
		if _, err := rand.Read(input); err != nil {
			panic(err)
		}
		return input
	}
	wsConnMgrTestCases = []wsConnMgrTestCase{
		{
			name:   "Read_CloseViaQuitChan",
			connId: connIdOne,
			testSequence: func(t *testing.T, clientConn *websocket.Conn, recvChan, sendChan chan msg.Msg, quitChan chan struct{}, quitSignalToServer chan string) {
				// send message
				t.Log("sending message over websocket connection...")
				m := randMsg()
				if err := clientConn.WriteMessage(websocket.BinaryMessage, m); err != nil {
					t.Errorf("unexpected error when sending message over websocket connection: %v", err)
				}
				// make sure the message we get is what we expect
				t.Log("waiting for message from websocket connection manager...")
				message, ok := <-sendChan
				if !ok {
					t.Error("websocket connection manager send channel unexpectedely closed")
				}
				if !reflect.DeepEqual(message, msg.Msg{ConnectionId: connIdOne, Data: m}) {
					t.Errorf("received unexpected message from websocket connection manager: expected %v, got %v", msg.Msg{ConnectionId: connIdOne, Data: m}, message)
				}
				t.Log("shutting down websocket connection manager...")
				close(quitChan)
				message, ok = <-sendChan
				if !ok {
					t.Error("websocket connection manager send channel unexpectedely closed")
				}
				if !reflect.DeepEqual(message, msg.Msg{ConnectionId: connIdOne, CloseConn: true}) {
					t.Errorf("received unexpected message from websocket connection manager: expected %v, got %v", msg.Msg{ConnectionId: connIdOne, CloseConn: true}, message)
				}
				// shutdown websocket client
				t.Log("shutting down websocket client...")
				if err := clientConn.Close(); err != nil {
					t.Fatalf("unexpected error when shutting down websocket client: %v", err)
				}
				close(recvChan)
				close(sendChan)
				close(quitSignalToServer)
			},
		},
		{
			name:   "Read_CloseViaWebsocketConnection",
			connId: connIdTwo,
			testSequence: func(t *testing.T, clientConn *websocket.Conn, recvChan, sendChan chan msg.Msg, quitChan chan struct{}, quitSignalToServer chan string) {
				t.Log("shutting down websocket connection manager and websocket connection...")
				if err := clientConn.Close(); err != nil {
					t.Fatalf("unexpected error when shutting down websocket client: %v", err)
				}
				message, ok := <-sendChan
				if !ok {
					t.Error("websocket connection manager send channel unexpectedely closed")
				}
				if !reflect.DeepEqual(message, msg.Msg{ConnectionId: connIdTwo, CloseConn: true}) {
					t.Errorf("received unexpected message from websocket connection manager: expected %v, got %v", msg.Msg{ConnectionId: connIdTwo, CloseConn: true}, message)
				}
				recvConnId, ok := <-quitSignalToServer
				if !ok {
					t.Errorf("quit signal channel from websocket connection manager unexpectedly closed")
				}
				if recvConnId != connIdTwo {
					t.Errorf("received an unexpected connection id from websocket connection manager: expected %s, got %s", connIdTwo, recvConnId)
				}
				close(recvChan)
				close(sendChan)
				close(quitSignalToServer)
			},
		},
		{
			name:   "Write_ValidConnId",
			connId: connIdOne,
			testSequence: func(t *testing.T, clientConn *websocket.Conn, recvChan, sendChan chan msg.Msg, quitChan chan struct{}, quitSignalToServer chan string) {
				randMessage := randMsg()
				recvChan <- msg.Msg{ConnectionId: connIdOne, Data: randMessage}
				// wait for it on the client side
				_, m, err := clientConn.ReadMessage()
				if err != nil {
					t.Errorf("unexpected error reading from websocket client connection: %v", err)
				}
				if !bytes.Equal(m, randMessage) {
					t.Errorf("unexpected message received from websocket client connection: expected %v, got %v", randMessage, m)
				}
				// shutting down client manager and client connection
				t.Log("shutting down websocket connection manager and websocket connection...")
				if err := clientConn.Close(); err != nil {
					t.Fatalf("unexpected error when shutting down websocket client: %v", err)
				}
				message, ok := <-sendChan
				if !ok {
					t.Error("websocket connection manager send channel unexpectedely closed")
				}
				if !reflect.DeepEqual(message, msg.Msg{ConnectionId: connIdOne, CloseConn: true}) {
					t.Errorf("received unexpected message from websocket connection manager: expected %v, got %v", msg.Msg{ConnectionId: connIdOne, CloseConn: true}, message)
				}
				recvConnId, ok := <-quitSignalToServer
				if !ok {
					t.Errorf("quit signal channel from websocket connection manager unexpectedly closed")
				}
				if recvConnId != connIdOne {
					t.Errorf("received an unexpected connection id from websocket connection manager: expected %s, got %s", connIdOne, recvConnId)
				}
				close(recvChan)
				close(sendChan)
				close(quitSignalToServer)
			},
		},
		{
			name:   "Write_InvalidConnId",
			connId: connIdOne,
			testSequence: func(t *testing.T, clientConn *websocket.Conn, recvChan, sendChan chan msg.Msg, quitChan chan struct{}, quitSignalToServer chan string) {
				randMessage := randMsg()
				recvChan <- msg.Msg{ConnectionId: connIdTwo, Data: randMessage}
				// wait for it on the client side
				if err := clientConn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
					t.Fatalf("unexpected error when setting read deadline on client websocket connection: %v", err)
				}
				_, m, err := clientConn.ReadMessage()
				if err == nil && m != nil {
					t.Errorf("receieved an unexpected message over the websocket client connection: %v", m)
				}
				// shutting down client manager and client connection
				t.Log("shutting down websocket connection manager and websocket connection...")
				if err := clientConn.Close(); err != nil {
					t.Fatalf("unexpected error when shutting down websocket client: %v", err)
				}
				message, ok := <-sendChan
				if !ok {
					t.Error("websocket connection manager send channel unexpectedely closed")
				}
				if !reflect.DeepEqual(message, msg.Msg{ConnectionId: connIdOne, CloseConn: true}) {
					t.Errorf("received unexpected message from websocket connection manager: expected %v, got %v", msg.Msg{ConnectionId: connIdOne, CloseConn: true}, message)
				}
				recvConnId, ok := <-quitSignalToServer
				if !ok {
					t.Errorf("quit signal channel from websocket connection manager unexpectedly closed")
				}
				if recvConnId != connIdOne {
					t.Errorf("received an unexpected connection id from websocket connection manager: expected %s, got %s", connIdOne, recvConnId)
				}
				close(recvChan)
				close(sendChan)
				close(quitSignalToServer)
			},
		},
		{
			name:   "Write_CloseRecvChan",
			connId: connIdOne,
			testSequence: func(t *testing.T, clientConn *websocket.Conn, recvChan, sendChan chan msg.Msg, quitChan chan struct{}, quitSignalToServer chan string) {
				close(recvChan)
				message, ok := <-sendChan
				if !ok {
					t.Error("websocket connection manager send channel unexpectedely closed")
				}
				if !reflect.DeepEqual(message, msg.Msg{ConnectionId: connIdOne, CloseConn: true}) {
					t.Errorf("received unexpected message from websocket connection manager: expected %v, got %v", msg.Msg{ConnectionId: connIdOne, CloseConn: true}, message)
				}
				recvConnId, ok := <-quitSignalToServer
				if !ok {
					t.Errorf("quit signal channel from websocket connection manager unexpectedly closed")
				}
				if recvConnId != connIdOne {
					t.Errorf("received an unexpected connection id from websocket connection manager: expected %s, got %s", connIdOne, recvConnId)
				}
				close(sendChan)
				close(quitSignalToServer)
			},
		},
	}
)

// TestWebsocketConnectionManager ensures the behaviour of the websocketConnectionManager is nominal when receiving and sending messages over the websocket connection
func TestWebsocketConnectionManager(t *testing.T) {
	// initialize logger
	mainLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, FormatLevel: test.FormatLvlFunc, TimeLocation: time.UTC}).With().Timestamp().Logger()
	// initialize connection manager
	wsConnMgr := &websocketConnectionManager{
		logger: mainLogger.With().Str("module", "connMgr").Logger(),
	}
	// start http server
	var wg sync.WaitGroup
	srvr := http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				mainLogger.Error().Err(err).Msg("failed to upgrade the HTTP connection to WebSocket connection")
				return
			}
			wsConnMgr.conn = conn
			wg.Add(2)
			go func() {
				defer wg.Done()
				wsConnMgr.read()
			}()
			go func() {
				defer wg.Done()
				wsConnMgr.write()
			}()
		}),
	}
	t.Log("starting websocket server...")
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srvr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			mainLogger.Error().Err(err).Msg("failed to start websocket server")
		}
	}()
	// test cases
	for _, testCase := range wsConnMgrTestCases {
		time.Sleep(2 * time.Second)
		t.Logf("starting test case %s...", testCase.name)
		// initialize connection manager
		wsConnMgr.id = testCase.connId
		wsConnMgr.recv = make(chan msg.Msg)
		wsConnMgr.send = make(chan msg.Msg)
		wsConnMgr.quit = make(chan struct{})
		wsConnMgr.quitSignalToServer = make(chan string)
		wsConnMgr.quitWrite = make(chan struct{})
		// start websocket client
		t.Log("starting websocket client...")
		client, resp, err := websocket.DefaultDialer.Dial("ws://localhost:8080/", nil)
		if err != nil {
			t.Fatalf("failed to initialize websocket client: %v", err)
		} else if resp.StatusCode >= 400 {
			t.Fatalf("failed to initialize websocket client: %s", resp.Status)
		}
		defer resp.Body.Close()
		testCase.testSequence(t, client, wsConnMgr.recv, wsConnMgr.send, wsConnMgr.quit, wsConnMgr.quitSignalToServer)
	}
	t.Log("test completed")
	// shutdown websocket server
	t.Log("shutting down websocket server...")
	if err := srvr.Shutdown(context.Background()); err != nil {
		t.Fatalf("failed to safely shutdown websocket server: %v", err)
	}
	t.Log("waiting for goroutines to shutdown...")
	wg.Wait()
}

type clientConn struct {
	conn   *websocket.Conn
	connId string
}

var wsServerConfig = config.HTTP{
	Host: "localhost",
	Port: 8080,
}

// TestWebsocketServer tests that the new server can accept new connections and pass them off to connection managers, properly relay messages to the correct connection manager and ensure a proper cleanup when shutting down
func TestWebsocketServer(t *testing.T) {
	// initialize logger
	mainLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, FormatLevel: test.FormatLvlFunc, TimeLocation: time.UTC}).With().Timestamp().Logger()
	// init server
	recvFromIngester := make(chan msg.Msg)
	recvFromFilterManager := make(chan msg.Msg)
	srvr := NewWebsocketServer(wsServerConfig, mainLogger.With().Str("module", "websocketServer").Logger(), recvFromIngester, recvFromFilterManager)
	// start server
	if err := srvr.Start(); err != nil {
		t.Fatalf("unexpected error when starting websocket server: %v", err)
	}
	// shutdown server and ensure it's properly cleaned up
	defer func() {
		if err := srvr.Stop(); err != nil {
			t.Errorf("unexpected error when shutting down websocket server: %v", err)
		}
		if sessions := len(srvr.(*WebsocketServer).connMgrChans); sessions > 0 {
			t.Errorf("unexpected %v lingering connection manager sessions", sessions)
		}
		if state := srvr.(*WebsocketServer).isClosing(); !state {
			t.Errorf("websocket server in unexpected state: isClosing? %v", state)
		}
	}()
	time.Sleep(2 * time.Second)
	// create random number (between 5 and 25) simultaneous client conns
	var (
		clients []clientConn
		connIds []string
	)
	for i := 0; i < test.RandRange(5, 25); i++ {
		client, resp, err := websocket.DefaultDialer.Dial("ws://localhost:8080/", nil)
		if err != nil {
			t.Fatalf("failed to initialize websocket client: %v", err)
		} else if resp.StatusCode >= 400 {
			t.Fatalf("failed to initialize websocket client: %s", resp.Status)
		}
		defer resp.Body.Close()
		time.Sleep(1 * time.Second)
		// grab the connId
		for connId := range srvr.(*WebsocketServer).connMgrChans {
			if !slices.Contains(connIds, connId) {
				connIds = append(connIds, connId)
				clients = append(clients, clientConn{conn: client, connId: connId})
				break
			}
		}
	}
	time.Sleep(1 * time.Second)
	// ensure connMgrMap is updated
	if sessions := len(srvr.(*WebsocketServer).connMgrChans); sessions != len(clients) {
		t.Errorf("unexpected number of connection manager sessions: expected %v, got %v", len(clients), sessions)
	}
	// send message and ensure the connId matches
	m := randMsg()
	for _, client := range clients {
		time.Sleep(1 * time.Second)
		if err := client.conn.WriteMessage(websocket.BinaryMessage, m); err != nil {
			t.Errorf("unexpected error when writing to websocket client connection with id %s: %v", client.connId, err)
		}
		message, ok := <-srvr.SendChannel()
		if !ok {
			t.Error("websocket server send channel unexpectedely closed")
		}
		if message.ConnectionId != client.connId {
			t.Errorf("websocket server send channel message has an unexpected connection id: expected %s, got %s", client.connId, message.ConnectionId)
		}
		if !bytes.Equal(message.Data, m) {
			t.Errorf("websocket server send channel message from connection id %s contains unexpected data: expected %v, got %v", client.connId, m, message.Data)
		}
		// send ingester/filterMgr messages with connId and ensure we receive them
		recvFromFilterManager <- msg.Msg{ConnectionId: client.connId, Data: m}
		time.Sleep(1 * time.Second)
		_, wsMsg, err := client.conn.ReadMessage()
		if err != nil {
			t.Errorf("unexpected error receiving message over websocket client connection with id %s: %v", client.connId, err)
		}
		if !bytes.Equal(wsMsg, m) {
			t.Errorf("message received on websocket client connection with id %s contains unexpected data: expected %v, got %v", client.connId, m, wsMsg)
		}
		// TODO - test Unparseable flag
		recvFromIngester <- msg.Msg{ConnectionId: client.connId, Data: m}
		time.Sleep(1 * time.Second)
		_, wsMsg, err = client.conn.ReadMessage()
		if err != nil {
			t.Errorf("unexpected error receiving message over websocket client connection with id %s: %v", client.connId, err)
		}
		if !bytes.Equal(wsMsg, m) {
			t.Errorf("message received on websocket client connection with id %s contains unexpected data: expected %v, got %v", client.connId, m, wsMsg)
		}
		// send ingester/filterMgr messages with different connId and ensure we don't receive them
		randConnId := uuid.NewString()
		recvFromIngester <- msg.Msg{ConnectionId: randConnId, Data: m}
		client.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, wsMsg, err = client.conn.ReadMessage()
		if err == nil && wsMsg != nil {
			t.Errorf("receieved an unexpected message over the websocket client connection with id %s: %v", client.connId, wsMsg)
		}
		// close client connection and ensure it's properly cleaned up
		if err := client.conn.Close(); err != nil {
			t.Fatalf("unexpected error when closing websocket client connection with id %s: %v", client.connId, err)
		}
		message, ok = <-srvr.SendChannel()
		if !ok {
			t.Error("websocket server send channel unexpectedely closed")
		}
		if message.ConnectionId != client.connId {
			t.Errorf("websocket server send channel message has an unexpected connection id: expected %s, got %s", client.connId, message.ConnectionId)
		}
		if !message.CloseConn {
			t.Errorf("websocket server send channel message has unexpected close connection flag state: %v", message.CloseConn)
		}
		time.Sleep(1 * time.Second)
		if _, ok := srvr.(*WebsocketServer).connMgrChans[client.connId]; ok {
			t.Errorf("websocket server connection manager map still contains connection id %s when it shouldn't", client.connId)
		}
	}
}
