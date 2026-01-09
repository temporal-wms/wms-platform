## WMS Platform API Documentation

This document provides comprehensive API documentation for the WMS (Warehouse Management System) Platform.

---

## Table of Contents

1. [Overview](#overview)
2. [REST APIs (OpenAPI)](#rest-apis-openapi)
3. [Event-Driven APIs (AsyncAPI)](#event-driven-apis-asyncapi)
4. [Authentication](#authentication)
5. [Error Handling](#error-handling)
6. [Pagination](#pagination)
7. [Rate Limiting](#rate-limiting)
8. [Getting Started](#getting-started)

---

## Overview

The WMS Platform exposes two types of APIs:

### REST APIs (Synchronous)
- **Protocol**: HTTP/HTTPS
- **Format**: JSON
- **Style**: RESTful
- **Documentation**: OpenAPI 3.0.3
- **Use Case**: Direct service operations (create, read, update, delete)

### Event-Driven APIs (Asynchronous)
- **Protocol**: Apache Kafka
- **Format**: CloudEvents (JSON)
- **Style**: Pub/Sub
- **Documentation**: AsyncAPI 3.0.0
- **Use Case**: Event notifications, inter-service communication

---

## REST APIs (OpenAPI)

### Services

#### Core Warehouse Operations

| Service | Port | Base URL | Description |
|---------|------|----------|-------------|
| **order-service** | 8001 | `/api/v1` | Order lifecycle management |
| **waving-service** | 8002 | `/api/v1` | Batch grouping for picking |
| **routing-service** | 8003 | `/api/v1` | Pick path optimization |
| **picking-service** | 8004 | `/api/v1` | Warehouse picking operations |
| **consolidation-service** | 8005 | `/api/v1` | Multi-item order combining |
| **packing-service** | 8006 | `/api/v1` | Package preparation |
| **shipping-service** | 8007 | `/api/v1` | Carrier integration, SLAM |
| **inventory-service** | 8008 | `/api/v1` | Stock levels, reservations |
| **labor-service** | 8009 | `/api/v1` | Workforce management |

#### Business Services

| Service | Port | Base URL | Description |
|---------|------|----------|-------------|
| **seller-service** | 8010 | `/api/v1` | Merchant account management |
| **billing-service** | 8011 | `/api/v1` | Invoicing and billable activities |
| **channel-service** | 8012 | `/api/v1` | Sales channel integrations |
| **seller-portal** | 8013 | `/api/v1` | Seller dashboard BFF |

#### System Services

| Service | Port | Base URL | Description |
|---------|------|----------|-------------|
| **unit-service** | 8014 | `/api/v1` | Individual unit tracking |
| **process-path-service** | 8015 | `/api/v1` | Fulfillment path determination |
| **wes-service** | 8016 | `/api/v1` | Warehouse execution orchestration |
| **walling-service** | 8017 | `/api/v1` | Put-wall sorting operations |

### Common Endpoints

All services expose standard health and metrics endpoints:

```
GET /health          - Liveness probe (always returns 200 if service is running)
GET /ready           - Readiness probe (checks MongoDB connection)
GET /metrics         - Prometheus metrics
```

### Example: Create Order

**Request:**
```bash
curl -X POST http://localhost:8001/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customerId": "CUST-123",
    "items": [
      {
        "sku": "SKU-12345",
        "quantity": 2,
        "weight": 1.5
      }
    ],
    "shippingAddress": {
      "street": "123 Main St",
      "city": "San Francisco",
      "state": "CA",
      "postalCode": "94105",
      "country": "US"
    },
    "priority": "same_day",
    "promisedDeliveryAt": "2025-12-25T15:00:00Z"
  }'
```

**Response (201 Created):**
```json
{
  "orderId": "ORD-a1b2c3d4",
  "customerId": "CUST-123",
  "items": [
    {
      "sku": "SKU-12345",
      "quantity": 2,
      "weight": 1.5
    }
  ],
  "shippingAddress": {
    "street": "123 Main St",
    "city": "San Francisco",
    "state": "CA",
    "postalCode": "94105",
    "country": "US"
  },
  "priority": "same_day",
  "status": "received",
  "promisedDeliveryAt": "2025-12-25T15:00:00Z",
  "createdAt": "2025-12-23T10:00:00Z",
  "updatedAt": "2025-12-23T10:00:00Z"
}
```

---

## Event-Driven APIs (AsyncAPI)

### Kafka Topics

| Topic | Producers | Consumers | Purpose |
|-------|-----------|-----------|---------|
| `wms.orders.events` | order-service | waving-service, inventory-service | Order lifecycle events |
| `wms.waves.events` | waving-service | orchestrator, labor-service | Wave management events |
| `wms.picking.events` | picking-service | orchestrator, labor-service | Picking task events |
| `wms.inventory.events` | inventory-service | order-service, reporting | Inventory change events |
| `wms.labor.events` | labor-service | reporting, analytics | Labor and shift events |
| `wms.sellers.events` | seller-service | billing-service, channel-service | Seller account events |
| `wms.billing.events` | billing-service | seller-portal, notifications | Invoice and billing events |
| `wms.channels.events` | channel-service | order-service, inventory-service | Channel sync events |
| `wms.units.events` | unit-service | picking-service, consolidation-service | Unit lifecycle events |
| `wms.wes.events` | wes-service | picking-service, packing-service | Execution route events |
| `wms.walling.events` | walling-service | wes-service, consolidation-service | Put-wall sorting events |

### AsyncAPI Specification

Full specification: [asyncapi.yaml](asyncapi.yaml)

View interactive documentation:
```bash
# Install AsyncAPI CLI
npm install -g @asyncapi/cli

# Generate HTML documentation
asyncapi generate html asyncapi.yaml -o ./asyncapi-docs

# Start documentation server
cd asyncapi-docs && python3 -m http.server 8080
```

Then open: http://localhost:8080

### CloudEvents Format

All events follow the [CloudEvents](https://cloudevents.io/) specification:

```json
{
  "specversion": "1.0",
  "type": "com.wms.orders.OrderReceived",
  "source": "/order-service",
  "id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "time": "2025-12-23T10:00:00Z",
  "datacontenttype": "application/json",
  "wmscorrelationid": "corr-12345",
  "wmswavenumber": "WV-20251223-120000",
  "wmsworkflowid": "order-fulfillment-ORD-a1b2c3d4",
  "data": {
    "orderId": "ORD-a1b2c3d4",
    "customerId": "CUST-123",
    "orderLines": [
      {
        "sku": "SKU-12345",
        "quantity": 2,
        "weight": 1.5
      }
    ],
    "priority": "same_day",
    "promisedDeliveryAt": "2025-12-25T15:00:00Z"
  }
}
```

### Event Types

#### Order Events
- `com.wms.orders.OrderReceived` - New order created
- `com.wms.orders.OrderValidated` - Order passed validation
- `com.wms.orders.OrderCancelled` - Order cancelled
- `com.wms.orders.OrderAssignedToWave` - Order assigned to wave
- `com.wms.orders.OrderShipped` - Order shipped to customer
- `com.wms.orders.OrderCompleted` - Order fulfillment completed

#### Wave Events
- `com.wms.waves.WaveCreated` - New wave created
- `com.wms.waves.WaveScheduled` - Wave scheduled for execution
- `com.wms.waves.WaveReleased` - Wave released for picking
- `com.wms.waves.WaveCompleted` - All orders in wave picked
- `com.wms.waves.WaveCancelled` - Wave cancelled

#### Picking Events
- `com.wms.picking.PickTaskCreated` - Pick task created
- `com.wms.picking.PickTaskAssigned` - Task assigned to worker
- `com.wms.picking.PickTaskStarted` - Worker started picking
- `com.wms.picking.PickTaskCompleted` - Picking completed
- `com.wms.picking.PickTaskCancelled` - Pick task cancelled

#### Inventory Events
- `com.wms.inventory.InventoryReceived` - New stock received
- `com.wms.inventory.InventoryReserved` - Stock reserved for order
- `com.wms.inventory.InventoryReleased` - Reservation released
- `com.wms.inventory.InventoryAdjusted` - Stock quantity adjusted
- `com.wms.inventory.LowStockAlert` - Stock below reorder point

#### Labor Events
- `com.wms.labor.ShiftStarted` - Worker started shift
- `com.wms.labor.ShiftEnded` - Worker ended shift
- `com.wms.labor.WorkerAssigned` - Worker assigned to task
- `com.wms.labor.TaskCompleted` - Worker completed task

#### Seller Events
- `com.wms.sellers.SellerCreated` - New seller account created
- `com.wms.sellers.SellerActivated` - Seller account activated
- `com.wms.sellers.SellerSuspended` - Seller account suspended
- `com.wms.sellers.SellerClosed` - Seller account closed
- `com.wms.sellers.FacilityAssigned` - Facility assigned to seller
- `com.wms.sellers.ChannelConnected` - Sales channel connected

#### Billing Events
- `com.wms.billing.InvoiceCreated` - Invoice created
- `com.wms.billing.InvoiceFinalized` - Invoice finalized for payment
- `com.wms.billing.InvoicePaid` - Invoice marked as paid
- `com.wms.billing.InvoiceOverdue` - Invoice is overdue
- `com.wms.billing.ActivityRecorded` - Billable activity recorded

#### Channel Events
- `com.wms.channels.ChannelConnected` - Channel connected
- `com.wms.channels.ChannelDisconnected` - Channel disconnected
- `com.wms.channels.OrderImported` - Order imported from channel
- `com.wms.channels.TrackingPushed` - Tracking pushed to channel
- `com.wms.channels.InventorySynced` - Inventory synced to channel

#### Unit Events
- `com.wms.units.UnitCreated` - Unit created at receiving
- `com.wms.units.UnitReserved` - Unit reserved for order
- `com.wms.units.UnitPicked` - Unit picked into tote
- `com.wms.units.UnitConsolidated` - Unit consolidated
- `com.wms.units.UnitPacked` - Unit packed into package
- `com.wms.units.UnitShipped` - Unit shipped
- `com.wms.units.UnitException` - Exception occurred

#### WES Events
- `com.wms.wes.RouteCreated` - Task route created
- `com.wms.wes.StageAssigned` - Worker assigned to stage
- `com.wms.wes.StageStarted` - Stage execution started
- `com.wms.wes.StageCompleted` - Stage completed
- `com.wms.wes.StageFailed` - Stage failed
- `com.wms.wes.RouteCompleted` - All stages completed

#### Walling Events
- `com.wms.walling.TaskCreated` - Walling task created
- `com.wms.walling.TaskAssigned` - Walliner assigned
- `com.wms.walling.ItemSorted` - Item sorted to bin
- `com.wms.walling.TaskCompleted` - All items sorted

---

## Authentication

### Current Implementation

⚠️ **Note**: Authentication is not yet implemented. All endpoints are currently public.

### Planned Implementation

JWT-based authentication with the following flow:

1. **Obtain Token**
```bash
POST /auth/login
{
  "username": "user@example.com",
  "password": "password"
}

Response:
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresIn": 3600
}
```

2. **Use Token**
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8001/api/v1/orders
```

### Roles & Permissions

Planned roles:
- **admin**: Full access to all endpoints
- **warehouse_manager**: Manage waves, tasks, workers
- **warehouse_worker**: View assigned tasks, update task status
- **api_client**: Limited read/write access for integrations

---

## Error Handling

### Standard Error Response

All errors return a consistent format:

```json
{
  "code": "VALIDATION_ERROR",
  "message": "Invalid order data",
  "details": {
    "customerId": "required",
    "items": "must contain at least 1 item"
  },
  "requestId": "req-a1b2c3d4",
  "timestamp": "2025-12-23T10:00:00Z",
  "path": "/api/v1/orders"
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Request validation failed |
| `BAD_REQUEST` | 400 | Malformed request |
| `RESOURCE_NOT_FOUND` | 404 | Resource does not exist |
| `CONFLICT` | 409 | Resource conflict (duplicate, state mismatch) |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `INTERNAL_ERROR` | 500 | Unexpected server error |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable |
| `TIMEOUT` | 504 | Operation timed out |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |

---

## Pagination

### Request Parameters

All list endpoints support pagination:

```
GET /api/v1/orders?page=1&pageSize=20
```

**Parameters:**
- `page` (integer, default: 1) - Page number (1-indexed)
- `pageSize` (integer, default: 20, max: 100) - Items per page

### Response Format

```json
{
  "data": [...],
  "page": 1,
  "pageSize": 20,
  "totalItems": 150,
  "totalPages": 8,
  "hasNext": true,
  "hasPrev": false
}
```

### Additional Query Parameters

Most list endpoints also support:
- **Sorting**: `?sortBy=createdAt&order=desc`
- **Filtering**: `?status=validated&priority=same_day`
- **Search**: `?search=CUST-123`
- **Date Range**: `?dateFrom=2025-12-01&dateTo=2025-12-31`

---

## Rate Limiting

### Current Implementation

⚠️ **Note**: Rate limiting is not yet enforced.

### Planned Limits

| Endpoint Type | Limit | Window |
|---------------|-------|--------|
| Public endpoints | 100 req/min | Per IP |
| Authenticated endpoints | 1000 req/min | Per user |
| Admin endpoints | 10000 req/min | Per user |

### Rate Limit Headers

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640260800
```

### Rate Limit Exceeded Response

```json
{
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "Rate limit exceeded",
  "requestId": "req-123",
  "timestamp": "2025-12-23T10:00:00Z"
}
```

---

## Getting Started

### Prerequisites

1. **Start Infrastructure**
```bash
docker-compose up -d
```

This starts:
- MongoDB (port 27017)
- Kafka (port 9092)
- Zookeeper (port 2181)
- Temporal (port 7233)
- Jaeger (port 16686)

2. **Start Services**
```bash
# Terminal 1: Order Service
cd services/order-service
go run cmd/api/main.go

# Terminal 2: Waving Service
cd services/waving-service
go run cmd/api/main.go

# Terminal 3: Orchestrator Worker
cd orchestrator
go run cmd/worker/main.go

# ... repeat for other services
```

### Quick Example: Complete Order Flow

**1. Create Order**
```bash
curl -X POST http://localhost:8001/api/v1/orders \
  -H "Content-Type: application/json" \
  -d @examples/create-order.json

# Returns: { "orderId": "ORD-abc123", ... }
```

**2. Create Wave**
```bash
curl -X POST http://localhost:8002/api/v1/waves \
  -H "Content-Type: application/json" \
  -d '{
    "waveType": "priority",
    "zone": "A1"
  }'

# Returns: { "waveId": "WV-PRI-20251223-120000", ... }
```

**3. Add Order to Wave**
```bash
curl -X POST http://localhost:8002/api/v1/waves/WV-PRI-20251223-120000/orders \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "ORD-abc123",
    "priority": 1
  }'
```

**4. Schedule Wave**
```bash
curl -X POST http://localhost:8002/api/v1/waves/WV-PRI-20251223-120000/schedule \
  -H "Content-Type: application/json" \
  -d '{
    "scheduledStart": "2025-12-23T14:00:00Z",
    "scheduledEnd": "2025-12-23T16:00:00Z"
  }'
```

**5. Release Wave**
```bash
curl -X POST http://localhost:8002/api/v1/waves/WV-PRI-20251223-120000/release
```

**6. Check Order Status**
```bash
curl http://localhost:8001/api/v1/orders/ORD-abc123

# Status should progress through:
# received → validated → wave_assigned → picking → ...
```

### Monitoring Events

**Subscribe to Order Events:**
```bash
kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic wms.orders.events \
  --from-beginning \
  --property print.headers=true
```

**View in Temporal UI:**
```
http://localhost:8233
```

**View Traces in Jaeger:**
```
http://localhost:16686
```

**View Metrics:**
```
http://localhost:8001/metrics
```

---

## API Testing

### Postman Collection

Import the Postman collection for easy API testing:
```
docs/postman/WMS-Platform.postman_collection.json
```

### Example Requests

Example request/response files are in:
```
docs/examples/
├── orders/
│   ├── create-order.json
│   ├── cancel-order.json
│   └── list-orders.json
├── waves/
│   ├── create-wave.json
│   └── release-wave.json
└── ...
```

---

## API Versioning

### Current Version: v1

All endpoints are prefixed with `/api/v1`.

### Version Strategy

- **URL Versioning**: `/api/v1`, `/api/v2`, etc.
- **Backward Compatibility**: v1 will be maintained for at least 6 months after v2 release
- **Deprecation Notice**: Deprecated endpoints return `Deprecation` header

---

## Support

### Documentation

- **REST API**: See OpenAPI specs in `docs/openapi/`
- **Events**: See AsyncAPI spec in `docs/asyncapi.yaml`
- **Resilience**: See `shared/pkg/RESILIENCE.md`
- **Implementation Status**: See `IMPLEMENTATION_STATUS.md`

### Generating Documentation

**OpenAPI HTML:**
```bash
npx @redocly/cli build-docs docs/openapi/order-service.yaml -o docs/html/order-service.html
```

**AsyncAPI HTML:**
```bash
asyncapi generate html docs/asyncapi.yaml -o docs/html/asyncapi
```

### Contact

- **Email**: wms-platform@example.com
- **Issues**: https://github.com/wms-platform/issues
- **Slack**: #wms-platform

---

## Appendix

### HTTP Status Codes

| Code | Meaning | When to Use |
|------|---------|-------------|
| 200 | OK | Successful GET, PUT |
| 201 | Created | Successful POST |
| 204 | No Content | Successful DELETE |
| 400 | Bad Request | Validation error, malformed request |
| 401 | Unauthorized | Authentication required |
| 403 | Forbidden | Authenticated but not authorized |
| 404 | Not Found | Resource does not exist |
| 409 | Conflict | Resource conflict |
| 422 | Unprocessable Entity | Business rule violation |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Unexpected server error |
| 503 | Service Unavailable | Service down or overloaded |
| 504 | Gateway Timeout | Upstream timeout |

### Content Types

- **Request**: `Content-Type: application/json`
- **Response**: `Content-Type: application/json`
- **CloudEvents**: `Content-Type: application/cloudevents+json`
- **Metrics**: `Content-Type: text/plain; version=0.0.4`

### Headers

**Request Headers:**
- `Content-Type`: Request content type
- `Accept`: Accepted response format
- `Authorization`: Authentication token (future)
- `X-Request-ID`: Optional request ID for tracing
- `X-Correlation-ID`: Optional correlation ID for distributed tracing

**Response Headers:**
- `Content-Type`: Response content type
- `X-Request-ID`: Request ID (echoed or generated)
- `X-Correlation-ID`: Correlation ID
- `X-RateLimit-*`: Rate limit information (future)
