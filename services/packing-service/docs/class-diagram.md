# Packing Service - Class Diagram

This diagram shows the domain model for the Packing Service bounded context.

## Domain Model

```mermaid
classDiagram
    class PackTask {
        <<Aggregate Root>>
        +TaskID string
        +OrderID string
        +Status TaskStatus
        +WorkerID string
        +Station string
        +Items []PackItem
        +Package Package
        +CreatedAt time.Time
        +StartedAt time.Time
        +CompletedAt time.Time
        +Assign(workerID, station string)
        +Start()
        +VerifyItem(sku string)
        +SelectPackaging(packageType string)
        +SealPackage()
        +ApplyLabel(label ShippingLabel)
        +Complete()
        +Cancel()
    }

    class PackItem {
        <<Entity>>
        +SKU string
        +ProductName string
        +Quantity int
        +Weight float64
        +Fragile bool
        +Verified bool
        +VerifiedAt time.Time
    }

    class Package {
        <<Entity>>
        +PackageID string
        +Type PackageType
        +Dimensions Dimensions
        +TotalWeight float64
        +Materials []string
        +Sealed bool
        +SealedAt time.Time
    }

    class ShippingLabel {
        <<Value Object>>
        +TrackingNumber string
        +Carrier string
        +ServiceType string
        +LabelFormat string
        +LabelData []byte
        +GeneratedAt time.Time
    }

    class Dimensions {
        <<Value Object>>
        +Length float64
        +Width float64
        +Height float64
        +Volume() float64
    }

    class TaskStatus {
        <<Enumeration>>
        pending
        in_progress
        packed
        labeled
        completed
        cancelled
    }

    class PackageType {
        <<Enumeration>>
        box
        envelope
        bag
        padded_envelope
        custom
    }

    PackTask "1" *-- "*" PackItem : contains
    PackTask "1" *-- "1" Package : creates
    Package "1" *-- "1" ShippingLabel : has
    Package "1" *-- "1" Dimensions : sized
    PackTask --> TaskStatus : has
    Package --> PackageType : type of
```

## Pack Task Flow

```mermaid
stateDiagram-v2
    [*] --> pending: Task Created
    pending --> in_progress: Start()
    in_progress --> packed: SealPackage()
    packed --> labeled: ApplyLabel()
    labeled --> completed: Complete()
    completed --> [*]
    pending --> cancelled: Cancel()
    in_progress --> cancelled: Cancel()
    cancelled --> [*]
```

## Related Diagrams

- [Aggregate Diagram](ddd/aggregates.md) - DDD aggregate structure
- [Packing Workflow](../../../orchestrator/docs/diagrams/packing-workflow.md) - Workflow details
