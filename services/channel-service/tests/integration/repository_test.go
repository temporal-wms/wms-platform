package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/services/channel-service/internal/domain"
	"github.com/wms-platform/services/channel-service/internal/infrastructure/mongodb"
	sharedtesting "github.com/wms-platform/shared/pkg/testing"
)

// Test fixtures
func createTestChannel(sellerID string, channelType domain.ChannelType) *domain.Channel {
	channel, _ := domain.NewChannel(
		"TNT-001",
		sellerID,
		channelType,
		"Test Store",
		"https://teststore.myshopify.com",
		domain.ChannelCredentials{
			APIKey:      "test-api-key",
			APISecret:   "test-api-secret",
			AccessToken: "test-access-token",
			StoreDomain: "teststore.myshopify.com",
		},
		domain.SyncSettings{
			AutoImportOrders:     true,
			AutoSyncInventory:    true,
			AutoPushTracking:     true,
			OrderSyncIntervalMin: 15,
		},
	)
	return channel
}

func createTestChannelOrder(channelID, externalOrderID string) *domain.ChannelOrder {
	now := time.Now().UTC()
	return &domain.ChannelOrder{
		TenantID:            "TNT-001",
		SellerID:            "SLR-001",
		ChannelID:           channelID,
		ExternalOrderID:     externalOrderID,
		ExternalOrderNumber: "1001",
		ExternalCreatedAt:   now,
		Customer: domain.ChannelCustomer{
			ExternalID: "CUST-001",
			Email:      "customer@example.com",
			FirstName:  "John",
			LastName:   "Doe",
			Phone:      "+1234567890",
		},
		ShippingAddr: domain.ChannelAddress{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "New York",
			Province:  "NY",
			Zip:       "10001",
			Country:   "US",
		},
		LineItems: []domain.ChannelLineItem{
			{
				ExternalID:       "LI-001",
				SKU:              "SKU-001",
				Title:            "Test Product",
				Quantity:         2,
				Price:            29.99,
				RequiresShipping: true,
			},
		},
		Currency:          "USD",
		Subtotal:          59.98,
		ShippingCost:      9.99,
		Tax:               5.50,
		Total:             75.47,
		FinancialStatus:   "paid",
		FulfillmentStatus: "unfulfilled",
		Imported:          false,
		TrackingPushed:    false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func setupChannelRepository(t *testing.T) (*mongodb.ChannelRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	db := client.Database("test_channel_db")
	repo := mongodb.NewChannelRepository(db)

	cleanup := func() {
		if err := client.Disconnect(ctx); err != nil {
			t.Logf("Failed to disconnect MongoDB client: %v", err)
		}
		if err := mongoContainer.Close(ctx); err != nil {
			t.Logf("Failed to close MongoDB container: %v", err)
		}
	}

	return repo, mongoContainer, cleanup
}

func setupChannelOrderRepository(t *testing.T) (*mongodb.ChannelOrderRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	db := client.Database("test_channel_db")
	repo := mongodb.NewChannelOrderRepository(db)

	cleanup := func() {
		if err := client.Disconnect(ctx); err != nil {
			t.Logf("Failed to disconnect MongoDB client: %v", err)
		}
		if err := mongoContainer.Close(ctx); err != nil {
			t.Logf("Failed to close MongoDB container: %v", err)
		}
	}

	return repo, mongoContainer, cleanup
}

func setupSyncJobRepository(t *testing.T) (*mongodb.SyncJobRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	db := client.Database("test_channel_db")
	repo := mongodb.NewSyncJobRepository(db)

	cleanup := func() {
		if err := client.Disconnect(ctx); err != nil {
			t.Logf("Failed to disconnect MongoDB client: %v", err)
		}
		if err := mongoContainer.Close(ctx); err != nil {
			t.Logf("Failed to close MongoDB container: %v", err)
		}
	}

	return repo, mongoContainer, cleanup
}

// ChannelRepository Tests

func TestChannelRepository_Save(t *testing.T) {
	repo, _, cleanup := setupChannelRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new channel", func(t *testing.T) {
		channel := createTestChannel("SLR-001", domain.ChannelTypeShopify)

		err := repo.Save(ctx, channel)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, channel.ChannelID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, channel.ChannelID, found.ChannelID)
		assert.Equal(t, "SLR-001", found.SellerID)
		assert.Equal(t, domain.ChannelTypeShopify, found.Type)
		assert.Equal(t, domain.ChannelStatusActive, found.Status)
	})

	t.Run("Update existing channel (upsert)", func(t *testing.T) {
		channel := createTestChannel("SLR-002", domain.ChannelTypeAmazon)
		err := repo.Save(ctx, channel)
		require.NoError(t, err)

		// Update and save again
		channel.Name = "Updated Store Name"
		channel.Pause()
		err = repo.Save(ctx, channel)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, channel.ChannelID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "Updated Store Name", found.Name)
		assert.Equal(t, domain.ChannelStatusPaused, found.Status)
	})
}

func TestChannelRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupChannelRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing channel", func(t *testing.T) {
		channel := createTestChannel("SLR-003", domain.ChannelTypeShopify)
		err := repo.Save(ctx, channel)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, channel.ChannelID)
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, channel.ChannelID, found.ChannelID)
	})

	t.Run("Find non-existent channel", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "CH-NONEXISTENT")
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Equal(t, domain.ErrChannelNotFound, err)
	})
}

func TestChannelRepository_FindBySellerID(t *testing.T) {
	repo, _, cleanup := setupChannelRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sellerID := "SLR-004"

	// Create channels for the seller
	channel1 := createTestChannel(sellerID, domain.ChannelTypeShopify)
	channel2 := createTestChannel(sellerID, domain.ChannelTypeAmazon)
	err := repo.Save(ctx, channel1)
	require.NoError(t, err)
	err = repo.Save(ctx, channel2)
	require.NoError(t, err)

	t.Run("Find all channels for seller", func(t *testing.T) {
		channels, err := repo.FindBySellerID(ctx, sellerID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(channels), 2)

		for _, ch := range channels {
			assert.Equal(t, sellerID, ch.SellerID)
		}
	})

	t.Run("Find for non-existent seller", func(t *testing.T) {
		channels, err := repo.FindBySellerID(ctx, "SLR-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, channels)
	})
}

func TestChannelRepository_FindByType(t *testing.T) {
	repo, _, cleanup := setupChannelRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create channels of different types
	shopifyChannel := createTestChannel("SLR-005", domain.ChannelTypeShopify)
	amazonChannel := createTestChannel("SLR-006", domain.ChannelTypeAmazon)
	err := repo.Save(ctx, shopifyChannel)
	require.NoError(t, err)
	err = repo.Save(ctx, amazonChannel)
	require.NoError(t, err)

	t.Run("Find Shopify channels", func(t *testing.T) {
		channels, err := repo.FindByType(ctx, domain.ChannelTypeShopify)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(channels), 1)

		for _, ch := range channels {
			assert.Equal(t, domain.ChannelTypeShopify, ch.Type)
		}
	})

	t.Run("Find Amazon channels", func(t *testing.T) {
		channels, err := repo.FindByType(ctx, domain.ChannelTypeAmazon)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(channels), 1)

		for _, ch := range channels {
			assert.Equal(t, domain.ChannelTypeAmazon, ch.Type)
		}
	})
}

func TestChannelRepository_FindActiveChannels(t *testing.T) {
	repo, _, cleanup := setupChannelRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create active and paused channels
	activeChannel := createTestChannel("SLR-007", domain.ChannelTypeShopify)
	pausedChannel := createTestChannel("SLR-008", domain.ChannelTypeShopify)
	pausedChannel.Pause()

	err := repo.Save(ctx, activeChannel)
	require.NoError(t, err)
	err = repo.Save(ctx, pausedChannel)
	require.NoError(t, err)

	t.Run("Find only active channels", func(t *testing.T) {
		channels, err := repo.FindActiveChannels(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(channels), 1)

		for _, ch := range channels {
			assert.Equal(t, domain.ChannelStatusActive, ch.Status)
		}
	})
}

func TestChannelRepository_UpdateStatus(t *testing.T) {
	repo, _, cleanup := setupChannelRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Update channel status", func(t *testing.T) {
		channel := createTestChannel("SLR-009", domain.ChannelTypeShopify)
		err := repo.Save(ctx, channel)
		require.NoError(t, err)

		err = repo.UpdateStatus(ctx, channel.ChannelID, domain.ChannelStatusPaused)
		assert.NoError(t, err)

		// Verify status was updated
		found, err := repo.FindByID(ctx, channel.ChannelID)
		require.NoError(t, err)
		assert.Equal(t, domain.ChannelStatusPaused, found.Status)
	})

	t.Run("Update non-existent channel", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, "CH-NONEXISTENT", domain.ChannelStatusPaused)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrChannelNotFound, err)
	})
}

func TestChannelRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupChannelRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Delete existing channel", func(t *testing.T) {
		channel := createTestChannel("SLR-010", domain.ChannelTypeShopify)
		err := repo.Save(ctx, channel)
		require.NoError(t, err)

		err = repo.Delete(ctx, channel.ChannelID)
		assert.NoError(t, err)

		// Verify it was deleted
		found, err := repo.FindByID(ctx, channel.ChannelID)
		assert.Error(t, err)
		assert.Nil(t, found)
	})

	t.Run("Delete non-existent channel", func(t *testing.T) {
		err := repo.Delete(ctx, "CH-NONEXISTENT")
		assert.Error(t, err)
		assert.Equal(t, domain.ErrChannelNotFound, err)
	})
}

// ChannelOrderRepository Tests

func TestChannelOrderRepository_Save(t *testing.T) {
	repo, _, cleanup := setupChannelOrderRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new order", func(t *testing.T) {
		order := createTestChannelOrder("CH-001", "EXT-ORD-001")

		err := repo.Save(ctx, order)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByExternalID(ctx, "CH-001", "EXT-ORD-001")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "EXT-ORD-001", found.ExternalOrderID)
		assert.Equal(t, "CH-001", found.ChannelID)
		assert.Equal(t, 75.47, found.Total)
	})

	t.Run("Update existing order (upsert)", func(t *testing.T) {
		order := createTestChannelOrder("CH-001", "EXT-ORD-002")
		err := repo.Save(ctx, order)
		require.NoError(t, err)

		// Update and save again
		order.FulfillmentStatus = "partial"
		err = repo.Save(ctx, order)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByExternalID(ctx, "CH-001", "EXT-ORD-002")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "partial", found.FulfillmentStatus)
	})
}

func TestChannelOrderRepository_SaveAll(t *testing.T) {
	repo, _, cleanup := setupChannelOrderRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save multiple orders", func(t *testing.T) {
		orders := []*domain.ChannelOrder{
			createTestChannelOrder("CH-002", "EXT-ORD-003"),
			createTestChannelOrder("CH-002", "EXT-ORD-004"),
			createTestChannelOrder("CH-002", "EXT-ORD-005"),
		}

		err := repo.SaveAll(ctx, orders)
		assert.NoError(t, err)

		// Verify all were saved
		pagination := domain.Pagination{Page: 1, PageSize: 10}
		found, err := repo.FindByChannelID(ctx, "CH-002", pagination)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(found), 3)
	})

	t.Run("Save empty list", func(t *testing.T) {
		err := repo.SaveAll(ctx, []*domain.ChannelOrder{})
		assert.NoError(t, err)
	})
}

func TestChannelOrderRepository_FindByExternalID(t *testing.T) {
	repo, _, cleanup := setupChannelOrderRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing order", func(t *testing.T) {
		order := createTestChannelOrder("CH-003", "EXT-ORD-006")
		err := repo.Save(ctx, order)
		require.NoError(t, err)

		found, err := repo.FindByExternalID(ctx, "CH-003", "EXT-ORD-006")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "EXT-ORD-006", found.ExternalOrderID)
	})

	t.Run("Find non-existent order", func(t *testing.T) {
		found, err := repo.FindByExternalID(ctx, "CH-003", "EXT-ORD-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestChannelOrderRepository_FindByChannelID(t *testing.T) {
	repo, _, cleanup := setupChannelOrderRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	channelID := "CH-004"

	// Create orders for the channel
	for i := 0; i < 5; i++ {
		order := createTestChannelOrder(channelID, "EXT-ORD-10"+string(rune('0'+i)))
		err := repo.Save(ctx, order)
		require.NoError(t, err)
	}

	t.Run("Find all orders for channel", func(t *testing.T) {
		pagination := domain.Pagination{Page: 1, PageSize: 10}
		orders, err := repo.FindByChannelID(ctx, channelID, pagination)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(orders), 5)

		for _, o := range orders {
			assert.Equal(t, channelID, o.ChannelID)
		}
	})

	t.Run("Find with pagination", func(t *testing.T) {
		pagination := domain.Pagination{Page: 1, PageSize: 2}
		orders, err := repo.FindByChannelID(ctx, channelID, pagination)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(orders), 2)
	})
}

