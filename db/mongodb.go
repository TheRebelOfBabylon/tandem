package db

import (
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
)

type MongoDB struct {
	client *mongo.Client
	sync.RWMutex
}
