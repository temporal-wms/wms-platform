package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wms-platform/shared/pkg/outbox"
)

const (
	// DefaultCollectionName is the default name for the outbox collection
	DefaultCollectionName = "outbox_events"
)

// OutboxRepository implements outbox.Repository for MongoDB
type OutboxRepository struct {
	collection *mongo.Collection
}

// NewOutboxRepository creates a new MongoDB outbox repository
func NewOutboxRepository(db *mongo.Database) *OutboxRepository {
	return NewOutboxRepositoryWithCollection(db, DefaultCollectionName)
}

// NewOutboxRepositoryWithCollection creates a new MongoDB outbox repository with custom collection name
func NewOutboxRepositoryWithCollection(db *mongo.Database, collectionName string) *OutboxRepository {
	return &OutboxRepository{
		collection: db.Collection(collectionName),
	}
}

// Save saves an outbox event
func (r *OutboxRepository) Save(ctx context.Context, event *outbox.OutboxEvent) error {
	_, err := r.collection.InsertOne(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to save outbox event: %w", err)
	}
	return nil
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
	if err != nil {
		return fmt.Errorf("failed to save outbox events: %w", err)
	}
	return nil
}

// FindUnpublished retrieves unpublished events up to the specified limit
func (r *OutboxRepository) FindUnpublished(ctx context.Context, limit int) ([]*outbox.OutboxEvent, error) {
	// Find events that haven't been published yet and haven't exceeded max retries (default 10)
	// Note: $ifNull is an aggregation operator and can't be used in find queries
	filter := bson.M{
		"publishedAt": bson.M{"$exists": false},
		"$or": []bson.M{
			{"retryCount": bson.M{"$lt": 10}},            // retry count below max
			{"retryCount": bson.M{"$exists": false}},     // no retries yet
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find unpublished events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*outbox.OutboxEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode outbox events: %w", err)
	}

	return events, nil
}

// MarkPublished marks an event as published
func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID string) error {
	now := time.Now()
	filter := bson.M{"_id": eventID}
	update := bson.M{
		"$set": bson.M{
			"publishedAt": now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mark event as published: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("event not found: %s", eventID)
	}

	return nil
}

// IncrementRetry increments the retry count and updates last error
func (r *OutboxRepository) IncrementRetry(ctx context.Context, eventID string, errorMsg string) error {
	filter := bson.M{"_id": eventID}
	update := bson.M{
		"$inc": bson.M{
			"retryCount": 1,
		},
		"$set": bson.M{
			"lastError": errorMsg,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("event not found: %s", eventID)
	}

	return nil
}

// DeletePublished deletes published events older than the specified duration (in seconds)
func (r *OutboxRepository) DeletePublished(ctx context.Context, olderThan int64) error {
	threshold := time.Now().Add(-time.Duration(olderThan) * time.Second)
	filter := bson.M{
		"publishedAt": bson.M{
			"$exists": true,
			"$lt":     threshold,
		},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete published events: %w", err)
	}

	if result.DeletedCount > 0 {
		// Log deletion count if needed
	}

	return nil
}

// GetByID retrieves an outbox event by ID
func (r *OutboxRepository) GetByID(ctx context.Context, eventID string) (*outbox.OutboxEvent, error) {
	filter := bson.M{"_id": eventID}

	var event outbox.OutboxEvent
	err := r.collection.FindOne(ctx, filter).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get outbox event: %w", err)
	}

	return &event, nil
}

// FindByAggregateID retrieves all events for a specific aggregate
func (r *OutboxRepository) FindByAggregateID(ctx context.Context, aggregateID string) ([]*outbox.OutboxEvent, error) {
	filter := bson.M{"aggregateId": aggregateID}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find events by aggregate ID: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*outbox.OutboxEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode outbox events: %w", err)
	}

	return events, nil
}

// EnsureIndexes creates necessary indexes for the outbox collection
func (r *OutboxRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "publishedAt", Value: 1},
				{Key: "createdAt", Value: 1},
			},
			Options: options.Index().SetName("idx_publishedAt_createdAt"),
		},
		{
			Keys: bson.D{
				{Key: "aggregateId", Value: 1},
				{Key: "createdAt", Value: 1},
			},
			Options: options.Index().SetName("idx_aggregateId_createdAt"),
		},
		{
			Keys: bson.D{
				{Key: "eventType", Value: 1},
			},
			Options: options.Index().SetName("idx_eventType"),
		},
		{
			Keys: bson.D{
				{Key: "createdAt", Value: 1},
			},
			Options: options.Index().SetName("idx_createdAt"),
		},
		{
			// TTL index to auto-delete published events after 7 days (604800 seconds)
			// Only affects documents with publishedAt field set (unpublished events are preserved)
			Keys: bson.D{
				{Key: "publishedAt", Value: 1},
			},
			Options: options.Index().
				SetName("idx_publishedAt_ttl").
				SetExpireAfterSeconds(604800), // 7 days
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}