func TestChannelOrderRepository_FindUnimported(t *testing.T) {
	repo, _, cleanup := setupChannelOrderRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	channelID := "CH-005"

	// Create imported and unimported orders
	unimportedOrder := createTestChannelOrder(channelID, "EXT-ORD-UNIMP-001")
	importedOrder := createTestChannelOrder(channelID, "EXT-ORD-IMP-001")
	importedOrder.MarkImported("WMS-ORD-001")

	err := repo.Save(ctx, unimportedOrder)
	require.NoError(t, err)
	err = repo.Save(ctx, importedOrder)
	require.NoError(t, err)

	t.Run("Find unimported orders", func(t *testing.T) {
		orders, err := repo.FindUnimported(ctx, channelID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(orders), 1)

		for _, o := range orders {
			assert.False(t, o.Imported)
		}
	})
}

func TestChannelOrderRepository_FindWithoutTracking(t *testing.T) {
	repo, _, cleanup := setupChannelOrderRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	channelID := "CH-006"

	// Create orders with and without tracking
	orderWithTracking := createTestChannelOrder(channelID, "EXT-ORD-TRACK-001")
	orderWithTracking.MarkImported("WMS-ORD-002")
	orderWithTracking.MarkTrackingPushed()

	orderWithoutTracking := createTestChannelOrder(channelID, "EXT-ORD-NOTRACK-001")
	orderWithoutTracking.MarkImported("WMS-ORD-003")

	err := repo.Save(ctx, orderWithTracking)
	require.NoError(t, err)
	err = repo.Save(ctx, orderWithoutTracking)
	require.NoError(t, err)

	t.Run("Find orders without tracking", func(t *testing.T) {
		orders, err := repo.FindWithoutTracking(ctx, channelID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(orders), 1)

		for _, o := range orders {
			assert.True(t, o.Imported)
			assert.False(t, o.TrackingPushed)
		}
	})
}

func TestChannelOrderRepository_MarkImported(t *testing.T) {
	repo, _, cleanup := setupChannelOrderRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Mark order as imported", func(t *testing.T) {
		order := createTestChannelOrder("CH-007", "EXT-ORD-MARK-IMP")
		err := repo.Save(ctx, order)
		require.NoError(t, err)

		err = repo.MarkImported(ctx, "EXT-ORD-MARK-IMP", "WMS-ORD-004")
		assert.NoError(t, err)

		// Verify it was marked
		found, err := repo.FindByExternalID(ctx, "CH-007", "EXT-ORD-MARK-IMP")
		require.NoError(t, err)
		assert.True(t, found.Imported)
		assert.Equal(t, "WMS-ORD-004", found.WMSOrderID)
		assert.NotNil(t, found.ImportedAt)
	})
}

func TestChannelOrderRepository_MarkTrackingPushed(t *testing.T) {
	repo, _, cleanup := setupChannelOrderRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Mark tracking as pushed", func(t *testing.T) {
		order := createTestChannelOrder("CH-008", "EXT-ORD-MARK-TRACK")
		order.MarkImported("WMS-ORD-005")
		err := repo.Save(ctx, order)
		require.NoError(t, err)

		err = repo.MarkTrackingPushed(ctx, "EXT-ORD-MARK-TRACK")
		assert.NoError(t, err)

		// Verify it was marked
		found, err := repo.FindByExternalID(ctx, "CH-008", "EXT-ORD-MARK-TRACK")
		require.NoError(t, err)
		assert.True(t, found.TrackingPushed)
		assert.NotNil(t, found.TrackingPushedAt)
	})
}

