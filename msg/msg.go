package msg

import "github.com/nbd-wtf/go-nostr"

type Msg struct {
	ConnectionId string
	Data         []byte
	CloseConn    bool
	Unparseable  bool
}

type ParsedMsg struct {
	ConnectionId string
	Data         nostr.Envelope
	CloseConn    bool
	Callback     func(error)
	DeleteEvent  bool
}
