---
sidebar_position: 2
---

# Events API (AsyncAPI)

This document describes the event-driven API using CloudEvents specification.

## CloudEvents Format

All events follow CloudEvents 1.0 specification:

```json
{
  "specversion": "1.0",
  "type": "wms.order.received",
  "source": "/wms/order-service",
  "subject": "ORD-12345",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "time": "2024-01-15T10:30:00Z",
  "datacontenttype": "application/json",
  "data": { ... }
}
```

## Kafka Topics

| Topic | Description | Partitions |
|-------|-------------|------------|
| wms.orders.events | Order lifecycle events | 3 |
| wms.waves.events | Wave management events | 3 |
| wms.wes.events | WES execution events | 3 |
| wms.walling.events | Walling/put-wall events | 3 |
| wms.routes.events | Route calculation events | 3 |
| wms.picking.events | Picking operation events | 6 |
| wms.consolidation.events | Consolidation events | 3 |
| wms.packing.events | Packing operation events | 3 |
| wms.shipping.events | Shipping/SLAM events | 3 |
| wms.inventory.events | Stock management events | 6 |
| wms.labor.events | Workforce events | 3 |

## Order Events

### wms.order.received

Published when a new order is placed.

```json
{
  "type": "wms.order.received",
  "source": "/wms/order-service",
  "subject": "ORD-12345",
  "data": {
    "orderId": "ORD-12345",
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
    },
    "totalAmount": { "amount": 59.98, "currency": "USD" }
  }
}
```

### wms.order.validated

```json
{
  "type": "wms.order.validated",
  "data": {
    "orderId": "ORD-12345",
    "validatedAt": "2024-01-15T10:31:00Z"
  }
}
```

### wms.order.shipped

```json
{
  "type": "wms.order.shipped",
  "data": {
    "orderId": "ORD-12345",
    "trackingNumber": "1Z999AA10123456784",
    "carrier": "UPS",
    "shippedAt": "2024-01-15T14:00:00Z"
  }
}
```

### wms.order.cancelled

```json
{
  "type": "wms.order.cancelled",
  "data": {
    "orderId": "ORD-12345",
    "reason": "Customer request",
    "cancelledAt": "2024-01-15T10:35:00Z"
  }
}
```

## Wave Events

### wms.wave.released

```json
{
  "type": "wms.wave.released",
  "source": "/wms/waving-service",
  "subject": "WAVE-2024-001",
  "data": {
    "waveId": "WAVE-2024-001",
    "waveNumber": "W-001",
    "orderIds": ["ORD-12345", "ORD-12346"],
    "totalItems": 15,
    "priority": "standard",
    "releasedAt": "2024-01-15T10:30:00Z"
  }
}
```

## WES Events

### wms.wes.route-created

Published when a new task route is created for an order.

```json
{
  "type": "wms.wes.route-created",
  "source": "/wms/wes-service",
  "subject": "RT-a1b2c3d4",
  "data": {
    "routeId": "RT-a1b2c3d4",
    "orderId": "ORD-12345",
    "waveId": "WAVE-2024-001",
    "pathType": "pick_wall_pack",
    "stages": ["picking", "walling", "packing"],
    "createdAt": "2024-01-15T10:30:00Z"
  }
}
```

### wms.wes.stage-completed

Published when a stage completes successfully.

```json
{
  "type": "wms.wes.stage-completed",
  "source": "/wms/wes-service",
  "subject": "RT-a1b2c3d4",
  "data": {
    "routeId": "RT-a1b2c3d4",
    "orderId": "ORD-12345",
    "stageType": "picking",
    "stageIndex": 0,
    "taskId": "PT-001",
    "workerId": "PICKER-001",
    "duration": "PT12M30S",
    "completedAt": "2024-01-15T10:45:00Z"
  }
}
```

### wms.wes.route-completed

Published when all stages are complete.

```json
{
  "type": "wms.wes.route-completed",
  "source": "/wms/wes-service",
  "subject": "RT-a1b2c3d4",
  "data": {
    "routeId": "RT-a1b2c3d4",
    "orderId": "ORD-12345",
    "pathType": "pick_wall_pack",
    "stagesCompleted": 3,
    "totalDuration": "PT45M",
    "completedAt": "2024-01-15T11:15:00Z"
  }
}
```

## Walling Events

### wms.walling.task-created

Published when a new walling task is created.

