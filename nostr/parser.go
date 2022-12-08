package nostr

import (
	"encoding/hex"
	"encoding/json"
	"sync"

	bg "github.com/SSSOCPaulCote/blunderguard"
	"github.com/TheRebelOfBabylon/fastjson"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

const (
	ErrInvalidFormat   = bg.Error("invalid websocket message format")
	ErrInvalidEvent    = bg.Error("invalid event message format")
	ErrInvalidReq      = bg.Error("invalid request message format")
	ErrInvalidClose    = bg.Error("invalid close message format")
	ErrInvalidNostrMsg = bg.Error("invalid nostr message format")
	ErrNoContent       = bg.Error("no content in event")
	ErrInvalidSig      = bg.Error("invalid signature")
	ErrMissingField    = bg.Error("missing one or more mandatory fields")
)

type SafeParser struct {
	fastjson.Parser
	sync.RWMutex
}

// keyDuplicate checks a slice of strings for any duplicates.
// Returns true if yes, false if not.
func keyDuplicate(keys []string) bool {
	visited := make(map[string]bool, 0)
	for _, key := range keys {
		if visited[key] == true {
			return true
		} else {
			visited[key] = true
		}
	}
	return false
}

// ValidateSignature validates the signature of a serialized event against a pubkey
func ValidateSignature(event Event) error {
	pub, err := hex.DecodeString(event.Pubkey)
	if err != nil {
		return err
	}
	pk, err := schnorr.ParsePubKey(pub)
	if err != nil {
		return err
	}
	signature, err := hex.DecodeString(event.Sig)
	if err != nil {
		return err
	}
	s, err := schnorr.ParseSignature(signature)
	if err != nil {
		return err
	}
	e := event.CreateEventId()
	ok := s.Verify(e, pk)
	if !ok {
		return ErrInvalidSig
	}
	return nil
}

// ValidateNostr validates the parsed nostr message for formatting, signatures, etc. and returns a proper struct
func ValidateNostr(msg interface{}) (interface{}, error) {
	switch m := msg.(type) {
	case Event:
		if m.Content == "" {
			return nil, ErrNoContent
		} else if m.Pubkey == "" || m.Sig == "" || m.CreatedAt == 0 || m.Kind == 0 || m.EventId == "" {
			return nil, ErrMissingField
		} else if err := ValidateSignature(m); err != nil {
			return nil, err
		} // else if CreatedAt is close to time.Now()
		return m, nil
	case string:
		if m == "" {
			return nil, ErrMissingField
		}
		return m, nil
	case []map[string]any:
		// TODO - Parse filters
		return m, nil
	default:
		return nil, ErrInvalidNostrMsg
	}
}

// ParseNostr parses the raw WebSocket message and checks if it's a Nostr EVENT, REQ or CLOSE message
func ParseNostr(msg []byte, p *SafeParser) (interface{}, error) {
	var result []string // Raw nostr messages are JSON arrays
	json.Unmarshal(msg, &result)
	if len(result) == 0 {
		return nil, ErrInvalidFormat
	}
	switch result[0] {
	case "EVENT":
		var event []Event
		json.Unmarshal(msg, &event)
		if len(event) == 0 || len(event) != 2 {
			return nil, ErrInvalidEvent
		}
		return event[1], nil
	case "REQ":
		var request []map[string]any
		json.Unmarshal(msg, &request)
		if len(request) == 0 || len(request) < 3 {
			return nil, ErrInvalidReq
		}
		resp := request[2:]
		resp = append(resp, map[string]any{"subscription_id": result[1]})
		return resp, nil
	case "CLOSE":
		if len(result) > 2 {
			return nil, ErrInvalidClose
		}
		return result[1], nil
	default:
		return nil, ErrInvalidFormat
	}
}
