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

type InventoryTransactionRepository struct {
	collection   *mongo.Collection
	tenantHelper *tenant.RepositoryHelper
}

func NewInventoryTransactionRepository(db *mongo.Database) *InventoryTransactionRepository {
	collection := db.Collection("inventory_transactions")

	repo := &InventoryTransactionRepository{
		collection:   collection,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
	repo.ensureIndexes(context.Background())

	return repo
}

func (r *InventoryTransactionRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		// Primary lookup by SKU + tenant
		{Keys: bson.D{
			{Key: "tenantId", Value: 1},
			{Key: "facilityId", Value: 1},
			{Key: "sku", Value: 1},
			{Key: "createdAt", Value: -1},
		}},
		// Lookup by location
		{Keys: bson.D{
			{Key: "tenantId", Value: 1},
			{Key: "facilityId", Value: 1},
			{Key: "locationId", Value: 1},
			{Key: "createdAt", Value: -1},
		}},
		// Lookup by reference (order ID, etc.)
		{Keys: bson.D{
			{Key: "tenantId", Value: 1},
			{Key: "facilityId", Value: 1},
			{Key: "referenceId", Value: 1},
			{Key: "createdAt", Value: -1},
		}},
		// Lookup by transaction type
		{Keys: bson.D{
			{Key: "tenantId", Value: 1},
			{Key: "facilityId", Value: 1},
			{Key: "type", Value: 1},
			{Key: "createdAt", Value: -1},
		}},
		// TTL index for old transactions (archive after 90 days)
		{
			Keys: bson.D{{Key: "createdAt", Value: 1}},
			Options: options.Index().
				SetName("idx_createdAt_ttl").
				SetExpireAfterSeconds(7776000), // 90 days
		},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

func (r *InventoryTransactionRepository) Save(ctx context.Context, txn *domain.InventoryTransactionAggregate) error {
	_, err := r.collection.InsertOne(ctx, txn)
	if err != nil {
		return fmt.Errorf("failed to save inventory transaction: %w", err)
	}
	return nil
}

func (r *InventoryTransactionRepository) FindBySKU(ctx context.Context, sku string, limit int) ([]*domain.InventoryTransactionAggregate, error) {
	filter := bson.M{"sku": sku}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*domain.InventoryTransactionAggregate
	err = cursor.All(ctx, &transactions)
	return transactions, err
}

func (r *InventoryTransactionRepository) FindByLocation(ctx context.Context, locationID string, limit int) ([]*domain.InventoryTransactionAggregate, error) {
	filter := bson.M{"locationId": locationID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*domain.InventoryTransactionAggregate
	err = cursor.All(ctx, &transactions)
	return transactions, err
}

func (r *InventoryTransactionRepository) FindByReferenceID(ctx context.Context, referenceID string) ([]*domain.InventoryTransactionAggregate, error) {
	filter := bson.M{"referenceId": referenceID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*domain.InventoryTransactionAggregate
	err = cursor.All(ctx, &transactions)
	return transactions, err
}

func (r *InventoryTransactionRepository) FindByType(ctx context.Context, txnType string, startTime, endTime time.Time, limit int) ([]*domain.InventoryTransactionAggregate, error) {
	filter := bson.M{
		"type": txnType,
		"createdAt": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
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

	var transactions []*domain.InventoryTransactionAggregate
	err = cursor.All(ctx, &transactions)
	return transactions, err
}

// GetTransactionCountBySKU returns the total transaction count for a SKU
func (r *InventoryTransactionRepository) GetTransactionCountBySKU(ctx context.Context, sku string) (int64, error) {
	filter := bson.M{"sku": sku}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}
