# Seller Portal

The Seller Portal is a Backend-for-Frontend (BFF) service that provides a unified API for seller dashboards, aggregating data from multiple WMS services to deliver order metrics, inventory status, billing information, and channel integrations.

## Overview

| Property | Value |
|----------|-------|
| Port | 8013 |
| Type | BFF (Backend-for-Frontend) |
| Database | None (aggregates from other services) |

## Features

- Dashboard summary with order, inventory, billing, and channel metrics
- Order management with filtering, search, and pagination
- Inventory visibility across warehouses
- Invoice and billing information
- Channel integration management (connect, disconnect, sync)
- API key management for programmatic access
- Configurable alert notifications
- CORS support for frontend integration

## Required Headers

| Header | Description | Required |
|--------|-------------|----------|
| `X-WMS-Seller-ID` | Seller identifier | Yes |
| `X-WMS-Tenant-ID` | Tenant identifier | Optional |

## API Endpoints

### Dashboard

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/dashboard/summary` | Get dashboard metrics summary |

Query parameters for `/dashboard/summary`:
- `period`: today, week, month, custom (default: today)
- `startDate`: Start date for custom period (YYYY-MM-DD)
- `endDate`: End date for custom period (YYYY-MM-DD)

### Order Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/orders` | List orders (paginated) |
| GET | `/api/v1/orders/:id` | Get order details |

Query parameters for `/orders`:
- `page`, `pageSize`: Pagination
- `status`: Filter by order status
- `channelId`: Filter by channel
- `search`: Search by order ID or customer
- `sortBy`, `sortOrder`: Sorting options

### Inventory Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/inventory` | List inventory (paginated) |

Query parameters:
- `page`, `pageSize`: Pagination
- `warehouseId`: Filter by warehouse
- `status`: available, low_stock, out_of_stock
- `search`: Search by SKU or name

### Billing

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/billing/invoices` | List invoices (paginated) |
| GET | `/api/v1/billing/invoices/:id` | Get invoice details |

### Channel Integrations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/integrations` | List connected channels |
| POST | `/api/v1/integrations` | Connect a new channel |
| DELETE | `/api/v1/integrations/:id` | Disconnect a channel |
| POST | `/api/v1/integrations/:id/sync` | Trigger channel sync |

### API Key Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/api-keys` | List API keys |
| POST | `/api/v1/api-keys` | Generate new API key |
| DELETE | `/api/v1/api-keys/:id` | Revoke API key |

## Dashboard Summary Response

```json
{
  "sellerId": "SLR-12345",
  "period": { "start": "...", "end": "...", "type": "today" },
  "orderMetrics": {
    "totalOrders": 150,
    "pendingOrders": 25,
    "shippedOrders": 100,
    "fulfillmentRate": 95.5,
    "totalRevenue": 15000.00
  },
  "inventoryMetrics": {
    "totalSkus": 500,
    "lowStockSkus": 15,
    "outOfStockSkus": 3,
    "inventoryValue": 50000.00
  },
  "billingMetrics": {
    "currentBalance": 1250.00,
    "mtdCharges": 3500.00,
    "nextInvoiceDate": "2025-01-31"
  },
  "channelMetrics": [...],
  "alerts": [...]
}
```

## Alert Types

| Type | Description |
|------|-------------|
| `low_stock` | SKU below reorder point |
| `out_of_stock` | SKU has zero availability |
| `order_issue` | Order requires attention |
| `shipping_delay` | Shipment delayed |
| `billing_due` | Invoice payment due |
| `channel_error` | Channel sync error |
| `performance` | Performance metric alert |
| `announcement` | System announcement |

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-seller-portal
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8013` |
| `SELLER_SERVICE_URL` | Seller service URL | `http://localhost:8010` |
| `ORDER_SERVICE_URL` | Order service URL | `http://localhost:8001` |
| `INVENTORY_SERVICE_URL` | Inventory service URL | `http://localhost:8002` |
| `BILLING_SERVICE_URL` | Billing service URL | `http://localhost:8011` |
| `CHANNEL_SERVICE_URL` | Channel service URL | `http://localhost:8012` |
| `LOG_LEVEL` | Logging level | `info` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector | `localhost:4317` |
| `TRACING_ENABLED` | Enable distributed tracing | `true` |
| `ENVIRONMENT` | Deployment environment | `development` |

## Architecture

```
                    +-----------------+
                    | Seller Portal   |
                    | (BFF - :8013)   |
                    +--------+--------+
                             |
        +--------------------+--------------------+
        |          |         |         |         |
        v          v         v         v         v
   +--------+ +--------+ +--------+ +--------+ +--------+
   | Seller | | Order  | |Inventory| |Billing | |Channel |
   | :8010  | | :8001  | | :8002  | | :8011  | | :8012  |
   +--------+ +--------+ +--------+ +--------+ +--------+
```

## Testing

```bash
# Unit tests
go test ./internal/...

# Integration tests
go test ./tests/integration/...
```

## Related Services

- **seller-service**: Seller account and API key management
- **order-service**: Order data and status
- **inventory-service**: Stock levels and locations
- **billing-service**: Invoices and billing activities
- **channel-service**: Sales channel integrations
