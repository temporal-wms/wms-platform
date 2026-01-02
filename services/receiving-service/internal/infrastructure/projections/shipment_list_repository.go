package projections

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ShipmentListProjectionRepository handles read model persistence
type ShipmentListProjectionRepository interface {
	Upsert(ctx context.Context, projection *ShipmentListProjection) error
	UpdateFields(ctx context.Context, shipmentID string, updates map[string]interface{}) error
	FindByID(ctx context.Context, shipmentID string) (*ShipmentListProjection, error)
	FindAll(ctx context.Context, filter ShipmentListFilter, pagination Pagination) (*PagedResult[ShipmentListProjection], error)
	Delete(ctx context.Context, shipmentID string) error
	GetDashboardStats(ctx context.Context) (*ReceivingDashboardStats, error)
}

// MongoShipmentListRepository implements ShipmentListProjectionRepository using MongoDB
type MongoShipmentListRepository struct {
	collection *mongo.Collection
}

// NewMongoShipmentListRepository creates a new MongoDB-backed projection repository
func NewMongoShipmentListRepository(db *mongo.Database) *MongoShipmentListRepository {
	collection := db.Collection("shipment_projections")

	repo := &MongoShipmentListRepository{collection: collection}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *MongoShipmentListRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "shipmentId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "supplierId", Value: 1}}},
		{Keys: bson.D{{Key: "dockId", Value: 1}}},
		{Keys: bson.D{{Key: "expectedArrival", Value: 1}}},
		{Keys: bson.D{{Key: "arrivedAt", Value: -1}}},
		{Keys: bson.D{{Key: "isLate", Value: 1}}},
		{Keys: bson.D{{Key: "hasIssues", Value: 1}}},
		{Keys: bson.D{
			{Key: "shipmentId", Value: "text"},
			{Key: "asnId", Value: "text"},
			{Key: "supplierId", Value: "text"},
		}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

// Upsert creates or updates a projection
func (r *MongoShipmentListRepository) Upsert(ctx context.Context, projection *ShipmentListProjection) error {
	projection.UpdatedAt = time.Now().UTC()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"shipmentId": projection.ShipmentID}
	update := bson.M{"$set": projection}

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert projection: %w", err)
	}
	return nil
}

// UpdateFields updates specific fields of a projection
func (r *MongoShipmentListRepository) UpdateFields(ctx context.Context, shipmentID string, updates map[string]interface{}) error {
	updates["updatedAt"] = time.Now().UTC()

	filter := bson.M{"shipmentId": shipmentID}
	update := bson.M{"$set": updates}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update projection: %w", err)
	}
	return nil
}

// FindByID finds a projection by shipment ID
func (r *MongoShipmentListRepository) FindByID(ctx context.Context, shipmentID string) (*ShipmentListProjection, error) {
	var projection ShipmentListProjection
	err := r.collection.FindOne(ctx, bson.M{"shipmentId": shipmentID}).Decode(&projection)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find projection: %w", err)
	}
	return &projection, nil
}

// FindAll finds projections with filtering and pagination
func (r *MongoShipmentListRepository) FindAll(ctx context.Context, filter ShipmentListFilter, pagination Pagination) (*PagedResult[ShipmentListProjection], error) {
	query := r.buildFilterQuery(filter)

	// Count total
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count projections: %w", err)
	}

	// Build sort
	sortField := "createdAt"
	if pagination.SortBy != "" {
		sortField = pagination.SortBy
	}
	sortOrder := -1 // desc
	if pagination.SortOrder == "asc" {
		sortOrder = 1
	}

	// Find with pagination
	opts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: sortOrder}}).
		SetSkip(int64(pagination.Offset)).
		SetLimit(int64(pagination.Limit))

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find projections: %w", err)
	}
	defer cursor.Close(ctx)

	var projections []ShipmentListProjection
	if err := cursor.All(ctx, &projections); err != nil {
		return nil, fmt.Errorf("failed to decode projections: %w", err)
	}

	return &PagedResult[ShipmentListProjection]{
		Items:   projections,
		Total:   total,
		Limit:   pagination.Limit,
		Offset:  pagination.Offset,
		HasMore: int64(pagination.Offset+len(projections)) < total,
	}, nil
}

func (r *MongoShipmentListRepository) buildFilterQuery(filter ShipmentListFilter) bson.M {
	query := bson.M{}

	if filter.Status != nil {
		query["status"] = *filter.Status
	}
	if filter.SupplierID != nil {
		query["supplierId"] = *filter.SupplierID
	}
	if filter.DockID != nil {
		query["dockId"] = *filter.DockID
	}
	if filter.IsLate != nil {
		query["isLate"] = *filter.IsLate
	}
	if filter.HasIssues != nil {
		query["hasIssues"] = *filter.HasIssues
	}
	if filter.ExpectedAfter != nil {
		if _, ok := query["expectedArrival"]; !ok {
			query["expectedArrival"] = bson.M{}
		}
		query["expectedArrival"].(bson.M)["$gte"] = *filter.ExpectedAfter
	}
	if filter.ExpectedBefore != nil {
		if _, ok := query["expectedArrival"]; !ok {
			query["expectedArrival"] = bson.M{}
		}
		query["expectedArrival"].(bson.M)["$lte"] = *filter.ExpectedBefore
	}
	if filter.SearchTerm != "" {
		query["$text"] = bson.M{"$search": filter.SearchTerm}
	}

	return query
}

// Delete removes a projection
func (r *MongoShipmentListRepository) Delete(ctx context.Context, shipmentID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"shipmentId": shipmentID})
	if err != nil {
		return fmt.Errorf("failed to delete projection: %w", err)
	}
	return nil
}

// GetDashboardStats returns aggregate statistics for dashboard
func (r *MongoShipmentListRepository) GetDashboardStats(ctx context.Context) (*ReceivingDashboardStats, error) {
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	stats := &ReceivingDashboardStats{}

	// Total shipments
	total, _ := r.collection.CountDocuments(ctx, bson.M{})
	stats.TotalShipments = total

	// Expected today
	expectedToday, _ := r.collection.CountDocuments(ctx, bson.M{
		"expectedArrival": bson.M{"$gte": startOfDay, "$lt": endOfDay},
	})
	stats.ExpectedToday = expectedToday

	// Arrived today
	arrivedToday, _ := r.collection.CountDocuments(ctx, bson.M{
		"arrivedAt": bson.M{"$gte": startOfDay, "$lt": endOfDay},
	})
	stats.ArrivedToday = arrivedToday

	// Completed today
	completedToday, _ := r.collection.CountDocuments(ctx, bson.M{
		"completedAt": bson.M{"$gte": startOfDay, "$lt": endOfDay},
	})
	stats.CompletedToday = completedToday

	// In progress
	inProgress, _ := r.collection.CountDocuments(ctx, bson.M{
		"status": bson.M{"$in": []string{"arrived", "receiving"}},
	})
	stats.InProgress = inProgress

	// With discrepancies
	withDiscrepancies, _ := r.collection.CountDocuments(ctx, bson.M{
		"hasIssues": true,
	})
	stats.WithDiscrepancies = withDiscrepancies

	// Late shipments
	lateShipments, _ := r.collection.CountDocuments(ctx, bson.M{
		"isLate": true,
	})
	stats.LateShipments = lateShipments

	return stats, nil
}
