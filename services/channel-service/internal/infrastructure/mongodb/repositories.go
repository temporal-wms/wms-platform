package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wms-platform/services/channel-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/outbox"
)

const (
	channelEventsTopic = "wms.channel.events"
)

// ChannelRepository implements domain.ChannelRepository
type ChannelRepository struct {
	collection       *mongo.Collection
	outboxCollection *mongo.Collection
	eventFactory     *cloudevents.EventFactory
}

// NewChannelRepository creates a new channel repository
func NewChannelRepository(db *mongo.Database) *ChannelRepository {
	collection := db.Collection("channels")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "channelId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "sellerId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "sellerId", Value: 1},
				{Key: "type", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "syncSettings.orderSync.lastSyncAt", Value: 1},
			},
		},
	}

	collection.Indexes().CreateMany(ctx, indexes)

	return &ChannelRepository{
		collection:       collection,
		outboxCollection: db.Collection("outbox"),
		eventFactory:     cloudevents.NewEventFactory(cloudevents.SourceChannel),
	}
}

func (r *ChannelRepository) Save(ctx context.Context, channel *domain.Channel) error {
	session, err := r.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		opts := options.Replace().SetUpsert(true)
		_, err := r.collection.ReplaceOne(sessCtx, bson.M{"channelId": channel.ChannelID}, channel, opts)
		if err != nil {
			return nil, err
		}

		// Store domain events in outbox as CloudEvents
		for _, event := range channel.DomainEvents() {
			cloudEvent := r.domainEventToCloudEvent(ctx, channel.ChannelID, event)
			outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
				channel.ChannelID,
				"Channel",
				channelEventsTopic,
				cloudEvent,
			)
			if err != nil {
				return nil, err
			}

			_, err = r.outboxCollection.InsertOne(sessCtx, outboxEvent)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	})

	if err == nil {
		channel.ClearDomainEvents()
	}

	return err
}

// domainEventToCloudEvent converts a domain event to a CloudEvent
func (r *ChannelRepository) domainEventToCloudEvent(ctx context.Context, channelID string, event domain.DomainEvent) *cloudevents.WMSCloudEvent {
	var eventType string
	switch event.EventType() {
	case "channel.connected":
		eventType = cloudevents.ChannelConnected
	case "channel.disconnected":
		eventType = cloudevents.ChannelDisconnected
	case "channel.order.imported":
		eventType = cloudevents.ChannelOrderImported
	case "channel.tracking.pushed":
		eventType = cloudevents.ChannelTrackingPushed
	case "channel.inventory.synced":
		eventType = cloudevents.ChannelInventorySynced
	case "channel.sync.completed":
		eventType = cloudevents.ChannelSyncCompleted
	case "channel.webhook.received":
		eventType = cloudevents.ChannelWebhookReceived
	default:
		eventType = "wms.channel." + event.EventType()
	}

	return r.eventFactory.CreateEvent(ctx, eventType, "channel/"+channelID, event)
}

func (r *ChannelRepository) FindByID(ctx context.Context, channelID string) (*domain.Channel, error) {
	var channel domain.Channel
	err := r.collection.FindOne(ctx, bson.M{"channelId": channelID}).Decode(&channel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrChannelNotFound
		}
		return nil, err
	}
	return &channel, nil
}

func (r *ChannelRepository) FindBySellerID(ctx context.Context, sellerID string) ([]*domain.Channel, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"sellerId": sellerID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var channels []*domain.Channel
	if err := cursor.All(ctx, &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *ChannelRepository) FindByType(ctx context.Context, channelType domain.ChannelType) ([]*domain.Channel, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"type": channelType})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var channels []*domain.Channel
	if err := cursor.All(ctx, &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *ChannelRepository) FindActiveChannels(ctx context.Context) ([]*domain.Channel, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": domain.ChannelStatusActive})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var channels []*domain.Channel
	if err := cursor.All(ctx, &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *ChannelRepository) FindChannelsNeedingSync(ctx context.Context, syncType domain.SyncType, threshold time.Duration) ([]*domain.Channel, error) {
	cutoff := time.Now().Add(-threshold)

	var filter bson.M
	switch syncType {
	case domain.SyncTypeOrders:
		filter = bson.M{
			"status": domain.ChannelStatusActive,
			"syncSettings.orderSync.enabled": true,
			"$or": []bson.M{
				{"syncSettings.orderSync.lastSyncAt": bson.M{"$lt": cutoff}},
				{"syncSettings.orderSync.lastSyncAt": bson.M{"$exists": false}},
			},
		}
	case domain.SyncTypeInventory:
		filter = bson.M{
			"status": domain.ChannelStatusActive,
			"syncSettings.inventorySync.enabled": true,
			"$or": []bson.M{
				{"syncSettings.inventorySync.lastSyncAt": bson.M{"$lt": cutoff}},
				{"syncSettings.inventorySync.lastSyncAt": bson.M{"$exists": false}},
			},
		}
	default:
		filter = bson.M{"status": domain.ChannelStatusActive}
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var channels []*domain.Channel
	if err := cursor.All(ctx, &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *ChannelRepository) UpdateStatus(ctx context.Context, channelID string, status domain.ChannelStatus) error {
	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"channelId": channelID},
		bson.M{
			"$set": bson.M{
				"status":    status,
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrChannelNotFound
	}
	return nil
}

func (r *ChannelRepository) Delete(ctx context.Context, channelID string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"channelId": channelID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return domain.ErrChannelNotFound
	}
	return nil
}

// ChannelOrderRepository implements domain.ChannelOrderRepository
type ChannelOrderRepository struct {
	collection       *mongo.Collection
	outboxCollection *mongo.Collection
}

// NewChannelOrderRepository creates a new channel order repository
func NewChannelOrderRepository(db *mongo.Database) *ChannelOrderRepository {
	collection := db.Collection("channel_orders")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "externalOrderId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "channelId", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "channelId", Value: 1},
				{Key: "externalOrderId", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "channelId", Value: 1},
				{Key: "imported", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "channelId", Value: 1},
				{Key: "trackingPushed", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "sellerId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "externalCreatedAt", Value: -1}},
		},
	}

	collection.Indexes().CreateMany(ctx, indexes)

	return &ChannelOrderRepository{
		collection:       collection,
		outboxCollection: db.Collection("outbox"),
	}
}

func (r *ChannelOrderRepository) Save(ctx context.Context, order *domain.ChannelOrder) error {
	opts := options.Replace().SetUpsert(true)
	_, err := r.collection.ReplaceOne(ctx, bson.M{"externalOrderId": order.ExternalOrderID}, order, opts)
	return err
}

func (r *ChannelOrderRepository) SaveAll(ctx context.Context, orders []*domain.ChannelOrder) error {
	if len(orders) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, len(orders))
	for i, order := range orders {
		models[i] = mongo.NewReplaceOneModel().
			SetFilter(bson.M{"externalOrderId": order.ExternalOrderID}).
			SetReplacement(order).
			SetUpsert(true)
	}

	_, err := r.collection.BulkWrite(ctx, models)
	return err
}

