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

type InventoryReservationRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	tenantHelper *tenant.RepositoryHelper
}

func NewInventoryReservationRepository(db *mongo.Database) *InventoryReservationRepository {
	collection := db.Collection("inventory_reservations")

	repo := &InventoryReservationRepository{
		collection:   collection,
		db:           db,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
	repo.ensureIndexes(context.Background())

	return repo
}

func (r *InventoryReservationRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		// Unique constraint on reservationId
		{
			Keys:    bson.D{{Key: "reservationId", Value: 1}},
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
		// Lookup by location
		{Keys: bson.D{
			{Key: "tenantId", Value: 1},
			{Key: "facilityId", Value: 1},
			{Key: "locationId", Value: 1},
			{Key: "status", Value: 1},
		}},
		// Expiration index for active reservations
		{Keys: bson.D{
			{Key: "status", Value: 1},
			{Key: "expiresAt", Value: 1},
		}},
		// TTL index for old fulfilled/cancelled reservations (archive after 30 days)
		{
			Keys: bson.D{{Key: "updatedAt", Value: 1}},
			Options: options.Index().
				SetName("idx_updatedAt_ttl").
				SetPartialFilterExpression(bson.M{
					"status": bson.M{"$in": []string{"fulfilled", "cancelled", "expired"}},
				}).
				SetExpireAfterSeconds(2592000), // 30 days
		},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

func (r *InventoryReservationRepository) Save(ctx context.Context, reservation *domain.InventoryReservationAggregate) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"reservationId": reservation.ReservationID}
	update := bson.M{"$set": reservation}

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to save reservation: %w", err)
	}
	return nil
}

func (r *InventoryReservationRepository) FindByID(ctx context.Context, reservationID string) (*domain.InventoryReservationAggregate, error) {
	filter := bson.M{"reservationId": reservationID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var reservation domain.InventoryReservationAggregate
	err := r.collection.FindOne(ctx, filter).Decode(&reservation)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &reservation, err
}

func (r *InventoryReservationRepository) FindBySKU(ctx context.Context, sku string, status domain.ReservationStatus) ([]*domain.InventoryReservationAggregate, error) {
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

	var reservations []*domain.InventoryReservationAggregate
	err = cursor.All(ctx, &reservations)
	return reservations, err
}

func (r *InventoryReservationRepository) FindByOrderID(ctx context.Context, orderID string) ([]*domain.InventoryReservationAggregate, error) {
	filter := bson.M{"orderId": orderID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reservations []*domain.InventoryReservationAggregate
	err = cursor.All(ctx, &reservations)
	return reservations, err
}

func (r *InventoryReservationRepository) FindByLocation(ctx context.Context, locationID string, status domain.ReservationStatus) ([]*domain.InventoryReservationAggregate, error) {
	filter := bson.M{"locationId": locationID}
	if status != "" {
		filter["status"] = status
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reservations []*domain.InventoryReservationAggregate
	err = cursor.All(ctx, &reservations)
	return reservations, err
}

// FindExpired finds all active reservations that have expired
func (r *InventoryReservationRepository) FindExpired(ctx context.Context, limit int) ([]*domain.InventoryReservationAggregate, error) {
	filter := bson.M{
		"status":    domain.ReservationStatusActive,
		"expiresAt": bson.M{"$lt": time.Now()},
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().
		SetSort(bson.D{{Key: "expiresAt", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reservations []*domain.InventoryReservationAggregate
	err = cursor.All(ctx, &reservations)
	return reservations, err
}

// GetActiveReservationCountBySKU returns the count of active reservations for a SKU
func (r *InventoryReservationRepository) GetActiveReservationCountBySKU(ctx context.Context, sku string) (int64, error) {
	filter := bson.M{
		"sku":    sku,
		"status": domain.ReservationStatusActive,
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count reservations: %w", err)
	}

	return count, nil
}

// UpdateStatus updates the status of a reservation (bulk operation)
func (r *InventoryReservationRepository) UpdateStatus(ctx context.Context, reservationID string, newStatus domain.ReservationStatus) error {
	filter := bson.M{"reservationId": reservationID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	update := bson.M{
		"$set": bson.M{
			"status":    newStatus,
			"updatedAt": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update reservation status: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.ErrReservationNotFound
	}

	return nil
}
