//go:build storage

package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/TheRebelOfBabylon/tandem/test"
	"github.com/nbd-wtf/go-nostr"
	"github.com/rs/zerolog"
)

type storageBackendTestCase struct {
	name        string
	inputMsg    msg.ParsedMsg
	expectedErr error
}

var (
	connIdOne    = "5D827151-17DB-4AA9-8DCA-6A3CE71BF3C1"
	defaultEvent = nostr.Event{
		ID:        "4edfccdec007edf614a1a7355260f461ce6f7970b85f479d8f61a13bee83a4f6",
		PubKey:    "44dc1c2db9c3fbd7bee9257eceb52be3cf8c40baf7b63f46e56b58a131c74f0b",
		CreatedAt: 1725319661,
		Kind:      1,
		Tags: nostr.Tags{
			{
				"e",
				"2a8f2f2d4cc831e22695792636169c06f7cb9baea09b9a65c8a870035288283c",
				"",
				"root",
			},
			{
				"p",
				"6140478c9ae12f1d0b540e7c57806649327a91b040b07f7ba3dedc357cab0da5",
			},
		},
		Content: "Lmao. ",
		Sig:     "6da33343f86617acc68654652de083fbf24d86986cfc3cbfc82cd8017f086c5ac2d2b0da4bc48717b98660306bfdda2e2005ae7c8178cb1ee2df751b20326fad",
	}
	defaultNotice           = nostr.NoticeEnvelope("some notice")
	storageBackendTestCases = []storageBackendTestCase{
		{
			name: "ValidCase_NormalEvent",
			inputMsg: msg.ParsedMsg{
				ConnectionId: connIdOne,
				Data: &nostr.EventEnvelope{
					Event: defaultEvent,
				},
			},
			expectedErr: nil,
		},
		{
			name: "InvalidCase_InvalidMessageType",
			inputMsg: msg.ParsedMsg{
				ConnectionId: connIdOne,
				Data:         &defaultNotice,
			},
			expectedErr: ErrUnsupportedMsgType,
		},
	}
)

// TestStorageBackend ensures that the storage backend behaves in an expected way by providing input messages via the storage backend channel and ensuring the response is as expected
func TestStorageBackend(t *testing.T) {
	// initialize logger
	mainLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, FormatLevel: test.FormatLvlFunc, TimeLocation: time.UTC}).With().Timestamp().Logger()
	sendToStorageChan := make(chan msg.ParsedMsg)
	db, err := Connect(TestStorageBackendConfig(), mainLogger.With().Str("module", "storageBackend").Logger(), sendToStorageChan)
	if err != nil {
		t.Fatalf("unexpected error initializing storage backend: %v", err)
	}
	if err := db.Start(); err != nil {
		t.Fatalf("unexpected error when starting storage backend: %v", err)
	}
	defer db.Stop()
	defer func() {
		if err := db.Store.DeleteEvent(context.TODO(), &defaultEvent); err != nil {
			t.Errorf("unexpected error when deleting event from storage: %v", err)
		}
	}()
	// test cases
	for _, testCase := range storageBackendTestCases {
		t.Logf("starting test case %s...", testCase.name)
		dbErrChan := make(chan error)
		testCase.inputMsg.Callback = func(err error) {
			dbErrChan <- err
		}
		// send message to storage backend
		sendToStorageChan <- testCase.inputMsg
		timeout := time.NewTimer(15 * time.Second)
		select {
		case err := <-dbErrChan:
			if err != testCase.expectedErr {
				t.Errorf("unexpected error from storage backend: expected %v, got %v", testCase.expectedErr, err)
			}
		case <-timeout.C:
			t.Errorf("test case %s timed out", testCase.name)
		}
		close(dbErrChan)
	}
}
