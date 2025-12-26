# Waving Service - DDD Aggregates

This document describes the aggregate structure for the Waving bounded context.

## Aggregate: Wave

The Wave aggregate manages the grouping of orders for batch fulfillment.

```mermaid
graph TD
    subgraph "Wave Aggregate"
        Wave[Wave<br/><<Aggregate Root>>]

        subgraph "Entities"
            WaveOrder[WaveOrder]
        end

        subgraph "Value Objects"
            Config[WaveConfiguration]
            Labor[LaborAllocation]
            WaveType[WaveType]
            WaveStatus[WaveStatus]
        end

        Wave -->|contains| WaveOrder
        Wave -->|configured by| Config
        Wave -->|allocated| Labor
        Wave -->|type| WaveType
        Wave -->|status| WaveStatus
    end

    style Wave fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "Wave Aggregate Boundary"
        W[Wave]
        WO[WaveOrder]
        C[Configuration]
        L[LaborAllocation]
    end

    subgraph "External References"
        O[OrderID]
        Z[Zone]
    end

    WO -.->|references| O
    W -.->|assigned to| Z

    style W fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Max orders respected | Cannot exceed configured max orders |
| Max weight respected | Total weight cannot exceed max weight |
| Valid status transitions | Wave follows defined state machine |
| Released waves immutable | Cannot add orders to released wave |
| Zone consistency | All orders must be in same zone |

## Domain Events

```mermaid
graph LR
    Wave -->|emits| E1[WaveCreatedEvent]
    Wave -->|emits| E2[OrderAddedToWaveEvent]
    Wave -->|emits| E3[WaveScheduledEvent]
    Wave -->|emits| E4[WaveReleasedEvent]
    Wave -->|emits| E5[WaveCompletedEvent]
    Wave -->|emits| E6[WaveCancelledEvent]
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [Context Map](../../../../docs/diagrams/ddd/context-map.md) - Bounded context relationships
