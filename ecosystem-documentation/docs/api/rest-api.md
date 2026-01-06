---
sidebar_position: 1
---

# REST API Reference

This document provides an overview of the REST APIs exposed by WMS Platform services.

## API Conventions

### Base URL Pattern

```
http://{service-name}:{port}/api/v1/{resource}
```

### Authentication

Currently uses API keys via header:
```http
Authorization: Bearer {api-key}
```

### Response Format

All responses follow this structure:

```json
{
  "data": { ... },
  "meta": {
    "requestId": "uuid",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### Error Response

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid order data",
    "details": [
      { "field": "items", "message": "At least one item required" }
    ]
  },
  "meta": {
    "requestId": "uuid",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 404 | Not Found |
| 409 | Conflict |
| 500 | Internal Error |

## Order Service API (Port 8001)

### Create Order

```http
POST /api/v1/orders
```

Request:
```json
{
  "customerId": "CUST-001",
  "priority": "standard",
  "items": [
    {
      "sku": "SKU-001",
      "productName": "Widget A",
      "quantity": 2,
      "price": { "amount": 29.99, "currency": "USD" }
    }
  ],
  "shippingAddress": {
    "street": "123 Main St",
    "city": "New York",
    "state": "NY",
    "zipCode": "10001",
    "country": "US"
  }
}
```

Response (201):
```json
{
  "data": {
    "id": "ORD-12345",
    "status": "received",
    "createdAt": "2024-01-15T10:30:00Z"
  }
}
```

### Get Order

```http
GET /api/v1/orders/{id}
```

### List Orders

```http
GET /api/v1/orders?status=received&customerId=CUST-001&limit=10&offset=0
```

### Validate Order

```http
PUT /api/v1/orders/{id}/validate
```

### Cancel Order

```http
PUT /api/v1/orders/{id}/cancel
```

Request:
```json
{
  "reason": "Customer request"
}
```

### Reprocessing API

#### Get Eligible Orders for Retry

```http
GET /api/v1/reprocessing/eligible?limit=100&maxRetries=5
```

#### Get Retry Metadata

```http
GET /api/v1/reprocessing/orders/{orderId}/retry-count
```

#### Increment Retry Count

```http
POST /api/v1/reprocessing/orders/{orderId}/retry-count
```

Request:
```json
{
  "failureReason": "Picking station offline",
  "failedAt": "2024-01-15T10:30:00Z"
}
```

#### Reset Order for Retry

```http
POST /api/v1/reprocessing/orders/{orderId}/reset
```

#### Move to Dead Letter Queue

```http
POST /api/v1/reprocessing/orders/{orderId}/dlq
```

Request:
```json
{
  "failureStatus": "picking",
  "failureReason": "Max retries exceeded"
}
```

### Dead Letter Queue API

#### List DLQ Entries

```http
GET /api/v1/dead-letter-queue?resolved=false&limit=50
```

#### Get DLQ Statistics

```http
GET /api/v1/dead-letter-queue/stats
```

Response:
```json
{
  "totalEntries": 42,
  "unresolvedCount": 15,
  "resolvedCount": 27,
  "byFailureStatus": { "picking": 8, "packing": 4 }
}
```

#### Resolve DLQ Entry

```http
PATCH /api/v1/dead-letter-queue/{orderId}/resolve
```

Request:
```json
{
  "resolution": "manual_retry",
  "notes": "Inventory replenished",
  "resolvedBy": "SUPERVISOR-001"
}
```

## Unit Service API (Port 8014)

The Unit Service provides individual unit-level tracking throughout fulfillment.

### Create Units

```http
POST /api/v1/units
```

Request:
```json
{
  "sku": "SKU-001",
  "shipmentId": "SHIP-12345",
  "locationId": "RECV-DOCK-01",
  "quantity": 10,
  "createdBy": "WORKER-001"
}
```

### Reserve Units

```http
POST /api/v1/units/reserve
```

Request:
```json
{
  "orderId": "ORD-12345",
  "pathId": "PATH-001",
  "items": [
    { "sku": "SKU-001", "quantity": 2 }
  ],
  "handlerId": "SYSTEM"
}
```

### Get Unit

```http
GET /api/v1/units/{unitId}
GET /api/v1/units/order/{orderId}
```

### Get Unit Audit Trail

```http
GET /api/v1/units/{unitId}/audit
```

### Unit Operations

```http
POST /api/v1/units/{unitId}/pick
POST /api/v1/units/{unitId}/consolidate
POST /api/v1/units/{unitId}/pack
POST /api/v1/units/{unitId}/ship
POST /api/v1/units/{unitId}/exception
```

### Exception Management

```http
GET /api/v1/exceptions/order/{orderId}
GET /api/v1/exceptions/unresolved
POST /api/v1/exceptions/{exceptionId}/resolve
```

## Process Path Service API (Port 8015)

The Process Path Service determines optimal fulfillment paths based on item characteristics.

### Determine Process Path

```http
POST /api/v1/process-paths/determine
```

Request:
```json
{
  "orderId": "ORD-12345",
  "items": [
    {
      "sku": "SKU-001",
      "quantity": 2,
      "weight": 1.5,
      "isFragile": true,
      "isHazmat": false,
      "requiresColdChain": false
    }
  ],
  "giftWrap": true,
  "totalValue": 299.99
}
```

Response:
```json
{
  "pathId": "PATH-001",
  "requirements": ["multi_item", "gift_wrap", "fragile"],
  "consolidationRequired": true,
  "giftWrapRequired": true,
  "specialHandling": ["fragile_packing"]
}
```

### Get Process Path

```http
GET /api/v1/process-paths/{pathId}
GET /api/v1/process-paths/order/{orderId}
```

### Assign Station

```http
PUT /api/v1/process-paths/{pathId}/station
```

Request:
```json
{
  "stationId": "STATION-A01"
}
```

## WES Service API (Port 8016)

The Warehouse Execution System coordinates order execution through configurable process paths.

### Resolve Execution Plan

```http
POST /api/v1/execution-plans/resolve
```

Request:
```json
{
  "itemCount": 8,
  "multiZone": false
}
```

Response (200):
```json
{
  "templateId": "tpl-pick-wall-pack",
  "pathType": "pick_wall_pack",
  "stages": [
    {"stageType": "picking", "sequence": 0},
    {"stageType": "walling", "sequence": 1},
    {"stageType": "packing", "sequence": 2}
  ]
}
```

### Create Task Route

```http
POST /api/v1/routes
```

Request:
```json
{
  "orderId": "ORD-12345",
  "waveId": "WAVE-001",
  "templateId": "tpl-pick-wall-pack",
  "specialHandling": ["fragile"]
}
```

### Get Task Route

```http
GET /api/v1/routes/{routeId}
GET /api/v1/routes/order/{orderId}
```

### Stage Operations

```http
POST /api/v1/routes/{routeId}/stages/current/assign
POST /api/v1/routes/{routeId}/stages/current/start
POST /api/v1/routes/{routeId}/stages/current/complete
POST /api/v1/routes/{routeId}/stages/current/fail
```

Assign Worker Request:
```json
{
  "workerId": "PICKER-001",
  "taskId": "PT-12345"
}
```

Fail Stage Request:
```json
{
  "error": "Picker reported item shortage"
}
```

### List Templates

```http
GET /api/v1/templates?activeOnly=true
GET /api/v1/templates/{templateId}
```

## Walling Service API (Port 8017)

The Walling Service manages put-wall sorting operations for medium-sized orders.

### Create Walling Task

```http
POST /api/v1/tasks
```

Request:
```json
{
  "orderId": "ORD-12345",
  "waveId": "WAVE-001",
  "routeId": "RT-xyz",
  "putWallId": "PUTWALL-1",
  "destinationBin": "BIN-A1",
  "sourceTotes": [
    {"toteId": "TOTE-001", "pickTaskId": "PT-001", "itemCount": 3}
  ],
  "itemsToSort": [
    {"sku": "SKU-001", "quantity": 2, "fromToteId": "TOTE-001"}
  ]
}
```

### Get Tasks

```http
GET /api/v1/tasks/{taskId}
GET /api/v1/tasks/pending?putWallId=PUTWALL-1&limit=10
GET /api/v1/tasks/walliner/{wallinerId}/active
```

### Assign Walliner

```http
POST /api/v1/tasks/{taskId}/assign
```

Request:
```json
{
  "wallinerId": "WALLINER-001",
  "station": "STATION-1"
}
```

### Sort Item

```http
POST /api/v1/tasks/{taskId}/sort
```

Request:
```json
{
  "sku": "SKU-001",
  "quantity": 1,
  "fromToteId": "TOTE-001"
}
```

### Complete Task

```http
POST /api/v1/tasks/{taskId}/complete
```

## Waving Service API (Port 8002)

### Create Wave

```http
POST /api/v1/waves
```

Request:
```json
{
  "type": "standard",
  "maxOrders": 50
}
```

### Add Order to Wave

```http
POST /api/v1/waves/{id}/orders
```

Request:
```json
{
  "orderId": "ORD-12345"
}
```

### Schedule Wave

```http
PUT /api/v1/waves/{id}/schedule
```

Request:
```json
{
  "scheduledAt": "2024-01-15T14:00:00Z"
}
```

### Release Wave

```http
PUT /api/v1/waves/{id}/release
```

### Get Scheduler Status

```http
GET /api/v1/scheduler/status
```

## Picking Service API (Port 8004)

### Create Pick Task

```http
POST /api/v1/pick-tasks
```

### Assign Worker

```http
PUT /api/v1/pick-tasks/{id}/assign
```

Request:
```json
{
  "workerId": "WORKER-001"
}
```

### Pick Item

```http
POST /api/v1/pick-tasks/{id}/items/{itemId}/pick
```

Request:
```json
{
  "quantity": 2,
  "location": "A-01-02-3"
}
```

### Report Exception

```http
POST /api/v1/pick-tasks/{id}/items/{itemId}/exception
```

Request:
```json
{
  "reason": "item_not_found",
  "notes": "Location empty"
}
```

### Complete Task

```http
PUT /api/v1/pick-tasks/{id}/complete
```

## Inventory Service API (Port 8008)

### Get by SKU

```http
GET /api/v1/inventory/sku/{sku}
```

### Reserve Stock

```http
POST /api/v1/inventory/{id}/reserve
```

Request:
```json
{
  "orderId": "ORD-12345",
  "quantity": 5
}
```

### Release Reservation

```http
POST /api/v1/inventory/{id}/release
```

Request:
```json
{
  "orderId": "ORD-12345"
}
```

### Pick Stock

```http
POST /api/v1/inventory/{id}/pick
```

### Receive Stock

```http
POST /api/v1/inventory/{id}/receive
```

### Adjust Stock

```http
POST /api/v1/inventory/{id}/adjust
```

Request:
```json
{
  "quantity": -5,
  "reason": "damaged"
}
```

## Shipping Service API (Port 8007)

### Create Shipment

```http
POST /api/v1/shipments
```

### Generate Label

```http
POST /api/v1/shipments/{id}/label
```

### Get Rates

```http
POST /api/v1/rates
```

### SLAM Operations

```http
PUT /api/v1/shipments/{id}/scan
PUT /api/v1/shipments/{id}/verify-label
PUT /api/v1/shipments/{id}/stage
PUT /api/v1/shipments/{id}/manifest
PUT /api/v1/shipments/{id}/confirm
```

## Health Endpoints

All services expose:

```http
GET /health    # Liveness probe
GET /ready     # Readiness probe
GET /metrics   # Prometheus metrics
```

## Common Query Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| limit | Max results | ?limit=10 |
| offset | Skip results | ?offset=20 |
| sort | Sort field | ?sort=createdAt |
| order | Sort order | ?order=desc |

## Related Documentation

- [Events API](./events-api) - AsyncAPI specification
- [Services](/services/order-service) - Service details
