---
sidebar_position: 20
slug: /temporal/signals-queries
---

# Signals & Queries

Reference for all workflow signals and query handlers in the WMS Platform.

## Signals Overview

Signals enable external events to communicate with running workflows. They are used extensively for:
- Worker assignment notifications
- Task completion notifications
- Physical event confirmations (scans, arrivals)
- Multi-route tote coordination

## Signal Reference

### Order Fulfillment Signals

| Signal | Workflow | Payload | Purpose |
|--------|----------|---------|---------|
| `waveAssigned` | PlanningWorkflow | `WaveAssignment` | Wave assignment notification |

```go
type WaveAssignment struct {
    WaveID         string    `json:"waveId"`
    ScheduledStart time.Time `json:"scheduledStart"`
}
```

---

### Picking Signals

| Signal | Workflow | Payload | Purpose |
|--------|----------|---------|---------|
| `pickCompleted` | OrchestratedPickingWorkflow | `PickCompletedSignal` | Picking complete |
| `workerAssigned` | PickingWorkflow (service) | `WorkerAssignment` | Worker claims task |
| `itemPicked` | PickingWorkflow (service) | `PickedItem` | Item pick confirmation |
| `pickingComplete` | PickingWorkflow (service) | `{Success: bool}` | All items picked |
| `pickException` | PickingWorkflow (service) | `PickException` | Item pick failed |

```go
type PickCompletedSignal struct {
    TaskID      string       `json:"taskId"`
    PickedItems []PickedItem `json:"pickedItems"`
    Success     bool         `json:"success"`
}

type WorkerAssignment struct {
    WorkerID string `json:"workerId"`
    ToteID   string `json:"toteId"`
}

type PickException struct {
    SKU       string `json:"sku"`
    Reason    string `json:"reason"`
    Available int    `json:"available"`
}
```

---

### Consolidation Signals

| Signal | Workflow | Payload | Purpose |
|--------|----------|---------|---------|
| `toteArrived` | ConsolidationWorkflow | `ToteArrivedSignal` | Tote arrival for multi-route |
| `stationAssigned` | ConsolidationWorkflow (service) | `StationInfo` | Station claims work |
| `itemConsolidated` | ConsolidationWorkflow (service) | `ConsolidatedItem` | Item consolidated |
| `consolidationComplete` | ConsolidationWorkflow (service) | `CompletionInfo` | All items done |

```go
type ToteArrivedSignal struct {
    ToteID     string `json:"toteId"`
    RouteID    string `json:"routeId"`
    RouteIndex int    `json:"routeIndex"`
    ArrivedAt  string `json:"arrivedAt"`
}

type StationInfo struct {
    Station        string `json:"station"`
    WorkerID       string `json:"workerId"`
    DestinationBin string `json:"destinationBin"`
}
```

---

### Packing Signals

| Signal | Workflow | Payload | Purpose |
|--------|----------|---------|---------|
| `packerAssigned` | PackingWorkflow (service) | `PackerInfo` | Packer claims task |
| `itemVerified` | PackingWorkflow (service) | `ItemVerification` | Item verified |
| `packageSealed` | PackingWorkflow (service) | `PackageSealed` | Package sealed |
| `labelApplied` | PackingWorkflow (service) | `LabelInfo` | Label applied |
| `packingComplete` | PackingWorkflow (service) | `{Success: bool}` | Packing done |

```go
type PackerInfo struct {
    PackerID string `json:"packerId"`
    Station  string `json:"station"`
}

type PackageSealed struct {
    PackageID string  `json:"packageId"`
    Weight    float64 `json:"weight"`
}

type LabelInfo struct {
    TrackingNumber string `json:"trackingNumber"`
    Carrier        string `json:"carrier"`
}
```

---

### Shipping Signals

| Signal | Workflow | Payload | Purpose |
|--------|----------|---------|---------|
| `shipConfirmed` | ShippingWorkflow (service) | `ShipConfirmation` | Manual ship confirmation |
| `packageScanned` | ShippingWorkflow (service) | `PackageScan` | Carrier scan (auto-confirm) |

```go
type ShipConfirmation struct {
    ShippedAt         time.Time  `json:"shippedAt"`
    EstimatedDelivery *time.Time `json:"estimatedDelivery,omitempty"`
}

type PackageScan struct {
    Location  string    `json:"location"`
    ScannedAt time.Time `json:"scannedAt"`
}
```

---

### WES Signals

