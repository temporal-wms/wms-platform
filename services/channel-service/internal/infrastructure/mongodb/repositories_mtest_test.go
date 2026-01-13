package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"

	"github.com/wms-platform/services/channel-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/outbox"
)

func TestRepositoryConstructors(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("channel repository", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		repo := NewChannelRepository(mt.DB)
		require.NotNil(t, repo)
	})

	mt.Run("channel order repository", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		repo := NewChannelOrderRepository(mt.DB)
		require.NotNil(t, repo)
	})

	mt.Run("sync job repository", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		repo := NewSyncJobRepository(mt.DB)
		require.NotNil(t, repo)
	})

	mt.Run("outbox repository", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		repo := NewOutboxRepository(mt.DB)
		require.NotNil(t, repo)
	})
}

func TestChannelRepository_MockOps(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("operations", func(mt *mtest.T) {
		coll := mt.DB.Collection("channels")
		repo := &ChannelRepository{
			collection:       coll,
			outboxCollection: mt.DB.Collection("outbox"),
			eventFactory:     cloudevents.NewEventFactory(cloudevents.SourceChannel),
		}
		ctx := context.Background()
		ns := coll.Database().Name() + "." + coll.Name()

		channel, err := domain.NewChannel(
			"tenant-1",
			"seller-1",
			domain.ChannelTypeShopify,
			"Shop",
			"",
			domain.ChannelCredentials{},
			domain.SyncSettings{},
		)
		require.NoError(t, err)

		mt.AddMockResponses(
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
			mtest.CreateSuccessResponse(),
			mtest.CreateSuccessResponse(),
		)
		err = repo.Save(ctx, channel)
		require.NoError(t, err)
		assert.Empty(t, channel.DomainEvents())

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "channelId", Value: channel.ChannelID},
			{Key: "sellerId", Value: channel.SellerID},
			{Key: "type", Value: string(domain.ChannelTypeShopify)},
		}))
		found, err := repo.FindByID(ctx, channel.ChannelID)
		require.NoError(t, err)
		require.NotNil(t, found)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		_, err = repo.FindByID(ctx, "missing")
		require.ErrorIs(t, err, domain.ErrChannelNotFound)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "channelId", Value: "ch-1"},
			{Key: "sellerId", Value: "seller-1"},
		}))
		list, err := repo.FindBySellerID(ctx, "seller-1")
		require.NoError(t, err)
		require.Len(t, list, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "channelId", Value: "ch-2"},
			{Key: "type", Value: string(domain.ChannelTypeShopify)},
		}))
		list, err = repo.FindByType(ctx, domain.ChannelTypeShopify)
		require.NoError(t, err)
		require.Len(t, list, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "channelId", Value: "ch-3"},
			{Key: "status", Value: string(domain.ChannelStatusActive)},
		}))
		list, err = repo.FindActiveChannels(ctx)
		require.NoError(t, err)
		require.Len(t, list, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		list, err = repo.FindChannelsNeedingSync(ctx, domain.SyncTypeOrders, time.Hour)
		require.NoError(t, err)
		require.Len(t, list, 0)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		list, err = repo.FindChannelsNeedingSync(ctx, domain.SyncTypeInventory, time.Hour)
		require.NoError(t, err)
		require.Len(t, list, 0)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		list, err = repo.FindChannelsNeedingSync(ctx, domain.SyncTypeProducts, time.Hour)
		require.NoError(t, err)
		require.Len(t, list, 0)

		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err = repo.UpdateStatus(ctx, "ch-1", domain.ChannelStatusPaused)
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}, bson.E{Key: "nModified", Value: 0}))
		err = repo.UpdateStatus(ctx, "missing", domain.ChannelStatusPaused)
		require.ErrorIs(t, err, domain.ErrChannelNotFound)

		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
		err = repo.Delete(ctx, "ch-1")
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		err = repo.Delete(ctx, "missing")
		require.ErrorIs(t, err, domain.ErrChannelNotFound)
	})
}

