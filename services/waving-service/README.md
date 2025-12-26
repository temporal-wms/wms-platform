# Waving Service

The Waving Service manages batch grouping of orders for efficient warehouse picking operations.

## Overview

- **Port**: 8002
- **Database**: waves_db (MongoDB)
- **Aggregate Root**: Wave

## Features

- Wave creation and management
- Order grouping by zone, priority, and carrier
- Wave scheduling and release
- Labor allocation planning
- Wave optimization

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/waves` | Create a new wave |
| GET | `/api/v1/waves` | List active waves |
| GET | `/api/v1/waves/:waveId` | Get wave by ID |
| PUT | `/api/v1/waves/:waveId` | Update wave |
| DELETE | `/api/v1/waves/:waveId` | Delete wave |
| POST | `/api/v1/waves/:waveId/orders` | Add order to wave |
| DELETE | `/api/v1/waves/:waveId/orders/:orderId` | Remove order from wave |
| POST | `/api/v1/waves/:waveId/schedule` | Schedule wave |
| POST | `/api/v1/waves/:waveId/release` | Release wave for picking |
| POST | `/api/v1/waves/:waveId/cancel` | Cancel wave |
| GET | `/api/v1/waves/status/:status` | Get waves by status |
| GET | `/api/v1/waves/zone/:zone` | Get waves by zone |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `WaveCreated` | wms.waves.events | New wave created |
| `WaveScheduled` | wms.waves.events | Wave scheduled for release |
| `WaveReleased` | wms.waves.events | Wave released to picking |
| `WaveCompleted` | wms.waves.events | All orders in wave picked |
| `WaveCancelled` | wms.waves.events | Wave cancelled |
| `OrderAddedToWave` | wms.waves.events | Order added to wave |
| `OrderRemovedFromWave` | wms.waves.events | Order removed from wave |

## Domain Model

```go
type Wave struct {
    ID           string
    Type         WaveType
    Status       WaveStatus
    Priority     int
    Zone         string
    Orders       []WaveOrder
    ScheduledAt  *time.Time
    ReleasedAt   *time.Time
    CompletedAt  *time.Time
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type WaveStatus string
const (
    WaveStatusPending   WaveStatus = "pending"
    WaveStatusScheduled WaveStatus = "scheduled"
    WaveStatusReleased  WaveStatus = "released"
    WaveStatusInProgress WaveStatus = "in_progress"
    WaveStatusCompleted WaveStatus = "completed"
    WaveStatusCancelled WaveStatus = "cancelled"
)
```

## Running Locally

```bash
go run cmd/api/main.go
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8002` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `waves_db` |
| `KAFKA_BROKERS` | Kafka brokers | `localhost:9092` |

## Testing

```bash
# Unit tests
go test ./internal/domain/...

# Integration tests
go test ./tests/integration/...
```

## Related Services

- **order-service**: Provides orders for waving
- **routing-service**: Calculates pick routes for released waves
- **labor-service**: Assigns workers to waves
