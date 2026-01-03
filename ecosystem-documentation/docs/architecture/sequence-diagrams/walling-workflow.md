---
sidebar_position: 7
---

# Walling Workflow

The Walling Workflow handles the put-wall sorting process where walliners (workers) sort picked items from totes into order-specific bins before packing.

## Overview

Walling is used in the `pick_wall_pack` process path for medium-sized orders (4-20 items). It improves packing efficiency by pre-sorting items.

```mermaid
graph LR
    subgraph "Picking"
        Pick[Picker] --> Tote[Tote with items]
    end

    subgraph "Walling"
        Tote --> Wall[Put Wall]
        Wall --> Bin1[Order 1 Bin]
        Wall --> Bin2[Order 2 Bin]
        Wall --> Bin3[Order 3 Bin]
    end

    subgraph "Packing"
        Bin1 --> Pack[Packer]
    end
```

## Complete Flow

```mermaid
sequenceDiagram
    autonumber
    participant WES as WES Workflow
    participant WS as Walling Service
    participant Sim as Walling Simulator
    participant Orch as Orchestrator

    Note over WES: Picking stage completed
    WES->>WS: Create walling task
    Note over WS: Task created (pending)

    WES->>WES: Wait for wallingCompleted signal
    Note over WES: 15 minute timeout

    Sim->>WS: GET /tasks/pending
    WS-->>Sim: Pending walling tasks

    Sim->>WS: POST /tasks/{id}/assign
    Note over WS: Task assigned to walliner

    loop For each item
        Sim->>WS: POST /tasks/{id}/sort
        Note over WS: Item sorted to bin
    end

    alt All items sorted
        WS->>WS: Auto-complete task
    else Manual completion
        Sim->>WS: POST /tasks/{id}/complete
    end

    Sim->>Orch: POST /signals/walling-completed
    Orch->>WES: Signal wallingCompleted

    Note over WES: Advance to packing stage
```

## Task Lifecycle

```mermaid
stateDiagram-v2
    [*] --> pending: WES creates task

    pending --> assigned: Walliner assigned
    Note right of assigned: Scan badge + station

    assigned --> in_progress: First item sorted
    Note right of in_progress: Sorting items

    in_progress --> in_progress: Sort item
    in_progress --> completed: All items sorted

    completed --> [*]: Signal sent to WES
```

## Sorting Process Detail

```mermaid
sequenceDiagram
    participant W as Walliner
    participant PDA as PDA/Scanner
    participant WS as Walling Service
    participant PW as Put Wall

    W->>PDA: Scan tote barcode
    PDA->>WS: GET /tasks?toteId={toteId}
    WS-->>PDA: Task with items to sort

    loop For each item in tote
        W->>PDA: Scan item SKU
        PDA->>PW: Light indicates target bin

        W->>W: Place item in lit bin
        W->>PDA: Scan bin barcode

        PDA->>WS: POST /tasks/{id}/sort
        Note over WS: {sku, quantity, fromToteId}
        WS-->>PDA: Sort confirmed
    end

    Note over PDA: All items sorted
    PDA->>WS: Task auto-completed
```

## Signal Bridge

The signal from Walling Service to WES Workflow goes through the Orchestrator:

```mermaid
sequenceDiagram
    participant WS as Walling Service
    participant Sim as Simulator/Worker
    participant Orch as Orchestrator
    participant Temp as Temporal
    participant WES as WES Workflow

    WS-->>Sim: Task completed event
    Sim->>Orch: POST /api/v1/signals/walling-completed
    Note over Orch: Payload: orderId, taskId, routeId

    Orch->>Temp: SignalWorkflow("wes-execution-{orderId}", "wallingCompleted")
    Temp->>WES: Deliver signal
    WES->>WES: Resume workflow
```

### Signal Payload

```json
{
  "orderId": "ORD-12345",
  "taskId": "WT-a1b2c3d4",
  "routeId": "RT-xyz",
  "sortedItems": [
    {"sku": "SKU-001", "quantity": 2, "slotId": "BIN-A1"},
    {"sku": "SKU-002", "quantity": 1, "slotId": "BIN-A1"}
  ]
}
```

## Timeout Handling

```mermaid
sequenceDiagram
    participant WES as WES Workflow
    participant Timer as 15min Timer

    WES->>WES: Start walling stage
    WES->>Timer: Set 15 minute timeout

    alt Signal received in time
        Note over WES: wallingCompleted received
        WES->>WES: Continue to packing
    else Timeout
        Timer-->>WES: Timeout fired
        WES->>WES: Mark stage failed
        WES-->>WES: Return failure
        Note over WES: Parent handles compensation
    end
```

## Put Wall Configuration

A typical put wall has multiple slots, each assigned to an order:

```
+-------+-------+-------+-------+
| BIN-1 | BIN-2 | BIN-3 | BIN-4 |
| ORD-A | ORD-B | ORD-C | ORD-D |
+-------+-------+-------+-------+
| BIN-5 | BIN-6 | BIN-7 | BIN-8 |
| ORD-E | ORD-F | ORD-G | ORD-H |
+-------+-------+-------+-------+
```

## Error Scenarios

### Item Not Found
```mermaid
sequenceDiagram
    participant W as Walliner
    participant WS as Walling Service

    W->>WS: POST /tasks/{id}/sort (wrong SKU)
    WS-->>W: Error: Item not found in task
    Note over W: Re-scan correct item
```

### Wrong Quantity
```mermaid
sequenceDiagram
    participant W as Walliner
    participant WS as Walling Service

    W->>WS: POST /tasks/{id}/sort (qty: 5)
    Note over WS: Only 3 remaining
    WS->>WS: Limit to remaining (3)
    WS-->>W: Sorted 3 items
```

## Related Documentation

- [Walling Service](/services/walling-service) - Service documentation
- [WES Execution](/architecture/sequence-diagrams/wes-execution) - Parent workflow
- [WallingTask Aggregate](/domain-driven-design/aggregates/walling-task) - Domain model