| Signal | Workflow | Payload | Purpose |
|--------|----------|---------|---------|
| `wallingCompleted` | WESExecutionWorkflow | `WallingCompletedSignal` | Walling stage complete |

```go
type WallingCompletedSignal struct {
    TaskID      string   `json:"taskId"`
    RouteID     string   `json:"routeId"`
    SortedItems []string `json:"sortedItems"`
    Success     bool     `json:"success"`
}
```

---

### Gift Wrap Signals

| Signal | Workflow | Payload | Purpose |
|--------|----------|---------|---------|
| `gift-wrap-completed` | GiftWrapWorkflow | `GiftWrapCompletedSignal` | Gift wrap done |

```go
type GiftWrapCompletedSignal struct {
    TaskID      string `json:"taskId"`
    WorkerID    string `json:"workerId"`
    CompletedAt string `json:"completedAt"`
}
```

---

### Inbound Signals

| Signal | Workflow | Payload | Purpose |
|--------|----------|---------|---------|
| `shipmentArrived` | InboundFulfillmentWorkflow | `ShipmentArrivalSignal` | Shipment arrival |

```go
type ShipmentArrivalSignal struct {
    ShipmentID string    `json:"shipmentId"`
    DockID     string    `json:"dockId"`
    ArrivedAt  time.Time `json:"arrivedAt"`
}
```

---

## Signal Timeouts

| Workflow | Signal | Timeout | On Timeout |
|----------|--------|---------|------------|
| PlanningWorkflow | `waveAssigned` | Priority-based (30min - 4h) | Return error |
| PickingWorkflow | `pickCompleted` | 30 minutes | Return error |
| PickingWorkflow | `workerAssigned` | 30 minutes | Return error |
| ConsolidationWorkflow | `toteArrived` | 30 minutes | Proceed with partial |
| PackingWorkflow | `packerAssigned` | 20 minutes | Return error |
| GiftWrapWorkflow | `gift-wrap-completed` | 20 minutes | Poll status |
| WESExecutionWorkflow | `wallingCompleted` | 15 minutes | Return error |
| InboundFulfillment | `shipmentArrived` | Expected + 4h | Return error |

---

## Query Handlers

### OrderFulfillmentWorkflow

| Query | Returns | Purpose |
|-------|---------|---------|
| `getStatus` | `OrderFulfillmentQueryStatus` | Get current workflow status |

```go
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

**Usage:**
```go
var status OrderFulfillmentQueryStatus
err := we.QueryWorkflow(ctx, &status, "getStatus")
```

---

## Sending Signals

### From External Systems

```go
// Using Temporal SDK
err := client.SignalWorkflow(ctx, workflowID, runID, "waveAssigned", WaveAssignment{
    WaveID:         "WAVE-001",
    ScheduledStart: time.Now().Add(15 * time.Minute),
})
```

### From HTTP Bridge

```bash
# Using tctl CLI
tctl workflow signal --workflow_id "planning-ORD-123" \
  --name "waveAssigned" \
  --input '{"waveId":"WAVE-001","scheduledStart":"2024-01-04T10:00:00Z"}'
```

### From Another Workflow

```go
// Signal external workflow
err := workflow.SignalExternalWorkflow(ctx, workflowID, runID, "signalName", signalPayload).Get(ctx, nil)
```

---

## Signal Patterns

### Wait-for-Signal Pattern

```go
signalCh := workflow.GetSignalChannel(ctx, "signalName")
selector := workflow.NewSelector(ctx)

var received bool
selector.AddReceive(signalCh, func(c workflow.ReceiveChannel, more bool) {
    var payload SignalPayload
    c.Receive(ctx, &payload)
    received = true
})

selector.AddFuture(workflow.NewTimer(ctx, timeout), func(f workflow.Future) {
    // Timeout handling
})

selector.Select(ctx)
```

### Multi-Signal Aggregation

```go
// Collect multiple signals until condition met
receivedTotes := make(map[string]bool)
for len(receivedTotes) < expectedCount {
    selector := workflow.NewSelector(ctx)
    selector.AddReceive(toteArrivalCh, func(c workflow.ReceiveChannel, more bool) {
        var signal ToteArrivedSignal
        c.Receive(ctx, &signal)
        receivedTotes[signal.ToteID] = true
    })
    selector.Select(ctx)
}
```

## Related Documentation

- [Order Fulfillment Workflow](./workflows/order-fulfillment) - Uses queries and signals
- [Planning Workflow](./workflows/planning) - Wave assignment signal
- [Picking Workflow](./workflows/picking) - Pick completion signal
