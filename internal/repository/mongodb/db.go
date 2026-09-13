package mongodb

import (
	"context"
	"fmt"

	"github.com/luli-tech/twilio-Boss/internal/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	accountsCollection     = "accounts"
	transactionsCollection = "transactions"
	smsCollection          = "sms_messages"
	usersCollection        = "users"
)

// NewDB initializes a MongoDB client and returns the selected database.
func NewDB(ctx context.Context, cfg *config.MongoDBConfig) (*mongo.Client, *mongo.Database, error) {
	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create mongodb client: %w", err)
	}

	if err := client.Ping(connectCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, fmt.Errorf("mongodb ping failed: %w", err)
	}

	db := client.Database(cfg.Database)
	if err := ensureIndexes(connectCtx, db); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, fmt.Errorf("mongodb index setup failed: %w", err)
	}

	return client, db, nil
}

func ensureIndexes(ctx context.Context, db *mongo.Database) error {
	indexes := map[string][]mongo.IndexModel{
		accountsCollection: {
			{Keys: bson.D{{Key: "sid", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		transactionsCollection: {
			{Keys: bson.D{{Key: "account_id", Value: 1}}},
			{Keys: bson.D{{Key: "reference_id", Value: 1}}},
		},
		smsCollection: {
			{Keys: bson.D{{Key: "sid", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "carrier_id", Value: 1}}},
			{Keys: bson.D{{Key: "account_id", Value: 1}, {Key: "created_at", Value: -1}}},
		},
		usersCollection: {
			{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "account_id", Value: 1}}},
		},
	}

	for collection, models := range indexes {
		if _, err := db.Collection(collection).Indexes().CreateMany(ctx, models); err != nil {
			return err
		}
	}

	return nil
}
