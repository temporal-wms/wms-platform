---
sidebar_position: 0
---

# AsyncAPI Specifications

This directory contains AsyncAPI 3.0.0 specifications for all WMS Platform event-driven APIs.

## Specifications

| Service | File | Kafka Topic | Description |
|---------|------|-------------|-------------|
| Order Service | [order-service.yaml](./order-service.yaml) | wms.orders.events | Order lifecycle events |
| Waving Service | [waving-service.yaml](./waving-service.yaml) | wms.waves.events | Wave management events |
| WES Service | [wes-service.yaml](./wes-service.yaml) | wms.wes.events | Route execution events |
| Walling Service | [walling-service.yaml](./walling-service.yaml) | wms.walling.events | Put-wall sorting events |
| Picking Service | [picking-service.yaml](./picking-service.yaml) | wms.picking.events | Pick task events |
| Packing Service | [packing-service.yaml](./packing-service.yaml) | wms.packing.events | Pack task events |
| Shipping Service | [shipping-service.yaml](./shipping-service.yaml) | wms.shipping.events | Shipping events |
| Inventory Service | [inventory-service.yaml](./inventory-service.yaml) | wms.inventory.events | Stock events |
| Routing Service | [routing-service.yaml](./routing-service.yaml) | wms.routes.events | Route events |
| Labor Service | [labor-service.yaml](./labor-service.yaml) | wms.labor.events | Workforce events |
| Consolidation Service | [consolidation-service.yaml](./consolidation-service.yaml) | wms.consolidation.events | Consolidation events |

## AsyncAPI Version

All specifications use **AsyncAPI 3.0.0** with the following conventions:

- **Protocol**: Apache Kafka
- **Content Type**: application/json
- **Headers**: CloudEvents 1.0 specification

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

## WMS CloudEvents Extensions

| Extension | Description |
|-----------|-------------|
| `wmscorrelationid` | Distributed tracing correlation ID |
| `wmswavenumber` | Wave number for batch processing |
| `wmsworkflowid` | Temporal workflow ID |

## Kafka Topics

| Topic | Partitions | Key | Description |
|-------|------------|-----|-------------|
| wms.orders.events | 3 | orderId | Order lifecycle |
| wms.waves.events | 3 | waveId | Wave management |
| wms.wes.events | 3 | routeId | WES execution |
| wms.walling.events | 3 | taskId | Put-wall sorting |
| wms.picking.events | 6 | orderId | Picking operations |
| wms.packing.events | 3 | orderId | Packing operations |
| wms.shipping.events | 3 | orderId | Shipping operations |
| wms.inventory.events | 6 | sku | Stock management |
| wms.consolidation.events | 3 | unitId | Consolidation |
| wms.labor.events | 3 | workerId | Workforce |
| wms.routes.events | 3 | routeId | Route calculation |

## Consumer Groups

| Service | Consumer Group | Topics |
|---------|---------------|--------|
| Waving Service | wms-waving | wms.orders.events |
| WES Service | wms-wes | wms.waves.events, wms.walling.events |
| Walling Service | wms-walling | wms.wes.events |
| Picking Service | wms-picking | wms.waves.events |
| Inventory Service | wms-inventory | wms.orders.events, wms.picking.events |
| Analytics | wms-analytics | All topics |

## Using the Specifications

### View in AsyncAPI Studio

```bash
npx @asyncapi/studio@latest
```

### Generate Documentation

```bash
npx @asyncapi/generator asyncapi.yaml @asyncapi/html-template -o docs
```

### Validate Specifications

```bash
npx @asyncapi/parser asyncapi.yaml
```

## Idempotency

Consumers must handle duplicate events using the CloudEvents `id` field. Events are delivered at-least-once, so deduplication is required for exactly-once semantics.
