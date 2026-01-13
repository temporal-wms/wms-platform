package mongodb

import (
	"context"
	"time"

	"github.com/wms-platform/services/unit-service/internal/domain"
	"github.com/wms-platform/shared/pkg/tenant"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UnitRepository implements domain.UnitRepository using MongoDB
type UnitRepository struct {
	collection   *mongo.Collection
	tenantHelper *tenant.RepositoryHelper
}

// NewUnitRepository creates a new MongoDB unit repository
func NewUnitRepository(db *mongo.Database) *UnitRepository {
	collection := db.Collection("units")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "unitId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "orderId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "sku", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "shipmentId", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "sku", Value: 1},
				{Key: "status", Value: 1},
			},
		},
	}

	collection.Indexes().CreateMany(ctx, indexes)

	return &UnitRepository{
		collection:   collection,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
}

// Save persists a unit
func (r *UnitRepository) Save(ctx context.Context, unit *domain.Unit) error {
	unit.UpdatedAt = time.Now()
	result, err := r.collection.InsertOne(ctx, unit)
	if err != nil {
		return err
	}
	unit.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByID retrieves a unit by its MongoDB ID
func (r *UnitRepository) FindByID(ctx context.Context, id string) (*domain.Unit, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": objectID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var unit domain.Unit
	err = r.collection.FindOne(ctx, filter).Decode(&unit)
	if err != nil {
		return nil, err
	}
	return &unit, nil
}

// FindByUnitID retrieves a unit by its UUID
func (r *UnitRepository) FindByUnitID(ctx context.Context, unitID string) (*domain.Unit, error) {
	filter := bson.M{"unitId": unitID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var unit domain.Unit
	err := r.collection.FindOne(ctx, filter).Decode(&unit)
	if err != nil {
		return nil, err
	}
	return &unit, nil
}

// FindByOrderID retrieves all units for an order
func (r *UnitRepository) FindByOrderID(ctx context.Context, orderID string) ([]*domain.Unit, error) {
	filter := bson.M{"orderId": orderID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var units []*domain.Unit
	if err := cursor.All(ctx, &units); err != nil {
		return nil, err
	}
	return units, nil
}

// FindBySKU retrieves all units for a SKU
func (r *UnitRepository) FindBySKU(ctx context.Context, sku string) ([]*domain.Unit, error) {
	filter := bson.M{"sku": sku}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var units []*domain.Unit
	if err := cursor.All(ctx, &units); err != nil {
		return nil, err
	}
	return units, nil
}

// FindByShipmentID retrieves all units from an inbound shipment
func (r *UnitRepository) FindByShipmentID(ctx context.Context, shipmentID string) ([]*domain.Unit, error) {
	filter := bson.M{"shipmentId": shipmentID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var units []*domain.Unit
	if err := cursor.All(ctx, &units); err != nil {
		return nil, err
	}
	return units, nil
}

// FindByStatus retrieves all units with a specific status
func (r *UnitRepository) FindByStatus(ctx context.Context, status domain.UnitStatus) ([]*domain.Unit, error) {
	filter := bson.M{"status": status}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var units []*domain.Unit
	if err := cursor.All(ctx, &units); err != nil {
		return nil, err
	}
	return units, nil
}

// FindAvailableBySKU retrieves available (received) units for a SKU
func (r *UnitRepository) FindAvailableBySKU(ctx context.Context, sku string, limit int) ([]*domain.Unit, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "receivedAt", Value: 1}}). // FIFO
		SetLimit(int64(limit))

	filter := bson.M{
		"sku":    sku,
		"status": domain.UnitStatusReceived,
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var units []*domain.Unit
	if err := cursor.All(ctx, &units); err != nil {
		return nil, err
	}
	return units, nil
}

// Update updates a unit
func (r *UnitRepository) Update(ctx context.Context, unit *domain.Unit) error {
	unit.UpdatedAt = time.Now()
	filter := bson.M{"unitId": unit.UnitID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	_, err := r.collection.ReplaceOne(ctx, filter, unit)
	return err
}

// Delete removes a unit
func (r *UnitRepository) Delete(ctx context.Context, unitID string) error {
	filter := bson.M{"unitId": unitID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// UnitExceptionRepository implements domain.UnitExceptionRepository using MongoDB
type UnitExceptionRepository struct {
	collection *mongo.Collection
}

// NewUnitExceptionRepository creates a new MongoDB unit exception repository
func NewUnitExceptionRepository(db *mongo.Database) *UnitExceptionRepository {
	collection := db.Collection("unit_exceptions")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "exceptionId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "unitId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "orderId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "stage", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "resolvedAt", Value: 1}},
		},
	}

	collection.Indexes().CreateMany(ctx, indexes)

	return &UnitExceptionRepository{collection: collection}
}

// Save persists a unit exception
func (r *UnitExceptionRepository) Save(ctx context.Context, exception *domain.UnitException) error {
	exception.UpdatedAt = time.Now()
	result, err := r.collection.InsertOne(ctx, exception)
	if err != nil {
		return err
	}
	exception.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByID retrieves an exception by its ID
func (r *UnitExceptionRepository) FindByID(ctx context.Context, exceptionID string) (*domain.UnitException, error) {
	var exception domain.UnitException
	err := r.collection.FindOne(ctx, bson.M{"exceptionId": exceptionID}).Decode(&exception)
	if err != nil {
		return nil, err
	}
	return &exception, nil
}

// FindByUnitID retrieves all exceptions for a unit
func (r *UnitExceptionRepository) FindByUnitID(ctx context.Context, unitID string) ([]*domain.UnitException, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"unitId": unitID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var exceptions []*domain.UnitException
	if err := cursor.All(ctx, &exceptions); err != nil {
		return nil, err
	}
	return exceptions, nil
}

// FindByOrderID retrieves all exceptions for an order
func (r *UnitExceptionRepository) FindByOrderID(ctx context.Context, orderID string) ([]*domain.UnitException, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"orderId": orderID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var exceptions []*domain.UnitException
	if err := cursor.All(ctx, &exceptions); err != nil {
		return nil, err
	}
	return exceptions, nil
}

// FindUnresolved retrieves all unresolved exceptions
func (r *UnitExceptionRepository) FindUnresolved(ctx context.Context, limit int) ([]*domain.UnitException, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, bson.M{"resolvedAt": nil}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var exceptions []*domain.UnitException
	if err := cursor.All(ctx, &exceptions); err != nil {
		return nil, err
	}
	return exceptions, nil
}

// FindByStage retrieves exceptions by process stage
func (r *UnitExceptionRepository) FindByStage(ctx context.Context, stage domain.ExceptionStage) ([]*domain.UnitException, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"stage": stage})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var exceptions []*domain.UnitException
	if err := cursor.All(ctx, &exceptions); err != nil {
		return nil, err
	}
	return exceptions, nil
}

// Update updates an exception
func (r *UnitExceptionRepository) Update(ctx context.Context, exception *domain.UnitException) error {
	exception.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"exceptionId": exception.ExceptionID}, exception)
	return err
}
