package filter

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/msg"
	"github.com/TheRebelOfBabylon/tandem/storage"
	"github.com/nbd-wtf/go-nostr"
	"github.com/rs/zerolog"
)

var (
	formatLvlFunc = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %s |", i))
	}
)

// initFilterManager initializes the FilterManager
func initFilterManager(recvChan chan msg.ParsedMsg, filters map[string][]*nostr.ReqEnvelope, logger zerolog.Logger, dbConn storage.StorageBackend) *FilterManager {
	return &FilterManager{
		recvFromIngester: recvChan,
		sendToWSHandler:  make(chan msg.Msg),
		quit:             make(chan struct{}),
		filters:          filters,
		dbConn:           dbConn,
		logger:           logger,
		stopping:         false,
	}
}

var (
	preloadedEventsforTestFilters = []string{
		`["EVENT", {
			"kind": 1,
			"id": "46a036884e791733f61a498324f718103d0bb3d46ab3662824a82e24156a0e63",
			"pubkey": "da66d621d05bb7a7d64c1adfe0ea6421ca7db60d1089cd98b06ccfcd0ea2ed78",
			"created_at": 1725319757,
			"tags": [
				[
				"e",
				"125839f727e02998049148c5729fbbf9d75b0abc0504692a9fe557a20387785b",
				"",
				"root"
				],
				[
				"p",
				"6389be6491e7b693e9f368ece88fcd145f07c068d2c1bbae4247b9b5ef439d32"
				]
			],
			"content": "🔥🔥🔥",
			"sig": "97c93bf37f0de0709413215c3b7d7d29111596159592c5bdd45f14110fe8df848b979d63314c809a7a65a7deaa361a589e5dc5b2d68da78fc8d2b301f69dee1b"
		}]`,
		`["EVENT", {
			"kind": 1,
			"id": "eb56f7333053beaad6a69098cbd8c9bd7c34c4dadebe06826149cb648bf6a1b1",
			"pubkey": "e43f16ab84552a8680d3ade518803770fa16c9835da0a0f5b376cddef7f12786",
			"created_at": 1725319677,
			"tags": [],
			"content": "Zap request to send at:09/02/2024, 19:27:57 zapped: npub1q6mcr8tlr3l4gus3sfnw6772s7zae6hqncmw5wj27ejud5wcxf7q0nx7d5",
			"sig": "97bd6d257f8f868604c2164cc47d82f87dec569af45889912051b1054bf4ca68e92ae50665963384b3b8714f602c3072f76139cf23e3b50af1ec5b3d149ab677"
		}]`,
		`["EVENT", {
			"kind": 1,
			"id": "4a168ae74f74acf7ade3ad8cfaea80c5beb86edc10e102e57e5d862adc3cb00b",
			"pubkey": "e43f16ab84552a8680d3ade518803770fa16c9835da0a0f5b376cddef7f12786",
			"created_at": 1725319664,
			"tags": [],
			"content": "Zap request to send at:09/02/2024, 19:27:44 zapped: npub1q6mcr8tlr3l4gus3sfnw6772s7zae6hqncmw5wj27ejud5wcxf7q0nx7d5",
			"sig": "a828c69e669a2a68bde832ef465fad7b956d6b9827f2a9de2e72475c8f55610cafefc31d4aaecd7e220bb381d78949f4554cfa498af2dd24394dbb9ec3c07c99"
		}]`,
		`["EVENT", {
			"kind": 1,
			"id": "4edfccdec007edf614a1a7355260f461ce6f7970b85f479d8f61a13bee83a4f6",
			"pubkey": "44dc1c2db9c3fbd7bee9257eceb52be3cf8c40baf7b63f46e56b58a131c74f0b",
			"created_at": 1725319661,
			"tags": [
				[
				"e",
				"2a8f2f2d4cc831e22695792636169c06f7cb9baea09b9a65c8a870035288283c",
				"",
				"root"
				],
				[
				"p",
				"6140478c9ae12f1d0b540e7c57806649327a91b040b07f7ba3dedc357cab0da5"
				]
			],
			"content": "Lmao. ",
			"sig": "6da33343f86617acc68654652de083fbf24d86986cfc3cbfc82cd8017f086c5ac2d2b0da4bc48717b98660306bfdda2e2005ae7c8178cb1ee2df751b20326fad"
		}]`,
		`["EVENT", {
			"kind": 1,
			"id": "5fa0f011d76ad5ee386532e800b92e2b8a1154d42358ee78f8d0ca02cc6fe070",
			"pubkey": "fb67e428f2d0f152a37b25d44b5447268e7bdf9c1220a6500ac8c5e3d719442f",
			"created_at": 1725319624,
			"tags": [
				[
				"t",
				"RandomThoughts"
				],
				[
				"t",
				"randomthoughts"
				],
				[
				"r",
				"https://image.nostr.build/2d8fa6254ebe9de92e3c5861ca7519dfe0aae5f812abe3d1e873707b161c450f.jpg"
				],
				[
				"imeta",
				"url https://image.nostr.build/2d8fa6254ebe9de92e3c5861ca7519dfe0aae5f812abe3d1e873707b161c450f.jpg",
				"m image/jpeg",
				"alt Verifiable file url",
				"x cf3377c29aa23b478916a6f0bb45d047d994b6c53b355dec3d141d51b262b1e9",
				"size 86536",
				"dim 719x1007",
				"blurhash ^8R:QgR%~W%M%M%MIUjZofRjs:ayxuWBM{WBWBWBNGWBofWBayfR%MayM{WBayR*t7RjR*ofWBayM{s:j[NGs;WBt7ayRjayj[WBxuWBWBWVWBj[",
				"ox 2d8fa6254ebe9de92e3c5861ca7519dfe0aae5f812abe3d1e873707b161c450f"
				]
			],
			"content": "My sisters and I used to watch Punky Brewster when we were kids.\n\nI wonder what she's (Punky Brewster) up to these days. \n\n#RandomThoughts\n\nhttps://image.nostr.build/2d8fa6254ebe9de92e3c5861ca7519dfe0aae5f812abe3d1e873707b161c450f.jpg\n\n",
			"sig": "4476430ba8badd7c1c4d99f0bb5b1797ef65c977db61be992206b147049def7409215523e5540b737a5d0b91e33b86325f6d72a4e3267322fc87df991a354940"
		}]`,
	}
	existingConnId                 = "0d8f8ea5-65e4-4614-8ce2-8dca6966b028"
	existingSubId                  = "fe0e4851-c65d-4adf-93a4-c507cb1ab271"
	preloadedFiltersforTestFilters = map[string][]*nostr.ReqEnvelope{
		existingConnId: {
			{
				SubscriptionID: existingSubId,
				Filters: nostr.Filters{
					{
						IDs: []string{},
					},
				},
			},
		},
	}
	newSubId    = "44f835ed-bd39-48b9-b1b1-7c1be3e0b87f"
	newConnId   = "b19a7e3a-6f6a-47ef-9bdc-30bbb4848e59"
	newSubIdTwo = "6f933105-ec6f-402a-8267-989a29000175"
)

