package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/labor-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WorkerRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
}

func NewWorkerRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *WorkerRepository {
	collection := db.Collection("workers")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &WorkerRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
	}
	repo.ensureIndexes(context.Background())

	// Create outbox indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = outboxRepo.EnsureIndexes(ctx)

	return repo
}

func (r *WorkerRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "workerId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "employeeId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "currentZone", Value: 1}}},
		{Keys: bson.D{{Key: "skills.type", Value: 1}}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

func (r *WorkerRepository) Save(ctx context.Context, worker *domain.Worker) error {
	worker.UpdatedAt = time.Now()

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
		filter := bson.M{"workerId": worker.WorkerID}
		update := bson.M{"$set": worker}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save worker: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := worker.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.ShiftStartedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "worker/"+e.WorkerID, e)
				case *domain.ShiftEndedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "worker/"+e.WorkerID, e)
				case *domain.TaskAssignedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "worker/"+e.WorkerID, e)
				case *domain.TaskCompletedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "worker/"+e.WorkerID, e)
				case *domain.PerformanceRecordedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "worker/"+e.WorkerID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					worker.WorkerID,
					"Worker",
					kafka.Topics.LaborEvents,
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
		worker.ClearDomainEvents()

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (r *WorkerRepository) FindByID(ctx context.Context, workerID string) (*domain.Worker, error) {
	var worker domain.Worker
	err := r.collection.FindOne(ctx, bson.M{"workerId": workerID}).Decode(&worker)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &worker, err
}

func (r *WorkerRepository) FindByEmployeeID(ctx context.Context, employeeID string) (*domain.Worker, error) {
	var worker domain.Worker
	err := r.collection.FindOne(ctx, bson.M{"employeeId": employeeID}).Decode(&worker)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &worker, err
}

func (r *WorkerRepository) FindByStatus(ctx context.Context, status domain.WorkerStatus) ([]*domain.Worker, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var workers []*domain.Worker
	err = cursor.All(ctx, &workers)
	return workers, err
}

func (r *WorkerRepository) FindByZone(ctx context.Context, zone string) ([]*domain.Worker, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"currentZone": zone})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var workers []*domain.Worker
	err = cursor.All(ctx, &workers)
	return workers, err
}

func (r *WorkerRepository) FindAvailableBySkill(ctx context.Context, taskType domain.TaskType, zone string) ([]*domain.Worker, error) {
	filter := bson.M{
		"status":      domain.WorkerStatusAvailable,
		"skills.type": taskType,
	}
	if zone != "" {
		filter["currentZone"] = zone
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var workers []*domain.Worker
	err = cursor.All(ctx, &workers)
	return workers, err
}

func (r *WorkerRepository) FindAll(ctx context.Context, limit, offset int) ([]*domain.Worker, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var workers []*domain.Worker
	err = cursor.All(ctx, &workers)
	return workers, err
}

func (r *WorkerRepository) Delete(ctx context.Context, workerID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"workerId": workerID})
	return err
}

// GetOutboxRepository returns the outbox repository for this service
func (r *WorkerRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
