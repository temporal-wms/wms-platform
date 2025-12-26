# Waving Service - Class Diagram

This diagram shows the domain model for the Waving Service bounded context.

## Domain Model

```mermaid
classDiagram
    class Wave {
        <<Aggregate Root>>
        +WaveID string
        +WaveNumber string
        +Type WaveType
        +Status WaveStatus
        +FulfillmentMode FulfillmentMode
        +ScheduledStart time.Time
        +ActualStart time.Time
        +CompletedAt time.Time
        +Zone string
        +Priority int
        +Orders []WaveOrder
        +LaborAllocation LaborAllocation
        +Configuration WaveConfiguration
        +AddOrder(order WaveOrder) error
        +RemoveOrder(orderID string)
        +Schedule(startTime time.Time)
        +Release()
        +Complete()
        +Cancel()
        +AllocateLabor(workers int)
    }

    class WaveOrder {
        <<Entity>>
        +OrderID string
        +CustomerID string
        +ItemCount int
        +TotalWeight float64
        +Priority Priority
        +PromisedDelivery time.Time
        +Status string
        +AddedAt time.Time
        +CompletedAt time.Time
    }

    class WaveConfiguration {
        <<Value Object>>
        +MaxOrders int
        +MaxItems int
        +MaxWeight float64
        +ZoneFilter []string
        +PriorityFilter []Priority
        +CarrierFilter []string
        +AutoRelease bool
    }

    class LaborAllocation {
        <<Value Object>>
        +Pickers int
        +Packers int
        +Zone string
        +EstimatedDuration time.Duration
    }

    class WaveType {
        <<Enumeration>>
        digital
        wholesale
        priority
        mixed
    }

    class WaveStatus {
        <<Enumeration>>
        planning
        scheduled
        released
        in_progress
        completed
        cancelled
    }

    class FulfillmentMode {
        <<Enumeration>>
        wave
        waveless
        hybrid
    }

    Wave "1" *-- "*" WaveOrder : contains
    Wave "1" *-- "1" WaveConfiguration : configured by
    Wave "1" *-- "1" LaborAllocation : allocated
    Wave --> WaveType : has type
    Wave --> WaveStatus : has status
    Wave --> FulfillmentMode : uses mode
```

## Wave Lifecycle

```mermaid
stateDiagram-v2
    [*] --> planning: Create Wave
    planning --> planning: Add/Remove Orders
    planning --> scheduled: Schedule()
    scheduled --> released: Release()
    released --> in_progress: First Pick Started
    in_progress --> in_progress: Order Completed
    in_progress --> completed: All Orders Done

    planning --> cancelled: Cancel()
    scheduled --> cancelled: Cancel()
    released --> cancelled: Cancel()
```

## Repository Interface

```mermaid
classDiagram
    class WaveRepository {
        <<Interface>>
        +Create(wave Wave) error
        +GetByID(id string) Wave
        +Update(wave Wave) error
        +FindByStatus(status WaveStatus) []Wave
        +FindByZone(zone string) []Wave
        +FindScheduledBefore(time time.Time) []Wave
        +FindActiveWaves() []Wave
    }
```

## Related Diagrams

- [Aggregate Diagram](ddd/aggregates.md) - DDD aggregate structure
- [Order Fulfillment Flow](../../../docs/diagrams/order-fulfillment-flow.md) - Wave assignment step
