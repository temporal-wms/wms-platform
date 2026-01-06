---
sidebar_position: 1
slug: /temporal/workflows/order-fulfillment
---

# OrderFulfillmentWorkflow

The main saga workflow that orchestrates the entire order fulfillment process from validation through shipping.

## Overview

This is the primary workflow that coordinates all bounded contexts in the WMS Platform:
**Order → Waving → Routing → Picking → Consolidation → Packing → SLAM → Sortation → Shipping**

## Configuration

| Property | Value |
|----------|-------|
| Task Queue | `orchestrator` |
| Execution Timeout | 24 hours |
| Activity Timeout | 5 minutes (30 min total with retries) |
| Heartbeat Timeout | 30 seconds |
| Retry Policy | Standard (3 attempts, exponential backoff) |

## Input

```go
// OrderFulfillmentInput represents the input for the order fulfillment workflow
type OrderFulfillmentInput struct {
    OrderID            string    `json:"orderId"`           // Unique order identifier
    CustomerID         string    `json:"customerId"`        // Customer identifier
    Items              []Item    `json:"items"`             // Order line items
    Priority           string    `json:"priority"`          // same_day, next_day, standard
    PromisedDeliveryAt time.Time `json:"promisedDeliveryAt"` // Delivery promise time
    IsMultiItem        bool      `json:"isMultiItem"`       // Multi-item order flag

    // Process path fields
    GiftWrap         bool                   `json:"giftWrap"`              // Gift wrap required
    GiftWrapDetails  *GiftWrapDetailsInput  `json:"giftWrapDetails,omitempty"`
    HazmatDetails    *HazmatDetailsInput    `json:"hazmatDetails,omitempty"`
    ColdChainDetails *ColdChainDetailsInput `json:"coldChainDetails,omitempty"`
    TotalValue       float64                `json:"totalValue"`            // Order total value

    // Unit-level tracking fields (always enabled)
    UnitIDs         []string `json:"unitIds,omitempty"`         // Pre-reserved unit IDs if any
}

// Item represents an order item
type Item struct {
    SKU               string  `json:"sku"`
    Quantity          int     `json:"quantity"`
    Weight            float64 `json:"weight"`
    IsFragile         bool    `json:"isFragile"`
    IsHazmat          bool    `json:"isHazmat"`
    RequiresColdChain bool    `json:"requiresColdChain"`
}

// GiftWrapDetailsInput contains gift wrap configuration
type GiftWrapDetailsInput struct {
    WrapType    string `json:"wrapType"`
    GiftMessage string `json:"giftMessage"`
    HidePrice   bool   `json:"hidePrice"`
}

// HazmatDetailsInput contains hazmat details
type HazmatDetailsInput struct {
    Class              string `json:"class"`
    UNNumber           string `json:"unNumber"`
    PackingGroup       string `json:"packingGroup"`
    ProperShippingName string `json:"properShippingName"`
    LimitedQuantity    bool   `json:"limitedQuantity"`
}

// ColdChainDetailsInput contains cold chain requirements
type ColdChainDetailsInput struct {
    MinTempCelsius  float64 `json:"minTempCelsius"`
    MaxTempCelsius  float64 `json:"maxTempCelsius"`
    RequiresDryIce  bool    `json:"requiresDryIce"`
    RequiresGelPack bool    `json:"requiresGelPack"`
}
```

## Output

```go
// OrderFulfillmentResult represents the result of the order fulfillment workflow
type OrderFulfillmentResult struct {
    OrderID        string `json:"orderId"`
    Status         string `json:"status"`                    // completed, partial_success, failed
    TrackingNumber string `json:"trackingNumber,omitempty"`
    WaveID         string `json:"waveId,omitempty"`
    Error          string `json:"error,omitempty"`

    // Unit-level tracking results
    PathID         string   `json:"pathId,omitempty"`         // Persisted process path ID
    CompletedUnits []string `json:"completedUnits,omitempty"` // Successfully processed units
    FailedUnits    []string `json:"failedUnits,omitempty"`    // Failed units
    ExceptionIDs   []string `json:"exceptionIds,omitempty"`   // Exception IDs for failures
    PartialSuccess bool     `json:"partialSuccess,omitempty"` // Some units succeeded
}
```

