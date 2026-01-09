package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewChannel tests channel creation
func TestNewChannel(t *testing.T) {
	tests := []struct {
		name         string
		tenantID     string
		sellerID     string
		channelType  ChannelType
		channelName  string
		storeURL     string
		expectError  error
	}{
		{
			name:        "Valid Shopify channel",
			tenantID:    "TNT-001",
			sellerID:    "SLR-001",
			channelType: ChannelTypeShopify,
			channelName: "My Shopify Store",
			storeURL:    "https://mystore.myshopify.com",
			expectError: nil,
		},
		{
			name:        "Valid Amazon channel",
			tenantID:    "TNT-001",
			sellerID:    "SLR-001",
			channelType: ChannelTypeAmazon,
			channelName: "Amazon US",
			storeURL:    "",
			expectError: nil,
		},
		{
			name:        "Invalid channel type",
			tenantID:    "TNT-001",
			sellerID:    "SLR-001",
			channelType: ChannelType("invalid"),
			channelName: "Invalid Channel",
			storeURL:    "",
			expectError: ErrInvalidChannelType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := NewChannel(
				tt.tenantID, tt.sellerID,
				tt.channelType, tt.channelName, tt.storeURL,
				ChannelCredentials{}, SyncSettings{},
			)

			if tt.expectError != nil {
				assert.Equal(t, tt.expectError, err)
				assert.Nil(t, channel)
			} else {
				require.NoError(t, err)
				require.NotNil(t, channel)
				assert.NotEmpty(t, channel.ChannelID)
				assert.Equal(t, tt.tenantID, channel.TenantID)
				assert.Equal(t, tt.sellerID, channel.SellerID)
				assert.Equal(t, tt.channelType, channel.Type)
				assert.Equal(t, tt.channelName, channel.Name)
				assert.Equal(t, tt.storeURL, channel.StoreURL)
				assert.Equal(t, ChannelStatusActive, channel.Status)
				assert.Equal(t, 0, channel.ErrorCount)
				assert.NotZero(t, channel.CreatedAt)

				// Should have domain event
				events := channel.DomainEvents()
				assert.Len(t, events, 1)
			}
		})
	}
}

// TestChannelTypeIsValid tests channel type validation
func TestChannelTypeIsValid(t *testing.T) {
	validTypes := []ChannelType{
		ChannelTypeShopify,
		ChannelTypeAmazon,
		ChannelTypeEbay,
		ChannelTypeWooCommerce,
		ChannelTypeCustom,
	}

	for _, ct := range validTypes {
		assert.True(t, ct.IsValid(), "Expected %s to be valid", ct)
	}

	assert.False(t, ChannelType("invalid").IsValid())
}

// TestChannelPause tests pausing a channel
func TestChannelPause(t *testing.T) {
	channel := createTestChannel()
	assert.Equal(t, ChannelStatusActive, channel.Status)

	channel.Pause()
	assert.Equal(t, ChannelStatusPaused, channel.Status)
}

// TestChannelResume tests resuming a channel
func TestChannelResume(t *testing.T) {
	tests := []struct {
		name        string
		setupStatus ChannelStatus
		expectError bool
	}{
		{
			name:        "Resume paused channel",
			setupStatus: ChannelStatusPaused,
			expectError: false,
		},
		{
			name:        "Resume active channel",
			setupStatus: ChannelStatusActive,
			expectError: false,
		},
		{
			name:        "Cannot resume disconnected channel",
			setupStatus: ChannelStatusDisconnected,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := createTestChannel()
			channel.Status = tt.setupStatus

			err := channel.Resume()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, ChannelStatusActive, channel.Status)
			}
		})
	}
}

// TestChannelDisconnect tests disconnecting a channel
func TestChannelDisconnect(t *testing.T) {
	channel := createTestChannel()
	channel.ClearDomainEvents()

	channel.Disconnect()
	assert.Equal(t, ChannelStatusDisconnected, channel.Status)
	assert.Len(t, channel.DomainEvents(), 1)
}

