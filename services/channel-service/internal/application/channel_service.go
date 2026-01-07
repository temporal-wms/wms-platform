package application

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wms-platform/services/channel-service/internal/domain"
)

// ChannelService handles channel operations
type ChannelService struct {
	channelRepo      domain.ChannelRepository
	orderRepo        domain.ChannelOrderRepository
	syncJobRepo      domain.SyncJobRepository
	adapterFactory   *domain.AdapterFactory
}

// NewChannelService creates a new channel service
func NewChannelService(
	channelRepo domain.ChannelRepository,
	orderRepo domain.ChannelOrderRepository,
	syncJobRepo domain.SyncJobRepository,
	adapterFactory *domain.AdapterFactory,
) *ChannelService {
	return &ChannelService{
		channelRepo:    channelRepo,
		orderRepo:      orderRepo,
		syncJobRepo:    syncJobRepo,
		adapterFactory: adapterFactory,
	}
}

// ConnectChannel connects a new sales channel
func (s *ChannelService) ConnectChannel(ctx context.Context, cmd ConnectChannelCommand) (*ChannelDTO, error) {
	channelType := domain.ChannelType(cmd.Type)

	// Get the adapter for this channel type
	adapter, err := s.adapterFactory.GetAdapter(channelType)
	if err != nil {
		return nil, fmt.Errorf("unsupported channel type: %s", cmd.Type)
	}

	// Validate credentials with the channel
	if err := adapter.ValidateCredentials(ctx, cmd.Credentials); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	// Create default sync settings
	syncSettings := domain.SyncSettings{
		AutoImportOrders:  true,
		AutoSyncInventory: true,
		AutoPushTracking:  true,
		OrderSyncIntervalMin: 15,
		InventorySyncIntervalMin: 30,
		ImportPaidOnly: true,
	}

	// Create the channel with all required parameters
	channel, err := domain.NewChannel(cmd.TenantID, cmd.SellerID, channelType, cmd.Name, "", cmd.Credentials, syncSettings)
	if err != nil {
		return nil, err
	}

	// Register webhooks if URL provided
	if cmd.WebhookURL != "" {
		if err := adapter.RegisterWebhooks(ctx, channel, cmd.WebhookURL); err != nil {
			log.Printf("Warning: failed to register webhooks: %v", err)
			// Don't fail the connection, just log the warning
		}
	}

	// Save the channel
	if err := s.channelRepo.Save(ctx, channel); err != nil {
		return nil, err
	}

	return ToChannelDTO(channel), nil
}

// GetChannel retrieves a channel by ID
func (s *ChannelService) GetChannel(ctx context.Context, channelID string) (*ChannelDTO, error) {
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return ToChannelDTO(channel), nil
}

// GetChannelsBySeller retrieves all channels for a seller
func (s *ChannelService) GetChannelsBySeller(ctx context.Context, sellerID string) ([]*ChannelDTO, error) {
	channels, err := s.channelRepo.FindBySellerID(ctx, sellerID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*ChannelDTO, len(channels))
	for i, channel := range channels {
		dtos[i] = ToChannelDTO(channel)
	}
	return dtos, nil
}

// UpdateChannel updates channel settings
func (s *ChannelService) UpdateChannel(ctx context.Context, channelID string, cmd UpdateChannelCommand) (*ChannelDTO, error) {
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	if cmd.Name != "" {
		channel.Name = cmd.Name
	}

	if cmd.SyncSettings != nil {
		channel.SyncSettings = *cmd.SyncSettings
	}

	if cmd.Metadata != nil {
		channel.Metadata = cmd.Metadata
	}

	channel.UpdatedAt = time.Now()

	if err := s.channelRepo.Save(ctx, channel); err != nil {
		return nil, err
	}

	return ToChannelDTO(channel), nil
}

// DisconnectChannel disconnects a channel
func (s *ChannelService) DisconnectChannel(ctx context.Context, channelID string) error {
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return err
	}

	channel.Disconnect()

	return s.channelRepo.Save(ctx, channel)
}

