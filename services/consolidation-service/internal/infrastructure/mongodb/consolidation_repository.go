package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/consolidation-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"github.com/wms-platform/shared/pkg/tenant"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConsolidationRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
	tenantHelper *tenant.RepositoryHelper
}

func NewConsolidationRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *ConsolidationRepository {
	collection := db.Collection("consolidations")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &ConsolidationRepository{
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

func (r *ConsolidationRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "consolidationId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "orderId", Value: 1}}},
		{Keys: bson.D{{Key: "waveId", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "station", Value: 1}}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

func (r *ConsolidationRepository) Save(ctx context.Context, unit *domain.ConsolidationUnit) error {
	unit.UpdatedAt = time.Now()

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
		filter := bson.M{"consolidationId": unit.ConsolidationID}
		update := bson.M{"$set": unit}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save consolidation unit: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := unit.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.ConsolidationStartedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "consolidation/"+e.ConsolidationID, e)
				case *domain.ItemConsolidatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "consolidation/"+e.ConsolidationID, e)
				case *domain.ConsolidationCompletedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "consolidation/"+e.ConsolidationID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					unit.ConsolidationID,
					"ConsolidationUnit",
					kafka.Topics.ConsolidationEvents,
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
		unit.ClearDomainEvents()

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (r *ConsolidationRepository) FindByID(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
	filter := bson.M{"consolidationId": consolidationID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var unit domain.ConsolidationUnit
	err := r.collection.FindOne(ctx, filter).Decode(&unit)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &unit, err
}

func (r *ConsolidationRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.ConsolidationUnit, error) {
	filter := bson.M{"orderId": orderID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var unit domain.ConsolidationUnit
	err := r.collection.FindOne(ctx, filter).Decode(&unit)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &unit, err
}

func (r *ConsolidationRepository) FindByWaveID(ctx context.Context, waveID string) ([]*domain.ConsolidationUnit, error) {
	filter := bson.M{"waveId": waveID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var units []*domain.ConsolidationUnit
	err = cursor.All(ctx, &units)
	return units, err
}

func (r *ConsolidationRepository) FindByStatus(ctx context.Context, status domain.ConsolidationStatus) ([]*domain.ConsolidationUnit, error) {
	filter := bson.M{"status": status}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var units []*domain.ConsolidationUnit
	err = cursor.All(ctx, &units)
	return units, err
}

func (r *ConsolidationRepository) FindByStation(ctx context.Context, station string) ([]*domain.ConsolidationUnit, error) {
	filter := bson.M{"station": station}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var units []*domain.ConsolidationUnit
	err = cursor.All(ctx, &units)
	return units, err
}

func (r *ConsolidationRepository) FindPending(ctx context.Context, limit int) ([]*domain.ConsolidationUnit, error) {
	filter := bson.M{"status": domain.ConsolidationStatusPending}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().SetLimit(int64(limit))
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var units []*domain.ConsolidationUnit
	err = cursor.All(ctx, &units)
	return units, err
}

func (r *ConsolidationRepository) Delete(ctx context.Context, consolidationID string) error {
	filter := bson.M{"consolidationId": consolidationID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// GetOutboxRepository returns the outbox repository for this service
func (r *ConsolidationRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
