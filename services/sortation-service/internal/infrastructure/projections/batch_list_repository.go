package projections

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BatchListProjectionRepository handles read model persistence
type BatchListProjectionRepository interface {
	Upsert(ctx context.Context, projection *BatchListProjection) error
	UpdateFields(ctx context.Context, batchID string, updates map[string]interface{}) error
	FindByID(ctx context.Context, batchID string) (*BatchListProjection, error)
	FindAll(ctx context.Context, filter BatchListFilter, pagination Pagination) (*PagedResult[BatchListProjection], error)
	Delete(ctx context.Context, batchID string) error
	GetDashboardStats(ctx context.Context) (*SortationDashboardStats, error)
	GetChuteStatuses(ctx context.Context) ([]ChuteStatus, error)
	GetDestinationSummary(ctx context.Context) ([]DestinationSummary, error)
}

// MongoBatchListRepository implements BatchListProjectionRepository using MongoDB
type MongoBatchListRepository struct {
	collection *mongo.Collection
}

// NewMongoBatchListRepository creates a new MongoDB-backed projection repository
func NewMongoBatchListRepository(db *mongo.Database) *MongoBatchListRepository {
	collection := db.Collection("batch_projections")

	repo := &MongoBatchListRepository{collection: collection}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *MongoBatchListRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "batchId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "sortationCenter", Value: 1}}},
		{Keys: bson.D{{Key: "destinationGroup", Value: 1}}},
		{Keys: bson.D{{Key: "carrierId", Value: 1}}},
		{Keys: bson.D{{Key: "assignedChuteId", Value: 1}}},
		{Keys: bson.D{{Key: "trailerId", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "isReady", Value: 1}}},
		{Keys: bson.D{{Key: "isDispatched", Value: 1}}},
		{Keys: bson.D{
			{Key: "batchId", Value: "text"},
			{Key: "destinationGroup", Value: "text"},
		}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

// Upsert creates or updates a projection
func (r *MongoBatchListRepository) Upsert(ctx context.Context, projection *BatchListProjection) error {
	projection.UpdatedAt = time.Now().UTC()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"batchId": projection.BatchID}
	update := bson.M{"$set": projection}

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert projection: %w", err)
	}
	return nil
}

// UpdateFields updates specific fields of a projection
func (r *MongoBatchListRepository) UpdateFields(ctx context.Context, batchID string, updates map[string]interface{}) error {
	updates["updatedAt"] = time.Now().UTC()

	filter := bson.M{"batchId": batchID}
	update := bson.M{"$set": updates}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update projection: %w", err)
	}
	return nil
}

// FindByID finds a projection by batch ID
func (r *MongoBatchListRepository) FindByID(ctx context.Context, batchID string) (*BatchListProjection, error) {
	var projection BatchListProjection
	err := r.collection.FindOne(ctx, bson.M{"batchId": batchID}).Decode(&projection)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find projection: %w", err)
	}
	return &projection, nil
}

// FindAll finds projections with filtering and pagination
func (r *MongoBatchListRepository) FindAll(ctx context.Context, filter BatchListFilter, pagination Pagination) (*PagedResult[BatchListProjection], error) {
	query := r.buildFilterQuery(filter)

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count projections: %w", err)
	}

	sortField := "createdAt"
	if pagination.SortBy != "" {
		sortField = pagination.SortBy
	}
	sortOrder := -1
	if pagination.SortOrder == "asc" {
		sortOrder = 1
	}

	opts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: sortOrder}}).
		SetSkip(int64(pagination.Offset)).
		SetLimit(int64(pagination.Limit))

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find projections: %w", err)
	}
	defer cursor.Close(ctx)

	var projections []BatchListProjection
	if err := cursor.All(ctx, &projections); err != nil {
		return nil, fmt.Errorf("failed to decode projections: %w", err)
	}

	return &PagedResult[BatchListProjection]{
		Items:   projections,
		Total:   total,
		Limit:   pagination.Limit,
		Offset:  pagination.Offset,
		HasMore: int64(pagination.Offset+len(projections)) < total,
	}, nil
}

func (r *MongoBatchListRepository) buildFilterQuery(filter BatchListFilter) bson.M {
	query := bson.M{}

	if filter.Status != nil {
		query["status"] = *filter.Status
	}
	if filter.SortationCenter != nil {
		query["sortationCenter"] = *filter.SortationCenter
	}
	if filter.DestinationGroup != nil {
		query["destinationGroup"] = *filter.DestinationGroup
	}
	if filter.CarrierID != nil {
		query["carrierId"] = *filter.CarrierID
	}
	if filter.ChuteID != nil {
		query["assignedChuteId"] = *filter.ChuteID
	}
	if filter.TrailerID != nil {
		query["trailerId"] = *filter.TrailerID
	}
	if filter.IsReady != nil {
		query["isReady"] = *filter.IsReady
	}
	if filter.IsDispatched != nil {
		query["isDispatched"] = *filter.IsDispatched
	}
	if filter.SearchTerm != "" {
		query["$text"] = bson.M{"$search": filter.SearchTerm}
	}

	return query
}