// SyncOrders fetches and syncs orders from a channel
func (s *ChannelService) SyncOrders(ctx context.Context, cmd SyncOrdersCommand) (*SyncJobDTO, error) {
	channel, err := s.channelRepo.FindByID(ctx, cmd.ChannelID)
	if err != nil {
		return nil, err
	}

	// Check for existing running job
	existingJob, err := s.syncJobRepo.FindRunning(ctx, cmd.ChannelID, domain.SyncTypeOrders)
	if err != nil {
		return nil, err
	}
	if existingJob != nil {
		return nil, fmt.Errorf("order sync already in progress")
	}

	// Get adapter
	adapter, err := s.adapterFactory.GetAdapterForChannel(channel)
	if err != nil {
		return nil, err
	}

	// Create sync job
	job := domain.NewSyncJob(channel.TenantID, channel.SellerID, channel.ChannelID, domain.SyncTypeOrders, "inbound")
	if err := s.syncJobRepo.Save(ctx, job); err != nil {
		return nil, err
	}

	// Determine since time
	since := cmd.Since
	if since.IsZero() && channel.LastOrderSync != nil {
		since = *channel.LastOrderSync
	}
	if since.IsZero() {
		since = time.Now().Add(-7 * 24 * time.Hour) // Default to last 7 days
	}

	// Fetch orders
	orders, err := adapter.FetchOrders(ctx, channel, since)
	if err != nil {
		job.Fail(err.Error())
		s.syncJobRepo.Save(ctx, job)
		return ToSyncJobDTO(job), fmt.Errorf("failed to fetch orders: %w", err)
	}

	job.TotalItems = len(orders)

	// Save orders
	var newOrders []*domain.ChannelOrder
	for _, order := range orders {
		// Check if order already exists
		existing, _ := s.orderRepo.FindByExternalID(ctx, channel.ChannelID, order.ExternalOrderID)
		if existing == nil {
			newOrders = append(newOrders, order)
		}
		job.ProcessedItems++
	}

	if len(newOrders) > 0 {
		if err := s.orderRepo.SaveAll(ctx, newOrders); err != nil {
			job.Fail(err.Error())
			s.syncJobRepo.Save(ctx, job)
			return ToSyncJobDTO(job), fmt.Errorf("failed to save orders: %w", err)
		}
	}

	// Complete job
	job.Complete()
	s.syncJobRepo.Save(ctx, job)

	// Update channel sync status
	channel.UpdateLastSync(domain.SyncTypeOrders)
	s.channelRepo.Save(ctx, channel)

	return ToSyncJobDTO(job), nil
}

// SyncInventory pushes inventory levels to a channel
func (s *ChannelService) SyncInventory(ctx context.Context, cmd SyncInventoryCommand) (*SyncJobDTO, error) {
	channel, err := s.channelRepo.FindByID(ctx, cmd.ChannelID)
	if err != nil {
		return nil, err
	}

	adapter, err := s.adapterFactory.GetAdapterForChannel(channel)
	if err != nil {
		return nil, err
	}

	// Create sync job
	job := domain.NewSyncJob(channel.TenantID, channel.SellerID, channel.ChannelID, domain.SyncTypeInventory, "outbound")
	job.TotalItems = len(cmd.Items)
	if err := s.syncJobRepo.Save(ctx, job); err != nil {
		return nil, err
	}

	// Sync inventory
	err = adapter.SyncInventory(ctx, channel, cmd.Items)
	if err != nil {
		job.Fail(err.Error())
		s.syncJobRepo.Save(ctx, job)
		return ToSyncJobDTO(job), fmt.Errorf("failed to sync inventory: %w", err)
	}

	job.ProcessedItems = len(cmd.Items)
	job.Complete()
	s.syncJobRepo.Save(ctx, job)

	// Update channel sync status
	channel.UpdateLastSync(domain.SyncTypeInventory)
	s.channelRepo.Save(ctx, channel)

	return ToSyncJobDTO(job), nil
}

// PushTracking pushes tracking info to a channel
func (s *ChannelService) PushTracking(ctx context.Context, cmd PushTrackingCommand) error {
	channel, err := s.channelRepo.FindByID(ctx, cmd.ChannelID)
	if err != nil {
		return err
	}

	adapter, err := s.adapterFactory.GetAdapterForChannel(channel)
	if err != nil {
		return err
	}

	tracking := domain.TrackingInfo{
		TrackingNumber: cmd.TrackingNumber,
		Carrier:        cmd.Carrier,
		TrackingURL:    cmd.TrackingURL,
		NotifyCustomer: cmd.NotifyCustomer,
	}

	err = adapter.PushTracking(ctx, channel, cmd.ExternalOrderID, tracking)
	if err != nil {
		return fmt.Errorf("failed to push tracking: %w", err)
	}

	// Mark tracking as pushed
	if err := s.orderRepo.MarkTrackingPushed(ctx, cmd.ExternalOrderID); err != nil {
		log.Printf("Warning: failed to mark tracking as pushed: %v", err)
	}

	return nil
}

