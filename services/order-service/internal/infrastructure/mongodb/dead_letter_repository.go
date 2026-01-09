package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wms-platform/services/order-service/internal/domain"
)

// DeadLetterRepository implements the dead letter queue repository using MongoDB
type DeadLetterRepository struct {
	collection *mongo.Collection
}

// NewDeadLetterRepository creates a new DeadLetterRepository
func NewDeadLetterRepository(db *mongo.Database) *DeadLetterRepository {
	collection := db.Collection("order_dead_letter_queue")

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
				{Key: "resolution", Value: 1},
				{Key: "movedToQueueAt", Value: -1},
			},
		},
		{
			Keys: bson.D{{Key: "customerId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "finalFailureStatus", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "movedToQueueAt", Value: -1}},
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	return &DeadLetterRepository{
		collection: collection,
	}
}

// Create creates a new dead letter entry
func (r *DeadLetterRepository) Create(ctx context.Context, entry *domain.DeadLetterEntry) error {
	_, err := r.collection.InsertOne(ctx, entry)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("dead letter entry already exists for this order")
		}
		return fmt.Errorf("failed to create dead letter entry: %w", err)
	}
	return nil
}

// FindByOrderID retrieves a dead letter entry by order ID
func (r *DeadLetterRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.DeadLetterEntry, error) {
	var entry domain.DeadLetterEntry
	filter := bson.M{"orderId": orderID}

	err := r.collection.FindOne(ctx, filter).Decode(&entry)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find dead letter entry: %w", err)
	}

	return &entry, nil
}

// DLQFilter defines filter criteria for querying DLQ entries
type DLQFilter struct {
	Resolved       *bool
	FailureStatus  *string
	CustomerID     *string
	OlderThanHours *float64
}

// List retrieves dead letter entries matching the filter
func (r *DeadLetterRepository) List(ctx context.Context, filter DLQFilter, limit int, offset int) ([]*domain.DeadLetterEntry, error) {
	mongoFilter := r.buildFilter(filter)

	opts := options.Find().
		SetSort(bson.D{{Key: "movedToQueueAt", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, mongoFilter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list dead letter entries: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []*domain.DeadLetterEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode dead letter entries: %w", err)
	}

	return entries, nil
}

// ListUnresolved retrieves unresolved dead letter entries
func (r *DeadLetterRepository) ListUnresolved(ctx context.Context, limit int) ([]*domain.DeadLetterEntry, error) {
	filter := bson.M{
		"resolution": bson.M{"$exists": false},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "movedToQueueAt", Value: 1}}). // Oldest first
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list unresolved entries: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []*domain.DeadLetterEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode entries: %w", err)
	}

	return entries, nil
}

