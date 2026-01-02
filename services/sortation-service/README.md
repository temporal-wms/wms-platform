# Sortation Service

The Sortation Service manages package sortation operations, grouping packages by destination and carrier for efficient dispatch.

## Overview

- **Port**: 8012
- **Database**: sortation_db (MongoDB)
- **Aggregate Root**: SortationBatch

## Features

- Batch-based package grouping by destination
- Chute assignment for physical sortation
- Destination group routing (zip code prefix/region)
- Multi-carrier support
- Dispatch dock and trailer management
- Sortation progress tracking

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/batches` | List batches (paginated) |
| POST | `/api/v1/batches` | Create a new sortation batch |
| GET | `/api/v1/batches/status/:status` | Get batches by status |
| GET | `/api/v1/batches/ready` | Get batches ready for dispatch |
| GET | `/api/v1/batches/:batchId` | Get batch by ID |
| POST | `/api/v1/batches/:batchId/packages` | Add package to batch |
| POST | `/api/v1/batches/:batchId/sort` | Sort package to chute |
| POST | `/api/v1/batches/:batchId/ready` | Mark batch as ready |
| POST | `/api/v1/batches/:batchId/dispatch` | Dispatch batch |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `sortation.batch.created` | wms.sortation.events | Sortation batch created |
| `sortation.package.received` | wms.sortation.events | Package received for sortation |
| `sortation.package.sorted` | wms.sortation.events | Package sorted to chute |
| `sortation.batch.ready` | wms.sortation.events | Batch ready for dispatch |
| `sortation.batch.dispatched` | wms.sortation.events | Batch dispatched |

## Domain Model

```go
type SortationBatch struct {
    BatchID          string
    SortationCenter  string
    DestinationGroup string
    CarrierID        string
    Packages         []SortedPackage
    AssignedChute    string
    Status           SortationStatus
    TotalPackages    int
    SortedCount      int
    TotalWeight      float64
    TrailerID        string
    DispatchDock     string
    ScheduledDispatch *time.Time
    CreatedAt        time.Time
    UpdatedAt        time.Time
    DispatchedAt     *time.Time
}

type SortedPackage struct {
    PackageID      string
    OrderID        string
    TrackingNumber string
    Destination    string
    CarrierID      string
    Weight         float64
    AssignedChute  string
    SortedAt       *time.Time
    SortedBy       string
    IsSorted       bool
}

type SortationStatus string
const (
    SortationStatusReceiving   SortationStatus = "receiving"
    SortationStatusSorting     SortationStatus = "sorting"
    SortationStatusReady       SortationStatus = "ready"
    SortationStatusDispatching SortationStatus = "dispatching"
    SortationStatusDispatched  SortationStatus = "dispatched"
    SortationStatusCancelled   SortationStatus = "cancelled"
)
```

## Status Transitions

```
receiving -> sorting -> ready -> dispatching -> dispatched
    |           |          |
    v           v          v
cancelled   cancelled  cancelled
```

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-sortation-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8012` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `sortation_db` |
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

- **packing-service**: Creates packages for sortation
- **shipping-service**: Manages manifests and dispatch
- **orchestrator**: Manages sortation workflow
