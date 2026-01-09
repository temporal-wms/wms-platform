# Receiving Service - Class Diagram

This diagram shows the domain model for the Receiving Service bounded context.

## Domain Model

```mermaid
classDiagram
    class InboundShipment {
        <<Aggregate Root>>
        +ShipmentID string
        +ASN AdvanceShippingNotice
        +PurchaseOrderID string
        +Supplier Supplier
        +ExpectedItems []ExpectedItem
        +ReceiptRecords []ReceiptRecord
        +Discrepancies []Discrepancy
        +Status ShipmentStatus
        +ReceivingDockID string
        +AssignedWorkerID string
        +ArrivedAt *time.Time
        +CompletedAt *time.Time
        +CreatedAt time.Time
        +UpdatedAt time.Time
        +MarkArrived(dockID string)
        +StartReceiving(workerID string)
        +ReceiveItem(item ReceiveItemInput) error
        +Complete()
        +Cancel(reason string)
    }

    class AdvanceShippingNotice {
        <<Value Object>>
        +ASNID string
        +ShippingCarrier string
        +TrackingNumber string
        +EstimatedArrival time.Time
        +Validate() error
    }

    class Supplier {
        <<Value Object>>
        +SupplierID string
        +Name string
        +Code string
        +ContactEmail string
    }

    class ExpectedItem {
        <<Entity>>
        +SKU string
        +ProductName string
        +ExpectedQuantity int
        +ReceivedQuantity int
        +DamagedQuantity int
        +UnitCost float64
        +Weight float64
        +IsHazmat bool
        +RequiresColdChain bool
        +GetRemainingQuantity() int
        +IsComplete() bool
    }

    class ReceiptRecord {
        <<Entity>>
        +ReceiptID string
        +SKU string
        +ReceivedQty int
        +Condition ItemCondition
        +ToteID string
        +LocationID string
        +ReceivedBy string
        +ReceivedAt time.Time
        +Notes string
    }

    class Discrepancy {
        <<Value Object>>
        +SKU string
        +Type DiscrepancyType
        +ExpectedQty int
        +ActualQty int
        +Description string
        +DetectedAt time.Time
    }

    class ShipmentStatus {
        <<Enumeration>>
        expected
        arrived
        receiving
        inspection
        completed
        cancelled
    }

    class ItemCondition {
        <<Enumeration>>
        good
        damaged
        rejected
    }

    class DiscrepancyType {
        <<Enumeration>>
        shortage
        overage
        damage
        wrong_item
    }

    InboundShipment "1" *-- "1" AdvanceShippingNotice : has
    InboundShipment "1" *-- "1" Supplier : from
    InboundShipment "1" *-- "*" ExpectedItem : expects
    InboundShipment "1" *-- "*" ReceiptRecord : records
    InboundShipment "1" *-- "*" Discrepancy : has
    InboundShipment --> ShipmentStatus : has status
    ReceiptRecord --> ItemCondition : has condition
    Discrepancy --> DiscrepancyType : has type
```

## State Transitions

```mermaid
stateDiagram-v2
    [*] --> expected
    expected --> arrived: MarkArrived()
    arrived --> receiving: StartReceiving()
    receiving --> inspection: RequiresInspection()
    receiving --> completed: Complete() [no inspection]
    inspection --> completed: Complete()

    expected --> cancelled: Cancel()
    arrived --> cancelled: Cancel()
    receiving --> cancelled: Cancel()
    inspection --> cancelled: Cancel()
```

## Repository Interface

```mermaid
classDiagram
    class InboundShipmentRepository {
        <<Interface>>
        +Save(shipment InboundShipment) error
        +FindByID(id string) InboundShipment
        +Update(shipment InboundShipment) error
        +FindByStatus(status ShipmentStatus) []InboundShipment
        +FindExpectedArrivals(date time.Time) []InboundShipment
        +FindBySupplier(supplierID string) []InboundShipment
    }
```

## Related Diagrams

- [DDD Aggregates](ddd/aggregates.md) - Aggregate documentation
- [AsyncAPI Specification](asyncapi.yaml) - Event contracts
