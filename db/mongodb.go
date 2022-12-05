package db

import (
	"context"
	"fmt"

	"github.com/TheRebelOfBabylon/tandem/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
