# Routing Service - Class Diagram

This diagram shows the domain model for the Routing Service bounded context.

## Domain Model

```mermaid
classDiagram
    class PickRoute {
        <<Aggregate Root>>
        +RouteID string
        +OrderID string
        +WaveID string
        +PickerID string
        +Status RouteStatus
        +Strategy RoutingStrategy
        +Stops []RouteStop
        +EstimatedDistance float64
        +ActualDistance float64
        +EstimatedTime time.Duration
        +ActualTime time.Duration
        +StartedAt time.Time
        +CompletedAt time.Time
        +OptimizeRoute()
        +Start()
        +CompleteStop(stopNumber int)
        +SkipStop(stopNumber int, reason string)
        +Complete()
        +Pause()
        +Cancel()
    }

    class RouteStop {
        <<Entity>>
        +StopNumber int
        +LocationID string
        +Location Location
        +SKU string
        +ProductName string
        +Quantity int
        +PickedQty int
        +Status StopStatus
        +ToteID string
        +SkipReason string
        +CompletedAt time.Time
    }

    class Location {
        <<Value Object>>
        +LocationID string
        +Zone string
        +Aisle string
        +Rack string
        +Level string
        +Position string
        +X int
        +Y int
        +DistanceFrom(other Location) float64
        +IsSameAisle(other Location) bool
        +IsSameZone(other Location) bool
    }

    class RoutingStrategy {
        <<Enumeration>>
        return
        s_shape
        largest_gap
        combined
        nearest
    }

    class RouteStatus {
        <<Enumeration>>
        pending
        in_progress
        paused
        completed
        cancelled
    }

    class StopStatus {
        <<Enumeration>>
        pending
        in_progress
        completed
        skipped
    }

    PickRoute "1" *-- "*" RouteStop : contains
    RouteStop "1" *-- "1" Location : at
    PickRoute --> RoutingStrategy : uses
    PickRoute --> RouteStatus : has status
    RouteStop --> StopStatus : has status
```

## Routing Strategies

```mermaid
graph TD
    subgraph "S-Shape Strategy"
        A1[Start] --> A2[Aisle 1 Full]
        A2 --> A3[Aisle 2 Full]
        A3 --> A4[Aisle 3 Full]
        A4 --> A5[End]
    end

    subgraph "Return Strategy"
        B1[Start] --> B2[Aisle 1 In]
        B2 --> B3[Aisle 1 Out]
        B3 --> B4[Aisle 2 In]
        B4 --> B5[Aisle 2 Out]
        B5 --> B6[End]
    end

    subgraph "Largest Gap Strategy"
        C1[Start] --> C2[Skip Empty Section]
        C2 --> C3[Pick Items]
        C3 --> C4[Skip Empty Section]
        C4 --> C5[End]
    end
```

## Distance Calculation

```mermaid
flowchart TD
    Start[Start Position] --> Calc[Calculate Distance]
    Calc --> Type{Strategy?}

    Type -->|S-Shape| SS[Full Aisle Traversal]
    Type -->|Return| RT[In-Out Each Aisle]
    Type -->|Largest Gap| LG[Skip Empty Sections]
    Type -->|Nearest| NN[Greedy Nearest]

    SS --> Total[Sum Total Distance]
    RT --> Total
    LG --> Total
    NN --> Total

    Total --> Time[Estimate Time]
    Time --> Result[Route Optimized]
```

## Related Diagrams

- [Aggregate Diagram](ddd/aggregates.md) - DDD aggregate structure
- [Picking Workflow](../../../orchestrator/docs/diagrams/picking-workflow.md) - Uses calculated routes
