---
sidebar_position: 3
---

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

    Picking-->>Parent: PickResult

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

## Route Optimization

```mermaid
graph TB
    subgraph "Warehouse Layout"
        subgraph "Zone A"
            A1[A-01-01]
            A2[A-01-02]
            A3[A-02-01]
        end

        subgraph "Zone B"
            B1[B-01-01]
            B2[B-01-02]
        end

        subgraph "Zone C"
            C1[C-01-01]
        end
    end

    subgraph "Optimized Route"
        Start[Start] --> A1
        A1 --> A2
        A2 --> A3
        A3 --> B1
        B1 --> B2
        B2 --> C1
        C1 --> End[End]
    end

    style Start fill:#90EE90
    style End fill:#FFB6C1
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

### PickItem

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Item identifier |
| SKU | string | Product SKU |
| ProductName | string | Product name |
| Quantity | int | Quantity to pick |
| PickedQuantity | int | Quantity picked |
| Location | Location | Pick location |
| Status | string | Item status |

### PickResult

| Field | Type | Description |
|-------|------|-------------|
| TaskID | string | Completed task ID |
| PickedItems | []PickedItem | Successfully picked items |
| Success | bool | Completion status |
| Duration | Duration | Time to complete |

## Exception Types

| Exception | Cause | Resolution |
|-----------|-------|------------|
| ItemNotFound | Item not at location | Check alternate location |
| Damaged | Item damaged | Report for adjustment |
| QuantityMismatch | Less than expected | Short pick or recount |
| WrongItem | SKU mismatch | Find correct item |
| LocationEmpty | Location is empty | Check inventory system |

## Exception Handling Flow

```mermaid
sequenceDiagram
    participant Picker
    participant PickingSvc as Picking Service
    participant Supervisor
    participant InventorySvc as Inventory Service

    Picker->>PickingSvc: Report Exception
    PickingSvc->>PickingSvc: Create Exception Record

    alt ItemNotFound
        PickingSvc->>InventorySvc: Check Alternate Locations
        InventorySvc-->>PickingSvc: Alternate Location
        PickingSvc-->>Picker: Go to Alternate
    else Damaged Item
        PickingSvc->>Supervisor: Notify Supervisor
        Supervisor->>InventorySvc: Adjust Inventory
        Supervisor-->>Picker: Continue or Cancel
    else Short Pick
        PickingSvc->>PickingSvc: Record Partial Pick
        PickingSvc-->>Picker: Continue with Available
    end
```

## Performance Metrics

| Metric | Description | Target |
|--------|-------------|--------|
| Pick Rate | Items picked per hour | 60-100 items/hr |
| Pick Accuracy | Correct picks / Total picks | > 99.5% |
| Travel Time | Time between picks | Minimize |
| Exception Rate | Exceptions / Total picks | < 1% |

## Events Published

| Event | Topic | Trigger |
|-------|-------|---------|
| PickTaskCreatedEvent | wms.picking.events | Task created |
| PickTaskAssignedEvent | wms.picking.events | Worker assigned |
| ItemPickedEvent | wms.picking.events | Each item picked |
| PickExceptionEvent | wms.picking.events | Exception reported |
| PickTaskCompletedEvent | wms.picking.events | All items picked |

## Related Diagrams

- [Order Fulfillment](./order-fulfillment) - Parent workflow
- [Packing Workflow](./packing-workflow) - Next step (single item)
- [PickTask Aggregate](/domain-driven-design/aggregates/pick-task) - Domain model
