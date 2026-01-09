# Packing Service

The Packing Service manages package preparation, labeling, and sealing operations.

## Overview

- **Port**: 8006
- **Database**: packing_db (MongoDB)
- **Aggregate Root**: PackTask

## Features

- Pack task management
- Package type selection
- Weight measurement
- Shipping label generation
- Package sealing verification

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/tasks` | Create pack task |
| GET | `/api/v1/tasks/:taskId` | Get task by ID |
| POST | `/api/v1/tasks/:taskId/suggest-packaging` | Get package suggestion |
| POST | `/api/v1/tasks/:taskId/seal` | Seal package |
| POST | `/api/v1/tasks/:taskId/label` | Apply shipping label |
| POST | `/api/v1/tasks/:taskId/complete` | Complete pack task |
| GET | `/api/v1/tasks/order/:orderId` | Get tasks for order |
| GET | `/api/v1/tasks/station/:station` | Get tasks by station |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `PackTaskCreated` | wms.packing.events | New pack task created |
| `PackagingSuggested` | wms.packing.events | Package type selected |
| `PackageSealed` | wms.packing.events | Package sealed |
| `LabelApplied` | wms.packing.events | Shipping label applied |
| `PackTaskCompleted` | wms.packing.events | Packing complete |

## Domain Model

```go
type PackTask struct {
    ID              string
    OrderID         string
    ConsolidationID string
    Station         string
    Items           []PackItem
    Package         *Package
    Status          PackTaskStatus
    WorkerID        *string
    StartedAt       *time.Time
    CompletedAt     *time.Time
    CreatedAt       time.Time
}

type Package struct {
    Type         string
    Weight       float64
    Dimensions   Dimensions
    TrackingCode string
    LabelURL     string
    Carrier      string
}

type PackTaskStatus string
const (
    PackTaskStatusPending    PackTaskStatus = "pending"
    PackTaskStatusInProgress PackTaskStatus = "in_progress"
    PackTaskStatusSealed     PackTaskStatus = "sealed"
    PackTaskStatusLabeled    PackTaskStatus = "labeled"
    PackTaskStatusCompleted  PackTaskStatus = "completed"
)
```

## Running Locally

```bash
go run cmd/api/main.go
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8006` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `packing_db` |
| `KAFKA_BROKERS` | Kafka brokers | `localhost:9092` |

## Testing

```bash
go test ./...
```

## Related Services

- **consolidation-service**: Provides consolidated orders
- **shipping-service**: Receives packed packages
- **labor-service**: Manages packer assignments
