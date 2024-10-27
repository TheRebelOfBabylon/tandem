package test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

// FormatLvlFunc formats the log level
func FormatLvlFunc(i interface{}) string {
	return strings.ToUpper(fmt.Sprintf("| %s |", i))
}

// EventBytes takes a nostr event and serializes it
func EventBytes(event nostr.EventEnvelope) []byte {
	eventBytes, err := event.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return eventBytes
}

// OKBytes takes a nostr OK message and serializes it
func OKBytes(ok nostr.OKEnvelope) []byte {
	okBytes, err := ok.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return okBytes
}

// ReqBytes takes a nostr req message and serializes it
func ReqBytes(req nostr.ReqEnvelope) []byte {
	reqBytes, err := req.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return reqBytes
}

// ClosedBytes takes a nostr closed message and serializes it
func ClosedBytes(closed nostr.ClosedEnvelope) []byte {
	closedBytes, err := closed.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return closedBytes
}

// CloseBytes takes a nostr close message and serializes it
func CloseBytes(close nostr.CloseEnvelope) []byte {
	closeBytes, err := close.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return closeBytes
}

// NoticeBytes takes a nostr notice message and serializes it
func NoticeBytes(notice nostr.NoticeEnvelope) []byte {
	noticeBytes, err := notice.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return noticeBytes
}

// CompareEnvelope compares two different nostr envelope messages ensuring they're equal
func CompareEnvelope(t *testing.T, expected, got nostr.Envelope) {
	switch e := expected.(type) {
	case *nostr.EventEnvelope:
		g, ok := got.(*nostr.EventEnvelope)
		if !ok {
			t.Fatalf("got is an unexpected type %T", got)
		}
		CompareEventEnvelope(t, e, g)
	case *nostr.ReqEnvelope:
		g, ok := got.(*nostr.ReqEnvelope)
		if !ok {
			t.Fatalf("got is an unexpected type %T", got)
		}
		CompareReqEnvelope(t, e, g)
	case *nostr.CloseEnvelope:
		g, ok := got.(*nostr.CloseEnvelope)
		if !ok {
			t.Fatalf("got is an unexpected type %T", got)
		}
		if *e != *g {
			t.Errorf("unexpected close message: expected %v, got %v", *e, *g)
		}
	case *nostr.NoticeEnvelope:
		g, ok := got.(*nostr.NoticeEnvelope)
		if !ok {
			t.Fatalf("got is an unexpected type %T", got)
		}
		if *e != *g {
			t.Errorf("unexpected notice message: expected %v, got %v", *e, *g)
		}
	}
}

// CompareReqEnvelope compares two req envelopes and ensures they're equal
func CompareReqEnvelope(t *testing.T, expected, got *nostr.ReqEnvelope) {
	if expected.SubscriptionID != got.SubscriptionID {
		t.Errorf("unexpected subscription id in req message: expected %s, got %s", expected.SubscriptionID, got.SubscriptionID)
	}
	if !reflect.DeepEqual(expected.Filters, got.Filters) {
		t.Errorf("unexpected filters in req message: expected %v, got %v", expected.Filters, got.Filters)
	}
	// if len(expected.Filters) != len(got.Filters) {
	// 	t.Errorf("unexpected number of filter in req message: expected %v, got %v", len(expected.Filters), len(got.Filters))
	// }
	// for i := 0; i < len(expected.Filters); i++ {
	// 	e := expected.Filters[i]
	// 	g := got.Filters[i]
	// 	if slices.Compare(e.IDs, g.IDs) != 0 {
	// 		t.Errorf("unexpected IDs in filter in req message: expected %v, got %v", e.IDs, g.IDs)
	// 	}
	// 	if slices.Compare(e.Authors, g.Authors) != 0 {
	// 		t.Errorf("unexpected authors in filter in req message: expected %v, got %v", e.Authors, g.Authors)
	// 	}
	// 	if slices.Compare(e.Kinds, g.Kinds) != 0 {
	// 		t.Errorf("unexpected kinds in filter in req message: expected %v, got %v", e.Kinds, g.Kinds)
	// 	}
	// 	if e.Limit != g.Limit {
	// 		t.Errorf("unexpected limit in filter in req message: expected %v, got %v", e.Limit, g.Limit)
	// 	}
	// 	if e.LimitZero != g.LimitZero {
	// 		t.Errorf("unexpected limit zero in filter in req message: expected %v, got %v", e.LimitZero, g.LimitZero)
	// 	}
	// 	if e.Search != g.Search {
	// 		t.Errorf("unexpected search field in filter in req message: expected %v, got %v", e.Search, g.Search)
	// 	}
	// 	if e.Since != g.Since {
	// 		t.Errorf("unexpected since field in filter in req message: expected %v, got %v", e.Since, g.Since)
	// 	}
	// 	if !reflect.DeepEqual(e.Tags, g.Tags) {
	// 		t.Errorf("unexpected tags in filter in req message: expected %v, got %v", e.Tags, g.Tags)
	// 	}
	// 	if e.Until != g.Until {
	// 		t.Errorf("unexpected until field in filter in req message: expected %v, got %v", e.Until, g.Until)
	// 	}
	// }
}

// CompareEventEnvelope compares two event envelopes and ensures they're equal
func CompareEventEnvelope(t *testing.T, expected, got *nostr.EventEnvelope) {
	// Compare Subscription ID
	if expected.SubscriptionID != nil && got.SubscriptionID != nil && *expected.SubscriptionID != *got.SubscriptionID {
		t.Errorf("unexpected subscription id in event message: expected %s, got %s", *expected.SubscriptionID, *got.SubscriptionID)
	}
	// Compare event ID
	if expected.ID != got.ID {
		t.Errorf("unexpected event id in event message: expected %s, got %s", expected.ID, got.ID)
	}
	// Compare kind
	if expected.Kind != got.Kind {
		t.Errorf("unexpected event kind in event message: expected %v, got %v", expected.Kind, got.Kind)
	}
	// Compare pubkeys
	if expected.PubKey != got.PubKey {
		t.Errorf("unexpected pubkey in event message: expected %s, got %s", expected.PubKey, got.PubKey)
	}
	// Compare signature
	if expected.Sig != got.Sig {
		t.Errorf("unexpected signature in event message: expected %s, got %s", expected.Sig, got.Sig)
	}
	// Compare content
	if expected.Content != got.Content {
		t.Errorf("unexpected content in event message: expected %s, got %s", expected.Content, got.Content)
	}
	// Compare created_at
	if expected.CreatedAt != got.CreatedAt {
		t.Errorf("unexpected created at time in event message: expected %v, got %v", expected.CreatedAt, got.CreatedAt)
	}
	// Compare tags
	if !reflect.DeepEqual(expected.Tags, got.Tags) {
		t.Errorf("unexpected tags in event message: expected %v, got %v", expected.Tags, got.Tags)
	}
}
