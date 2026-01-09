# Picking Service

The Picking Service manages warehouse picking operations and task assignments.

## Overview

- **Port**: 8004
- **Database**: picking_db (MongoDB)
- **Aggregate Root**: PickTask

## Features

- Pick task creation and assignment
- Real-time picking progress tracking
- Exception handling (stock-outs, damage)
- Worker productivity metrics
- Batch picking support

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/tasks` | Create pick task |
| GET | `/api/v1/tasks/:taskId` | Get task by ID |
| POST | `/api/v1/tasks/:taskId/assign` | Assign worker to task |
| POST | `/api/v1/tasks/:taskId/pick` | Confirm item pick |
| POST | `/api/v1/tasks/:taskId/exception` | Report exception |
| POST | `/api/v1/tasks/:taskId/complete` | Complete task |
| GET | `/api/v1/tasks/wave/:waveId` | Get tasks for a wave |
| GET | `/api/v1/tasks/worker/:workerId` | Get tasks for a worker |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `PickTaskCreated` | wms.picking.events | New pick task created |
| `PickTaskAssigned` | wms.picking.events | Worker assigned to task |
| `ItemPicked` | wms.picking.events | Item picked successfully |
| `PickException` | wms.picking.events | Exception reported |
| `PickTaskCompleted` | wms.picking.events | All items picked |

## Domain Model

```go
type PickTask struct {
    ID          string
    OrderID     string
    WaveID      string
    WorkerID    *string
    RouteID     *string
    Items       []PickItem
    Status      PickTaskStatus
    Priority    int
    Exceptions  []PickException
    StartedAt   *time.Time
    CompletedAt *time.Time
    CreatedAt   time.Time
}

type PickItem struct {
    SKU         string
    Location    string
    Quantity    int
    PickedQty   int
    PickedAt    *time.Time
}

type PickTaskStatus string
const (
    PickTaskStatusPending    PickTaskStatus = "pending"
    PickTaskStatusAssigned   PickTaskStatus = "assigned"
    PickTaskStatusInProgress PickTaskStatus = "in_progress"
    PickTaskStatusCompleted  PickTaskStatus = "completed"
    PickTaskStatusFailed     PickTaskStatus = "failed"
)
```

## Running Locally

```bash
go run cmd/api/main.go
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8004` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `picking_db` |
| `KAFKA_BROKERS` | Kafka brokers | `localhost:9092` |

## Testing

```bash
go test ./...
```

## Related Services

- **routing-service**: Provides pick routes
- **inventory-service**: Confirms stock availability
- **consolidation-service**: Receives picked items
- **labor-service**: Manages worker assignments
