---
sidebar_position: 15
slug: /temporal/workflows/service-packing
---

# Packing Service - PackingWorkflow

Service-level workflow that handles the packing process with signal-based progress tracking.

## Overview

The Packing Service's PackingWorkflow provides:
1. Signal-based packer assignment
2. Real-time packing progress via signals (verify, seal, label)
3. Automatic task lifecycle management
4. Package and tracking number generation

## Configuration

| Property | Value |
|----------|-------|
| Task Queue | `packing-queue` |
| Execution Timeout | 4 hours |
| Activity Timeout | 10 minutes |
| Heartbeat Timeout | 30 seconds |
| Packing Timeout | 1 hour |

## Input

```go
// PackingWorkflowInput represents the input for the packing workflow
type PackingWorkflowInput struct {
    OrderID string `json:"orderId"`
}
```

## Output

```go
// PackingWorkflowResult represents the result of the packing workflow
type PackingWorkflowResult struct {
    PackageID      string  `json:"packageId"`
    TrackingNumber string  `json:"trackingNumber"`
    Carrier        string  `json:"carrier"`
    Weight         float64 `json:"weight"`
    Success        bool    `json:"success"`
    Error          string  `json:"error,omitempty"`
}
```

## Workflow Steps

```mermaid
sequenceDiagram
    participant WES as WESExecution
    participant PACK as PackingWorkflow
    participant CT as CreatePackTask
    participant PA as WaitForPacker
    participant VE as VerifyItems
    participant SE as SealPackage
    participant LA as ApplyLabel

    WES->>PACK: Start packing (packing-queue)

    Note over PACK: Step 1: Create Pack Task
    PACK->>CT: CreatePackTask activity
    CT-->>PACK: TaskID

    Note over PACK: Step 2: Wait for Packer Assignment
    PACK->>PA: Wait for packerAssigned signal
    PA-->>PACK: PackerID, Station

    Note over PACK: Step 3: Assign Packer
    PACK->>PACK: AssignPacker activity

    Note over PACK: Step 4: Process Packing via Signals
    loop Packing Process
        PACK->>VE: Wait for itemVerified signal
        PACK->>SE: Wait for packageSealed signal
        PACK->>LA: Wait for labelApplied signal
    end

    Note over PACK: Step 5: Wait for Completion
    PACK->>PACK: Wait for packingComplete signal

    Note over PACK: Step 6: Complete Task
    PACK->>PACK: CompletePackTask activity

    PACK-->>WES: PackingWorkflowResult
```

## Signals

| Signal | Payload | Purpose |
|--------|---------|---------|
| `packerAssigned` | `PackerInfo` | Packer claims the task |
| `itemVerified` | `ItemVerification` | Item verification completed |
| `packageSealed` | `PackageSealed` | Package sealed with weight |
| `labelApplied` | `LabelInfo` | Tracking label applied |
| `packingComplete` | `{Success: bool}` | All packing complete |

### Signal Payloads

```go
// PackerInfo signal payload
type PackerInfo struct {
    PackerID string `json:"packerId"`
    Station  string `json:"station"`
}

// ItemVerification signal payload
type ItemVerification struct {
    SKU      string `json:"sku"`
    Verified bool   `json:"verified"`
}

// PackageSealed signal payload
type PackageSealed struct {
    PackageID string  `json:"packageId"`
    Weight    float64 `json:"weight"`
}

// LabelInfo signal payload
type LabelInfo struct {
    TrackingNumber string `json:"trackingNumber"`
    Carrier        string `json:"carrier"`
}
```

## Signal Flow

```mermaid
sequenceDiagram
    participant STATION as Pack Station
    participant WF as PackingWorkflow
    participant SCALE as Scale System

    Note over STATION: Packer starts packing
    STATION->>WF: packerAssigned signal

    loop For each item
        Note over STATION: Packer verifies item
        STATION->>WF: itemVerified signal
    end

    Note over STATION: Package is sealed
    SCALE->>WF: packageSealed signal (with weight)

    Note over STATION: Label is applied
    STATION->>WF: labelApplied signal

    STATION->>WF: packingComplete signal
```

## Activities Used

| Activity | Purpose | On Failure |
|----------|---------|------------|
| `CreatePackTask` | Creates pack task | Return error |
| `AssignPacker` | Records packer assignment | Log warning |
| `CompletePackTask` | Marks task complete | Log warning |

## Error Handling

| Scenario | Handling |
|----------|----------|
| Task creation fails | Return error |
| Packer assignment timeout (20 min) | Return timeout error |
| Packing timeout (1 hour) | Set error, complete workflow |

## Success Criteria

Packing is considered successful when:
- `TrackingNumber` is not empty (label was applied)

```go
result.Success = packageInfo.TrackingNumber != ""
```

## Usage Example

```go
// Called as child workflow from WES service
childWorkflowOptions := workflow.ChildWorkflowOptions{
    TaskQueue: "packing-queue",
    RetryPolicy: &temporal.RetryPolicy{
        MaximumAttempts: 3,
    },
}
childCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)

input := map[string]interface{}{
    "orderId": "ORD-123",
}

var result PackingWorkflowResult
err := workflow.ExecuteChildWorkflow(childCtx, "PackingWorkflow", input).Get(ctx, &result)
```

## Related Documentation

- [Orchestrator Packing Workflow](./packing) - Orchestrator version
- [WES Service Workflow](./service-wes) - Parent workflow
- [Packing Activities](../activities/packing-activities) - Activity details
