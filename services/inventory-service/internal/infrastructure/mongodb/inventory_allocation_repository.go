package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/inventory-service/internal/domain"
	"github.com/wms-platform/shared/pkg/tenant"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InventoryAllocationRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	tenantHelper *tenant.RepositoryHelper
}

func NewInventoryAllocationRepository(db *mongo.Database) *InventoryAllocationRepository {
	collection := db.Collection("inventory_allocations")

	repo := &InventoryAllocationRepository{
		collection:   collection,
		db:           db,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
	repo.ensureIndexes(context.Background())

	return repo
}

func (r *InventoryAllocationRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		// Unique constraint on allocationId
		{
			Keys:    bson.D{{Key: "allocationId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		// Primary lookup by SKU + status + tenant
		{Keys: bson.D{
			{Key: "tenantId", Value: 1},
			{Key: "facilityId", Value: 1},
			{Key: "sku", Value: 1},
			{Key: "status", Value: 1},
		}},
		// Lookup by order ID
		{Keys: bson.D{
			{Key: "tenantId", Value: 1},
			{Key: "facilityId", Value: 1},
			{Key: "orderId", Value: 1},
		}},
		// Lookup by reservation ID
		{Keys: bson.D{
			{Key: "reservationId", Value: 1},
		}},
		// Lookup by source location
		{Keys: bson.D{
			{Key: "tenantId", Value: 1},
			{Key: "facilityId", Value: 1},
			{Key: "sourceLocationId", Value: 1},
			{Key: "status", Value: 1},
		}},
		// Lookup by staging location
		{Keys: bson.D{
			{Key: "tenantId", Value: 1},
			{Key: "facilityId", Value: 1},
			{Key: "stagingLocationId", Value: 1},
			{Key: "status", Value: 1},
		}},
		// TTL index for old shipped/returned allocations (archive after 30 days)
		{
			Keys: bson.D{{Key: "updatedAt", Value: 1}},
			Options: options.Index().
				SetName("idx_updatedAt_ttl").
				SetPartialFilterExpression(bson.M{
					"status": bson.M{"$in": []string{"shipped", "returned"}},
				}).
				SetExpireAfterSeconds(2592000), // 30 days
		},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

func (r *InventoryAllocationRepository) Save(ctx context.Context, allocation *domain.InventoryAllocationAggregate) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"allocationId": allocation.AllocationID}
	update := bson.M{"$set": allocation}

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to save allocation: %w", err)
	}
	return nil
}

func (r *InventoryAllocationRepository) FindByID(ctx context.Context, allocationID string) (*domain.InventoryAllocationAggregate, error) {
	filter := bson.M{"allocationId": allocationID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var allocation domain.InventoryAllocationAggregate
	err := r.collection.FindOne(ctx, filter).Decode(&allocation)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &allocation, err
}

func (r *InventoryAllocationRepository) FindByReservationID(ctx context.Context, reservationID string) (*domain.InventoryAllocationAggregate, error) {
	filter := bson.M{"reservationId": reservationID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var allocation domain.InventoryAllocationAggregate
	err := r.collection.FindOne(ctx, filter).Decode(&allocation)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &allocation, err
}

func (r *InventoryAllocationRepository) FindBySKU(ctx context.Context, sku string, status domain.AllocationStatus) ([]*domain.InventoryAllocationAggregate, error) {
	filter := bson.M{"sku": sku}
	if status != "" {
		filter["status"] = status
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var allocations []*domain.InventoryAllocationAggregate
	err = cursor.All(ctx, &allocations)
	return allocations, err
}

func (r *InventoryAllocationRepository) FindByOrderID(ctx context.Context, orderID string) ([]*domain.InventoryAllocationAggregate, error) {
	filter := bson.M{"orderId": orderID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var allocations []*domain.InventoryAllocationAggregate
	err = cursor.All(ctx, &allocations)
	return allocations, err
}

func (r *InventoryAllocationRepository) FindByLocation(ctx context.Context, locationID string, locationType string, status domain.AllocationStatus) ([]*domain.InventoryAllocationAggregate, error) {
	filter := bson.M{}

	// locationType can be "source" or "staging"
	if locationType == "source" {
		filter["sourceLocationId"] = locationID
	} else if locationType == "staging" {
		filter["stagingLocationId"] = locationID
	} else {
		// Search both
		filter["$or"] = []bson.M{
			{"sourceLocationId": locationID},
			{"stagingLocationId": locationID},
		}
	}

	if status != "" {
		filter["status"] = status
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var allocations []*domain.InventoryAllocationAggregate
	err = cursor.All(ctx, &allocations)
	return allocations, err
}

// FindActive finds all active (staged or packed) allocations
func (r *InventoryAllocationRepository) FindActive(ctx context.Context, limit int) ([]*domain.InventoryAllocationAggregate, error) {
	filter := bson.M{
		"status": bson.M{"$in": []string{string(domain.AllocationStatusStaged), string(domain.AllocationStatusPacked)}},
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var allocations []*domain.InventoryAllocationAggregate
	err = cursor.All(ctx, &allocations)
	return allocations, err
}

// GetActiveAllocationCountBySKU returns the count of active allocations for a SKU
func (r *InventoryAllocationRepository) GetActiveAllocationCountBySKU(ctx context.Context, sku string) (int64, error) {
	filter := bson.M{
		"sku":    sku,
		"status": bson.M{"$in": []string{string(domain.AllocationStatusStaged), string(domain.AllocationStatusPacked)}},
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count allocations: %w", err)
	}

	return count, nil
}

// UpdateStatus updates the status of an allocation (bulk operation)
func (r *InventoryAllocationRepository) UpdateStatus(ctx context.Context, allocationID string, newStatus domain.AllocationStatus) error {
	filter := bson.M{"allocationId": allocationID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	update := bson.M{
		"$set": bson.M{
			"status":    newStatus,
			"updatedAt": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update allocation status: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.ErrAllocationNotFound
	}

	return nil
}
