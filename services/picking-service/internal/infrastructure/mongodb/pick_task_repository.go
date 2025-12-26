package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/picking-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PickTaskRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
}

func NewPickTaskRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *PickTaskRepository {
	collection := db.Collection("pick_tasks")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &PickTaskRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
	}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *PickTaskRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "taskId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "orderId", Value: 1}}},
		{Keys: bson.D{{Key: "waveId", Value: 1}}},
		{Keys: bson.D{{Key: "pickerId", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "zone", Value: 1}, {Key: "status", Value: 1}}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)

	// Create outbox indexes
	_ = r.outboxRepo.EnsureIndexes(ctx)
}

// Save persists a pick task with its domain events in a single transaction
func (r *PickTaskRepository) Save(ctx context.Context, task *domain.PickTask) error {
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
			return nil, fmt.Errorf("failed to save pick task: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := task.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.PickTaskCreatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "task/"+e.TaskID, e)
				case *domain.PickTaskAssignedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "task/"+e.TaskID, e)
				case *domain.ItemPickedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "task/"+e.TaskID, e)
				case *domain.PickTaskCompletedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "task/"+e.TaskID, e)
				case *domain.PickExceptionEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "task/"+e.TaskID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					task.TaskID,
					"PickTask",
					kafka.Topics.PickingEvents,
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

func (r *PickTaskRepository) FindByID(ctx context.Context, taskID string) (*domain.PickTask, error) {
	var task domain.PickTask
	err := r.collection.FindOne(ctx, bson.M{"taskId": taskID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &task, err
}

func (r *PickTaskRepository) FindByOrderID(ctx context.Context, orderID string) ([]*domain.PickTask, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"orderId": orderID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PickTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PickTaskRepository) FindByWaveID(ctx context.Context, waveID string) ([]*domain.PickTask, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"waveId": waveID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PickTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PickTaskRepository) FindByPickerID(ctx context.Context, pickerID string) ([]*domain.PickTask, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"pickerId": pickerID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PickTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PickTaskRepository) FindByStatus(ctx context.Context, status domain.PickTaskStatus) ([]*domain.PickTask, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PickTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PickTaskRepository) FindActiveByPicker(ctx context.Context, pickerID string) (*domain.PickTask, error) {
	var task domain.PickTask
	filter := bson.M{"pickerId": pickerID, "status": domain.PickTaskStatusInProgress}
	err := r.collection.FindOne(ctx, filter).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &task, err
}

func (r *PickTaskRepository) FindPendingByZone(ctx context.Context, zone string, limit int) ([]*domain.PickTask, error) {
	filter := bson.M{"status": domain.PickTaskStatusPending}
	if zone != "" {
		filter["zone"] = zone
	}
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "priority", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PickTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PickTaskRepository) Delete(ctx context.Context, taskID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"taskId": taskID})
	return err
}

// GetOutboxRepository returns the outbox repository for this service
func (r *PickTaskRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
