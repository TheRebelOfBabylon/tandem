package test

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

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

// CreateRandomKeypair generates a random key pair
func CreateRandomKeypair() (string, string, error) {
	sk := nostr.GeneratePrivateKey()
	pk, err := nostr.GetPublicKey(sk)
	if err != nil {
		return "", "", err
	}
	return sk, pk, nil
}

// RandRange creates a random number from a given range
func RandRange(min, max int) int {
	return rand.Intn(int(math.Max(float64(max-min), 1))) + min
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789:/.%&?#@!")

// randStr creates a random string of length n
func randStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// CreateRandomTags creates random nostr tags of a given total length and where each tag is up to maxTagLen in length
func CreateRandomTags(totalLen, maxTagLen int) nostr.Tags {
	tags := nostr.Tags{}
	for i := 0; i < totalLen; i++ {
		tag := nostr.Tag{}
		randLen := RandRange(2, maxTagLen)
		for j := 0; j < randLen; j++ {
			tag = append(tag, randStr(RandRange(j+1, 256)))
		}
		tags = append(tags, tag)
	}
	return tags
}

func powInt(x, y int) int {
	return int(math.Pow(float64(x), float64(y)))
}

type RandomEventOption func(e *nostr.Event) error

var (
	// UseGeneratedKeypair overwrites the pubkey field of an event and the sig field of an event by signing the event with the provided keypair
	UsePreGeneratedKeypair = func(sk, pk string) RandomEventOption {
		return func(e *nostr.Event) error {
			e.PubKey = pk
			return e.Sign(pk)
		}
	}
	// UseKind overwrites the event kind field
	UseKind = func(kind int) RandomEventOption {
		return func(e *nostr.Event) error {
			e.Kind = kind
			return nil
		}
	}
	// UseKinds ensures the event kind field is set to one of the provided kinds
	UseKinds = func(kinds []int) RandomEventOption {
		return func(e *nostr.Event) error {
			e.Kind = kinds[rand.Intn(len(kinds))]
			return nil
		}
	}
	// MustBeNewerThan ensures the event created_at field is greater than the provided since time value
	MustBeNewerThan = func(since nostr.Timestamp) RandomEventOption {
		return func(e *nostr.Event) error {
			e.CreatedAt = nostr.Timestamp(since.Time().Add(time.Duration(RandRange(1, powInt(2, 16))) * time.Second).Unix()) // add a random amount of seconds from 0 to 2^16-1
			return nil
		}
	}
	// MustBeOlderThan ensures the event created_at field is less than the provided until time value
	MustBeOlderThan = func(until nostr.Timestamp) RandomEventOption {
		return func(e *nostr.Event) error {
			e.CreatedAt = nostr.Timestamp(until.Time().Add(time.Duration(-1*RandRange(1, powInt(2, 16))) * time.Second).Unix()) // remove a random amount of seconds from 0 to 2^16-1
			return nil
		}
	}
	// OverwriteID overwrites the event ID. NOTE: use this option last if providing more than one to ensure it is not overwritten
	OverwriteID = func(id string) RandomEventOption {
		return func(e *nostr.Event) error {
			e.ID = id
			return nil
		}
	}
	// AppendSuffixToContent appends a suffix to the events content field
	AppendSuffixToContent = func(suffix string) RandomEventOption {
		return func(e *nostr.Event) error {
			e.Content = e.Content + suffix
			return nil
		}
	}
	// AppendTags appends the contents of a tag map to the events tags field
	AppendTags = func(tagMap nostr.TagMap) RandomEventOption {
		return func(e *nostr.Event) error {
			for letter, values := range tagMap {
				tag := nostr.Tag{letter}
				if len(values) <= 1 {
					tag = append(tag, values...)
				} else {
					tag = append(tag, values[int(math.Max(0, float64(RandRange(1, len(values)))))])
				}
				e.Tags = append(e.Tags, tag)
			}
			return nil
		}
	}
)

// RandomKind creates a random kind within the range of 0 to 2^32-1
func RandomKind() int {
	return RandRange(1, powInt(2, 16)) - 1
}

// CreateRandomEvent creates an event with random pubkey, kind, value, tags, content and created at fields. The ID and signature are generated from the random content
func CreateRandomEvent(opts ...RandomEventOption) nostr.Event {
	sk, pk, err := CreateRandomKeypair()
	if err != nil {
		panic(err)
	}
	event := nostr.Event{
		PubKey:    pk,
		CreatedAt: nostr.Timestamp(time.Now().Add(time.Duration(-1*RandRange(1, powInt(2, 30))) * time.Second).Unix()),
		Kind:      RandomKind(),
		Tags:      CreateRandomTags(RandRange(1, 32), RandRange(2, 10)),
		Content:   randStr(RandRange(1, 2048) - 1),
	}
	if err := event.Sign(sk); err != nil {
		panic(err)
	}
	oldId := event.ID
	// Apply options here since it could override the signature
	for _, opt := range opts {
		if err := opt(&event); err != nil {
			panic(err)
		}
	}
	// if we didn't override the ID, then let's resign
	if event.ID == oldId {
		if err := event.Sign(sk); err != nil {
			panic(err)
		}
	}
	return event
}
