# WES Service (Warehouse Execution System)

The WES Service orchestrates order execution through multiple warehouse stages, managing task routes from picking through shipping with real-time progress tracking.

## Overview

| Property | Value |
|----------|-------|
| Port | 8016 |
| Database | wes_db (MongoDB) |
| Aggregate Roots | TaskRoute, StageTemplate |

## Features

- Stage template management for different process paths
- Task route creation and lifecycle management
- Multi-stage workflow orchestration
- Worker assignment to stages
- Stage execution tracking (start, complete, fail)
- Process path integration
- Outbox pattern for reliable event publishing
- Idempotent operations support

## Process Path Types

| Path Type | Stages | Description |
|-----------|--------|-------------|
| `single_item` | pick -> pack -> ship | Direct fulfillment |
| `multi_item` | pick -> consolidate -> pack -> ship | Consolidation required |
| `gift_wrap` | pick -> consolidate -> gift_wrap -> pack -> ship | Gift wrapping included |

## API Endpoints

### Execution Plans

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/execution-plans/resolve` | Resolve execution plan for order |

### Task Routes

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/routes` | Create task route |
| GET | `/api/v1/routes/:routeId` | Get route details |
| GET | `/api/v1/routes/order/:orderId` | Get routes for order |

### Stage Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/routes/:routeId/assign` | Assign worker to current stage |
| POST | `/api/v1/routes/:routeId/stages/start` | Start current stage |
| POST | `/api/v1/routes/:routeId/stages/complete` | Complete current stage |
| POST | `/api/v1/routes/:routeId/stages/fail` | Fail current stage |

### Templates

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/templates` | List stage templates |
| GET | `/api/v1/templates/:templateId` | Get template details |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `wes.route.created` | wms.wes.events | Route created |
| `wes.stage.assigned` | wms.wes.events | Worker assigned to stage |
| `wes.stage.started` | wms.wes.events | Stage execution started |
| `wes.stage.completed` | wms.wes.events | Stage completed |
| `wes.stage.failed` | wms.wes.events | Stage failed |
| `wes.route.completed` | wms.wes.events | All stages completed |

## Domain Model

```go
type RouteStatus string
const (
    RouteStatusPending    RouteStatus = "pending"
    RouteStatusInProgress RouteStatus = "in_progress"
    RouteStatusCompleted  RouteStatus = "completed"
    RouteStatusFailed     RouteStatus = "failed"
)

type StageType string
const (
    StageTypePick        StageType = "pick"
    StageTypeConsolidate StageType = "consolidate"
    StageTypeGiftWrap    StageType = "gift_wrap"
    StageTypePack        StageType = "pack"
    StageTypeShip        StageType = "ship"
)

type TaskRoute struct {
    ID              string
    RouteID         string
    OrderID         string
    WaveID          string
    PathTemplateID  string
    PathType        ProcessPathType
    CurrentStageIdx int
    Stages          []StageStatus
    Status          RouteStatus
    SpecialHandling []string
    ProcessPathID   string
    CreatedAt       time.Time
    CompletedAt     *time.Time
}

type StageStatus struct {
    StageType   StageType
    Status      StageStatusType
    WorkerID    string
    TaskID      string
    StartedAt   *int64
    CompletedAt *int64
    Error       string
}
```

## Stage Flow

```
┌─────────┐   ┌─────────────┐   ┌──────────┐   ┌─────────┐   ┌─────────┐
│  Pick   │──>│ Consolidate │──>│ GiftWrap │──>│  Pack   │──>│  Ship   │
└─────────┘   └─────────────┘   └──────────┘   └─────────┘   └─────────┘
                 (optional)       (optional)
```

## Running Locally

```bash
# Start the API server
go run cmd/api/main.go

# Start the Temporal worker (separate terminal)
go run cmd/worker/main.go
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8016` |
| `MONGODB_URI` | MongoDB connection URI | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `wes_db` |
| `KAFKA_BROKERS` | Kafka broker addresses | `localhost:9092` |
| `PROCESS_PATH_SERVICE_URL` | Process path service URL | `http://process-path-service:8015` |
| `LOG_LEVEL` | Logging level | `info` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector | `localhost:4317` |
| `TRACING_ENABLED` | Enable distributed tracing | `true` |
| `ENVIRONMENT` | Deployment environment | `development` |

## Testing & Coverage

```bash
# Run focused unit suites inside tests/
go test ./tests/...

# Collect cross-package coverage (fails locally if <90%)
make coverage
```

## Integration Testing

```bash
# Start MongoDB, Kafka, Temporal, API, and worker containers
make integration-up

# Run health-check integration tests (requires API up)
make integration-test

# Tear the stack down
make integration-down
```

Set `WES_BASE_URL` when running `make integration-test` if the API is exposed on a non-default host or port.

## Related Services

- **process-path-service**: Provides process path requirements
- **picking-service**: Executes pick stages
- **consolidation-service**: Executes consolidation stages
- **packing-service**: Executes pack stages
- **shipping-service**: Executes ship stages
