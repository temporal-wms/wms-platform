---
sidebar_position: 19
slug: /temporal/activities/channel-activities
---

# Channel Activities

Activities for integrating with external sales channels (e.g., Shopify, Amazon, eBay) to sync tracking, fulfillments, inventory, and orders.

## Activity Struct

```go
type ChannelActivities struct {
    clients *ServiceClients
    logger  *slog.Logger
}
```

## Activities

### SyncTrackingToChannel

Pushes tracking information to an external sales channel.

**Signature:**
```go
func (a *ChannelActivities) SyncTrackingToChannel(ctx context.Context, input map[string]interface{}) (*TrackingSyncResult, error)
```

**Input Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `channelId` | string | Yes | Channel identifier (e.g., "shopify-store-1") |
| `externalOrderId` | string | Yes | Order ID in the external channel |
| `trackingNumber` | string | Yes | Carrier tracking number |
| `carrier` | string | No | Carrier name (e.g., "UPS", "FedEx") |
| `trackingUrl` | string | No | Direct tracking URL |
| `notifyCustomer` | bool | No | Trigger customer notification |

**Output:**
```go
type TrackingSyncResult struct {
    Success   bool   `json:"success"`
    ChannelID string `json:"channelId"`
    Message   string `json:"message,omitempty"`
    Error     string `json:"error,omitempty"`
}
```

**Error Handling:** Non-fatal - returns result with `Success: false` instead of error.

---

### CreateChannelFulfillment

Creates a fulfillment record in an external sales channel.

**Signature:**
```go
func (a *ChannelActivities) CreateChannelFulfillment(ctx context.Context, input FulfillmentSyncInput) (*FulfillmentSyncResult, error)
```

**Input:**
```go
type FulfillmentSyncInput struct {
    ChannelID       string            `json:"channelId"`
    ExternalOrderID string            `json:"externalOrderId"`
    LocationID      string            `json:"locationId,omitempty"`
    TrackingNumber  string            `json:"trackingNumber"`
    TrackingURL     string            `json:"trackingUrl,omitempty"`
    Carrier         string            `json:"carrier"`
    LineItems       []FulfillmentItem `json:"lineItems,omitempty"`
    NotifyCustomer  bool              `json:"notifyCustomer"`
}

type FulfillmentItem struct {
    LineItemID string `json:"lineItemId"`
    Quantity   int    `json:"quantity"`
}
```

**Output:**
```go
type FulfillmentSyncResult struct {
    Success       bool   `json:"success"`
    ChannelID     string `json:"channelId"`
    FulfillmentID string `json:"fulfillmentId,omitempty"`
    Message       string `json:"message,omitempty"`
    Error         string `json:"error,omitempty"`
}
```

---

### SyncInventoryToChannel

Pushes inventory levels to an external sales channel.

**Signature:**
```go
func (a *ChannelActivities) SyncInventoryToChannel(ctx context.Context, input InventorySyncInput) (*InventorySyncResult, error)
```

**Input:**
```go
type InventorySyncInput struct {
    ChannelID string              `json:"channelId"`
    Items     []InventorySyncItem `json:"items"`
}

type InventorySyncItem struct {
    SKU        string `json:"sku"`
    VariantID  string `json:"variantId,omitempty"`
    LocationID string `json:"locationId,omitempty"`
    Quantity   int    `json:"quantity"`
    Available  int    `json:"available"`
}
```

**Output:**
```go
type InventorySyncResult struct {
    Success     bool   `json:"success"`
    ChannelID   string `json:"channelId"`
    ItemsSynced int    `json:"itemsSynced"`
    ItemsFailed int    `json:"itemsFailed"`
    JobID       string `json:"jobId,omitempty"`
    Error       string `json:"error,omitempty"`
}
```

---

### FetchChannelOrders

Fetches new orders from an external sales channel.

**Signature:**
```go
func (a *ChannelActivities) FetchChannelOrders(ctx context.Context, input FetchChannelOrdersInput) (*FetchChannelOrdersResult, error)
```

**Input:**
```go
type FetchChannelOrdersInput struct {
    ChannelID string `json:"channelId"`
    Since     string `json:"since,omitempty"` // ISO8601 timestamp
}
```

**Output:**
```go
type FetchChannelOrdersResult struct {
    Success    bool   `json:"success"`
    ChannelID  string `json:"channelId"`
    OrderCount int    `json:"orderCount"`
    NewOrders  int    `json:"newOrders"`
    JobID      string `json:"jobId,omitempty"`
    Error      string `json:"error,omitempty"`
}
```

## Configuration

| Property | Value |
|----------|-------|
| Default Timeout | 2 minutes |
| Retry Policy | Standard (3 attempts) |
| Heartbeat | Not required |

## Supported Channels

| Channel | Channel ID Format | Features |
|---------|-------------------|----------|
| Shopify | `shopify-{shop-name}` | All operations |
| Amazon | `amazon-{seller-id}` | Fulfillment, tracking |
| eBay | `ebay-{seller-id}` | Fulfillment, tracking |
| WooCommerce | `woo-{site-id}` | All operations |

## Usage Example

```go
// Sync tracking after shipment
trackingInput := map[string]interface{}{
    "channelId":       "shopify-mystore",
    "externalOrderId": "1234567890",
    "trackingNumber":  "1Z999AA10123456784",
    "carrier":         "UPS",
    "notifyCustomer":  true,
}

var result activities.TrackingSyncResult
err := workflow.ExecuteActivity(ctx, channelActivities.SyncTrackingToChannel, trackingInput).Get(ctx, &result)

if !result.Success {
    // Log warning but don't fail - channel sync is best-effort
    logger.Warn("Channel sync failed", "error", result.Error)
}
```

## Related Workflows

- [Shipping Workflow](../workflows/shipping) - Syncs tracking after shipment
- [Order Fulfillment Workflow](../workflows/order-fulfillment) - May sync fulfillment

## Related Documentation

- [Channel Service](/services/channel-service) - Channel integration service
