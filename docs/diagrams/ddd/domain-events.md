# WMS Platform - Domain Events Flow

This document shows the domain events and their flow across bounded contexts.

## Event Flow Overview

```mermaid
sequenceDiagram
    autonumber
    participant Order as Order Context
    participant Kafka as Apache Kafka
    participant Waving as Waving Context
    participant Inventory as Inventory Context
    participant Routing as Routing Context
    participant Picking as Picking Context
    participant Consolidation as Consolidation Context
    participant Packing as Packing Context
    participant Shipping as Shipping Context
    participant Labor as Labor Context

    Note over Order,Labor: Order Fulfillment Event Flow

    Order->>Kafka: OrderReceivedEvent
    Kafka->>Waving: Subscribe
    Waving->>Kafka: OrderAddedToWaveEvent

    Order->>Kafka: OrderValidatedEvent
    Kafka->>Inventory: Subscribe
    Inventory->>Kafka: InventoryReservedEvent

    Waving->>Kafka: WaveReleasedEvent
    Kafka->>Routing: Subscribe
    Kafka->>Picking: Subscribe

    Routing->>Kafka: RouteCalculatedEvent
    Kafka->>Picking: Subscribe

    Labor->>Kafka: TaskAssignedEvent
    Kafka->>Picking: Subscribe

    Picking->>Kafka: ItemPickedEvent
    Picking->>Kafka: PickTaskCompletedEvent
    Kafka->>Consolidation: Subscribe
    Kafka->>Inventory: Subscribe

    Consolidation->>Kafka: ConsolidationCompletedEvent
    Kafka->>Packing: Subscribe

    Packing->>Kafka: PackageSealedEvent
    Packing->>Kafka: LabelAppliedEvent
    Kafka->>Shipping: Subscribe

    Shipping->>Kafka: ShipConfirmedEvent
    Kafka->>Order: Subscribe
    Order->>Kafka: OrderShippedEvent
```

## Event Categories

### Order Events

```mermaid
graph LR
    subgraph "Order Context Events"
        OR[OrderReceivedEvent]
        OV[OrderValidatedEvent]
        OWA[OrderWaveAssignedEvent]
        OS[OrderShippedEvent]
        OC[OrderCancelledEvent]
        OD[OrderCompletedEvent]
    end

    OR --> OV
    OV --> OWA
    OWA --> OS
    OS --> OD

    OV -.->|failure| OC
    OWA -.->|timeout| OC
```

### Wave Events

```mermaid
graph LR
    subgraph "Waving Context Events"
        WC[WaveCreatedEvent]
        OA[OrderAddedToWaveEvent]
        WS[WaveScheduledEvent]
        WR[WaveReleasedEvent]
        WCO[WaveCompletedEvent]
        WCA[WaveCancelledEvent]
    end

    WC --> OA
    OA --> WS
    WS --> WR
    WR --> WCO

    WS -.->|cancelled| WCA
```

### Picking Events

```mermaid
graph LR
    subgraph "Picking Context Events"
        PTC[PickTaskCreatedEvent]
        PTA[PickTaskAssignedEvent]
        IP[ItemPickedEvent]
        PE[PickExceptionEvent]
        PTE[PickTaskCompletedEvent]
    end

    PTC --> PTA
    PTA --> IP
    IP --> IP
    IP --> PTE
    IP -.->|problem| PE
    PE --> IP
```

### Shipping Events

```mermaid
graph LR
    subgraph "Shipping Context Events"
        SC[ShipmentCreatedEvent]
        LG[LabelGeneratedEvent]
        SM[ShipmentManifestedEvent]
        SC2[ShipConfirmedEvent]
        DC[DeliveryConfirmedEvent]
    end

    SC --> LG
    LG --> SM
    SM --> SC2
    SC2 --> DC
```

## Event Catalog

### All Domain Events (58 Total)

