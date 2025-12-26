package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/packing-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PackTaskRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
}

func NewPackTaskRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *PackTaskRepository {
	collection := db.Collection("pack_tasks")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &PackTaskRepository{
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

func (r *PackTaskRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "taskId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "orderId", Value: 1}}},
		{Keys: bson.D{{Key: "waveId", Value: 1}}},
		{Keys: bson.D{{Key: "packerId", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "station", Value: 1}}},
		{Keys: bson.D{{Key: "shippingLabel.trackingNumber", Value: 1}}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

func (r *PackTaskRepository) Save(ctx context.Context, task *domain.PackTask) error {
	task.UpdatedAt = time.Now()

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
		filter := bson.M{"taskId": task.TaskID}
		update := bson.M{"$set": task}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save pack task: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := task.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.PackTaskCreatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "pack-task/"+e.TaskID, e)
				case *domain.PackagingSuggestedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "pack-task/"+e.TaskID, e)
				case *domain.PackageSealedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "pack-task/"+e.TaskID, e)
				case *domain.LabelAppliedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "pack-task/"+e.TaskID, e)
				case *domain.PackTaskCompletedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "pack-task/"+e.TaskID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					task.TaskID,
					"PackTask",
					kafka.Topics.PackingEvents,
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
		task.ClearDomainEvents()

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (r *PackTaskRepository) FindByID(ctx context.Context, taskID string) (*domain.PackTask, error) {
	var task domain.PackTask
	err := r.collection.FindOne(ctx, bson.M{"taskId": taskID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &task, err
}

func (r *PackTaskRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.PackTask, error) {
	var task domain.PackTask
	err := r.collection.FindOne(ctx, bson.M{"orderId": orderID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &task, err
}

func (r *PackTaskRepository) FindByWaveID(ctx context.Context, waveID string) ([]*domain.PackTask, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"waveId": waveID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PackTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PackTaskRepository) FindByPackerID(ctx context.Context, packerID string) ([]*domain.PackTask, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"packerId": packerID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PackTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PackTaskRepository) FindByStatus(ctx context.Context, status domain.PackTaskStatus) ([]*domain.PackTask, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PackTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PackTaskRepository) FindByStation(ctx context.Context, station string) ([]*domain.PackTask, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"station": station})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PackTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PackTaskRepository) FindPending(ctx context.Context, limit int) ([]*domain.PackTask, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "priority", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{"status": domain.PackTaskStatusPending}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PackTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PackTaskRepository) FindByTrackingNumber(ctx context.Context, trackingNumber string) (*domain.PackTask, error) {
	var task domain.PackTask
	err := r.collection.FindOne(ctx, bson.M{"shippingLabel.trackingNumber": trackingNumber}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &task, err
}

func (r *PackTaskRepository) Delete(ctx context.Context, taskID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"taskId": taskID})
	return err
}

// GetOutboxRepository returns the outbox repository for this service
func (r *PackTaskRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
