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

type LedgerEntryRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
	tenantHelper *tenant.RepositoryHelper
}

func NewLedgerEntryRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *LedgerEntryRepository {
	collection := db.Collection("ledger_entries")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &LedgerEntryRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
	repo.ensureIndexes(context.Background())

	return repo
}

func (r *LedgerEntryRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		// Primary lookup by SKU + time
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "facilityId", Value: 1},
				{Key: "entry.sku", Value: 1},
				{Key: "entry.createdAt", Value: -1},
			},
		},
		// Lookup by transaction ID (to find debit/credit pairs)
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "entry.transactionId", Value: 1},
			},
		},
		// Lookup by account type
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "facilityId", Value: 1},
				{Key: "entry.accountType", Value: 1},
				{Key: "entry.createdAt", Value: -1},
			},
		},
		// Lookup by location
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "facilityId", Value: 1},
				{Key: "entry.locationId", Value: 1},
				{Key: "entry.createdAt", Value: -1},
			},
		},
		// Lookup by reference
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "facilityId", Value: 1},
				{Key: "entry.referenceId", Value: 1},
			},
		},
		// TTL index for archival (2 years = 63072000 seconds)
		{
			Keys: bson.D{{Key: "entry.createdAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(63072000),
		},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

func (r *LedgerEntryRepository) Save(ctx context.Context, entry *domain.LedgerEntryAggregate) error {
	// Start a MongoDB session for transaction
	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// 1. Insert the entry (append-only)
		if _, err := r.collection.InsertOne(sessCtx, entry); err != nil {
			return nil, fmt.Errorf("failed to insert ledger entry: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := entry.PullEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				cloudEvent := r.eventFactory.CreateEvent(sessCtx, event.EventType(), "ledger-entry/"+entry.Entry.SKU, event)

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					entry.Entry.SKU,
					"LedgerEntry",
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
		entry.ClearDomainEvents()

		return nil, nil
	})

	return err
}

func (r *LedgerEntryRepository) SaveAll(ctx context.Context, entries []*domain.LedgerEntryAggregate) error {
	if len(entries) == 0 {
		return nil
	}

	// Start a MongoDB session for transaction
	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// 1. Insert all entries
		documents := make([]interface{}, len(entries))
		for i, entry := range entries {
			documents[i] = entry
		}

		if _, err := r.collection.InsertMany(sessCtx, documents); err != nil {
			return nil, fmt.Errorf("failed to insert ledger entries: %w", err)
		}

		// 2. Save domain events from all entries to outbox
		allOutboxEvents := make([]*outbox.OutboxEvent, 0)

		for _, entry := range entries {
			domainEvents := entry.PullEvents()
			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				cloudEvent := r.eventFactory.CreateEvent(sessCtx, event.EventType(), "ledger-entry/"+entry.Entry.SKU, event)

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					entry.Entry.SKU,
					"LedgerEntry",
					"wms.inventory.events",
					cloudEvent,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create outbox event: %w", err)
				}

				allOutboxEvents = append(allOutboxEvents, outboxEvent)
			}
		}

		if len(allOutboxEvents) > 0 {
			if err := r.outboxRepo.SaveAll(sessCtx, allOutboxEvents); err != nil {
				return nil, fmt.Errorf("failed to save outbox events: %w", err)
			}
		}

		// 3. Clear events from all aggregates
		for _, entry := range entries {
			entry.ClearDomainEvents()
		}

		return nil, nil
	})

	return err
}

func (r *LedgerEntryRepository) FindBySKU(ctx context.Context, tenantID, facilityID, sku string, limit int) ([]*domain.LedgerEntryAggregate, error) {
	filter := bson.M{
		"tenantId":   tenantID,
		"facilityId": facilityID,
		"entry.sku":  sku,
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "entry.createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find entries: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []*domain.LedgerEntryAggregate
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode entries: %w", err)
	}

	return entries, nil
}

func (r *LedgerEntryRepository) FindByTransactionID(ctx context.Context, tenantID, transactionID string) ([]*domain.LedgerEntryAggregate, error) {
	filter := bson.M{
		"tenantId":            tenantID,
		"entry.transactionId": transactionID,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find entries: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []*domain.LedgerEntryAggregate
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode entries: %w", err)
	}

	return entries, nil
}

func (r *LedgerEntryRepository) FindByTimeRange(ctx context.Context, tenantID, facilityID, sku string, start, end time.Time) ([]*domain.LedgerEntryAggregate, error) {
	filter := bson.M{
		"tenantId":   tenantID,
		"facilityId": facilityID,
		"entry.sku":  sku,
		"entry.createdAt": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "entry.createdAt", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find entries: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []*domain.LedgerEntryAggregate
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode entries: %w", err)
	}

	return entries, nil
}

func (r *LedgerEntryRepository) FindByAccountType(ctx context.Context, tenantID, facilityID, sku string, accountType domain.AccountType, limit int) ([]*domain.LedgerEntryAggregate, error) {
	filter := bson.M{
		"tenantId":          tenantID,
		"facilityId":        facilityID,
		"entry.sku":         sku,
		"entry.accountType": accountType.String(),
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "entry.createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find entries: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []*domain.LedgerEntryAggregate
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode entries: %w", err)
	}

	return entries, nil
}

func (r *LedgerEntryRepository) GetBalanceAtTime(ctx context.Context, tenantID, facilityID, sku string, timestamp time.Time) (int, domain.Money, error) {
	filter := bson.M{
		"tenantId":   tenantID,
		"facilityId": facilityID,
		"entry.sku":  sku,
		"entry.createdAt": bson.M{
			"$lte": timestamp,
		},
		"entry.accountType": domain.AccountInventory.String(),
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "entry.createdAt", Value: -1}}).
		SetLimit(1)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return 0, domain.Money{}, fmt.Errorf("failed to find entries: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []*domain.LedgerEntryAggregate
	if err := cursor.All(ctx, &entries); err != nil {
		return 0, domain.Money{}, fmt.Errorf("failed to decode entries: %w", err)
	}

	if len(entries) == 0 {
		return 0, domain.ZeroMoney("USD"), nil
	}

	entry := entries[0]
	return entry.Entry.RunningBalance, entry.Entry.RunningValue, nil
}
