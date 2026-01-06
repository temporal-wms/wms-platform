---
sidebar_position: 11
---

# Unit Tracking Flow

This document describes the unit-level tracking throughout the warehouse fulfillment process, from receiving to shipping.

## Overview

Unit tracking provides granular visibility into each physical item's journey through the warehouse. Every status change creates an audit record, enabling complete traceability and exception handling at the unit level.

## Complete Unit Lifecycle

```mermaid
sequenceDiagram
    autonumber
    participant R as Receiving
    participant US as Unit Service
    participant O as Orchestrator
    participant PS as Picking Service
    participant CS as Consolidation Service
    participant PK as Packing Service
    participant SS as Shipping Service

    rect rgb(240, 248, 255)
        Note over R,US: Phase 1: Unit Creation
        R->>US: POST /units (create from receiving)
        US->>US: Create unit with status=received
        US->>US: Record movement: → received
        US-->>R: Unit IDs created
    end

    rect rgb(255, 248, 240)
        Note over O,US: Phase 2: Reservation
        O->>US: POST /units/reserve
        US->>US: Find available units by SKU
        US->>US: Update status: received → reserved
        US->>US: Record movement: reserved
        US-->>O: Reserved unit IDs
    end

    rect rgb(240, 255, 240)
        Note over O,PS: Phase 3: Staging & Picking
        O->>US: Stage units for picking
        US->>US: Update status: reserved → staged
        PS->>US: POST /units/{id}/pick
        US->>US: Update status: staged → picked
        US->>US: Assign toteId
        US-->>PS: Pick confirmed
    end

    rect rgb(255, 240, 255)
        Note over CS,US: Phase 4: Consolidation (Multi-item)
        CS->>US: POST /units/{id}/consolidate
        US->>US: Update status: picked → consolidated
        US->>US: Record consolidation bin
        US-->>CS: Consolidation confirmed
    end

    rect rgb(248, 255, 240)
        Note over PK,US: Phase 5: Packing
        PK->>US: POST /units/{id}/pack
        US->>US: Update status → packed
        US->>US: Assign packageId
        US-->>PK: Pack confirmed
    end

    rect rgb(240, 240, 255)
        Note over SS,US: Phase 6: Shipping
        SS->>US: POST /units/{id}/ship
        US->>US: Update status: packed → shipped
        US->>US: Record tracking number
        US-->>SS: Ship confirmed
    end
```

## Unit State Machine

```mermaid
stateDiagram-v2
    [*] --> received: Unit created at receiving

    received --> reserved: Reserve for order
    note right of reserved: OrderID assigned

    reserved --> staged: Hard allocation
    note right of staged: AllocationID assigned

    staged --> picked: Picker confirms
    reserved --> picked: Direct pick (single route)
    note right of picked: ToteID assigned

    picked --> consolidated: Multi-item order
    note right of consolidated: ConsolidationBin assigned

    picked --> packed: Single-item order
    consolidated --> packed: From consolidation
    note right of packed: PackageID assigned

    packed --> shipped: Carrier handoff
    note right of shipped: TrackingNumber assigned

    received --> exception: Exception reported
    reserved --> exception: Exception reported
    staged --> exception: Exception reported
    picked --> exception: Exception reported
```

## Multi-Route Consolidation

```mermaid
sequenceDiagram
    autonumber
    participant O as Orchestrator
    participant R1 as Route 1 (Zone A)
    participant R2 as Route 2 (Zone B)
    participant US as Unit Service
    participant CS as Consolidation Service

    Note over O,CS: Order with items in multiple zones

    par Pick Route 1
        R1->>US: Pick units (Zone A)
        US-->>R1: Units in Tote-1
    and Pick Route 2
        R2->>US: Pick units (Zone B)
        US-->>R2: Units in Tote-2
    end

    O->>CS: Start consolidation
    CS->>US: Get units by order
    US-->>CS: Units from Tote-1, Tote-2

    loop For each unit
        CS->>US: Consolidate unit to bin
        US->>US: Update sourceToteId
        US->>US: Status: picked → consolidated
    end

    CS-->>O: All units consolidated
```

## Movement Audit Trail

Each status transition creates a movement record:

```mermaid
classDiagram
    class UnitMovement {
        +MovementID string
        +FromLocationID string
        +ToLocationID string
        +FromStatus UnitStatus
        +ToStatus UnitStatus
        +StationID string
        +HandlerID string
        +Timestamp time.Time
        +Notes string
    }
```

### Example Audit Trail

```json
{
  "unitId": "unit-abc-123",
  "movements": [
    {
      "movementId": "mov-001",
      "fromLocationId": "",
      "toLocationId": "RECV-DOCK-01",
      "fromStatus": "",
      "toStatus": "received",
      "handlerId": "WORKER-001",
      "timestamp": "2024-01-15T08:00:00Z",
      "notes": "Unit created at receiving"
    },
    {
      "movementId": "mov-002",
      "fromStatus": "received",
      "toStatus": "reserved",
      "handlerId": "SYSTEM",
      "timestamp": "2024-01-15T08:30:00Z",
      "notes": "Reserved for order ORD-12345"
    },
    {
      "movementId": "mov-003",
      "fromLocationId": "RECV-DOCK-01",
      "toLocationId": "A-01-02",
      "fromStatus": "reserved",
      "toStatus": "picked",
      "handlerId": "PICKER-001",
      "timestamp": "2024-01-15T09:15:00Z",
      "notes": "Picked to TOTE-001"
    }
  ]
}
```

## Exception Handling

```mermaid
sequenceDiagram
    participant W as Worker
    participant US as Unit Service
    participant O as Orchestrator

    W->>US: POST /units/{id}/exception
    Note over US: Exception details:<br/>type, stage, description
    US->>US: Status → exception
    US->>US: Record exception movement
    US-->>W: Exception ID

    Note over US,O: Unit blocked until resolved

    O->>US: GET /exceptions/unresolved
    US-->>O: List of exceptions

    O->>US: POST /exceptions/{id}/resolve
    US->>US: Apply resolution
    US-->>O: Resolution confirmed
```

## API Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/units` | Create units at receiving |
| POST | `/units/reserve` | Reserve units for order |
| GET | `/units/{id}` | Get unit details |
| GET | `/units/order/{orderId}` | Get units for order |
| GET | `/units/{id}/audit` | Get unit movement history |
| POST | `/units/{id}/pick` | Confirm pick |
| POST | `/units/{id}/consolidate` | Confirm consolidation |
| POST | `/units/{id}/pack` | Confirm pack |
| POST | `/units/{id}/ship` | Confirm ship |
| POST | `/units/{id}/exception` | Report exception |
| POST | `/exceptions/{id}/resolve` | Resolve exception |

## Related Documentation

- [Unit Service](/services/unit-service) - Service documentation
- [Unit Aggregate](/domain-driven-design/aggregates/unit) - Domain model
- [Unit Activities](/temporal/activities/unit-activities) - Temporal activities
- [Order Fulfillment Workflow](/temporal/workflows/order-fulfillment) - Parent workflow
