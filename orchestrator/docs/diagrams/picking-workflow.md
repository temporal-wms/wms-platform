# Picking Workflow

This diagram shows the detailed picking child workflow, including task creation, worker assignment, and pick completion signaling.

## Picking Sequence Diagram

```mermaid
sequenceDiagram
    autonumber
    participant Parent as OrderFulfillmentWorkflow
    participant Picking as PickingWorkflow
    participant PickingSvc as Picking Service
    participant LaborSvc as Labor Service
    participant Picker as Warehouse Picker
    participant InventorySvc as Inventory Service

    Parent->>Picking: Start PickingWorkflow
    Note over Picking: WorkflowID: picking-{orderId}

    rect rgb(240, 248, 255)
        Note over Picking,PickingSvc: Step 1: Create Pick Task
        Picking->>PickingSvc: CreatePickTask Activity
        PickingSvc->>PickingSvc: Create task with route stops
        PickingSvc-->>Picking: TaskID
    end

    rect rgb(255, 250, 240)
        Note over Picking,LaborSvc: Step 2: Assign Worker
        Picking->>LaborSvc: AssignPickerToTask Activity
        LaborSvc->>LaborSvc: Find available picker in zone
        LaborSvc->>LaborSvc: Update worker status
        LaborSvc-->>Picking: WorkerID
    end

    rect rgb(240, 255, 240)
        Note over Picking,Picker: Step 3: Wait for Pick Completion
        Picking->>Picking: Wait for Signal (pickCompleted)
        Note right of Picking: Timeout: 30 minutes

        loop For Each Route Stop
            Picker->>PickingSvc: Scan Location Barcode
            PickingSvc->>InventorySvc: Verify Item at Location
            InventorySvc-->>PickingSvc: Item Confirmed
            Picker->>PickingSvc: Scan Item Barcode
            Picker->>PickingSvc: Confirm Quantity
            PickingSvc->>PickingSvc: Record Pick
            Picker->>PickingSvc: Place in Tote
        end

        Picker->>PickingSvc: Complete Pick Task
        PickingSvc->>Picking: Signal: pickCompleted
    end

    rect rgb(255, 245, 238)
        Note over Picking,InventorySvc: Step 4: Confirm Inventory Pick
        Picking->>InventorySvc: ConfirmInventoryPick Activity
        Note right of Picking: Decrement stock quantities
        loop For Each Picked Item
            InventorySvc->>InventorySvc: POST /inventory/{sku}/pick
            InventorySvc->>InventorySvc: Decrement quantity at location
        end
        InventorySvc-->>Picking: Inventory Confirmed
    end

    rect rgb(230, 255, 230)
        Note over Picking,InventorySvc: Step 5: Stage Inventory (Hard Allocation)
        Picking->>InventorySvc: StageInventory Activity
        Note right of Picking: Convert soft reservation to hard allocation
        InventorySvc->>InventorySvc: POST /inventory/stage
        InventorySvc->>InventorySvc: Create physical claim on items
        InventorySvc-->>Picking: AllocationIDs
    end

    Picking-->>Parent: PickResult (includes AllocationIDs)

    Note over Parent: Continue to Consolidation/Packing
```

## Pick Task State Machine

```mermaid
stateDiagram-v2
    [*] --> Pending: Task Created
    Pending --> Assigned: Worker Assigned
    Assigned --> InProgress: Picker Started
    InProgress --> InProgress: Item Picked
    InProgress --> Exception: Problem Reported
    Exception --> InProgress: Exception Resolved
    InProgress --> Completed: All Items Picked
    Completed --> [*]

    Pending --> Cancelled: Timeout/Cancel
    Assigned --> Cancelled: Worker Unavailable
    Exception --> Cancelled: Unresolvable
    Cancelled --> [*]
```

## Pick Item Flow

```mermaid
flowchart TD
    A[Arrive at Location] --> B{Scan Location}
    B -->|Valid| C{Scan Item}
    B -->|Invalid| D[Report Exception]
    C -->|Match| E[Enter Quantity]
    C -->|Mismatch| D
    E --> F{Quantity Correct?}
    F -->|Yes| G[Place in Tote]
    F -->|Short| H[Report Short Pick]
    H --> G
    G --> I{More Stops?}
    I -->|Yes| A
    I -->|No| J[Complete Task]
    D --> K[Supervisor Review]
    K --> A
```

## Data Structures

### PickTask
| Field | Type | Description |
|-------|------|-------------|
| TaskID | string | Unique task identifier |
| OrderID | string | Associated order |
| WaveID | string | Wave assignment |
| RouteID | string | Optimized pick route |
| WorkerID | string | Assigned picker |
| Status | string | Current status |
| Items | []PickItem | Items to pick |
| ToteID | string | Output container |

### PickResult
| Field | Type | Description |
|-------|------|-------------|
| TaskID | string | Completed task ID |
| PickedItems | []PickedItem | Successfully picked items |
| AllocationIDs | []string | Hard allocation IDs from staging |
| Success | bool | Completion status |

### PickedItem
| Field | Type | Description |
|-------|------|-------------|
| SKU | string | Item SKU |
| Quantity | int | Quantity picked |
| LocationID | string | Pick location |
| ToteID | string | Destination tote |

## Exception Types

| Exception | Cause | Resolution |
|-----------|-------|------------|
| ItemNotFound | Item not at location | Check alternate location |
| Damaged | Item damaged | Report for adjustment |
| QuantityMismatch | Less than expected | Short pick or recount |
| WrongItem | SKU mismatch | Find correct item |

## Related Diagrams

- [Order Fulfillment Flow](order-fulfillment.md) - Parent workflow
- [Consolidation Workflow](consolidation-workflow.md) - Next step (multi-item)
- [Packing Workflow](packing-workflow.md) - Next step (single item)