// TestChannelRecordError tests error recording
func TestChannelRecordError(t *testing.T) {
	channel := createTestChannel()
	assert.Equal(t, 0, channel.ErrorCount)
	assert.Empty(t, channel.LastError)

	channel.RecordError("Connection failed")
	assert.Equal(t, 1, channel.ErrorCount)
	assert.Equal(t, "Connection failed", channel.LastError)
	assert.NotNil(t, channel.LastErrorAt)
	assert.Equal(t, ChannelStatusActive, channel.Status)

	// Record more errors to trigger error status
	for i := 0; i < 4; i++ {
		channel.RecordError("Error")
	}
	assert.Equal(t, 5, channel.ErrorCount)
	assert.Equal(t, ChannelStatusError, channel.Status)
}

// TestChannelClearErrors tests clearing errors
func TestChannelClearErrors(t *testing.T) {
	channel := createTestChannel()
	for i := 0; i < 5; i++ {
		channel.RecordError("Error")
	}
	assert.Equal(t, ChannelStatusError, channel.Status)

	channel.ClearErrors()
	assert.Equal(t, 0, channel.ErrorCount)
	assert.Empty(t, channel.LastError)
	assert.Nil(t, channel.LastErrorAt)
	assert.Equal(t, ChannelStatusActive, channel.Status)
}

// TestChannelUpdateLastSync tests updating sync timestamps
func TestChannelUpdateLastSync(t *testing.T) {
	channel := createTestChannel()

	assert.Nil(t, channel.LastOrderSync)
	assert.Nil(t, channel.LastInventorySync)
	assert.Nil(t, channel.LastTrackingSync)

	channel.UpdateLastSync(SyncTypeOrders)
	assert.NotNil(t, channel.LastOrderSync)
	assert.Nil(t, channel.LastInventorySync)

	channel.UpdateLastSync(SyncTypeInventory)
	assert.NotNil(t, channel.LastInventorySync)

	channel.UpdateLastSync(SyncTypeTracking)
	assert.NotNil(t, channel.LastTrackingSync)
}

// TestChannelUpdateCredentials tests updating credentials
func TestChannelUpdateCredentials(t *testing.T) {
	channel := createTestChannel()

	newCreds := ChannelCredentials{
		AccessToken: "new-token",
		StoreDomain: "newstore.myshopify.com",
	}

	channel.UpdateCredentials(newCreds)
	assert.Equal(t, "new-token", channel.Credentials.AccessToken)
	assert.Equal(t, "newstore.myshopify.com", channel.Credentials.StoreDomain)
}

// TestChannelUpdateSyncSettings tests updating sync settings
func TestChannelUpdateSyncSettings(t *testing.T) {
	channel := createTestChannel()

	newSettings := SyncSettings{
		AutoImportOrders:  true,
		AutoSyncInventory: true,
		AutoPushTracking:  true,
		OrderSyncIntervalMin: 10,
		InventorySyncIntervalMin: 30,
	}

	channel.UpdateSyncSettings(newSettings)
	assert.True(t, channel.SyncSettings.AutoImportOrders)
	assert.True(t, channel.SyncSettings.AutoSyncInventory)
	assert.Equal(t, 10, channel.SyncSettings.OrderSyncIntervalMin)
}

// TestChannelIsActive tests active status check
func TestChannelIsActive(t *testing.T) {
	channel := createTestChannel()
	assert.True(t, channel.IsActive())

	channel.Pause()
	assert.False(t, channel.IsActive())

	channel.Resume()
	assert.True(t, channel.IsActive())

	channel.Disconnect()
	assert.False(t, channel.IsActive())
}