func TestChannelOrderRepository_MockOps(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("operations", func(mt *mtest.T) {
		coll := mt.DB.Collection("channel_orders")
		repo := &ChannelOrderRepository{
			collection:       coll,
			outboxCollection: mt.DB.Collection("outbox"),
		}
		ctx := context.Background()
		ns := coll.Database().Name() + "." + coll.Name()

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err := repo.Save(ctx, &domain.ChannelOrder{ExternalOrderID: "ext-1"})
		require.NoError(t, err)

		err = repo.SaveAll(ctx, nil)
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err = repo.SaveAll(ctx, []*domain.ChannelOrder{{ExternalOrderID: "ext-1"}})
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "externalOrderId", Value: "ext-1"},
			{Key: "channelId", Value: "ch-1"},
		}))
		order, err := repo.FindByExternalID(ctx, "ch-1", "ext-1")
		require.NoError(t, err)
		require.NotNil(t, order)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		order, err = repo.FindByExternalID(ctx, "ch-1", "missing")
		require.NoError(t, err)
		require.Nil(t, order)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "externalOrderId", Value: "ext-2"},
			{Key: "channelId", Value: "ch-1"},
		}))
		list, err := repo.FindByChannelID(ctx, "ch-1", domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		require.Len(t, list, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "externalOrderId", Value: "ext-3"},
			{Key: "imported", Value: false},
		}))
		list, err = repo.FindUnimported(ctx, "ch-1")
		require.NoError(t, err)
		require.Len(t, list, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "externalOrderId", Value: "ext-4"},
			{Key: "trackingPushed", Value: false},
		}))
		list, err = repo.FindWithoutTracking(ctx, "ch-1")
		require.NoError(t, err)
		require.Len(t, list, 1)

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err = repo.MarkImported(ctx, "ext-1", "wms-1")
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err = repo.MarkTrackingPushed(ctx, "ext-1")
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "n", Value: int64(2)},
		}))
		count, err := repo.Count(ctx, "ch-1")
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})
}

func TestSyncJobRepository_MockOps(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("operations", func(mt *mtest.T) {
		coll := mt.DB.Collection("sync_jobs")
		repo := &SyncJobRepository{collection: coll}
		ctx := context.Background()
		ns := coll.Database().Name() + "." + coll.Name()

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		job := domain.NewSyncJob("tenant-1", "seller-1", "ch-1", domain.SyncTypeOrders, "inbound")
		err := repo.Save(ctx, job)
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "jobId", Value: job.JobID},
			{Key: "channelId", Value: "ch-1"},
		}))
		found, err := repo.FindByID(ctx, job.JobID)
		require.NoError(t, err)
		require.NotNil(t, found)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		found, err = repo.FindByID(ctx, "missing")
		require.NoError(t, err)
		require.Nil(t, found)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "jobId", Value: "job-2"},
			{Key: "channelId", Value: "ch-1"},
		}))
		list, err := repo.FindByChannelID(ctx, "ch-1", domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		require.Len(t, list, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "jobId", Value: "job-3"},
			{Key: "status", Value: string(domain.SyncStatusRunning)},
		}))
		found, err = repo.FindRunning(ctx, "ch-1", domain.SyncTypeOrders)
		require.NoError(t, err)
		require.NotNil(t, found)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		found, err = repo.FindRunning(ctx, "ch-1", domain.SyncTypeOrders)
		require.NoError(t, err)
		require.Nil(t, found)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "jobId", Value: "job-4"},
			{Key: "channelId", Value: "ch-1"},
		}))
		found, err = repo.FindLatest(ctx, "ch-1", domain.SyncTypeOrders)
		require.NoError(t, err)
		require.NotNil(t, found)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		found, err = repo.FindLatest(ctx, "ch-1", domain.SyncTypeOrders)
		require.NoError(t, err)
		require.Nil(t, found)
	})
}

func TestOutboxRepository_MockOps(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("operations", func(mt *mtest.T) {
		coll := mt.DB.Collection("outbox")
		repo := &OutboxRepository{collection: coll}
		ctx := context.Background()
		ns := coll.Database().Name() + "." + coll.Name()

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err := repo.Save(ctx, &outbox.OutboxEvent{ID: "evt-1"})
		require.NoError(t, err)

		err = repo.SaveAll(ctx, nil)
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err = repo.SaveAll(ctx, []*outbox.OutboxEvent{{ID: "evt-2"}})
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "_id", Value: "evt-3"},
			{Key: "aggregateId", Value: "agg-1"},
		}))
		list, err := repo.FindUnpublished(ctx, 10)
		require.NoError(t, err)
		require.Len(t, list, 1)

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err = repo.MarkPublished(ctx, "evt-1")
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err = repo.IncrementRetry(ctx, "evt-1", "boom")
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err = repo.DeletePublished(ctx, 3600)
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "_id", Value: "evt-1"},
			{Key: "aggregateId", Value: "agg-1"},
		}))
		event, err := repo.GetByID(ctx, "evt-1")
		require.NoError(t, err)
		require.NotNil(t, event)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		event, err = repo.GetByID(ctx, "missing")
		require.NoError(t, err)
		require.Nil(t, event)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "_id", Value: "evt-2"},
			{Key: "aggregateId", Value: "agg-1"},
		}))
		list, err = repo.FindByAggregateID(ctx, "agg-1")
		require.NoError(t, err)
		require.Len(t, list, 1)
	})
}
