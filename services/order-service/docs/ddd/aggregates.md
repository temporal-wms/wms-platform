# Order Service - DDD Aggregates

This document describes the aggregate structure for the Order bounded context following Domain-Driven Design principles.

## Aggregate: Order

The Order aggregate is the core domain entity that represents a customer order throughout its fulfillment lifecycle.

```mermaid
graph TD
    subgraph "Order Aggregate"
        Order[Order<br/><<Aggregate Root>>]

        subgraph "Entities"
            OrderItem[OrderItem]
        end

        subgraph "Value Objects"
            Address[Address]
            Dimensions[Dimensions]
            Priority[Priority]
            Status[OrderStatus]
        end

        Order -->|contains| OrderItem
        Order -->|ships to| Address
        Order -->|has| Priority
        Order -->|has| Status
        OrderItem -->|has| Dimensions
    end

    style Order fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "Order Aggregate Boundary"
        O[Order]
        OI[OrderItem]
        A[Address]
    end

    subgraph "External References"
        C[CustomerID]
        W[WaveID]
        T[TrackingNumber]
    end

    O -.->|references| C
    O -.->|assigned to| W
    O -.->|tracks via| T

    style O fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Order must have items | An order cannot exist without at least one item |
| Valid status transitions | Status can only change according to state machine |
| Priority immutable | Priority cannot change after validation |
| Quantity positive | OrderItem quantity must be > 0 |
| Address required | Shipping address must be valid for shipping |

## Domain Events

```mermaid
graph LR
    Order -->|emits| E1[OrderReceivedEvent]
    Order -->|emits| E2[OrderValidatedEvent]
    Order -->|emits| E3[OrderAssignedToWaveEvent]
    Order -->|emits| E4[OrderShippedEvent]
    Order -->|emits| E5[OrderCancelledEvent]
    Order -->|emits| E6[OrderCompletedEvent]
```

## Factory Pattern

```mermaid
classDiagram
    class OrderFactory {
        +CreateOrder(customerID, items, address, priority) Order
        +ReconstituteFromEvents(events) Order
    }

    class Order {
        -constructor()
        +static Create() Order
    }

    OrderFactory --> Order : creates
```

## Repository Pattern

```mermaid
classDiagram
    class OrderRepository {
        <<Interface>>
        +Save(order Order)
        +GetByID(id string) Order
        +FindByCustomer(customerID string) []Order
        +FindByStatus(status OrderStatus) []Order
    }

    class MongoOrderRepository {
        +Save(order Order)
        +GetByID(id string) Order
        +FindByCustomer(customerID string) []Order
        +FindByStatus(status OrderStatus) []Order
    }

    OrderRepository <|.. MongoOrderRepository
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [Context Map](../../../../docs/diagrams/ddd/context-map.md) - Bounded context relationships
