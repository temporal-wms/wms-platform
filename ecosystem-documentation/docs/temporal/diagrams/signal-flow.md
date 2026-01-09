---
sidebar_position: 3
slug: /temporal/diagrams/signal-flow
---

# Signal Flow Diagrams

Visual representation of signal timing and flow between workflows and external systems.

## Signal Overview

```mermaid
graph TD
    subgraph "External Systems"
        WMS[WMS System]
        HANDHELD[Handheld Devices]
        CONVEY[Conveyor System]
        CARRIER[Carrier API]
    end

    subgraph "Temporal Workflows"
        PL[PlanningWorkflow]
        PICK[PickingWorkflow]
        CON[ConsolidationWorkflow]
        PACK[PackingWorkflow]
        SHIP[ShippingWorkflow]
        GW[GiftWrapWorkflow]
        IB[InboundWorkflow]
        WES[WESWorkflow]
    end

    WMS -->|waveAssigned| PL
    HANDHELD -->|workerAssigned| PICK
    HANDHELD -->|itemPicked| PICK
    HANDHELD -->|pickingComplete| PICK
    HANDHELD -->|pickException| PICK

    CONVEY -->|toteArrived| CON
    HANDHELD -->|stationAssigned| CON
    HANDHELD -->|itemConsolidated| CON
    HANDHELD -->|consolidationComplete| CON

    HANDHELD -->|packerAssigned| PACK
    HANDHELD -->|itemVerified| PACK
    HANDHELD -->|packageSealed| PACK
    HANDHELD -->|labelApplied| PACK
    HANDHELD -->|packingComplete| PACK

    CARRIER -->|packageScanned| SHIP
    HANDHELD -->|shipConfirmed| SHIP

    HANDHELD -->|gift-wrap-completed| GW

    WMS -->|shipmentArrived| IB

    PICK -->|pickCompleted| WES
    CON -->|wallingCompleted| WES
```

---

## Picking Signal Timeline

```mermaid
sequenceDiagram
    participant W as Worker
    participant H as Handheld
    participant API as Signal API
    participant WF as PickingWorkflow

    Note over WF: Workflow waiting for worker

    W->>H: Scan task barcode
    H->>API: POST /signal/workerAssigned
    API->>WF: Signal: workerAssigned
    Note over WF: Timeout: 30 minutes

    WF->>WF: Create pick list

    loop For each item
        W->>H: Scan location
        W->>H: Scan item
        W->>H: Confirm quantity
        H->>API: POST /signal/itemPicked
        API->>WF: Signal: itemPicked
    end

    alt All items picked
        H->>API: POST /signal/pickingComplete
        API->>WF: Signal: pickingComplete {success: true}
    else Item not found
        H->>API: POST /signal/pickException
        API->>WF: Signal: pickException
        WF->>WF: Handle shortage
    end

    WF->>WF: Complete workflow
```

### Picking Signal Payloads

```go
// workerAssigned
type WorkerAssignment struct {
    WorkerID string `json:"workerId"`
    ToteID   string `json:"toteId"`
}

// itemPicked
type PickedItem struct {
    SKU        string `json:"sku"`
    LocationID string `json:"locationId"`
    Quantity   int    `json:"quantity"`
    ToteID     string `json:"toteId"`
}

// pickException
type PickException struct {
    SKU       string `json:"sku"`
    Reason    string `json:"reason"`     // not_found, damaged, quantity_mismatch
    Available int    `json:"available"`
}
```

---

## Consolidation Signal Timeline

```mermaid
sequenceDiagram
    participant Conv as Conveyor
    participant Station as Station
    participant API as Signal API
    participant WF as ConsolidationWorkflow

    Note over WF: Waiting for totes (multi-route order)

    loop For each route/tote
        Conv->>Conv: Tote arrives at station
        Conv->>API: POST /signal/toteArrived
        API->>WF: Signal: toteArrived
        Note over WF: Track: received X of Y totes
    end

    Note over WF: All totes arrived or timeout

    Station->>API: POST /signal/stationAssigned
    API->>WF: Signal: stationAssigned
    Note over WF: Timeout: 30 minutes

    loop For each item
        Station->>API: POST /signal/itemConsolidated
        API->>WF: Signal: itemConsolidated
    end

    Station->>API: POST /signal/consolidationComplete
    API->>WF: Signal: consolidationComplete
```

### Consolidation Signal Payloads

