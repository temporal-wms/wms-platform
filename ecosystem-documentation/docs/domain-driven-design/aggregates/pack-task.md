---
sidebar_position: 6
---

# PackTask Aggregate

The PackTask aggregate manages the packing process for orders.

## Aggregate Structure

```mermaid
classDiagram
    class PackTask {
        <<Aggregate Root>>
        +ID string
        +OrderID string
        +Status PackTaskStatus
        +WorkerID *string
        +StationID string
        +Items []PackItem
        +Package Package
        +TrackingNumber *string
        +StartedAt *time.Time
        +CompletedAt *time.Time
        +SelectPackaging(type string)
        +Pack()
        +Seal()
        +ApplyLabel(trackingNumber string)
    }

    class PackItem {
        <<Entity>>
        +ID string
        +SKU string
        +ProductName string
        +Quantity int
        +PackedQuantity int
        +Status PackItemStatus
    }

    class Package {
        <<Entity>>
        +ID string
        +Type PackageType
        +Dimensions Dimensions
        +Weight Weight
        +Materials []string
        +SealedAt *time.Time
    }

    class PackTaskStatus {
        <<Enumeration>>
        PENDING
        IN_PROGRESS
        PACKED
        LABELED
        SEALED
        COMPLETED
    }

    PackTask "1" *-- "*" PackItem : items
    PackTask "1" *-- "1" Package : package
    PackTask --> PackTaskStatus : status
```

## State Machine

```mermaid
stateDiagram-v2
    [*] --> Pending: Task Created
    Pending --> InProgress: Packer Assigned
    InProgress --> Packed: All Items Packed
    Packed --> Labeled: Label Applied
    Labeled --> Sealed: Package Sealed
    Sealed --> Completed: Complete
    Pending --> Cancelled: Timeout
    InProgress --> Cancelled: Items Missing
    Completed --> [*]
    Cancelled --> [*]
```

## Packing Process

```mermaid
sequenceDiagram
    participant Packer
    participant PackTask
    participant Scale
    participant Printer
    participant Shipping

    Packer->>PackTask: Select Packaging
    PackTask-->>Packer: Package Type

    loop For Each Item
        Packer->>PackTask: Scan Item
        PackTask-->>Packer: Item Verified
        Packer->>Packer: Place in Package
    end

    Packer->>Scale: Weigh Package
    Scale-->>PackTask: Record Weight

    PackTask->>Shipping: Generate Label
    Shipping-->>Printer: Print Label
    Printer-->>Packer: Label Ready

    Packer->>PackTask: Apply Label
    Packer->>PackTask: Seal Package
    PackTask-->>Packer: Complete
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| All Items Packed | All order items must be packed |
| Weight Recorded | Package must be weighed before labeling |
| Label Required | Tracking number required before sealing |
| Package Selected | Package type must be selected before packing |

## Commands

### CreatePackTask

```go
func NewPackTask(orderID string, items []PackItem, stationID string) *PackTask {
    return &PackTask{
        ID:        uuid.New().String(),
        OrderID:   orderID,
        Status:    PackTaskStatusPending,
        StationID: stationID,
        Items:     items,
        CreatedAt: time.Now(),
    }
}
```

### SelectPackaging

```go
func (pt *PackTask) SelectPackaging(packageType PackageType, dimensions Dimensions) error {
    if pt.Status != PackTaskStatusInProgress {
        return ErrInvalidStatusTransition
    }

    pt.Package = &Package{
        ID:         uuid.New().String(),
        Type:       packageType,
        Dimensions: dimensions,
        Materials:  getMaterialsForType(packageType),
    }

    pt.addEvent(NewPackagingSuggestedEvent(pt))
    return nil
}
```

### PackItem

```go
func (pt *PackTask) PackItem(itemID string, quantity int) error {
    if pt.Status != PackTaskStatusInProgress {
        return ErrInvalidStatusTransition
    }

    item := pt.findItem(itemID)
    if item == nil {
        return ErrItemNotFound
    }

    if quantity > item.Quantity-item.PackedQuantity {
        return ErrQuantityExceeded
    }

    item.PackedQuantity += quantity
    if item.PackedQuantity == item.Quantity {
        item.Status = PackItemStatusPacked
    }

    // Check if all items are packed
    if pt.allItemsPacked() {
        pt.Status = PackTaskStatusPacked
    }

    return nil
}
```

### RecordWeight

```go
func (pt *PackTask) RecordWeight(weight Weight) error {
    if pt.Package == nil {
        return ErrNoPackage
    }

    pt.Package.Weight = weight
    return nil
}
```

### ApplyLabel

```go
func (pt *PackTask) ApplyLabel(trackingNumber string) error {
    if pt.Status != PackTaskStatusPacked {
        return ErrInvalidStatusTransition
    }

    pt.TrackingNumber = &trackingNumber
    pt.Status = PackTaskStatusLabeled
    pt.addEvent(NewLabelAppliedEvent(pt))
    return nil
}
```

### Seal

```go
func (pt *PackTask) Seal() error {
    if pt.Status != PackTaskStatusLabeled {
        return ErrInvalidStatusTransition
    }

    now := time.Now()
    pt.Package.SealedAt = &now
    pt.Status = PackTaskStatusSealed
    pt.addEvent(NewPackageSealedEvent(pt))
    return nil
}
```

## Domain Events

| Event | Trigger | Data |
|-------|---------|------|
| PackTaskCreatedEvent | Task created | Task ID, order ID |
| PackagingSuggestedEvent | Package selected | Task ID, package type |
| PackageSealedEvent | Package sealed | Task ID, tracking number |
| LabelAppliedEvent | Label affixed | Task ID, tracking number |
| PackTaskCompletedEvent | Task complete | Task ID, duration |

## Package Type Selection

```mermaid
flowchart TD
    Start[Calculate Package] --> Size{Check Dimensions}

    Size -->|Small| Small[Envelope]
    Size -->|Medium| Medium[Standard Box]
    Size -->|Large| Large[Large Box]
    Size -->|Oversize| Custom[Custom]

    Small --> Fragile{Fragile?}
    Medium --> Fragile
    Large --> Fragile

    Fragile -->|Yes| Padded[Add Padding]
    Fragile -->|No| Ready[Ready]
    Padded --> Ready
```

## Repository Interface

```go
type PackTaskRepository interface {
    Save(ctx context.Context, task *PackTask) error
    FindByID(ctx context.Context, id string) (*PackTask, error)
    FindByOrderID(ctx context.Context, orderID string) (*PackTask, error)
    FindByStationID(ctx context.Context, stationID string) ([]*PackTask, error)
    FindPending(ctx context.Context) ([]*PackTask, error)
    Update(ctx context.Context, task *PackTask) error
}
```

## Related Documentation

- [Packing Service](/services/packing-service) - Service documentation
- [Packing Workflow](/architecture/sequence-diagrams/packing-workflow) - Workflow details
- [Shipment Aggregate](./shipment) - Next step
