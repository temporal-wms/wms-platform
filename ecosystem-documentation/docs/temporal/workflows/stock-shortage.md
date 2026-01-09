---
sidebar_position: 11
slug: /temporal/workflows/stock-shortage
---

# StockShortageWorkflow

Handles compensation for confirmed stock shortages during picking with automatic fulfillment strategy decisions.

## Overview

The Stock Shortage Workflow coordinates:
1. Recording inventory shortages
2. Calculating fulfillment ratio
3. Deciding on fulfillment strategy (partial ship, backorder, or hold)
4. Creating backorders for missing items
5. Notifying customers and supervisors

## Configuration

| Property | Value |
|----------|-------|
| Task Queue | `orchestrator` |
| Execution Timeout | 4 hours |
| Activity Timeout | 5 minutes |
| Partial Shipment Threshold | 50% |

## Input

```go
// StockShortageWorkflowInput represents input for the stock shortage handling workflow
type StockShortageWorkflowInput struct {
    OrderID        string       `json:"orderId"`
    CustomerID     string       `json:"customerId"`
    ShortItems     []ShortItem  `json:"shortItems"`
    CompletedItems []PickedItem `json:"completedItems"`
    ReportedBy     string       `json:"reportedBy"`
}

// ShortItem represents an item with a stock shortage
type ShortItem struct {
    SKU          string `json:"sku"`
    LocationID   string `json:"locationId"`
    RequestedQty int    `json:"requestedQty"`
    AvailableQty int    `json:"availableQty"`
    ShortageQty  int    `json:"shortageQty"`
    Reason       string `json:"reason"` // not_found, damaged, quantity_mismatch
}
```

## Output

```go
// StockShortageWorkflowResult represents the result of shortage handling
type StockShortageWorkflowResult struct {
    OrderID              string `json:"orderId"`
    Strategy             string `json:"strategy"` // partial_ship, full_backorder, hold_for_review
    ShippedItemCount     int    `json:"shippedItemCount"`
    BackorderedItemCount int    `json:"backorderedItemCount"`
    BackorderID          string `json:"backorderId,omitempty"`
    CustomerNotified     bool   `json:"customerNotified"`
}
```

## Workflow Steps

```mermaid
sequenceDiagram
    participant PICK as PickingWorkflow
    participant SS as StockShortageWorkflow
    participant RS as RecordShortage
    participant MO as MarkOrder
    participant BO as CreateBackorder
    participant NC as NotifyCustomer
    participant NS as NotifySupervisor

    PICK->>SS: Start shortage handling

    Note over SS: Step 1: Record Shortages
    loop For each short item
        SS->>RS: RecordStockShortage activity
    end

    Note over SS: Step 2: Calculate Fulfillment Ratio
    SS->>SS: Calculate ratio

    alt Ratio >= 50% & Has Completed Items
        Note over SS: Partial Ship Strategy
        SS->>MO: MarkOrderPartiallyFulfilled
        SS->>BO: CreateBackorder (short items)
        SS->>NC: NotifyCustomerPartialShipment
    else Ratio < 50% & Has Completed Items
        Note over SS: Hold for Review Strategy
        SS->>NS: NotifySupervisorShortageReview
    else No Completed Items
        Note over SS: Full Backorder Strategy
        SS->>BO: CreateBackorder (all items)
        SS->>NC: NotifyCustomerShortage
    end

    SS-->>PICK: StockShortageWorkflowResult
```

## Fulfillment Strategy Decision

```mermaid
graph TD
    START[Shortage Detected] --> CALC[Calculate Fulfillment Ratio]
    CALC --> HAS{Has Completed<br/>Items?}

    HAS -->|Yes| RATIO{Ratio >= 50%?}
    HAS -->|No| FULL[Full Backorder]

    RATIO -->|Yes| PARTIAL[Partial Ship]
    RATIO -->|No| HOLD[Hold for Review]

    PARTIAL --> SHIP[Ship Available Items]
    SHIP --> BACK1[Backorder Missing]
    BACK1 --> NOTIFY1[Notify Customer]

    HOLD --> SUPER[Notify Supervisor]

    FULL --> BACK2[Backorder All Items]
    BACK2 --> NOTIFY2[Notify Customer]
```