## Workflow Steps

### High-Level Flow

```mermaid
flowchart LR
    subgraph Stage1[" Stage 1: Entry "]
        A[📦 Order Received] --> B[✅ Validate]
    end

    subgraph Stage2[" Stage 2: Planning "]
        B --> C[🗺️ Process Path]
        C --> D[📋 Wave Assignment]
    end

    subgraph Stage3[" Stage 3: Execution "]
        D --> E[🏃 Pick]
        E --> F{Multi-route?}
        F -->|Yes| G[🧱 Wall/Consolidate]
        F -->|No| H[📦 Pack]
        G --> H
    end

    subgraph Stage4[" Stage 4: Ship "]
        H --> I[🏷️ SLAM]
        I --> J[📤 Sort]
        J --> K[🚚 Ship]
    end

    K --> L[✅ Complete]
```

### Order State Machine

```mermaid
stateDiagram-v2
    [*] --> received: Order Created

    received --> validating: Start Workflow
    validating --> validated: Validation Pass
    validating --> failed: Validation Fail

    validated --> planning: Begin Planning
    planning --> planned: Wave Assigned
    planning --> failed: Wave Timeout

    planned --> picking: Begin WES
    picking --> picked: Pick Complete
    picking --> failed: Pick Failed

    picked --> consolidating: Multi-route
    picked --> packing: Single-route
    consolidating --> packing: Consolidation Done

    packing --> packed: Pack Complete
    packed --> labeling: Begin SLAM
    labeling --> labeled: Label Applied

    labeled --> sorting: Begin Sort
    sorting --> sorted: Route Assigned

    sorted --> shipping: Begin Ship
    shipping --> shipped: Carrier Handoff

    shipped --> [*]: Complete
    failed --> [*]: Failed
```

### Detailed Sequence

```mermaid
sequenceDiagram
    participant C as Client
    participant OF as OrderFulfillment
    participant PL as PlanningWorkflow
    participant WES as WESExecutionWorkflow
    participant SLAM as SLAM Activity
    participant SORT as SortationWorkflow
    participant SHIP as ShippingWorkflow

    C->>OF: Start workflow (OrderFulfillmentInput)

    Note over OF: Step 1: Validate Order
    OF->>OF: ValidateOrder activity

    Note over OF: Step 2: Planning
    OF->>PL: Execute child workflow
    PL-->>OF: ProcessPath, WaveID, UnitIDs

    Note over OF: Step 3: WES Execution
    OF->>WES: Execute child workflow (wes-execution-queue)
    WES-->>OF: Pick, Wall, Pack results

    Note over OF: Step 4: SLAM Process
    OF->>SLAM: ExecuteSLAM activity
    SLAM-->>OF: TrackingNumber, ManifestID

    Note over OF: Step 5: Sortation
    OF->>SORT: Execute child workflow
    SORT-->>OF: BatchID, ChuteID

    Note over OF: Step 6: Shipping
    OF->>SHIP: Execute child workflow
    SHIP-->>OF: Shipping confirmed

    OF-->>C: OrderFulfillmentResult
```

### Stage Progression

```mermaid
gantt
    title Order Fulfillment Timeline
    dateFormat X
    axisFormat %s

    section Validation
    Validate Order     :v1, 0, 1

    section Planning
    Determine Path     :p1, 1, 2
    Wait for Wave      :p2, 2, 5

    section Execution
    Picking            :e1, 5, 8
    Consolidation      :e2, 8, 10
    Packing            :e3, 10, 13

    section Shipping
    SLAM Process       :s1, 13, 14
    Sortation          :s2, 14, 15
    Ship Confirm       :s3, 15, 16
```

## Query Handlers

| Query | Returns | Purpose |
|-------|---------|---------|
| `getStatus` | `OrderFulfillmentQueryStatus` | Get current workflow status |

