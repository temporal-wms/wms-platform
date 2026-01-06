---
sidebar_position: 12
---

# Receiving Workflow

This document describes the inbound receiving flow for processing incoming shipments, from dock arrival through putaway to storage locations.

## Overview

The receiving workflow handles the processing of inbound shipments from suppliers, including check-in, item verification, quality control, and putaway to storage locations using chaotic storage strategies.

## Complete Receiving Flow

```mermaid
sequenceDiagram
    autonumber
    participant D as Dock Worker
    participant RS as Receiving Service
    participant US as Unit Service
    participant IS as Inventory Service
    participant SA as Stow Activities
    participant K as Kafka

    rect rgb(240, 248, 255)
        Note over D,RS: Phase 1: Shipment Arrival
        D->>RS: Shipment arrived at dock
        RS->>RS: Create shipment record
        RS->>K: Publish ShipmentArrivedEvent
        RS-->>D: Shipment ID, dock assignment
    end

    rect rgb(255, 248, 240)
        Note over D,RS: Phase 2: Check-In
        D->>RS: Scan ASN / BOL
        RS->>RS: Validate against PO
        RS->>RS: Record check-in time
        RS->>K: Publish ShipmentCheckedInEvent
        RS-->>D: Check-in confirmed
    end

    rect rgb(240, 255, 240)
        Note over D,US: Phase 3: Item Receiving
        loop For each item/pallet
            D->>RS: Scan item barcode
            RS->>RS: Validate SKU, quantity
            RS->>US: Create unit records
            US-->>RS: Unit IDs created
            RS->>K: Publish ItemReceivedEvent

            alt Discrepancy found
                D->>RS: Report discrepancy
                RS->>K: Publish DiscrepancyReportedEvent
            end
        end
    end

    rect rgb(255, 240, 255)
        Note over RS,SA: Phase 4: Putaway
        RS->>K: Publish PutawayStartedEvent

        loop For each stow task
            SA->>SA: FindStorageLocation
            Note over SA: Chaotic storage algorithm
            SA->>SA: AssignLocation
            SA->>SA: ExecuteStow
            SA->>IS: UpdateInventoryLocation
        end

        RS->>K: Publish PutawayCompletedEvent
    end

    rect rgb(248, 255, 240)
        Note over RS,IS: Phase 5: Inventory Update
        RS->>RS: Mark shipment complete
        RS->>K: Publish ShipmentCompletedEvent
        IS->>IS: Update available inventory
    end
```

## Receiving State Machine

```mermaid
stateDiagram-v2
    [*] --> Arrived: Shipment at dock
    Arrived --> CheckedIn: ASN/BOL scanned
    CheckedIn --> Receiving: Begin item scan

    Receiving --> Receiving: Item received
    Receiving --> Discrepancy: Problem found
    Discrepancy --> Receiving: Resolved

    Receiving --> Putaway: All items received
    Putaway --> Stowing: Find locations
    Stowing --> Stowing: Item stowed
    Stowing --> Completed: All items stowed

    Completed --> [*]
```

## Chaotic Storage Algorithm

```mermaid
flowchart TD
    A[New Item to Stow] --> B{Item Type?}

    B -->|Hazmat| C[HAZMAT Zone]
    B -->|Cold Chain| D[COLD Zone]
    B -->|Oversized| E[OVERSIZE Zone]
    B -->|Standard| F[GENERAL Zone]

    C --> G[Find Available Bin]
    D --> G
    E --> G
    F --> G

    G --> H{Strategy}
    H -->|Chaotic| I[Random Aisle/Rack/Bin]
    H -->|Directed| J[Predefined Location]
    H -->|Velocity| K[Based on SKU velocity]

    I --> L[Assign Location]
    J --> L
    K --> L

    L --> M[Execute Stow]
    M --> N[Update Inventory]
```

## Storage Location Format

```
ZONE-AISLE-RACK-LEVEL-BIN

Example: GENERAL-A-05-3-B02
         ^^^^^^^ ^ ^^ ^ ^^^
         Zone    | |  | Bin
                 | |  Level
                 | Rack
                 Aisle
```

## Discrepancy Types

| Type | Description | Resolution |
|------|-------------|------------|
| `quantity_over` | More items than expected | Verify count, update PO |
| `quantity_short` | Fewer items than expected | Create discrepancy report |
| `damaged` | Items damaged in transit | Quarantine, create claim |
| `wrong_item` | Different SKU received | Return or reclassify |
| `quality_issue` | Quality below standard | QC review, quarantine |

## Receiving Events

```mermaid
graph LR
    A[ShipmentArrived] --> B[ShipmentCheckedIn]
    B --> C[ItemReceived]
    C --> C
    C -.->|problem| D[DiscrepancyReported]
    C --> E[ShipmentCompleted]
    E --> F[PutawayStarted]
    F --> G[PutawayCompleted]
```

### Event Payloads

#### ShipmentArrivedEvent

```json
{
  "type": "wms.receiving.shipment-arrived",
  "data": {
    "shipmentId": "SHIP-IN-001",
    "carrierId": "UPS",
    "dockId": "DOCK-01",
    "expectedItems": 50,
    "purchaseOrderIds": ["PO-001", "PO-002"],
    "arrivedAt": "2024-01-15T08:00:00Z"
  }
}
```

#### ItemReceivedEvent

```json
{
  "type": "wms.receiving.item-received",
  "data": {
    "shipmentId": "SHIP-IN-001",
    "sku": "SKU-001",
    "quantity": 100,
    "lotNumber": "LOT-2024-001",
    "expirationDate": "2025-01-15",
    "receivingLocationId": "RECV-DOCK-01",
    "receivedBy": "WORKER-001",
    "receivedAt": "2024-01-15T08:30:00Z"
  }
}
```

## Stow Activities

| Activity | Purpose |
|----------|---------|
| `FindStorageLocation` | Find available bin using storage strategy |
| `AssignLocation` | Assign location to stow task |
| `ExecuteStow` | Move item from tote to storage |
| `UpdateInventoryLocation` | Update inventory with new location |

## API Endpoints

### Receiving Service

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/shipments` | Register incoming shipment |
| PUT | `/shipments/{id}/check-in` | Check in shipment |
| POST | `/shipments/{id}/items` | Receive item |
| POST | `/shipments/{id}/discrepancy` | Report discrepancy |
| PUT | `/shipments/{id}/complete` | Mark shipment complete |

### Stow Service

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/stow/tasks` | Create stow task |
| GET | `/stow/tasks/{id}/location` | Get assigned location |
| PUT | `/stow/tasks/{id}/complete` | Complete stow task |

## Related Documentation

- [Receiving Service](/services/receiving-service) - Service documentation
- [Stow Activities](/temporal/activities/stow-activities) - Temporal activities
- [Receiving Events](/domain-driven-design/domain-events#receiving-events) - Domain events
- [Inbound Fulfillment Workflow](/temporal/workflows/inbound-fulfillment) - Parent workflow
