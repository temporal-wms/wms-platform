---
sidebar_position: 5
slug: /temporal/workflows/consolidation
---

# ConsolidationWorkflow

Coordinates the consolidation of multi-item orders from multiple pick routes or totes.

## Overview

The Consolidation Workflow (also known as "Walling") handles:
1. Waiting for all totes from multi-route orders
2. Creating consolidation units
3. Physically consolidating items from different totes
4. Verifying all items are present
5. Unit-level consolidation tracking (when enabled)

## Configuration

| Property | Value |
|----------|-------|
| Task Queue | `orchestrator` |
| Execution Timeout | 4 hours |
| Activity Timeout | 15 minutes |
| Tote Arrival Timeout | 30 minutes |

## Input

```go
// ConsolidationWorkflowInput represents input for the consolidation workflow
type ConsolidationWorkflowInput struct {
    OrderID     string       `json:"orderId"`
    PickedItems []PickedItem `json:"pickedItems"`
    // Unit-level tracking fields
    UnitIDs []string `json:"unitIds,omitempty"` // Specific units to consolidate
    PathID  string   `json:"pathId,omitempty"`  // Process path ID for consistency
    // Multi-route support fields
    IsMultiRoute       bool     `json:"isMultiRoute,omitempty"`       // Flag for multi-route order
    ExpectedRouteCount int      `json:"expectedRouteCount,omitempty"` // Total routes to wait for
    ExpectedTotes      []string `json:"expectedTotes,omitempty"`      // Expected tote IDs from all routes
}
```

## Output

The workflow returns `nil` on success or an error on failure.

## Workflow Steps

```mermaid
sequenceDiagram
    participant WES as WESExecution
    participant CON as ConsolidationWorkflow
    participant CU as CreateUnit
    participant CI as ConsolidateItems
    participant VC as VerifyConsolidation
    participant CC as CompleteConsolidation

    WES->>CON: Start consolidation

    alt Multi-Route Order
        Note over CON: Step 0: Wait for All Totes
        CON->>CON: Wait for toteArrived signals
        CON-->>CON: All totes received
    end

    Note over CON: Step 1: Create Consolidation Unit
    CON->>CU: CreateConsolidationUnit activity
    CU-->>CON: ConsolidationID

    Note over CON: Step 2: Consolidate Items
    CON->>CI: ConsolidateItems activity
    CI-->>CON: Success

    Note over CON: Step 3: Verify Consolidation
    CON->>VC: VerifyConsolidation activity
    VC-->>CON: Verified

    Note over CON: Step 4: Complete Consolidation
    CON->>CC: CompleteConsolidation activity
    CC-->>CON: Success

    CON-->>WES: Complete
```

## Signals

| Signal | Payload | Timeout | Purpose |
|--------|---------|---------|---------|
| `toteArrived` | `ToteArrivedSignal` | 30 minutes | Notifies workflow of tote arrival |

```go
// ToteArrivedSignal represents a tote arrival signal for multi-route orders
type ToteArrivedSignal struct {
    ToteID     string `json:"toteId"`
    RouteID    string `json:"routeId"`
    RouteIndex int    `json:"routeIndex"`
    ArrivedAt  string `json:"arrivedAt"`
}
```

## Multi-Route Tote Collection

For orders split across multiple picking routes:

```mermaid
graph TD
    START[Multi-Route Order] --> WAIT[Wait for Totes]
    WAIT --> CHECK{All Totes<br/>Received?}
    CHECK -->|Yes| CONSOLIDATE[Proceed to Consolidation]
    CHECK -->|No| SIGNAL[Receive toteArrived Signal]
    SIGNAL --> TRACK[Track Received Tote]
    TRACK --> CHECK
    WAIT --> TIMEOUT{30 min<br/>Timeout?}
    TIMEOUT -->|Yes| PARTIAL[Proceed with Partial<br/>Consolidation]
    TIMEOUT -->|No| WAIT
```

### Consolidation Flow Diagram

```mermaid
flowchart TD
    START[📦 Start Consolidation] --> MULTI{Multi-Route?}

    MULTI -->|Yes| WAIT[⏳ Wait for Totes]
    MULTI -->|No| CREATE[Create Unit]

    WAIT --> SIGNAL{toteArrived<br/>Signal?}
    SIGNAL -->|Yes| TRACK[Track Tote<br/>X of Y received]
    SIGNAL -->|Timeout| PARTIAL_START[⚠️ Start with Partial]

    TRACK --> ALL{All Totes?}
    ALL -->|Yes| CREATE
    ALL -->|No| SIGNAL

    PARTIAL_START --> CREATE

    CREATE[🏷️ Create Consolidation Unit] --> ITEMS[📋 Consolidate Items]
    ITEMS --> VERIFY{Verify All<br/>Items Present?}

    VERIFY -->|Yes| COMPLETE[✅ Complete Consolidation]
    VERIFY -->|No| EXCEPTION[⚠️ Log Exception]

    EXCEPTION --> COMPLETE

    style COMPLETE fill:#c8e6c9
    style PARTIAL_START fill:#fff9c4
    style EXCEPTION fill:#fff9c4
```

