package idempotency

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// Collection names
	idempotencyKeysCollection  = "idempotency_keys"
	processedMessagesCollection = "processed_messages"
)

// MongoKeyRepository implements KeyRepository using MongoDB
type MongoKeyRepository struct {
	collection *mongo.Collection
}

// NewMongoKeyRepository creates a new MongoDB-backed key repository
func NewMongoKeyRepository(db *mongo.Database) *MongoKeyRepository {
	return &MongoKeyRepository{
		collection: db.Collection(idempotencyKeysCollection),
	}
}

// AcquireLock attempts to acquire a lock for the given idempotency key
func (r *MongoKeyRepository) AcquireLock(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, bool, error) {
	now := time.Now().UTC()
	key.LockedAt = &now

	// Try to insert a new key with lock
	filter := bson.M{
		"serviceId": key.ServiceID,
		"key":       key.Key,
	}

	update := bson.M{
		"$setOnInsert": bson.M{
			"key":                key.Key,
			"serviceId":          key.ServiceID,
			"userId":             key.UserID,
			"requestPath":        key.RequestPath,
			"requestMethod":      key.RequestMethod,
			"requestFingerprint": key.RequestFingerprint,
			"createdAt":          key.CreatedAt,
			"expiresAt":          key.ExpiresAt,
		},
		"$set": bson.M{
			"lockedAt": now,
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var result IdempotencyKey
	err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return nil, false, err
	}

	// Check if this was a new insert (no completedAt and lockedAt just set)
	isNew := result.CompletedAt == nil && result.CreatedAt.Equal(key.CreatedAt)

	return &result, isNew, nil
}

// ReleaseLock releases the lock on an idempotency key
func (r *MongoKeyRepository) ReleaseLock(ctx context.Context, keyID string) error {
	objID, err := primitive.ObjectIDFromHex(keyID)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": objID}
	update := bson.M{
		"$unset": bson.M{"lockedAt": ""},
	}

	_, err = r.collection.UpdateOne(ctx, filter, update)
	return err
}

// StoreResponse stores the final response for a completed request
func (r *MongoKeyRepository) StoreResponse(ctx context.Context, keyID string, responseCode int, responseBody []byte, headers map[string]string) error {
	objID, err := primitive.ObjectIDFromHex(keyID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	filter := bson.M{"_id": objID}
	update := bson.M{
		"$set": bson.M{
			"responseCode":    responseCode,
			"responseBody":    responseBody,
			"responseHeaders": headers,
			"completedAt":     now,
		},
		"$unset": bson.M{"lockedAt": ""},
	}

	_, err = r.collection.UpdateOne(ctx, filter, update)
	return err
}

// UpdateRecoveryPoint updates the recovery point for atomic phases
func (r *MongoKeyRepository) UpdateRecoveryPoint(ctx context.Context, keyID string, phase string) error {
	objID, err := primitive.ObjectIDFromHex(keyID)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": objID}
	update := bson.M{
		"$set": bson.M{
			"recoveryPoint": phase,
		},
	}

	_, err = r.collection.UpdateOne(ctx, filter, update)
	return err
}

// Get retrieves an idempotency key by its key string and service ID
func (r *MongoKeyRepository) Get(ctx context.Context, key, serviceID string) (*IdempotencyKey, error) {
	filter := bson.M{
		"serviceId": serviceID,
		"key":       key,
	}

	var result IdempotencyKey
	err := r.collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &result, nil
}

// GetByID retrieves an idempotency key by its ID
func (r *MongoKeyRepository) GetByID(ctx context.Context, keyID string) (*IdempotencyKey, error) {
	objID, err := primitive.ObjectIDFromHex(keyID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": objID}

	var result IdempotencyKey
	err = r.collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &result, nil
}

// Clean removes expired idempotency keys
func (r *MongoKeyRepository) Clean(ctx context.Context, before time.Time) (int64, error) {
	filter := bson.M{
		"expiresAt": bson.M{"$lt": before},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// EnsureIndexes ensures that all required indexes are created
func (r *MongoKeyRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "serviceId", Value: 1},
				{Key: "key", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetName("idx_service_key"),
		},
		{
			Keys: bson.D{
				{Key: "expiresAt", Value: 1},
			},
			Options: options.Index().SetExpireAfterSeconds(0).SetName("idx_ttl"),
		},
		{
			Keys: bson.D{
				{Key: "lockedAt", Value: 1},
			},
			Options: options.Index().SetSparse(true).SetName("idx_locked"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// MongoMessageRepository implements MessageRepository using MongoDB
type MongoMessageRepository struct {
	collection *mongo.Collection
}

// NewMongoMessageRepository creates a new MongoDB-backed message repository
func NewMongoMessageRepository(db *mongo.Database) *MongoMessageRepository {
	return &MongoMessageRepository{
		collection: db.Collection(processedMessagesCollection),
	}
}

// MarkProcessed marks a message as processed
func (r *MongoMessageRepository) MarkProcessed(ctx context.Context, msg *ProcessedMessage) error {
	_, err := r.collection.InsertOne(ctx, msg)
	if err != nil {
		// Check if it's a duplicate key error
		if mongo.IsDuplicateKeyError(err) {
			return ErrMessageAlreadyProcessed
		}
		return err
	}

	return nil
}

// IsProcessed checks if a message has been processed
func (r *MongoMessageRepository) IsProcessed(ctx context.Context, messageID, topic, consumerGroup string) (bool, error) {
	filter := bson.M{
		"messageId":     messageID,
		"topic":         topic,
		"consumerGroup": consumerGroup,
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Clean removes expired processed messages
func (r *MongoMessageRepository) Clean(ctx context.Context, before time.Time) (int64, error) {
	filter := bson.M{
		"expiresAt": bson.M{"$lt": before},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// EnsureIndexes ensures that all required indexes are created
func (r *MongoMessageRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "messageId", Value: 1},
				{Key: "topic", Value: 1},
				{Key: "consumerGroup", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetName("idx_msg_topic_group"),
		},
		{
			Keys: bson.D{
				{Key: "expiresAt", Value: 1},
			},
			Options: options.Index().SetExpireAfterSeconds(0).SetName("idx_ttl"),
		},
		{
			Keys: bson.D{
				{Key: "processedAt", Value: 1},
			},
			Options: options.Index().SetName("idx_processed_at"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
