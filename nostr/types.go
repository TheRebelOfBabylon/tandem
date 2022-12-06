package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	bg "github.com/SSSOCPaulCote/blunderguard"
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

type Event struct {
	EventId   string `json:"id"`
	Pubkey    string
	CreatedAt uint32 `json:"created_at"`
	Kind      uint16
	Tags      [][]string
	Content   string
	Sig       string
}

// CreateEventId creates the EventId from the serialized Event content
func (e *Event) CreateEventId() []byte {
	serialTags := []byte{'['}
	for _, tag := range e.Tags {
		if len(tag) != 0 {
			serialTags = append(serialTags, '[')
			for _, t := range tag {
				serialTags = append(serialTags, '"')
				serialTags = append(serialTags, t...)
				serialTags = append(serialTags, '"')
				serialTags = append(serialTags, ',')
			}
			serialTags[len(serialTags)-1] = ']'
			serialTags = append(serialTags, ',')
		}
	}
	if len(serialTags) == 1 {
		serialTags = append(serialTags, ']')
	} else {
		serialTags[len(serialTags)-1] = ']'
	}
	serial := fmt.Sprintf("[0,\"%s\",%v,%v,%s,\"%s\"]", e.Pubkey, e.CreatedAt, e.Kind, serialTags, e.Content)
	hash := sha256.Sum256([]byte(serial))
	return hash[:]
}

type Filter struct {
	Ids     []string
	Authors []string
	Kinds   []uint16
	Tags    [][]string
	Since   uint64
	Until   uint64
	Limit   uint16
}

type Request struct {
	SubscriptionId string
	Filters        []Filter
}

type Close struct {
	SubscriptionId string
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
func ParseNostr(msg []byte) (interface{}, error) {
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
