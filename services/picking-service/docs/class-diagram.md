# Picking Service - Class Diagram

This diagram shows the domain model for the Picking Service bounded context.

## Domain Model

```mermaid
classDiagram
    class PickTask {
        <<Aggregate Root>>
        +TaskID string
        +OrderID string
        +WaveID string
        +RouteID string
        +WorkerID string
        +ToteID string
        +Status TaskStatus
        +Method PickMethod
        +Items []PickItem
        +Exceptions []PickException
        +CreatedAt time.Time
        +AssignedAt time.Time
        +StartedAt time.Time
        +CompletedAt time.Time
        +Assign(workerID, toteID string)
        +Start()
        +ConfirmPick(sku string, qty int, locationID string)
        +ReportException(exception PickException)
        +ResolveException(exceptionID string)
        +Complete()
        +Cancel()
    }

    class PickItem {
        <<Entity>>
        +SKU string
        +ProductName string
        +Quantity int
        +PickedQty int
        +LocationID string
        +Location Location
        +Status ItemStatus
        +ToteID string
        +IsFullyPicked() bool
    }

    class PickException {
        <<Entity>>
        +ExceptionID string
        +ItemSKU string
        +Type ExceptionType
        +Description string
        +ReportedAt time.Time
        +ResolvedAt time.Time
        +ResolvedBy string
        +Resolution string
    }

    class Location {
        <<Value Object>>
        +LocationID string
        +Zone string
        +Aisle string
        +Rack string
        +Level string
    }

    class TaskStatus {
        <<Enumeration>>
        pending
        assigned
        in_progress
        completed
        cancelled
        exception
    }

    class PickMethod {
        <<Enumeration>>
        single
        batch
        zone
        wave
    }

    class ExceptionType {
        <<Enumeration>>
        item_not_found
        damaged
        quantity_mismatch
        wrong_item
        location_empty
    }

    class ItemStatus {
        <<Enumeration>>
        pending
        picked
        short
        skipped
    }

    PickTask "1" *-- "*" PickItem : contains
    PickTask "1" *-- "*" PickException : has
    PickItem "1" *-- "1" Location : at
    PickTask --> TaskStatus : has
    PickTask --> PickMethod : uses
    PickException --> ExceptionType : type of
    PickItem --> ItemStatus : has
```

## Pick Item Flow

```mermaid
stateDiagram-v2
    [*] --> pending: Item Added
    pending --> picked: ConfirmPick()
    pending --> short: Short Pick
    pending --> skipped: Skip Item
    picked --> [*]
    short --> [*]
    skipped --> [*]
```

## Related Diagrams

- [Aggregate Diagram](ddd/aggregates.md) - DDD aggregate structure
- [Picking Workflow](../../../orchestrator/docs/diagrams/picking-workflow.md) - Workflow details
