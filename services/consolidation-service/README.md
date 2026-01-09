# Consolidation Service

The Consolidation Service manages the combining of picked items for multi-item orders.

## Overview

- **Port**: 8005
- **Database**: consolidation_db (MongoDB)
- **Aggregate Root**: ConsolidationUnit

## Features

- Consolidation unit management
- Multi-item order combining
- Item scanning and verification
- Order completeness validation
- Station assignment

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/consolidations` | Create consolidation unit |
| GET | `/api/v1/consolidations/:id` | Get consolidation unit |
| POST | `/api/v1/consolidations/:id/consolidate` | Scan/consolidate item |
| POST | `/api/v1/consolidations/:id/complete` | Complete consolidation |
| GET | `/api/v1/consolidations/order/:orderId` | Get by order ID |
| GET | `/api/v1/consolidations/station/:station` | Get by station |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `ConsolidationStarted` | wms.consolidation.events | Consolidation started |
| `ItemConsolidated` | wms.consolidation.events | Item scanned/added |
| `ConsolidationCompleted` | wms.consolidation.events | All items consolidated |

## Domain Model

```go
type ConsolidationUnit struct {
    ID            string
    OrderID       string
    Station       string
    ExpectedItems []ExpectedItem
    ScannedItems  []ScannedItem
    Status        ConsolidationStatus
    StartedAt     *time.Time
    CompletedAt   *time.Time
    CreatedAt     time.Time
}

type ExpectedItem struct {
    SKU      string
    Quantity int
}

type ScannedItem struct {
    SKU       string
    Quantity  int
    ScannedAt time.Time
}

type ConsolidationStatus string
const (
    ConsolidationStatusPending    ConsolidationStatus = "pending"
    ConsolidationStatusInProgress ConsolidationStatus = "in_progress"
    ConsolidationStatusCompleted  ConsolidationStatus = "completed"
)
```

## Running Locally

```bash
go run cmd/api/main.go
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8005` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `consolidation_db` |
| `KAFKA_BROKERS` | Kafka brokers | `localhost:9092` |

## Testing

```bash
go test ./...
```

## Related Services

- **picking-service**: Provides picked items
- **packing-service**: Receives consolidated orders
