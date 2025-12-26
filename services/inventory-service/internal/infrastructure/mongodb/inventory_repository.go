package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/inventory-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InventoryRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
}

func NewInventoryRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *InventoryRepository {
	collection := db.Collection("inventory")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &InventoryRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
	}
	repo.ensureIndexes(context.Background())

	// Create outbox indexes
	_ = outboxRepo.EnsureIndexes(context.Background())

	return repo
}

func (r *InventoryRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "sku", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "locations.locationId", Value: 1}}},
		{Keys: bson.D{{Key: "locations.zone", Value: 1}}},
		{Keys: bson.D{{Key: "availableQuantity", Value: 1}}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

func (r *InventoryRepository) Save(ctx context.Context, item *domain.InventoryItem) error {
	item.UpdatedAt = time.Now()

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
		filter := bson.M{"sku": item.SKU}
		update := bson.M{"$set": item}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save inventory item: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := item.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.InventoryReceivedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "inventory/"+e.SKU, e)
				case *domain.InventoryAdjustedEvent:
					cloudEvent = r.eventFactory.CreateInventoryAdjustedEvent(sessCtx, e.SKU, e.LocationID, e.OldQuantity, e.NewQuantity, "adjustment", e.Reason)
				case *domain.LowStockAlertEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "inventory/"+e.SKU, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					item.SKU,
					"InventoryItem",
					kafka.Topics.InventoryEvents,
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
		item.ClearDomainEvents()

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (r *InventoryRepository) FindBySKU(ctx context.Context, sku string) (*domain.InventoryItem, error) {
	var item domain.InventoryItem
	err := r.collection.FindOne(ctx, bson.M{"sku": sku}).Decode(&item)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &item, err
}

func (r *InventoryRepository) FindByLocation(ctx context.Context, locationID string) ([]*domain.InventoryItem, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"locations.locationId": locationID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var items []*domain.InventoryItem
	err = cursor.All(ctx, &items)
	return items, err
}

func (r *InventoryRepository) FindByZone(ctx context.Context, zone string) ([]*domain.InventoryItem, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"locations.zone": zone})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var items []*domain.InventoryItem
	err = cursor.All(ctx, &items)
	return items, err
}

func (r *InventoryRepository) FindLowStock(ctx context.Context) ([]*domain.InventoryItem, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"$expr": bson.M{"$lte": []string{"$availableQuantity", "$reorderPoint"}},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var items []*domain.InventoryItem
	err = cursor.All(ctx, &items)
	return items, err
}

func (r *InventoryRepository) FindAll(ctx context.Context, limit, offset int) ([]*domain.InventoryItem, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var items []*domain.InventoryItem
	err = cursor.All(ctx, &items)
	return items, err
}

func (r *InventoryRepository) Delete(ctx context.Context, sku string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"sku": sku})
	return err
}

// GetOutboxRepository returns the outbox repository for this service
func (r *InventoryRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
