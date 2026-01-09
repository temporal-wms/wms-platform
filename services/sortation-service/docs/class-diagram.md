# Sortation Service - Class Diagram

This diagram shows the domain model for the Sortation Service bounded context.

## Domain Model

```mermaid
classDiagram
    class SortationBatch {
        <<Aggregate Root>>
        +BatchID string
        +SortationCenter string
        +DestinationGroup string
        +CarrierID string
        +Packages []SortedPackage
        +AssignedChute string
        +Status SortationStatus
        +TotalPackages int
        +SortedCount int
        +TotalWeight float64
        +TrailerID string
        +DispatchDock string
        +ScheduledDispatch *time.Time
        +CreatedAt time.Time
        +UpdatedAt time.Time
        +DispatchedAt *time.Time
        +AddPackage(pkg SortedPackage) error
        +StartSorting()
        +SortPackage(packageID, chuteID, workerID string)
        +MarkReady()
        +AssignToTrailer(trailerID, dock string)
        +Dispatch()
        +Cancel(reason string)
        +GetUnsortedPackages() []SortedPackage
        +IsFullySorted() bool
        +GetSortingProgress() float64
    }

    class SortedPackage {
        <<Entity>>
        +PackageID string
        +OrderID string
        +TrackingNumber string
        +Destination string
        +CarrierID string
        +Weight float64
        +AssignedChute string
        +SortedAt *time.Time
        +SortedBy string
        +IsSorted bool
    }

    class Chute {
        <<Entity>>
        +ChuteID string
        +ChuteNumber int
        +Destination string
        +CarrierID string
        +Capacity int
        +CurrentCount int
        +Status string
        +AvailableCapacity() int
        +IsAvailable() bool
    }

    class SortationStatus {
        <<Enumeration>>
        receiving
        sorting
        ready
        dispatching
        dispatched
        cancelled
    }

    class ChuteStatus {
        <<Enumeration>>
        active
        full
        maintenance
    }

    SortationBatch "1" *-- "*" SortedPackage : contains
    SortationBatch "1" -- "0..1" Chute : assigned to
    SortationBatch --> SortationStatus : has status
    Chute --> ChuteStatus : has status
```

## Sortation Flow

```mermaid
flowchart TD
    A[Package Arrives] --> B{Find/Create Batch}
    B -->|Existing Batch| C[Add to Batch]
    B -->|New Batch| D[Create Batch]
    D --> C
    C --> E{Assign Chute}
    E --> F[Chute by Destination]
    F --> G[Sort Package to Chute]
    G --> H{Batch Full?}
    H -->|Yes| I[Mark Ready]
    H -->|No| J[Continue Sorting]
    J --> C
    I --> K[Assign Trailer]
    K --> L[Dispatch]
```

## State Transitions

```mermaid
stateDiagram-v2
    [*] --> receiving
    receiving --> sorting: StartSorting() / SortPackage()
    sorting --> ready: MarkReady()
    ready --> dispatching: AssignToTrailer()
    dispatching --> dispatched: Dispatch()

    receiving --> cancelled: Cancel()
    sorting --> cancelled: Cancel()
    ready --> cancelled: Cancel()
```

## Repository Interface

```mermaid
classDiagram
    class SortationBatchRepository {
        <<Interface>>
        +Save(batch SortationBatch) error
        +FindByID(id string) SortationBatch
        +Update(batch SortationBatch) error
        +FindByStatus(status SortationStatus) []SortationBatch
        +FindByCarrier(carrierID string) []SortationBatch
        +FindByDestinationGroup(group string) []SortationBatch
        +FindReady(limit int) []SortationBatch
    }
```

## Related Diagrams

- [DDD Aggregates](ddd/aggregates.md) - Aggregate documentation
- [AsyncAPI Specification](asyncapi.yaml) - Event contracts