// TestChannelDomainEvents tests domain event handling
func TestChannelDomainEvents(t *testing.T) {
	channel := createTestChannel()

	events := channel.DomainEvents()
	assert.Len(t, events, 1)

	channel.ClearDomainEvents()
	events = channel.DomainEvents()
	assert.Empty(t, events)
}

// TestChannelOrderMarkImported tests marking order as imported
func TestChannelOrderMarkImported(t *testing.T) {
	order := createTestChannelOrder()
	assert.False(t, order.Imported)
	assert.Empty(t, order.WMSOrderID)

	order.MarkImported("ORD-001")
	assert.True(t, order.Imported)
	assert.Equal(t, "ORD-001", order.WMSOrderID)
	assert.NotNil(t, order.ImportedAt)
}

// TestChannelOrderMarkTrackingPushed tests marking tracking as pushed
func TestChannelOrderMarkTrackingPushed(t *testing.T) {
	order := createTestChannelOrder()
	assert.False(t, order.TrackingPushed)

	order.MarkTrackingPushed()
	assert.True(t, order.TrackingPushed)
	assert.NotNil(t, order.TrackingPushedAt)
}

// TestNewSyncJob tests sync job creation
func TestNewSyncJob(t *testing.T) {
	job := NewSyncJob("TNT-001", "SLR-001", "CH-001", SyncTypeOrders, "inbound")

	require.NotNil(t, job)
	assert.NotEmpty(t, job.JobID)
	assert.Equal(t, "TNT-001", job.TenantID)
	assert.Equal(t, "SLR-001", job.SellerID)
	assert.Equal(t, "CH-001", job.ChannelID)
	assert.Equal(t, SyncTypeOrders, job.Type)
	assert.Equal(t, "inbound", job.Direction)
	assert.Equal(t, SyncStatusPending, job.Status)
	assert.Empty(t, job.Errors)
	assert.NotZero(t, job.CreatedAt)
}

// TestSyncJobStart tests starting a sync job
func TestSyncJobStart(t *testing.T) {
	job := NewSyncJob("TNT-001", "SLR-001", "CH-001", SyncTypeOrders, "inbound")
	assert.Nil(t, job.StartedAt)

	job.Start()
	assert.Equal(t, SyncStatusRunning, job.Status)
	assert.NotNil(t, job.StartedAt)
}

// TestSyncJobComplete tests completing a sync job
func TestSyncJobComplete(t *testing.T) {
	tests := []struct {
		name           string
		setupJob       func() *SyncJob
		expectedStatus SyncStatus
	}{
		{
			name: "Complete with success",
			setupJob: func() *SyncJob {
				job := NewSyncJob("TNT-001", "SLR-001", "CH-001", SyncTypeOrders, "inbound")
				job.Start()
				job.SuccessItems = 10
				return job
			},
			expectedStatus: SyncStatusCompleted,
		},
		{
			name: "Complete with all failures",
			setupJob: func() *SyncJob {
				job := NewSyncJob("TNT-001", "SLR-001", "CH-001", SyncTypeOrders, "inbound")
				job.Start()
				job.AddError("item1", "Error 1")
				job.AddError("item2", "Error 2")
				return job
			},
			expectedStatus: SyncStatusFailed,
		},
		{
			name: "Complete with partial success",
			setupJob: func() *SyncJob {
				job := NewSyncJob("TNT-001", "SLR-001", "CH-001", SyncTypeOrders, "inbound")
				job.Start()
				job.SuccessItems = 5
				job.AddError("item1", "Error")
				return job
			},
			expectedStatus: SyncStatusCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := tt.setupJob()
			job.Complete()
			assert.Equal(t, tt.expectedStatus, job.Status)
			assert.NotNil(t, job.CompletedAt)
		})
	}
}

