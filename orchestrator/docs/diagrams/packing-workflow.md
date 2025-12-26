# Packing Workflow

This diagram shows the packing child workflow with 7 steps: task creation, material selection, packing, weighing, label generation, label application, and sealing.

## Packing Sequence Diagram

```mermaid
sequenceDiagram
    autonumber
    participant Parent as OrderFulfillmentWorkflow
    participant Packing as PackingWorkflow
    participant PackingSvc as Packing Service
    participant LaborSvc as Labor Service
    participant ShippingSvc as Shipping Service
    participant Packer as Warehouse Packer
    participant Scale as Weighing Scale
    participant Printer as Label Printer

    Parent->>Packing: Start PackingWorkflow
    Note over Packing: WorkflowID: packing-{orderId}

    rect rgb(240, 248, 255)
        Note over Packing,PackingSvc: Step 1: Create Pack Task
        Packing->>PackingSvc: CreatePackTask Activity
        PackingSvc->>LaborSvc: Assign Packer
        LaborSvc-->>PackingSvc: WorkerID
        PackingSvc-->>Packing: TaskID, Station
    end

    rect rgb(255, 250, 240)
        Note over Packing,Packer: Step 2: Select Packaging Materials
        Packing->>PackingSvc: SelectPackagingMaterials Activity
        PackingSvc->>PackingSvc: Calculate package size
        PackingSvc->>PackingSvc: Check fragile items
        PackingSvc-->>Packing: PackageType, Materials
        Packer->>Packer: Retrieve packaging
    end

    rect rgb(240, 255, 240)
        Note over Packing,Packer: Step 3: Pack Items
        Packing->>PackingSvc: PackItems Activity

        loop For Each Item
            Packer->>PackingSvc: Scan Item Barcode
            PackingSvc->>PackingSvc: Verify Item
            Packer->>Packer: Place in Package
            Note right of Packer: Add padding if fragile
        end

        PackingSvc-->>Packing: Items Packed
    end

    rect rgb(255, 240, 245)
        Note over Packing,Scale: Step 4: Weigh Package
        Packing->>PackingSvc: WeighPackage Activity
        Packer->>Scale: Place Package
        Scale->>PackingSvc: Weight Reading
        PackingSvc->>PackingSvc: Record dimensions
        PackingSvc-->>Packing: Weight, Dimensions
    end

    rect rgb(245, 245, 255)
        Note over Packing,ShippingSvc: Step 5: Generate Shipping Label
        Packing->>ShippingSvc: GenerateShippingLabel Activity
        ShippingSvc->>ShippingSvc: Calculate shipping rate
        ShippingSvc->>ShippingSvc: Select carrier
        ShippingSvc->>Printer: Print Label
        Printer-->>Packer: Label Printed
        ShippingSvc-->>Packing: TrackingNumber, Carrier
    end

    rect rgb(255, 255, 240)
        Note over Packing,Packer: Step 6: Apply Label
        Packing->>PackingSvc: ApplyLabelToPackage Activity
        Packer->>Packer: Apply label to package
        Packer->>PackingSvc: Scan label barcode
        PackingSvc-->>Packing: Label Applied
    end

    rect rgb(240, 255, 255)
        Note over Packing,PackingSvc: Step 7: Seal Package
        Packing->>PackingSvc: SealPackage Activity
        Packer->>Packer: Seal with tape/adhesive
        PackingSvc->>PackingSvc: Mark task complete
        PackingSvc-->>Packing: Package Sealed
    end

    Packing-->>Parent: PackResult

    Note over Parent: Continue to Shipping
```

## Pack Task State Machine

```mermaid
stateDiagram-v2
    [*] --> Pending: Task Created
    Pending --> InProgress: Packer Assigned
    InProgress --> Packed: Items Verified & Packed
    Packed --> Labeled: Label Applied
    Labeled --> Completed: Package Sealed
    Completed --> [*]

    Pending --> Cancelled: Timeout
    InProgress --> Cancelled: Items Missing
    Cancelled --> [*]
```

## Package Type Selection

```mermaid
flowchart TD
    Start[Calculate Package] --> Size{Check Dimensions}

    Size -->|Small| Small[Envelope/Poly Mailer]
    Size -->|Medium| Medium[Standard Box]
    Size -->|Large| Large[Large Box]
    Size -->|Oversize| Custom[Custom Packaging]

    Small --> Fragile{Fragile Items?}
    Medium --> Fragile
    Large --> Fragile

    Fragile -->|Yes| Padded[Add Padding/Bubble Wrap]
    Fragile -->|No| Ready[Ready to Pack]
    Padded --> Ready

    Custom --> Special[Special Handling Required]
```

## Data Structures

### PackTask
| Field | Type | Description |
|-------|------|-------------|
| TaskID | string | Unique identifier |
| OrderID | string | Associated order |
| Status | string | Current status |
| WorkerID | string | Assigned packer |
| Items | []PackItem | Items to pack |
| PackageID | string | Package identifier |
| PackageType | string | box/envelope/bag |
| TrackingNumber | string | Shipping tracking |
| Carrier | string | Shipping carrier |
| Weight | float64 | Package weight (kg) |
| Dimensions | Dimensions | L x W x H |

### PackResult
| Field | Type | Description |
|-------|------|-------------|
| PackageID | string | Sealed package ID |
| TrackingNumber | string | Carrier tracking number |
| Carrier | string | Carrier name |
| Weight | float64 | Final weight |

### Package Types
| Type | Use Case | Max Weight |
|------|----------|------------|
| envelope | Documents, thin items | 0.5 kg |
| padded_envelope | Small fragile items | 1 kg |
| bag | Soft goods, clothing | 5 kg |
| box | General merchandise | 30 kg |
| custom | Oversize items | Varies |

## Packaging Materials

```mermaid
graph LR
    subgraph "Primary Containers"
        Box[Cardboard Box]
        Envelope[Envelope]
        Bag[Poly Bag]
    end

    subgraph "Protective Materials"
        Bubble[Bubble Wrap]
        Paper[Packing Paper]
        Foam[Foam Inserts]
        Air[Air Pillows]
    end

    subgraph "Sealing"
        Tape[Packing Tape]
        Adhesive[Self-Seal]
    end

    Box --> Bubble
    Box --> Paper
    Box --> Foam
    Box --> Air
    Envelope --> Paper
    Bag --> Adhesive
    Box --> Tape
```

## Related Diagrams

- [Consolidation Workflow](consolidation-workflow.md) - Previous step (multi-item)
- [Picking Workflow](picking-workflow.md) - Previous step (single item)
- [Shipping Workflow](shipping-workflow.md) - Next step (SLAM)
- [Order Fulfillment Flow](../../../docs/diagrams/order-fulfillment-flow.md) - Parent workflow