### Put Wall Operation

```mermaid
flowchart LR
    subgraph Picking["Multiple Pick Routes"]
        R1[Route 1<br/>Zone A] --> T1[Tote 1]
        R2[Route 2<br/>Zone B] --> T2[Tote 2]
        R3[Route 3<br/>Zone C] --> T3[Tote 3]
    end

    subgraph PutWall["Put Wall Station"]
        T1 --> SCAN1[Scan Tote]
        T2 --> SCAN2[Scan Tote]
        T3 --> SCAN3[Scan Tote]

        SCAN1 --> WALL[🧱 Put Wall<br/>Order Slots]
        SCAN2 --> WALL
        SCAN3 --> WALL
    end

    subgraph Output["Consolidated"]
        WALL --> BIN[📦 Order Bin<br/>All Items Together]
    end
```

### Tote State Machine

```mermaid
stateDiagram-v2
    [*] --> in_transit: Picking Complete

    in_transit --> arrived: toteArrived Signal
    in_transit --> timed_out: 30min Timeout

    arrived --> scanned: Worker Scans
    scanned --> items_sorted: Items Put to Wall

    items_sorted --> complete: All Items Sorted
    items_sorted --> exception: Item Missing

    exception --> resolved: Found Item
    exception --> partial: Proceed Partial
    resolved --> items_sorted

    timed_out --> partial: Proceed Without
    partial --> complete: Best Effort

    complete --> [*]: Done
```

### Multi-Order Consolidation Timeline

```mermaid
sequenceDiagram
    participant P1 as Picker Route 1
    participant P2 as Picker Route 2
    participant P3 as Picker Route 3
    participant CONV as Conveyor
    participant WF as Consolidation WF
    participant W as Wall Worker

    par Parallel Picking
        P1->>P1: Pick Zone A items
        P2->>P2: Pick Zone B items
        P3->>P3: Pick Zone C items
    end

    P1->>CONV: Tote 1 to conveyor
    CONV->>WF: Signal: toteArrived (1/3)

    P2->>CONV: Tote 2 to conveyor
    CONV->>WF: Signal: toteArrived (2/3)

    P3->>CONV: Tote 3 to conveyor
    CONV->>WF: Signal: toteArrived (3/3)

    Note over WF: All totes received!

    WF->>W: Assign to put wall station
    W->>W: Scan each tote
    W->>W: Sort items to order slot
    W->>WF: Consolidation complete
```

## Activities Used

| Activity | Purpose | On Failure |
|----------|---------|------------|
| `CreateConsolidationUnit` | Creates a consolidation container | Return error |
| `ConsolidateItems` | Physically moves items to consolidation container | Return error |
| `VerifyConsolidation` | Verifies all expected items are present | Return error |
| `CompleteConsolidation` | Marks consolidation as complete | Return error |
| `ConfirmUnitConsolidation` | Confirms unit-level consolidation (if tracking enabled) | Log warning, continue |

## Unit-Level Tracking

When `useUnitTracking` is enabled:

1. Each unit is confirmed individually via `ConfirmUnitConsolidation`
2. Uses consolidationID as the destination bin
3. Partial failures are logged but don't fail the workflow
4. Parent workflow handles partial failure scenarios

## Error Handling

| Scenario | Handling |
|----------|----------|
| Tote arrival timeout | Proceed with partial consolidation, log warning |
| Consolidation unit creation fails | Return error |
| Item consolidation fails | Return error |
| Verification fails | Return error |
| Unit confirmation fails | Log warning, continue with other units |

## When Consolidation is Required

| Condition | Consolidation Required |
|-----------|----------------------|
| 1-3 items, single zone | No |
| 4-20 items, single zone | Yes |
| Any item count, multi-zone | Yes |
| Multi-route order | Yes |

## Usage Example

```go
// Called from WES Execution Workflow
consolidationInput := map[string]interface{}{
    "orderId":            input.OrderID,
    "waveId":             input.WaveID,
    "pickedItems":        pickResult.PickedItems,
    "isMultiRoute":       input.IsMultiRoute,
    "expectedRouteCount": input.ExpectedRouteCount,
    "expectedTotes":      input.ExpectedTotes,
    "unitIds":            input.UnitIDs,
    "pathId":             input.PathID,
}

err := workflow.ExecuteActivity(ctx, "ConsolidationWorkflow", consolidationInput).Get(ctx, nil)
```

## Related Documentation

- [WES Execution Workflow](./wes-execution) - Parent workflow
- [Picking Workflow](./picking) - Previous step
- [Packing Workflow](./packing) - Next step
- [Consolidation Activities](../activities/consolidation-activities) - Activity details