// TestSyncJobFail tests failing a sync job
func TestSyncJobFail(t *testing.T) {
	job := NewSyncJob("TNT-001", "SLR-001", "CH-001", SyncTypeOrders, "inbound")
	job.Start()

	job.Fail("Connection timeout")
	assert.Equal(t, SyncStatusFailed, job.Status)
	assert.NotNil(t, job.CompletedAt)
	assert.Len(t, job.Errors, 1)
	assert.Equal(t, "Connection timeout", job.Errors[0].Error)
}

// TestSyncJobAddError tests adding errors to sync job
func TestSyncJobAddError(t *testing.T) {
	job := NewSyncJob("TNT-001", "SLR-001", "CH-001", SyncTypeOrders, "inbound")
	job.Start()
	assert.Equal(t, 0, job.FailedItems)

	job.AddError("order-123", "Failed to import")
	assert.Len(t, job.Errors, 1)
	assert.Equal(t, "order-123", job.Errors[0].ItemID)
	assert.Equal(t, "Failed to import", job.Errors[0].Error)
	assert.Equal(t, 1, job.FailedItems)
}

// TestSyncJobIncrementProgress tests progress tracking
func TestSyncJobIncrementProgress(t *testing.T) {
	job := NewSyncJob("TNT-001", "SLR-001", "CH-001", SyncTypeOrders, "inbound")
	job.Start()
	job.SetTotalItems(10)

	job.IncrementProgress(true)
	assert.Equal(t, 1, job.ProcessedItems)
	assert.Equal(t, 1, job.SuccessItems)
	assert.Equal(t, 0, job.FailedItems)

	job.IncrementProgress(false)
	assert.Equal(t, 2, job.ProcessedItems)
	assert.Equal(t, 1, job.SuccessItems)
	assert.Equal(t, 1, job.FailedItems)
}

// TestSyncJobSetTotalItems tests setting total items
func TestSyncJobSetTotalItems(t *testing.T) {
	job := NewSyncJob("TNT-001", "SLR-001", "CH-001", SyncTypeOrders, "inbound")
	assert.Equal(t, 0, job.TotalItems)

	job.SetTotalItems(100)
	assert.Equal(t, 100, job.TotalItems)
}

// Helper functions
func createTestChannel() *Channel {
	channel, _ := NewChannel(
		"TNT-001", "SLR-001",
		ChannelTypeShopify, "Test Store", "https://test.myshopify.com",
		ChannelCredentials{
			AccessToken: "test-token",
			StoreDomain: "test.myshopify.com",
		},
		SyncSettings{
			AutoImportOrders:  true,
			AutoSyncInventory: false,
		},
	)
	return channel
}

func createTestChannelOrder() *ChannelOrder {
	return &ChannelOrder{
		TenantID:            "TNT-001",
		SellerID:            "SLR-001",
		ChannelID:           "CH-001",
		ExternalOrderID:     "12345",
		ExternalOrderNumber: "#1001",
		ExternalCreatedAt:   time.Now().UTC(),
		Customer: ChannelCustomer{
			Email:     "customer@example.com",
			FirstName: "John",
			LastName:  "Doe",
		},
		Currency:          "USD",
		Total:             100.00,
		FinancialStatus:   "paid",
		FulfillmentStatus: "unfulfilled",
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
}

// BenchmarkNewChannel benchmarks channel creation
func BenchmarkNewChannel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewChannel(
			"TNT-001", "SLR-001",
			ChannelTypeShopify, "Test Store", "https://test.myshopify.com",
			ChannelCredentials{}, SyncSettings{},
		)
	}
}

// BenchmarkNewSyncJob benchmarks sync job creation
func BenchmarkNewSyncJob(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewSyncJob("TNT-001", "SLR-001", "CH-001", SyncTypeOrders, "inbound")
	}
}

// BenchmarkSyncJobAddError benchmarks error adding
func BenchmarkSyncJobAddError(b *testing.B) {
	job := NewSyncJob("TNT-001", "SLR-001", "CH-001", SyncTypeOrders, "inbound")
	job.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job.AddError("item", "error")
	}
}
