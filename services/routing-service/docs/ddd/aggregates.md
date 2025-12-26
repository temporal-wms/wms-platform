# Routing Service - DDD Aggregates

This document describes the aggregate structure for the Routing bounded context.

## Aggregate: PickRoute

The PickRoute aggregate manages the optimized sequence of locations for picking.

```mermaid
graph TD
    subgraph "PickRoute Aggregate"
        Route[PickRoute<br/><<Aggregate Root>>]

        subgraph "Entities"
            Stop[RouteStop]
        end

        subgraph "Value Objects"
            Location[Location]
            Strategy[RoutingStrategy]
            RouteStatus[RouteStatus]
            StopStatus[StopStatus]
        end

        Route -->|contains| Stop
        Route -->|uses| Strategy
        Route -->|has| RouteStatus
        Stop -->|at| Location
        Stop -->|has| StopStatus
    end

    style Route fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "PickRoute Aggregate Boundary"
        R[PickRoute]
        RS[RouteStop]
        L[Location]
    end

    subgraph "External References"
        O[OrderID]
        W[WaveID]
        P[PickerID]
    end

    R -.->|for| O
    R -.->|in| W
    R -.->|assigned to| P

    style R fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Ordered stops | Stops must be in sequence order |
| Valid locations | All locations must exist |
| Single active route | One active route per order |
| Complete before finish | All stops must be completed or skipped |

## Domain Events

```mermaid
graph LR
    Route[PickRoute] -->|emits| E1[RouteCreatedEvent]
    Route -->|emits| E2[RouteOptimizedEvent]
    Route -->|emits| E3[RouteStartedEvent]
    Route -->|emits| E4[StopCompletedEvent]
    Route -->|emits| E5[RouteCompletedEvent]
    Route -->|emits| E6[RouteCancelledEvent]
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [Context Map](../../../../docs/diagrams/ddd/context-map.md) - Bounded context relationships