// Delete removes a projection
func (r *MongoBatchListRepository) Delete(ctx context.Context, batchID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"batchId": batchID})
	if err != nil {
		return fmt.Errorf("failed to delete projection: %w", err)
	}
	return nil
}

// GetDashboardStats returns aggregate statistics for sortation dashboard
func (r *MongoBatchListRepository) GetDashboardStats(ctx context.Context) (*SortationDashboardStats, error) {
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	stats := &SortationDashboardStats{
		BatchesByCarrier: make(map[string]int64),
		BatchesByChute:   make(map[string]int64),
	}

	// Total batches
	total, _ := r.collection.CountDocuments(ctx, bson.M{})
	stats.TotalBatches = total

	// Open batches
	open, _ := r.collection.CountDocuments(ctx, bson.M{"status": "open"})
	stats.OpenBatches = open

	// Ready batches
	ready, _ := r.collection.CountDocuments(ctx, bson.M{"isReady": true, "isDispatched": false})
	stats.ReadyBatches = ready

	// Dispatched today
	dispatchedToday, _ := r.collection.CountDocuments(ctx, bson.M{
		"isDispatched": true,
		"dispatchedAt": bson.M{"$gte": startOfDay, "$lt": endOfDay},
	})
	stats.DispatchedToday = dispatchedToday

	// Total packages sorted (aggregation)
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":           nil,
			"totalPackages": bson.M{"$sum": "$sortedPackages"},
			"totalWeight":   bson.M{"$sum": "$totalWeight"},
		}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err == nil {
		defer cursor.Close(ctx)
		if cursor.Next(ctx) {
			var result struct {
				TotalPackages int64   `bson:"totalPackages"`
				TotalWeight   float64 `bson:"totalWeight"`
			}
			if err := cursor.Decode(&result); err == nil {
				stats.TotalPackagesSorted = result.TotalPackages
				stats.TotalWeightKg = result.TotalWeight
			}
		}
	}

	// Batches by carrier
	carrierPipeline := []bson.M{
		{"$match": bson.M{"status": bson.M{"$ne": "dispatched"}}},
		{"$group": bson.M{"_id": "$carrierId", "count": bson.M{"$sum": 1}}},
	}
	carrierCursor, err := r.collection.Aggregate(ctx, carrierPipeline)
	if err == nil {
		defer carrierCursor.Close(ctx)
		for carrierCursor.Next(ctx) {
			var result struct {
				ID    string `bson:"_id"`
				Count int64  `bson:"count"`
			}
			if err := carrierCursor.Decode(&result); err == nil && result.ID != "" {
				stats.BatchesByCarrier[result.ID] = result.Count
			}
		}
	}

	return stats, nil
}

// GetChuteStatuses returns status information for all chutes
func (r *MongoBatchListRepository) GetChuteStatuses(ctx context.Context) ([]ChuteStatus, error) {
	// This would aggregate from batch data to show chute statuses
	// For now, return placeholder data
	return []ChuteStatus{
		{ChuteID: "CHUTE-01", ChuteNumber: 1, Zone: "ZONE-A", IsActive: true, IsFull: false},
		{ChuteID: "CHUTE-02", ChuteNumber: 2, Zone: "ZONE-A", IsActive: true, IsFull: false},
		{ChuteID: "CHUTE-03", ChuteNumber: 3, Zone: "ZONE-A", IsActive: true, IsFull: true},
		{ChuteID: "CHUTE-04", ChuteNumber: 4, Zone: "ZONE-B", IsActive: true, IsFull: false},
		{ChuteID: "CHUTE-05", ChuteNumber: 5, Zone: "ZONE-B", IsActive: false, IsFull: false},
	}, nil
}

// GetDestinationSummary returns summary by destination group
func (r *MongoBatchListRepository) GetDestinationSummary(ctx context.Context) ([]DestinationSummary, error) {
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":           "$destinationGroup",
			"totalBatches":  bson.M{"$sum": 1},
			"totalPackages": bson.M{"$sum": "$totalPackages"},
			"totalWeight":   bson.M{"$sum": "$totalWeight"},
			"readyBatches":  bson.M{"$sum": bson.M{"$cond": []interface{}{"$isReady", 1, 0}}},
		}},
		{"$sort": bson.M{"totalPackages": -1}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate destination summary: %w", err)
	}
	defer cursor.Close(ctx)

	var summaries []DestinationSummary
	for cursor.Next(ctx) {
		var result struct {
			ID            string  `bson:"_id"`
			TotalBatches  int64   `bson:"totalBatches"`
			TotalPackages int64   `bson:"totalPackages"`
			TotalWeight   float64 `bson:"totalWeight"`
			ReadyBatches  int64   `bson:"readyBatches"`
		}
		if err := cursor.Decode(&result); err == nil {
			summaries = append(summaries, DestinationSummary{
				DestinationGroup: result.ID,
				TotalBatches:     result.TotalBatches,
				TotalPackages:    result.TotalPackages,
				TotalWeight:      result.TotalWeight,
				ReadyBatches:     result.ReadyBatches,
			})
		}
	}

	return summaries, nil
}