// Resolve marks a dead letter entry as resolved
func (r *DeadLetterRepository) Resolve(ctx context.Context, orderID string, resolution string, notes string, resolvedBy string) error {
	now := time.Now().UTC()
	filter := bson.M{
		"orderId":    orderID,
		"resolution": bson.M{"$exists": false}, // Only unresolved entries
	}
	update := bson.M{
		"$set": bson.M{
			"resolution":      resolution,
			"resolutionNotes": notes,
			"resolvedBy":      resolvedBy,
			"resolvedAt":      now,
			"updatedAt":       now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to resolve dead letter entry: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("dead letter entry not found or already resolved")
	}

	return nil
}

// DLQStats contains statistics about the dead letter queue
type DLQStats struct {
	TotalEntries     int64              `json:"totalEntries"`
	UnresolvedCount  int64              `json:"unresolvedCount"`
	ResolvedCount    int64              `json:"resolvedCount"`
	ByFailureStatus  map[string]int64   `json:"byFailureStatus"`
	ByResolution     map[string]int64   `json:"byResolution"`
	AverageRetries   float64            `json:"averageRetries"`
	OldestUnresolved *time.Time         `json:"oldestUnresolved,omitempty"`
}

// GetStats retrieves statistics about the dead letter queue
func (r *DeadLetterRepository) GetStats(ctx context.Context) (*DLQStats, error) {
	stats := &DLQStats{
		ByFailureStatus: make(map[string]int64),
		ByResolution:    make(map[string]int64),
	}

	// Total count
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count total entries: %w", err)
	}
	stats.TotalEntries = total

	// Unresolved count
	unresolvedCount, err := r.collection.CountDocuments(ctx, bson.M{
		"resolution": bson.M{"$exists": false},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count unresolved entries: %w", err)
	}
	stats.UnresolvedCount = unresolvedCount
	stats.ResolvedCount = total - unresolvedCount

	// Counts by failure status
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":   "$finalFailureStatus",
			"count": bson.M{"$sum": 1},
		}}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate by failure status: %w", err)
	}
	defer cursor.Close(ctx)

	var statusResults []struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := cursor.All(ctx, &statusResults); err != nil {
		return nil, fmt.Errorf("failed to decode status results: %w", err)
	}
	for _, r := range statusResults {
		stats.ByFailureStatus[r.ID] = r.Count
	}

	// Counts by resolution
	resolutionPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"resolution": bson.M{"$exists": true}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$resolution",
			"count": bson.M{"$sum": 1},
		}}},
	}
	resCursor, err := r.collection.Aggregate(ctx, resolutionPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate by resolution: %w", err)
	}
	defer resCursor.Close(ctx)

	var resResults []struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := resCursor.All(ctx, &resResults); err != nil {
		return nil, fmt.Errorf("failed to decode resolution results: %w", err)
	}
	for _, r := range resResults {
		stats.ByResolution[r.ID] = r.Count
	}

	// Average retries
	avgPipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":        nil,
			"avgRetries": bson.M{"$avg": "$totalRetryAttempts"},
		}}},
	}
	avgCursor, err := r.collection.Aggregate(ctx, avgPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate average retries: %w", err)
	}
	defer avgCursor.Close(ctx)

	var avgResult []struct {
		AvgRetries float64 `bson:"avgRetries"`
	}
	if err := avgCursor.All(ctx, &avgResult); err != nil {
		return nil, fmt.Errorf("failed to decode average: %w", err)
	}
	if len(avgResult) > 0 {
		stats.AverageRetries = avgResult[0].AvgRetries
	}

	// Oldest unresolved
	var oldestEntry domain.DeadLetterEntry
	opts := options.FindOne().SetSort(bson.D{{Key: "movedToQueueAt", Value: 1}})
	err = r.collection.FindOne(ctx, bson.M{"resolution": bson.M{"$exists": false}}, opts).Decode(&oldestEntry)
	if err == nil {
		stats.OldestUnresolved = &oldestEntry.MovedToQueueAt
	}

	return stats, nil
}

// Count returns the count of entries matching the filter
func (r *DeadLetterRepository) Count(ctx context.Context, filter DLQFilter) (int64, error) {
	mongoFilter := r.buildFilter(filter)
	count, err := r.collection.CountDocuments(ctx, mongoFilter)
	if err != nil {
		return 0, fmt.Errorf("failed to count dead letter entries: %w", err)
	}
	return count, nil
}

// buildFilter builds a MongoDB filter from DLQFilter
func (r *DeadLetterRepository) buildFilter(filter DLQFilter) bson.M {
	mongoFilter := bson.M{}

	if filter.Resolved != nil {
		if *filter.Resolved {
			mongoFilter["resolution"] = bson.M{"$exists": true, "$ne": ""}
		} else {
			mongoFilter["resolution"] = bson.M{"$exists": false}
		}
	}

	if filter.FailureStatus != nil {
		mongoFilter["finalFailureStatus"] = *filter.FailureStatus
	}

	if filter.CustomerID != nil {
		mongoFilter["customerId"] = *filter.CustomerID
	}

	if filter.OlderThanHours != nil {
		cutoff := time.Now().UTC().Add(-time.Duration(*filter.OlderThanHours) * time.Hour)
		mongoFilter["movedToQueueAt"] = bson.M{"$lt": cutoff}
	}

	return mongoFilter
}

// Delete removes a dead letter entry (use with caution)
func (r *DeadLetterRepository) Delete(ctx context.Context, orderID string) error {
	filter := bson.M{"orderId": orderID}
	_, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete dead letter entry: %w", err)
	}
	return nil
}
