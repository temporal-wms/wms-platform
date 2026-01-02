# Stow Service - DDD Aggregates

This document describes the aggregate structure for the Stow bounded context following Domain-Driven Design principles.

## Aggregate: PutawayTask

The PutawayTask aggregate represents a task to store received items in warehouse locations using various storage strategies.

```mermaid
graph TD
    subgraph "PutawayTask Aggregate"
        PutawayTask[PutawayTask<br/><<Aggregate Root>>]

        subgraph "Value Objects"
            StorageLocation[StorageLocation]
            ItemConstraints[ItemConstraints]
            Strategy[StorageStrategy]
            Status[PutawayStatus]
        end

        PutawayTask -->|targets| StorageLocation
        PutawayTask -->|has| ItemConstraints
        PutawayTask -->|uses| Strategy
        PutawayTask -->|has| Status
    end

    style PutawayTask fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "PutawayTask Aggregate Boundary"
        PT[PutawayTask]
        SL[StorageLocation]
        IC[ItemConstraints]
    end

    subgraph "External References"
        SHIP[ShipmentID]
        WORKER[AssignedWorkerID]
        TOTE[SourceToteID]
    end

    PT -.->|from| SHIP
    PT -.->|assigned to| WORKER
    PT -.->|from| TOTE

    style PT fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Valid SKU required | Task must have valid SKU and quantity |
| Quantity positive | Quantity must be > 0 |
| Status transitions valid | Status can only change according to state machine |
| Strategy immutable | Storage strategy cannot change after creation |
| Location matches constraints | Target location must support item constraints |
| Stowed qty <= total qty | Cannot stow more than total quantity |

## Storage Strategy Rules

| Strategy | Rule |
|----------|------|
| Chaotic | Random available location (Amazon-style) - maximizes space utilization |
| Directed | System assigns specific location based on predefined rules |
| Velocity | High-velocity SKUs near pick zones, slow-movers in reserve |
| Zone-Based | Products grouped by category/type in designated zones |

## Domain Events

```mermaid
graph LR
    PutawayTask -->|emits| E1[PutawayTaskCreatedEvent]
    PutawayTask -->|emits| E2[PutawayTaskAssignedEvent]
    PutawayTask -->|emits| E3[LocationAssignedEvent]
    PutawayTask -->|emits| E4[ItemStowedEvent]
    PutawayTask -->|emits| E5[PutawayTaskCompletedEvent]
    PutawayTask -->|emits| E6[PutawayTaskFailedEvent]
```

## Event Details

| Event | Trigger | Payload |
|-------|---------|---------|
| PutawayTaskCreatedEvent | Task created from receiving | taskId, shipmentId, sku, quantity, strategy |
| PutawayTaskAssignedEvent | Worker assigned to task | taskId, workerId, assignedAt |
| LocationAssignedEvent | Target location determined | taskId, locationId, zone, strategy |
| ItemStowedEvent | Items placed in location | taskId, sku, quantity, locationId |
| PutawayTaskCompletedEvent | All items stowed | taskId, totalStowed, completedAt |
| PutawayTaskFailedEvent | Task cannot be completed | taskId, reason, failedAt |

## Factory Pattern

```mermaid
classDiagram
    class PutawayTaskFactory {
        +CreateFromReceiving(shipmentId, sku, qty, constraints) PutawayTask
        +CreateWithStrategy(strategy, constraints) PutawayTask
        +ReconstituteFromEvents(events) PutawayTask
    }

    class PutawayTask {
        -constructor()
        +static Create() PutawayTask
    }

    PutawayTaskFactory --> PutawayTask : creates
```

## Repository Pattern

```mermaid
classDiagram
    class PutawayTaskRepository {
        <<Interface>>
        +Save(task PutawayTask)
        +FindByID(id string) PutawayTask
        +FindByStatus(status PutawayStatus) []PutawayTask
        +FindByWorker(workerID string) []PutawayTask
        +FindByShipment(shipmentID string) []PutawayTask
        +FindPending(limit int) []PutawayTask
    }

    class MongoPutawayTaskRepository {
        +Save(task PutawayTask)
        +FindByID(id string) PutawayTask
        +FindByStatus(status PutawayStatus) []PutawayTask
        +FindByWorker(workerID string) []PutawayTask
        +FindByShipment(shipmentID string) []PutawayTask
        +FindPending(limit int) []PutawayTask
    }

    PutawayTaskRepository <|.. MongoPutawayTaskRepository
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [AsyncAPI Specification](../asyncapi.yaml) - Event contracts
