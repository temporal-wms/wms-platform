# Facility Service - Class Diagram

This diagram shows the domain model for the Facility Service bounded context.

## Domain Model

```mermaid
classDiagram
    class Station {
        <<Aggregate Root>>
        +StationID string
        +Name string
        +Zone string
        +StationType StationType
        +Status StationStatus
        +Capabilities []StationCapability
        +MaxConcurrentTasks int
        +CurrentTasks int
        +AssignedWorkerID string
        +Equipment []StationEquipment
        +CreatedAt time.Time
        +UpdatedAt time.Time
        +HasCapability(cap StationCapability) bool
        +HasAllCapabilities(caps []StationCapability) bool
        +AddCapability(cap StationCapability)
        +RemoveCapability(cap StationCapability)
        +SetCapabilities(caps []StationCapability)
        +CanAcceptTask() bool
        +IncrementTasks()
        +DecrementTasks()
        +AssignWorker(workerID string)
        +UnassignWorker()
        +Activate()
        +Deactivate()
        +SetMaintenance()
        +GetAvailableCapacity() int
    }

    class StationEquipment {
        <<Entity>>
        +EquipmentID string
        +EquipmentType string
        +Status string
    }

    class StationType {
        <<Enumeration>>
        packing
        consolidation
        shipping
        receiving
        stow
        slam
        sortation
        qc
    }

    class StationStatus {
        <<Enumeration>>
        active
        inactive
        maintenance
    }

    class StationCapability {
        <<Enumeration>>
        single_item
        multi_item
        gift_wrap
        hazmat
        oversized
        fragile
        cold_chain
        high_value
    }

    class EquipmentType {
        <<Enumeration>>
        scale
        printer
        cold_storage
        hazmat_cabinet
        scanner
    }

    Station "1" *-- "*" StationEquipment : has
    Station "1" *-- "*" StationCapability : supports
    Station --> StationType : has type
    Station --> StationStatus : has status
    StationEquipment --> EquipmentType : has type
```

## Process Path Routing

```mermaid
flowchart TD
    A[Order Requirements] --> B{Analyze Requirements}
    B --> C[Extract Required Capabilities]
    C --> D{Find Matching Stations}
    D --> E[Filter by Type]
    E --> F[Filter by Zone]
    F --> G[Check Availability]
    G --> H{Station Available?}
    H -->|Yes| I[Route to Station]
    H -->|No| J[Queue for Station]

    subgraph Capability Matching
        K[Gift Wrap Order] --> L[gift_wrap capability]
        M[Hazmat Item] --> N[hazmat capability]
        O[Multi-Item Order] --> P[multi_item capability]
        Q[High Value Item] --> R[high_value capability]
    end
```

## Station Status Transitions

```mermaid
stateDiagram-v2
    [*] --> active: NewStation()
    active --> inactive: Deactivate()
    active --> maintenance: SetMaintenance()
    inactive --> active: Activate()
    inactive --> maintenance: SetMaintenance()
    maintenance --> active: Activate()
    maintenance --> inactive: Deactivate()
```

## Repository Interface

```mermaid
classDiagram
    class StationRepository {
        <<Interface>>
        +Save(station Station) error
        +FindByID(id string) Station
        +Update(station Station) error
        +Delete(id string) error
        +FindByStatus(status StationStatus) []Station
        +FindByType(type StationType) []Station
        +FindByZone(zone string) []Station
        +FindByCapabilities(caps []StationCapability) []Station
        +FindAvailable(type StationType) []Station
    }
```

## Related Diagrams

- [DDD Aggregates](ddd/aggregates.md) - Aggregate documentation
- [OpenAPI Specification](openapi.yaml) - REST API contracts
- [AsyncAPI Specification](asyncapi.yaml) - Event contracts
