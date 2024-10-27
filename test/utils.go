package test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

func FormatLvlFunc(i interface{}) string {
	return strings.ToUpper(fmt.Sprintf("| %s |", i))
}

func EventBytes(event nostr.EventEnvelope) []byte {
	eventBytes, err := event.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return eventBytes
}

func OKBytes(ok nostr.OKEnvelope) []byte {
	okBytes, err := ok.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return okBytes
}

func CompareEnvelope(t *testing.T, expected, got nostr.Envelope) {

}

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
