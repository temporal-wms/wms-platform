---
sidebar_position: 2
slug: /temporal/diagrams/order-flow
---

# Order Flow Diagram

Complete visualization of order processing through the WMS Platform.

## End-to-End Order Flow

```mermaid
flowchart TD
    subgraph "Order Entry"
        API[Order API] -->|Create| ORD[Order Created]
        ORD -->|Start| OF[OrderFulfillmentWorkflow]
    end

    subgraph "Stage 1: Validation & Planning"
        OF -->|1| VAL[ValidateOrder Activity]
        VAL -->|Valid| PL[PlanningWorkflow]
        VAL -->|Invalid| FAIL1[Order Failed]

        PL -->|Determine| PP[DetermineProcessPath]
        PP -->|Allocate| ALLOC[AllocateInventory]
        ALLOC -->|Wait| WAVE{Wave Assignment}
        WAVE -->|Signal| WAVE_OK[Wave Assigned]
        WAVE -->|Timeout| FAIL2[Planning Failed]
    end

    subgraph "Stage 2: Warehouse Execution"
        WAVE_OK -->|Start| WES[WESExecutionWorkflow]

        WES -->|Stage| PICK[Picking]
        PICK -->|Multi-route?| CON{Consolidation?}
        CON -->|Yes| CONS[Consolidation]
        CON -->|No| PACK[Packing]
        CONS --> PACK

        PICK -.->|Exception| SS[StockShortageWorkflow]
        SS -.->|Resolved| PICK
        SS -.->|Unresolved| PARTIAL[Partial Ship]
    end

    subgraph "Stage 3: Sortation"
        PACK -->|Complete| SORT[SortationWorkflow]
        SORT -->|Batch| CARRIER[Carrier Batching]
        CARRIER -->|Route| LANE[Lane Assignment]
    end

    subgraph "Stage 4: Shipping"
        LANE -->|Ready| SHIP[ShippingWorkflow]
        SHIP -->|SLAM| SLAM[Scan/Label/Apply/Manifest]
        SLAM -->|Confirm| CONFIRM[Ship Confirmed]
    end

    subgraph "Completion"
        CONFIRM -->|Update| COMPLETE[Order Complete]
        PARTIAL -->|Update| PARTIAL_COMPLETE[Partial Complete]
        FAIL1 -->|Notify| FAILED[Order Failed]
        FAIL2 -->|Notify| FAILED
    end

    classDef stage1 fill:#e3f2fd,stroke:#1565c0
    classDef stage2 fill:#fff3e0,stroke:#ef6c00
    classDef stage3 fill:#e8f5e9,stroke:#2e7d32
    classDef stage4 fill:#fce4ec,stroke:#c2185b
    classDef error fill:#ffebee,stroke:#c62828

    class VAL,PL,PP,ALLOC,WAVE,WAVE_OK stage1
    class WES,PICK,CON,CONS,PACK,SS stage2
    class SORT,CARRIER,LANE stage3
    class SHIP,SLAM,CONFIRM stage4
    class FAIL1,FAIL2,FAILED error
```

---

## Detailed Stage Breakdown

### Stage 1: Validation & Planning

```mermaid
sequenceDiagram
    participant OF as OrderFulfillment
    participant OA as OrderActivities
    participant PA as PlanningWorkflow
    participant IA as InventoryActivities
    participant PPA as ProcessPathActivities
    participant WMS as WMS System

    OF->>OA: ValidateOrder
    OA->>OA: Check required fields
    OA->>OA: Validate items exist
    OA-->>OF: ValidationResult

    OF->>PA: Start PlanningWorkflow
    PA->>PPA: DetermineProcessPath
    PPA->>PPA: Analyze item count, special handling
    PPA-->>PA: PathResult (pick_pack | pick_wall_pack | multi_route)

    PA->>IA: AllocateInventory
    IA->>WMS: Reserve inventory
    WMS-->>IA: Allocation result
    IA-->>PA: AllocationResult

    PA->>PA: Wait for waveAssigned signal
    Note over PA: Timeout: 30min-4h based on priority
    WMS-->>PA: Signal: waveAssigned
    PA-->>OF: PlanningResult
```

### Stage 2: Warehouse Execution

```mermaid
sequenceDiagram
    participant OF as OrderFulfillment
    participant WES as WESExecution
    participant PICK as Picking
    participant CON as Consolidation
    participant PACK as Packing
    participant Worker as Warehouse Worker

    OF->>WES: Start WESExecutionWorkflow

    loop For each stage in execution plan
        WES->>PICK: Execute PickingWorkflow
        Worker-->>PICK: Signal: workerAssigned

        loop For each item
            Worker-->>PICK: Signal: itemPicked
        end
        Worker-->>PICK: Signal: pickingComplete
        PICK-->>WES: PickResult

        alt Multi-route order
            WES->>CON: Execute ConsolidationWorkflow
            loop For each tote
                Worker-->>CON: Signal: toteArrived
            end
            Worker-->>CON: Signal: consolidationComplete
            CON-->>WES: ConsolidationResult
        end

        WES->>PACK: Execute PackingWorkflow
        Worker-->>PACK: Signal: packerAssigned
        Worker-->>PACK: Signal: itemVerified (per item)
        Worker-->>PACK: Signal: packageSealed
        Worker-->>PACK: Signal: labelApplied
        Worker-->>PACK: Signal: packingComplete
        PACK-->>WES: PackResult
    end

    WES-->>OF: WESExecutionResult
```

