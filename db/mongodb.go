package db

import (
	"context"
	"fmt"

	bg "github.com/SSSOCPaulCote/blunderguard"
	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/nostr"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	ErrEventDuplicate = bg.Error("duplicate event")
)

type MongoDB struct {
	client *mongo.Client
}

// NewDBConnection creates a connection with
func NewDBConnection(ctx context.Context, cfg *config.Data) (*MongoDB, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s/?maxPoolSize=20&w=majority", cfg.DBUser, cfg.DBPass, cfg.DBAddr)))
	if err != nil {
		return nil, err
	}
	return &MongoDB{
		client: client,
	}, nil
}

//InsertEvent inserts a new event into the database
func (db *MongoDB) InsertEvent(ctx context.Context, event nostr.Event) error {
	// Check if id is already in database
	result := db.client.Database("nostr").Collection("event").FindOne(ctx, event.EventIdBsonFilter())
	if result.Err() == nil {
		return ErrEventDuplicate
	}
	_, err := db.client.Database("nostr").Collection("event").InsertOne(ctx, event.ToBson())
	return err
}
