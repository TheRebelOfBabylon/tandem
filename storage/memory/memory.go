package memory

import (
	"github.com/TheRebelOfBabylon/eventstore/slicestore"
	"github.com/TheRebelOfBabylon/tandem/config"
)

// ConnectToMemory instantiates the Memory storage
func ConnectMemory(cfg config.Storage) (*slicestore.SliceStore, error) {
	sliceStore := &slicestore.SliceStore{}
	if err := sliceStore.Init(); err != nil {
		return nil, err
	}
	return sliceStore, nil
}
