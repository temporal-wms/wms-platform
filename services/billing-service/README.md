# Billing Service

The Billing Service manages invoicing, billable activities tracking, and fee calculations for the WMS Platform. It aggregates all billable events and generates invoices for sellers.

## Overview

| Property | Value |
|----------|-------|
| Port | 8011 |
| Database | billing_db (MongoDB) |
| Aggregate Roots | BillableActivity, Invoice, StorageCalculation |

## Features

- Billable activity tracking (pick, pack, storage, shipping, etc.)
- Batch activity recording
- Invoice generation from billable activities
- Invoice lifecycle management (draft, finalize, pay, void)
- Tax and discount application
- Storage fee calculation (daily)
- Fee calculation based on seller fee schedules
- Overdue invoice detection
- Multi-tenant support

## API Endpoints

### Activity Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/activities` | Record a billable activity |
| POST | `/api/v1/activities/batch` | Record multiple activities |
| GET | `/api/v1/activities/:activityId` | Get activity by ID |

### Seller-Scoped Queries

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/sellers/:sellerId/activities` | List seller activities (paginated) |
| GET | `/api/v1/sellers/:sellerId/activities/summary` | Get activity summary for period |
| GET | `/api/v1/sellers/:sellerId/invoices` | List seller invoices (paginated) |

### Invoice Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/invoices` | Create a new invoice |
| GET | `/api/v1/invoices/:invoiceId` | Get invoice by ID |
| PUT | `/api/v1/invoices/:invoiceId/finalize` | Finalize invoice for payment |
| PUT | `/api/v1/invoices/:invoiceId/pay` | Mark invoice as paid |
| PUT | `/api/v1/invoices/:invoiceId/void` | Void an invoice |

### Fee Calculation

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/fees/calculate` | Calculate fees for activities |
| POST | `/api/v1/storage/calculate` | Record daily storage calculation |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `billing.invoice.created` | wms.billing.events | New invoice created |
| `billing.invoice.finalized` | wms.billing.events | Invoice finalized for payment |
| `billing.invoice.paid` | wms.billing.events | Invoice marked as paid |
| `billing.invoice.overdue` | wms.billing.events | Invoice is overdue |
| `billing.activity.recorded` | wms.billing.events | Billable activity recorded |
| `billing.storage.calculated` | wms.billing.events | Daily storage fees calculated |

## Domain Model

```go
type ActivityType string
const (
    ActivityTypeStorage          ActivityType = "storage"
    ActivityTypePick             ActivityType = "pick"
    ActivityTypePack             ActivityType = "pack"
    ActivityTypeReceiving        ActivityType = "receiving"
    ActivityTypeShipping         ActivityType = "shipping"
    ActivityTypeReturnProcessing ActivityType = "return_processing"
    ActivityTypeGiftWrap         ActivityType = "gift_wrap"
    ActivityTypeHazmat           ActivityType = "hazmat"
    ActivityTypeOversized        ActivityType = "oversized"
    ActivityTypeColdChain        ActivityType = "cold_chain"
    ActivityTypeFragile          ActivityType = "fragile"
    ActivityTypeSpecialHandling  ActivityType = "special_handling"
)

type InvoiceStatus string
const (
    InvoiceStatusDraft     InvoiceStatus = "draft"
    InvoiceStatusFinalized InvoiceStatus = "finalized"
    InvoiceStatusPaid      InvoiceStatus = "paid"
    InvoiceStatusOverdue   InvoiceStatus = "overdue"
    InvoiceStatusVoided    InvoiceStatus = "voided"
)

type Invoice struct {
    ID            string
    InvoiceID     string           // INV-XXXXXXXX format
    TenantID      string
    SellerID      string
    Status        InvoiceStatus
    InvoiceNumber string
    PeriodStart   time.Time
    PeriodEnd     time.Time
    LineItems     []InvoiceLineItem
    Subtotal      float64
    TaxRate       float64
    TaxAmount     float64
    Discount      float64
    Total         float64
    Currency      string
    DueDate       time.Time
    PaidAt        *time.Time
    PaymentMethod string
    PaymentRef    string
    SellerName    string
    SellerEmail   string
    CreatedAt     time.Time
    UpdatedAt     time.Time
    FinalizedAt   *time.Time
}

type BillableActivity struct {
    ID            string
    ActivityID    string           // ACT-XXXXXXXX format
    TenantID      string
    SellerID      string
    FacilityID    string
    Type          ActivityType
    Description   string
    Quantity      float64
    UnitPrice     float64
    Amount        float64
    Currency      string
    ReferenceType string           // order, inventory, shipment
    ReferenceID   string
    ActivityDate  time.Time
    BillingDate   time.Time
    InvoiceID     *string
    Invoiced      bool
    CreatedAt     time.Time
}
```

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-billing-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8011` |
| `MONGODB_URI` | MongoDB connection URI | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `billing_db` |
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

# Full coverage report (local cache override for sandboxed envs)
GOCACHE=./.gocache go test ./... -coverprofile=cover.out
GOCACHE=./.gocache go tool cover -func=cover.out

# Or via Make
make coverage
make test
make coverage-html
make build
```

## Related Services

- **seller-service**: Provides fee schedules for sellers
- **order-service**: Triggers pick/pack activities
- **shipping-service**: Triggers shipping activities
- **inventory-service**: Triggers storage calculations
- **receiving-service**: Triggers receiving activities
