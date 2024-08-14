package storage

import (
	"github.com/TheRebelOfBabylon/eventstore"
	"github.com/TheRebelOfBabylon/tandem/msg"
)

type StorageBackend interface {
	eventstore.Store
	Start() error
	ReceiveFromIngester(recv chan msg.ParsedMsg)
	Stop() error
}
