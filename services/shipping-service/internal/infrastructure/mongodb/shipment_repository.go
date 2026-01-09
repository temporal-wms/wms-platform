package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/shipping-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ShipmentRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
}

func NewShipmentRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *ShipmentRepository {
	collection := db.Collection("shipments")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &ShipmentRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
	}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *ShipmentRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "shipmentId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "orderId", Value: 1}}},
		{Keys: bson.D{{Key: "label.trackingNumber", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "carrier.code", Value: 1}}},
		{Keys: bson.D{{Key: "manifest.manifestId", Value: 1}}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)

	// Create outbox indexes
	_ = r.outboxRepo.EnsureIndexes(ctx)
}

func (r *ShipmentRepository) Save(ctx context.Context, shipment *domain.Shipment) error {
	shipment.UpdatedAt = time.Now()

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
		filter := bson.M{"shipmentId": shipment.ShipmentID}
		update := bson.M{"$set": shipment}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save shipment: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := shipment.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.ShipmentCreatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "shipment/"+e.ShipmentID, e)
				case *domain.LabelGeneratedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "shipment/"+e.ShipmentID, e)
				case *domain.ShipmentManifestedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "shipment/"+e.ShipmentID, e)
				case *domain.ShipConfirmedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "shipment/"+e.ShipmentID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					shipment.ShipmentID,
					"Shipment",
					kafka.Topics.ShippingEvents,
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
		shipment.ClearDomainEvents()

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (r *ShipmentRepository) FindByID(ctx context.Context, shipmentID string) (*domain.Shipment, error) {
	var s domain.Shipment
	err := r.collection.FindOne(ctx, bson.M{"shipmentId": shipmentID}).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &s, err
}

func (r *ShipmentRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.Shipment, error) {
	var s domain.Shipment
	err := r.collection.FindOne(ctx, bson.M{"orderId": orderID}).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &s, err
}

func (r *ShipmentRepository) FindByTrackingNumber(ctx context.Context, trackingNumber string) (*domain.Shipment, error) {
	var s domain.Shipment
	err := r.collection.FindOne(ctx, bson.M{"label.trackingNumber": trackingNumber}).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &s, err
}

func (r *ShipmentRepository) FindByStatus(ctx context.Context, status domain.ShipmentStatus) ([]*domain.Shipment, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var shipments []*domain.Shipment
	err = cursor.All(ctx, &shipments)
	return shipments, err
}

func (r *ShipmentRepository) FindByCarrier(ctx context.Context, carrierCode string) ([]*domain.Shipment, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"carrier.code": carrierCode})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var shipments []*domain.Shipment
	err = cursor.All(ctx, &shipments)
	return shipments, err
}

func (r *ShipmentRepository) FindByManifestID(ctx context.Context, manifestID string) ([]*domain.Shipment, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"manifest.manifestId": manifestID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var shipments []*domain.Shipment
	err = cursor.All(ctx, &shipments)
	return shipments, err
}

func (r *ShipmentRepository) FindPendingForManifest(ctx context.Context, carrierCode string) ([]*domain.Shipment, error) {
	filter := bson.M{
		"carrier.code": carrierCode,
		"status":       domain.ShipmentStatusLabeled,
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var shipments []*domain.Shipment
	err = cursor.All(ctx, &shipments)
	return shipments, err
}

func (r *ShipmentRepository) Delete(ctx context.Context, shipmentID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"shipmentId": shipmentID})
	return err
}

// GetOutboxRepository returns the outbox repository for this service
func (r *ShipmentRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