func TestChannelOrderRepository_Count(t *testing.T) {
	repo, _, cleanup := setupChannelOrderRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	channelID := "CH-009"

	// Create orders
	for i := 0; i < 5; i++ {
		order := createTestChannelOrder(channelID, "EXT-ORD-CNT-"+string(rune('0'+i)))
		err := repo.Save(ctx, order)
		require.NoError(t, err)
	}

	t.Run("Count orders for channel", func(t *testing.T) {
		count, err := repo.Count(ctx, channelID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(5))
	})

	t.Run("Count for non-existent channel", func(t *testing.T) {
		count, err := repo.Count(ctx, "CH-NONEXISTENT")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

// SyncJobRepository Tests

func TestSyncJobRepository_Save(t *testing.T) {
	repo, _, cleanup := setupSyncJobRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new sync job", func(t *testing.T) {
		job := domain.NewSyncJob("TNT-001", "SLR-001", "CH-010", domain.SyncTypeOrders, "inbound")

		err := repo.Save(ctx, job)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, job.JobID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, job.JobID, found.JobID)
		assert.Equal(t, domain.SyncTypeOrders, found.Type)
		assert.Equal(t, domain.SyncStatusPending, found.Status)
	})

	t.Run("Update existing job (upsert)", func(t *testing.T) {
		job := domain.NewSyncJob("TNT-001", "SLR-001", "CH-011", domain.SyncTypeInventory, "outbound")
		err := repo.Save(ctx, job)
		require.NoError(t, err)

		// Update and save again
		job.Start()
		job.SetTotalItems(100)
		err = repo.Save(ctx, job)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, job.JobID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, domain.SyncStatusRunning, found.Status)
		assert.Equal(t, 100, found.TotalItems)
	})
}

func TestSyncJobRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupSyncJobRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing job", func(t *testing.T) {
		job := domain.NewSyncJob("TNT-001", "SLR-001", "CH-012", domain.SyncTypeOrders, "inbound")
		err := repo.Save(ctx, job)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, job.JobID)
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, job.JobID, found.JobID)
	})

	t.Run("Find non-existent job", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "SYNC-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestSyncJobRepository_FindByChannelID(t *testing.T) {
	repo, _, cleanup := setupSyncJobRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	channelID := "CH-013"

	// Create jobs for the channel
	for i := 0; i < 5; i++ {
		job := domain.NewSyncJob("TNT-001", "SLR-001", channelID, domain.SyncTypeOrders, "inbound")
		job.Start()
		err := repo.Save(ctx, job)
		require.NoError(t, err)
	}

	t.Run("Find all jobs for channel", func(t *testing.T) {
		pagination := domain.Pagination{Page: 1, PageSize: 10}
		jobs, err := repo.FindByChannelID(ctx, channelID, pagination)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(jobs), 5)

		for _, j := range jobs {
			assert.Equal(t, channelID, j.ChannelID)
		}
	})
}

func TestSyncJobRepository_FindRunning(t *testing.T) {
	repo, _, cleanup := setupSyncJobRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	channelID := "CH-014"

	// Create a running job
	runningJob := domain.NewSyncJob("TNT-001", "SLR-001", channelID, domain.SyncTypeOrders, "inbound")
	runningJob.Start()
	err := repo.Save(ctx, runningJob)
	require.NoError(t, err)

	// Create a completed job
	completedJob := domain.NewSyncJob("TNT-001", "SLR-001", channelID, domain.SyncTypeOrders, "inbound")
	completedJob.Start()
	completedJob.Complete()
	err = repo.Save(ctx, completedJob)
	require.NoError(t, err)

	t.Run("Find running job", func(t *testing.T) {
		found, err := repo.FindRunning(ctx, channelID, domain.SyncTypeOrders)
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, domain.SyncStatusRunning, found.Status)
	})

	t.Run("No running job for inventory sync", func(t *testing.T) {
		found, err := repo.FindRunning(ctx, channelID, domain.SyncTypeInventory)
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestSyncJobRepository_FindLatest(t *testing.T) {
	repo, _, cleanup := setupSyncJobRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	channelID := "CH-015"

	// Create multiple jobs
	for i := 0; i < 3; i++ {
		job := domain.NewSyncJob("TNT-001", "SLR-001", channelID, domain.SyncTypeOrders, "inbound")
		job.Start()
		if i == 2 {
			job.Complete()
		}
		err := repo.Save(ctx, job)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	}

	t.Run("Find latest job", func(t *testing.T) {
		found, err := repo.FindLatest(ctx, channelID, domain.SyncTypeOrders)
		assert.NoError(t, err)
		require.NotNil(t, found)
		// Should be the most recent job
		assert.NotNil(t, found.StartedAt)
	})

	t.Run("No jobs for inventory sync", func(t *testing.T) {
		found, err := repo.FindLatest(ctx, channelID, domain.SyncTypeInventory)
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}