// TestConnectionAndSubscriptionMgmt tests that the filter manager behaves as expected when adding new subscrptions and connections as well as ending subscriptions and connections
func TestConnectionAndSubscriptionMgmt(t *testing.T) {
	// initialize logger
	mainLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, FormatLevel: formatLvlFunc, TimeLocation: time.UTC}).With().Timestamp().Logger()
	// initalize and start storage backend
	toDb := make(chan msg.ParsedMsg)
	dbConn, err := storage.Connect(config.Storage{Uri: "memory://"}, mainLogger.With().Str("module", "storageBackend").Logger(), toDb)
	if err != nil {
		t.Fatalf("unexpected error when initializing storage backend: %v", err)
	}
	if err := dbConn.Start(); err != nil {
		t.Fatalf("unexpected err when starting storage backend: %v", err)
	}
	defer func() {
		if err := dbConn.Stop(); err != nil {
			t.Errorf("unexpected error when safely shutting down db connection: %v", err)
		}
	}()
	// load storage backend with events
	for _, event := range preloadedEventsforTestFilters {
		errChan := make(chan error)
		toDb <- msg.ParsedMsg{ConnectionId: "some-id", Data: nostr.ParseMessage([]byte(event)), Callback: func(err error) { errChan <- err }}
		timer := time.NewTimer(5 * time.Second)
	errLoop:
		for {
			select {
			case err := <-errChan:
				if err != nil {
					t.Fatalf("unexpected error when loading storage backend with events: %v", err)
				}
				break errLoop
			case <-timer.C:
				t.Fatal("timed out waiting for error response from storage backend")
			}
		}
	}
	// initialize filter manager
	fromIngester := make(chan msg.ParsedMsg)
	filterMgr := initFilterManager(fromIngester, preloadedFiltersforTestFilters, mainLogger.With().Str("module", "filterManager").Logger(), dbConn)
	// start the filter manager
	if err := filterMgr.Start(); err != nil {
		t.Fatalf("unexpected error when starting filter manager: %v", err)
	}
	defer func() {
		if err := filterMgr.Stop(); err != nil {
			t.Errorf("unexpected error when shutting down filter manager: %v", err)
		}
	}()
	// ensure the preloaded connection and subscription id exist
	if !filterMgr.contains(existingConnId, existingSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure that the contains function behaves as expected
	if filterMgr.contains(existingConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", existingSubId) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", "some-random-id") {
		t.Errorf("filter manager contains a connection id and/or subscription id that it shouldn't")
	}
	// overwrite existing subid and check if it exists
	filterMgr.addSubscription(existingConnId, &nostr.ReqEnvelope{SubscriptionID: existingSubId})
	// ensure the preloaded connection and subscription id exist
	if !filterMgr.contains(existingConnId, existingSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure that the contains function behaves as expected
	if filterMgr.contains(existingConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", existingSubId) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", "some-random-id") {
		t.Errorf("filter manager contains a connection id and/or subscription id that it shouldn't")
	}
	// add a new subscription and check it exists
	filterMgr.addSubscription(existingConnId, &nostr.ReqEnvelope{SubscriptionID: newSubId})
	if !filterMgr.contains(existingConnId, newSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, newSubId)
	}
	// ensure the preloaded connection and subscription id still exist
	if !filterMgr.contains(existingConnId, existingSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure that the contains function behaves as expected
	if filterMgr.contains(existingConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubId) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", "some-random-id") {
		t.Errorf("filter manager contains a connection id and/or subscription id that it shouldn't")
	}
	filterMgr.addSubscription(newConnId, &nostr.ReqEnvelope{SubscriptionID: newSubIdTwo})
	if !filterMgr.contains(newConnId, newSubIdTwo) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", newConnId, newSubIdTwo)
	}
	// ensure the preloaded connection and subscription id still exist
	if !filterMgr.contains(existingConnId, newSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, newSubId)
	}
	if !filterMgr.contains(existingConnId, existingSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure that the contains function behaves as expected
	if filterMgr.contains(existingConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubId) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains(newConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubIdTwo) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", "some-random-id") {
		t.Errorf("filter manager contains a connection id and/or subscription id that it shouldn't")
	}
	// endSub non-existing subid but existing connid
	filterMgr.endSubscription(existingConnId, "some-random-id")
	if !filterMgr.contains(newConnId, newSubIdTwo) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", newConnId, newSubIdTwo)
	}
	// ensure the preloaded connection and subscription id still exist
	if !filterMgr.contains(existingConnId, newSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, newSubId)
	}
	if !filterMgr.contains(existingConnId, existingSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure that the contains function behaves as expected
	if filterMgr.contains(existingConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubId) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains(newConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubIdTwo) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", "some-random-id") {
		t.Errorf("filter manager contains a connection id and/or subscription id that it shouldn't")
	}
	// endSub non-existing subid, non-existing connid
	filterMgr.endSubscription("new-conn-id", "some-random-id")
	if !filterMgr.contains(newConnId, newSubIdTwo) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", newConnId, newSubIdTwo)
	}
	// ensure the preloaded connection and subscription id still exist
	if !filterMgr.contains(existingConnId, newSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, newSubId)
	}
	if !filterMgr.contains(existingConnId, existingSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure that the contains function behaves as expected
	if filterMgr.contains(existingConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubId) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains(newConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubIdTwo) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", "some-random-id") {
		t.Errorf("filter manager contains a connection id and/or subscription id that it shouldn't")
	}
	// endSub existing subid, non-existing connid
	filterMgr.endSubscription("new-conn-id", existingSubId)
	if !filterMgr.contains(newConnId, newSubIdTwo) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", newConnId, newSubIdTwo)
	}
	// ensure the preloaded connection and subscription id still exist
	if !filterMgr.contains(existingConnId, newSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, newSubId)
	}
	if !filterMgr.contains(existingConnId, existingSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure that the contains function behaves as expected
	if filterMgr.contains(existingConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubId) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains(newConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubIdTwo) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", "some-random-id") {
		t.Errorf("filter manager contains a connection id and/or subscription id that it shouldn't")
	}
	// endSub existing subid, existing connid
	filterMgr.endSubscription(existingConnId, existingSubId)
	if filterMgr.contains(existingConnId, existingSubId) {
		t.Errorf("filter manager contains the given connection id or sub id and it shouldn't: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure the preloaded connection and subscription id still exist
	if !filterMgr.contains(existingConnId, newSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, newSubId)
	}
	if !filterMgr.contains(newConnId, newSubIdTwo) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure that the contains function behaves as expected
	if filterMgr.contains(existingConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubId) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains(newConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubIdTwo) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", "some-random-id") {
		t.Errorf("filter manager contains a connection id and/or subscription id that it shouldn't")
	}
	// endConn non-existing connid
	filterMgr.endConnection("new-conn-id")
	if filterMgr.contains(existingConnId, existingSubId) {
		t.Errorf("filter manager contains the given connection id or sub id and it shouldn't: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure the preloaded connection and subscription id still exist
	if !filterMgr.contains(existingConnId, newSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, newSubId)
	}
	if !filterMgr.contains(newConnId, newSubIdTwo) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure that the contains function behaves as expected
	if filterMgr.contains(existingConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubId) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains(newConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubIdTwo) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", "some-random-id") {
		t.Errorf("filter manager contains a connection id and/or subscription id that it shouldn't")
	}
	// endConn, existing connid
	filterMgr.endConnection(newConnId)
	if filterMgr.contains(newConnId, newSubIdTwo) {
		t.Errorf("filter manager contains the given connection id or sub id and it shoudln't: connectionId=%s\tsubscriptionId=%s", newConnId, newSubIdTwo)
	}
	if filterMgr.contains(existingConnId, existingSubId) {
		t.Errorf("filter manager contains the given connection id or sub id and it shouldn't: connectionId=%s\tsubscriptionId=%s", existingConnId, existingSubId)
	}
	// ensure the preloaded connection and subscription id still exist
	if !filterMgr.contains(existingConnId, newSubId) {
		t.Errorf("filter manager does not contain the given connection id or sub id as expected: connectionId=%s\tsubscriptionId=%s", existingConnId, newSubId)
	}
	// ensure that the contains function behaves as expected
	if filterMgr.contains(existingConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubId) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains(newConnId, "some-random-id") {
		t.Errorf("filter manager contains a subscription id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", newSubIdTwo) {
		t.Errorf("filter manager contains a connection id that it shouldn't")
	}
	if filterMgr.contains("new-conn-id", "some-random-id") {
		t.Errorf("filter manager contains a connection id and/or subscription id that it shouldn't")
	}
}

