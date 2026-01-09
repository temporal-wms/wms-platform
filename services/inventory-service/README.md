# Inventory Service

The Inventory Service manages stock levels, reservations, and inventory movements.

## Overview

- **Port**: 8008
- **Database**: inventory_db (MongoDB)
- **Aggregate Root**: InventoryItem

## Features

- Stock level management
- Inventory reservations
- Stock receipt processing
- Inventory adjustments
- Low stock alerts
- Location tracking

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/inventory/receive` | Receive stock |
| POST | `/api/v1/inventory/reserve` | Reserve inventory |
| POST | `/api/v1/inventory/release/:orderId` | Release reservation |
| GET | `/api/v1/inventory/sku/:sku` | Get by SKU |
| GET | `/api/v1/inventory/location/:location` | Get by location |
| POST | `/api/v1/inventory/adjust` | Adjust inventory |
| POST | `/api/v1/inventory/pick` | Confirm pick |
| GET | `/api/v1/inventory/low-stock` | Get low stock items |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `InventoryReceived` | wms.inventory.events | Stock received |
| `InventoryReserved` | wms.inventory.events | Stock reserved |
| `InventoryPicked` | wms.inventory.events | Stock picked |
| `InventoryAdjusted` | wms.inventory.events | Stock adjusted |
| `LowStockAlert` | wms.inventory.events | Low stock threshold |

## Domain Model

```go
type InventoryItem struct {
    ID           string
    SKU          string
    Location     Location
    AvailableQty int
    ReservedQty  int
    OnHandQty    int
    MinStock     int
    MaxStock     int
    LastReceived *time.Time
    LastPicked   *time.Time
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type Location struct {
    LocationID string
    Zone       string
    Aisle      string
    Rack       string
    Level      string
    Position   string
}

type Reservation struct {
    OrderID    string
    SKU        string
    Quantity   int
    ReservedAt time.Time
    ExpiresAt  time.Time
}
```

## Running Locally

```bash
go run cmd/api/main.go
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8008` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `inventory_db` |
| `KAFKA_BROKERS` | Kafka brokers | `localhost:9092` |

## Testing

```bash
go test ./...
```

## Related Services

- **order-service**: Reserves inventory for orders
- **picking-service**: Confirms stock picks
- **orchestrator**: Manages inventory in workflows