// CreateFulfillment creates a fulfillment in the channel
func (s *ChannelService) CreateFulfillment(ctx context.Context, cmd CreateFulfillmentCommand) error {
	channel, err := s.channelRepo.FindByID(ctx, cmd.ChannelID)
	if err != nil {
		return err
	}

	adapter, err := s.adapterFactory.GetAdapterForChannel(channel)
	if err != nil {
		return err
	}

	fulfillment := domain.FulfillmentRequest{
		OrderID:        cmd.ExternalOrderID,
		LocationID:     cmd.LocationID,
		TrackingNumber: cmd.TrackingNumber,
		TrackingURL:    cmd.TrackingURL,
		Carrier:        cmd.Carrier,
		LineItems:      cmd.LineItems,
		NotifyCustomer: cmd.NotifyCustomer,
	}

	err = adapter.CreateFulfillment(ctx, channel, fulfillment)
	if err != nil {
		return fmt.Errorf("failed to create fulfillment: %w", err)
	}

	// Mark tracking as pushed
	if err := s.orderRepo.MarkTrackingPushed(ctx, cmd.ExternalOrderID); err != nil {
		log.Printf("Warning: failed to mark tracking as pushed: %v", err)
	}

	return nil
}

// ImportOrder marks an order as imported to WMS
func (s *ChannelService) ImportOrder(ctx context.Context, cmd ImportOrderCommand) error {
	return s.orderRepo.MarkImported(ctx, cmd.ExternalOrderID, cmd.WMSOrderID)
}

// GetChannelOrders retrieves orders for a channel
func (s *ChannelService) GetChannelOrders(ctx context.Context, channelID string, page, pageSize int64) ([]*ChannelOrderDTO, error) {
	pagination := domain.Pagination{Page: page, PageSize: pageSize}
	if page <= 0 {
		pagination.Page = 1
	}
	if pageSize <= 0 {
		pagination.PageSize = 20
	}

	orders, err := s.orderRepo.FindByChannelID(ctx, channelID, pagination)
	if err != nil {
		return nil, err
	}

	dtos := make([]*ChannelOrderDTO, len(orders))
	for i, order := range orders {
		dtos[i] = ToChannelOrderDTO(order)
	}
	return dtos, nil
}

// GetUnimportedOrders retrieves orders not yet imported to WMS
func (s *ChannelService) GetUnimportedOrders(ctx context.Context, channelID string) ([]*ChannelOrderDTO, error) {
	orders, err := s.orderRepo.FindUnimported(ctx, channelID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*ChannelOrderDTO, len(orders))
	for i, order := range orders {
		dtos[i] = ToChannelOrderDTO(order)
	}
	return dtos, nil
}

// GetSyncJobs retrieves sync jobs for a channel
func (s *ChannelService) GetSyncJobs(ctx context.Context, channelID string, page, pageSize int64) ([]*SyncJobDTO, error) {
	pagination := domain.Pagination{Page: page, PageSize: pageSize}
	if page <= 0 {
		pagination.Page = 1
	}
	if pageSize <= 0 {
		pagination.PageSize = 20
	}

	jobs, err := s.syncJobRepo.FindByChannelID(ctx, channelID, pagination)
	if err != nil {
		return nil, err
	}

	dtos := make([]*SyncJobDTO, len(jobs))
	for i, job := range jobs {
		dtos[i] = ToSyncJobDTO(job)
	}
	return dtos, nil
}

// HandleWebhook processes an incoming webhook
func (s *ChannelService) HandleWebhook(ctx context.Context, cmd WebhookCommand) error {
	channel, err := s.channelRepo.FindByID(ctx, cmd.ChannelID)
	if err != nil {
		return err
	}

	adapter, err := s.adapterFactory.GetAdapterForChannel(channel)
	if err != nil {
		return err
	}

	// Validate webhook signature
	if !adapter.ValidateWebhook(ctx, channel, cmd.Signature, cmd.Body) {
		return fmt.Errorf("invalid webhook signature")
	}

	// Process based on topic
	log.Printf("Received webhook for channel %s, topic: %s", channel.ID, cmd.Topic)

	// TODO: Process different webhook topics (orders/create, orders/updated, etc.)
	// This would typically trigger order sync or inventory update

	return nil
}

// GetInventoryLevels gets inventory levels from a channel
func (s *ChannelService) GetInventoryLevels(ctx context.Context, channelID string, skus []string) ([]domain.InventoryLevel, error) {
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	adapter, err := s.adapterFactory.GetAdapterForChannel(channel)
	if err != nil {
		return nil, err
	}

	return adapter.GetInventoryLevels(ctx, channel, skus)
}
