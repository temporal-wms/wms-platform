# Walling Service

The Walling Service manages put-wall operations for sorting items from picking totes into destination bins for multi-item order consolidation.

## Overview

| Property | Value |
|----------|-------|
| Port | 8017 |
| Database | walling_db (MongoDB) |
| Aggregate Root | WallingTask |

## Features

- Walling task creation and lifecycle management
- Item sorting from source totes to destination bins
- Put-wall assignment and tracking
- Item verification during sorting
- Progress tracking for sorted items
- Priority-based task assignment
- WES route integration
- Idempotent operations support

## What is Walling?

Walling (or "put-wall") is a warehouse operation where items from multiple orders are picked into totes during the picking process, then sorted ("walled") into individual order bins at a put-wall station. This enables efficient batch picking while maintaining order accuracy.

```
┌─────────────────────────────────────────────────────────────┐
│                        Put Wall                              │
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐   │
│  │Bin 1│ │Bin 2│ │Bin 3│ │Bin 4│ │Bin 5│ │Bin 6│ │Bin 7│   │
│  │ORD-1│ │ORD-2│ │ORD-3│ │ORD-4│ │ORD-5│ │ORD-6│ │ORD-7│   │
│  └──▲──┘ └──▲──┘ └──▲──┘ └──▲──┘ └──▲──┘ └──▲──┘ └──▲──┘   │
│     │      │      │      │      │      │      │            │
│     └──────┴──────┴──────┼──────┴──────┴──────┘            │
│                          │                                  │
│                     ┌────┴────┐                             │
│                     │  Tote   │  ◄── Items from picking     │
│                     └─────────┘                             │
└─────────────────────────────────────────────────────────────┘
```

## API Endpoints

### Task Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/tasks` | Create walling task |
| GET | `/api/v1/tasks/:taskId` | Get task details |
| GET | `/api/v1/tasks/pending` | Get pending tasks |
| GET | `/api/v1/tasks/order/:orderId` | Get tasks for order |

### Task Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/tasks/:taskId/assign` | Assign walliner to task |
| POST | `/api/v1/tasks/:taskId/start` | Start task |
| POST | `/api/v1/tasks/:taskId/sort` | Sort item to bin |
| POST | `/api/v1/tasks/:taskId/complete` | Complete task |
| POST | `/api/v1/tasks/:taskId/cancel` | Cancel task |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `walling.task.created` | wms.walling.events | Task created |
| `walling.task.assigned` | wms.walling.events | Walliner assigned |
| `walling.item.sorted` | wms.walling.events | Item sorted to bin |
| `walling.task.completed` | wms.walling.events | All items sorted |

## Domain Model

```go
type WallingTaskStatus string
const (
    WallingTaskStatusPending    WallingTaskStatus = "pending"
    WallingTaskStatusAssigned   WallingTaskStatus = "assigned"
    WallingTaskStatusInProgress WallingTaskStatus = "in_progress"
    WallingTaskStatusCompleted  WallingTaskStatus = "completed"
    WallingTaskStatusCancelled  WallingTaskStatus = "cancelled"
)

type WallingTask struct {
    ID             string
    TaskID         string             // WT-XXXXXXXX format
    OrderID        string
    WaveID         string
    RouteID        string             // WES route reference
    WallinerID     string
    Status         WallingTaskStatus
    SourceTotes    []SourceTote
    DestinationBin string
    PutWallID      string
    ItemsToSort    []ItemToSort
    SortedItems    []SortedItem
    Station        string
    Priority       int
    CreatedAt      time.Time
    CompletedAt    *time.Time
}

type ItemToSort struct {
    SKU        string
    Quantity   int
    FromToteID string
    SortedQty  int    // Progress tracking
}

type SortedItem struct {
    SKU        string
    Quantity   int
    FromToteID string
    ToBinID    string
    SortedAt   time.Time
    Verified   bool
}
```

## Task Lifecycle

```
┌─────────┐   ┌──────────┐   ┌─────────────┐   ┌───────────┐
│ Pending │──>│ Assigned │──>│ In Progress │──>│ Completed │
└─────────┘   └──────────┘   └─────────────┘   └───────────┘
                                    │
                              (sort items)
```

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-walling-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8017` |
| `MONGODB_URI` | MongoDB connection URI | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `walling_db` |
| `LOG_LEVEL` | Logging level | `info` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector | `localhost:4317` |
| `TRACING_ENABLED` | Enable distributed tracing | `true` |
| `ENVIRONMENT` | Deployment environment | `development` |

## Testing

```bash
# Unit tests
go test ./internal/domain/...

# Integration tests
go test ./tests/integration/...
```

## Related Services

- **picking-service**: Provides source totes for walling
- **wes-service**: Orchestrates walling as a stage
- **consolidation-service**: Receives completed walling for packing
- **facility-service**: Provides put-wall station information
