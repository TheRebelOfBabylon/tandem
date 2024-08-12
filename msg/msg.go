package msg

import "github.com/nbd-wtf/go-nostr"

type Msg struct {
	ConnectionId string
	Data         []byte
}

type ParsedMsg struct {
	ConnectionId string
	Data         nostr.Envelope
}
