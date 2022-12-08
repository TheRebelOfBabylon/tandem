package nostr

import (
	"crypto/sha256"
	"fmt"
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
