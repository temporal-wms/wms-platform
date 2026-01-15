package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/inventory-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/tenant"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InventoryLedgerRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
	tenantHelper *tenant.RepositoryHelper
}

func NewInventoryLedgerRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *InventoryLedgerRepository {
	collection := db.Collection("inventory_ledgers")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &InventoryLedgerRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
	repo.ensureIndexes(context.Background())

	return repo
}

func (r *InventoryLedgerRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		// Primary lookup by SKU + tenant (unique)
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "facilityId", Value: 1},
				{Key: "sku", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

func (r *InventoryLedgerRepository) Save(ctx context.Context, ledger *domain.InventoryLedger) error {
	ledger.UpdatedAt = time.Now().UTC()

	// Start a MongoDB session for transaction
	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// 1. Save the ledger aggregate
		opts := options.Update().SetUpsert(true)
		filter := bson.M{
			"tenantId":   ledger.TenantID,
			"facilityId": ledger.FacilityID,
			"sku":        ledger.SKU,
		}
		update := bson.M{"$set": ledger}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save inventory ledger: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := ledger.PullEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				cloudEvent := r.eventFactory.CreateEvent(sessCtx, event.EventType(), "ledger/"+ledger.SKU, event)

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					ledger.SKU,
					"InventoryLedger",
					"wms.inventory.events",
					cloudEvent,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create outbox event: %w", err)
				}

				outboxEvents = append(outboxEvents, outboxEvent)
			}

			if len(outboxEvents) > 0 {
				if err := r.outboxRepo.SaveAll(sessCtx, outboxEvents); err != nil {
					return nil, fmt.Errorf("failed to save outbox events: %w", err)
				}
			}
		}

		// 3. Clear events from aggregate
		ledger.ClearDomainEvents()

		return nil, nil
	})

	return err
}

func (r *InventoryLedgerRepository) FindBySKU(ctx context.Context, tenantID, facilityID, sku string) (*domain.InventoryLedger, error) {
	filter := bson.M{
		"tenantId":   tenantID,
		"facilityId": facilityID,
		"sku":        sku,
	}

	var ledger domain.InventoryLedger
	err := r.collection.FindOne(ctx, filter).Decode(&ledger)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrLedgerNotFound
		}
		return nil, fmt.Errorf("failed to find ledger: %w", err)
	}

	return &ledger, nil
}

func (r *InventoryLedgerRepository) FindByLocation(ctx context.Context, tenantID, facilityID, locationID string) ([]*domain.InventoryLedger, error) {
	// Note: This would require tracking locationID in the ledger aggregate
	// For now, we'll return empty list
	// TODO: Implement location-based querying if needed
	return []*domain.InventoryLedger{}, nil
}

func (r *InventoryLedgerRepository) FindAll(ctx context.Context, tenantID, facilityID string, limit, offset int) ([]*domain.InventoryLedger, error) {
	filter := bson.M{
		"tenantId":   tenantID,
		"facilityId": facilityID,
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "updatedAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find ledgers: %w", err)
	}
	defer cursor.Close(ctx)

	var ledgers []*domain.InventoryLedger
	if err := cursor.All(ctx, &ledgers); err != nil {
		return nil, fmt.Errorf("failed to decode ledgers: %w", err)
	}

	return ledgers, nil
}

func (r *InventoryLedgerRepository) Delete(ctx context.Context, tenantID, facilityID, sku string) error {
	filter := bson.M{
		"tenantId":   tenantID,
		"facilityId": facilityID,
		"sku":        sku,
	}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete ledger: %w", err)
	}

	if result.DeletedCount == 0 {
		return domain.ErrLedgerNotFound
	}

	return nil
}