type filterMgrTestCase struct {
	name           string
	inputMsg       msg.ParsedMsg
	validationFunc func(t *testing.T, filterMgr *FilterManager, filterMgrChan chan msg.Msg)
}

var (
	filterTime         = nostr.Timestamp(int64(1725319700))
	filterMgrTestCases = []filterMgrTestCase{
		{
			name: "REQ_Overwrite_SubId_IDs",
			inputMsg: msg.ParsedMsg{
				ConnectionId: existingConnId,
				Data: &nostr.ReqEnvelope{
					SubscriptionID: existingSubId,
					Filters: []nostr.Filter{
						{
							IDs: []string{"5fa0f011d76ad5ee386532e800b92e2b8a1154d42358ee78f8d0ca02cc6fe070"},
						},
					},
				},
			},
			validationFunc: func(t *testing.T, filterMgr *FilterManager, filterMgrChan chan msg.Msg) {
				if !filterMgr.contains(existingConnId, existingSubId) {
					t.Errorf("filter manager does not contain the given connection id or subscription id as expected: connId=%s\tsubId=%s", existingConnId, existingSubId)
				}
				for i := 0; i < 2; i++ {
					message := <-filterMgrChan
					if message.ConnectionId != existingConnId {
						t.Errorf("message from filter manager contains an unexpected connection id %s", message.ConnectionId)
					}
					if message.CloseConn {
						t.Error("message from filter manager close connection flag unexpectedely set to true")
					}
					switch m := nostr.ParseMessage(message.Data).(type) {
					case *nostr.EventEnvelope:
						if *m.SubscriptionID != existingSubId {
							t.Errorf("event from filter manager contains an unexpected subscription id %s", *m.SubscriptionID)
						}
						if m.ID != "5fa0f011d76ad5ee386532e800b92e2b8a1154d42358ee78f8d0ca02cc6fe070" {
							t.Errorf("event from filter manager does not contain expected id: %s", m.ID)
						}
					case *nostr.EOSEEnvelope:
						if *m != nostr.EOSEEnvelope(existingSubId) {
							t.Errorf("EOSE from filter manager contains an unexpected subscription id %s", *m)
						}
					default:
						t.Errorf("message from filter manager of an unexpected type %T", m)
					}
				}
				time.Sleep(1 * time.Second)
				if !filterMgr.contains(existingConnId, existingSubId) {
					t.Errorf("filter manager does not contain the given connection id or subscription id as expected: connId=%s\tsubId=%s", existingConnId, existingSubId)
				}
			},
		},
		{
			name: "REQ_New_SubId_Authors",
			inputMsg: msg.ParsedMsg{
				ConnectionId: existingConnId,
				Data: &nostr.ReqEnvelope{
					SubscriptionID: newSubId,
					Filters: []nostr.Filter{
						{
							Authors: []string{"e43f16ab84552a8680d3ade518803770fa16c9835da0a0f5b376cddef7f12786"},
						},
					},
				},
			},
			validationFunc: func(t *testing.T, filterMgr *FilterManager, filterMgrChan chan msg.Msg) {
				for i := 0; i < 3; i++ {
					message := <-filterMgrChan
					if message.ConnectionId != existingConnId {
						t.Errorf("message from filter manager contains an unexpected connection id %s", message.ConnectionId)
					}
					if message.CloseConn {
						t.Error("message from filter manager close connection flag unexpectedely set to true")
					}
					switch m := nostr.ParseMessage(message.Data).(type) {
					case *nostr.EventEnvelope:
						if *m.SubscriptionID != newSubId {
							t.Errorf("event from filter manager contains an unexpected subscription id %s", *m.SubscriptionID)
						}
						if m.PubKey != "e43f16ab84552a8680d3ade518803770fa16c9835da0a0f5b376cddef7f12786" {
							t.Errorf("event from filter manager does not contain expected pubkey: %s", m.PubKey)
						}
					case *nostr.EOSEEnvelope:
						if *m != nostr.EOSEEnvelope(newSubId) {
							t.Errorf("EOSE from filter manager contains an unexpected subscription id %s", *m)
						}
					default:
						t.Errorf("message from filter manager of an unexpected type %T", m)
					}
				}
				time.Sleep(1 * time.Second) // wait for a second to allow the manage go routine the opportunity to register the new sub id
				if !filterMgr.contains(existingConnId, existingSubId) {
					t.Errorf("filter manager does not contain the given connection id or subscription id as expected: connId=%s\tsubId=%s", existingConnId, existingSubId)
				}
				if !filterMgr.contains(existingConnId, newSubId) {
					t.Errorf("filter manager does not contain the given connection id or subscription id as expected: connId=%s\tsubId=%s", existingConnId, newSubId)
				}
			},
		},
		{
			name: "REQ_New_ConnId_Since",
			inputMsg: msg.ParsedMsg{
				ConnectionId: newConnId,
				Data: &nostr.ReqEnvelope{
					SubscriptionID: newSubIdTwo,
					Filters: []nostr.Filter{
						{
							Since: &filterTime,
						},
					},
				},
			},
			validationFunc: func(t *testing.T, filterMgr *FilterManager, filterMgrChan chan msg.Msg) {
				for i := 0; i < 2; i++ {
					message := <-filterMgrChan
					if message.ConnectionId != newConnId {
						t.Errorf("message from filter manager contains an unexpected connection id %s", message.ConnectionId)
					}
					if message.CloseConn {
						t.Error("message from filter manager close connection flag unexpectedely set to true")
					}
					switch m := nostr.ParseMessage(message.Data).(type) {
					case *nostr.EventEnvelope:
						if *m.SubscriptionID != newSubIdTwo {
							t.Errorf("event from filter manager contains an unexpected subscription id %s", *m.SubscriptionID)
						}
						if m.CreatedAt < filterTime {
							t.Errorf("event from filter manager was created before filter threshold: %s", m.CreatedAt.Time().String())
						}
					case *nostr.EOSEEnvelope:
						if *m != nostr.EOSEEnvelope(newSubIdTwo) {
							t.Errorf("EOSE from filter manager contains an unexpected subscription id %s", *m)
						}
					default:
						t.Errorf("message from filter manager of an unexpected type %T", m)
					}
				}
				time.Sleep(1 * time.Second) // wait for a second to allow the manage go routine the opportunity to register the new sub id
				if !filterMgr.contains(existingConnId, existingSubId) {
					t.Errorf("filter manager does not contain the given connection id or subscription id as expected: connId=%s\tsubId=%s", existingConnId, existingSubId)
				}
				if !filterMgr.contains(existingConnId, newSubId) {
					t.Errorf("filter manager does not contain the given connection id or subscription id as expected: connId=%s\tsubId=%s", existingConnId, newSubId)
				}
				if !filterMgr.contains(newConnId, newSubIdTwo) {
					t.Errorf("filter manager does not contain the given connection id or subscription id as expected: connId=%s\tsubId=%s", newConnId, newSubIdTwo)
				}
			},
		},
		{
			name: "REQ_Overwrite_SubId_Until",
			inputMsg: msg.ParsedMsg{
				ConnectionId: newConnId,
				Data: &nostr.ReqEnvelope{
					SubscriptionID: newSubIdTwo,
					Filters: []nostr.Filter{
						{
							Until: &filterTime,
						},
					},
				},
			},
			validationFunc: func(t *testing.T, filterMgr *FilterManager, filterMgrChan chan msg.Msg) {
				for i := 0; i < 5; i++ {
					message := <-filterMgrChan
					if message.ConnectionId != newConnId {
						t.Errorf("message from filter manager contains an unexpected connection id %s", message.ConnectionId)
					}
					if message.CloseConn {
						t.Error("message from filter manager close connection flag unexpectedely set to true")
					}
					switch m := nostr.ParseMessage(message.Data).(type) {
					case *nostr.EventEnvelope:
						if *m.SubscriptionID != newSubIdTwo {
							t.Errorf("event from filter manager contains an unexpected subscription id %s", *m.SubscriptionID)
						}
						if m.CreatedAt > filterTime {
							t.Errorf("event from filter manager was created after filter threshold: %s", m.CreatedAt.Time().String())
						}
					case *nostr.EOSEEnvelope:
						if *m != nostr.EOSEEnvelope(newSubIdTwo) {
							t.Errorf("EOSE from filter manager contains an unexpected subscription id %s", *m)
						}
					default:
						t.Errorf("message from filter manager of an unexpected type %T", m)
					}
				}
				time.Sleep(1 * time.Second) // wait for a second to allow the manage go routine the opportunity to register the new sub id
				if !filterMgr.contains(existingConnId, existingSubId) {
					t.Errorf("filter manager does not contain the given connection id or subscription id as expected: connId=%s\tsubId=%s", existingConnId, existingSubId)
				}
				if !filterMgr.contains(existingConnId, newSubId) {
					t.Errorf("filter manager does not contain the given connection id or subscription id as expected: connId=%s\tsubId=%s", existingConnId, newSubId)
				}
				if !filterMgr.contains(newConnId, newSubIdTwo) {
					t.Errorf("filter manager does not contain the given connection id or subscription id as expected: connId=%s\tsubId=%s", newConnId, newSubIdTwo)
				}
			},
		},
		// Test Kinds
		// Test Tags
		// Test Limit
		// Test CLOSE message
		// Test multiple filter conditions
		// Test Sending different types of REQ
		// Test CloseConn = true
	}
)

