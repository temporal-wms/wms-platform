---
sidebar_position: 1
---

# AsyncAPI Specifications

All WMS Platform event APIs are documented using AsyncAPI 3.0.0. Events follow the CloudEvents 1.0 specification and are published to Apache Kafka.

## Event Standards

| Standard | Value |
|----------|-------|
| **AsyncAPI Version** | 3.0.0 |
| **Event Format** | CloudEvents 1.0 |
| **Message Broker** | Apache Kafka |
| **Content-Type** | `application/cloudevents+json` |
| **Retention** | 7 days |

## Service Specifications

### Core Fulfillment

| Service | Topic | Spec | Events | Description |
|---------|-------|------|--------|-------------|
| Order Service | `wms.orders.events` | [asyncapi.yaml](./order-service.yaml) | 6 | Order lifecycle events |
| Waving Service | `wms.waves.events` | [asyncapi.yaml](./waving-service.yaml) | 7 | Wave management events |
| WES Service | `wms.wes.events` | [asyncapi.yaml](./wes-service.yaml) | 7 | Route and stage execution |

### Warehouse Operations

| Service | Topic | Spec | Events | Description |
|---------|-------|------|--------|-------------|
| Routing Service | `wms.routes.events` | [asyncapi.yaml](./routing-service.yaml) | 8 | Route lifecycle events |
| Picking Service | `wms.picking.events` | [asyncapi.yaml](./picking-service.yaml) | 7 | Pick task events |
| Walling Service | `wms.walling.events` | [asyncapi.yaml](./walling-service.yaml) | 5 | Put-wall sorting events |
| Consolidation Service | `wms.consolidation.events` | [asyncapi.yaml](./consolidation-service.yaml) | 4 | Consolidation events |
| Packing Service | `wms.packing.events` | [asyncapi.yaml](./packing-service.yaml) | 9 | Packing workflow events |
| Shipping Service | `wms.shipping.events` | [asyncapi.yaml](./shipping-service.yaml) | 8 | SLAM and shipping events |

### Inventory & Inbound

| Service | Topic | Spec | Events | Description |
|---------|-------|------|--------|-------------|
| Inventory Service | `wms.inventory.events` | [asyncapi.yaml](./inventory-service.yaml) | 8 | Stock management events |
| Receiving Service | `wms.receiving.events` | [asyncapi.yaml](./receiving-service.yaml) | 9 | Inbound receiving events |
| Stow Service | `wms.stow.events` | [asyncapi.yaml](./stow-service.yaml) | 7 | Putaway task events |

### Infrastructure & Support

| Service | Topic | Spec | Events | Description |
|---------|-------|------|--------|-------------|
| Labor Service | `wms.labor.events` | [asyncapi.yaml](./labor-service.yaml) | 10 | Workforce events |
| Facility Service | `wms.facility.events` | [asyncapi.yaml](./facility-service.yaml) | 11 | Station status events |
| Sortation Service | `wms.sortation.events` | [asyncapi.yaml](./sortation-service.yaml) | 8 | Package sortation events |

## Shared Definitions

- [wms-events.yaml](./wms-events.yaml) - Platform-wide event schema definitions

## CloudEvents Format

All events follow the CloudEvents 1.0 specification:

```json
{
  "specversion": "1.0",
  "type": "wms.orders.order-received",
  "source": "/wms/order-service",
  "subject": "ORD-12345678",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "time": "2024-01-15T10:30:00Z",
  "datacontenttype": "application/json",
  "data": {
    "orderId": "ORD-12345678",
    "customerId": "CUST-001",
    "items": [...]
  }
}
```

### CloudEvents Headers (Kafka)

| Header | Description |
|--------|-------------|
| `ce_specversion` | CloudEvents version (1.0) |
| `ce_type` | Event type (e.g., `wms.orders.order-received`) |
| `ce_source` | Event source (e.g., `/wms/order-service`) |
| `ce_subject` | Event subject (e.g., order ID) |
| `ce_id` | Unique event ID (UUID) |
| `ce_time` | Event timestamp (ISO 8601) |
| `ce_datacontenttype` | Content type (`application/json`) |

## Kafka Topics Configuration

| Topic | Partitions | Retention | Compression |
|-------|------------|-----------|-------------|
| `wms.orders.events` | 3 | 7 days | gzip |
| `wms.waves.events` | 3 | 7 days | gzip |
| `wms.wes.events` | 3 | 7 days | gzip |
| `wms.walling.events` | 3 | 7 days | gzip |
| `wms.routes.events` | 3 | 7 days | gzip |
| `wms.picking.events` | 6 | 7 days | gzip |
| `wms.consolidation.events` | 3 | 7 days | gzip |
| `wms.packing.events` | 3 | 7 days | gzip |
| `wms.shipping.events` | 3 | 7 days | gzip |
| `wms.inventory.events` | 6 | 7 days | gzip |
| `wms.labor.events` | 3 | 7 days | gzip |

## Consumer Groups

| Consumer Group | Services | Topics Consumed |
|---------------|----------|-----------------|
| `wms-waving` | Waving Service | `wms.orders.events` |
| `wms-wes` | WES Service | `wms.waves.events`, `wms.walling.events` |
| `wms-walling` | Walling Service | `wms.wes.events` |
| `wms-picking` | Picking Service | `wms.waves.events` |
| `wms-inventory` | Inventory Service | `wms.orders.events`, `wms.picking.events` |
| `wms-analytics` | Analytics | All topics |

## Event Patterns

### State Change Events

```yaml
OrderValidated:
  type: wms.orders.order-validated
  data:
    orderId: string
    validatedAt: datetime
    validationResult: object
```

### Progress Events

```yaml
ItemPicked:
  type: wms.picking.item-picked
  data:
    taskId: string
    itemSku: string
    quantity: integer
    locationId: string
    pickedAt: datetime
```

### Completion Events

```yaml
RouteCompleted:
  type: wms.wes.route-completed
  data:
    routeId: string
    orderId: string
    stagesCompleted: integer
    totalDuration: string (ISO 8601 duration)
    completedAt: datetime
```

## Viewing Specifications

You can view these specifications using:

- **AsyncAPI Studio**: Load the YAML file into [AsyncAPI Studio](https://studio.asyncapi.com/)
- **AsyncAPI Generator**: Generate documentation with `@asyncapi/generator`
- **Kafka UI**: Monitor events in real-time
