package ingester

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/TheRebelOfBabylon/tandem/test"
	"github.com/nbd-wtf/go-nostr"
	"github.com/rs/zerolog"
)

type ingesterTestCase struct {
	name                 string
	inputRawMsg          msg.Msg
	expectedMsg          *msg.Msg
	expectedDbMsg        *msg.ParsedMsg
	dbCallbackHandler    func(callback func(err error))
	expectedFilterMgrMsg *msg.ParsedMsg
}

var (
	connIdOne          = "8f1899dc-59ea-40c8-831c-85cb68a1e323"
	defaultEventBadSig = nostr.Event{
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
		Sig:     "6da33343f86617acc68654652de083fbf24d86986cfc3cbfc83cd8017f086c5ac2d2b0da4bc48717b98660306bfdda2e2005ae7c8178cb1ee2df751b20326fad",
	}
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
	largeSubId           = "4ezw1s6zwl6zgp96t4m5haxsc8kp007py229f6gtlzwmyrkrvhqtlmu7fudvtcpkc"
	defaultReqLargeSubId = nostr.ReqEnvelope{
		SubscriptionID: largeSubId,
		Filters: nostr.Filters{
			{
				Kinds: []int{1},
			},
		},
	}
	subIdOne   = "a64ea55b-1cb2-42a1-9d30-e1d2bc7074d0"
	defaultReq = nostr.ReqEnvelope{
		SubscriptionID: subIdOne,
		Filters: nostr.Filters{
			{
				Kinds: []int{1},
				Tags:  nostr.TagMap{},
			},
		},
	}
	defaultClose      = nostr.CloseEnvelope(subIdOne)
	ingesterTestCases = []ingesterTestCase{
		{
			name: "ValidCase_ClosedConn",
			inputRawMsg: msg.Msg{
				ConnectionId: connIdOne,
				CloseConn:    true,
			},
			expectedFilterMgrMsg: &msg.ParsedMsg{
				ConnectionId: connIdOne,
				CloseConn:    true,
			},
		},
		{
			name: "ValidCase_Event_InvalidSignature",
			inputRawMsg: msg.Msg{
				ConnectionId: connIdOne,
				Data: test.EventBytes(nostr.EventEnvelope{
					Event: defaultEventBadSig,
				}),
			},
			expectedMsg: &msg.Msg{
				ConnectionId: connIdOne,
				Data: test.OKBytes(nostr.OKEnvelope{
					EventID: "4edfccdec007edf614a1a7355260f461ce6f7970b85f479d8f61a13bee83a4f6",
					OK:      false,
					Reason:  "error: invalid event signature or event id",
				}),
			},
		},
		{
			name: "ValidCase_Event_DbError",
			inputRawMsg: msg.Msg{
				ConnectionId: connIdOne,
				Data: test.EventBytes(nostr.EventEnvelope{
					Event: defaultEvent,
				}),
			},
			expectedDbMsg: &msg.ParsedMsg{
				ConnectionId: connIdOne,
				Data: &nostr.EventEnvelope{
					Event: defaultEvent,
				},
			},
			dbCallbackHandler: func(callback func(err error)) {
				callback(errors.New("some db related error"))
			},
			expectedMsg: &msg.Msg{
				ConnectionId: connIdOne,
				Data: test.OKBytes(nostr.OKEnvelope{
					EventID: "4edfccdec007edf614a1a7355260f461ce6f7970b85f479d8f61a13bee83a4f6",
					OK:      false,
					Reason:  "error: failed to store event",
				}),
			},
		},
		{
			name: "ValidCase_Event_NoDbErr",
			inputRawMsg: msg.Msg{
				ConnectionId: connIdOne,
				Data: test.EventBytes(nostr.EventEnvelope{
					Event: defaultEvent,
				}),
			},
			expectedDbMsg: &msg.ParsedMsg{
				ConnectionId: connIdOne,
				Data: &nostr.EventEnvelope{
					Event: defaultEvent,
				},
			},
			dbCallbackHandler: func(callback func(err error)) {
				callback(nil)
			},
			expectedMsg: &msg.Msg{
				ConnectionId: connIdOne,
				Data: test.OKBytes(nostr.OKEnvelope{
					EventID: "4edfccdec007edf614a1a7355260f461ce6f7970b85f479d8f61a13bee83a4f6",
					OK:      true,
				}),
			},
			expectedFilterMgrMsg: &msg.ParsedMsg{
				ConnectionId: connIdOne,
				Data: &nostr.EventEnvelope{
					Event: defaultEvent,
				},
			},
		},
		{
			name: "ValidCase_Event_DbTimeout",
			inputRawMsg: msg.Msg{
				ConnectionId: connIdOne,
				Data: test.EventBytes(nostr.EventEnvelope{
					Event: defaultEvent,
				}),
			},
			expectedDbMsg: &msg.ParsedMsg{
				ConnectionId: connIdOne,
				Data: &nostr.EventEnvelope{
					Event: defaultEvent,
				},
			},
			expectedMsg: &msg.Msg{
				ConnectionId: connIdOne,
				Data: test.OKBytes(nostr.OKEnvelope{
					EventID: "4edfccdec007edf614a1a7355260f461ce6f7970b85f479d8f61a13bee83a4f6",
					OK:      false,
					Reason:  "error: failed to store event",
				}),
			},
		},
		{
			name: "ValidCase_Req_SubscriptionIDTooLarge",
			inputRawMsg: msg.Msg{
				ConnectionId: connIdOne,
				Data:         test.ReqBytes(defaultReqLargeSubId),
			},
			expectedMsg: &msg.Msg{
				ConnectionId: connIdOne,
				Data: test.ClosedBytes(nostr.ClosedEnvelope{
					SubscriptionID: largeSubId,
					Reason:         "error: subscription id exceeds 64 character limit",
				}),
			},
		},
		{
			name: "ValidCase_Req",
			inputRawMsg: msg.Msg{
				ConnectionId: connIdOne,
				Data:         test.ReqBytes(defaultReq),
			},
			expectedFilterMgrMsg: &msg.ParsedMsg{
				ConnectionId: connIdOne,
				Data:         &defaultReq,
			},
		},
		{
			name: "ValidCase_Close",
			inputRawMsg: msg.Msg{
				ConnectionId: connIdOne,
				Data:         test.CloseBytes(defaultClose),
			},
			expectedFilterMgrMsg: &msg.ParsedMsg{
				ConnectionId: connIdOne,
				Data:         &defaultClose,
			},
		},
		{
			name: "ValidCase_InvalidMessage",
			inputRawMsg: msg.Msg{
				ConnectionId: connIdOne,
				Data:         []byte("some invalid input from a griefer"),
			},
			expectedMsg: &msg.Msg{
				ConnectionId: connIdOne,
				Data:         test.NoticeBytes(nostr.NoticeEnvelope("error: failed to parse message and continued failure to parse future messages will result in a ban")),
				Unparseable:  true,
			},
		},
	}
)

