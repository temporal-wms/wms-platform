package activities

import (
	"context"
	"fmt"
	"log/slog"

	"go.temporal.io/sdk/activity"
)

// ChannelActivities contains activities related to channel operations
type ChannelActivities struct {
	clients *ServiceClients
	logger  *slog.Logger
}

// NewChannelActivities creates a new ChannelActivities instance
func NewChannelActivities(clients *ServiceClients, logger *slog.Logger) *ChannelActivities {
	return &ChannelActivities{
		clients: clients,
		logger:  logger,
	}
}

// TrackingSyncInput represents input for syncing tracking to a channel
type TrackingSyncInput struct {
	ChannelID       string `json:"channelId"`
	ExternalOrderID string `json:"externalOrderId"`
	TrackingNumber  string `json:"trackingNumber"`
	Carrier         string `json:"carrier"`
	TrackingURL     string `json:"trackingUrl,omitempty"`
	NotifyCustomer  bool   `json:"notifyCustomer"`
}

// TrackingSyncResult represents the result of syncing tracking
type TrackingSyncResult struct {
	Success   bool   `json:"success"`
	ChannelID string `json:"channelId"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SyncTrackingToChannel pushes tracking information to an external sales channel
func (a *ChannelActivities) SyncTrackingToChannel(ctx context.Context, input map[string]interface{}) (*TrackingSyncResult, error) {
	logger := activity.GetLogger(ctx)

	channelID, _ := input["channelId"].(string)
	externalOrderID, _ := input["externalOrderId"].(string)
	trackingNumber, _ := input["trackingNumber"].(string)
	carrier, _ := input["carrier"].(string)
	notifyCustomer, _ := input["notifyCustomer"].(bool)

	logger.Info("Syncing tracking to channel",
		"channelId", channelID,
		"externalOrderId", externalOrderID,
		"trackingNumber", trackingNumber,
	)

	result := &TrackingSyncResult{
		ChannelID: channelID,
	}

	if channelID == "" || externalOrderID == "" {
		result.Success = false
		result.Error = "channelId and externalOrderId are required"
		return result, nil
	}

	if trackingNumber == "" {
		result.Success = false
		result.Error = "trackingNumber is required"
		return result, nil
	}

	// Build request for channel service
	trackingRequest := map[string]interface{}{
		"externalOrderId": externalOrderID,
		"trackingNumber":  trackingNumber,
		"carrier":         carrier,
		"notifyCustomer":  notifyCustomer,
	}

	// Call channel service to push tracking
	endpoint := fmt.Sprintf("/api/v1/channels/%s/tracking", channelID)
	_, err := a.clients.PostJSON(ctx, "channel", endpoint, trackingRequest)
	if err != nil {
		logger.Warn("Failed to sync tracking to channel",
			"channelId", channelID,
			"error", err,
		)
		result.Success = false
		result.Error = err.Error()
		return result, nil // Don't fail the activity, just report the error
	}

	result.Success = true
	result.Message = fmt.Sprintf("Tracking %s synced to channel %s for order %s", trackingNumber, channelID, externalOrderID)

	logger.Info("Tracking synced to channel successfully",
		"channelId", channelID,
		"externalOrderId", externalOrderID,
		"trackingNumber", trackingNumber,
	)

	return result, nil
}

// FulfillmentSyncInput represents input for creating a fulfillment in a channel
type FulfillmentSyncInput struct {
	ChannelID       string              `json:"channelId"`
	ExternalOrderID string              `json:"externalOrderId"`
	LocationID      string              `json:"locationId,omitempty"`
	TrackingNumber  string              `json:"trackingNumber"`
	TrackingURL     string              `json:"trackingUrl,omitempty"`
	Carrier         string              `json:"carrier"`
	LineItems       []FulfillmentItem   `json:"lineItems,omitempty"`
	NotifyCustomer  bool                `json:"notifyCustomer"`
}

// FulfillmentItem represents a line item in a fulfillment
type FulfillmentItem struct {
	LineItemID string `json:"lineItemId"`
	Quantity   int    `json:"quantity"`
}

// FulfillmentSyncResult represents the result of creating a fulfillment
type FulfillmentSyncResult struct {
	Success       bool   `json:"success"`
	ChannelID     string `json:"channelId"`
	FulfillmentID string `json:"fulfillmentId,omitempty"`
	Message       string `json:"message,omitempty"`
	Error         string `json:"error,omitempty"`
}

// CreateChannelFulfillment creates a fulfillment in an external sales channel
func (a *ChannelActivities) CreateChannelFulfillment(ctx context.Context, input FulfillmentSyncInput) (*FulfillmentSyncResult, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("Creating fulfillment in channel",
		"channelId", input.ChannelID,
		"externalOrderId", input.ExternalOrderID,
		"trackingNumber", input.TrackingNumber,
	)

	result := &FulfillmentSyncResult{
		ChannelID: input.ChannelID,
	}

	if input.ChannelID == "" || input.ExternalOrderID == "" {
		result.Success = false
		result.Error = "channelId and externalOrderId are required"
		return result, nil
	}

	// Build request for channel service
	fulfillmentRequest := map[string]interface{}{
		"externalOrderId": input.ExternalOrderID,
		"locationId":      input.LocationID,
		"trackingNumber":  input.TrackingNumber,
		"trackingUrl":     input.TrackingURL,
		"carrier":         input.Carrier,
		"lineItems":       input.LineItems,
		"notifyCustomer":  input.NotifyCustomer,
	}

	// Call channel service to create fulfillment
	endpoint := fmt.Sprintf("/api/v1/channels/%s/fulfillment", input.ChannelID)
	resp, err := a.clients.PostJSON(ctx, "channel", endpoint, fulfillmentRequest)
	if err != nil {
		logger.Warn("Failed to create fulfillment in channel",
			"channelId", input.ChannelID,
			"error", err,
		)
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}

	// Extract fulfillment ID if returned
	if respMap, ok := resp.(map[string]interface{}); ok {
		if fulfillmentID, ok := respMap["fulfillmentId"].(string); ok {
			result.FulfillmentID = fulfillmentID
		}
	}

	result.Success = true
	result.Message = fmt.Sprintf("Fulfillment created in channel %s for order %s", input.ChannelID, input.ExternalOrderID)

	logger.Info("Fulfillment created in channel successfully",
		"channelId", input.ChannelID,
		"externalOrderId", input.ExternalOrderID,
		"fulfillmentId", result.FulfillmentID,
	)

	return result, nil
}

// InventorySyncInput represents input for syncing inventory to a channel
type InventorySyncInput struct {
	ChannelID string                `json:"channelId"`
	Items     []InventorySyncItem   `json:"items"`
}

// InventorySyncItem represents an inventory item to sync
type InventorySyncItem struct {
	SKU        string `json:"sku"`
	VariantID  string `json:"variantId,omitempty"`
	LocationID string `json:"locationId,omitempty"`
	Quantity   int    `json:"quantity"`
	Available  int    `json:"available"`
}

// InventorySyncResult represents the result of syncing inventory
type InventorySyncResult struct {
	Success     bool   `json:"success"`
	ChannelID   string `json:"channelId"`
	ItemsSynced int    `json:"itemsSynced"`
	ItemsFailed int    `json:"itemsFailed"`
	JobID       string `json:"jobId,omitempty"`
	Error       string `json:"error,omitempty"`
}

// SyncInventoryToChannel pushes inventory levels to an external sales channel
func (a *ChannelActivities) SyncInventoryToChannel(ctx context.Context, input InventorySyncInput) (*InventorySyncResult, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("Syncing inventory to channel",
		"channelId", input.ChannelID,
		"itemCount", len(input.Items),
	)

	result := &InventorySyncResult{
		ChannelID: input.ChannelID,
	}

	if input.ChannelID == "" {
		result.Success = false
		result.Error = "channelId is required"
		return result, nil
	}

	if len(input.Items) == 0 {
		result.Success = true
		result.ItemsSynced = 0
		return result, nil
	}

	// Build request for channel service
	syncRequest := map[string]interface{}{
		"items": input.Items,
	}

	// Call channel service to sync inventory
	endpoint := fmt.Sprintf("/api/v1/channels/%s/sync/inventory", input.ChannelID)
	resp, err := a.clients.PostJSON(ctx, "channel", endpoint, syncRequest)
	if err != nil {
		logger.Warn("Failed to sync inventory to channel",
			"channelId", input.ChannelID,
			"error", err,
		)
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}

	// Extract job info if returned
	if respMap, ok := resp.(map[string]interface{}); ok {
		if jobID, ok := respMap["id"].(string); ok {
			result.JobID = jobID
		}
		if processed, ok := respMap["processedItems"].(float64); ok {
			result.ItemsSynced = int(processed)
		}
		if failed, ok := respMap["failedItems"].(float64); ok {
			result.ItemsFailed = int(failed)
		}
	}

	result.Success = true

	logger.Info("Inventory synced to channel successfully",
		"channelId", input.ChannelID,
		"itemsSynced", result.ItemsSynced,
		"itemsFailed", result.ItemsFailed,
	)

	return result, nil
}

// FetchChannelOrdersInput represents input for fetching orders from a channel
type FetchChannelOrdersInput struct {
	ChannelID string `json:"channelId"`
	Since     string `json:"since,omitempty"` // ISO8601 timestamp
}

// FetchChannelOrdersResult represents the result of fetching orders
type FetchChannelOrdersResult struct {
	Success     bool   `json:"success"`
	ChannelID   string `json:"channelId"`
	OrderCount  int    `json:"orderCount"`
	NewOrders   int    `json:"newOrders"`
	JobID       string `json:"jobId,omitempty"`
	Error       string `json:"error,omitempty"`
}

// FetchChannelOrders fetches new orders from an external sales channel
func (a *ChannelActivities) FetchChannelOrders(ctx context.Context, input FetchChannelOrdersInput) (*FetchChannelOrdersResult, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("Fetching orders from channel",
		"channelId", input.ChannelID,
		"since", input.Since,
	)

	result := &FetchChannelOrdersResult{
		ChannelID: input.ChannelID,
	}

	if input.ChannelID == "" {
		result.Success = false
		result.Error = "channelId is required"
		return result, nil
	}

	// Build request for channel service
	syncRequest := map[string]interface{}{}
	if input.Since != "" {
		syncRequest["since"] = input.Since
	}

	// Call channel service to sync orders
	endpoint := fmt.Sprintf("/api/v1/channels/%s/sync/orders", input.ChannelID)
	resp, err := a.clients.PostJSON(ctx, "channel", endpoint, syncRequest)
	if err != nil {
		logger.Warn("Failed to fetch orders from channel",
			"channelId", input.ChannelID,
			"error", err,
		)
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}

	// Extract job info if returned
	if respMap, ok := resp.(map[string]interface{}); ok {
		if jobID, ok := respMap["id"].(string); ok {
			result.JobID = jobID
		}
		if total, ok := respMap["totalItems"].(float64); ok {
			result.OrderCount = int(total)
		}
		if processed, ok := respMap["processedItems"].(float64); ok {
			result.NewOrders = int(processed)
		}
	}

	result.Success = true

	logger.Info("Orders fetched from channel successfully",
		"channelId", input.ChannelID,
		"orderCount", result.OrderCount,
		"newOrders", result.NewOrders,
	)

	return result, nil
}
