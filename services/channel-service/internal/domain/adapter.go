package domain

import (
	"context"
	"time"
)

// ChannelAdapter defines the interface for channel integrations
type ChannelAdapter interface {
	// GetType returns the channel type this adapter handles
	GetType() ChannelType

	// ValidateCredentials validates the channel credentials
	ValidateCredentials(ctx context.Context, creds ChannelCredentials) error

	// FetchOrders fetches orders from the channel
	FetchOrders(ctx context.Context, channel *Channel, since time.Time) ([]*ChannelOrder, error)

	// FetchOrder fetches a single order by ID
	FetchOrder(ctx context.Context, channel *Channel, externalOrderID string) (*ChannelOrder, error)

	// PushTracking pushes tracking info to the channel
	PushTracking(ctx context.Context, channel *Channel, externalOrderID string, tracking TrackingInfo) error

	// SyncInventory syncs inventory levels to the channel
	SyncInventory(ctx context.Context, channel *Channel, items []InventoryUpdate) error

	// GetInventoryLevels gets current inventory levels from channel
	GetInventoryLevels(ctx context.Context, channel *Channel, skus []string) ([]InventoryLevel, error)

	// CreateFulfillment creates a fulfillment in the channel
	CreateFulfillment(ctx context.Context, channel *Channel, fulfillment FulfillmentRequest) error

	// RegisterWebhooks registers webhooks with the channel
	RegisterWebhooks(ctx context.Context, channel *Channel, webhookURL string) error

	// ValidateWebhook validates an incoming webhook
	ValidateWebhook(ctx context.Context, channel *Channel, signature string, body []byte) bool
}

// TrackingInfo represents tracking information to push
type TrackingInfo struct {
	TrackingNumber string   `json:"trackingNumber"`
	Carrier        string   `json:"carrier"`
	TrackingURL    string   `json:"trackingUrl,omitempty"`
	LineItemIDs    []string `json:"lineItemIds,omitempty"`
	NotifyCustomer bool     `json:"notifyCustomer"`
}

// InventoryUpdate represents an inventory update to push
type InventoryUpdate struct {
	SKU           string `json:"sku"`
	VariantID     string `json:"variantId,omitempty"`
	LocationID    string `json:"locationId,omitempty"`
	Quantity      int    `json:"quantity"`
	Available     int    `json:"available"`
}

// InventoryLevel represents inventory level from channel
type InventoryLevel struct {
	SKU           string `json:"sku"`
	VariantID     string `json:"variantId"`
	ProductID     string `json:"productId"`
	LocationID    string `json:"locationId"`
	Available     int    `json:"available"`
	OnHand        int    `json:"onHand"`
	Committed     int    `json:"committed"`
}

// FulfillmentRequest represents a fulfillment request
type FulfillmentRequest struct {
	OrderID        string         `json:"orderId"`
	LocationID     string         `json:"locationId"`
	TrackingNumber string         `json:"trackingNumber"`
	TrackingURL    string         `json:"trackingUrl"`
	Carrier        string         `json:"carrier"`
	LineItems      []FulfillmentLineItem `json:"lineItems"`
	NotifyCustomer bool           `json:"notifyCustomer"`
}

// FulfillmentLineItem represents a line item in fulfillment
type FulfillmentLineItem struct {
	LineItemID string `json:"lineItemId"`
	Quantity   int    `json:"quantity"`
}

// WebhookPayload represents a webhook payload
type WebhookPayload struct {
	Type      string                 `json:"type"`
	ChannelID string                 `json:"channelId"`
	Topic     string                 `json:"topic"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// AdapterFactory creates channel adapters
type AdapterFactory struct {
	adapters map[ChannelType]ChannelAdapter
}

// NewAdapterFactory creates a new adapter factory
func NewAdapterFactory() *AdapterFactory {
	return &AdapterFactory{
		adapters: make(map[ChannelType]ChannelAdapter),
	}
}

// Register registers an adapter for a channel type
func (f *AdapterFactory) Register(adapter ChannelAdapter) {
	f.adapters[adapter.GetType()] = adapter
}

// GetAdapter returns the adapter for a channel type
func (f *AdapterFactory) GetAdapter(channelType ChannelType) (ChannelAdapter, error) {
	adapter, ok := f.adapters[channelType]
	if !ok {
		return nil, ErrInvalidChannelType
	}
	return adapter, nil
}

// GetAdapterForChannel returns the adapter for a channel
func (f *AdapterFactory) GetAdapterForChannel(channel *Channel) (ChannelAdapter, error) {
	return f.GetAdapter(channel.Type)
}