func (r *ChannelOrderRepository) FindByExternalID(ctx context.Context, channelID, externalOrderID string) (*domain.ChannelOrder, error) {
	var order domain.ChannelOrder
	err := r.collection.FindOne(ctx, bson.M{
		"channelId":       channelID,
		"externalOrderId": externalOrderID,
	}).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *ChannelOrderRepository) FindByChannelID(ctx context.Context, channelID string, pagination domain.Pagination) ([]*domain.ChannelOrder, error) {
	opts := options.Find().
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit()).
		SetSort(bson.D{{Key: "externalCreatedAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{"channelId": channelID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []*domain.ChannelOrder
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *ChannelOrderRepository) FindUnimported(ctx context.Context, channelID string) ([]*domain.ChannelOrder, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"channelId": channelID,
		"imported":  false,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []*domain.ChannelOrder
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *ChannelOrderRepository) FindWithoutTracking(ctx context.Context, channelID string) ([]*domain.ChannelOrder, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"channelId":      channelID,
		"imported":       true,
		"trackingPushed": false,
		"wmsOrderId":     bson.M{"$ne": ""},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []*domain.ChannelOrder
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *ChannelOrderRepository) MarkImported(ctx context.Context, externalOrderID, wmsOrderID string) error {
	now := time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"externalOrderId": externalOrderID},
		bson.M{
			"$set": bson.M{
				"imported":   true,
				"wmsOrderId": wmsOrderID,
				"importedAt": &now,
				"updatedAt":  now,
			},
		},
	)
	return err
}

func (r *ChannelOrderRepository) MarkTrackingPushed(ctx context.Context, externalOrderID string) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"externalOrderId": externalOrderID},
		bson.M{
			"$set": bson.M{
				"trackingPushed":   true,
				"trackingPushedAt": time.Now(),
			},
		},
	)
	return err
}

func (r *ChannelOrderRepository) Count(ctx context.Context, channelID string) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"channelId": channelID})
}

// SyncJobRepository implements domain.SyncJobRepository
type SyncJobRepository struct {
	collection *mongo.Collection
}

// NewSyncJobRepository creates a new sync job repository
func NewSyncJobRepository(db *mongo.Database) *SyncJobRepository {
	collection := db.Collection("sync_jobs")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "jobId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "channelId", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "channelId", Value: 1},
				{Key: "type", Value: 1},
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "channelId", Value: 1},
				{Key: "type", Value: 1},
				{Key: "startedAt", Value: -1},
			},
		},
	}

	collection.Indexes().CreateMany(ctx, indexes)

	return &SyncJobRepository{collection: collection}
}

func (r *SyncJobRepository) Save(ctx context.Context, job *domain.SyncJob) error {
	opts := options.Replace().SetUpsert(true)
	_, err := r.collection.ReplaceOne(ctx, bson.M{"jobId": job.ID}, job, opts)
	return err
}

func (r *SyncJobRepository) FindByID(ctx context.Context, jobID string) (*domain.SyncJob, error) {
	var job domain.SyncJob
	err := r.collection.FindOne(ctx, bson.M{"jobId": jobID}).Decode(&job)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

func (r *SyncJobRepository) FindByChannelID(ctx context.Context, channelID string, pagination domain.Pagination) ([]*domain.SyncJob, error) {
	opts := options.Find().
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit()).
		SetSort(bson.D{{Key: "startedAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{"channelId": channelID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []*domain.SyncJob
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *SyncJobRepository) FindRunning(ctx context.Context, channelID string, syncType domain.SyncType) (*domain.SyncJob, error) {
	var job domain.SyncJob
	err := r.collection.FindOne(ctx, bson.M{
		"channelId": channelID,
		"type":      syncType,
		"status":    domain.SyncStatusRunning,
	}).Decode(&job)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

func (r *SyncJobRepository) FindLatest(ctx context.Context, channelID string, syncType domain.SyncType) (*domain.SyncJob, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "startedAt", Value: -1}})

	var job domain.SyncJob
	err := r.collection.FindOne(ctx, bson.M{
		"channelId": channelID,
		"type":      syncType,
	}, opts).Decode(&job)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}
