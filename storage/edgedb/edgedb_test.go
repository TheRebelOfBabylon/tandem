//go:build edgedb
// +build edgedb

package edgedb

import (
	"context"
	"testing"
	"time"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/test"
	"github.com/nbd-wtf/go-nostr"
)

var (
	edgedbConfig = func() config.Storage {
		cfg, err := config.ReadConfig("../../test.toml")
		if err != nil {
			panic(err)
		}
		return cfg.Storage
	}
)

// TestEdgeDBSaveEvent tests that the edge db storage backend can store events
func TestEdgeDBSaveEvent(t *testing.T) {
	// connect to db
	dbConn, err := ConnectEdgeDB(edgedbConfig())
	if err != nil {
		t.Fatalf("unexpected error when connection to edge db: %v", err)
	}
	defer dbConn.Close()
	// create a random event and save it
	randomEvent := test.CreateRandomEvent()
	if err := dbConn.SaveEvent(context.Background(), &randomEvent); err != nil {
		t.Errorf("unexpected error when saving random event %s: %v", randomEvent.String(), err)
	}
}

type queryTestCase struct {
	name   string
	filter nostr.Filter
}

var (
	randTime        = nostr.Timestamp(test.RandRange(int(time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC).Unix()), int(time.Now().Unix())))
	randTimeTwo     = nostr.Timestamp(test.RandRange(int(time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC).Unix()), int(time.Now().Unix())))
	basketCaseSince = nostr.Timestamp(time.Date(1971, time.January, 1, 0, 0, 0, 0, time.UTC).Unix())
	basketCaseUntil = nostr.Timestamp(time.Date(2001, time.September, 11, 0, 0, 0, 0, time.UTC).Unix())
	queryTestCases  = []queryTestCase{
		{
			name: "IDsCase",
			filter: nostr.Filter{
				IDs: []string{},
			},
		},
		{
			name: "KindsCase_OneKind",
			filter: nostr.Filter{
				Kinds: []int{test.RandomKind()},
			},
		},
		{
			name: "KindsCase_TwoKinds",
			filter: nostr.Filter{
				Kinds: []int{test.RandomKind(), test.RandomKind()},
			},
		},
		{
			name: "SinceCase",
			filter: nostr.Filter{
				Since: &randTime,
			},
		},
		{
			name: "UntilCase",
			filter: nostr.Filter{
				Until: &randTimeTwo,
			},
		},
		{
			name: "AuthorCase",
			filter: nostr.Filter{
				Authors: []string{},
			},
		},
		{
			name: "SearchCase",
			filter: nostr.Filter{
				Search: "babaganoush",
			},
		},
		{
			name: "TagsCase_OneTag",
			filter: nostr.Filter{
				Tags: nostr.TagMap{
					"x": []string{"y", "z"},
				},
			},
		},
		{
			name: "TagsCase_TwoTags",
			filter: nostr.Filter{
				Tags: nostr.TagMap{
					"1": []string{"2", "3"},
					"a": []string{"b", "c"},
				},
			},
		},
		{
			name: "LimitCase",
			filter: nostr.Filter{
				Limit: 10,
			},
		},
		{
			name: "BasketCase",
			filter: nostr.Filter{
				IDs:     []string{},
				Authors: []string{},
				Kinds: []int{
					test.RandomKind(),
					test.RandomKind(),
					test.RandomKind(),
				},
				Since:  &basketCaseSince,
				Until:  &basketCaseUntil,
				Search: "foobar",
				Tags: nostr.TagMap{
					"a":   []string{"b", "c", "d", "e"},
					"123": []string{"456", "789", "101112"},
					"x":   []string{"y", "z", "w"},
				},
				Limit: test.RandRange(1, 100),
			},
		},
	}
)

