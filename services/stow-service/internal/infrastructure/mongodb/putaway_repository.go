package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/services/stow-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PutawayTaskRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
}

func NewPutawayTaskRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *PutawayTaskRepository {
	collection := db.Collection("putaway_tasks")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &PutawayTaskRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
	}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *PutawayTaskRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "taskId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "shipmentId", Value: 1}}},
		{Keys: bson.D{{Key: "sku", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "assignedWorkerId", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "priority", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "targetLocationId", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)

	// Create outbox indexes
	_ = r.outboxRepo.EnsureIndexes(ctx)
}

// Save persists a putaway task with its domain events in a single transaction
func (r *PutawayTaskRepository) Save(ctx context.Context, task *domain.PutawayTask) error {
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
			return nil, fmt.Errorf("failed to save putaway task: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := task.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.PutawayTaskCreatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "task/"+e.TaskID, e)
				case *domain.LocationAssignedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "task/"+e.TaskID, e)
				case *domain.ItemStowedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "task/"+e.TaskID, e)
				case *domain.PutawayTaskAssignedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "task/"+e.TaskID, e)
				case *domain.PutawayTaskCompletedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "task/"+e.TaskID, e)
				case *domain.PutawayTaskFailedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "task/"+e.TaskID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					task.TaskID,
					"PutawayTask",
					kafka.Topics.StowEvents,
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

func (r *PutawayTaskRepository) FindByID(ctx context.Context, taskID string) (*domain.PutawayTask, error) {
	var task domain.PutawayTask
	err := r.collection.FindOne(ctx, bson.M{"taskId": taskID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &task, err
}

func (r *PutawayTaskRepository) FindByStatus(ctx context.Context, status domain.PutawayStatus, pagination domain.Pagination) ([]*domain.PutawayTask, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "priority", Value: 1}, {Key: "createdAt", Value: 1}}).
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit())
	cursor, err := r.collection.Find(ctx, bson.M{"status": status}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PutawayTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PutawayTaskRepository) FindByWorkerID(ctx context.Context, workerID string, pagination domain.Pagination) ([]*domain.PutawayTask, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit())
	cursor, err := r.collection.Find(ctx, bson.M{"assignedWorkerId": workerID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PutawayTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PutawayTaskRepository) FindPendingTasks(ctx context.Context, limit int) ([]*domain.PutawayTask, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "priority", Value: 1}, {Key: "createdAt", Value: 1}}).
		SetLimit(int64(limit))
	cursor, err := r.collection.Find(ctx, bson.M{"status": domain.PutawayStatusPending}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PutawayTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PutawayTaskRepository) FindByShipmentID(ctx context.Context, shipmentID string) ([]*domain.PutawayTask, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{"shipmentId": shipmentID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PutawayTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PutawayTaskRepository) FindBySKU(ctx context.Context, sku string, pagination domain.Pagination) ([]*domain.PutawayTask, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit())
	cursor, err := r.collection.Find(ctx, bson.M{"sku": sku}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*domain.PutawayTask
	err = cursor.All(ctx, &tasks)
	return tasks, err
}

func (r *PutawayTaskRepository) UpdateStatus(ctx context.Context, taskID string, status domain.PutawayStatus) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"taskId": taskID},
		bson.M{"$set": bson.M{"status": status, "updatedAt": time.Now()}},
	)
	return err
}

func (r *PutawayTaskRepository) Delete(ctx context.Context, taskID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"taskId": taskID})
	return err
}

func (r *PutawayTaskRepository) Count(ctx context.Context, filter domain.TaskFilter) (int64, error) {
	mongoFilter := bson.M{}
	if filter.Status != nil {
		mongoFilter["status"] = *filter.Status
	}
	if filter.WorkerID != nil {
		mongoFilter["assignedWorkerId"] = *filter.WorkerID
	}
	if filter.ShipmentID != nil {
		mongoFilter["shipmentId"] = *filter.ShipmentID
	}
	if filter.SKU != nil {
		mongoFilter["sku"] = *filter.SKU
	}
	if filter.Strategy != nil {
		mongoFilter["strategy"] = *filter.Strategy
	}
	return r.collection.CountDocuments(ctx, mongoFilter)
}

// GetOutboxRepository returns the outbox repository for this service
func (r *PutawayTaskRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