```go
// toteArrived
type ToteArrivedSignal struct {
    ToteID     string `json:"toteId"`
    RouteID    string `json:"routeId"`
    RouteIndex int    `json:"routeIndex"`
    ArrivedAt  string `json:"arrivedAt"`
}

// stationAssigned
type StationInfo struct {
    Station        string `json:"station"`
    WorkerID       string `json:"workerId"`
    DestinationBin string `json:"destinationBin"`
}

// itemConsolidated
type ConsolidatedItem struct {
    SKU            string `json:"sku"`
    SourceToteID   string `json:"sourceToteId"`
    DestinationBin string `json:"destinationBin"`
}
```

---

## Packing Signal Timeline

```mermaid
sequenceDiagram
    participant P as Packer
    participant H as Handheld
    participant API as Signal API
    participant WF as PackingWorkflow

    Note over WF: Workflow waiting for packer

    P->>H: Scan pack task
    H->>API: POST /signal/packerAssigned
    API->>WF: Signal: packerAssigned
    Note over WF: Timeout: 20 minutes

    WF->>WF: Determine materials

    loop For each item
        P->>H: Scan item barcode
        H->>API: POST /signal/itemVerified
        API->>WF: Signal: itemVerified
    end

    P->>H: Seal package
    H->>API: POST /signal/packageSealed
    API->>WF: Signal: packageSealed

    WF->>WF: Print label

    P->>H: Apply label
    H->>API: POST /signal/labelApplied
    API->>WF: Signal: labelApplied

    H->>API: POST /signal/packingComplete
    API->>WF: Signal: packingComplete
```

### Packing Signal Payloads

```go
// packerAssigned
type PackerInfo struct {
    PackerID string `json:"packerId"`
    Station  string `json:"station"`
}

// itemVerified
type ItemVerification struct {
    SKU      string `json:"sku"`
    Verified bool   `json:"verified"`
}

// packageSealed
type PackageSealed struct {
    PackageID string  `json:"packageId"`
    Weight    float64 `json:"weight"`
}

// labelApplied
type LabelInfo struct {
    TrackingNumber string `json:"trackingNumber"`
    Carrier        string `json:"carrier"`
}
```

---

## Shipping Signal Timeline

```mermaid
sequenceDiagram
    participant SLAM as SLAM Station
    participant Carrier as Carrier System
    participant API as Signal API
    participant WF as ShippingWorkflow

    Note over WF: Package arrives at SLAM

    WF->>WF: Scan package (activity)
    WF->>WF: Generate label (activity)
    WF->>WF: Apply label (activity)
    WF->>Carrier: Add to manifest (activity)
    Carrier-->>WF: Manifest confirmed

    alt Auto-confirm after manifest
        WF->>WF: Complete shipping
    else Wait for physical scan
        Note over WF: Waiting for ship confirmation
        Carrier->>API: Webhook: package scanned
        API->>WF: Signal: packageScanned
        WF->>WF: Complete shipping
    end

    alt Manual confirmation needed
        SLAM->>API: POST /signal/shipConfirmed
        API->>WF: Signal: shipConfirmed
    end
```

### Shipping Signal Payloads

```go
// shipConfirmed
type ShipConfirmation struct {
    ShippedAt         time.Time  `json:"shippedAt"`
    EstimatedDelivery *time.Time `json:"estimatedDelivery,omitempty"`
}

// packageScanned (from carrier webhook)
type PackageScan struct {
    Location  string    `json:"location"`
    ScannedAt time.Time `json:"scannedAt"`
}
```

---

## Wave Assignment Signal

```mermaid
sequenceDiagram
    participant WMS as Wave Planner
    participant API as Temporal API
    participant PL as PlanningWorkflow

    Note over PL: Waiting for wave assignment
    Note over PL: Timeout based on priority

    WMS->>WMS: Plan wave batch
    WMS->>WMS: Optimize pick paths
    WMS->>WMS: Assign orders to wave

    WMS->>API: SignalWorkflow(waveAssigned)
    API->>PL: Signal: waveAssigned

    PL->>PL: Continue with WES execution
```

### Wave Assignment Payload

```go
type WaveAssignment struct {
    WaveID         string    `json:"waveId"`
    ScheduledStart time.Time `json:"scheduledStart"`
}
```

### Priority-Based Timeouts

| Priority | Timeout | Description |
|----------|---------|-------------|
| `same_day` | 30 minutes | Must ship today |
| `next_day` | 2 hours | Ship by tomorrow |
| `standard` | 4 hours | Standard shipping |

---

## Inbound Fulfillment Signal

