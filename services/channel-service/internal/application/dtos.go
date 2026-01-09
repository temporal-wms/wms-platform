package application

import (
	"fmt"
	"time"

	"github.com/wms-platform/services/channel-service/internal/domain"
)

// ConnectChannelCommand represents a command to connect a channel
type ConnectChannelCommand struct {
	TenantID    string                    `json:"tenantId" binding:"required"`
	SellerID    string                    `json:"sellerId" binding:"required"`
	Type        string                    `json:"type" binding:"required"`
	Name        string                    `json:"name" binding:"required"`
	Credentials domain.ChannelCredentials `json:"credentials" binding:"required"`
	WebhookURL  string                    `json:"webhookUrl"`
}

// UpdateChannelCommand represents a command to update channel settings
type UpdateChannelCommand struct {
	Name         string                 `json:"name"`
	SyncSettings *domain.SyncSettings   `json:"syncSettings"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// GetSyncSettingsValue returns the sync settings value (not pointer)
func (c *UpdateChannelCommand) GetSyncSettingsValue() domain.SyncSettings {
	if c.SyncSettings != nil {
		return *c.SyncSettings
	}
	return domain.SyncSettings{}
}

// SyncOrdersCommand represents a command to sync orders
type SyncOrdersCommand struct {
	ChannelID string    `json:"channelId" binding:"required"`
	Since     time.Time `json:"since"`
}

// SyncInventoryCommand represents a command to sync inventory
type SyncInventoryCommand struct {
	ChannelID string                 `json:"channelId" binding:"required"`
	Items     []domain.InventoryUpdate `json:"items" binding:"required"`
}

// PushTrackingCommand represents a command to push tracking info
type PushTrackingCommand struct {
	ChannelID       string `json:"channelId" binding:"required"`
	ExternalOrderID string `json:"externalOrderId" binding:"required"`
	TrackingNumber  string `json:"trackingNumber" binding:"required"`
	Carrier         string `json:"carrier" binding:"required"`
	TrackingURL     string `json:"trackingUrl"`
	NotifyCustomer  bool   `json:"notifyCustomer"`
}

// CreateFulfillmentCommand represents a command to create fulfillment
type CreateFulfillmentCommand struct {
	ChannelID       string                      `json:"channelId" binding:"required"`
	ExternalOrderID string                      `json:"externalOrderId" binding:"required"`
	LocationID      string                      `json:"locationId"`
	TrackingNumber  string                      `json:"trackingNumber" binding:"required"`
	TrackingURL     string                      `json:"trackingUrl"`
	Carrier         string                      `json:"carrier" binding:"required"`
	LineItems       []domain.FulfillmentLineItem `json:"lineItems"`
	NotifyCustomer  bool                        `json:"notifyCustomer"`
}

// ImportOrderCommand represents a command to import an order to WMS
type ImportOrderCommand struct {
	ChannelID       string `json:"channelId" binding:"required"`
	ExternalOrderID string `json:"externalOrderId" binding:"required"`
	WMSOrderID      string `json:"wmsOrderId" binding:"required"`
}

// WebhookCommand represents a webhook payload
type WebhookCommand struct {
	ChannelID string `json:"channelId"`
	Topic     string `json:"topic"`
	Signature string `json:"signature"`
	Body      []byte `json:"body"`
}

// ChannelDTO represents a channel response
type ChannelDTO struct {
	ID           string                 `json:"id"`
	SellerID     string                 `json:"sellerId"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Status       string                 `json:"status"`
	SyncSettings *SyncSettingsDTO       `json:"syncSettings"`
	Stats        *ChannelStatsDTO       `json:"stats"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	ConnectedAt  time.Time              `json:"connectedAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// SyncSettingsDTO represents sync settings response
type SyncSettingsDTO struct {
	OrderSync     *SyncConfigDTO `json:"orderSync"`
	InventorySync *SyncConfigDTO `json:"inventorySync"`
	ProductSync   *SyncConfigDTO `json:"productSync"`
}

// SyncConfigDTO represents a sync config response
type SyncConfigDTO struct {
	Enabled    bool       `json:"enabled"`
	Interval   string     `json:"interval"`
	LastSyncAt *time.Time `json:"lastSyncAt,omitempty"`
	LastError  string     `json:"lastError,omitempty"`
}

// ChannelStatsDTO represents channel stats response
type ChannelStatsDTO struct {
	TotalOrders       int64     `json:"totalOrders"`
	ImportedOrders    int64     `json:"importedOrders"`
	PendingOrders     int64     `json:"pendingOrders"`
	LastOrderDate     time.Time `json:"lastOrderDate,omitempty"`
	TrackingPushCount int64     `json:"trackingPushCount"`
}

// ChannelOrderDTO represents a channel order response
type ChannelOrderDTO struct {
	ID              string           `json:"id"`
	ChannelID       string           `json:"channelId"`
	ExternalOrderID string           `json:"externalOrderId"`
	ExternalNumber  string           `json:"externalNumber"`
	Status          string           `json:"status"`
	Customer        CustomerDTO      `json:"customer"`
	ShippingAddress AddressDTO       `json:"shippingAddress"`
	LineItems       []LineItemDTO    `json:"lineItems"`
	TotalAmount     float64          `json:"totalAmount"`
	Currency        string           `json:"currency"`
	ImportedToWMS   bool             `json:"importedToWms"`
	WMSOrderID      string           `json:"wmsOrderId,omitempty"`
	TrackingPushed  bool             `json:"trackingPushed"`
	OrderDate       time.Time        `json:"orderDate"`
	ImportedAt      *time.Time       `json:"importedAt,omitempty"`
}

// CustomerDTO represents customer info
type CustomerDTO struct {
	ExternalID string `json:"externalId"`
	Email      string `json:"email"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Phone      string `json:"phone,omitempty"`
}

// AddressDTO represents an address
type AddressDTO struct {
	Name       string `json:"name"`
	Address1   string `json:"address1"`
	Address2   string `json:"address2,omitempty"`
	City       string `json:"city"`
	Province   string `json:"province"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
	Phone      string `json:"phone,omitempty"`
}

// LineItemDTO represents an order line item
type LineItemDTO struct {
	ExternalID  string  `json:"externalId"`
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	TotalPrice  float64 `json:"totalPrice"`
	VariantID   string  `json:"variantId,omitempty"`
	ProductID   string  `json:"productId,omitempty"`
}

// SyncJobDTO represents a sync job response
type SyncJobDTO struct {
	ID           string     `json:"id"`
	ChannelID    string     `json:"channelId"`
	Type         string     `json:"type"`
	Status       string     `json:"status"`
	TotalItems   int        `json:"totalItems"`
	ProcessedItems int      `json:"processedItems"`
	FailedItems  int        `json:"failedItems"`
	StartedAt    time.Time  `json:"startedAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	Error        string     `json:"error,omitempty"`
}

// ToChannelDTO converts a domain Channel to DTO
func ToChannelDTO(channel *domain.Channel) *ChannelDTO {
	dto := &ChannelDTO{
		ID:          channel.ChannelID,
		SellerID:    channel.SellerID,
		Type:        string(channel.Type),
		Name:        channel.Name,
		Status:      string(channel.Status),
		Metadata:    channel.Metadata,
		ConnectedAt: channel.CreatedAt,
		UpdatedAt:   channel.UpdatedAt,
	}

	// Convert sync settings to DTO format
	dto.SyncSettings = &SyncSettingsDTO{
		OrderSync: &SyncConfigDTO{
			Enabled:    channel.SyncSettings.AutoImportOrders,
			Interval:   fmt.Sprintf("%dm", channel.SyncSettings.OrderSyncIntervalMin),
			LastSyncAt: channel.LastOrderSync,
			LastError:  channel.LastError,
		},
		InventorySync: &SyncConfigDTO{
			Enabled:    channel.SyncSettings.AutoSyncInventory,
			Interval:   fmt.Sprintf("%dm", channel.SyncSettings.InventorySyncIntervalMin),
			LastSyncAt: channel.LastInventorySync,
		},
	}

	return dto
}

// ToChannelOrderDTO converts a domain ChannelOrder to DTO
func ToChannelOrderDTO(order *domain.ChannelOrder) *ChannelOrderDTO {
	dto := &ChannelOrderDTO{
		ID:              order.ID.Hex(),
		ChannelID:       order.ChannelID,
		ExternalOrderID: order.ExternalOrderID,
		ExternalNumber:  order.ExternalOrderNumber,
		Status:          order.FulfillmentStatus,
		TotalAmount:     order.Total,
		Currency:        order.Currency,
		ImportedToWMS:   order.Imported,
		WMSOrderID:      order.WMSOrderID,
		TrackingPushed:  order.TrackingPushed,
		OrderDate:       order.ExternalCreatedAt,
	}

	if order.ImportedAt != nil {
		dto.ImportedAt = order.ImportedAt
	}

	dto.Customer = CustomerDTO{
		ExternalID: order.Customer.ExternalID,
		Email:      order.Customer.Email,
		FirstName:  order.Customer.FirstName,
		LastName:   order.Customer.LastName,
		Phone:      order.Customer.Phone,
	}

	dto.ShippingAddress = AddressDTO{
		Name:       fmt.Sprintf("%s %s", order.ShippingAddr.FirstName, order.ShippingAddr.LastName),
		Address1:   order.ShippingAddr.Address1,
		Address2:   order.ShippingAddr.Address2,
		City:       order.ShippingAddr.City,
		Province:   order.ShippingAddr.Province,
		PostalCode: order.ShippingAddr.Zip,
		Country:    order.ShippingAddr.Country,
		Phone:      order.ShippingAddr.Phone,
	}

	dto.LineItems = make([]LineItemDTO, len(order.LineItems))
	for i, item := range order.LineItems {
		dto.LineItems[i] = LineItemDTO{
			ExternalID: item.ExternalID,
			SKU:        item.SKU,
			Name:       item.Title,
			Quantity:   item.Quantity,
			UnitPrice:  item.Price,
			TotalPrice: item.Price * float64(item.Quantity),
			VariantID:  item.VariantID,
			ProductID:  item.ProductID,
		}
	}

	return dto
}

// ToSyncJobDTO converts a domain SyncJob to DTO
func ToSyncJobDTO(job *domain.SyncJob) *SyncJobDTO {
	dto := &SyncJobDTO{
		ID:             job.JobID,
		ChannelID:      job.ChannelID,
		Type:           string(job.Type),
		Status:         string(job.Status),
		TotalItems:     job.TotalItems,
		ProcessedItems: job.ProcessedItems,
		FailedItems:    job.FailedItems,
	}

	if job.StartedAt != nil {
		dto.StartedAt = *job.StartedAt
	}

	if job.CompletedAt != nil {
		dto.CompletedAt = job.CompletedAt
	}

	if len(job.Errors) > 0 {
		dto.Error = job.Errors[len(job.Errors)-1].Error
	}

	return dto
}