// TestFilterManager tests that the filter manager behaves as expected based on different messages received into it's main go routine
func TestFilterManager(t *testing.T) {
	// initialize logger
	mainLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, FormatLevel: formatLvlFunc, TimeLocation: time.UTC}).With().Timestamp().Logger()
	// initalize and start storage backend
	toDb := make(chan msg.ParsedMsg)
	dbConn, err := storage.Connect(config.Storage{Uri: "memory://"}, mainLogger.With().Str("module", "storageBackend").Logger(), toDb)
	if err != nil {
		t.Fatalf("unexpected error when initializing storage backend: %v", err)
	}
	if err := dbConn.Start(); err != nil {
		t.Fatalf("unexpected err when starting storage backend: %v", err)
	}
	defer func() {
		if err := dbConn.Stop(); err != nil {
			t.Errorf("unexpected error when safely shutting down db connection: %v", err)
		}
	}()
	// load storage backend with events
	for _, event := range preloadedEventsforTestFilters {
		errChan := make(chan error)
		toDb <- msg.ParsedMsg{ConnectionId: "some-id", Data: nostr.ParseMessage([]byte(event)), Callback: func(err error) { errChan <- err }}
		timer := time.NewTimer(5 * time.Second)
	errLoop:
		for {
			select {
			case err := <-errChan:
				if err != nil {
					t.Fatalf("unexpected error when loading storage backend with events: %v", err)
				}
				break errLoop
			case <-timer.C:
				t.Fatal("timed out waiting for error response from storage backend")
			}
		}
	}
	// initialize filter manager
	fromIngester := make(chan msg.ParsedMsg)
	filterMgr := initFilterManager(fromIngester, preloadedFiltersforTestFilters, mainLogger.With().Str("module", "filterManager").Logger(), dbConn)
	// start the filter manager
	if err := filterMgr.Start(); err != nil {
		t.Fatalf("unexpected error when starting filter manager: %v", err)
	}
	defer func() {
		if err := filterMgr.Stop(); err != nil {
			t.Errorf("unexpected error when shutting down filter manager: %v", err)
		}
	}()
	filterMgrChan := filterMgr.SendChannel()
	for _, testCase := range filterMgrTestCases {
		t.Logf("start test case %s...", testCase.name)
		fromIngester <- testCase.inputMsg
		testCase.validationFunc(t, filterMgr, filterMgrChan)
	}
}
