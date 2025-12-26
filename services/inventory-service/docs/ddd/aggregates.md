# Inventory Service - DDD Aggregates

This document describes the aggregate structure for the Inventory bounded context.

## Aggregate: InventoryItem

The InventoryItem aggregate manages stock levels, locations, and reservations for a single SKU.

```mermaid
graph TD
    subgraph "InventoryItem Aggregate"
        Item[InventoryItem<br/><<Aggregate Root>>]

        subgraph "Entities"
            Location[StockLocation]
            Reservation[Reservation]
            Transaction[InventoryTransaction]
        end

        subgraph "Value Objects"
            ReservationStatus[ReservationStatus]
            TransactionType[TransactionType]
        end

        Item -->|stored at| Location
        Item -->|has| Reservation
        Item -->|logs| Transaction
        Reservation -->|has| ReservationStatus
        Transaction -->|type| TransactionType
    end

    style Item fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "InventoryItem Aggregate Boundary"
        I[InventoryItem]
        SL[StockLocation]
        R[Reservation]
        T[Transaction]
    end

    subgraph "External References"
        O[OrderID]
        SKU[SKU]
    end

    I -.->|identified by| SKU
    R -.->|for| O

    style I fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Available >= 0 | Available quantity cannot be negative |
| Reserved <= Total | Reserved cannot exceed total quantity |
| Single reservation per order | One reservation per order per SKU |
| Expiration enforced | Reservations expire after 24 hours |
| Location quantity >= 0 | Location quantity must be non-negative |

## Domain Events

```mermaid
graph LR
    Item[InventoryItem] -->|emits| E1[InventoryReceivedEvent]
    Item -->|emits| E2[InventoryReservedEvent]
    Item -->|emits| E3[InventoryPickedEvent]
    Item -->|emits| E4[InventoryAdjustedEvent]
    Item -->|emits| E5[LowStockAlertEvent]
    Item -->|emits| E6[ReservationReleasedEvent]
```

## Stock Calculations

```mermaid
flowchart TD
    Total[Total Quantity] --> Available
    Reserved[Reserved Qty] --> Available
    Available[Available = Total - Reserved]

    subgraph "Operations"
        Receive[Receive: Total += qty]
        Reserve[Reserve: Reserved += qty]
        Pick[Pick: Total -= qty, Reserved -= qty]
        Release[Release: Reserved -= qty]
        Adjust[Adjust: Total = newQty]
    end
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [Context Map](../../../../docs/diagrams/ddd/context-map.md) - Bounded context relationships
