package mongodb

import (
	"github.com/wms-platform/shared/pkg/tenant"
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/walling-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// WallingTaskRepository implements domain.WallingTaskRepository using MongoDB
type WallingTaskRepository struct {
	collection   *mongo.Collection
	tenantHelper *tenant.RepositoryHelper
}

// NewWallingTaskRepository creates a new WallingTaskRepository
func NewWallingTaskRepository(db *mongo.Database) *WallingTaskRepository {
	collection := db.Collection("walling_tasks")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "taskId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "orderId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "wallinerId", Value: 1}, {Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "putWallId", Value: 1}, {Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	return &WallingTaskRepository{
		collection:   collection,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
}

// Save saves a walling task
func (r *WallingTaskRepository) Save(ctx context.Context, task *domain.WallingTask) error {
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to insert walling task: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		task.ID = oid
	}

	return nil
}

// FindByID finds a task by its MongoDB ObjectID
func (r *WallingTaskRepository) FindByID(ctx context.Context, id string) (*domain.WallingTask, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid object id: %w", err)
	}

	var task domain.WallingTask
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find walling task: %w", err)
	}

	return &task, nil
}

// FindByTaskID finds a task by its task ID
func (r *WallingTaskRepository) FindByTaskID(ctx context.Context, taskID string) (*domain.WallingTask, error) {
	var task domain.WallingTask
	filter := bson.M{"taskId": taskID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	err := r.collection.FindOne(ctx, filter).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find walling task: %w", err)
	}

	return &task, nil
}

// FindByOrderID finds a task by order ID
func (r *WallingTaskRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.WallingTask, error) {
	var task domain.WallingTask
	filter := bson.M{"orderId": orderID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	err := r.collection.FindOne(ctx, filter).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find walling task: %w", err)
	}

	return &task, nil
}

// FindActiveByWalliner finds the active task for a walliner
func (r *WallingTaskRepository) FindActiveByWalliner(ctx context.Context, wallinerID string) (*domain.WallingTask, error) {
	var task domain.WallingTask
	err := r.collection.FindOne(ctx, bson.M{
		"wallinerId": wallinerID,
		"status":     bson.M{"$in": []string{string(domain.WallingTaskStatusAssigned), string(domain.WallingTaskStatusInProgress)}},
	}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find active walling task: %w", err)
	}

	return &task, nil
}

// FindPendingByPutWall finds pending tasks for a put wall
func (r *WallingTaskRepository) FindPendingByPutWall(ctx context.Context, putWallID string, limit int) ([]*domain.WallingTask, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "priority", Value: -1}, {Key: "createdAt", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, bson.M{
		"putWallId": putWallID,
		"status":    domain.WallingTaskStatusPending,
	}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find pending walling tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []*domain.WallingTask
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode walling tasks: %w", err)
	}

	return tasks, nil
}

// FindByStatus finds tasks by status
func (r *WallingTaskRepository) FindByStatus(ctx context.Context, status domain.WallingTaskStatus) ([]*domain.WallingTask, error) {
	filter := bson.M{"status": status}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find walling tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []*domain.WallingTask
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode walling tasks: %w", err)
	}

	return tasks, nil
}

// Update updates a walling task
func (r *WallingTaskRepository) Update(ctx context.Context, task *domain.WallingTask) error {
	task.UpdatedAt = time.Now()

	result, err := r.collection.ReplaceOne(
		ctx,
		bson.M{"taskId": task.TaskID},
		task,
	)
	if err != nil {
		return fmt.Errorf("failed to update walling task: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("walling task not found: %s", task.TaskID)
	}

	return nil
}
