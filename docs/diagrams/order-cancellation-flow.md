# Order Cancellation Flow

This diagram shows the order cancellation workflow with the Saga compensation pattern.

## Cancellation Sequence

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Orchestrator
    participant Temporal
    participant OrderSvc as Order Service
    participant InventorySvc as Inventory Service
    participant NotificationSvc as Notification

    Client->>Orchestrator: Cancel Order Request
    Orchestrator->>Temporal: Start OrderCancellationWorkflow

    rect rgb(255, 240, 240)
        Note over Orchestrator,OrderSvc: Step 1: Cancel Order
        Orchestrator->>OrderSvc: CancelOrder Activity
        OrderSvc->>OrderSvc: Update Status to Cancelled
        OrderSvc-->>Orchestrator: Order Cancelled
    end

    rect rgb(240, 255, 240)
        Note over Orchestrator,InventorySvc: Step 2: Release Inventory (Compensation)
        Orchestrator->>InventorySvc: ReleaseInventoryReservation Activity
        InventorySvc->>InventorySvc: Release Reserved Stock
        InventorySvc-->>Orchestrator: Inventory Released
    end

    rect rgb(240, 240, 255)
        Note over Orchestrator,NotificationSvc: Step 3: Notify Customer (Best Effort)
        Orchestrator->>NotificationSvc: NotifyCustomerCancellation Activity
        Note right of NotificationSvc: Non-critical step<br/>Errors are logged only
        NotificationSvc-->>Orchestrator: Notification Sent
    end

    Orchestrator->>Temporal: Workflow Complete
    Orchestrator-->>Client: Cancellation Confirmed
```

## Compensation Pattern

```mermaid
graph TD
    subgraph "Normal Flow"
        A[Order Created] --> B[Inventory Reserved]
        B --> C[Wave Assigned]
        C --> D[Picking Started]
    end

    subgraph "Compensation Flow"
        D -->|Failure| E[Release Inventory]
        C -->|Failure| E
        B -->|Cancel Request| E
        E --> F[Cancel Order]
        F --> G[Notify Customer]
    end

    style E fill:#ffcccc
    style F fill:#ffcccc
    style G fill:#ffffcc
```

## Error Handling Strategy

```mermaid
flowchart TD
    Start[Cancellation Request] --> CancelOrder{Cancel Order}

    CancelOrder -->|Success| ReleaseInv{Release Inventory}
    CancelOrder -->|Failure| Fail[Return Error]

    ReleaseInv -->|Success| Notify{Notify Customer}
    ReleaseInv -->|Failure| LogError1[Log Error]
    LogError1 --> Notify

    Notify -->|Success| Complete[Workflow Complete]
    Notify -->|Failure| LogError2[Log Error - Best Effort]
    LogError2 --> Complete

    style Fail fill:#ff6666
    style Complete fill:#66ff66
    style LogError1 fill:#ffcc66
    style LogError2 fill:#ffcc66
```

## Retry Policy

| Activity | Max Attempts | Initial Interval | Backoff |
|----------|--------------|------------------|---------|
| CancelOrder | 3 | 1 second | 2.0x |
| ReleaseInventoryReservation | 3 | 1 second | 2.0x |
| NotifyCustomerCancellation | 1 | - | None (best-effort) |

## Activity Options

- **StartToCloseTimeout**: 5 minutes
- **RetryPolicy**: Standard retry with 3 max attempts
- **Non-Retryable Errors**: ValidationError, NotFoundError

## Cancellation Rules

| Order Status | Can Cancel? | Notes |
|--------------|-------------|-------|
| received | Yes | Full refund |
| validated | Yes | Full refund |
| wave_assigned | Yes | Full refund |
| picking | Yes | May have partial pick |
| consolidated | Yes | Items returned to stock |
| packed | Yes | Package unpacked |
| shipped | No | Contact carrier |
| delivered | No | Return process required |

## Related Diagrams

- [Order Fulfillment Flow](order-fulfillment-flow.md) - Normal order flow
- [Ecosystem](ecosystem.md) - Platform overview
- [Domain Events](ddd/domain-events.md) - OrderCancelledEvent
