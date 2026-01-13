package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"github.com/wms-platform/shared/pkg/tenant"
	"github.com/wms-platform/waving-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// WaveRepository implements domain.WaveRepository using MongoDB
type WaveRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
	tenantHelper *tenant.RepositoryHelper
}

// NewWaveRepository creates a new WaveRepository
func NewWaveRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *WaveRepository {
	collection := db.Collection("waves")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &WaveRepository{
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
func (r *WaveRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "waveId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}, {Key: "scheduledStart", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "waveType", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "zone", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "orders.orderId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: -1}},
		},
	}

	r.collection.Indexes().CreateMany(ctx, indexes)
}

// Save persists a wave with its domain events in a single transaction
func (r *WaveRepository) Save(ctx context.Context, wave *domain.Wave) error {
	wave.UpdatedAt = time.Now()

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
		filter := bson.M{"waveId": wave.WaveID}
		update := bson.M{"$set": wave}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save wave: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := wave.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.WaveCreatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "wave/"+e.WaveID, e)
				case *domain.WaveScheduledEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "wave/"+e.WaveID, e)
				case *domain.WaveReleasedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "wave/"+e.WaveID, e)
				case *domain.WaveCompletedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "wave/"+e.WaveID, e)
				case *domain.WaveCancelledEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "wave/"+e.WaveID, e)
				case *domain.OrderAddedToWaveEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "wave/"+e.WaveID, e)
				case *domain.OrderRemovedFromWaveEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "wave/"+e.WaveID, e)
				case *domain.WaveOptimizedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "wave/"+e.WaveID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					wave.WaveID,
					"Wave",
					kafka.Topics.WavesEvents,
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
		wave.ClearDomainEvents()

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

// FindByID retrieves a wave by its ID
func (r *WaveRepository) FindByID(ctx context.Context, waveID string) (*domain.Wave, error) {
	filter := bson.M{"waveId": waveID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var wave domain.Wave
	err := r.collection.FindOne(ctx, filter).Decode(&wave)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &wave, nil
}

// FindByStatus retrieves waves by status
func (r *WaveRepository) FindByStatus(ctx context.Context, status domain.WaveStatus) ([]*domain.Wave, error) {
	filter := bson.M{"status": status}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)
	opts := options.Find().SetSort(bson.D{{Key: "scheduledStart", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var waves []*domain.Wave
	if err := cursor.All(ctx, &waves); err != nil {
		return nil, err
	}

	return waves, nil
}

// FindByType retrieves waves by type
func (r *WaveRepository) FindByType(ctx context.Context, waveType domain.WaveType) ([]*domain.Wave, error) {
	filter := bson.M{"waveType": waveType}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var waves []*domain.Wave
	if err := cursor.All(ctx, &waves); err != nil {
		return nil, err
	}

	return waves, nil
}

// FindByZone retrieves waves by warehouse zone
func (r *WaveRepository) FindByZone(ctx context.Context, zone string) ([]*domain.Wave, error) {
	filter := bson.M{"zone": zone}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)
	opts := options.Find().SetSort(bson.D{{Key: "scheduledStart", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var waves []*domain.Wave
	if err := cursor.All(ctx, &waves); err != nil {
		return nil, err
	}

	return waves, nil
}

// FindScheduledBefore retrieves waves scheduled before a given time
func (r *WaveRepository) FindScheduledBefore(ctx context.Context, before time.Time) ([]*domain.Wave, error) {
	filter := bson.M{
		"status":         domain.WaveStatusScheduled,
		"scheduledStart": bson.M{"$lte": before},
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)
	opts := options.Find().SetSort(bson.D{{Key: "scheduledStart", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var waves []*domain.Wave
	if err := cursor.All(ctx, &waves); err != nil {
		return nil, err
	}

	return waves, nil
}

// FindReadyForRelease retrieves waves that are ready to be released
func (r *WaveRepository) FindReadyForRelease(ctx context.Context) ([]*domain.Wave, error) {
	now := time.Now()
	filter := bson.M{
		"status": domain.WaveStatusScheduled,
		"scheduledStart": bson.M{"$lte": now},
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: 1}, {Key: "scheduledStart", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var waves []*domain.Wave
	if err := cursor.All(ctx, &waves); err != nil {
		return nil, err
	}

	return waves, nil
}

// FindByOrderID retrieves the wave containing a specific order
func (r *WaveRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.Wave, error) {
	filter := bson.M{"orders.orderId": orderID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var wave domain.Wave
	err := r.collection.FindOne(ctx, filter).Decode(&wave)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &wave, nil
}

// FindActive retrieves all active waves
func (r *WaveRepository) FindActive(ctx context.Context) ([]*domain.Wave, error) {
	filter := bson.M{
		"status": bson.M{
			"$nin": []domain.WaveStatus{domain.WaveStatusCompleted, domain.WaveStatusCancelled},
		},
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: 1}, {Key: "scheduledStart", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var waves []*domain.Wave
	if err := cursor.All(ctx, &waves); err != nil {
		return nil, err
	}

	return waves, nil
}

// FindByDateRange retrieves waves created within a date range
func (r *WaveRepository) FindByDateRange(ctx context.Context, start, end time.Time) ([]*domain.Wave, error) {
	filter := bson.M{
		"createdAt": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var waves []*domain.Wave
	if err := cursor.All(ctx, &waves); err != nil {
		return nil, err
	}

	return waves, nil
}

// Delete removes a wave
func (r *WaveRepository) Delete(ctx context.Context, waveID string) error {
	filter := bson.M{"waveId": waveID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// Count returns the total number of waves matching a status
func (r *WaveRepository) Count(ctx context.Context, status domain.WaveStatus) (int64, error) {
	filter := bson.M{"status": status}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)
	return r.collection.CountDocuments(ctx, filter)
}

// GetOutboxRepository returns the outbox repository for this service
func (r *WaveRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
