package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wms-platform/services/order-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
)

// OrderRepository implements domain.OrderRepository using MongoDB
type OrderRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
}

// NewOrderRepository creates a new OrderRepository
func NewOrderRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *OrderRepository {
	collection := db.Collection("orders")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "orderId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		},
		{
			Keys: bson.D{{Key: "customerId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "waveId", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "priority", Value: 1},
			},
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	// Create outbox indexes
	_ = outboxRepo.EnsureIndexes(ctx)

	return &OrderRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
	}
}

// Save persists an order with its domain events in a single transaction
func (r *OrderRepository) Save(ctx context.Context, order *domain.Order) error {
	order.UpdatedAt = time.Now().UTC()

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
		filter := bson.M{"orderId": order.OrderID}
		update := bson.M{"$set": order}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save order: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := order.DomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.OrderReceivedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "order/"+e.OrderID, e)
				case *domain.OrderValidatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "order/"+e.OrderID, e)
				case *domain.OrderCancelledEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "order/"+e.OrderID, e)
				case *domain.OrderAssignedToWaveEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "order/"+e.OrderID, e)
				case *domain.OrderShippedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "order/"+e.OrderID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					order.OrderID,
					"Order",
					kafka.Topics.OrdersEvents,
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
		order.ClearDomainEvents()

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

// FindByID retrieves an order by its OrderID
func (r *OrderRepository) FindByID(ctx context.Context, orderID string) (*domain.Order, error) {
	var order domain.Order
	filter := bson.M{"orderId": orderID}

	err := r.collection.FindOne(ctx, filter).Decode(&order)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &order, nil
}

// FindByCustomerID retrieves all orders for a customer
func (r *OrderRepository) FindByCustomerID(ctx context.Context, customerID string, pagination domain.Pagination) ([]*domain.Order, error) {
	filter := bson.M{"customerId": customerID}
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit())

	return r.findMany(ctx, filter, opts)
}

// FindByStatus retrieves orders by status
func (r *OrderRepository) FindByStatus(ctx context.Context, status domain.Status, pagination domain.Pagination) ([]*domain.Order, error) {
	filter := bson.M{"status": status}
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit())

	return r.findMany(ctx, filter, opts)
}

// FindByWaveID retrieves all orders in a wave
func (r *OrderRepository) FindByWaveID(ctx context.Context, waveID string) ([]*domain.Order, error) {
	filter := bson.M{"waveId": waveID}
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: 1}})

	return r.findMany(ctx, filter, opts)
}

// FindValidatedOrders retrieves orders ready for wave assignment
func (r *OrderRepository) FindValidatedOrders(ctx context.Context, priority domain.Priority, limit int) ([]*domain.Order, error) {
	filter := bson.M{
		"status": domain.StatusValidated,
		"waveId": bson.M{"$exists": false},
	}

	if priority != "" {
		filter["priority"] = priority
	}

	opts := options.Find().
		SetSort(bson.D{
			{Key: "priority", Value: 1},      // Same-day first
			{Key: "promisedDeliveryAt", Value: 1},
			{Key: "createdAt", Value: 1},
		}).
		SetLimit(int64(limit))

	return r.findMany(ctx, filter, opts)
}

// UpdateStatus updates the order status
func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID string, status domain.Status) error {
	filter := bson.M{"orderId": orderID}
	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("order not found")
	}

	return nil
}

// AssignToWave assigns an order to a wave
func (r *OrderRepository) AssignToWave(ctx context.Context, orderID string, waveID string) error {
	filter := bson.M{
		"orderId": orderID,
		"status":  domain.StatusValidated,
	}
	update := bson.M{
		"$set": bson.M{
			"waveId":    waveID,
			"status":    domain.StatusWaveAssigned,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("order not found or not in validated status")
	}

	return nil
}

// Delete deletes an order (soft delete in practice)
func (r *OrderRepository) Delete(ctx context.Context, orderID string) error {
	filter := bson.M{"orderId": orderID}
	update := bson.M{
		"$set": bson.M{
			"status":    domain.StatusCancelled,
			"updatedAt": time.Now().UTC(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// Count returns the total number of orders matching the filter
func (r *OrderRepository) Count(ctx context.Context, filter domain.OrderFilter) (int64, error) {
	mongoFilter := r.buildFilter(filter)
	return r.collection.CountDocuments(ctx, mongoFilter)
}

// findMany is a helper for finding multiple orders
func (r *OrderRepository) findMany(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*domain.Order, error) {
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []*domain.Order
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

// buildFilter builds a MongoDB filter from an OrderFilter
func (r *OrderRepository) buildFilter(filter domain.OrderFilter) bson.M {
	mongoFilter := bson.M{}

	if filter.CustomerID != nil {
		mongoFilter["customerId"] = *filter.CustomerID
	}

	if filter.Status != nil {
		mongoFilter["status"] = *filter.Status
	}

	if filter.Priority != nil {
		mongoFilter["priority"] = *filter.Priority
	}

	if filter.WaveID != nil {
		mongoFilter["waveId"] = *filter.WaveID
	}

	return mongoFilter
}

// GetOutboxRepository returns the outbox repository for this service
func (r *OrderRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
