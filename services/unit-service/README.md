# Unit Service

The Unit Service tracks individual physical units through the warehouse lifecycle, providing granular visibility and audit trails for each item from receiving to shipping.

## Overview

| Property | Value |
|----------|-------|
| Port | 8014 |
| Database | unit_db (MongoDB) |
| Aggregate Root | Unit |

## Features

- Individual unit tracking through warehouse lifecycle
- Multi-tenant support for 3PL operations
- Status management (received, reserved, staged, picked, consolidated, packed, shipped)
- Complete movement audit trail
- Order-to-unit association
- Exception handling and resolution
- Route and tote assignment tracking
- Idempotent operations support

## API Endpoints

### Unit Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/units` | Create units at receiving |
| POST | `/api/v1/units/reserve` | Reserve units for an order |
| GET | `/api/v1/units/order/:orderId` | Get units for an order |
| GET | `/api/v1/units/:unitId` | Get unit details |
| GET | `/api/v1/units/:unitId/audit` | Get unit audit trail |

### Status Transitions

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/units/:unitId/pick` | Confirm unit picked |
| POST | `/api/v1/units/:unitId/consolidate` | Confirm unit consolidated |
| POST | `/api/v1/units/:unitId/pack` | Confirm unit packed |
| POST | `/api/v1/units/:unitId/ship` | Confirm unit shipped |

### Exception Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/units/:unitId/exception` | Create exception for unit |
| GET | `/api/v1/exceptions/order/:orderId` | Get exceptions for order |
| GET | `/api/v1/exceptions/unresolved` | Get unresolved exceptions |
| POST | `/api/v1/exceptions/:exceptionId/resolve` | Resolve exception |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `unit.created` | wms.units.events | Unit created at receiving |
| `unit.reserved` | wms.units.events | Unit reserved for order |
| `unit.staged` | wms.units.events | Unit staged for picking |
| `unit.picked` | wms.units.events | Unit picked into tote |
| `unit.consolidated` | wms.units.events | Unit consolidated |
| `unit.packed` | wms.units.events | Unit packed into package |
| `unit.shipped` | wms.units.events | Unit shipped |
| `unit.exception` | wms.units.events | Exception occurred |

## Domain Model

```go
type UnitStatus string
const (
    UnitStatusReceived     UnitStatus = "received"
    UnitStatusReserved     UnitStatus = "reserved"
    UnitStatusStaged       UnitStatus = "staged"
    UnitStatusPicked       UnitStatus = "picked"
    UnitStatusConsolidated UnitStatus = "consolidated"
    UnitStatusPacked       UnitStatus = "packed"
    UnitStatusShipped      UnitStatus = "shipped"
    UnitStatusException    UnitStatus = "exception"
)

type Unit struct {
    ID                string
    UnitID            string
    SKU               string
    TenantID          string      // 3PL operator
    FacilityID        string      // Physical facility
    WarehouseID       string      // Specific warehouse
    SellerID          string      // Merchant owner
    OrderID           string
    ShipmentID        string
    Status            UnitStatus
    CurrentLocationID string
    ToteID            string
    PackageID         string
    RouteID           string      // Picking route
    Movements         []UnitMovement
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

type UnitMovement struct {
    MovementID     string
    FromLocationID string
    ToLocationID   string
    FromStatus     UnitStatus
    ToStatus       UnitStatus
    StationID      string
    HandlerID      string
    Timestamp      time.Time
    Notes          string
}
```

## Unit Lifecycle

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ Received │───>│ Reserved │───>│  Staged  │───>│  Picked  │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
                                                      │
┌──────────┐    ┌──────────┐    ┌──────────────┐     │
│ Shipped  │<───│  Packed  │<───│ Consolidated │<────┘
└──────────┘    └──────────┘    └──────────────┘
                                (multi-item only)
```

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-unit-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8014` |
| `MONGODB_URI` | MongoDB connection URI | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `unit_db` |
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

- **receiving-service**: Creates units at receiving
- **picking-service**: Updates unit status during picking
- **consolidation-service**: Consolidates multi-item orders
- **packing-service**: Records packing status
- **shipping-service**: Records shipment status
