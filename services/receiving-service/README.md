# Receiving Service

The Receiving Service manages inbound shipments and the receiving process within the WMS Platform.

## Overview

- **Port**: 8013
- **Database**: receiving_db (MongoDB)
- **Aggregate Root**: InboundShipment

## Features

- Advance Shipping Notice (ASN) processing
- Shipment arrival tracking
- Item receiving with condition tracking (good/damaged)
- Discrepancy detection and reporting (shortage, overage, damage)
- Putaway task creation trigger
- Worker assignment for receiving operations

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/shipments` | List shipments (paginated) |
| POST | `/api/v1/shipments` | Create a new shipment |
| GET | `/api/v1/shipments/status/:status` | Get shipments by status |
| GET | `/api/v1/shipments/expected` | Get expected arrivals |
| GET | `/api/v1/shipments/:shipmentId` | Get shipment by ID |
| POST | `/api/v1/shipments/:shipmentId/arrive` | Mark shipment as arrived |
| POST | `/api/v1/shipments/:shipmentId/start` | Start receiving process |
| POST | `/api/v1/shipments/:shipmentId/receive` | Receive an item |
| POST | `/api/v1/shipments/:shipmentId/complete` | Complete receiving |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `receiving.shipment.expected` | wms.receiving.events | New shipment expected |
| `receiving.shipment.arrived` | wms.receiving.events | Shipment arrived at dock |
| `receiving.item.received` | wms.receiving.events | Item received |
| `receiving.completed` | wms.receiving.events | Receiving process completed |
| `receiving.discrepancy` | wms.receiving.events | Discrepancy detected |
| `receiving.putaway.created` | wms.receiving.events | Putaway task created |

## Domain Model

```go
type InboundShipment struct {
    ShipmentID       string
    ASN              AdvanceShippingNotice
    PurchaseOrderID  string
    Supplier         Supplier
    ExpectedItems    []ExpectedItem
    ReceiptRecords   []ReceiptRecord
    Discrepancies    []Discrepancy
    Status           ShipmentStatus
    ReceivingDockID  string
    AssignedWorkerID string
    ArrivedAt        *time.Time
    CompletedAt      *time.Time
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type ShipmentStatus string
const (
    ShipmentStatusExpected   ShipmentStatus = "expected"
    ShipmentStatusArrived    ShipmentStatus = "arrived"
    ShipmentStatusReceiving  ShipmentStatus = "receiving"
    ShipmentStatusInspection ShipmentStatus = "inspection"
    ShipmentStatusCompleted  ShipmentStatus = "completed"
    ShipmentStatusCancelled  ShipmentStatus = "cancelled"
)
```

## Status Transitions

```
expected -> arrived -> receiving -> inspection -> completed
    |          |           |            |
    v          v           v            v
cancelled  cancelled   cancelled    cancelled
```

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-receiving-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8010` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `receiving_db` |
| `KAFKA_BROKERS` | Kafka brokers | `localhost:9092` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry endpoint | `localhost:4317` |
| `LOG_LEVEL` | Log level | `info` |
| `TRACING_ENABLED` | Enable tracing | `true` |

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

- **stow-service**: Receives putaway tasks for received items
- **inventory-service**: Updates stock levels after receiving
- **orchestrator**: Manages receiving workflow