```mermaid
sequenceDiagram
    participant Dock as Receiving Dock
    participant API as Signal API
    participant IB as InboundFulfillmentWorkflow

    Note over IB: Shipment expected
    Note over IB: Timeout: Expected + 4 hours

    Dock->>Dock: Truck arrives
    Dock->>Dock: Check-in shipment
    Dock->>API: POST /signal/shipmentArrived
    API->>IB: Signal: shipmentArrived

    IB->>IB: Start unload activities
    IB->>IB: Quality check
    IB->>IB: Put away
```

### Inbound Signal Payload

```go
type ShipmentArrivalSignal struct {
    ShipmentID string    `json:"shipmentId"`
    DockID     string    `json:"dockId"`
    ArrivedAt  time.Time `json:"arrivedAt"`
}
```

---

## Gift Wrap Signal

```mermaid
sequenceDiagram
    participant W as Gift Wrapper
    participant H as Handheld
    participant API as Signal API
    participant GW as GiftWrapWorkflow

    Note over GW: Workflow waiting (20 min timeout)

    W->>H: Complete wrapping
    W->>H: Scan task barcode
    H->>API: POST /signal/gift-wrap-completed
    API->>GW: Signal: gift-wrap-completed

    GW->>GW: Mark task complete
    GW->>GW: Continue to packing
```

### Gift Wrap Signal Payload

```go
type GiftWrapCompletedSignal struct {
    TaskID      string `json:"taskId"`
    WorkerID    string `json:"workerId"`
    CompletedAt string `json:"completedAt"`
}
```

---

## WES Completion Signals

Signals sent from child workflows back to WES:

```mermaid
sequenceDiagram
    participant WES as WESExecutionWorkflow
    participant PICK as PickingWorkflow
    participant CON as ConsolidationWorkflow
    participant PACK as PackingWorkflow

    WES->>PICK: Execute child workflow
    PICK-->>WES: Signal: pickCompleted

    alt Consolidation required
        WES->>CON: Execute child workflow
        CON-->>WES: Signal: wallingCompleted
    end

    WES->>PACK: Execute child workflow
    Note over WES: Child returns result directly
```

### WES Signal Payloads

```go
// pickCompleted (to orchestrator)
type PickCompletedSignal struct {
    TaskID      string       `json:"taskId"`
    PickedItems []PickedItem `json:"pickedItems"`
    Success     bool         `json:"success"`
}

// wallingCompleted
type WallingCompletedSignal struct {
    TaskID      string   `json:"taskId"`
    RouteID     string   `json:"routeId"`
    SortedItems []string `json:"sortedItems"`
    Success     bool     `json:"success"`
}
```

---

## Signal Timeout Summary

| Workflow | Signal | Timeout | On Timeout |
|----------|--------|---------|------------|
| PlanningWorkflow | `waveAssigned` | 30min - 4h | Fail workflow |
| PickingWorkflow | `workerAssigned` | 30 min | Fail workflow |
| PickingWorkflow | `itemPicked` | Per item | Continue waiting |
| PickingWorkflow | `pickingComplete` | 30 min | Fail workflow |
| ConsolidationWorkflow | `toteArrived` | 30 min | Proceed partial |
| ConsolidationWorkflow | `stationAssigned` | 30 min | Fail workflow |
| PackingWorkflow | `packerAssigned` | 20 min | Fail workflow |
| PackingWorkflow | `packingComplete` | 1 hour | Fail workflow |
| ShippingWorkflow | `shipConfirmed` | Auto or signal | Auto-complete |
| GiftWrapWorkflow | `gift-wrap-completed` | 20 min | Poll status |
| InboundFulfillment | `shipmentArrived` | Expected + 4h | Alert |
| WESExecution | `wallingCompleted` | 15 min | Fail stage |

---

## Sending Signals

### Via Temporal SDK

```go
err := client.SignalWorkflow(ctx, workflowID, runID, "signalName", payload)
```

### Via HTTP Bridge

```bash
curl -X POST "http://temporal-bridge/signal" \
  -H "Content-Type: application/json" \
  -d '{
    "workflowId": "picking-ORD-123",
    "signalName": "itemPicked",
    "payload": {"sku": "SKU-001", "quantity": 1}
  }'
```

### Via tctl CLI

```bash
tctl workflow signal \
  --workflow_id "picking-ORD-123" \
  --name "itemPicked" \
  --input '{"sku":"SKU-001","quantity":1}'
```

## Related Documentation

- [Signals & Queries](../signals-queries) - Complete signal reference
- [Workflow Hierarchy](./workflow-hierarchy) - Parent-child relationships
- [Order Flow](./order-flow) - Complete order processing flow
