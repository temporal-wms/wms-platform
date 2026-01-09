# Order Service

The Order Service manages the complete lifecycle of customer orders within the WMS Platform.

## Overview

- **Port**: 8001
- **Database**: orders_db (MongoDB)
- **Aggregate Root**: Order

## Features

- Order creation and validation
- Order status management
- Wave assignment tracking
- Customer order history
- Temporal workflow integration

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/orders` | Create a new order |
| GET | `/api/v1/orders/:orderId` | Get order by ID |
| PUT | `/api/v1/orders/:orderId/validate` | Validate order |
| PUT | `/api/v1/orders/:orderId/cancel` | Cancel order |
| GET | `/api/v1/orders` | List orders (paginated) |
| GET | `/api/v1/orders/status/:status` | List orders by status |
| GET | `/api/v1/orders/customer/:customerId` | List customer orders |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `OrderReceived` | wms.orders.events | New order received |
| `OrderValidated` | wms.orders.events | Order passed validation |
| `OrderCancelled` | wms.orders.events | Order cancelled |
| `OrderAssignedToWave` | wms.orders.events | Order added to wave |
| `OrderShipped` | wms.orders.events | Order shipped |
| `OrderCompleted` | wms.orders.events | Order delivered |

## Domain Model

```go
type Order struct {
    ID              string
    CustomerID      string
    Status          OrderStatus
    Priority        Priority
    Items           []OrderItem
    ShippingAddress Address
    WaveID          *string
    TotalValue      float64
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type OrderStatus string
const (
    OrderStatusPending     OrderStatus = "pending"
    OrderStatusValidated   OrderStatus = "validated"
    OrderStatusInWave      OrderStatus = "in_wave"
    OrderStatusPicking     OrderStatus = "picking"
    OrderStatusPacking     OrderStatus = "packing"
    OrderStatusShipped     OrderStatus = "shipped"
    OrderStatusCompleted   OrderStatus = "completed"
    OrderStatusCancelled   OrderStatus = "cancelled"
)
```

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-order-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8001` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `orders_db` |
| `KAFKA_BROKERS` | Kafka brokers | `localhost:9092` |
| `TEMPORAL_HOST` | Temporal server | `localhost:7233` |

## Testing

```bash
# Unit tests
go test ./internal/domain/...

# Integration tests
go test ./tests/integration/...

# Contract tests
go test ./tests/contracts/...
```

## Related Services

- **waving-service**: Receives orders for wave assignment
- **orchestrator**: Manages order fulfillment workflow
- **inventory-service**: Reserves stock for orders
