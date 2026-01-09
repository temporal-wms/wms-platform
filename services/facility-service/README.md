# Facility Service

The Facility Service manages warehouse stations, their capabilities, and process path routing within the WMS Platform.

## Overview

- **Port**: 8010
- **Database**: facility_db (MongoDB)
- **Aggregate Root**: Station

## Features

- Station management (CRUD operations)
- Capability-based process path routing
- Station type classification
- Equipment tracking
- Worker assignment to stations
- Zone-based organization
- Concurrent task management

## Station Types

| Type | Description |
|------|-------------|
| `packing` | Item packing stations |
| `consolidation` | Multi-item order consolidation |
| `shipping` | Shipping label/manifest stations |
| `receiving` | Inbound shipment receiving |
| `stow` | Putaway stations |
| `slam` | Scan-Label-Apply-Manifest stations |
| `sortation` | Package sortation stations |
| `qc` | Quality control/inspection stations |

## Station Capabilities

| Capability | Description |
|------------|-------------|
| `single_item` | Single-item order handling |
| `multi_item` | Multi-item order handling |
| `gift_wrap` | Gift wrapping service |
| `hazmat` | Hazardous materials handling |
| `oversized` | Oversized item handling |
| `fragile` | Fragile item handling |
| `cold_chain` | Cold chain/refrigerated items |
| `high_value` | High-value item handling |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/stations` | List stations (paginated) |
| POST | `/api/v1/stations` | Create a new station |
| GET | `/api/v1/stations/:stationId` | Get station by ID |
| PUT | `/api/v1/stations/:stationId` | Update station |
| DELETE | `/api/v1/stations/:stationId` | Delete station |
| PUT | `/api/v1/stations/:stationId/capabilities` | Set all capabilities |
| POST | `/api/v1/stations/:stationId/capabilities/:capability` | Add capability |
| DELETE | `/api/v1/stations/:stationId/capabilities/:capability` | Remove capability |
| PUT | `/api/v1/stations/:stationId/status` | Set station status |
| POST | `/api/v1/stations/find-capable` | Find stations by capabilities |
| GET | `/api/v1/stations/zone/:zone` | Get stations by zone |
| GET | `/api/v1/stations/type/:type` | Get stations by type |
| GET | `/api/v1/stations/status/:status` | Get stations by status |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `station.created` | wms.facility.events | Station created |
| `station.capability.added` | wms.facility.events | Capability added to station |
| `station.capability.removed` | wms.facility.events | Capability removed from station |
| `station.capabilities.updated` | wms.facility.events | Capabilities bulk updated |
| `station.status.changed` | wms.facility.events | Station status changed |
| `station.worker.assigned` | wms.facility.events | Worker assigned to station |

## Domain Model

```go
type Station struct {
    StationID          string
    Name               string
    Zone               string
    StationType        StationType
    Status             StationStatus
    Capabilities       []StationCapability
    MaxConcurrentTasks int
    CurrentTasks       int
    AssignedWorkerID   string
    Equipment          []StationEquipment
    CreatedAt          time.Time
    UpdatedAt          time.Time
}

type StationEquipment struct {
    EquipmentID   string
    EquipmentType string  // scale, printer, cold_storage, hazmat_cabinet
    Status        string  // active, inactive, maintenance
}

type StationStatus string
const (
    StationStatusActive      StationStatus = "active"
    StationStatusInactive    StationStatus = "inactive"
    StationStatusMaintenance StationStatus = "maintenance"
)
```

## Process Path Routing

The facility service enables intelligent process path routing by:

1. **Capability Matching**: Find stations that have ALL required capabilities for an order
2. **Zone Filtering**: Route to stations in specific warehouse zones
3. **Type Selection**: Select appropriate station types for each process step
4. **Capacity Check**: Ensure stations have available capacity

Example: A gift-wrapped hazmat order routes to packing stations with `gift_wrap` + `hazmat` capabilities.

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-facility-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8010` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `facility_db` |
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

- **packing-service**: Uses stations for packing operations
- **stow-service**: Uses stations for putaway operations
- **orchestrator**: Queries capabilities for process path routing
