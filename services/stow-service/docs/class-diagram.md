# Stow Service - Class Diagram

This diagram shows the domain model for the Stow Service bounded context.

## Domain Model

```mermaid
classDiagram
    class PutawayTask {
        <<Aggregate Root>>
        +TaskID string
        +ShipmentID string
        +SKU string
        +ProductName string
        +Quantity int
        +SourceToteID string
        +SourceLocationID string
        +TargetLocationID string
        +TargetLocation *StorageLocation
        +Strategy StorageStrategy
        +Constraints ItemConstraints
        +Status PutawayStatus
        +AssignedWorkerID string
        +Priority int
        +StowedQuantity int
        +FailureReason string
        +CreatedAt time.Time
        +UpdatedAt time.Time
        +AssignWorker(workerID string)
        +AssignLocation(location StorageLocation)
        +StartStow()
        +RecordProgress(qty int)
        +Complete()
        +Fail(reason string)
        +Cancel(reason string)
    }

    class StorageLocation {
        <<Value Object>>
        +LocationID string
        +Zone string
        +Aisle string
        +Rack string
        +Level string
        +Bin string
        +LocationType LocationType
        +Capacity float64
        +CurrentOccupancy float64
        +HasCapacity(qty int) bool
    }

    class ItemConstraints {
        <<Value Object>>
        +IsHazmat bool
        +RequiresColdChain bool
        +IsOversized bool
        +IsFragile bool
        +IsHighValue bool
        +Weight float64
        +RequiresSpecialHandling() bool
    }

    class StorageStrategy {
        <<Enumeration>>
        chaotic
        directed
        velocity
        zone_based
    }

    class PutawayStatus {
        <<Enumeration>>
        pending
        assigned
        in_progress
        completed
        cancelled
        failed
    }

    class LocationType {
        <<Enumeration>>
        pick_face
        reserve
        floor_stack
        pallet_rack
        cold_storage
        hazmat_zone
    }

    PutawayTask "1" *-- "0..1" StorageLocation : targets
    PutawayTask "1" *-- "1" ItemConstraints : has
    PutawayTask --> StorageStrategy : uses
    PutawayTask --> PutawayStatus : has status
    StorageLocation --> LocationType : has type
```

## Storage Strategy Decision

```mermaid
flowchart TD
    A[New Putaway Task] --> B{Check Constraints}
    B -->|Hazmat| C[Hazmat Zone Only]
    B -->|Cold Chain| D[Cold Storage Only]
    B -->|Standard| E{Check Strategy}
    E -->|Chaotic| F[Random Available Location]
    E -->|Directed| G[System-Assigned Location]
    E -->|Velocity| H[Based on Pick Frequency]
    E -->|Zone-Based| I[Category-Specific Zone]
    C --> J[Assign Location]
    D --> J
    F --> J
    G --> J
    H --> J
    I --> J
```

## State Transitions

```mermaid
stateDiagram-v2
    [*] --> pending
    pending --> assigned: AssignWorker()
    assigned --> in_progress: StartStow()
    in_progress --> completed: Complete()
    in_progress --> failed: Fail()

    pending --> cancelled: Cancel()
    assigned --> cancelled: Cancel()
    in_progress --> cancelled: Cancel()
```

## Repository Interface

```mermaid
classDiagram
    class PutawayTaskRepository {
        <<Interface>>
        +Save(task PutawayTask) error
        +FindByID(id string) PutawayTask
        +Update(task PutawayTask) error
        +FindByStatus(status PutawayStatus) []PutawayTask
        +FindByWorker(workerID string) []PutawayTask
        +FindByShipment(shipmentID string) []PutawayTask
        +FindPending(limit int) []PutawayTask
    }
```

## Related Diagrams

- [DDD Aggregates](ddd/aggregates.md) - Aggregate documentation
- [AsyncAPI Specification](asyncapi.yaml) - Event contracts
