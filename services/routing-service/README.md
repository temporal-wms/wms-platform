# Routing Service

The Routing Service calculates optimal pick paths through the warehouse to minimize travel time.

## Overview

- **Port**: 8003
- **Database**: routes_db (MongoDB)
- **Aggregate Root**: PickRoute

## Features

- Pick path optimization
- Zone-based routing
- Distance calculation
- Route status tracking
- Multi-order route batching

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/routes` | Calculate a new route |
| GET | `/api/v1/routes/:routeId` | Get route by ID |
| GET | `/api/v1/routes/wave/:waveId` | Get routes for a wave |
| GET | `/api/v1/routes/picker/:pickerId` | Get routes for a picker |
| PUT | `/api/v1/routes/:routeId/start` | Start route execution |
| PUT | `/api/v1/routes/:routeId/complete` | Complete route |
| GET | `/api/v1/routes/zone/:zone` | Get routes by zone |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `RouteCalculated` | wms.routes.events | Route calculated |
| `RouteStarted` | wms.routes.events | Picker started route |
| `StopCompleted` | wms.routes.events | Pick location completed |
| `RouteCompleted` | wms.routes.events | All stops completed |

## Domain Model

```go
type PickRoute struct {
    ID            string
    WaveID        string
    PickerID      *string
    Stops         []RouteStop
    TotalDistance float64
    EstimatedTime int // minutes
    Status        RouteStatus
    StartedAt     *time.Time
    CompletedAt   *time.Time
    CreatedAt     time.Time
}

type RouteStop struct {
    Sequence   int
    LocationID string
    Zone       string
    Aisle      string
    Rack       string
    Level      string
    Items      []RouteItem
    Completed  bool
}
```

## Running Locally

```bash
go run cmd/api/main.go
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8003` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `routes_db` |
| `KAFKA_BROKERS` | Kafka brokers | `localhost:9092` |

## Testing

```bash
go test ./...
```

## Related Services

- **waving-service**: Provides waves for routing
- **picking-service**: Uses routes for pick tasks
- **labor-service**: Assigns pickers to routes