```go
// OrderFulfillmentQueryStatus represents the current status
type OrderFulfillmentQueryStatus struct {
    OrderID          string `json:"orderId"`
    CurrentStage     string `json:"currentStage"`     // validation, planning, wes_execution, etc.
    Status           string `json:"status"`           // in_progress, completed, failed
    CompletionPercent int   `json:"completionPercent"` // 0-100
    TotalStages      int    `json:"totalStages"`      // Always 5
    CompletedStages  int    `json:"completedStages"`
    Error            string `json:"error,omitempty"`
}
```

## Child Workflows

| Child | Workflow ID Pattern | Task Queue | Purpose |
|-------|---------------------|------------|---------|
| [PlanningWorkflow](./planning) | `planning-{orderId}` | `orchestrator` | Process path and wave assignment |
| [WESExecutionWorkflow](./wes-execution) | `wes-{orderId}` | `wes-execution-queue` | Pick → Wall → Pack |
| [SortationWorkflow](./sortation) | `sortation-{orderId}` | `orchestrator` | Route to destination chute |
| [ShippingWorkflow](./shipping) | `shipping-{orderId}` | `orchestrator` | Carrier handoff |

## Activities Used

| Activity | Purpose |
|----------|---------|
| `ValidateOrder` | Validates order data and inventory availability |
| `ExecuteSLAM` | Scan, Label, Apply, Manifest process |
| `MarkPacked` | Updates order status to packed |
| `ReleaseInventoryReservation` | Compensation on failure |

## Error Handling

### Compensation Flow

```mermaid
graph TD
    VAL[Validation Failed] --> DONE[Return Error]
    PLAN[Planning Failed] --> DONE
    WES[WES Execution Failed] --> REL[ReleaseInventoryReservation]
    REL --> DONE
    SLAM[SLAM Failed] --> DONE
    SORT[Sortation Failed] --> DONE
    SHIP[Shipping Failed] --> DONE
```

### Failure Statuses

| Status | Description | Compensation |
|--------|-------------|--------------|
| `validation_failed` | Order validation failed | None |
| `planning_failed` | Wave assignment timeout or path error | None |
| `wes_execution_failed` | Picking, walling, or packing failed | Release inventory |
| `slam_failed` | Label generation failed | None |
| `sortation_failed` | Package routing failed | None |
| `shipping_failed` | Carrier handoff failed | None |

## Unit-Level Tracking

Unit-level tracking is **always enabled** in the current version. This provides:

- Individual unit tracking through the fulfillment process
- Granular audit trails for each physical unit
- Better exception handling at the unit level
- Accurate consolidation for multi-route orders

When `UnitIDs` are provided in the input, those pre-reserved units are used. Otherwise, units are reserved during the planning phase.

## Versioning

```go
// Current version
OrderFulfillmentWorkflowVersion = 1

// Change IDs for specific features
OrderFulfillmentMultiRouteSupport = "multi-route-support"
OrderFulfillmentUnitTracking      = "unit-level-tracking"  // Now always enabled
```

## Usage Example

```go
// Start workflow
options := client.StartWorkflowOptions{
    ID:                       fmt.Sprintf("order-fulfillment-%s", orderID),
    TaskQueue:                "orchestrator",
    WorkflowExecutionTimeout: 24 * time.Hour,
}

input := workflows.OrderFulfillmentInput{
    OrderID:            "ORD-123",
    CustomerID:         "CUST-456",
    Items:              items,
    Priority:           "same_day",
    PromisedDeliveryAt: time.Now().Add(8 * time.Hour),
    IsMultiItem:        true,
    TotalValue:         149.99,
    // Unit tracking is always enabled - UnitIDs are optional (reserved during planning if not provided)
}

we, err := client.ExecuteWorkflow(ctx, options, workflows.OrderFulfillmentWorkflow, input)

// Query status
var status workflows.OrderFulfillmentQueryStatus
err = we.QueryWorkflow(ctx, &status, "getStatus")
```

## Related Documentation

- [Planning Workflow](./planning) - Process path determination
- [WES Execution Workflow](./wes-execution) - Warehouse execution
- [Order Activities](../activities/order-activities) - Order management activities
- [Inventory Activities](../activities/inventory-activities) - Inventory operations
- [Architecture - Order Fulfillment](/architecture/sequence-diagrams/order-fulfillment)