// TestEdgeDBQueryEvents tests that the edge db storage backend can query events
func TestEdgeDBQueryEvents(t *testing.T) {
	// connect to db
	dbConn, err := ConnectEdgeDB(edgedbConfig())
	if err != nil {
		t.Fatalf("unexpected error when connection to edge db: %v", err)
	}
	defer dbConn.Close()
	for _, testCase := range queryTestCases {
		t.Logf("starting test case %s...", testCase.name)
		// determine number of events we will Save
		numEvents := test.RandRange(1, 11)
		// determine our event options
		var opts []test.RandomEventOption
		if testCase.filter.Kinds != nil {
			opts = append(opts, test.UseKinds(testCase.filter.Kinds))
		}
		if testCase.filter.Authors != nil {
			sk, pk, err := test.CreateRandomKeypair()
			if err != nil {
				t.Fatalf("unexpected error when generating random keypair: %v", err)
			}
			testCase.filter.Authors = []string{pk}
			opts = append(opts, test.UsePreGeneratedKeypair(sk, pk))
		}
		if testCase.filter.Since != nil {
			opts = append(opts, test.MustBeNewerThan(*testCase.filter.Since))
		}
		if testCase.filter.Until != nil {
			opts = append(opts, test.MustBeOlderThan(*testCase.filter.Until))
		}
		if testCase.filter.Search != "" {
			opts = append(opts, test.AppendSuffixToContent(testCase.filter.Search))
		}
		if testCase.filter.Tags != nil {
			opts = append(opts, test.AppendTags(testCase.filter.Tags))
		}
		// create our events
		for i := 0; i < numEvents; i++ {
			randEvent := test.CreateRandomEvent(opts...)
			if err := dbConn.SaveEvent(context.Background(), &randEvent); err != nil {
				t.Fatalf("unexpected event when saving event %s to edgedb: %v", randEvent.String(), err)
			}
			if testCase.filter.IDs != nil {
				testCase.filter.IDs = append(testCase.filter.IDs, randEvent.ID)
			}
		}
		// query using our filter
		recv, err := dbConn.QueryEvents(context.Background(), testCase.filter)
		if err != nil {
			t.Errorf("unexpected error when querying for events using filter %s: %v", testCase.filter.String(), err)
		}
		var count int
		timeout := time.NewTimer(15 * time.Second)
	recvLoop:
		for {
			select {
			case <-timeout.C:
				t.Errorf("timed out waiting for events from edgedb")
				break recvLoop
			case event, ok := <-recv:
				if !ok {
					t.Logf("receive events channel closed for test case %s", testCase.name)
					break recvLoop
				}
				count += 1
				// ensure the event returned conforms to the filter
				if !testCase.filter.Matches(event) {
					t.Errorf("unexpected event returned from storage: %s", event.String())
				}
			}
		}
		if testCase.filter.Limit != 0 && count > testCase.filter.Limit {
			t.Errorf("unexpected number of events returned from storage: expected less than %v or the same, got %v", testCase.filter.Limit, count)
		}
	}
}

// TestEdgeDBDeleteEvent tests that the edge db storage backend can delete events
func TestEdgeDBDeleteEvent(t *testing.T) {
	// connect to db
	dbConn, err := ConnectEdgeDB(edgedbConfig())
	if err != nil {
		t.Fatalf("unexpected error when connection to edge db: %v", err)
	}
	defer dbConn.Close()
	// determine number of events we will Save
	numEvents := test.RandRange(1, 11)
	// create our events and keep track of their IDs
	filter := nostr.Filter{}
	for i := 0; i < numEvents; i++ {
		randEvent := test.CreateRandomEvent()
		if err := dbConn.SaveEvent(context.Background(), &randEvent); err != nil {
			t.Fatalf("unexpected event when saving event %s to edgedb: %v", randEvent.String(), err)
		}
		filter.IDs = append(filter.IDs, randEvent.ID)
	}
	// delete them
	for _, id := range filter.IDs {
		if err := dbConn.DeleteEvent(context.Background(), &nostr.Event{ID: id}); err != nil {
			t.Errorf("unexpected error when deleting event with id %s from edgedb: %v", id, err)
		}
	}
	// query using our filter
	recv, err := dbConn.QueryEvents(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error when querying for events using filter %s: %v", filter.String(), err)
	}
	var count int
	timeout := time.NewTimer(15 * time.Second)
recvLoop:
	for {
		select {
		case <-timeout.C:
			t.Errorf("timed out waiting for events from edgedb")
			break recvLoop
		case _, ok := <-recv:
			if !ok {
				t.Log("receive events channel closed for test")
				break recvLoop
			}
			count += 1
		}
	}
	if count != 0 {
		t.Errorf("unexpected number of events returned from storage: expected 0, got %v", count)
	}
}
