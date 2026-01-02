package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/services/receiving-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InboundShipmentRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
}

func NewInboundShipmentRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *InboundShipmentRepository {
	collection := db.Collection("inbound_shipments")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &InboundShipmentRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
	}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *InboundShipmentRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "shipmentId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "asn.asnId", Value: 1}}},
		{Keys: bson.D{{Key: "supplier.supplierId", Value: 1}}},
		{Keys: bson.D{{Key: "purchaseOrderId", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "receivingDockId", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "asn.expectedArrival", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)

	// Create outbox indexes
	_ = r.outboxRepo.EnsureIndexes(ctx)
}

// Save persists an inbound shipment with its domain events in a single transaction
func (r *InboundShipmentRepository) Save(ctx context.Context, shipment *domain.InboundShipment) error {
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
			return nil, fmt.Errorf("failed to save inbound shipment: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := shipment.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.ShipmentExpectedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "shipment/"+e.ShipmentID, e)
				case *domain.ShipmentArrivedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "shipment/"+e.ShipmentID, e)
				case *domain.ItemReceivedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "shipment/"+e.ShipmentID, e)
				case *domain.ReceivingCompletedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "shipment/"+e.ShipmentID, e)
				case *domain.ReceivingDiscrepancyEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "shipment/"+e.ShipmentID, e)
				case *domain.PutawayTaskCreatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "shipment/"+e.ShipmentID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					shipment.ShipmentID,
					"InboundShipment",
					kafka.Topics.ReceivingEvents,
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

func (r *InboundShipmentRepository) FindByID(ctx context.Context, shipmentID string) (*domain.InboundShipment, error) {
	var shipment domain.InboundShipment
	err := r.collection.FindOne(ctx, bson.M{"shipmentId": shipmentID}).Decode(&shipment)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &shipment, err
}

func (r *InboundShipmentRepository) FindByASNID(ctx context.Context, asnID string) (*domain.InboundShipment, error) {
	var shipment domain.InboundShipment
	err := r.collection.FindOne(ctx, bson.M{"asn.asnId": asnID}).Decode(&shipment)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &shipment, err
}

func (r *InboundShipmentRepository) FindBySupplierID(ctx context.Context, supplierID string) ([]*domain.InboundShipment, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"supplier.supplierId": supplierID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var shipments []*domain.InboundShipment
	err = cursor.All(ctx, &shipments)
	return shipments, err
}

func (r *InboundShipmentRepository) FindByPurchaseOrderID(ctx context.Context, poID string) ([]*domain.InboundShipment, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"purchaseOrderId": poID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var shipments []*domain.InboundShipment
	err = cursor.All(ctx, &shipments)
	return shipments, err
}

func (r *InboundShipmentRepository) FindByStatus(ctx context.Context, status domain.ShipmentStatus) ([]*domain.InboundShipment, error) {
	opts := options.Find().SetSort(bson.D{{Key: "asn.expectedArrival", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{"status": status}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var shipments []*domain.InboundShipment
	err = cursor.All(ctx, &shipments)
	return shipments, err
}

func (r *InboundShipmentRepository) FindByDockAndStatus(ctx context.Context, dockID string, status domain.ShipmentStatus) ([]*domain.InboundShipment, error) {
	filter := bson.M{"receivingDockId": dockID, "status": status}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var shipments []*domain.InboundShipment
	err = cursor.All(ctx, &shipments)
	return shipments, err
}

func (r *InboundShipmentRepository) FindExpectedArrivals(ctx context.Context, from, to time.Time) ([]*domain.InboundShipment, error) {
	filter := bson.M{
		"status": domain.ShipmentStatusExpected,
		"asn.expectedArrival": bson.M{
			"$gte": from,
			"$lte": to,
		},
	}
	opts := options.Find().SetSort(bson.D{{Key: "asn.expectedArrival", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var shipments []*domain.InboundShipment
	err = cursor.All(ctx, &shipments)
	return shipments, err
}

func (r *InboundShipmentRepository) FindAll(ctx context.Context, limit int) ([]*domain.InboundShipment, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit))
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var shipments []*domain.InboundShipment
	err = cursor.All(ctx, &shipments)
	return shipments, err
}

func (r *InboundShipmentRepository) Delete(ctx context.Context, shipmentID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"shipmentId": shipmentID})
	return err
}

// GetOutboxRepository returns the outbox repository for this service
func (r *InboundShipmentRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
