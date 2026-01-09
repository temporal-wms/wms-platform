# Picking Service - DDD Aggregates

This document describes the aggregate structure for the Picking bounded context.

## Aggregate: PickTask

The PickTask aggregate manages the picking operation for an order.

```mermaid
graph TD
    subgraph "PickTask Aggregate"
        Task[PickTask<br/><<Aggregate Root>>]

        subgraph "Entities"
            Item[PickItem]
            Exception[PickException]
        end

        subgraph "Value Objects"
            Location[Location]
            TaskStatus[TaskStatus]
            PickMethod[PickMethod]
            ItemStatus[ItemStatus]
            ExceptionType[ExceptionType]
        end

        Task -->|contains| Item
        Task -->|has| Exception
        Task -->|status| TaskStatus
        Task -->|method| PickMethod
        Item -->|at| Location
        Item -->|status| ItemStatus
        Exception -->|type| ExceptionType
    end

    style Task fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "PickTask Aggregate Boundary"
        PT[PickTask]
        PI[PickItem]
        PE[PickException]
    end

    subgraph "External References"
        O[OrderID]
        W[WaveID]
        R[RouteID]
        WK[WorkerID]
        T[ToteID]
    end

    PT -.->|for| O
    PT -.->|in| W
    PT -.->|follows| R
    PT -.->|assigned to| WK
    PT -.->|uses| T

    style PT fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Worker required | Cannot start without assigned worker |
| Tote required | Cannot pick without assigned tote |
| Quantity valid | Picked quantity <= expected quantity |
| Exception resolution | Exceptions must be resolved to complete |

## Domain Events

```mermaid
graph LR
    Task[PickTask] -->|emits| E1[PickTaskCreatedEvent]
    Task -->|emits| E2[PickTaskAssignedEvent]
    Task -->|emits| E3[ItemPickedEvent]
    Task -->|emits| E4[PickExceptionEvent]
    Task -->|emits| E5[PickTaskCompletedEvent]
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [Picking Workflow](../../../../orchestrator/docs/diagrams/picking-workflow.md) - Workflow details