// TestIngester ensures the ingester behaves in an expected manner by iterating through various test cases
func TestIngester(t *testing.T) {
	// initialize logger
	mainLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, FormatLevel: test.FormatLvlFunc, TimeLocation: time.UTC}).With().Timestamp().Logger()
	// initialize ingester
	ingester := NewIngester(mainLogger.With().Str("module", "ingester").Logger())
	if err := ingester.Start(); err != nil {
		t.Fatalf("unexpected error when starting ingester: %v", err)
	}
	defer ingester.Stop()
	// set the recv channel
	fromWSChan := make(chan msg.Msg)
	defer close(fromWSChan)
	ingester.SetRecvChannel(fromWSChan)
	// grab the channels
	dbChan := ingester.SendToDBChannel()
	filterMgrChan := ingester.SendToFilterManager()
	wsChan := ingester.SendToWSHandlerChannel()

	// test cases
	for _, testCase := range ingesterTestCases {
		t.Logf("starting test case %s...", testCase.name)
		// send the input message
		fromWSChan <- testCase.inputRawMsg
		// wait for expected messages
		if testCase.expectedDbMsg != nil {
			t.Log("waiting for message on storage backend channel...")
			timeout := time.NewTimer(15 * time.Second)
			select {
			case parsedMsg, ok := <-dbChan:
				if !ok {
					t.Error("storage channel unexpectedely closed")
				}
				if testCase.expectedDbMsg.ConnectionId != parsedMsg.ConnectionId {
					t.Errorf("unexpected connection ID in message from ingester to storage backend: expected %s, got %s", testCase.expectedDbMsg.ConnectionId, parsedMsg.ConnectionId)
				}
				if testCase.expectedDbMsg.CloseConn != parsedMsg.CloseConn {
					t.Errorf("unexpected close connection state from ingester to storage backend message: expected %v, got %v", testCase.expectedDbMsg.CloseConn, parsedMsg.CloseConn)
				}
				// use the callback to send an error if we want to do this
				if parsedMsg.Callback != nil && testCase.dbCallbackHandler != nil {
					testCase.dbCallbackHandler(parsedMsg.Callback)
				}
				test.CompareEventEnvelope(t, testCase.expectedDbMsg.Data.(*nostr.EventEnvelope), parsedMsg.Data.(*nostr.EventEnvelope))
			case <-timeout.C:
				t.Error("timed out waiting for message on storage channel")
			}
		}
		if testCase.expectedMsg != nil {
			t.Log("waiting for message on websocket channel...")
			timeout := time.NewTimer(15 * time.Second)
			select {
			case message, ok := <-wsChan:
				if !ok {
					t.Error("websocket channel unexpectedely closed")
				}
				if !reflect.DeepEqual(*testCase.expectedMsg, message) {
					t.Errorf("unexpected message from ingester to websocket manager: expected %v, got %v", *testCase.expectedMsg, message)
				}
			case <-timeout.C:
				t.Error("timed out waiting for message on websocket channel")
			}
		}
		if testCase.expectedFilterMgrMsg != nil {
			t.Log("waiting for message on filter manager channel...")
			timeout := time.NewTimer(15 * time.Second)
			select {
			case parsedMsg, ok := <-filterMgrChan:
				if !ok {
					t.Error("filter manager channel unexpectedely closed")
				}
				if testCase.expectedFilterMgrMsg.ConnectionId != parsedMsg.ConnectionId {
					t.Errorf("unexpected connection id from ingester to filter manager message: expected %s, got %s", testCase.expectedFilterMgrMsg.ConnectionId, parsedMsg.ConnectionId)
				}
				if testCase.expectedFilterMgrMsg.CloseConn != parsedMsg.CloseConn {
					t.Errorf("unexpected close connection state from ingester to filter manager message: expected %v, got %v", testCase.expectedFilterMgrMsg.CloseConn, parsedMsg.CloseConn)
				}
				test.CompareEnvelope(t, testCase.expectedFilterMgrMsg.Data, parsedMsg.Data)
			case <-timeout.C:
				t.Error("timed out waiting for message on filter manager channel")
			}
		}
	}
	t.Log("completed test")
}

