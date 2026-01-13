package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/routing-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"github.com/wms-platform/shared/pkg/tenant"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// RouteRepository implements domain.RouteRepository using MongoDB
type RouteRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
	tenantHelper *tenant.RepositoryHelper
}

// NewRouteRepository creates a new RouteRepository
func NewRouteRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *RouteRepository {
	collection := db.Collection("routes")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &RouteRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
	repo.ensureIndexes(context.Background())

	// Create outbox indexes
	_ = outboxRepo.EnsureIndexes(context.Background())

	return repo
}

// ensureIndexes creates the necessary indexes
func (r *RouteRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "routeId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "orderId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "waveId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "pickerId", Value: 1}, {Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "zone", Value: 1}, {Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: -1}},
		},
	}

	r.collection.Indexes().CreateMany(ctx, indexes)
}

// Save persists a route with its domain events in a single transaction
func (r *RouteRepository) Save(ctx context.Context, route *domain.PickRoute) error {
	route.UpdatedAt = time.Now()

	// Start a MongoDB session for transaction
	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// 1. Save the aggregate
		opts := options.Update().SetUpsert(true)
		filter := bson.M{"routeId": route.RouteID}
		update := bson.M{"$set": route}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save route: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := route.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.RouteCreatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "route/"+e.RouteID, e)
				case *domain.RouteOptimizedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "route/"+e.RouteID, e)
				case *domain.RouteStartedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "route/"+e.RouteID, e)
				case *domain.StopCompletedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "route/"+e.RouteID, e)
				case *domain.RouteCompletedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "route/"+e.RouteID, e)
				case *domain.RouteCancelledEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "route/"+e.RouteID, e)
				case *domain.RouteRecalculatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "route/"+e.RouteID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					route.RouteID,
					"PickRoute",
					kafka.Topics.RoutingEvents,
					cloudEvent,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create outbox event: %w", err)
				}

				outboxEvents = append(outboxEvents, outboxEvent)
			}

			// Save all outbox events in the same transaction
			if len(outboxEvents) > 0 {
				if err := r.outboxRepo.SaveAll(sessCtx, outboxEvents); err != nil {
					return nil, fmt.Errorf("failed to save outbox events: %w", err)
				}
			}
		}

		// 3. Clear domain events from the aggregate
		route.ClearDomainEvents()

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

// FindByID retrieves a route by its ID
func (r *RouteRepository) FindByID(ctx context.Context, routeID string) (*domain.PickRoute, error) {
	filter := bson.M{"routeId": routeID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var route domain.PickRoute
	err := r.collection.FindOne(ctx, filter).Decode(&route)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &route, nil
}

// FindByOrderID retrieves routes for an order
func (r *RouteRepository) FindByOrderID(ctx context.Context, orderID string) ([]*domain.PickRoute, error) {
	filter := bson.M{"orderId": orderID}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var routes []*domain.PickRoute
	if err := cursor.All(ctx, &routes); err != nil {
		return nil, err
	}

	return routes, nil
}

// FindByWaveID retrieves routes for a wave
func (r *RouteRepository) FindByWaveID(ctx context.Context, waveID string) ([]*domain.PickRoute, error) {
	filter := bson.M{"waveId": waveID}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var routes []*domain.PickRoute
	if err := cursor.All(ctx, &routes); err != nil {
		return nil, err
	}

	return routes, nil
}

// FindByPickerID retrieves routes assigned to a picker
func (r *RouteRepository) FindByPickerID(ctx context.Context, pickerID string) ([]*domain.PickRoute, error) {
	filter := bson.M{"pickerId": pickerID}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var routes []*domain.PickRoute
	if err := cursor.All(ctx, &routes); err != nil {
		return nil, err
	}

	return routes, nil
}

// FindByStatus retrieves routes by status
func (r *RouteRepository) FindByStatus(ctx context.Context, status domain.RouteStatus) ([]*domain.PickRoute, error) {
	filter := bson.M{"status": status}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var routes []*domain.PickRoute
	if err := cursor.All(ctx, &routes); err != nil {
		return nil, err
	}

	return routes, nil
}

// FindByZone retrieves routes for a zone
func (r *RouteRepository) FindByZone(ctx context.Context, zone string) ([]*domain.PickRoute, error) {
	filter := bson.M{"zone": zone}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var routes []*domain.PickRoute
	if err := cursor.All(ctx, &routes); err != nil {
		return nil, err
	}

	return routes, nil
}

// FindActiveByPicker retrieves the active route for a picker
func (r *RouteRepository) FindActiveByPicker(ctx context.Context, pickerID string) (*domain.PickRoute, error) {
	filter := bson.M{
		"pickerId": pickerID,
		"status":   domain.RouteStatusInProgress,
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var route domain.PickRoute
	err := r.collection.FindOne(ctx, filter).Decode(&route)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &route, nil
}

// FindPendingRoutes retrieves pending routes ready for assignment
func (r *RouteRepository) FindPendingRoutes(ctx context.Context, zone string, limit int) ([]*domain.PickRoute, error) {
	filter := bson.M{
		"status": domain.RouteStatusPending,
	}
	if zone != "" {
		filter["zone"] = zone
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var routes []*domain.PickRoute
	if err := cursor.All(ctx, &routes); err != nil {
		return nil, err
	}

	return routes, nil
}

// Delete removes a route
func (r *RouteRepository) Delete(ctx context.Context, routeID string) error {
	filter := bson.M{"routeId": routeID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// CountByStatus counts routes by status
func (r *RouteRepository) CountByStatus(ctx context.Context, status domain.RouteStatus) (int64, error) {
	filter := bson.M{"status": status}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	return r.collection.CountDocuments(ctx, filter)
}

// GetOutboxRepository returns the outbox repository for this service
func (r *RouteRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
