# Order Cancellation Workflow

This diagram shows the order cancellation workflow that handles compensation and cleanup when orders are cancelled at various stages of fulfillment.

## Cancellation Overview

```mermaid
graph LR
    subgraph "Order State"
        A[Order Pending]
        B[Soft Reserved]
        C[Hard Allocated/Staged]
        D[Packed]
        E[Shipped]
    end

    subgraph "Cancellation Action"
        A --> |"Cancel"| Release1[Release Reservation]
        B --> |"Cancel"| Release2[Release Reservation]
        C --> |"Cancel"| Return[Return to Shelf]
        D --> |"Cancel"| Return
        E --> |"Cannot Cancel"| Refuse[Process Return Instead]
    end

    style A fill:#e8f5e9
    style B fill:#fff9c4
    style C fill:#ffcc80
    style D fill:#90caf9
    style E fill:#ef9a9a
```

## Simple Cancellation Sequence (Soft Reservation Only)

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Workflow as OrderCancellationWorkflow
    participant OrderSvc as Order Service
    participant InventorySvc as Inventory Service
    participant NotifySvc as Notification Service
    participant Customer

    Client->>Workflow: Start Cancellation(orderId, reason)
    Note over Workflow: For orders with soft reservation only

    rect rgb(255, 240, 240)
        Note over Workflow,OrderSvc: Step 1: Cancel Order
        Workflow->>OrderSvc: CancelOrder Activity
        OrderSvc->>OrderSvc: Update order status
        OrderSvc->>OrderSvc: Mark as cancelled
        OrderSvc-->>Workflow: Order Cancelled
    end

    rect rgb(240, 255, 240)
        Note over Workflow,InventorySvc: Step 2: Release Soft Reservation
        Workflow->>InventorySvc: ReleaseInventoryReservation Activity
        InventorySvc->>InventorySvc: Find reservations for order
        InventorySvc->>InventorySvc: Release reserved quantities
        InventorySvc->>InventorySvc: Return to available stock
        InventorySvc-->>Workflow: Reservation Released
    end

    rect rgb(240, 240, 255)
        Note over Workflow,Customer: Step 3: Notify Customer
        Workflow->>NotifySvc: NotifyCustomerCancellation Activity
        NotifySvc->>Customer: Email: Order Cancelled
        NotifySvc-->>Workflow: Notification Sent
    end

    Workflow-->>Client: Cancellation Complete
```

## Cancellation with Hard Allocations Sequence

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Workflow as OrderCancellationWorkflowWithAllocations
    participant OrderSvc as Order Service
    participant InventorySvc as Inventory Service
    participant NotifySvc as Notification Service
    participant Customer

    Client->>Workflow: Start Cancellation(input)
    Note over Workflow: For orders that have been staged/packed

    rect rgb(255, 240, 240)
        Note over Workflow,OrderSvc: Step 1: Cancel Order
        Workflow->>OrderSvc: CancelOrder Activity
        OrderSvc->>OrderSvc: Update order status
        OrderSvc->>OrderSvc: Mark as cancelled
        OrderSvc-->>Workflow: Order Cancelled
    end

    alt Has Hard Allocations (isHardAllocated = true)
        rect rgb(255, 250, 230)
            Note over Workflow,InventorySvc: Step 2a: Return Inventory to Shelf
            Workflow->>InventorySvc: ReturnInventoryToShelf Activity
            Note right of Workflow: Physical return required

            loop For Each Allocation
                InventorySvc->>InventorySvc: Find allocation by ID
                InventorySvc->>InventorySvc: Mark for return
                InventorySvc->>InventorySvc: Create return-to-shelf task
            end

            InventorySvc->>InventorySvc: Return items to original locations
            InventorySvc->>InventorySvc: Increment available quantity
            InventorySvc-->>Workflow: Inventory Returned
        end
    else Soft Reservation Only
        rect rgb(240, 255, 240)
            Note over Workflow,InventorySvc: Step 2b: Release Soft Reservation
            Workflow->>InventorySvc: ReleaseInventoryReservation Activity
            InventorySvc->>InventorySvc: Release reserved quantities
            InventorySvc-->>Workflow: Reservation Released
        end
    end

    rect rgb(240, 240, 255)
        Note over Workflow,Customer: Step 3: Notify Customer
        Workflow->>NotifySvc: NotifyCustomerCancellation Activity
        NotifySvc->>Customer: Email: Order Cancelled
        Note right of Customer: Best effort notification
        NotifySvc-->>Workflow: Notification Sent
    end

    Workflow-->>Client: Cancellation Complete
```

## Cancellation Decision Flow

