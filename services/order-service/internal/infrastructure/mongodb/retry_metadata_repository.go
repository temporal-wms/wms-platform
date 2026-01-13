package mongodb

import (
	"github.com/wms-platform/shared/pkg/tenant"
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wms-platform/services/order-service/internal/domain"
)

// RetryMetadataRepository implements the retry metadata repository using MongoDB
type RetryMetadataRepository struct {
	collection *mongo.Collection
	tenantHelper *tenant.RepositoryHelper
}

// NewRetryMetadataRepository creates a new RetryMetadataRepository
func NewRetryMetadataRepository(db *mongo.Database) *RetryMetadataRepository {
	collection := db.Collection("order_retry_metadata")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "orderId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "failureStatus", Value: 1},
				{Key: "retryCount", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "lastFailureAt", Value: -1}},
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	return &RetryMetadataRepository{
		collection: collection,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
}

// Save saves or updates retry metadata for an order
func (r *RetryMetadataRepository) Save(ctx context.Context, metadata *domain.RetryMetadata) error {
	metadata.UpdatedAt = time.Now().UTC()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"orderId": metadata.OrderID}
	update := bson.M{"$set": metadata}

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to save retry metadata: %w", err)
	}

	return nil
}

// FindByOrderID retrieves retry metadata for a specific order
func (r *RetryMetadataRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.RetryMetadata, error) {
	var metadata domain.RetryMetadata
	filter := bson.M{"orderId": orderID}

	err := r.collection.FindOne(ctx, filter).Decode(&metadata)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find retry metadata: %w", err)
	}

	return &metadata, nil
}

// IncrementRetryCount increments the retry count for an order and updates failure info
func (r *RetryMetadataRepository) IncrementRetryCount(ctx context.Context, orderID string, failureStatus string, failureReason string, workflowID string, runID string) error {
	now := time.Now().UTC()
	filter := bson.M{"orderId": orderID}
	update := bson.M{
		"$inc": bson.M{"retryCount": 1},
		"$set": bson.M{
			"lastFailureAt":  now,
			"failureStatus":  failureStatus,
			"failureReason":  failureReason,
			"lastWorkflowId": workflowID,
			"lastRunId":      runID,
			"updatedAt":      now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("retry metadata not found for order")
	}

	return nil
}

// FindOrdersEligibleForRetry finds orders that can be retried
func (r *RetryMetadataRepository) FindOrdersEligibleForRetry(ctx context.Context, failureStatuses []string, maxRetries int, limit int) ([]*domain.RetryMetadata, error) {
	filter := bson.M{
		"failureStatus": bson.M{"$in": failureStatuses},
		"retryCount":    bson.M{"$lt": maxRetries},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "lastFailureAt", Value: 1}}). // Oldest failures first
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find eligible orders: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*domain.RetryMetadata
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode retry metadata: %w", err)
	}

	return results, nil
}

// FindOrdersForDeadLetter finds orders that have exceeded max retries
func (r *RetryMetadataRepository) FindOrdersForDeadLetter(ctx context.Context, failureStatuses []string, maxRetries int, limit int) ([]*domain.RetryMetadata, error) {
	filter := bson.M{
		"failureStatus": bson.M{"$in": failureStatuses},
		"retryCount":    bson.M{"$gte": maxRetries},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "lastFailureAt", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find orders for dead letter: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*domain.RetryMetadata
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode retry metadata: %w", err)
	}

	return results, nil
}

// Delete removes retry metadata for an order
func (r *RetryMetadataRepository) Delete(ctx context.Context, orderID string) error {
	filter := bson.M{"orderId": orderID}
	_, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete retry metadata: %w", err)
	}
	return nil
}

// Count returns the count of retry metadata entries matching the filter
func (r *RetryMetadataRepository) Count(ctx context.Context, failureStatuses []string) (int64, error) {
	filter := bson.M{}
	if len(failureStatuses) > 0 {
		filter["failureStatus"] = bson.M{"$in": failureStatuses}
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count retry metadata: %w", err)
	}

	return count, nil
}
