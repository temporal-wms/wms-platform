package mongodb

import (
	"github.com/wms-platform/shared/pkg/tenant"
	"context"
	"time"

	"github.com/wms-platform/services/order-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ProcessPathRepository implements domain.ProcessPathRepository using MongoDB
type ProcessPathRepository struct {
	collection *mongo.Collection
	tenantHelper *tenant.RepositoryHelper
}

// NewProcessPathRepository creates a new MongoDB process path repository
func NewProcessPathRepository(db *mongo.Database) *ProcessPathRepository {
	collection := db.Collection("process_paths")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "pathId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "orderId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	collection.Indexes().CreateMany(ctx, indexes)

	return &ProcessPathRepository{collection: collection}
}

// Save persists a process path
func (r *ProcessPathRepository) Save(ctx context.Context, path *domain.ProcessPath) error {
	path.UpdatedAt = time.Now()
	result, err := r.collection.InsertOne(ctx, path)
	if err != nil {
		return err
	}
	path.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByID retrieves a process path by its MongoDB ID
func (r *ProcessPathRepository) FindByID(ctx context.Context, id string) (*domain.ProcessPath, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var path domain.ProcessPath
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&path)
	if err != nil {
		return nil, err
	}
	return &path, nil
}

// FindByPathID retrieves a process path by its UUID
func (r *ProcessPathRepository) FindByPathID(ctx context.Context, pathID string) (*domain.ProcessPath, error) {
	var path domain.ProcessPath
	filter := bson.M{"pathId": pathID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	err := r.collection.FindOne(ctx, filter).Decode(&path)
	if err != nil {
		return nil, err
	}
	return &path, nil
}

// FindByOrderID retrieves the process path for an order
func (r *ProcessPathRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.ProcessPath, error) {
	var path domain.ProcessPath
	filter := bson.M{"orderId": orderID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	err := r.collection.FindOne(ctx, filter).Decode(&path)
	if err != nil {
		return nil, err
	}
	return &path, nil
}

// Update updates a process path
func (r *ProcessPathRepository) Update(ctx context.Context, path *domain.ProcessPath) error {
	path.UpdatedAt = time.Now()
	filter := bson.M{"pathId": path.PathID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	_, err := r.collection.ReplaceOne(ctx, filter, path)
	return err
}

// Delete removes a process path
func (r *ProcessPathRepository) Delete(ctx context.Context, pathID string) error {
	filter := bson.M{"pathId": pathID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}
