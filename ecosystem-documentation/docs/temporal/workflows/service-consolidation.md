---
sidebar_position: 16
slug: /temporal/workflows/service-consolidation
---

# Consolidation Service - ConsolidationWorkflow

Service-level workflow that handles item consolidation with signal-based progress tracking.

## Overview

The Consolidation Service's ConsolidationWorkflow provides:
1. Signal-based station assignment
2. Real-time consolidation progress via signals
3. Destination bin management
4. Multi-tote consolidation support

## Configuration

| Property | Value |
|----------|-------|
| Task Queue | `consolidation-queue` |
| Execution Timeout | 4 hours |
| Activity Timeout | 10 minutes |
| Heartbeat Timeout | 30 seconds |
| Consolidation Timeout | 1 hour |

## Input

```go
// ConsolidationWorkflowInput represents the input for the consolidation workflow
type ConsolidationWorkflowInput struct {
    OrderID     string       `json:"orderId"`
    PickedItems []PickedItem `json:"pickedItems"`
}

// PickedItem represents a picked item from the picking workflow
type PickedItem struct {
    SKU        string `json:"sku"`
    Quantity   int    `json:"quantity"`
    LocationID string `json:"locationId"`
    ToteID     string `json:"toteId"`
}
```

## Output

```go
// ConsolidationWorkflowResult represents the result of the consolidation workflow
type ConsolidationWorkflowResult struct {
    ConsolidationID   string `json:"consolidationId"`
    DestinationBin    string `json:"destinationBin"`
    TotalConsolidated int    `json:"totalConsolidated"`
    Success           bool   `json:"success"`
    Error             string `json:"error,omitempty"`
}
```

## Workflow Steps

```mermaid
sequenceDiagram
    participant WES as WESExecution
    participant CON as ConsolidationWorkflow
    participant CU as CreateUnit
    participant WS as WaitForStation
    participant AS as AssignStation
    participant PI as ProcessItems

    WES->>CON: Start consolidation (consolidation-queue)

    Note over CON: Step 1: Create Consolidation Unit
    CON->>CU: CreateConsolidationUnit activity
    CU-->>CON: ConsolidationID

    Note over CON: Step 2: Wait for Station Assignment
    CON->>WS: Wait for stationAssigned signal
    WS-->>CON: Station, WorkerID, DestinationBin

    Note over CON: Step 3: Assign Station
    CON->>AS: AssignStation activity
    AS-->>CON: Success

    Note over CON: Step 4: Process Consolidation via Signals
    loop For each item
        CON->>PI: Wait for itemConsolidated signal
        PI-->>CON: SKU, Quantity, ToteID
    end

    Note over CON: Step 5: Wait for Completion
    CON->>CON: Wait for consolidationComplete signal

    Note over CON: Step 6: Complete Consolidation
    CON->>CON: CompleteConsolidation activity

    CON-->>WES: ConsolidationWorkflowResult
```

## Signals

| Signal | Payload | Timeout | Purpose |
|--------|---------|---------|---------|
| `stationAssigned` | `StationInfo` | 15 minutes | Station claims the work |
| `itemConsolidated` | `ConsolidatedItem` | - | Item moved to destination |
| `consolidationComplete` | `CompletionInfo` | - | All items consolidated |

### Signal Payloads

```go
// StationInfo signal payload
type StationInfo struct {
    Station        string `json:"station"`
    WorkerID       string `json:"workerId"`
    DestinationBin string `json:"destinationBin"`
}

// ConsolidatedItem signal payload
type ConsolidatedItem struct {
    SKU      string `json:"sku"`
    Quantity int    `json:"quantity"`
    ToteID   string `json:"toteId"`  // Source tote
}

// CompletionInfo signal payload
type CompletionInfo struct {
    Success           bool `json:"success"`
    TotalConsolidated int  `json:"totalConsolidated"`
}
```

## Signal Flow

```mermaid
sequenceDiagram
    participant WALL as Put-Wall Station
    participant WF as ConsolidationWorkflow
    participant WMS as WMS System

    Note over WALL: Worker claims station
    WALL->>WF: stationAssigned signal

    loop For each source tote
        Note over WALL: Worker scans item from tote
        WALL->>WF: itemConsolidated signal
        WF->>WF: Update consolidated count
    end

    Note over WALL: All items consolidated
    WALL->>WF: consolidationComplete signal
```

## Activities Used

| Activity | Purpose | On Failure |
|----------|---------|------------|
| `CreateConsolidationUnit` | Creates consolidation unit | Return error |
| `AssignStation` | Records station assignment | Log warning |
| `CompleteConsolidation` | Marks consolidation complete | Log warning |

## Consolidation Flow

```mermaid
graph TD
    subgraph "Source Totes"
        T1[Tote A - Zone 1]
        T2[Tote B - Zone 2]
        T3[Tote C - Zone 3]
    end

    subgraph "Put Wall Station"
        BIN[Destination Bin]
    end

    T1 -->|Consolidate| BIN
    T2 -->|Consolidate| BIN
    T3 -->|Consolidate| BIN

    BIN --> PACK[Ready for Packing]
```

## Error Handling

| Scenario | Handling |
|----------|----------|
| Unit creation fails | Return error |
| Station assignment timeout (15 min) | Return timeout error |
| Consolidation timeout (1 hour) | Set error, complete workflow |

## Success Criteria

Consolidation is considered successful when:
- `TotalConsolidated > 0` (at least one item consolidated)

```go
result.Success = totalConsolidated > 0
```

## Usage Example

```go
// Called as child workflow from WES service
childWorkflowOptions := workflow.ChildWorkflowOptions{
    TaskQueue: "consolidation-queue",
    RetryPolicy: &temporal.RetryPolicy{
        MaximumAttempts: 3,
    },
}
childCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)

input := map[string]interface{}{
    "orderId":     "ORD-123",
    "pickedItems": pickedItems,
}

var result ConsolidationWorkflowResult
err := workflow.ExecuteChildWorkflow(childCtx, "ConsolidationWorkflow", input).Get(ctx, &result)
```

## Related Documentation

- [Orchestrator Consolidation Workflow](./consolidation) - Orchestrator version
- [WES Service Workflow](./service-wes) - Parent workflow
- [Consolidation Activities](../activities/consolidation-activities) - Activity details
