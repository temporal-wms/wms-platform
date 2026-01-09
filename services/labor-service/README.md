# Labor Service

The Labor Service manages workforce assignments, shifts, and performance tracking.

## Overview

- **Port**: 8009
- **Database**: labor_db (MongoDB)
- **Aggregate Root**: Worker

## Features

- Worker management
- Skill tracking
- Shift scheduling
- Task assignment
- Performance metrics
- Availability tracking

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/workers` | List all workers |
| GET | `/api/v1/workers/:workerId` | Get worker by ID |
| GET | `/api/v1/workers/available` | Get available workers |
| POST | `/api/v1/workers/:workerId/shift/start` | Start shift |
| POST | `/api/v1/workers/:workerId/shift/end` | End shift |
| POST | `/api/v1/tasks` | Assign task to worker |
| POST | `/api/v1/tasks/:taskId/complete` | Complete task |
| GET | `/api/v1/workers/:workerId/performance` | Get performance metrics |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `ShiftStarted` | wms.labor.events | Worker started shift |
| `ShiftEnded` | wms.labor.events | Worker ended shift |
| `TaskAssigned` | wms.labor.events | Task assigned to worker |
| `TaskCompleted` | wms.labor.events | Worker completed task |
| `PerformanceRecorded` | wms.labor.events | Performance metrics updated |

## Domain Model

```go
type Worker struct {
    ID           string
    EmployeeID   string
    Name         string
    Email        string
    Skills       []Skill
    CurrentShift *Shift
    CurrentTask  *TaskAssignment
    Zone         string
    Status       WorkerStatus
    Performance  PerformanceMetrics
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type Skill struct {
    Type        SkillType  // picking, packing, forklift
    Level       int        // 1-5
    CertifiedAt time.Time
    ExpiresAt   *time.Time
}

type Shift struct {
    ID        string
    Type      ShiftType  // morning, afternoon, night
    StartedAt time.Time
    EndedAt   *time.Time
    Breaks    []Break
}

type WorkerStatus string
const (
    WorkerStatusOffline   WorkerStatus = "offline"
    WorkerStatusAvailable WorkerStatus = "available"
    WorkerStatusOnTask    WorkerStatus = "on_task"
    WorkerStatusOnBreak   WorkerStatus = "on_break"
)
```

## Running Locally

```bash
go run cmd/api/main.go
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8009` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `labor_db` |
| `KAFKA_BROKERS` | Kafka brokers | `localhost:9092` |

## Testing

```bash
go test ./...
```

## Related Services

- **picking-service**: Assigns pickers
- **packing-service**: Assigns packers
- **orchestrator**: Manages labor in workflows
