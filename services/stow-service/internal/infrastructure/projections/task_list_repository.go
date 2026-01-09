package projections

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TaskListProjectionRepository handles read model persistence
type TaskListProjectionRepository interface {
	Upsert(ctx context.Context, projection *TaskListProjection) error
	UpdateFields(ctx context.Context, taskID string, updates map[string]interface{}) error
	FindByID(ctx context.Context, taskID string) (*TaskListProjection, error)
	FindAll(ctx context.Context, filter TaskListFilter, pagination Pagination) (*PagedResult[TaskListProjection], error)
	Delete(ctx context.Context, taskID string) error
	GetDashboardStats(ctx context.Context) (*StowDashboardStats, error)
	GetZoneCapacity(ctx context.Context) ([]ZoneCapacity, error)
}

// MongoTaskListRepository implements TaskListProjectionRepository using MongoDB
type MongoTaskListRepository struct {
	collection *mongo.Collection
}

// NewMongoTaskListRepository creates a new MongoDB-backed projection repository
func NewMongoTaskListRepository(db *mongo.Database) *MongoTaskListRepository {
	collection := db.Collection("task_projections")

	repo := &MongoTaskListRepository{collection: collection}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *MongoTaskListRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "taskId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "strategy", Value: 1}}},
		{Keys: bson.D{{Key: "targetZone", Value: 1}}},
		{Keys: bson.D{{Key: "assignedWorkerId", Value: 1}}},
		{Keys: bson.D{{Key: "shipmentId", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "isOverdue", Value: 1}}},
		{Keys: bson.D{
			{Key: "taskId", Value: "text"},
			{Key: "sku", Value: "text"},
		}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

// Upsert creates or updates a projection
func (r *MongoTaskListRepository) Upsert(ctx context.Context, projection *TaskListProjection) error {
	projection.UpdatedAt = time.Now().UTC()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"taskId": projection.TaskID}
	update := bson.M{"$set": projection}

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert projection: %w", err)
	}
	return nil
}

// UpdateFields updates specific fields of a projection
func (r *MongoTaskListRepository) UpdateFields(ctx context.Context, taskID string, updates map[string]interface{}) error {
	updates["updatedAt"] = time.Now().UTC()

	filter := bson.M{"taskId": taskID}
	update := bson.M{"$set": updates}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update projection: %w", err)
	}
	return nil
}

// FindByID finds a projection by task ID
func (r *MongoTaskListRepository) FindByID(ctx context.Context, taskID string) (*TaskListProjection, error) {
	var projection TaskListProjection
	err := r.collection.FindOne(ctx, bson.M{"taskId": taskID}).Decode(&projection)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find projection: %w", err)
	}
	return &projection, nil
}

// FindAll finds projections with filtering and pagination
func (r *MongoTaskListRepository) FindAll(ctx context.Context, filter TaskListFilter, pagination Pagination) (*PagedResult[TaskListProjection], error) {
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

	var projections []TaskListProjection
	if err := cursor.All(ctx, &projections); err != nil {
		return nil, fmt.Errorf("failed to decode projections: %w", err)
	}

	return &PagedResult[TaskListProjection]{
		Items:   projections,
		Total:   total,
		Limit:   pagination.Limit,
		Offset:  pagination.Offset,
		HasMore: int64(pagination.Offset+len(projections)) < total,
	}, nil
}

func (r *MongoTaskListRepository) buildFilterQuery(filter TaskListFilter) bson.M {
	query := bson.M{}

	if filter.Status != nil {
		query["status"] = *filter.Status
	}
	if filter.Strategy != nil {
		query["strategy"] = *filter.Strategy
	}
	if filter.Zone != nil {
		query["targetZone"] = *filter.Zone
	}
	if filter.AssignedWorkerID != nil {
		query["assignedWorkerId"] = *filter.AssignedWorkerID
	}
	if filter.ShipmentID != nil {
		query["shipmentId"] = *filter.ShipmentID
	}
	if filter.IsHazmat != nil {
		query["isHazmat"] = *filter.IsHazmat
	}
	if filter.RequiresColdChain != nil {
		query["requiresColdChain"] = *filter.RequiresColdChain
	}
	if filter.IsOversized != nil {
		query["isOversized"] = *filter.IsOversized
	}
	if filter.IsOverdue != nil {
		query["isOverdue"] = *filter.IsOverdue
	}
	if filter.SearchTerm != "" {
		query["$text"] = bson.M{"$search": filter.SearchTerm}
	}

	return query
}

// Delete removes a projection
func (r *MongoTaskListRepository) Delete(ctx context.Context, taskID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"taskId": taskID})
	if err != nil {
		return fmt.Errorf("failed to delete projection: %w", err)
	}
	return nil
}

// GetDashboardStats returns aggregate statistics for stow dashboard
func (r *MongoTaskListRepository) GetDashboardStats(ctx context.Context) (*StowDashboardStats, error) {
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	stats := &StowDashboardStats{
		TasksByZone: make(map[string]int64),
	}

	// Total tasks
	total, _ := r.collection.CountDocuments(ctx, bson.M{})
	stats.TotalTasks = total

	// Pending tasks
	pending, _ := r.collection.CountDocuments(ctx, bson.M{"status": "pending"})
	stats.PendingTasks = pending

	// Assigned tasks
	assigned, _ := r.collection.CountDocuments(ctx, bson.M{"status": "assigned"})
	stats.AssignedTasks = assigned

	// In progress tasks
	inProgress, _ := r.collection.CountDocuments(ctx, bson.M{"status": "in_progress"})
	stats.InProgressTasks = inProgress

	// Completed today
	completedToday, _ := r.collection.CountDocuments(ctx, bson.M{
		"status":      "completed",
		"completedAt": bson.M{"$gte": startOfDay, "$lt": endOfDay},
	})
	stats.CompletedToday = completedToday

	// Failed tasks
	failed, _ := r.collection.CountDocuments(ctx, bson.M{"status": "failed"})
	stats.FailedTasks = failed

	// Overdue tasks
	overdue, _ := r.collection.CountDocuments(ctx, bson.M{"isOverdue": true})
	stats.OverdueTasks = overdue

	// Tasks by zone aggregation
	pipeline := []bson.M{
		{"$match": bson.M{"status": bson.M{"$in": []string{"pending", "assigned", "in_progress"}}}},
		{"$group": bson.M{"_id": "$targetZone", "count": bson.M{"$sum": 1}}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err == nil {
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var result struct {
				ID    string `bson:"_id"`
				Count int64  `bson:"count"`
			}
			if err := cursor.Decode(&result); err == nil && result.ID != "" {
				stats.TasksByZone[result.ID] = result.Count
			}
		}
	}

	return stats, nil
}

// GetZoneCapacity returns capacity information by zone
func (r *MongoTaskListRepository) GetZoneCapacity(ctx context.Context) ([]ZoneCapacity, error) {
	// This would typically come from a separate locations collection
	// For now, return placeholder data
	return []ZoneCapacity{
		{Zone: "GENERAL", TotalLocations: 1000, UsedLocations: 750, AvailableLocations: 250, UtilizationPct: 75},
		{Zone: "HAZMAT", TotalLocations: 100, UsedLocations: 30, AvailableLocations: 70, UtilizationPct: 30},
		{Zone: "COLD", TotalLocations: 200, UsedLocations: 150, AvailableLocations: 50, UtilizationPct: 75},
		{Zone: "OVERSIZE", TotalLocations: 50, UsedLocations: 20, AvailableLocations: 30, UtilizationPct: 40},
	}, nil
}
