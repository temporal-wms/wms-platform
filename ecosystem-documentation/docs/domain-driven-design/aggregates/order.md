---
sidebar_position: 1
---

# Order Aggregate

The Order aggregate is the root of the Order bounded context, managing the lifecycle of customer orders.

## Aggregate Structure

```mermaid
classDiagram
    class Order {
        <<Aggregate Root>>
        +ID string
        +CustomerID string
        +Status OrderStatus
        +Priority Priority
        +Items []OrderItem
        +ShippingAddress Address
        +WaveID *string
        +TotalAmount Money
        +CreatedAt time.Time
        +UpdatedAt time.Time
        +Validate() error
        +AssignToWave(waveID string)
        +Cancel(reason string)
        +MarkShipped(trackingNumber string)
        +Complete()
    }

    class OrderItem {
        <<Entity>>
        +ID string
        +SKU string
        +ProductName string
        +Quantity int
        +PickedQuantity int
        +Price Money
        +Weight Weight
        +Dimensions Dimensions
        +Pick(quantity int)
        +IsFullyPicked() bool
    }

    class Address {
        <<Value Object>>
        +Street string
        +City string
        +State string
        +ZipCode string
        +Country string
    }

    class Money {
        <<Value Object>>
        +Amount float64
        +Currency string
    }

    class OrderStatus {
        <<Enumeration>>
        RECEIVED
        VALIDATED
        WAVE_ASSIGNED
        PICKING
        CONSOLIDATED
        PACKED
        SHIPPED
        COMPLETED
        CANCELLED
    }

    Order "1" *-- "*" OrderItem : items
    Order "1" *-- "1" Address : shippingAddress
    Order "1" *-- "1" Money : totalAmount
    OrderItem "1" *-- "1" Money : price
    Order --> OrderStatus : status
```

## State Machine

```mermaid
stateDiagram-v2
    [*] --> Received: Order Created
    Received --> Validated: Validation Passed
    Received --> Cancelled: Validation Failed
    Validated --> WaveAssigned: Assigned to Wave
    Validated --> Cancelled: Timeout
    WaveAssigned --> Picking: Pick Started
    Picking --> Consolidated: Multi-item
    Picking --> Packed: Single item
    Picking --> Cancelled: Pick Failed
    Consolidated --> Packed: Consolidation Complete
    Packed --> Shipped: SLAM Complete
    Shipped --> Completed: Delivered
    Shipped --> [*]
    Cancelled --> [*]
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Valid Address | Shipping address must be complete and valid |
| Items Required | Order must have at least one item |
| Positive Quantities | All item quantities must be positive |
| Valid Status Transitions | Status can only transition to valid next states |
| Immutable After Ship | Order cannot be modified after shipping |

## Commands

### CreateOrder

```go
type CreateOrderCommand struct {
    CustomerID      string
    Priority        Priority
    Items           []OrderItemInput
    ShippingAddress Address
}

func (s *OrderService) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*Order, error) {
    order, err := NewOrder(cmd.CustomerID, cmd.Priority, cmd.Items, cmd.ShippingAddress)
    if err != nil {
        return nil, err
    }

    if err := s.repo.Save(ctx, order); err != nil {
        return nil, err
    }

    s.publisher.Publish(order.Events())
    return order, nil
}
```

### ValidateOrder

```go
func (o *Order) Validate() error {
    if o.Status != OrderStatusReceived {
        return ErrInvalidStatusTransition
    }

    // Validate address
    if err := o.ShippingAddress.Validate(); err != nil {
        return err
    }

    // Validate items
    for _, item := range o.Items {
        if item.Quantity <= 0 {
            return ErrInvalidQuantity
        }
    }

    o.Status = OrderStatusValidated
    o.addEvent(NewOrderValidatedEvent(o))
    return nil
}
```

### AssignToWave

```go
func (o *Order) AssignToWave(waveID string) error {
    if o.Status != OrderStatusValidated {
        return ErrInvalidStatusTransition
    }

    o.WaveID = &waveID
    o.Status = OrderStatusWaveAssigned
    o.addEvent(NewOrderWaveAssignedEvent(o, waveID))
    return nil
}
```

### CancelOrder

```go
func (o *Order) Cancel(reason string) error {
    if !o.CanCancel() {
        return ErrCannotCancel
    }

    o.Status = OrderStatusCancelled
    o.addEvent(NewOrderCancelledEvent(o, reason))
    return nil
}

func (o *Order) CanCancel() bool {
    nonCancellableStates := []OrderStatus{
        OrderStatusShipped,
        OrderStatusCompleted,
        OrderStatusCancelled,
    }
    return !contains(nonCancellableStates, o.Status)
}
```

## Domain Events

| Event | Trigger | Data |
|-------|---------|------|
| OrderReceivedEvent | Order created | Full order details |
| OrderValidatedEvent | Validation passed | Order ID, validated at |
| OrderWaveAssignedEvent | Assigned to wave | Order ID, wave ID |
| OrderShippedEvent | Shipped to carrier | Order ID, tracking number |
| OrderCancelledEvent | Order cancelled | Order ID, reason |
| OrderCompletedEvent | Delivery confirmed | Order ID, completed at |

## Repository Interface

```go
type OrderRepository interface {
    Save(ctx context.Context, order *Order) error
    FindByID(ctx context.Context, id string) (*Order, error)
    FindByCustomerID(ctx context.Context, customerID string) ([]*Order, error)
    FindByStatus(ctx context.Context, status OrderStatus) ([]*Order, error)
    FindPendingForWaving(ctx context.Context) ([]*Order, error)
    Update(ctx context.Context, order *Order) error
}
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/orders | Create order |
| GET | /api/v1/orders/\{id\} | Get order by ID |
| GET | /api/v1/orders | List orders |
| PUT | /api/v1/orders/\{id\}/validate | Validate order |
| PUT | /api/v1/orders/\{id\}/cancel | Cancel order |

## Related Documentation

- [Order Service](/services/order-service) - Service documentation
- [Domain Events](../domain-events) - Event catalog
- [Value Objects](../value-objects) - Address, Money types
