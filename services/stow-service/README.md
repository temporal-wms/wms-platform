# Stow Service

The Stow Service manages putaway tasks and storage location assignment within the WMS Platform.

## Overview

- **Port**: 8011
- **Database**: stow_db (MongoDB)
- **Aggregate Root**: PutawayTask

## Features

- Putaway task creation and management
- Storage strategy support (chaotic, directed, velocity, zone-based)
- Chaotic storage (Amazon-style random placement) as default
- Location assignment based on constraints
- Worker task assignment
- Task lifecycle tracking
- Support for hazmat, cold chain, and oversized items

## Storage Strategies

| Strategy | Description |
|----------|-------------|
| `chaotic` | Random placement (Amazon-style) - **Default** |
| `directed` | System-assigned locations based on rules |
| `velocity` | Places items based on pick frequency |
| `zone_based` | Places items by product category |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/tasks` | List tasks |
| POST | `/api/v1/tasks` | Create a putaway task |
| GET | `/api/v1/tasks/pending` | Get pending tasks |
| GET | `/api/v1/tasks/status/:status` | Get tasks by status |
| GET | `/api/v1/tasks/worker/:workerId` | Get tasks by worker |
| GET | `/api/v1/tasks/shipment/:shipmentId` | Get tasks by shipment |
| GET | `/api/v1/tasks/:taskId` | Get task by ID |
| POST | `/api/v1/tasks/:taskId/assign` | Assign task to worker |
| POST | `/api/v1/tasks/:taskId/start` | Start stow process |
| POST | `/api/v1/tasks/:taskId/stow` | Record stow progress |
| POST | `/api/v1/tasks/:taskId/complete` | Complete task |
| POST | `/api/v1/tasks/:taskId/fail` | Mark task as failed |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `stow.task.created` | wms.stow.events | Putaway task created |
| `stow.task.assigned` | wms.stow.events | Task assigned to worker |
| `stow.location.assigned` | wms.stow.events | Location assigned to task |
| `stow.item.stowed` | wms.stow.events | Items stowed at location |
| `stow.task.completed` | wms.stow.events | Task completed |
| `stow.task.failed` | wms.stow.events | Task failed |

## Domain Model

```go
type PutawayTask struct {
    TaskID           string
    ShipmentID       string
    SKU              string
    ProductName      string
    Quantity         int
    SourceToteID     string
    SourceLocationID string
    TargetLocationID string
    TargetLocation   *StorageLocation
    Strategy         StorageStrategy
    Constraints      ItemConstraints
    Status           PutawayStatus
    AssignedWorkerID string
    Priority         int
    StowedQuantity   int
    FailureReason    string
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type PutawayStatus string
const (
    PutawayStatusPending    PutawayStatus = "pending"
    PutawayStatusAssigned   PutawayStatus = "assigned"
    PutawayStatusInProgress PutawayStatus = "in_progress"
    PutawayStatusCompleted  PutawayStatus = "completed"
    PutawayStatusCancelled  PutawayStatus = "cancelled"
    PutawayStatusFailed     PutawayStatus = "failed"
)
```

## Status Transitions

```
pending -> assigned -> in_progress -> completed
    |          |            |
    v          v            v
cancelled  cancelled    cancelled
                             |
    ^                        v
    |____________________  failed
```

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-stow-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8011` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `stow_db` |
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

- **receiving-service**: Creates putaway tasks after receiving
- **inventory-service**: Updates stock levels and locations
- **orchestrator**: Manages stow workflow
