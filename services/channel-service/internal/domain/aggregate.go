package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors for Channel domain
var (
	ErrChannelNotActive     = errors.New("channel is not active")
	ErrChannelNotFound      = errors.New("channel not found")
	ErrInvalidChannelType   = errors.New("invalid channel type")
	ErrOrderAlreadyImported = errors.New("order already imported")
	ErrSyncInProgress       = errors.New("sync already in progress")
)

// ChannelType represents the type of sales channel
type ChannelType string

const (
	ChannelTypeShopify     ChannelType = "shopify"
	ChannelTypeAmazon      ChannelType = "amazon"
	ChannelTypeEbay        ChannelType = "ebay"
	ChannelTypeWooCommerce ChannelType = "woocommerce"
	ChannelTypeCustom      ChannelType = "custom"
)

// IsValid checks if the channel type is valid
func (c ChannelType) IsValid() bool {
	switch c {
	case ChannelTypeShopify, ChannelTypeAmazon, ChannelTypeEbay, ChannelTypeWooCommerce, ChannelTypeCustom:
		return true
	}
	return false
}

// ChannelStatus represents the status of a channel connection
type ChannelStatus string

const (
	ChannelStatusActive       ChannelStatus = "active"
	ChannelStatusPaused       ChannelStatus = "paused"
	ChannelStatusDisconnected ChannelStatus = "disconnected"
	ChannelStatusError        ChannelStatus = "error"
)

// SyncType represents what is being synced
type SyncType string

const (
	SyncTypeOrders    SyncType = "orders"
	SyncTypeInventory SyncType = "inventory"
	SyncTypeTracking  SyncType = "tracking"
	SyncTypeProducts  SyncType = "products"
)

// SyncStatus represents the status of a sync job
type SyncStatus string

const (
	SyncStatusPending    SyncStatus = "pending"
	SyncStatusRunning    SyncStatus = "running"
	SyncStatusCompleted  SyncStatus = "completed"
	SyncStatusFailed     SyncStatus = "failed"
	SyncStatusCancelled  SyncStatus = "cancelled"
)

