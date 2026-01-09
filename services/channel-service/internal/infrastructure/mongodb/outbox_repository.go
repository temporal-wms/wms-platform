package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wms-platform/shared/pkg/outbox"
)

// OutboxRepository implements outbox.Repository for MongoDB
type OutboxRepository struct {
	collection *mongo.Collection
}

// NewOutboxRepository creates a new outbox repository
func NewOutboxRepository(db *mongo.Database) *OutboxRepository {
	collection := db.Collection("outbox")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create indexes for efficient querying
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "publishedAt", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "publishedAt", Value: 1},
				{Key: "retryCount", Value: 1},
				{Key: "maxRetries", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "aggregateId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: 1}},
		},
	}

	collection.Indexes().CreateMany(ctx, indexes)

	return &OutboxRepository{
		collection: collection,
	}
}

// Save saves an outbox event
func (r *OutboxRepository) Save(ctx context.Context, event *outbox.OutboxEvent) error {
	_, err := r.collection.InsertOne(ctx, event)
	return err
}

// SaveAll saves multiple outbox events in a single operation
func (r *OutboxRepository) SaveAll(ctx context.Context, events []*outbox.OutboxEvent) error {
	if len(events) == 0 {
		return nil
	}

	docs := make([]interface{}, len(events))
	for i, event := range events {
		docs[i] = event
	}

	_, err := r.collection.InsertMany(ctx, docs)
	return err
}

// FindUnpublished retrieves unpublished events up to the specified limit
func (r *OutboxRepository) FindUnpublished(ctx context.Context, limit int) ([]*outbox.OutboxEvent, error) {
	filter := bson.M{
		"publishedAt": nil,
		"$expr": bson.M{
			"$lt": []string{"$retryCount", "$maxRetries"},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*outbox.OutboxEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}

	return events, nil
}

// MarkPublished marks an event as published
func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID string) error {
	now := time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": eventID},
		bson.M{
			"$set": bson.M{
				"publishedAt": now,
			},
		},
	)
	return err
}

// IncrementRetry increments the retry count and updates last error
func (r *OutboxRepository) IncrementRetry(ctx context.Context, eventID string, errorMsg string) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": eventID},
		bson.M{
			"$inc": bson.M{"retryCount": 1},
			"$set": bson.M{"lastError": errorMsg},
		},
	)
	return err
}

// DeletePublished deletes published events older than the specified duration (in seconds)
func (r *OutboxRepository) DeletePublished(ctx context.Context, olderThan int64) error {
	cutoff := time.Now().Add(-time.Duration(olderThan) * time.Second)
	_, err := r.collection.DeleteMany(
		ctx,
		bson.M{
			"publishedAt": bson.M{
				"$ne":  nil,
				"$lt": cutoff,
			},
		},
	)
	return err
}

// GetByID retrieves an outbox event by ID
func (r *OutboxRepository) GetByID(ctx context.Context, eventID string) (*outbox.OutboxEvent, error) {
	var event outbox.OutboxEvent
	err := r.collection.FindOne(ctx, bson.M{"_id": eventID}).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

// FindByAggregateID retrieves all events for a specific aggregate
func (r *OutboxRepository) FindByAggregateID(ctx context.Context, aggregateID string) ([]*outbox.OutboxEvent, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"aggregateId": aggregateID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*outbox.OutboxEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}

	return events, nil
}
