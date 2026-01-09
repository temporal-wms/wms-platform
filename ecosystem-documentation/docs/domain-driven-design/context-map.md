---
sidebar_position: 3
---

# Context Map

This document shows the relationships between bounded contexts in the WMS Platform.

## Context Map Diagram

```mermaid
graph TB
    subgraph "Upstream Contexts"
        O[Order Context]
        I[Inventory Context]
        L[Labor Context]
    end

    subgraph "Core Contexts"
        W[Waving Context]
        R[Routing Context]
        P[Picking Context]
    end

    subgraph "Downstream Contexts"
        C[Consolidation Context]
        PK[Packing Context]
        S[Shipping Context]
    end

    O -->|Conformist| W
    I -->|OHS| P
    I -->|OHS| C
    L -->|OHS| P
    L -->|OHS| PK

    W -->|PL| P
    W -->|PL| R
    R -->|SK| P

    P -->|U/D| C
    C -->|U/D| PK
    PK -->|U/D| S

    style O fill:#e3f2fd
    style I fill:#e3f2fd
    style L fill:#e3f2fd
    style W fill:#ffcdd2
    style R fill:#ffcdd2
    style P fill:#ffcdd2
    style C fill:#fff9c4
    style PK fill:#fff9c4
    style S fill:#fff9c4
```

## Relationship Types

| Pattern | Abbreviation | Description |
|---------|--------------|-------------|
| **Conformist** | CF | Downstream conforms to upstream model |
| **Customer-Supplier** | U/D | Upstream serves downstream needs |
| **Published Language** | PL | Shared language via events |
| **Shared Kernel** | SK | Shared code between contexts |
| **Open Host Service** | OHS | Public API for multiple consumers |
| **Anti-Corruption Layer** | ACL | Translation layer for external systems |

## Detailed Relationships

### Order → Waving (Conformist)

```mermaid
graph LR
    subgraph "Order Context"
        Order[Order Aggregate]
    end

    subgraph "Waving Context"
        Wave[Wave Aggregate]
    end

    Order -->|"Conformist"| Wave
```

The Waving context accepts the Order model as-is:
- **Order ID** - Used as reference
- **Priority** - Used for wave scheduling
- **Items** - Used for pick planning

### Waving → Picking (Published Language)

```mermaid
sequenceDiagram
    participant Waving
    participant Kafka
    participant Picking

    Waving->>Kafka: WaveReleasedEvent
    Note over Kafka: CloudEvents 1.0
    Kafka->>Picking: Subscribe

    Note over Picking: Creates PickTask<br/>based on event data
```

The shared language is CloudEvents:
- `wms.wave.released` event type
- Includes wave ID, order IDs, item details
- Self-contained for processing

### Routing ↔ Picking (Shared Kernel)

```mermaid
classDiagram
    class Location {
        <<Shared Kernel>>
        +Zone string
        +Aisle string
        +Rack string
        +Level string
        +Position string
        +X int
        +Y int
        +DistanceFrom(Location) float64
    }

    Routing --> Location
    Picking --> Location
```

The `Location` value object is shared:
- Same code in both contexts
- Coordinated changes required
- Minimized shared surface

### Inventory → Picking (Open Host Service)

```mermaid
sequenceDiagram
    participant Picking
    participant Inventory

    Picking->>Inventory: GET /api/v1/inventory/{sku}
    Inventory-->>Picking: { location, quantity }

    Picking->>Inventory: POST /api/v1/inventory/pick
    Inventory-->>Picking: { success: true }
```

Open Host Service characteristics:
- Well-documented REST API
- Multiple consumers supported
- Version-controlled endpoints

### Picking → Consolidation (Customer-Supplier)

```mermaid
graph LR
    subgraph "Upstream: Picking"
        PickTask[PickTask Aggregate]
        PickEvent[PickTaskCompletedEvent]
    end

    subgraph "Downstream: Consolidation"
        ConUnit[ConsolidationUnit]
    end

    PickTask --> PickEvent
    PickEvent -->|"Drives"| ConUnit
```

Customer-Supplier relationship:
- Picking (upstream) serves Consolidation (downstream)
- Downstream needs influence upstream API design
- Clear ownership and responsibility

### Shipping → Carriers (Anti-Corruption Layer)

```mermaid
graph LR
    subgraph "Shipping Context"
        Domain[Shipment Aggregate]
        ACL[Anti-Corruption Layer]
    end

    subgraph "External Systems"
        UPS[UPS API]
        FedEx[FedEx API]
        USPS[USPS API]
    end

    Domain --> ACL
    ACL -->|"Translate"| UPS
    ACL -->|"Translate"| FedEx
    ACL -->|"Translate"| USPS
```

ACL responsibilities:
- Translate domain models to carrier formats
- Normalize carrier responses
- Isolate domain from external changes

## Integration Patterns

### Event-Based Integration

```mermaid
graph TB
    subgraph "Publishers"
        Order[Order Context]
        Waving[Waving Context]
        Picking[Picking Context]
    end

    subgraph "Kafka Topics"
        T1[wms.orders.events]
        T2[wms.waves.events]
        T3[wms.picking.events]
    end

    subgraph "Consumers"
        C1[Waving Context]
        C2[Picking Context]
        C3[Inventory Context]
    end

    Order --> T1
    Waving --> T2
    Picking --> T3

    T1 --> C1
    T2 --> C2
    T3 --> C3
```

### API-Based Integration

```mermaid
graph LR
    subgraph "Orchestrator"
        Orch[Workflow Activities]
    end

    subgraph "Services"
        Order[Order API]
        Inventory[Inventory API]
        Picking[Picking API]
    end

    Orch -->|"REST"| Order
    Orch -->|"REST"| Inventory
    Orch -->|"REST"| Picking
```

## Team Topology

| Context | Team | Communication |
|---------|------|---------------|
| Order | Order Team | Event + API |
| Inventory | Inventory Team | API (OHS) |
| Waving | Fulfillment Team | Event (PL) |
| Routing | Fulfillment Team | Shared Kernel |
| Picking | Fulfillment Team | Event + API |
| Consolidation | Fulfillment Team | Event |
| Packing | Shipping Team | Event |
| Shipping | Shipping Team | Event + ACL |
| Labor | Operations Team | API (OHS) |

## Evolution Patterns

### Adding New Consumer

When a new context needs data from an existing context:

1. **Prefer Events** - Subscribe to existing topics
2. **Request API** - If real-time data needed
3. **Avoid Shared DB** - Never share tables

### Splitting a Context

When a context becomes too large:

1. **Identify Seams** - Find natural boundaries
2. **Define New Context** - New aggregate, new events
3. **Maintain Events** - Publish bridge events during transition

## Related Documentation

- [Bounded Contexts](./bounded-contexts) - Context descriptions
- [Domain Events](./domain-events) - Event catalog
- [Overview](./overview) - DDD overview