// Channel represents an external sales channel connection
type Channel struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChannelID  string             `bson:"channelId" json:"channelId"`
	TenantID   string             `bson:"tenantId" json:"tenantId"`
	SellerID   string             `bson:"sellerId" json:"sellerId"`
	FacilityID string             `bson:"facilityId,omitempty" json:"facilityId,omitempty"`

	// Channel info
	Type      ChannelType   `bson:"type" json:"type"`
	Name      string        `bson:"name" json:"name"`
	StoreURL  string        `bson:"storeUrl,omitempty" json:"storeUrl,omitempty"`
	Status    ChannelStatus `bson:"status" json:"status"`

	// Credentials (encrypted)
	Credentials ChannelCredentials `bson:"credentials" json:"-"`

	// Sync settings
	SyncSettings SyncSettings `bson:"syncSettings" json:"syncSettings"`

	// Sync state
	LastOrderSync     *time.Time `bson:"lastOrderSync,omitempty" json:"lastOrderSync,omitempty"`
	LastInventorySync *time.Time `bson:"lastInventorySync,omitempty" json:"lastInventorySync,omitempty"`
	LastTrackingSync  *time.Time `bson:"lastTrackingSync,omitempty" json:"lastTrackingSync,omitempty"`

	// Error tracking
	LastError     string     `bson:"lastError,omitempty" json:"lastError,omitempty"`
	LastErrorAt   *time.Time `bson:"lastErrorAt,omitempty" json:"lastErrorAt,omitempty"`
	ErrorCount    int        `bson:"errorCount" json:"errorCount"`

	// Metadata
	ExternalStoreID string                 `bson:"externalStoreId,omitempty" json:"externalStoreId,omitempty"`
	Metadata        map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`

	// Domain events
	domainEvents []DomainEvent `bson:"-" json:"-"`
}

// ChannelCredentials holds encrypted credentials for a channel
type ChannelCredentials struct {
	// Shopify
	APIKey       string `bson:"apiKey,omitempty" json:"-"`
	APISecret    string `bson:"apiSecret,omitempty" json:"-"`
	AccessToken  string `bson:"accessToken,omitempty" json:"-"`
	StoreDomain  string `bson:"storeDomain,omitempty" json:"-"`

	// WooCommerce
	ShopURL string `bson:"shopUrl,omitempty" json:"-"` // Full URL (https://mystore.com)

	// Amazon
	MWSAuthToken   string `bson:"mwsAuthToken,omitempty" json:"-"`
	SellerID       string `bson:"sellerId,omitempty" json:"-"`
	MarketplaceID  string `bson:"marketplaceId,omitempty" json:"-"`

	// OAuth (generic)
	ClientID       string     `bson:"clientId,omitempty" json:"-"`
	ClientSecret   string     `bson:"clientSecret,omitempty" json:"-"`
	RefreshToken   string     `bson:"refreshToken,omitempty" json:"-"`
	TokenExpiresAt *time.Time `bson:"tokenExpiresAt,omitempty" json:"-"`

	// Webhook
	WebhookSecret string `bson:"webhookSecret,omitempty" json:"-"`

	// Additional config for channel-specific settings
	AdditionalConfig map[string]interface{} `bson:"additionalConfig,omitempty" json:"-"`
}

// SyncSettings defines how syncing works for a channel
type SyncSettings struct {
	AutoImportOrders     bool `bson:"autoImportOrders" json:"autoImportOrders"`
	AutoSyncInventory    bool `bson:"autoSyncInventory" json:"autoSyncInventory"`
	AutoPushTracking     bool `bson:"autoPushTracking" json:"autoPushTracking"`
	OrderSyncIntervalMin int  `bson:"orderSyncIntervalMin" json:"orderSyncIntervalMin"`
	InventorySyncIntervalMin int `bson:"inventorySyncIntervalMin" json:"inventorySyncIntervalMin"`

	// Fulfillment settings
	FulfillmentLocationID string `bson:"fulfillmentLocationId,omitempty" json:"fulfillmentLocationId,omitempty"`
	DefaultWarehouseID    string `bson:"defaultWarehouseId,omitempty" json:"defaultWarehouseId,omitempty"`

	// Order import filters
	ImportPaidOnly       bool     `bson:"importPaidOnly" json:"importPaidOnly"`
	ImportFulfilledOrders bool    `bson:"importFulfilledOrders" json:"importFulfilledOrders"`
	ExcludeTags          []string `bson:"excludeTags,omitempty" json:"excludeTags,omitempty"`
}

// NewChannel creates a new Channel
func NewChannel(
	tenantID, sellerID string,
	channelType ChannelType,
	name, storeURL string,
	credentials ChannelCredentials,
	syncSettings SyncSettings,
) (*Channel, error) {
	if !channelType.IsValid() {
		return nil, ErrInvalidChannelType
	}

	now := time.Now().UTC()
	channelID := fmt.Sprintf("CH-%s", uuid.New().String()[:8])

	channel := &Channel{
		ID:           primitive.NewObjectID(),
		ChannelID:    channelID,
		TenantID:     tenantID,
		SellerID:     sellerID,
		Type:         channelType,
		Name:         name,
		StoreURL:     storeURL,
		Status:       ChannelStatusActive,
		Credentials:  credentials,
		SyncSettings: syncSettings,
		ErrorCount:   0,
		CreatedAt:    now,
		UpdatedAt:    now,
		domainEvents: make([]DomainEvent, 0),
	}

	channel.addDomainEvent(&ChannelConnectedEvent{
		ChannelID:   channelID,
		SellerID:    sellerID,
		Type:        channelType,
		ConnectedAt: now,
	})

	return channel, nil
}

// Pause pauses the channel
func (c *Channel) Pause() {
	c.Status = ChannelStatusPaused
	c.UpdatedAt = time.Now().UTC()
}

// Resume resumes the channel
func (c *Channel) Resume() error {
	if c.Status == ChannelStatusDisconnected {
		return errors.New("cannot resume disconnected channel")
	}
	c.Status = ChannelStatusActive
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// Disconnect disconnects the channel
func (c *Channel) Disconnect() {
	c.Status = ChannelStatusDisconnected
	c.UpdatedAt = time.Now().UTC()

	c.addDomainEvent(&ChannelDisconnectedEvent{
		ChannelID:      c.ChannelID,
		SellerID:       c.SellerID,
		DisconnectedAt: c.UpdatedAt,
	})
}

// RecordError records an error
func (c *Channel) RecordError(err string) {
	now := time.Now().UTC()
	c.LastError = err
	c.LastErrorAt = &now
	c.ErrorCount++
	c.UpdatedAt = now

	if c.ErrorCount >= 5 {
		c.Status = ChannelStatusError
	}
}

// ClearErrors clears error state
func (c *Channel) ClearErrors() {
	c.LastError = ""
	c.LastErrorAt = nil
	c.ErrorCount = 0
	if c.Status == ChannelStatusError {
		c.Status = ChannelStatusActive
	}
	c.UpdatedAt = time.Now().UTC()
}

// UpdateLastSync updates the last sync timestamp for a sync type
func (c *Channel) UpdateLastSync(syncType SyncType) {
	now := time.Now().UTC()
	switch syncType {
	case SyncTypeOrders:
		c.LastOrderSync = &now
	case SyncTypeInventory:
		c.LastInventorySync = &now
	case SyncTypeTracking:
		c.LastTrackingSync = &now
	}
	c.UpdatedAt = now
}

// UpdateCredentials updates channel credentials
func (c *Channel) UpdateCredentials(creds ChannelCredentials) {
	c.Credentials = creds
	c.UpdatedAt = time.Now().UTC()
}

// UpdateSyncSettings updates sync settings
func (c *Channel) UpdateSyncSettings(settings SyncSettings) {
	c.SyncSettings = settings
	c.UpdatedAt = time.Now().UTC()
}

// IsActive checks if the channel is active
func (c *Channel) IsActive() bool {
	return c.Status == ChannelStatusActive
}

// Domain event helpers
func (c *Channel) addDomainEvent(event DomainEvent) {
	c.domainEvents = append(c.domainEvents, event)
}

func (c *Channel) DomainEvents() []DomainEvent {
	return c.domainEvents
}

func (c *Channel) ClearDomainEvents() {
	c.domainEvents = make([]DomainEvent, 0)
}

// ChannelOrder represents an order imported from an external channel
type ChannelOrder struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID  string             `bson:"tenantId" json:"tenantId"`
	SellerID  string             `bson:"sellerId" json:"sellerId"`
	ChannelID string             `bson:"channelId" json:"channelId"`

	// External order info
	ExternalOrderID     string    `bson:"externalOrderId" json:"externalOrderId"`
	ExternalOrderNumber string    `bson:"externalOrderNumber" json:"externalOrderNumber"`
	ExternalCreatedAt   time.Time `bson:"externalCreatedAt" json:"externalCreatedAt"`

	// WMS order reference
	WMSOrderID string `bson:"wmsOrderId,omitempty" json:"wmsOrderId,omitempty"`
	Imported   bool   `bson:"imported" json:"imported"`
	ImportedAt *time.Time `bson:"importedAt,omitempty" json:"importedAt,omitempty"`

	// Order details (from channel)
	Customer     ChannelCustomer    `bson:"customer" json:"customer"`
	ShippingAddr ChannelAddress     `bson:"shippingAddress" json:"shippingAddress"`
	BillingAddr  *ChannelAddress    `bson:"billingAddress,omitempty" json:"billingAddress,omitempty"`
	LineItems    []ChannelLineItem  `bson:"lineItems" json:"lineItems"`

	// Financial
	Currency      string  `bson:"currency" json:"currency"`
	Subtotal      float64 `bson:"subtotal" json:"subtotal"`
	ShippingCost  float64 `bson:"shippingCost" json:"shippingCost"`
	Tax           float64 `bson:"tax" json:"tax"`
	Discount      float64 `bson:"discount" json:"discount"`
	Total         float64 `bson:"total" json:"total"`

	// Status
	FinancialStatus   string `bson:"financialStatus" json:"financialStatus"`   // paid, pending, refunded
	FulfillmentStatus string `bson:"fulfillmentStatus" json:"fulfillmentStatus"` // unfulfilled, partial, fulfilled

	// Fulfillment tracking
	TrackingPushed   bool       `bson:"trackingPushed" json:"trackingPushed"`
	TrackingPushedAt *time.Time `bson:"trackingPushedAt,omitempty" json:"trackingPushedAt,omitempty"`

	// Metadata
	Tags     []string               `bson:"tags,omitempty" json:"tags,omitempty"`
	Notes    string                 `bson:"notes,omitempty" json:"notes,omitempty"`
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// ChannelCustomer represents customer info from a channel
type ChannelCustomer struct {
	ExternalID string `bson:"externalId,omitempty" json:"externalId,omitempty"`
	Email      string `bson:"email" json:"email"`
	FirstName  string `bson:"firstName" json:"firstName"`
	LastName   string `bson:"lastName" json:"lastName"`
	Phone      string `bson:"phone,omitempty" json:"phone,omitempty"`
}

// ChannelAddress represents an address from a channel
type ChannelAddress struct {
	FirstName string `bson:"firstName" json:"firstName"`
	LastName  string `bson:"lastName" json:"lastName"`
	Company   string `bson:"company,omitempty" json:"company,omitempty"`
	Address1  string `bson:"address1" json:"address1"`
	Address2  string `bson:"address2,omitempty" json:"address2,omitempty"`
	City      string `bson:"city" json:"city"`
	Province  string `bson:"province" json:"province"`
	Zip       string `bson:"zip" json:"zip"`
	Country   string `bson:"country" json:"country"`
	Phone     string `bson:"phone,omitempty" json:"phone,omitempty"`
}

// ChannelLineItem represents a line item from a channel order
type ChannelLineItem struct {
	ExternalID      string  `bson:"externalId" json:"externalId"`
	SKU             string  `bson:"sku" json:"sku"`
	ProductID       string  `bson:"productId,omitempty" json:"productId,omitempty"`
	VariantID       string  `bson:"variantId,omitempty" json:"variantId,omitempty"`
	Title           string  `bson:"title" json:"title"`
	Quantity        int     `bson:"quantity" json:"quantity"`
	Price           float64 `bson:"price" json:"price"`
	TotalDiscount   float64 `bson:"totalDiscount" json:"totalDiscount"`
	RequiresShipping bool   `bson:"requiresShipping" json:"requiresShipping"`
	Grams           int     `bson:"grams,omitempty" json:"grams,omitempty"`
}

// MarkImported marks the order as imported to WMS
func (o *ChannelOrder) MarkImported(wmsOrderID string) {
	now := time.Now().UTC()
	o.WMSOrderID = wmsOrderID
	o.Imported = true
	o.ImportedAt = &now
	o.UpdatedAt = now
}

// MarkTrackingPushed marks that tracking was pushed to the channel
func (o *ChannelOrder) MarkTrackingPushed() {
	now := time.Now().UTC()
	o.TrackingPushed = true
	o.TrackingPushedAt = &now
	o.UpdatedAt = now
}

// SyncJob represents a synchronization job
type SyncJob struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	JobID     string             `bson:"jobId" json:"jobId"`
	TenantID  string             `bson:"tenantId" json:"tenantId"`
	SellerID  string             `bson:"sellerId" json:"sellerId"`
	ChannelID string             `bson:"channelId" json:"channelId"`

	// Job details
	Type      SyncType   `bson:"type" json:"type"`
	Status    SyncStatus `bson:"status" json:"status"`
	Direction string     `bson:"direction" json:"direction"` // inbound, outbound

	// Progress
	TotalItems     int `bson:"totalItems" json:"totalItems"`
	ProcessedItems int `bson:"processedItems" json:"processedItems"`
	SuccessItems   int `bson:"successItems" json:"successItems"`
	FailedItems    int `bson:"failedItems" json:"failedItems"`

	// Timing
	StartedAt   *time.Time `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt *time.Time `bson:"completedAt,omitempty" json:"completedAt,omitempty"`

	// Error info
	Errors []SyncError `bson:"errors,omitempty" json:"errors,omitempty"`

	// Metadata
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// SyncError represents an error during sync
type SyncError struct {
	ItemID    string    `bson:"itemId" json:"itemId"`
	Error     string    `bson:"error" json:"error"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
}

// NewSyncJob creates a new sync job
func NewSyncJob(tenantID, sellerID, channelID string, syncType SyncType, direction string) *SyncJob {
	now := time.Now().UTC()
	return &SyncJob{
		ID:        primitive.NewObjectID(),
		JobID:     fmt.Sprintf("SYNC-%s", uuid.New().String()[:8]),
		TenantID:  tenantID,
		SellerID:  sellerID,
		ChannelID: channelID,
		Type:      syncType,
		Status:    SyncStatusPending,
		Direction: direction,
		Errors:    make([]SyncError, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Start starts the sync job
func (j *SyncJob) Start() {
	now := time.Now().UTC()
	j.Status = SyncStatusRunning
	j.StartedAt = &now
	j.UpdatedAt = now
}

// Complete completes the sync job
func (j *SyncJob) Complete() {
	now := time.Now().UTC()
	if len(j.Errors) > 0 && j.SuccessItems == 0 {
		j.Status = SyncStatusFailed
	} else {
		j.Status = SyncStatusCompleted
	}
	j.CompletedAt = &now
	j.UpdatedAt = now
}

// Fail fails the sync job
func (j *SyncJob) Fail(err string) {
	now := time.Now().UTC()
	j.Status = SyncStatusFailed
	j.CompletedAt = &now
	j.Errors = append(j.Errors, SyncError{
		Error:     err,
		Timestamp: now,
	})
	j.UpdatedAt = now
}

// AddError adds an error for a specific item
func (j *SyncJob) AddError(itemID, err string) {
	j.Errors = append(j.Errors, SyncError{
		ItemID:    itemID,
		Error:     err,
		Timestamp: time.Now().UTC(),
	})
	j.FailedItems++
	j.UpdatedAt = time.Now().UTC()
}

// IncrementProgress increments progress counters
func (j *SyncJob) IncrementProgress(success bool) {
	j.ProcessedItems++
	if success {
		j.SuccessItems++
	} else {
		j.FailedItems++
	}
	j.UpdatedAt = time.Now().UTC()
}

// SetTotalItems sets the total items to process
func (j *SyncJob) SetTotalItems(total int) {
	j.TotalItems = total
	j.UpdatedAt = time.Now().UTC()
}
