package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"github.com/wms-platform/wes-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TaskRouteRepository implements domain.TaskRouteRepository using MongoDB
type TaskRouteRepository struct {
	db           *mongo.Database
	collection   *mongo.Collection
	eventFactory *cloudevents.EventFactory
	outboxRepo   *outboxMongo.OutboxRepository
}

// NewTaskRouteRepository creates a new TaskRouteRepository
func NewTaskRouteRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *TaskRouteRepository {
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
		db:           db,
		collection:   collection,
		eventFactory: eventFactory,
		outboxRepo:   outboxMongo.NewOutboxRepository(db),
	}
}

// GetOutboxRepository returns the outbox repository
func (r *TaskRouteRepository) GetOutboxRepository() *outboxMongo.OutboxRepository {
	return r.outboxRepo
}

// Save saves a task route with event publishing via outbox (idempotent - upserts by orderID)
func (r *TaskRouteRepository) Save(ctx context.Context, route *domain.TaskRoute) error {
	// Check if route already exists for this order (for idempotency)
	// The unique index on orderID ensures one route per order
	existingRoute, err := r.FindByOrderID(ctx, route.OrderID)
	if err != nil && err.Error() != "failed to find task route: mongo: no documents in result" {
		return fmt.Errorf("failed to check existing route: %w", err)
	}

	// If route already exists, return it (idempotent behavior for Temporal retries)
	if existingRoute != nil {
		*route = *existingRoute
		return nil
	}

	// Set timestamps for new route
	route.CreatedAt = time.Now()
	route.UpdatedAt = time.Now()

	// Start MongoDB session for transaction
	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Use upsert with orderID filter for true idempotency
		// This prevents duplicate route creation on Temporal retries
		opts := options.Update().SetUpsert(true)
		filter := bson.M{"orderId": route.OrderID}
		update := bson.M{"$setOnInsert": route}

		result, err := r.collection.UpdateOne(sessCtx, filter, update, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert task route: %w", err)
		}

		// Get the document ID if this was an insert
		if result.UpsertedID != nil {
			// This was a new insert
			if oid, ok := result.UpsertedID.(primitive.ObjectID); ok {
				route.ID = oid
			}
		} else {
			// This was a retry - fetch existing route
			var existing domain.TaskRoute
			err := r.collection.FindOne(sessCtx, filter).Decode(&existing)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch existing route: %w", err)
			}
			route.ID = existing.ID
			route.RouteID = existing.RouteID
			route.CreatedAt = existing.CreatedAt
		}

		// Only publish event if this was a new insert (not a retry)
		if result.UpsertedID != nil {
			// Create CloudEvent
			cloudEvent := r.eventFactory.CreateEvent(
				sessCtx,
				"wms.wes.route-created",
				"route/"+route.RouteID,
				map[string]interface{}{
					"routeID":    route.RouteID,
					"orderID":    route.OrderID,
					"waveID":     route.WaveID,
					"stageCount": len(route.Stages),
				},
			)

			// Create outbox event from CloudEvent
			outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
				route.RouteID,
				"TaskRoute",
				kafka.Topics.WESEvents,
				cloudEvent,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create outbox event: %w", err)
			}

			// Save to outbox atomically
			if err := r.outboxRepo.Save(sessCtx, outboxEvent); err != nil {
				return nil, fmt.Errorf("failed to save event to outbox: %w", err)
			}
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
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
