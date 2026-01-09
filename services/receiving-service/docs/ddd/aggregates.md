# Receiving Service - DDD Aggregates

This document describes the aggregate structure for the Receiving bounded context following Domain-Driven Design principles.

## Aggregate: InboundShipment

The InboundShipment aggregate represents an incoming shipment from a supplier and tracks the receiving process.

```mermaid
graph TD
    subgraph "InboundShipment Aggregate"
        InboundShipment[InboundShipment<br/><<Aggregate Root>>]

        subgraph "Entities"
            ExpectedItem[ExpectedItem]
            ReceiptRecord[ReceiptRecord]
        end

        subgraph "Value Objects"
            ASN[AdvanceShippingNotice]
            Supplier[Supplier]
            Discrepancy[Discrepancy]
            Status[ShipmentStatus]
        end

        InboundShipment -->|has| ASN
        InboundShipment -->|from| Supplier
        InboundShipment -->|expects| ExpectedItem
        InboundShipment -->|records| ReceiptRecord
        InboundShipment -->|detects| Discrepancy
        InboundShipment -->|has| Status
    end

    style InboundShipment fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "InboundShipment Aggregate Boundary"
        IS[InboundShipment]
        EI[ExpectedItem]
        RR[ReceiptRecord]
        ASN[ASN]
        SUP[Supplier]
    end

    subgraph "External References"
        PO[PurchaseOrderID]
        DOCK[ReceivingDockID]
        WORKER[AssignedWorkerID]
    end

    IS -.->|references| PO
    IS -.->|at| DOCK
    IS -.->|assigned to| WORKER

    style IS fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Valid ASN required | Shipment must have valid ASN with tracking info |
| Supplier required | Shipment must have supplier information |
| Expected items exist | Cannot receive items without expected items list |
| Status transitions valid | Status can only change according to state machine |
| Received qty <= expected | Cannot receive more than expected (logs overage discrepancy) |
| Condition tracking | Every received item must have condition recorded |

## Domain Events

```mermaid
graph LR
    InboundShipment -->|emits| E1[ShipmentExpectedEvent]
    InboundShipment -->|emits| E2[ShipmentArrivedEvent]
    InboundShipment -->|emits| E3[ItemReceivedEvent]
    InboundShipment -->|emits| E4[ReceivingCompletedEvent]
    InboundShipment -->|emits| E5[ReceivingDiscrepancyEvent]
    InboundShipment -->|emits| E6[PutawayTaskCreatedEvent]
```

## Event Details

| Event | Trigger | Payload |
|-------|---------|---------|
| ShipmentExpectedEvent | New shipment created | shipmentId, supplierId, expectedArrival, items |
| ShipmentArrivedEvent | Shipment arrives at dock | shipmentId, dockId, arrivedAt |
| ItemReceivedEvent | Item scanned/received | shipmentId, sku, quantity, condition, toteId |
| ReceivingCompletedEvent | All items received | shipmentId, totalReceived, totalDamaged |
| ReceivingDiscrepancyEvent | Discrepancy detected | shipmentId, type, sku, expected, actual |
| PutawayTaskCreatedEvent | Putaway triggered | shipmentId, taskId, sku, quantity, toteId |

## Factory Pattern

```mermaid
classDiagram
    class InboundShipmentFactory {
        +CreateFromASN(asn, supplier, items) InboundShipment
        +ReconstituteFromEvents(events) InboundShipment
    }

    class InboundShipment {
        -constructor()
        +static Create() InboundShipment
    }

    InboundShipmentFactory --> InboundShipment : creates
```

## Repository Pattern

```mermaid
classDiagram
    class InboundShipmentRepository {
        <<Interface>>
        +Save(shipment InboundShipment)
        +FindByID(id string) InboundShipment
        +FindByStatus(status ShipmentStatus) []InboundShipment
        +FindExpectedArrivals(date time.Time) []InboundShipment
        +FindBySupplier(supplierID string) []InboundShipment
    }

    class MongoInboundShipmentRepository {
        +Save(shipment InboundShipment)
        +FindByID(id string) InboundShipment
        +FindByStatus(status ShipmentStatus) []InboundShipment
        +FindExpectedArrivals(date time.Time) []InboundShipment
        +FindBySupplier(supplierID string) []InboundShipment
    }

    InboundShipmentRepository <|.. MongoInboundShipmentRepository
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [AsyncAPI Specification](../asyncapi.yaml) - Event contracts
