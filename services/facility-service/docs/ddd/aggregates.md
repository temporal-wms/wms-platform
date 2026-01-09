# Facility Service - DDD Aggregates

This document describes the aggregate structure for the Facility bounded context following Domain-Driven Design principles.

## Aggregate: Station

The Station aggregate represents a work station in the warehouse with specific capabilities for process path routing.

```mermaid
graph TD
    subgraph "Station Aggregate"
        Station[Station<br/><<Aggregate Root>>]

        subgraph "Entities"
            Equipment[StationEquipment]
        end

        subgraph "Value Objects"
            Capability[StationCapability]
            Type[StationType]
            Status[StationStatus]
        end

        Station -->|has| Equipment
        Station -->|supports| Capability
        Station -->|is| Type
        Station -->|has| Status
    end

    style Station fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "Station Aggregate Boundary"
        ST[Station]
        EQ[Equipment]
        CAP[Capabilities]
    end

    subgraph "External References"
        ZONE[Zone]
        WORKER[AssignedWorkerID]
    end

    ST -.->|in| ZONE
    ST -.->|assigned to| WORKER

    style ST fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Valid station type | Station type must be from defined enum |
| Valid capabilities | All capabilities must be from defined enum |
| Max concurrent tasks positive | MaxConcurrentTasks must be > 0 |
| Current tasks <= max | Cannot exceed maximum concurrent tasks |
| Active for task assignment | Cannot assign tasks to inactive/maintenance stations |
| Unique capability per station | Cannot add duplicate capabilities |

## Capability Matching Rules

| Order Characteristic | Required Capability |
|---------------------|---------------------|
| Single item order | `single_item` |
| Multi-item order | `multi_item` |
| Gift wrap requested | `gift_wrap` |
| Hazardous materials | `hazmat` |
| Oversized items | `oversized` |
| Fragile items | `fragile` |
| Temperature-sensitive | `cold_chain` |
| High-value items | `high_value` |

## Domain Events

```mermaid
graph LR
    Station -->|emits| E1[StationCreatedEvent]
    Station -->|emits| E2[StationCapabilityAddedEvent]
    Station -->|emits| E3[StationCapabilityRemovedEvent]
    Station -->|emits| E4[StationCapabilitiesUpdatedEvent]
    Station -->|emits| E5[StationStatusChangedEvent]
    Station -->|emits| E6[WorkerAssignedToStationEvent]
```

## Event Details

| Event | Trigger | Payload |
|-------|---------|---------|
| StationCreatedEvent | New station created | stationId, name, zone, stationType |
| StationCapabilityAddedEvent | Capability added | stationId, capability, addedAt |
| StationCapabilityRemovedEvent | Capability removed | stationId, capability, removedAt |
| StationCapabilitiesUpdatedEvent | Bulk capability update | stationId, capabilities, updatedAt |
| StationStatusChangedEvent | Status changed | stationId, oldStatus, newStatus |
| WorkerAssignedToStationEvent | Worker assigned | stationId, workerId, assignedAt |

## Process Path Routing Flow

```mermaid
sequenceDiagram
    participant O as Order Service
    participant F as Facility Service
    participant S as Station

    O->>F: Find capable station
    F->>F: Extract order requirements
    F->>F: Query stations by capabilities
    F->>F: Filter by type and zone
    F->>F: Check availability
    F->>S: Verify capacity
    S-->>F: Available capacity
    F-->>O: Best matching station
```

## Factory Pattern

```mermaid
classDiagram
    class StationFactory {
        +CreateStation(id, name, zone, type, maxTasks) Station
        +CreatePackingStation(id, name, zone) Station
        +CreateShippingStation(id, name, zone) Station
        +ReconstituteFromEvents(events) Station
    }

    class Station {
        -constructor()
        +static Create() Station
    }

    StationFactory --> Station : creates
```

## Repository Pattern

```mermaid
classDiagram
    class StationRepository {
        <<Interface>>
        +Save(station Station)
        +FindByID(id string) Station
        +FindByStatus(status StationStatus) []Station
        +FindByType(type StationType) []Station
        +FindByZone(zone string) []Station
        +FindByCapabilities(caps []StationCapability) []Station
        +FindAvailable(type StationType) []Station
        +Delete(id string) error
    }

    class MongoStationRepository {
        +Save(station Station)
        +FindByID(id string) Station
        +FindByStatus(status StationStatus) []Station
        +FindByType(type StationType) []Station
        +FindByZone(zone string) []Station
        +FindByCapabilities(caps []StationCapability) []Station
        +FindAvailable(type StationType) []Station
        +Delete(id string) error
    }

    StationRepository <|.. MongoStationRepository
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [OpenAPI Specification](../openapi.yaml) - REST API contracts
- [AsyncAPI Specification](../asyncapi.yaml) - Event contracts
