package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wms-platform/process-path-service/internal/domain"
)

const collectionName = "process_paths"

// ProcessPathRepository implements domain.ProcessPathRepository for MongoDB
type ProcessPathRepository struct {
	collection *mongo.Collection
}

// NewProcessPathRepository creates a new MongoDB process path repository
func NewProcessPathRepository(db *mongo.Database) *ProcessPathRepository {
	collection := db.Collection(collectionName)

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

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	return &ProcessPathRepository{
		collection: collection,
	}
}

// Save persists a process path to MongoDB
func (r *ProcessPathRepository) Save(ctx context.Context, processPath *domain.ProcessPath) error {
	_, err := r.collection.InsertOne(ctx, processPath)
	if err != nil {
		return fmt.Errorf("failed to save process path: %w", err)
	}
	return nil
}

// FindByID retrieves a process path by its MongoDB _id
func (r *ProcessPathRepository) FindByID(ctx context.Context, id string) (*domain.ProcessPath, error) {
	var processPath domain.ProcessPath
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&processPath)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("process path not found with id: %s", id)
		}
		return nil, fmt.Errorf("failed to find process path: %w", err)
	}
	return &processPath, nil
}

// FindByPathID retrieves a process path by pathId
func (r *ProcessPathRepository) FindByPathID(ctx context.Context, pathID string) (*domain.ProcessPath, error) {
	var processPath domain.ProcessPath
	err := r.collection.FindOne(ctx, bson.M{"pathId": pathID}).Decode(&processPath)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("process path not found with pathId: %s", pathID)
		}
		return nil, fmt.Errorf("failed to find process path: %w", err)
	}
	return &processPath, nil
}

// FindByOrderID retrieves a process path by order ID
func (r *ProcessPathRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.ProcessPath, error) {
	var processPath domain.ProcessPath
	err := r.collection.FindOne(ctx, bson.M{"orderId": orderID}).Decode(&processPath)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("process path not found for order: %s", orderID)
		}
		return nil, fmt.Errorf("failed to find process path: %w", err)
	}
	return &processPath, nil
}

// Update updates an existing process path
func (r *ProcessPathRepository) Update(ctx context.Context, processPath *domain.ProcessPath) error {
	processPath.UpdatedAt = time.Now()
	result, err := r.collection.ReplaceOne(
		ctx,
		bson.M{"pathId": processPath.PathID},
		processPath,
	)
	if err != nil {
		return fmt.Errorf("failed to update process path: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("process path not found with pathId: %s", processPath.PathID)
	}
	return nil
}
