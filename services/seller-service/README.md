# Seller Service

The Seller Service manages merchant/seller accounts, their contracts, facility assignments, fee schedules, and external sales channel integrations for the WMS Platform.

## Overview

| Property | Value |
|----------|-------|
| Port | 8010 |
| Database | sellers_db (MongoDB) |
| Aggregate Root | Seller |

## Features

- Seller account lifecycle management (create, activate, suspend, close)
- Contract management with billing cycles (daily, weekly, monthly)
- Facility assignment and warehouse access control
- Fee schedule configuration with volume discounts
- Channel integrations (Shopify, Amazon, eBay, WooCommerce)
- API key generation and management for programmatic access
- Multi-tenant support (3PL operators)

## API Endpoints

### Seller Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/sellers` | Create a new seller |
| GET | `/api/v1/sellers` | List sellers (paginated) |
| GET | `/api/v1/sellers/search` | Search sellers by query |
| GET | `/api/v1/sellers/:sellerId` | Get seller by ID |

### Status Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| PUT | `/api/v1/sellers/:sellerId/activate` | Activate seller account |
| PUT | `/api/v1/sellers/:sellerId/suspend` | Suspend seller account |
| PUT | `/api/v1/sellers/:sellerId/close` | Close seller account |

### Facility Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/sellers/:sellerId/facilities` | Assign facility to seller |
| DELETE | `/api/v1/sellers/:sellerId/facilities/:facilityId` | Remove facility assignment |

### Fee Schedule

| Method | Endpoint | Description |
|--------|----------|-------------|
| PUT | `/api/v1/sellers/:sellerId/fee-schedule` | Update fee schedule |

### Channel Integrations

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/sellers/:sellerId/integrations` | Connect sales channel |
| DELETE | `/api/v1/sellers/:sellerId/integrations/:channelId` | Disconnect sales channel |

### API Key Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/sellers/:sellerId/api-keys` | List active API keys |
| POST | `/api/v1/sellers/:sellerId/api-keys` | Generate new API key |
| DELETE | `/api/v1/sellers/:sellerId/api-keys/:keyId` | Revoke API key |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `seller.created` | wms.sellers.events | New seller account created |
| `seller.activated` | wms.sellers.events | Seller account activated |
| `seller.suspended` | wms.sellers.events | Seller account suspended |
| `seller.closed` | wms.sellers.events | Seller account closed |
| `seller.facility_assigned` | wms.sellers.events | Facility assigned to seller |
| `seller.channel_connected` | wms.sellers.events | Sales channel connected |
| `seller.fee_schedule_updated` | wms.sellers.events | Fee schedule updated |

## Domain Model

```go
type Seller struct {
    ID                 string
    SellerID           string           // SLR-XXXXXXXX format
    TenantID           string           // 3PL operator
    CompanyName        string
    ContactName        string
    ContactEmail       string
    ContactPhone       string
    Status             SellerStatus
    ContractStartDate  time.Time
    ContractEndDate    *time.Time
    BillingCycle       BillingCycle
    AssignedFacilities []FacilityAssignment
    FeeSchedule        *FeeSchedule
    Integrations       []ChannelIntegration
    APIKeys            []APIKey
    CreatedAt          time.Time
    UpdatedAt          time.Time
}

type SellerStatus string
const (
    SellerStatusPending   SellerStatus = "pending"
    SellerStatusActive    SellerStatus = "active"
    SellerStatusSuspended SellerStatus = "suspended"
    SellerStatusClosed    SellerStatus = "closed"
)

type BillingCycle string
const (
    BillingCycleDaily   BillingCycle = "daily"
    BillingCycleWeekly  BillingCycle = "weekly"
    BillingCycleMonthly BillingCycle = "monthly"
)
```

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-seller-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8010` |
| `MONGODB_URI` | MongoDB connection URI | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `sellers_db` |
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

- **billing-service**: Receives fee schedules and calculates charges
- **channel-service**: Handles channel sync operations
- **facility-service**: Provides facility information
- **order-service**: Associates orders with sellers
