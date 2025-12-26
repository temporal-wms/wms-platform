# Inventory Service - Class Diagram

This diagram shows the domain model for the Inventory Service bounded context.

## Domain Model

```mermaid
classDiagram
    class InventoryItem {
        <<Aggregate Root>>
        +SKU string
        +ProductName string
        +Description string
        +Category string
        +TotalQuantity int
        +AvailableQuantity int
        +ReservedQuantity int
        +ReorderPoint int
        +ReorderQuantity int
        +Locations []StockLocation
        +Reservations []Reservation
        +ReceiveStock(locationID string, qty int)
        +Reserve(orderID string, qty int) error
        +Pick(orderID string, qty int) error
        +ReleaseReservation(orderID string)
        +Adjust(qty int, reason string)
        +Transfer(fromLoc, toLoc string, qty int)
        +IsLowStock() bool
    }

    class StockLocation {
        <<Entity>>
        +LocationID string
        +Zone string
        +Aisle string
        +Rack string
        +Level string
        +Position string
        +Quantity int
        +LastCountedAt time.Time
        +GetCoordinates() (x, y int)
    }

    class Reservation {
        <<Entity>>
        +ReservationID string
        +OrderID string
        +Quantity int
        +Status ReservationStatus
        +CreatedAt time.Time
        +ExpiresAt time.Time
        +IsExpired() bool
    }

    class InventoryTransaction {
        <<Entity>>
        +TransactionID string
        +SKU string
        +Type TransactionType
        +Quantity int
        +LocationID string
        +OrderID string
        +Reason string
        +Timestamp time.Time
    }

    class ReservationStatus {
        <<Enumeration>>
        active
        fulfilled
        expired
        released
    }

    class TransactionType {
        <<Enumeration>>
        receive
        pick
        adjust
        transfer
        reserve
        release
    }

    InventoryItem "1" *-- "*" StockLocation : stored at
    InventoryItem "1" *-- "*" Reservation : has
    InventoryItem "1" *-- "*" InventoryTransaction : logs
    Reservation --> ReservationStatus : has
    InventoryTransaction --> TransactionType : has type
```

## Location Hierarchy

```mermaid
graph TD
    subgraph "Warehouse Location Structure"
        Warehouse[Warehouse]
        Warehouse --> ZoneA[Zone A]
        Warehouse --> ZoneB[Zone B]
        ZoneA --> Aisle01[Aisle 01]
        ZoneA --> Aisle02[Aisle 02]
        Aisle01 --> Rack01[Rack 01]
        Aisle01 --> Rack02[Rack 02]
        Rack01 --> Level1[Level 1]
        Rack01 --> Level2[Level 2]
        Level1 --> Pos1[Position 1]
        Level1 --> Pos2[Position 2]
    end
```

## Repository Interface

```mermaid
classDiagram
    class InventoryRepository {
        <<Interface>>
        +Create(item InventoryItem) error
        +GetBySKU(sku string) InventoryItem
        +Update(item InventoryItem) error
        +FindByLocation(locationID string) []InventoryItem
        +FindByZone(zone string) []InventoryItem
        +FindLowStock() []InventoryItem
        +GetTransactions(sku string) []InventoryTransaction
    }
```

## Related Diagrams

- [Aggregate Diagram](ddd/aggregates.md) - DDD aggregate structure
- [Order Fulfillment Flow](../../../docs/diagrams/order-fulfillment-flow.md) - Workflow integration
