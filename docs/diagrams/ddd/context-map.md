# WMS Platform - DDD Context Map

This document shows the bounded contexts and their relationships following Domain-Driven Design strategic patterns.

## Context Map Overview

```mermaid
graph TB
    subgraph "Strategic Design - Bounded Contexts"

        subgraph "Core Domain"
            Picking[Picking Context]
            Routing[Routing Context]
            Waving[Waving Context]
        end

        subgraph "Supporting Domain"
            Inventory[Inventory Context]
            Labor[Labor Context]
            Consolidation[Consolidation Context]
        end

        subgraph "Generic Domain"
            Order[Order Context]
            Packing[Packing Context]
            Shipping[Shipping Context]
        end

    end

    Order -->|Conformist| Waving
    Waving -->|Published Language| Picking
    Waving -->|Published Language| Routing
    Routing -->|Shared Kernel| Picking
    Inventory -->|Open Host Service| Picking
    Picking -->|Customer-Supplier| Consolidation
    Consolidation -->|Customer-Supplier| Packing
    Packing -->|Customer-Supplier| Shipping
    Labor -->|Open Host Service| Picking
    Labor -->|Open Host Service| Packing

    style Picking fill:#ff9999
    style Routing fill:#ff9999
    style Waving fill:#ff9999
```

## Detailed Relationship Map

```mermaid
graph LR
    subgraph "Upstream Contexts"
        O[Order]
        I[Inventory]
        L[Labor]
    end

    subgraph "Core Contexts"
        W[Waving]
        R[Routing]
        P[Picking]
    end

    subgraph "Downstream Contexts"
        C[Consolidation]
        PK[Packing]
        S[Shipping]
    end

    O -->|U/D| W
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
```

## Context Relationships

### Relationship Types

| Pattern | Description | Usage |
|---------|-------------|-------|
| **Conformist (CF)** | Downstream conforms to upstream model | Order → Waving |
| **Customer-Supplier (U/D)** | Upstream serves downstream needs | Picking → Consolidation |
| **Published Language (PL)** | Shared language via events | Waving → Picking (CloudEvents) |
| **Shared Kernel (SK)** | Shared code between contexts | Routing ↔ Picking (Location) |
| **Open Host Service (OHS)** | Public API for multiple consumers | Inventory, Labor services |
| **Anti-Corruption Layer (ACL)** | Translation layer | External carrier integration |

### Context Relationships Detail

```mermaid
graph TD
    subgraph "Order → Waving"
        O1[Order Context] -->|"Conformist"| W1[Waving Context]
        note1["Waving accepts Order model as-is"]
    end

    subgraph "Waving → Picking"
        W2[Waving Context] -->|"Published Language"| P1[Picking Context]
        note2["WaveReleasedEvent via Kafka"]
    end

    subgraph "Routing ↔ Picking"
        R1[Routing Context] <-->|"Shared Kernel"| P2[Picking Context]
        note3["Location value object shared"]
    end

    subgraph "Inventory → All"
        I1[Inventory Context] -->|"Open Host Service"| Multi["Picking, Consolidation"]
        note4["REST API for stock operations"]
    end
```

## Bounded Context Descriptions

### Core Domain Contexts

| Context | Responsibility | Aggregate Root |
|---------|---------------|----------------|
| **Picking** | Warehouse picking operations | PickTask |
| **Routing** | Pick path optimization | PickRoute |
| **Waving** | Batch order grouping | Wave |

### Supporting Domain Contexts

| Context | Responsibility | Aggregate Root |
|---------|---------------|----------------|
| **Inventory** | Stock management | InventoryItem |
| **Labor** | Workforce management | Worker |
| **Consolidation** | Multi-item combining | ConsolidationUnit |

### Generic Domain Contexts

| Context | Responsibility | Aggregate Root |
|---------|---------------|----------------|
| **Order** | Order lifecycle | Order |
| **Packing** | Package preparation | PackTask |
| **Shipping** | Carrier integration | Shipment |

## Integration Patterns

### Event-Based Integration

```mermaid
sequenceDiagram
    participant Order
    participant Kafka
    participant Waving
    participant Picking

    Order->>Kafka: OrderReceivedEvent
    Kafka->>Waving: Subscribe
    Waving->>Waving: Add to Wave
    Waving->>Kafka: WaveReleasedEvent
    Kafka->>Picking: Subscribe
    Picking->>Picking: Create PickTask
```

### API-Based Integration

```mermaid
sequenceDiagram
    participant Orchestrator
    participant Inventory
    participant Picking

    Orchestrator->>Inventory: Reserve Stock (OHS)
    Inventory-->>Orchestrator: Reservation Confirmed
    Orchestrator->>Picking: Create Pick Task
    Picking->>Inventory: Confirm Pick (OHS)
    Inventory-->>Picking: Stock Updated
```

## Shared Kernel: Location

The `Location` value object is shared between Routing and Picking contexts:

```mermaid
classDiagram
    class Location {
        <<Shared Kernel>>
        +LocationID string
        +Zone string
        +Aisle string
        +Rack string
        +Level string
        +Position string
        +X int
        +Y int
        +DistanceFrom(Location) float64
        +IsSameZone(Location) bool
        +IsSameAisle(Location) bool
    }

    Routing --> Location : uses
    Picking --> Location : uses
```

## Anti-Corruption Layer

```mermaid
graph LR
    subgraph "Shipping Context"
        Ship[Shipment Aggregate]
        ACL[Anti-Corruption Layer]
    end

    subgraph "External Carriers"
        UPS[UPS API]
        FedEx[FedEx API]
        USPS[USPS API]
    end

    Ship --> ACL
    ACL --> UPS
    ACL --> FedEx
    ACL --> USPS
```

## Team Ownership

| Context | Team | Deployment |
|---------|------|------------|
| Order | Order Team | order-service |
| Inventory | Inventory Team | inventory-service |
| Waving | Fulfillment Team | waving-service |
| Routing | Fulfillment Team | routing-service |
| Picking | Fulfillment Team | picking-service |
| Consolidation | Fulfillment Team | consolidation-service |
| Packing | Shipping Team | packing-service |
| Shipping | Shipping Team | shipping-service |
| Labor | Operations Team | labor-service |

## Related Documentation

- [Ecosystem](../ecosystem.md) - Platform architecture
- [Domain Events](domain-events.md) - Event flows
- [Order Fulfillment Flow](../order-fulfillment-flow.md) - End-to-end workflow