type ingesterReplaceableEventTestCase struct {
	name                 string
	inputEventKind       int
	inputDTag            string
	connId               string
	expectedOkMsg        *nostr.OKEnvelope
	expectedDbMsgs       []msg.ParsedMsg
	dbCallbackHandlers   []func(callback func(err error))
	expectedFilterMgrMsg *msg.ParsedMsg
	queryFunc            func(context.Context, nostr.Filter) (chan *nostr.Event, error)
}

var ingesterReplaceableEventsTestCases = []ingesterReplaceableEventTestCase{
	{
		name:           "ValidCase_Event_RepleaceableEvent",
		inputEventKind: 14567,
		connId:         connIdOne,
		expectedOkMsg: &nostr.OKEnvelope{
			OK: true,
		},
		expectedDbMsgs: []msg.ParsedMsg{
			{
				ConnectionId: connIdOne,
			},
		},
		dbCallbackHandlers: []func(callback func(err error)){
			func(callback func(err error)) {
				callback(nil)
			},
		},
		expectedFilterMgrMsg: &msg.ParsedMsg{
			ConnectionId: connIdOne,
		},
		queryFunc: func(ctx context.Context, f nostr.Filter) (chan *nostr.Event, error) {
			queryChan := make(chan *nostr.Event)
			go func() {
				close(queryChan)
			}()
			return queryChan, nil
		},
	},
	{
		name:           "ErrorCase_Event_RepleaceableEvent_QueryFuncError",
		connId:         connIdOne,
		inputEventKind: 14567,
		expectedOkMsg: &nostr.OKEnvelope{
			OK:     false,
			Reason: "error: failed to query storage for any existing replaceable events: rrrr matey this be an error",
		},
		queryFunc: func(ctx context.Context, f nostr.Filter) (chan *nostr.Event, error) {
			return nil, errors.New("rrrr matey this be an error")
		},
	},
	{
		name:           "ErrorCase_Event_RepleaceableEvent_DeleteOldEventError",
		connId:         connIdOne,
		inputEventKind: 14567,
		expectedDbMsgs: []msg.ParsedMsg{
			{
				ConnectionId: connIdOne,
				DeleteEvent:  true,
			},
		},
		dbCallbackHandlers: []func(callback func(err error)){
			func(callback func(err error)) {
				callback(errors.New("rrrr matey this be an error"))
			},
		},
		expectedOkMsg: &nostr.OKEnvelope{
			OK:     false,
			Reason: "error: failed to delete stale replaceable events: rrrr matey this be an error",
		},
		queryFunc: func(ctx context.Context, f nostr.Filter) (chan *nostr.Event, error) {
			queryChan := make(chan *nostr.Event)
			go func() {
				queryChan <- &nostr.Event{ID: "abc123", Kind: 14567}
				close(queryChan)
			}()
			return queryChan, nil
		},
	},
	{
		name:           "ValidCase_Event_RepleaceableEvent_DeleteOldEvent",
		connId:         connIdOne,
		inputEventKind: 14567,
		expectedDbMsgs: []msg.ParsedMsg{
			{
				ConnectionId: connIdOne,
				DeleteEvent:  true,
			},
			{
				ConnectionId: connIdOne,
			},
		},
		dbCallbackHandlers: []func(callback func(err error)){
			func(callback func(err error)) {
				callback(nil)
			},
			func(callback func(err error)) {
				callback(nil)
			},
		},
		expectedOkMsg: &nostr.OKEnvelope{
			OK: true,
		},
		queryFunc: func(ctx context.Context, f nostr.Filter) (chan *nostr.Event, error) {
			queryChan := make(chan *nostr.Event)
			go func() {
				queryChan <- &nostr.Event{ID: "abc123", Kind: 14567}
				close(queryChan)
			}()
			return queryChan, nil
		},
		expectedFilterMgrMsg: &msg.ParsedMsg{
			ConnectionId: connIdOne,
		},
	},
	{
		name:           "ValidCase_Event_ParametrizedRepleaceableEvent",
		inputEventKind: 34567,
		inputDTag:      "hello",
		connId:         connIdOne,
		expectedOkMsg: &nostr.OKEnvelope{
			OK: true,
		},
		expectedDbMsgs: []msg.ParsedMsg{
			{
				ConnectionId: connIdOne,
			},
		},
		dbCallbackHandlers: []func(callback func(err error)){
			func(callback func(err error)) {
				callback(nil)
			},
		},
		expectedFilterMgrMsg: &msg.ParsedMsg{
			ConnectionId: connIdOne,
		},
		queryFunc: func(ctx context.Context, f nostr.Filter) (chan *nostr.Event, error) {
			queryChan := make(chan *nostr.Event)
			go func() {
				close(queryChan)
			}()
			return queryChan, nil
		},
	},
	{
		name:           "ErrorCase_Event_ParametrizedRepleaceableEvent_QueryFuncError",
		connId:         connIdOne,
		inputEventKind: 34567,
		inputDTag:      "hello",
		expectedOkMsg: &nostr.OKEnvelope{
			OK:     false,
			Reason: "error: failed to query storage for any existing replaceable events: rrrr matey this be an error",
		},
		queryFunc: func(ctx context.Context, f nostr.Filter) (chan *nostr.Event, error) {
			return nil, errors.New("rrrr matey this be an error")
		},
	},
	{
		name:           "ErrorCase_Event_ParametrizedRepleaceableEvent_DeleteOldEventError",
		connId:         connIdOne,
		inputEventKind: 34567,
		inputDTag:      "hello",
		expectedDbMsgs: []msg.ParsedMsg{
			{
				ConnectionId: connIdOne,
				DeleteEvent:  true,
			},
		},
		dbCallbackHandlers: []func(callback func(err error)){
			func(callback func(err error)) {
				callback(errors.New("rrrr matey this be an error"))
			},
		},
		expectedOkMsg: &nostr.OKEnvelope{
			OK:     false,
			Reason: "error: failed to delete stale replaceable events: rrrr matey this be an error",
		},
		queryFunc: func(ctx context.Context, f nostr.Filter) (chan *nostr.Event, error) {
			queryChan := make(chan *nostr.Event)
			go func() {
				queryChan <- &nostr.Event{ID: "abc123", Kind: 34567, Tags: nostr.Tags{{"d", "hello"}}}
				close(queryChan)
			}()
			return queryChan, nil
		},
	},
	{
		name:           "ValidCase_Event_ParametrizedRepleaceableEvent_DeleteOldEvent",
		connId:         connIdOne,
		inputEventKind: 34567,
		inputDTag:      "hello",
		expectedDbMsgs: []msg.ParsedMsg{
			{
				ConnectionId: connIdOne,
				DeleteEvent:  true,
			},
			{
				ConnectionId: connIdOne,
			},
		},
		dbCallbackHandlers: []func(callback func(err error)){
			func(callback func(err error)) {
				callback(nil)
			},
			func(callback func(err error)) {
				callback(nil)
			},
		},
		expectedOkMsg: &nostr.OKEnvelope{
			OK: true,
		},
		queryFunc: func(ctx context.Context, f nostr.Filter) (chan *nostr.Event, error) {
			queryChan := make(chan *nostr.Event)
			go func() {
				queryChan <- &nostr.Event{ID: "abc123", Kind: 34567, Tags: nostr.Tags{{"d", "hello"}}}
				close(queryChan)
			}()
			return queryChan, nil
		},
		expectedFilterMgrMsg: &msg.ParsedMsg{
			ConnectionId: connIdOne,
		},
	},
	{
		name:           "ValidCase_EphemeralEvent",
		connId:         connIdOne,
		inputEventKind: 23456,
		expectedOkMsg: &nostr.OKEnvelope{
			OK: true,
		},
		queryFunc: func(ctx context.Context, f nostr.Filter) (chan *nostr.Event, error) {
			return nil, nil
		},
		expectedFilterMgrMsg: &msg.ParsedMsg{
			ConnectionId: connIdOne,
		},
	},
}

