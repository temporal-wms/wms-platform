package domain

import (
	"context"
	"time"
)

// ChannelRepository defines the interface for channel persistence
type ChannelRepository interface {
	// Save persists a channel
	Save(ctx context.Context, channel *Channel) error

	// FindByID retrieves a channel by ID
	FindByID(ctx context.Context, channelID string) (*Channel, error)

	// FindBySellerID retrieves channels for a seller
	FindBySellerID(ctx context.Context, sellerID string) ([]*Channel, error)

	// FindByType retrieves channels by type
	FindByType(ctx context.Context, channelType ChannelType) ([]*Channel, error)

	// FindActiveChannels retrieves all active channels
	FindActiveChannels(ctx context.Context) ([]*Channel, error)

	// FindChannelsNeedingSync retrieves channels that need syncing
	FindChannelsNeedingSync(ctx context.Context, syncType SyncType, threshold time.Duration) ([]*Channel, error)

	// UpdateStatus updates channel status
	UpdateStatus(ctx context.Context, channelID string, status ChannelStatus) error

	// Delete deletes a channel
	Delete(ctx context.Context, channelID string) error
}

// ChannelOrderRepository defines the interface for channel order persistence
type ChannelOrderRepository interface {
	// Save persists a channel order
	Save(ctx context.Context, order *ChannelOrder) error

	// SaveAll persists multiple orders
	SaveAll(ctx context.Context, orders []*ChannelOrder) error

	// FindByExternalID retrieves an order by external ID
	FindByExternalID(ctx context.Context, channelID, externalOrderID string) (*ChannelOrder, error)

	// FindByChannelID retrieves orders for a channel
	FindByChannelID(ctx context.Context, channelID string, pagination Pagination) ([]*ChannelOrder, error)

	// FindUnimported retrieves orders not yet imported to WMS
	FindUnimported(ctx context.Context, channelID string) ([]*ChannelOrder, error)

	// FindWithoutTracking retrieves imported orders without tracking pushed
	FindWithoutTracking(ctx context.Context, channelID string) ([]*ChannelOrder, error)

	// MarkImported marks an order as imported
	MarkImported(ctx context.Context, externalOrderID, wmsOrderID string) error

	// MarkTrackingPushed marks tracking as pushed
	MarkTrackingPushed(ctx context.Context, externalOrderID string) error

	// Count returns count of orders for a channel
	Count(ctx context.Context, channelID string) (int64, error)
}

// SyncJobRepository defines the interface for sync job persistence
type SyncJobRepository interface {
	// Save persists a sync job
	Save(ctx context.Context, job *SyncJob) error

	// FindByID retrieves a job by ID
	FindByID(ctx context.Context, jobID string) (*SyncJob, error)

	// FindByChannelID retrieves jobs for a channel
	FindByChannelID(ctx context.Context, channelID string, pagination Pagination) ([]*SyncJob, error)

	// FindRunning retrieves running jobs for a channel
	FindRunning(ctx context.Context, channelID string, syncType SyncType) (*SyncJob, error)

	// FindLatest retrieves the latest job for a channel and type
	FindLatest(ctx context.Context, channelID string, syncType SyncType) (*SyncJob, error)
}

// Pagination represents pagination options
type Pagination struct {
	Page     int64
	PageSize int64
}

// DefaultPagination returns default pagination options
func DefaultPagination() Pagination {
	return Pagination{
		Page:     1,
		PageSize: 20,
	}
}

// Skip returns the number of documents to skip
func (p Pagination) Skip() int64 {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the maximum number of documents to return
func (p Pagination) Limit() int64 {
	return p.PageSize
}
