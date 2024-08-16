package storage

import (
	"github.com/TheRebelOfBabylon/eventstore"
)

type StorageBackend interface {
	eventstore.Store
	Start() error
	Stop() error
}
