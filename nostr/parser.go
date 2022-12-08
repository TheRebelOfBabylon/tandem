package nostr

import (
	"encoding/hex"
	"regexp"
	"sync"

	bg "github.com/SSSOCPaulCote/blunderguard"
	"github.com/TheRebelOfBabylon/fastjson"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

const (
	ErrInvalidFormat    = bg.Error("invalid websocket message format")
	ErrInvalidEvent     = bg.Error("invalid event message format")
	ErrInvalidReq       = bg.Error("invalid request message format")
	ErrInvalidClose     = bg.Error("invalid close message format")
	ErrInvalidNostrMsg  = bg.Error("invalid nostr message format")
	ErrNoContent        = bg.Error("no content in event")
	ErrInvalidSig       = bg.Error("invalid signature")
	ErrMissingField     = bg.Error("missing one or more mandatory fields")
	ErrKeyDuplicates    = bg.Error("duplicate keys in JSON object")
	ErrInvalidTagFormat = bg.Error("invalid tag format")
)

var (
	tagRegexp      = `^\#[a-z]$`
	tagEventRegexp = `^[a-z]$`
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
		if visited[key] {
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

// ValidateEvent validates the parsed nostr event for formatting, signatures, etc.
func ValidateEvent(event Event) (interface{}, error) {
	if event.Content == "" {
		return nil, ErrNoContent
	} else if event.Pubkey == "" || event.Sig == "" || event.CreatedAt == 0 || event.Kind == 0 || event.EventId == "" {
		return nil, ErrMissingField
	} else if err := ValidateSignature(event); err != nil {
		return nil, err
	} // else if CreatedAt is close to time.Now() NIP-22
	return event, nil
}

// ParseAndValidateNostr parses the raw WebSocket message and checks if it's a Nostr EVENT, REQ or CLOSE message
// Performs some validation as well.
func ParseAndValidateNostr(msg []byte, p *SafeParser) (interface{}, error) {
	// First take the lock on the parser
	p.Lock()
	defer p.Unlock()
	result, err := p.ParseBytes(msg)
	if err != nil {
		return nil, err
	}
	// Nostr messages are JSON arrays
	arr, err := result.Array()
	if err != nil {
		return nil, err
	}
	if len(arr) == 0 || len(arr) < 2 {
		return nil, ErrInvalidFormat
	}
	switch string(arr[0].GetStringBytes()) {
	case "EVENT":
		e, err := arr[1].Object()
		if err != nil {
			return nil, err
		}
		// Get keys and check for any duplicates
		keys := e.GetKeys()
		if keyDuplicate(keys) {
			return nil, ErrKeyDuplicates
		}
		var event Event
		for _, key := range keys {
			switch key {
			case "id":
				id, err := e.Get(key).StringBytes()
				if err != nil {
					return nil, err
				}
				event.EventId = string(id)
			case "pubkey":
				pk, err := e.Get(key).StringBytes()
				if err != nil {
					return nil, err
				}
				event.Pubkey = string(pk)
			case "created_at":
				ts, err := e.Get(key).Uint64()
				if err != nil {
					return nil, err
				}
				event.CreatedAt = ts
			case "kind":
				ts, err := e.Get(key).Uint()
				if err != nil {
					return nil, err
				}
				event.Kind = uint16(ts)
			case "content":
				cnt, err := e.Get(key).StringBytes()
				if err != nil {
					return nil, err
				}
				event.Content = string(cnt)
			case "sig":
				sig, err := e.Get(key).StringBytes()
				if err != nil {
					return nil, err
				}
				event.Sig = string(sig)
			case "tags":
				tags, err := e.Get(key).Array()
				if err != nil {
					return nil, err
				}
				for _, t := range tags {
					tag, err := t.Array()
					if err != nil {
						return nil, err
					}
					var tagStrings []string
					reg := regexp.MustCompile(tagEventRegexp)
					for x, el := range tag {
						elem, err := el.StringBytes()
						if err != nil {
							return nil, err
						}
						if x == 0 && !reg.Match(elem) {
							return nil, ErrInvalidTagFormat
						}
						tagStrings = append(tagStrings, string(elem))
					}
					event.Tags = append(event.Tags, tagStrings[:])
				}
			default:
				return nil, ErrInvalidEvent
			}
		}
		return ValidateEvent(event)
	case "REQ":
		var req Request
		subId, err := arr[1].StringBytes()
		if err != nil {
			return nil, err
		}
		req.SubscriptionId = string(subId)
		filters := arr[2:]
		for _, f := range filters {
			filter, err := f.Object()
			if err != nil {
				return nil, err
			}
			keys := filter.GetKeys()
			if keyDuplicate(keys) {
				return nil, ErrKeyDuplicates
			}
			var fil Filter
			for _, key := range keys {
				switch key {
				case "ids":
					ids, err := filter.Get(key).Array()
					if err != nil {
						return nil, err
					}
					for _, id := range ids {
						i, err := id.StringBytes()
						if err != nil {
							return nil, err
						}
						fil.Ids = append(fil.Ids, string(i))
					}
				case "authors":
					authors, err := filter.Get(key).Array()
					if err != nil {
						return nil, err
					}
					for _, author := range authors {
						auth, err := author.StringBytes()
						if err != nil {
							return nil, err
						}
						fil.Authors = append(fil.Authors, string(auth))
					}
				case "kinds":
					kinds, err := filter.Get(key).Array()
					if err != nil {
						return nil, err
					}
					for _, kind := range kinds {
						k, err := kind.Uint()
						if err != nil {
							return nil, err
						}
						fil.Kinds = append(fil.Kinds, uint16(k))
					}
				case "since":
					since, err := filter.Get(key).Uint64()
					if err != nil {
						return nil, err
					}
					fil.Since = since
				case "until":
					until, err := filter.Get(key).Uint64()
					if err != nil {
						return nil, err
					}
					fil.Until = until
				case "limit":
					limit, err := filter.Get(key).Uint()
					if err != nil {
						return nil, err
					}
					fil.Limit = uint16(limit)
				default:
					reg := regexp.MustCompile(tagRegexp)
					if reg.MatchString(key) {
						tag := []string{key}
						tags, err := filter.Get(key).Array()
						if err != nil {
							return nil, err
						}
						for _, t := range tags {
							str, err := t.StringBytes()
							if err != nil {
								return nil, err
							}
							tag = append(tag, string(str))
						}
						fil.Tags = append(fil.Tags, tag[:])
					} else {
						return nil, ErrInvalidReq
					}
				}
			}
			req.Filters = append(req.Filters, fil)
		}
		return req, nil // TODO - ValidateReq function to perform some validation
	case "CLOSE":
		var close Close
		subId, err := arr[1].StringBytes()
		if err != nil {
			return nil, err
		}
		close.SubscriptionId = string(subId)
		return close, nil
	default:
		return nil, ErrInvalidFormat
	}
}
