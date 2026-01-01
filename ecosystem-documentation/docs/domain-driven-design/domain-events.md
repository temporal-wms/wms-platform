---
sidebar_position: 5
---

# Domain Events

This document catalogs all domain events in the WMS Platform.

## Event Structure

All events follow the CloudEvents 1.0 specification:

```json
{
  "specversion": "1.0",
  "type": "wms.order.received",
  "source": "/wms/order-service",
  "subject": "ORD-12345",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "time": "2024-01-15T10:30:00Z",
  "datacontenttype": "application/json",
  "data": {
    "orderId": "ORD-12345",
    "customerId": "CUST-001",
    ...
  }
}
```

## Event Catalog

### Order Events

| Event Type | Topic | Description |
|------------|-------|-------------|
| `wms.order.received` | wms.orders.events | Order placed by customer |
| `wms.order.validated` | wms.orders.events | Order validation passed |
| `wms.order.wave-assigned` | wms.orders.events | Order assigned to wave |
| `wms.order.shipped` | wms.orders.events | Order shipped to carrier |
| `wms.order.cancelled` | wms.orders.events | Order cancelled |
| `wms.order.completed` | wms.orders.events | Order delivered |

```mermaid
graph LR
    OR[OrderReceived] --> OV[OrderValidated]
    OV --> OWA[OrderWaveAssigned]
    OWA --> OS[OrderShipped]
    OS --> OC[OrderCompleted]
    OV -.->|failure| OCA[OrderCancelled]
```

#### OrderReceivedEvent

```json
{
  "type": "wms.order.received",
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

### Wave Events

| Event Type | Topic | Description |
|------------|-------|-------------|
| `wms.wave.created` | wms.waves.events | Wave created |
| `wms.wave.order-added` | wms.waves.events | Order added to wave |
| `wms.wave.scheduled` | wms.waves.events | Wave scheduled for release |
| `wms.wave.released` | wms.waves.events | Wave released for picking |
| `wms.wave.completed` | wms.waves.events | All orders in wave complete |
| `wms.wave.cancelled` | wms.waves.events | Wave cancelled |

```mermaid
graph LR
    WC[WaveCreated] --> WOA[OrderAdded]
    WOA --> WS[WaveScheduled]
    WS --> WR[WaveReleased]
    WR --> WCO[WaveCompleted]
    WS -.->|cancel| WCA[WaveCancelled]
