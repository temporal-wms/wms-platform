# Channel Service

The Channel Service manages external sales channel integrations, enabling order import, inventory synchronization, and tracking updates with platforms like Shopify, Amazon, eBay, and WooCommerce.

## Overview

| Property | Value |
|----------|-------|
| Port | 8012 |
| Database | channel_service (MongoDB) |
| Aggregate Roots | Channel, ChannelOrder, SyncJob |

## Features

- Multi-channel support (Shopify, Amazon, eBay, WooCommerce, Custom)
- Order import from external channels
- Inventory synchronization to channels
- Tracking information push
- Fulfillment creation on channels
- Webhook handling for real-time updates
- Sync job management with progress tracking
- Error tracking and automatic channel pause on repeated failures
- Encrypted credential storage

## Supported Channels

| Channel | Adapter | Features |
|---------|---------|----------|
| Shopify | ShopifyAdapter | Orders, Inventory, Tracking, Webhooks |
| Amazon | AmazonAdapter | Orders, Inventory, Tracking |
| eBay | EbayAdapter | Orders, Inventory, Tracking |
| WooCommerce | WooCommerceAdapter | Orders, Inventory, Tracking, Webhooks |
| Custom | N/A | Manual integration |

## API Endpoints

### Channel Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/channels` | Connect a new channel |
| GET | `/api/v1/channels/:id` | Get channel details |
| PUT | `/api/v1/channels/:id` | Update channel settings |
| DELETE | `/api/v1/channels/:id` | Disconnect channel |

### Order Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/channels/:id/orders` | List channel orders (paginated) |
| GET | `/api/v1/channels/:id/orders/unimported` | Get unimported orders |
| POST | `/api/v1/channels/:id/orders/import` | Mark order as imported |

### Synchronization

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/channels/:id/sync-jobs` | List sync jobs (paginated) |
| POST | `/api/v1/channels/:id/sync/orders` | Trigger order sync |
| POST | `/api/v1/channels/:id/sync/inventory` | Sync inventory to channel |

### Fulfillment

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/channels/:id/tracking` | Push tracking info to channel |
| POST | `/api/v1/channels/:id/fulfillment` | Create fulfillment on channel |

### Inventory

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/channels/:id/inventory` | Get inventory levels for SKUs |

### Seller Channels

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/sellers/:sellerId/channels` | List seller's channels |

### Webhooks

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/webhooks/:channelId` | Handle incoming webhook |
| POST | `/api/v1/webhooks/:channelId/:topic` | Handle webhook with topic |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `channel.connected` | wms.channels.events | Channel connected |
| `channel.disconnected` | wms.channels.events | Channel disconnected |
| `channel.order.imported` | wms.channels.events | Order imported to WMS |
| `channel.tracking.pushed` | wms.channels.events | Tracking pushed to channel |
| `channel.inventory.synced` | wms.channels.events | Inventory synced to channel |
| `channel.sync.completed` | wms.channels.events | Sync job completed |
| `channel.webhook.received` | wms.channels.events | Webhook received |

## Domain Model

```go
type ChannelType string
const (
    ChannelTypeShopify     ChannelType = "shopify"
    ChannelTypeAmazon      ChannelType = "amazon"
    ChannelTypeEbay        ChannelType = "ebay"
    ChannelTypeWooCommerce ChannelType = "woocommerce"
    ChannelTypeCustom      ChannelType = "custom"
)

type ChannelStatus string
const (
    ChannelStatusActive       ChannelStatus = "active"
    ChannelStatusPaused       ChannelStatus = "paused"
    ChannelStatusDisconnected ChannelStatus = "disconnected"
    ChannelStatusError        ChannelStatus = "error"
)

type SyncType string
const (
    SyncTypeOrders    SyncType = "orders"
    SyncTypeInventory SyncType = "inventory"
    SyncTypeTracking  SyncType = "tracking"
    SyncTypeProducts  SyncType = "products"
)

type SyncStatus string
const (
    SyncStatusPending    SyncStatus = "pending"
    SyncStatusRunning    SyncStatus = "running"
    SyncStatusCompleted  SyncStatus = "completed"
    SyncStatusFailed     SyncStatus = "failed"
    SyncStatusCancelled  SyncStatus = "cancelled"
)

type Channel struct {
    ID                string
    ChannelID         string          // CH-XXXXXXXX format
    TenantID          string
    SellerID          string
    Type              ChannelType
    Name              string
    StoreURL          string
    Status            ChannelStatus
    Credentials       ChannelCredentials
    SyncSettings      SyncSettings
    LastOrderSync     *time.Time
    LastInventorySync *time.Time
    LastTrackingSync  *time.Time
    ErrorCount        int
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

type ChannelOrder struct {
    ID                  string
    ChannelID           string
    ExternalOrderID     string
    ExternalOrderNumber string
    WMSOrderID          string
    Imported            bool
    Customer            ChannelCustomer
    ShippingAddr        ChannelAddress
    LineItems           []ChannelLineItem
    Total               float64
    FinancialStatus     string
    FulfillmentStatus   string
    TrackingPushed      bool
    CreatedAt           time.Time
}

type SyncJob struct {
    ID             string
    JobID          string          // SYNC-XXXXXXXX format
    ChannelID      string
    Type           SyncType
    Status         SyncStatus
    Direction      string          // inbound, outbound
    TotalItems     int
    ProcessedItems int
    SuccessItems   int
    FailedItems    int
    Errors         []SyncError
    StartedAt      *time.Time
    CompletedAt    *time.Time
}
```

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-channel-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8012` |
| `MONGODB_URI` | MongoDB connection URI | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `channel_service` |
| `KAFKA_BROKERS` | Kafka broker addresses | `localhost:9092` |
| `LOG_LEVEL` | Logging level | `info` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector | `localhost:4317` |
| `TRACING_ENABLED` | Enable distributed tracing | `true` |
| `ENVIRONMENT` | Deployment environment | `development` |

## Testing

```bash
# Unit tests
go test ./internal/domain/...

# Integration tests
go test ./tests/integration/...
```

## Related Services

- **seller-service**: Manages seller channel configurations
- **order-service**: Receives imported orders
- **inventory-service**: Provides inventory levels for sync
- **shipping-service**: Provides tracking information