### Stage 3: Sortation

```mermaid
sequenceDiagram
    participant OF as OrderFulfillment
    participant SORT as SortationWorkflow
    participant SA as SortationActivities
    participant Sorter as Sortation System

    OF->>SORT: Start SortationWorkflow

    SORT->>SA: CreateSortationBatch
    SA->>Sorter: Group by carrier/destination
    Sorter-->>SA: BatchID
    SA-->>SORT: BatchResult

    SORT->>SA: AssignSortationLane
    SA->>Sorter: Get available lane
    Sorter-->>SA: Lane assignment
    SA-->>SORT: LaneResult

    SORT->>SA: ConfirmSortation
    SA->>Sorter: Package sorted to lane
    Sorter-->>SA: Confirmation
    SA-->>SORT: SortationComplete

    SORT-->>OF: SortationResult
```

### Stage 4: Shipping

```mermaid
sequenceDiagram
    participant OF as OrderFulfillment
    participant SHIP as ShippingWorkflow
    participant SLAM as SLAMActivities
    participant Carrier as Carrier System

    OF->>SHIP: Start ShippingWorkflow

    SHIP->>SLAM: ScanPackage
    SLAM-->>SHIP: ScanResult

    SHIP->>SLAM: GenerateLabel
    SLAM->>Carrier: Request shipping label
    Carrier-->>SLAM: TrackingNumber, Label
    SLAM-->>SHIP: LabelResult

    SHIP->>SLAM: ApplyLabel
    SLAM-->>SHIP: ApplyResult

    SHIP->>SLAM: AddToManifest
    SLAM->>Carrier: Add to carrier manifest
    Carrier-->>SLAM: ManifestConfirmation
    SLAM-->>SHIP: ManifestResult

    SHIP->>SHIP: Wait for shipConfirmed signal
    Note over SHIP: Or auto-confirm after manifest
    SHIP-->>OF: ShippingResult

    OF->>OF: CompleteOrderFulfillment
```

---

## Process Path Variations

### Single-Item Order (pick_pack)

```mermaid
flowchart LR
    ORD[Order] --> PICK[Pick] --> PACK[Pack] --> SORT[Sort] --> SHIP[Ship]
```

### Multi-Item Order (pick_wall_pack)

```mermaid
flowchart LR
    ORD[Order] --> PICK[Pick] --> WALL[Wall/Consolidate] --> PACK[Pack] --> SORT[Sort] --> SHIP[Ship]
```

### Multi-Route Order (multi_route)

```mermaid
flowchart TD
    ORD[Order] --> SPLIT[Split by Zone]

    SPLIT --> R1[Route 1]
    SPLIT --> R2[Route 2]
    SPLIT --> R3[Route N]

    R1 --> P1[Pick Zone A]
    R2 --> P2[Pick Zone B]
    R3 --> P3[Pick Zone N]

    P1 --> WALL[Put Wall]
    P2 --> WALL
    P3 --> WALL

    WALL --> CON[Consolidate]
    CON --> PACK[Pack]
    PACK --> SORT[Sort]
    SORT --> SHIP[Ship]
```

---

## Special Handling Flows

### Gift Wrap Order

```mermaid
flowchart LR
    PICK[Pick] --> GW[Gift Wrap] --> PACK[Pack] --> SORT[Sort] --> SHIP[Ship]
```

### Hazmat Order

```mermaid
flowchart LR
    PICK[Pick<br/>Certified Worker] --> PACK[Pack<br/>Certified Station] --> SORT[Sort<br/>Hazmat Lane] --> SHIP[Ship<br/>Hazmat Carrier]
```

### Cold Chain Order

```mermaid
flowchart LR
    PICK[Pick<br/>Cold Zone] --> PACK[Pack<br/>Insulated] --> SORT[Sort<br/>Priority] --> SHIP[Ship<br/>Express]
```

---

## Exception Handling Flows

### Stock Shortage

```mermaid
flowchart TD
    PICK[Picking] -->|Item Not Found| EXC[Exception]
    EXC --> SS[StockShortageWorkflow]
    SS -->|Check Alt Location| ALT{Found?}
    ALT -->|Yes| RESUME[Resume Pick]
    ALT -->|No| PARTIAL[Partial Ship Decision]
    PARTIAL -->|Ship Partial| CONTINUE[Continue with Available]
    PARTIAL -->|Wait| BACKORDER[Backorder]
    PARTIAL -->|Cancel| CANCEL[Cancel Order]
```

### Order Cancellation

```mermaid
flowchart TD
    CANCEL[Cancellation Request] --> CHECK{Stage?}

    CHECK -->|Pre-Pick| SIMPLE[Simple Cancel]
    SIMPLE --> RELEASE[Release Inventory]
    RELEASE --> REFUND[Refund]

    CHECK -->|In Progress| COMPLEX[Complex Cancel]
    COMPLEX --> STOP[Stop Current Stage]
    STOP --> REVERSE[Reverse Completed Stages]
    REVERSE --> RELEASE

    CHECK -->|Shipped| RETURN[Initiate Return]
```

## Related Documentation

- [Workflow Hierarchy](./workflow-hierarchy) - Parent-child relationships
- [Signal Flow](./signal-flow) - Signal timing between workflows
- [Order Fulfillment Workflow](../workflows/order-fulfillment) - Detailed workflow documentation
