package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/services/sortation-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SortationBatchRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
}

func NewSortationBatchRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *SortationBatchRepository {
	collection := db.Collection("sortation_batches")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &SortationBatchRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
	}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *SortationBatchRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "batchId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "carrierId", Value: 1}}},
		{Keys: bson.D{{Key: "destinationGroup", Value: 1}}},
		{Keys: bson.D{{Key: "sortationCenter", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "carrierId", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)

	// Create outbox indexes
	_ = r.outboxRepo.EnsureIndexes(ctx)
}

// Save persists a sortation batch with its domain events in a single transaction
func (r *SortationBatchRepository) Save(ctx context.Context, batch *domain.SortationBatch) error {
	batch.UpdatedAt = time.Now()

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
		filter := bson.M{"batchId": batch.BatchID}
		update := bson.M{"$set": batch}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save sortation batch: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := batch.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.SortationBatchCreatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "batch/"+e.BatchID, e)
				case *domain.PackageReceivedForSortationEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "batch/"+e.BatchID, e)
				case *domain.PackageSortedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "batch/"+e.BatchID, e)
				case *domain.BatchDispatchedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "batch/"+e.BatchID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					batch.BatchID,
					"SortationBatch",
					kafka.Topics.SortationEvents,
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
		batch.ClearDomainEvents()

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (r *SortationBatchRepository) FindByID(ctx context.Context, batchID string) (*domain.SortationBatch, error) {
	var batch domain.SortationBatch
	err := r.collection.FindOne(ctx, bson.M{"batchId": batchID}).Decode(&batch)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &batch, err
}

func (r *SortationBatchRepository) FindByStatus(ctx context.Context, status domain.SortationStatus) ([]*domain.SortationBatch, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"status": status}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var batches []*domain.SortationBatch
	err = cursor.All(ctx, &batches)
	return batches, err
}

func (r *SortationBatchRepository) FindByCarrier(ctx context.Context, carrierID string) ([]*domain.SortationBatch, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"carrierId": carrierID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var batches []*domain.SortationBatch
	err = cursor.All(ctx, &batches)
	return batches, err
}

func (r *SortationBatchRepository) FindByDestination(ctx context.Context, destinationGroup string) ([]*domain.SortationBatch, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"destinationGroup": destinationGroup}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var batches []*domain.SortationBatch
	err = cursor.All(ctx, &batches)
	return batches, err
}

func (r *SortationBatchRepository) FindByCenter(ctx context.Context, centerID string) ([]*domain.SortationBatch, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"sortationCenter": centerID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var batches []*domain.SortationBatch
	err = cursor.All(ctx, &batches)
	return batches, err
}

func (r *SortationBatchRepository) FindReadyForDispatch(ctx context.Context, carrierID string, limit int) ([]*domain.SortationBatch, error) {
	filter := bson.M{"status": domain.SortationStatusReady}
	if carrierID != "" {
		filter["carrierId"] = carrierID
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "scheduledDispatch", Value: 1}, {Key: "createdAt", Value: 1}}).
		SetLimit(int64(limit))
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var batches []*domain.SortationBatch
	err = cursor.All(ctx, &batches)
	return batches, err
}

func (r *SortationBatchRepository) FindAll(ctx context.Context, limit int) ([]*domain.SortationBatch, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit))
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var batches []*domain.SortationBatch
	err = cursor.All(ctx, &batches)
	return batches, err
}

func (r *SortationBatchRepository) Delete(ctx context.Context, batchID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"batchId": batchID})
	return err
}

// GetOutboxRepository returns the outbox repository for this service
func (r *SortationBatchRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