```json
{
  "type": "wms.walling.task-created",
  "source": "/wms/walling-service",
  "subject": "WT-a1b2c3d4",
  "data": {
    "taskId": "WT-a1b2c3d4",
    "orderId": "ORD-12345",
    "waveId": "WAVE-2024-001",
    "routeId": "RT-xyz",
    "putWallId": "PUTWALL-1",
    "destinationBin": "BIN-A1",
    "itemCount": 5,
    "sourceTotes": ["TOTE-001", "TOTE-002"],
    "createdAt": "2024-01-15T10:30:00Z"
  }
}
```

### wms.walling.item-sorted

Published when an item is sorted to a bin.

```json
{
  "type": "wms.walling.item-sorted",
  "source": "/wms/walling-service",
  "subject": "WT-a1b2c3d4",
  "data": {
    "taskId": "WT-a1b2c3d4",
    "orderId": "ORD-12345",
    "sku": "SKU-001",
    "quantity": 1,
    "fromToteId": "TOTE-001",
    "toBinId": "BIN-A1",
    "sortedAt": "2024-01-15T10:35:00Z"
  }
}
```

### wms.walling.task-completed

Published when all items are sorted.

```json
{
  "type": "wms.walling.task-completed",
  "source": "/wms/walling-service",
  "subject": "WT-a1b2c3d4",
  "data": {
    "taskId": "WT-a1b2c3d4",
    "orderId": "ORD-12345",
    "routeId": "RT-xyz",
    "itemsSorted": 5,
    "duration": "PT8M15S",
    "completedAt": "2024-01-15T10:38:15Z"
  }
}
```

## Picking Events

### wms.picking.task-completed

```json
{
  "type": "wms.picking.task-completed",
  "source": "/wms/picking-service",
  "subject": "PICK-001",
  "data": {
    "taskId": "PICK-001",
    "orderId": "ORD-12345",
    "waveId": "WAVE-2024-001",
    "workerId": "WORKER-001",
    "itemsPicked": 5,
    "duration": "PT12M30S",
    "toteId": "TOTE-001",
    "completedAt": "2024-01-15T10:45:00Z"
  }
}
```

### wms.picking.exception

```json
{
  "type": "wms.picking.exception",
  "data": {
    "taskId": "PICK-001",
    "itemId": "ITEM-001",
    "sku": "SKU-001",
    "reason": "item_not_found",
    "location": "A-01-02-3",
    "reportedAt": "2024-01-15T10:40:00Z"
  }
}
```

## Shipping Events

### wms.shipping.confirmed

```json
{
  "type": "wms.shipping.confirmed",
  "source": "/wms/shipping-service",
  "subject": "SHIP-001",
  "data": {
    "shipmentId": "SHIP-001",
    "orderId": "ORD-12345",
    "trackingNumber": "1Z999AA10123456784",
    "carrier": "UPS",
    "service": "GROUND",
    "weight": { "value": 2.5, "unit": "kg" },
    "shippedAt": "2024-01-15T14:00:00Z"
  }
}
```

## Inventory Events

### wms.inventory.reserved

```json
{
  "type": "wms.inventory.reserved",
  "source": "/wms/inventory-service",
  "subject": "INV-001",
  "data": {
    "inventoryId": "INV-001",
    "sku": "SKU-001",
    "orderId": "ORD-12345",
    "quantity": 5,
    "reservedAt": "2024-01-15T10:30:00Z"
  }
}
```

### wms.inventory.low-stock-alert

```json
{
  "type": "wms.inventory.low-stock-alert",
  "data": {
    "inventoryId": "INV-001",
    "sku": "SKU-001",
    "currentQuantity": 8,
    "minStock": 10,
    "location": "A-01-02-3"
  }
}
```

## Consumer Groups

| Service | Consumer Group | Topics |
|---------|---------------|--------|
| Waving Service | wms-waving | wms.orders.events |
| WES Service | wms-wes | wms.waves.events, wms.walling.events |
| Walling Service | wms-walling | wms.wes.events |
| Picking Service | wms-picking | wms.waves.events |
| Inventory Service | wms-inventory | wms.orders.events, wms.picking.events |
| Analytics | wms-analytics | All topics |

## Message Ordering

Events are ordered by:
- **Order Events**: Partitioned by orderId
- **Picking Events**: Partitioned by waveId
- **Inventory Events**: Partitioned by sku

## Idempotency

Consumers must handle duplicate events using the event `id` field.

## Related Documentation

- [REST API](./rest-api) - Synchronous API
- [Domain Events](/domain-driven-design/domain-events) - Event catalog
- [Kafka Infrastructure](/infrastructure/kafka) - Kafka setup
