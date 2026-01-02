# Sortation Service - DDD Aggregates

This document describes the aggregate structure for the Sortation bounded context following Domain-Driven Design principles.

## Aggregate: SortationBatch

The SortationBatch aggregate represents a group of packages being sorted for the same destination/carrier combination.

```mermaid
graph TD
    subgraph "SortationBatch Aggregate"
        SortationBatch[SortationBatch<br/><<Aggregate Root>>]

        subgraph "Entities"
            SortedPackage[SortedPackage]
            Chute[Chute]
        end

        subgraph "Value Objects"
            Status[SortationStatus]
            DestGroup[DestinationGroup]
        end

        SortationBatch -->|contains| SortedPackage
        SortationBatch -->|assigned to| Chute
        SortationBatch -->|has| Status
        SortationBatch -->|for| DestGroup
    end

    style SortationBatch fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "SortationBatch Aggregate Boundary"
        SB[SortationBatch]
        SP[SortedPackage]
        CH[Chute]
    end

    subgraph "External References"
        CENTER[SortationCenter]
        CARRIER[CarrierID]
        TRAILER[TrailerID]
        DOCK[DispatchDock]
    end

    SB -.->|at| CENTER
    SB -.->|for| CARRIER
    SB -.->|loads to| TRAILER
    SB -.->|from| DOCK

    style SB fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Status transitions valid | Status can only change according to state machine |
| Carrier required | Batch must have carrier assignment |
| Destination group required | Must have destination grouping (zip prefix/region) |
| Package uniqueness | Same package cannot be in multiple batches |
| Sort before ready | All packages must be sorted before marking ready |
| Trailer before dispatch | Trailer must be assigned before dispatch |

## Domain Events

```mermaid
graph LR
    SortationBatch -->|emits| E1[SortationBatchCreatedEvent]
    SortationBatch -->|emits| E2[PackageReceivedForSortationEvent]
    SortationBatch -->|emits| E3[PackageSortedEvent]
    SortationBatch -->|emits| E4[BatchReadyEvent]
    SortationBatch -->|emits| E5[BatchDispatchedEvent]
```

## Event Details

| Event | Trigger | Payload |
|-------|---------|---------|
| SortationBatchCreatedEvent | New batch created | batchId, sortationCenter, destinationGroup, carrierId |
| PackageReceivedForSortationEvent | Package added to batch | batchId, packageId, orderId, destination |
| PackageSortedEvent | Package sorted to chute | batchId, packageId, chuteId, sortedBy |
| BatchReadyEvent | All packages sorted | batchId, destinationGroup, carrierId, packageCount |
| BatchDispatchedEvent | Batch leaves facility | batchId, trailerId, dispatchDock, packageCount, totalWeight |

## Sortation Flow

```mermaid
sequenceDiagram
    participant P as Packing
    participant S as Sortation
    participant C as Chute
    participant T as Trailer

    P->>S: Package ready for sortation
    S->>S: Find/Create batch by destination
    S->>S: Add package to batch
    S->>C: Assign package to chute
    C->>S: Package sorted confirmation
    S->>S: Update sorted count

    Note over S: When batch is full or complete
    S->>T: Assign trailer
    S->>S: Dispatch batch
```

## Factory Pattern

```mermaid
classDiagram
    class SortationBatchFactory {
        +CreateForDestination(center, destGroup, carrier) SortationBatch
        +GetOrCreate(center, destGroup, carrier) SortationBatch
        +ReconstituteFromEvents(events) SortationBatch
    }

    class SortationBatch {
        -constructor()
        +static Create() SortationBatch
    }

    SortationBatchFactory --> SortationBatch : creates
```

## Repository Pattern

```mermaid
classDiagram
    class SortationBatchRepository {
        <<Interface>>
        +Save(batch SortationBatch)
        +FindByID(id string) SortationBatch
        +FindByStatus(status SortationStatus) []SortationBatch
        +FindByCarrier(carrierID string) []SortationBatch
        +FindByDestinationGroup(group string) []SortationBatch
        +FindReady(limit int) []SortationBatch
        +FindOpenBatch(destGroup, carrier string) SortationBatch
    }

    class MongoSortationBatchRepository {
        +Save(batch SortationBatch)
        +FindByID(id string) SortationBatch
        +FindByStatus(status SortationStatus) []SortationBatch
        +FindByCarrier(carrierID string) []SortationBatch
        +FindByDestinationGroup(group string) []SortationBatch
        +FindReady(limit int) []SortationBatch
        +FindOpenBatch(destGroup, carrier string) SortationBatch
    }

    SortationBatchRepository <|.. MongoSortationBatchRepository
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [AsyncAPI Specification](../asyncapi.yaml) - Event contracts
