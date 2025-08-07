package mongo

import (
	"context"
	"go-order-eda/src/config"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	clientInstance *mongo.Client
	clientOnce     sync.Once
)

func GetMongoClient(cfg *config.Config) (*mongo.Client, error) {
	var err error
	clientOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, e := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDBConnectionString))
		if e != nil {
			err = e
			return
		}
		clientInstance = client
	})
	return clientInstance, err
}

func GetCollection(cfg *config.Config, collectionName string) *mongo.Collection {
	client, err := GetMongoClient(cfg)
	if err != nil {
		log.Fatalf("Failed to get MongoDB client: %v", err)
	}
	return client.Database(cfg.MongoDBDatabaseName).Collection(collectionName)
}
