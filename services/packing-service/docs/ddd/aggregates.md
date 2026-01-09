# Packing Service - DDD Aggregates

This document describes the aggregate structure for the Packing bounded context.

## Aggregate: PackTask

The PackTask aggregate manages the packing and labeling operation.

```mermaid
graph TD
    subgraph "PackTask Aggregate"
        Task[PackTask<br/><<Aggregate Root>>]

        subgraph "Entities"
            Item[PackItem]
            Package[Package]
        end

        subgraph "Value Objects"
            Label[ShippingLabel]
            Dimensions[Dimensions]
            TaskStatus[TaskStatus]
            PackageType[PackageType]
        end

        Task -->|contains| Item
        Task -->|creates| Package
        Task -->|status| TaskStatus
        Package -->|has| Label
        Package -->|sized| Dimensions
        Package -->|type| PackageType
    end

    style Task fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "PackTask Aggregate Boundary"
        PT[PackTask]
        PI[PackItem]
        P[Package]
        SL[ShippingLabel]
    end

    subgraph "External References"
        O[OrderID]
        W[WorkerID]
        S[Station]
        C[Carrier]
    end

    PT -.->|for| O
    PT -.->|assigned to| W
    PT -.->|at| S
    SL -.->|from| C

    style PT fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Items verified | All items must be verified before sealing |
| Package selected | Must have package type before packing |
| Label before seal | Label must be generated before sealing |
| Weight recorded | Package weight must be recorded |

## Domain Events

```mermaid
graph LR
    Task[PackTask] -->|emits| E1[PackTaskCreatedEvent]
    Task -->|emits| E2[PackagingSuggestedEvent]
    Task -->|emits| E3[PackageSealedEvent]
    Task -->|emits| E4[LabelAppliedEvent]
    Task -->|emits| E5[PackTaskCompletedEvent]
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [Packing Workflow](../../../../orchestrator/docs/diagrams/packing-workflow.md) - Workflow details