// TestIngesterReplaceableEvents ensures the ingester handles ephemeral and replaceable events as expected
func TestIngesterReplaceableEvents(t *testing.T) {
	// initialize logger
	mainLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, FormatLevel: test.FormatLvlFunc, TimeLocation: time.UTC}).With().Timestamp().Logger()
	// initialize ingester
	ingester := NewIngester(mainLogger.With().Str("module", "ingester").Logger())
	if err := ingester.Start(); err != nil {
		t.Fatalf("unexpected error when starting ingester: %v", err)
	}
	defer ingester.Stop()
	// set the recv channel
	fromWSChan := make(chan msg.Msg)
	defer close(fromWSChan)
	ingester.SetRecvChannel(fromWSChan)
	// grab the channels
	dbChan := ingester.SendToDBChannel()
	filterMgrChan := ingester.SendToFilterManager()
	wsChan := ingester.SendToWSHandlerChannel()
	// test cases
	for _, testCase := range ingesterReplaceableEventsTestCases {
		t.Logf("starting test case %s...", testCase.name)
		// set the query func
		ingester.SetQueryFunc(testCase.queryFunc)
		// create the event
		event := &nostr.EventEnvelope{Event: test.CreateRandomEvent(test.UseKind(testCase.inputEventKind), test.AppendTags(nostr.TagMap{"d": []string{testCase.inputDTag}}))}
		eventBytes, err := event.MarshalJSON()
		if err != nil {
			t.Fatalf("failed to JSON encode event: %v", err)
		}
		// send the input message
		fromWSChan <- msg.Msg{ConnectionId: testCase.connId, Data: eventBytes}
		// wait for expected messages
		if testCase.expectedDbMsgs != nil {
			t.Log("waiting for messages on storage backend channel...")
			for i, expectedDbMsg := range testCase.expectedDbMsgs {
				timeout := time.NewTimer(15 * time.Second)
				select {
				case parsedMsg, ok := <-dbChan:
					if !ok {
						t.Error("storage channel unexpectedely closed")
					}
					if testCase.connId != parsedMsg.ConnectionId {
						t.Errorf("unexpected connection ID in message from ingester to storage backend: expected %s, got %s", expectedDbMsg.ConnectionId, parsedMsg.ConnectionId)
					}
					if expectedDbMsg.CloseConn != parsedMsg.CloseConn {
						t.Errorf("unexpected close connection state from ingester to storage backend message: expected %v, got %v", expectedDbMsg.CloseConn, parsedMsg.CloseConn)
					}
					if expectedDbMsg.DeleteEvent != parsedMsg.DeleteEvent {
						t.Errorf("unexpected delete event state from ingester to storage backend message: expected %v, got %v", expectedDbMsg.DeleteEvent, parsedMsg.DeleteEvent)
					}
					// use the callback to send an error if we want to do this
					if parsedMsg.Callback != nil && testCase.dbCallbackHandlers != nil {
						testCase.dbCallbackHandlers[i](parsedMsg.Callback)
					}
					if !parsedMsg.DeleteEvent {
						test.CompareEventEnvelope(t, event, parsedMsg.Data.(*nostr.EventEnvelope))
					}
				case <-timeout.C:
					t.Error("timed out waiting for message on storage channel")

				}
			}
		}
		if testCase.expectedOkMsg != nil {
			t.Log("waiting for message on websocket channel...")
			timeout := time.NewTimer(15 * time.Second)
			select {
			case message, ok := <-wsChan:
				if !ok {
					t.Error("websocket channel unexpectedely closed")
				}
				testCase.expectedOkMsg.EventID = event.ID
				okBytes, err := testCase.expectedOkMsg.MarshalJSON()
				if err != nil {
					t.Fatalf("unexpected message when JSON marshalling OK message: %v", err)
				}
				expectedMsg := msg.Msg{ConnectionId: testCase.connId, Data: okBytes}
				if !reflect.DeepEqual(expectedMsg, message) {
					t.Errorf("unexpected message from ingester to websocket manager: expected %v, got %v", expectedMsg, message)
					t.Logf("expected %s, got %s", string(expectedMsg.Data), string(message.Data))
				}
			case <-timeout.C:
				t.Error("timed out waiting for message on websocket channel")
			}
		}
		if testCase.expectedFilterMgrMsg != nil {
			t.Log("waiting for message on filter manager channel...")
			timeout := time.NewTimer(15 * time.Second)
			select {
			case parsedMsg, ok := <-filterMgrChan:
				if !ok {
					t.Error("filter manager channel unexpectedely closed")
				}
				if testCase.expectedFilterMgrMsg.ConnectionId != parsedMsg.ConnectionId {
					t.Errorf("unexpected connection id from ingester to filter manager message: expected %s, got %s", testCase.expectedFilterMgrMsg.ConnectionId, parsedMsg.ConnectionId)
				}
				if testCase.expectedFilterMgrMsg.CloseConn != parsedMsg.CloseConn {
					t.Errorf("unexpected close connection state from ingester to filter manager message: expected %v, got %v", testCase.expectedFilterMgrMsg.CloseConn, parsedMsg.CloseConn)
				}
				test.CompareEnvelope(t, event, parsedMsg.Data)
			case <-timeout.C:
				t.Error("timed out waiting for message on filter manager channel")
			}
		}
	}
	t.Log("completed test")
}
