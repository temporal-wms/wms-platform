# Order Service - Class Diagram

This diagram shows the domain model for the Order Service bounded context.

## Domain Model

```mermaid
classDiagram
    class Order {
        <<Aggregate Root>>
        +OrderID string
        +CustomerID string
        +Status OrderStatus
        +Priority Priority
        +Items []OrderItem
        +ShippingAddress Address
        +PromisedDeliveryAt time.Time
        +WaveID string
        +TrackingNumber string
        +CreatedAt time.Time
        +UpdatedAt time.Time
        +Validate() error
        +AssignToWave(waveID string)
        +StartPicking()
        +MarkItemPicked(sku string, qty int)
        +MarkConsolidated()
        +MarkPacked(trackingNumber string)
        +MarkShipped()
        +Cancel(reason string)
    }

    class OrderItem {
        <<Entity>>
        +SKU string
        +ProductName string
        +Quantity int
        +Weight float64
        +Dimensions Dimensions
        +PickedQty int
        +UnitPrice float64
        +IsFullyPicked() bool
    }

    class Address {
        <<Value Object>>
        +Street string
        +City string
        +State string
        +PostalCode string
        +Country string
        +Validate() error
    }

    class Dimensions {
        <<Value Object>>
        +Length float64
        +Width float64
        +Height float64
        +Volume() float64
    }

    class OrderStatus {
        <<Enumeration>>
        received
        validated
        wave_assigned
        picking
        consolidated
        packed
        shipped
        delivered
        cancelled
    }

    class Priority {
        <<Enumeration>>
        same_day
        next_day
        standard
    }

    Order "1" *-- "*" OrderItem : contains
    Order "1" *-- "1" Address : ships to
    OrderItem "1" *-- "1" Dimensions : has
    Order --> OrderStatus : has status
    Order --> Priority : has priority
```

## State Transitions

```mermaid
stateDiagram-v2
    [*] --> received
    received --> validated: Validate()
    validated --> wave_assigned: AssignToWave()
    wave_assigned --> picking: StartPicking()
    picking --> consolidated: MarkConsolidated()
    picking --> packed: MarkPacked() [single item]
    consolidated --> packed: MarkPacked()
    packed --> shipped: MarkShipped()
    shipped --> delivered: ConfirmDelivery()

    received --> cancelled: Cancel()
    validated --> cancelled: Cancel()
    wave_assigned --> cancelled: Cancel()
    picking --> cancelled: Cancel()
    consolidated --> cancelled: Cancel()
    packed --> cancelled: Cancel()
```

## Repository Interface

```mermaid
classDiagram
    class OrderRepository {
        <<Interface>>
        +Create(order Order) error
        +GetByID(id string) Order
        +Update(order Order) error
        +Delete(id string) error
        +FindByCustomer(customerID string) []Order
        +FindByStatus(status OrderStatus) []Order
        +FindByWave(waveID string) []Order
    }
```

## Related Diagrams

- [Aggregate Diagram](ddd/aggregates.md) - DDD aggregate structure
- [Order Fulfillment Flow](../../../docs/diagrams/order-fulfillment-flow.md) - Workflow integration
