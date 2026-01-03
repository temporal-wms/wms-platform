---
sidebar_position: 1
---

# Order Fulfillment Workflow

This diagram shows the complete end-to-end order fulfillment saga, from order receipt to shipment confirmation using the WES (Warehouse Execution System) as the central execution engine.

## High-Level Flow

```mermaid
sequenceDiagram
    autonumber
    participant Customer
    participant OrderSvc as Order Service
    participant Orchestrator
    participant Temporal
    participant WavingSvc as Waving Service
    participant WES as WES Execution
    participant ShippingSvc as Shipping Service
    participant Carrier

    Customer->>OrderSvc: Place Order
    OrderSvc->>Orchestrator: Start OrderFulfillmentWorkflow
    Orchestrator->>Temporal: Register Workflow

    rect rgb(240, 248, 255)
        Note over Orchestrator,OrderSvc: Step 1: Validate Order
        Orchestrator->>OrderSvc: ValidateOrder Activity
        OrderSvc-->>Orchestrator: Order Valid
    end

    rect rgb(255, 250, 240)
        Note over Orchestrator,WavingSvc: Step 2: Wave Assignment
        Orchestrator->>Orchestrator: Wait for Signal (waveAssigned)
        WavingSvc->>Orchestrator: Signal: WaveAssignment
        Note right of Orchestrator: Timeout: 30min-4hr<br/>based on priority
    end

    rect rgb(240, 255, 240)
        Note over Orchestrator,WES: Step 3: WES Execution (Child Workflow)
        Orchestrator->>WES: Start WESExecutionWorkflow
        Note over WES: Resolves execution plan<br/>Creates task route<br/>Executes stages
        WES-->>Orchestrator: WESExecutionResult
    end

    rect rgb(240, 255, 255)
        Note over Orchestrator,ShippingSvc: Step 4: Shipping - SLAM Process
        Orchestrator->>ShippingSvc: Start ShippingWorkflow
        ShippingSvc->>ShippingSvc: Scan, Label, Apply, Manifest
        ShippingSvc->>Carrier: Hand off to carrier
        ShippingSvc-->>Orchestrator: Shipment Confirmed
    end

    Orchestrator->>Customer: Order Shipped Notification
    Orchestrator->>Temporal: Workflow Complete
```

## WES Execution Detail

The WES workflow handles all warehouse execution internally based on the order profile:

```mermaid
sequenceDiagram
    autonumber
    participant Orch as Orchestrator
    participant WES as WES Workflow
    participant Pick as Picking Service
    participant Wall as Walling Service
    participant Pack as Packing Service

    Orch->>WES: Start WESExecutionWorkflow

    Note over WES: Resolve Execution Plan
    WES->>WES: Determine path type based on items

    alt pick_pack (1-3 items)
        WES->>Pick: Picking Stage
        Pick-->>WES: Pick Complete
        WES->>Pack: Packing Stage
        Pack-->>WES: Pack Complete
    else pick_wall_pack (4-20 items)
        WES->>Pick: Picking Stage
        Pick-->>WES: Pick Complete
        WES->>Wall: Create Walling Task
        WES->>WES: Wait for wallingCompleted signal
        Wall->>Orch: POST /signals/walling-completed
        Orch->>WES: Signal wallingCompleted
        WES->>Pack: Packing Stage
        Pack-->>WES: Pack Complete
    else pick_consolidate_pack (multi-zone)
        WES->>Pick: Picking Stage (multiple zones)
        Pick-->>WES: Pick Complete
        WES->>WES: Consolidation Stage
        WES->>Pack: Packing Stage
        Pack-->>WES: Pack Complete
    end

    WES-->>Orch: WESExecutionResult
```

## Process Path Selection

```mermaid
graph TB
    Start[Order Received] --> Check{Check Order Profile}

    Check -->|1-3 items| PP[pick_pack]
    Check -->|4-20 items| PWP[pick_wall_pack]
    Check -->|Multi-zone| PCP[pick_consolidate_pack]

    subgraph "pick_pack"
        PP --> PP1[Picking]
        PP1 --> PP2[Packing]
    end

    subgraph "pick_wall_pack"
        PWP --> PWP1[Picking]
        PWP1 --> PWP2[Walling]
        PWP2 --> PWP3[Packing]
    end

    subgraph "pick_consolidate_pack"
        PCP --> PCP1[Picking]
        PCP1 --> PCP2[Consolidation]
        PCP2 --> PCP3[Packing]
    end

    PP2 --> Ship[Shipping]
    PWP3 --> Ship
    PCP3 --> Ship
```

