---
sidebar_position: 1
---

# Order Fulfillment Workflow

This diagram shows the complete end-to-end order fulfillment saga, from order receipt to shipment confirmation.

## High-Level Flow

```mermaid
sequenceDiagram
    autonumber
    participant Customer
    participant OrderSvc as Order Service
    participant Orchestrator
    participant Temporal
    participant WavingSvc as Waving Service
    participant RoutingSvc as Routing Service
    participant PickingSvc as Picking Service
    participant ConsolidationSvc as Consolidation Service
    participant PackingSvc as Packing Service
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
        Note over Orchestrator,RoutingSvc: Step 3: Route Calculation
        Orchestrator->>RoutingSvc: CalculateRoute Activity
        RoutingSvc-->>Orchestrator: RouteResult (stops, distance)
    end

    rect rgb(255, 240, 245)
        Note over Orchestrator,PickingSvc: Step 4: Picking (Child Workflow)
        Orchestrator->>PickingSvc: Start PickingWorkflow
        PickingSvc->>PickingSvc: CreatePickTask
        PickingSvc->>PickingSvc: AssignPickerToTask
        PickingSvc->>Orchestrator: Signal: pickCompleted
        PickingSvc-->>Orchestrator: PickResult
    end

    rect rgb(245, 245, 255)
        Note over Orchestrator,ConsolidationSvc: Step 5: Consolidation (if multi-item)
        alt Multi-Item Order
            Orchestrator->>ConsolidationSvc: Start ConsolidationWorkflow
            ConsolidationSvc-->>Orchestrator: Consolidation Complete
        else Single Item
            Note over Orchestrator: Skip Consolidation
        end
    end

    rect rgb(255, 255, 240)
        Note over Orchestrator,PackingSvc: Step 6: Packing (Child Workflow)
        Orchestrator->>PackingSvc: Start PackingWorkflow
        PackingSvc->>PackingSvc: Pack & Label
        PackingSvc-->>Orchestrator: PackResult (tracking number)
    end

    rect rgb(240, 255, 255)
        Note over Orchestrator,ShippingSvc: Step 7: Shipping - SLAM Process
        Orchestrator->>ShippingSvc: Start ShippingWorkflow
        ShippingSvc->>ShippingSvc: Scan, Label, Apply, Manifest
        ShippingSvc->>Carrier: Hand off to carrier
        ShippingSvc-->>Orchestrator: Shipment Confirmed
    end

    Orchestrator->>Customer: Order Shipped Notification
    Orchestrator->>Temporal: Workflow Complete
```

## Workflow States

```mermaid
stateDiagram-v2
    [*] --> Received: Order Placed
    Received --> Validated: Validation Passed
    Received --> Cancelled: Validation Failed
    Validated --> WaveAssigned: Wave Signal
    Validated --> Cancelled: Wave Timeout
    WaveAssigned --> Picking: Route Calculated
    Picking --> Consolidated: Multi-item
    Picking --> Packed: Single item
    Consolidated --> Packed: Items Combined
    Picking --> Cancelled: Picking Failed
    Packed --> Shipped: SLAM Complete
    Packed --> Cancelled: Packing Failed
    Shipped --> Delivered: Carrier Delivery
    Shipped --> [*]
    Cancelled --> [*]
```

## Priority-Based Timeouts

| Priority | Wave Timeout | Description |
|----------|--------------|-------------|
| same_day | 30 minutes | Same-day delivery orders |
| next_day | 2 hours | Next-day delivery orders |
| standard | 4 hours | Standard delivery orders |

## Activity Sequence

```mermaid
graph LR
    subgraph "Phase 1: Preparation"
        A1[ValidateOrder] --> A2[ReserveInventory]
    end

    subgraph "Phase 2: Wave"
        A2 --> A3[Wait for Wave Signal]
        A3 --> A4[CalculateRoute]
    end

    subgraph "Phase 3: Execution"
        A4 --> A5[CreatePickTask]
        A5 --> A6[WaitForPickComplete]
        A6 --> A7[ConsolidateItems]
    end

    subgraph "Phase 4: Fulfillment"
        A7 --> A8[CreatePackTask]
        A8 --> A9[CreateShipment]
        A9 --> A10[ConfirmShip]
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
| PickingWorkflow | Coordinate picking operations | picking-queue |
| ConsolidationWorkflow | Combine multi-item orders | consolidation-queue |
| PackingWorkflow | Package preparation | packing-queue |
| ShippingWorkflow | SLAM process | shipping-queue |

## Error Handling and Compensation

When any step fails, the workflow triggers compensation:

```mermaid
sequenceDiagram
    participant Workflow
    participant Order
    participant Inventory
    participant Notification

    Note over Workflow: Activity Failed

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
| Picking Failed | Release reservation, cancel order |
| Packing Failed | Return items to stock, cancel order |
| Shipping Failed | Reschedule or cancel |

## Related Diagrams

- [Order Cancellation](./order-cancellation) - Compensation pattern
- [Picking Workflow](./picking-workflow) - Detailed picking flow
- [Packing Workflow](./packing-workflow) - Detailed packing flow
- [Shipping Workflow](./shipping-workflow) - SLAM process
