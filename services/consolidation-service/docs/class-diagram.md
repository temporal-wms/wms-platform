# Consolidation Service - Class Diagram

This diagram shows the domain model for the Consolidation Service bounded context.

## Domain Model

```mermaid
classDiagram
    class ConsolidationUnit {
        <<Aggregate Root>>
        +ConsolidationID string
        +OrderID string
        +WaveID string
        +Status UnitStatus
        +Strategy ConsolidationStrategy
        +Station string
        +ExpectedItems []ExpectedItem
        +ConsolidatedItems []ConsolidatedItem
        +DestinationBin string
        +CreatedAt time.Time
        +StartedAt time.Time
        +CompletedAt time.Time
        +AssignStation(station string)
        +Start()
        +ConsolidateItem(sku string, sourceTote string)
        +MarkShort(sku string, reason string)
        +Complete()
        +Cancel()
        +GetProgress() float64
    }

    class ExpectedItem {
        <<Entity>>
        +SKU string
        +ProductName string
        +Quantity int
        +SourceToteID string
        +Status ExpectedStatus
        +ReceivedQty int
    }

    class ConsolidatedItem {
        <<Entity>>
        +SKU string
        +Quantity int
        +SourceToteID string
        +ScannedAt time.Time
        +VerifiedBy string
    }

    class UnitStatus {
        <<Enumeration>>
        pending
        in_progress
        completed
        cancelled
    }

    class ConsolidationStrategy {
        <<Enumeration>>
        order
        carrier
        route
        time
    }

    class ExpectedStatus {
        <<Enumeration>>
        pending
        partial
        received
        short
    }

    ConsolidationUnit "1" *-- "*" ExpectedItem : expects
    ConsolidationUnit "1" *-- "*" ConsolidatedItem : received
    ConsolidationUnit --> UnitStatus : has
    ConsolidationUnit --> ConsolidationStrategy : uses
    ExpectedItem --> ExpectedStatus : has
```

## Consolidation Flow

```mermaid
stateDiagram-v2
    [*] --> pending: Unit Created
    pending --> in_progress: Start()
    in_progress --> in_progress: ConsolidateItem()
    in_progress --> completed: All Items Received
    in_progress --> completed: Complete() [with shorts]
    completed --> [*]
    pending --> cancelled: Cancel()
    in_progress --> cancelled: Cancel()
    cancelled --> [*]
```

## Related Diagrams

- [Aggregate Diagram](ddd/aggregates.md) - DDD aggregate structure
- [Consolidation Workflow](../../../orchestrator/docs/diagrams/consolidation-workflow.md) - Workflow details
