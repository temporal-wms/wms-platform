# Order Fulfillment Flow

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

## Error Handling

When any step fails, the workflow triggers compensation:
1. Release inventory reservations
2. Cancel order
3. Notify customer

## Related Diagrams

- [Ecosystem](ecosystem.md) - Platform overview
- [Order Cancellation Flow](order-cancellation-flow.md) - Compensation pattern
- [Picking Workflow](../../orchestrator/docs/diagrams/picking-workflow.md) - Detailed picking
- [Packing Workflow](../../orchestrator/docs/diagrams/packing-workflow.md) - Detailed packing
- [Shipping Workflow](../../orchestrator/docs/diagrams/shipping-workflow.md) - SLAM process
