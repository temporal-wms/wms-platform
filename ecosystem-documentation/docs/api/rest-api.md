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
