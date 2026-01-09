# Labor Service - DDD Aggregates

This document describes the aggregate structure for the Labor bounded context.

## Aggregate: Worker

The Worker aggregate manages workforce assignments and performance.

```mermaid
graph TD
    subgraph "Worker Aggregate"
        Worker[Worker<br/><<Aggregate Root>>]

        subgraph "Entities"
            Skill[Skill]
            Shift[Shift]
            Task[TaskAssignment]
        end

        subgraph "Value Objects"
            Break[Break]
            Performance[PerformanceMetrics]
            WorkerStatus[WorkerStatus]
            SkillType[SkillType]
            ShiftType[ShiftType]
            TaskType[TaskType]
        end

        Worker -->|has| Skill
        Worker -->|works| Shift
        Worker -->|assigned| Task
        Worker -->|takes| Break
        Worker -->|tracked| Performance
        Worker -->|status| WorkerStatus
        Skill -->|type| SkillType
        Shift -->|type| ShiftType
        Task -->|type| TaskType
    end

    style Worker fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "Worker Aggregate Boundary"
        W[Worker]
        SK[Skill]
        SH[Shift]
        TA[TaskAssignment]
        B[Break]
        PM[PerformanceMetrics]
    end

    subgraph "External References"
        E[EmployeeID]
        Z[Zone]
        T[TaskID]
    end

    W -.->|identified by| E
    W -.->|in| Z
    TA -.->|references| T

    style W fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| One active shift | Worker can have only one active shift |
| One active task | Worker can have only one assigned task |
| Skill certification | Certified skills have expiration dates |
| Break limits | Maximum break time per shift enforced |
| Skill required | Cannot assign task without required skill |

## Domain Events

```mermaid
graph LR
    Worker -->|emits| E1[ShiftStartedEvent]
    Worker -->|emits| E2[ShiftEndedEvent]
    Worker -->|emits| E3[TaskAssignedEvent]
    Worker -->|emits| E4[TaskCompletedEvent]
    Worker -->|emits| E5[BreakStartedEvent]
    Worker -->|emits| E6[BreakEndedEvent]
    Worker -->|emits| E7[PerformanceRecordedEvent]
```

## Worker Availability

```mermaid
flowchart TD
    Start[Check Availability] --> Status{Worker Status?}

    Status -->|available| Skills{Has Required Skill?}
    Status -->|on_task| Busy[Not Available]
    Status -->|on_break| Break[Wait for Break End]
    Status -->|offline| Offline[Not Available]

    Skills -->|Yes| Zone{In Zone?}
    Skills -->|No| NoSkill[Not Qualified]

    Zone -->|Yes| Available[Available for Assignment]
    Zone -->|No| Move[Can Move to Zone]

    Move --> Available
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [Picking Workflow](../../../../orchestrator/docs/diagrams/picking-workflow.md) - Worker assignment
