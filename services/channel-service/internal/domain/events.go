package domain

import "time"

// DomainEvent is the base interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// ChannelConnectedEvent is emitted when a channel is connected
type ChannelConnectedEvent struct {
	ChannelID   string      `json:"channelId"`
	SellerID    string      `json:"sellerId"`
	Type        ChannelType `json:"type"`
	ConnectedAt time.Time   `json:"connectedAt"`
}

func (e *ChannelConnectedEvent) EventType() string    { return "channel.connected" }
func (e *ChannelConnectedEvent) OccurredAt() time.Time { return e.ConnectedAt }

// ChannelDisconnectedEvent is emitted when a channel is disconnected
type ChannelDisconnectedEvent struct {
	ChannelID      string    `json:"channelId"`
	SellerID       string    `json:"sellerId"`
	DisconnectedAt time.Time `json:"disconnectedAt"`
}

func (e *ChannelDisconnectedEvent) EventType() string    { return "channel.disconnected" }
func (e *ChannelDisconnectedEvent) OccurredAt() time.Time { return e.DisconnectedAt }

// OrderImportedEvent is emitted when an order is imported from a channel
type OrderImportedEvent struct {
	ChannelID       string    `json:"channelId"`
	SellerID        string    `json:"sellerId"`
	ExternalOrderID string    `json:"externalOrderId"`
	WMSOrderID      string    `json:"wmsOrderId"`
	ImportedAt      time.Time `json:"importedAt"`
}

func (e *OrderImportedEvent) EventType() string    { return "channel.order.imported" }
func (e *OrderImportedEvent) OccurredAt() time.Time { return e.ImportedAt }

// TrackingPushedEvent is emitted when tracking is pushed to a channel
type TrackingPushedEvent struct {
	ChannelID       string    `json:"channelId"`
	SellerID        string    `json:"sellerId"`
	ExternalOrderID string    `json:"externalOrderId"`
	TrackingNumber  string    `json:"trackingNumber"`
	Carrier         string    `json:"carrier"`
	PushedAt        time.Time `json:"pushedAt"`
}

func (e *TrackingPushedEvent) EventType() string    { return "channel.tracking.pushed" }
func (e *TrackingPushedEvent) OccurredAt() time.Time { return e.PushedAt }

// InventorySyncedEvent is emitted when inventory is synced to a channel
type InventorySyncedEvent struct {
	ChannelID  string    `json:"channelId"`
	SellerID   string    `json:"sellerId"`
	SKU        string    `json:"sku"`
	Quantity   int       `json:"quantity"`
	SyncedAt   time.Time `json:"syncedAt"`
}

func (e *InventorySyncedEvent) EventType() string    { return "channel.inventory.synced" }
func (e *InventorySyncedEvent) OccurredAt() time.Time { return e.SyncedAt }

// SyncCompletedEvent is emitted when a sync job completes
type SyncCompletedEvent struct {
	JobID          string     `json:"jobId"`
	ChannelID      string     `json:"channelId"`
	Type           SyncType   `json:"type"`
	Status         SyncStatus `json:"status"`
	TotalItems     int        `json:"totalItems"`
	SuccessItems   int        `json:"successItems"`
	FailedItems    int        `json:"failedItems"`
	CompletedAt    time.Time  `json:"completedAt"`
}

func (e *SyncCompletedEvent) EventType() string    { return "channel.sync.completed" }
func (e *SyncCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }

// WebhookReceivedEvent is emitted when a webhook is received from a channel
type WebhookReceivedEvent struct {
	ChannelID   string    `json:"channelId"`
	WebhookType string    `json:"webhookType"`
	ReceivedAt  time.Time `json:"receivedAt"`
}

func (e *WebhookReceivedEvent) EventType() string    { return "channel.webhook.received" }
func (e *WebhookReceivedEvent) OccurredAt() time.Time { return e.ReceivedAt }