```

#### WaveReleasedEvent

```json
{
  "type": "wms.wave.released",
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

### Routing Events

| Event Type | Topic | Description |
|------------|-------|-------------|
| `wms.routing.route-calculated` | wms.routes.events | Route optimized |
| `wms.routing.route-started` | wms.routes.events | Picking started on route |
| `wms.routing.stop-completed` | wms.routes.events | Route stop picked |
| `wms.routing.route-completed` | wms.routes.events | Route finished |

#### RouteCalculatedEvent

```json
{
  "type": "wms.routing.route-calculated",
  "data": {
    "routeId": "ROUTE-001",
    "taskId": "PICK-001",
    "stops": [
      { "location": "A-01-01", "sequence": 1, "itemId": "ITEM-001" },
      { "location": "A-02-03", "sequence": 2, "itemId": "ITEM-002" }
    ],
    "totalDistance": 150.5,
    "estimatedTime": "PT15M"
  }
}
```

### Picking Events

| Event Type | Topic | Description |
|------------|-------|-------------|
| `wms.picking.task-created` | wms.picking.events | Pick task created |
| `wms.picking.task-assigned` | wms.picking.events | Worker assigned to task |
| `wms.picking.item-picked` | wms.picking.events | Single item picked |
| `wms.picking.exception` | wms.picking.events | Pick exception reported |
| `wms.picking.task-completed` | wms.picking.events | All items picked |

```mermaid
graph LR
    PTC[TaskCreated] --> PTA[TaskAssigned]
    PTA --> IP[ItemPicked]
    IP --> IP
    IP --> PTE[TaskCompleted]
    IP -.->|problem| PE[Exception]
```

#### PickTaskCompletedEvent

```json
{
  "type": "wms.picking.task-completed",
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

### Consolidation Events

| Event Type | Topic | Description |
|------------|-------|-------------|
| `wms.consolidation.started` | wms.consolidation.events | Consolidation started |
| `wms.consolidation.item-consolidated` | wms.consolidation.events | Item added |
| `wms.consolidation.completed` | wms.consolidation.events | All items consolidated |

### Packing Events

| Event Type | Topic | Description |
|------------|-------|-------------|
| `wms.packing.task-created` | wms.packing.events | Pack task created |
| `wms.packing.packaging-suggested` | wms.packing.events | Package type selected |
| `wms.packing.package-sealed` | wms.packing.events | Package sealed |
| `wms.packing.label-applied` | wms.packing.events | Label affixed |
| `wms.packing.task-completed` | wms.packing.events | Packing complete |

### Shipping Events

| Event Type | Topic | Description |
|------------|-------|-------------|
| `wms.shipping.shipment-created` | wms.shipping.events | Shipment created |
| `wms.shipping.label-generated` | wms.shipping.events | Label printed |
| `wms.shipping.manifested` | wms.shipping.events | Added to manifest |
| `wms.shipping.confirmed` | wms.shipping.events | Carrier pickup confirmed |
| `wms.shipping.delivered` | wms.shipping.events | Delivery confirmed |

#### ShipConfirmedEvent

```json
{
  "type": "wms.shipping.confirmed",
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

### Inventory Events

| Event Type | Topic | Description |
|------------|-------|-------------|
| `wms.inventory.received` | wms.inventory.events | Stock received |
| `wms.inventory.reserved` | wms.inventory.events | Stock reserved for order |
| `wms.inventory.picked` | wms.inventory.events | Stock picked |
| `wms.inventory.adjusted` | wms.inventory.events | Manual adjustment |
| `wms.inventory.low-stock-alert` | wms.inventory.events | Below threshold |

### Labor Events

| Event Type | Topic | Description |
|------------|-------|-------------|
| `wms.labor.shift-started` | wms.labor.events | Worker shift began |
| `wms.labor.shift-ended` | wms.labor.events | Worker shift ended |
| `wms.labor.task-assigned` | wms.labor.events | Task assigned to worker |
| `wms.labor.task-completed` | wms.labor.events | Worker completed task |
| `wms.labor.performance-recorded` | wms.labor.events | Performance metrics updated |

## Event Flow Diagram

```mermaid
sequenceDiagram
    autonumber
    participant Order
    participant Kafka
    participant Waving
    participant Picking
    participant Packing
    participant Shipping

    Order->>Kafka: OrderReceivedEvent
    Kafka->>Waving: Subscribe

    Waving->>Kafka: WaveReleasedEvent
    Kafka->>Picking: Subscribe

    Picking->>Kafka: PickTaskCompletedEvent
    Kafka->>Packing: Subscribe

    Packing->>Kafka: PackageSealedEvent
    Kafka->>Shipping: Subscribe

    Shipping->>Kafka: ShipConfirmedEvent
    Kafka->>Order: Subscribe
    Order->>Kafka: OrderShippedEvent
```

## Kafka Topics

| Topic | Events | Retention |
|-------|--------|-----------|
| wms.orders.events | Order events | 7 days |
| wms.waves.events | Wave events | 7 days |
| wms.routes.events | Routing events | 7 days |
| wms.picking.events | Picking events | 7 days |
| wms.consolidation.events | Consolidation events | 7 days |
| wms.packing.events | Packing events | 7 days |
| wms.shipping.events | Shipping events | 7 days |
| wms.inventory.events | Inventory events | 7 days |
| wms.labor.events | Labor events | 7 days |

## Related Documentation

- [Overview](./overview) - DDD overview
- [Bounded Contexts](./bounded-contexts) - Context descriptions
- [API Events](/api/events-api) - AsyncAPI specification