```mermaid
flowchart TD
    Start[Cancel Request] --> CheckStatus{Order Status?}

    CheckStatus -->|Pending/Validated| SoftReserve[Soft Reservation Only]
    CheckStatus -->|Picking| InPicking{Picking Started?}
    CheckStatus -->|Staged/Packed| HardAlloc[Hard Allocated]
    CheckStatus -->|Shipped| CannotCancel[Cannot Cancel]

    InPicking -->|No| SoftReserve
    InPicking -->|Yes| HardAlloc

    SoftReserve --> Simple[OrderCancellationWorkflow]
    Simple --> Release[ReleaseInventoryReservation]

    HardAlloc --> WithAlloc[OrderCancellationWorkflowWithAllocations]
    WithAlloc --> Return[ReturnInventoryToShelf]

    CannotCancel --> Refuse[Reject - Process Return Instead]

    Release --> Notify[NotifyCustomerCancellation]
    Return --> Notify

    Notify --> Complete[Cancellation Complete]

    style Complete fill:#c8e6c9
    style Refuse fill:#ffcdd2
```

## Inventory Compensation States

```mermaid
stateDiagram-v2
    [*] --> Available: Initial Stock

    Available --> SoftReserved: Reserve (order placed)
    SoftReserved --> Available: ReleaseReservation (simple cancel)

    SoftReserved --> HardAllocated: StageInventory (picking complete)
    HardAllocated --> ReturnPending: Cancel with allocations

    HardAllocated --> Packed: PackInventory
    Packed --> ReturnPending: Cancel packed order

    ReturnPending --> Available: ReturnInventoryToShelf

    Packed --> Shipped: ShipInventory
    Shipped --> [*]: Cannot compensate in-system

    note right of ReturnPending : Physical return\nto shelf required
    note right of Shipped : Handle via\nreturns process
```

## Data Structures

### OrderCancellationInput (Simple)
| Field | Type | Description |
|-------|------|-------------|
| OrderID | string | Order to cancel |
| Reason | string | Cancellation reason |

### OrderCancellationInput (With Allocations)
| Field | Type | Description |
|-------|------|-------------|
| OrderID | string | Order to cancel |
| Reason | string | Cancellation reason |
| AllocationIDs | []string | Hard allocation IDs to return |
| PickedItems | []PickedItem | Items that were picked |
| IsHardAllocated | bool | Whether inventory has been staged |

### Return-to-Shelf Item
| Field | Type | Description |
|-------|------|-------------|
| SKU | string | Item SKU |
| AllocationID | string | Hard allocation to reverse |

## Cancellation Reasons

| Code | Reason | Description |
|------|--------|-------------|
| customer_request | Customer Requested | Customer cancelled order |
| payment_failed | Payment Failed | Payment could not be processed |
| inventory_unavailable | Inventory Unavailable | Items became unavailable |
| fraud_detected | Fraud Detected | Fraudulent order detected |
| address_invalid | Invalid Address | Shipping address invalid |
| duplicate_order | Duplicate Order | Duplicate of another order |

## Error Handling

```mermaid
flowchart TD
    Start[Start Cancellation] --> Cancel{Cancel Order OK?}
    Cancel -->|Error| FailCancel[Return Error]
    Cancel -->|OK| Inventory{Inventory Step}

    Inventory -->|Hard Allocated| Return{Return to Shelf OK?}
    Inventory -->|Soft Reserved| Release{Release OK?}

    Return -->|Error| WarnReturn[Log Warning - Continue]
    Return -->|OK| Notify
    WarnReturn --> Notify

    Release -->|Error| WarnRelease[Log Warning - Continue]
    Release -->|OK| Notify
    WarnRelease --> Notify

    Notify{Notify Customer OK?}
    Notify -->|Error| WarnNotify[Log Warning]
    Notify -->|OK| Complete
    WarnNotify --> Complete

    Complete[Cancellation Complete]

    style Complete fill:#c8e6c9
    style WarnReturn fill:#fff9c4
    style WarnRelease fill:#fff9c4
    style WarnNotify fill:#fff9c4
```

## Compensation Matrix

| Order State | Inventory State | Cancellation Action | Physical Action Required |
|-------------|-----------------|---------------------|--------------------------|
| Pending | None | None | No |
| Validated | Soft Reserved | ReleaseInventoryReservation | No |
| Wave Assigned | Soft Reserved | ReleaseInventoryReservation | No |
| Picking Started | Soft Reserved | ReleaseInventoryReservation | No |
| Picking Complete | Hard Allocated | ReturnInventoryToShelf | Yes - Return to location |
| Consolidated | Hard Allocated | ReturnInventoryToShelf | Yes - Return to location |
| Packed | Hard Allocated (Packed) | ReturnInventoryToShelf | Yes - Unpack & return |
| Shipped | Removed from System | Cannot Cancel | N/A - Use returns process |

## Related Diagrams

- [Order Fulfillment Flow](order-fulfillment.md) - Main workflow that may trigger cancellation
- [Picking Workflow](picking-workflow.md) - Where hard allocation begins
- [Shipping Workflow](shipping-workflow.md) - Point of no return for cancellation