## Workflow States

```mermaid
stateDiagram-v2
    [*] --> Received: Order Placed
    Received --> Validated: Validation Passed
    Received --> Cancelled: Validation Failed
    Validated --> WaveAssigned: Wave Signal
    Validated --> Cancelled: Wave Timeout
    WaveAssigned --> WESExecuting: WES Started
    WESExecuting --> Shipped: WES + Shipping Complete
    WESExecuting --> Cancelled: Stage Failed
    Shipped --> Delivered: Carrier Delivery
    Shipped --> [*]
    Cancelled --> [*]
```

## Priority-Based Timeouts

| Priority | Wave Timeout | WES Timeout | Description |
|----------|--------------|-------------|-------------|
| same_day | 30 minutes | 2 hours | Same-day delivery orders |
| next_day | 2 hours | 3 hours | Next-day delivery orders |
| standard | 4 hours | 4 hours | Standard delivery orders |

## Activity Sequence

```mermaid
graph LR
    subgraph "Phase 1: Preparation"
        A1[ValidateOrder] --> A2[ReserveInventory]
    end

    subgraph "Phase 2: Wave"
        A2 --> A3[Wait for Wave Signal]
    end

    subgraph "Phase 3: WES Execution"
        A3 --> A4[WES Child Workflow]
        A4 --> A5[Picking Stage]
        A5 --> A6[Walling/Consolidation?]
        A6 --> A7[Packing Stage]
    end

    subgraph "Phase 4: Fulfillment"
        A7 --> A8[CreateShipment]
        A8 --> A9[ConfirmShip]
    end
```

## Temporal Workflow Details

### Workflow Configuration

| Setting | Value |
|---------|-------|
| **TaskQueue** | orchestrator-queue |
| **WorkflowExecutionTimeout** | 24 hours |
| **WorkflowTaskTimeout** | 10 seconds |
| **RetryPolicy.MaximumAttempts** | 3 |

### Child Workflows

| Workflow | Purpose | Task Queue |
|----------|---------|------------|
| WESExecutionWorkflow | Coordinate picking → walling? → packing | wes-queue |
| ShippingWorkflow | SLAM process | orchestrator-queue |

## Signal Flow

The orchestrator routes signals to the appropriate workflows:

```mermaid
sequenceDiagram
    participant Sim as Simulator/Worker
    participant Orch as Orchestrator
    participant Temporal as Temporal Server
    participant Parent as OrderFulfillmentWorkflow
    participant Child as WESExecutionWorkflow

    Note over Sim,Child: Wave Assignment Signal
    Sim->>Orch: POST /api/v1/signals/wave-assigned
    Orch->>Temporal: Signal workflow
    Temporal->>Parent: Deliver waveAssigned
    Parent->>Parent: Resume workflow

    Note over Sim,Child: Walling Completed Signal
    Sim->>Orch: POST /api/v1/signals/walling-completed
    Note over Orch: Route to child workflow
    Orch->>Temporal: Signal child workflow
    Temporal->>Child: Deliver wallingCompleted
    Child->>Child: Resume to packing stage
```

## Error Handling and Compensation

When any step fails, the workflow triggers compensation:

```mermaid
sequenceDiagram
    participant Workflow
    participant WES
    participant Inventory
    participant Order
    participant Notification

    Note over Workflow: Activity Failed

    alt WES Stage Failed
        Workflow->>WES: Get stage status
        WES-->>Workflow: Failed at picking/walling/packing
    end

    Workflow->>Inventory: ReleaseReservation
    Inventory-->>Workflow: Released

    Workflow->>Order: CancelOrder
    Order-->>Workflow: Cancelled

    Workflow->>Notification: NotifyCustomer
    Note right of Notification: Best-effort (non-blocking)
```

### Compensation Actions

| Failure Point | Compensation |
|---------------|--------------|
| Validation Failed | Cancel order, refund payment |
| Wave Timeout | Release reservation, cancel order |
| WES Picking Failed | Release reservation, cancel order |
| WES Walling Timeout | Release reservation, cancel order |
| WES Packing Failed | Return items to stock, cancel order |
| Shipping Failed | Reschedule or cancel |

## Related Diagrams

- [Order Cancellation](./order-cancellation) - Compensation pattern
- [WES Execution](./wes-execution) - WES workflow details
- [Walling Workflow](./walling-workflow) - Put-wall sorting process
- [Shipping Workflow](./shipping-workflow) - SLAM process
