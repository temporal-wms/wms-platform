package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/wes-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TaskRouteRepository implements domain.TaskRouteRepository using MongoDB
type TaskRouteRepository struct {
	collection *mongo.Collection
}

// NewTaskRouteRepository creates a new TaskRouteRepository
func NewTaskRouteRepository(db *mongo.Database) *TaskRouteRepository {
	collection := db.Collection("task_routes")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "routeId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "orderId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "waveId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	return &TaskRouteRepository{
		collection: collection,
	}
}

// Save saves a task route
func (r *TaskRouteRepository) Save(ctx context.Context, route *domain.TaskRoute) error {
	route.CreatedAt = time.Now()
	route.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, route)
	if err != nil {
		return fmt.Errorf("failed to insert task route: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		route.ID = oid
	}

	return nil
}

// FindByID finds a route by its MongoDB ObjectID
func (r *TaskRouteRepository) FindByID(ctx context.Context, id string) (*domain.TaskRoute, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid object id: %w", err)
	}

	var route domain.TaskRoute
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&route)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find task route: %w", err)
	}

	return &route, nil
}

// FindByRouteID finds a route by its route ID
func (r *TaskRouteRepository) FindByRouteID(ctx context.Context, routeID string) (*domain.TaskRoute, error) {
	var route domain.TaskRoute
	err := r.collection.FindOne(ctx, bson.M{"routeId": routeID}).Decode(&route)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find task route: %w", err)
	}

	return &route, nil
}

// FindByOrderID finds a route by order ID
func (r *TaskRouteRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.TaskRoute, error) {
	var route domain.TaskRoute
	err := r.collection.FindOne(ctx, bson.M{"orderId": orderID}).Decode(&route)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find task route: %w", err)
	}

	return &route, nil
}

// FindByWaveID finds routes by wave ID
func (r *TaskRouteRepository) FindByWaveID(ctx context.Context, waveID string) ([]*domain.TaskRoute, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"waveId": waveID})
	if err != nil {
		return nil, fmt.Errorf("failed to find task routes: %w", err)
	}
	defer cursor.Close(ctx)

	var routes []*domain.TaskRoute
	if err := cursor.All(ctx, &routes); err != nil {
		return nil, fmt.Errorf("failed to decode task routes: %w", err)
	}

	return routes, nil
}

// FindByStatus finds routes by status
func (r *TaskRouteRepository) FindByStatus(ctx context.Context, status domain.RouteStatus) ([]*domain.TaskRoute, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"status": status}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find task routes: %w", err)
	}
	defer cursor.Close(ctx)

	var routes []*domain.TaskRoute
	if err := cursor.All(ctx, &routes); err != nil {
		return nil, fmt.Errorf("failed to decode task routes: %w", err)
	}

	return routes, nil
}

// FindInProgress finds all in-progress routes
func (r *TaskRouteRepository) FindInProgress(ctx context.Context) ([]*domain.TaskRoute, error) {
	return r.FindByStatus(ctx, domain.RouteStatusInProgress)
}

// Update updates a task route
func (r *TaskRouteRepository) Update(ctx context.Context, route *domain.TaskRoute) error {
	route.UpdatedAt = time.Now()

	result, err := r.collection.ReplaceOne(
		ctx,
		bson.M{"routeId": route.RouteID},
		route,
	)
	if err != nil {
		return fmt.Errorf("failed to update task route: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("task route not found: %s", route.RouteID)
	}

	return nil
}