## Shortage Reasons

| Reason | Description |
|--------|-------------|
| `not_found` | Item not at expected location |
| `damaged` | Item found damaged |
| `quantity_mismatch` | Less quantity than expected |

## Activities Used

| Activity | Purpose | On Failure |
|----------|---------|------------|
| `RecordStockShortage` | Records shortage in inventory system | Log warning, continue |
| `MarkOrderPartiallyFulfilled` | Updates order status | Log warning, continue |
| `CreateBackorder` | Creates backorder for missing items | Log error |
| `NotifyCustomerPartialShipment` | Notifies of partial fulfillment | Log warning, continue |
| `NotifyCustomerShortage` | Notifies of complete shortage | Log warning, continue |
| `NotifySupervisorShortageReview` | Escalates to supervisor | Log warning, continue |

## Strategy Comparison

| Strategy | Condition | Action |
|----------|-----------|--------|
| `partial_ship` | Ratio >= 50% with completed items | Ship available, backorder rest |
| `hold_for_review` | Ratio < 50% with completed items | Hold everything for supervisor |
| `full_backorder` | No completed items | Create full backorder |

## Partial Shipment Threshold

```go
// PartialShipmentThreshold is the minimum fulfillment ratio (0.0-1.0) to auto-ship
const PartialShipmentThreshold = 0.50
```

Orders with at least 50% fulfillment automatically proceed with partial shipment.

---

# BackorderFulfillmentWorkflow

Handles auto-fulfillment of backorders when stock arrives.

## Overview

Triggered by `InventoryReceivedEvent` for backordered SKUs, this workflow:
1. Reserves stock for backorder items
2. Creates new pick tasks linked to original order
3. Notifies customers of backorder shipping

## Input

```go
// BackorderFulfillmentInput
{
    "backorderId":     string,
    "originalOrderId": string,
    "customerId":      string
}
```

## Workflow Steps

```mermaid
sequenceDiagram
    participant INV as InventoryReceived
    participant BF as BackorderFulfillment
    participant RS as ReserveStock
    participant PT as CreatePickTask
    participant MB as MarkInProgress
    participant NC as NotifyCustomer

    INV->>BF: Start backorder fulfillment

    Note over BF: Step 1: Reserve Stock
    BF->>RS: ReserveStockForBackorder
    RS-->>BF: Success

    Note over BF: Step 2: Create Pick Task
    BF->>PT: CreateBackorderPickTask
    PT-->>BF: TaskID

    Note over BF: Step 3: Mark In Progress
    BF->>MB: MarkBackorderInProgress
    MB-->>BF: Success

    Note over BF: Step 4: Notify Customer
    BF->>NC: NotifyCustomerBackorderShipping
    NC-->>BF: Success

    BF-->>INV: Complete
```

## Activities Used

| Activity | Purpose | On Failure |
|----------|---------|------------|
| `ReserveStockForBackorder` | Reserves incoming stock | Return error |
| `CreateBackorderPickTask` | Creates pick task for backorder | Return error |
| `MarkBackorderInProgress` | Updates backorder status | Log warning |
| `NotifyCustomerBackorderShipping` | Sends shipping notification | Log warning |

## Usage Example

```go
// Called from StockShortageWorkflow on picking failure
shortageInput := StockShortageWorkflowInput{
    OrderID:    "ORD-123",
    CustomerID: "CUST-456",
    ShortItems: []ShortItem{
        {
            SKU:          "SKU-001",
            LocationID:   "LOC-A1",
            RequestedQty: 5,
            AvailableQty: 2,
            ShortageQty:  3,
            Reason:       "quantity_mismatch",
        },
    },
    CompletedItems: completedPickedItems,
    ReportedBy:     "PICKER-001",
}

var result StockShortageWorkflowResult
err := workflow.ExecuteChildWorkflow(ctx, StockShortageWorkflow, shortageInput).Get(ctx, &result)
```

## Related Documentation

- [Picking Workflow](./picking) - Detects shortages during picking
- [Order Fulfillment Workflow](./order-fulfillment) - Parent workflow
- [Inventory Activities](../activities/inventory-activities) - Inventory operations