| Context | Event Type | Kafka Topic | Description |
|---------|------------|-------------|-------------|
| **Order** | wms.order.received | wms.orders.events | Order placed |
| | wms.order.validated | wms.orders.events | Validation passed |
| | wms.order.wave-assigned | wms.orders.events | Assigned to wave |
| | wms.order.shipped | wms.orders.events | Order shipped |
| | wms.order.cancelled | wms.orders.events | Order cancelled |
| | wms.order.completed | wms.orders.events | Delivery confirmed |
| **Wave** | wms.wave.created | wms.waves.events | Wave created |
| | wms.wave.order-added | wms.waves.events | Order added |
| | wms.wave.scheduled | wms.waves.events | Wave scheduled |
| | wms.wave.released | wms.waves.events | Released to picking |
| | wms.wave.completed | wms.waves.events | All orders done |
| | wms.wave.cancelled | wms.waves.events | Wave cancelled |
| **Routing** | wms.routing.route-calculated | wms.routes.events | Route optimized |
| | wms.routing.route-started | wms.routes.events | Picking started |
| | wms.routing.stop-completed | wms.routes.events | Stop picked |
| | wms.routing.route-completed | wms.routes.events | Route finished |
| **Picking** | wms.picking.task-created | wms.picking.events | Task created |
| | wms.picking.task-assigned | wms.picking.events | Worker assigned |
| | wms.picking.item-picked | wms.picking.events | Item picked |
| | wms.picking.exception | wms.picking.events | Problem reported |
| | wms.picking.task-completed | wms.picking.events | Task done |
| **Consolidation** | wms.consolidation.started | wms.consolidation.events | Started |
| | wms.consolidation.item-consolidated | wms.consolidation.events | Item scanned |
| | wms.consolidation.completed | wms.consolidation.events | All items done |
| **Packing** | wms.packing.task-created | wms.packing.events | Task created |
| | wms.packing.packaging-suggested | wms.packing.events | Package selected |
| | wms.packing.package-sealed | wms.packing.events | Package sealed |
| | wms.packing.label-applied | wms.packing.events | Label affixed |
| | wms.packing.task-completed | wms.packing.events | Task done |
| **Shipping** | wms.shipping.shipment-created | wms.shipping.events | Shipment created |
| | wms.shipping.label-generated | wms.shipping.events | Label printed |
| | wms.shipping.manifested | wms.shipping.events | Added to manifest |
| | wms.shipping.confirmed | wms.shipping.events | Shipped |
| **Inventory** | wms.inventory.received | wms.inventory.events | Stock received |
| | wms.inventory.reserved | wms.inventory.events | Stock reserved |
| | wms.inventory.picked | wms.inventory.events | Stock picked |
| | wms.inventory.adjusted | wms.inventory.events | Stock adjusted |
| | wms.inventory.low-stock-alert | wms.inventory.events | Low stock |
| **Labor** | wms.labor.shift-started | wms.labor.events | Shift began |
| | wms.labor.shift-ended | wms.labor.events | Shift ended |
| | wms.labor.task-assigned | wms.labor.events | Task assigned |
| | wms.labor.task-completed | wms.labor.events | Task done |
| | wms.labor.performance-recorded | wms.labor.events | Metrics updated |

## Event Structure (CloudEvents 1.0)

```mermaid
classDiagram
    class WMSCloudEvent {
        +specversion: "1.0"
        +type: string
        +source: string
        +subject: string
        +id: string
        +time: timestamp
        +datacontenttype: "application/json"
        +data: object
    }

    class Extensions {
        +correlationid: string
        +wavenumber: string
        +workflowid: string
    }

    WMSCloudEvent --> Extensions : has
```

## Event Publishing Pattern

```mermaid
graph TD
    subgraph "Transactional Outbox Pattern"
        A[Domain Operation] --> B[Create Event]
        B --> C[Save to Outbox Table]
        C --> D[Commit Transaction]

        E[Outbox Publisher] --> F[Poll Outbox]
        F --> G[Publish to Kafka]
        G --> H[Mark as Published]
    end

    style C fill:#ffffcc
    style G fill:#ccffcc
```

## Event Consumers

```mermaid
graph TB
    subgraph "Kafka Topics"
        T1[wms.orders.events]
        T2[wms.waves.events]
        T3[wms.picking.events]
        T4[wms.inventory.events]
    end

    subgraph "Consumers"
        C1[Waving Service]
        C2[Picking Service]
        C3[Inventory Service]
        C4[Analytics Service]
    end

    T1 --> C1
    T1 --> C4
    T2 --> C2
    T2 --> C4
    T3 --> C3
    T3 --> C4
    T4 --> C4
```

## Saga Compensation Events

```mermaid
sequenceDiagram
    participant Order
    participant Kafka
    participant Inventory
    participant Picking

    Note over Order,Picking: Compensation on Failure

    Order->>Kafka: OrderReceivedEvent
    Kafka->>Inventory: Reserve Stock
    Inventory->>Kafka: InventoryReservedEvent

    Picking->>Kafka: PickTaskFailedEvent
    Kafka->>Inventory: Subscribe

    Note over Inventory: Compensation
    Inventory->>Kafka: ReservationReleasedEvent
    Kafka->>Order: Subscribe
    Order->>Kafka: OrderCancelledEvent
```

## Related Documentation

- [Context Map](context-map.md) - Bounded context relationships
- [Ecosystem](../ecosystem.md) - Platform architecture
- [Order Fulfillment Flow](../order-fulfillment-flow.md) - Workflow integration
