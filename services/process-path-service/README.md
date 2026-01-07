# Process Path Service

The Process Path Service determines the optimal fulfillment process path for orders based on their characteristics, handling requirements, and item properties.

## Overview

| Property | Value |
|----------|-------|
| Port | 8015 |
| Database | process_path_db (MongoDB) |
| Aggregate Root | ProcessPath |

## Features

- Single vs. multi-item determination
- Special handling requirements detection (gift wrap, hazmat, cold chain, fragile, high-value)
- Consolidation necessity determination
- Process requirement classification
- Station routing recommendations
- Idempotent operations support

## Process Requirements

| Requirement | Trigger | Description |
|-------------|---------|-------------|
| `single_item` | 1 item, qty 1 | Direct pack path |
| `multi_item` | >1 item or qty>1 | Consolidation required |
| `gift_wrap` | GiftWrap flag | Gift wrap station needed |
| `high_value` | Value >= $500 | Verification required |
| `fragile` | Fragile item | Special packing |
| `oversized` | Weight >= 25kg | Oversized handling |
| `hazmat` | Hazardous material | Hazmat compliance |
| `cold_chain` | Temperature sensitive | Cold chain packaging |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/process-paths/determine` | Determine process path for order |
| GET | `/api/v1/process-paths/:pathId` | Get process path by ID |
| GET | `/api/v1/process-paths/order/:orderId` | Get process path for order |

## Request Example

```json
POST /api/v1/process-paths/determine
{
  "orderId": "ORD-12345",
  "items": [
    {
      "sku": "SKU001",
      "quantity": 2,
      "weight": 1.5,
      "isFragile": true,
      "isHazmat": false,
      "requiresColdChain": false
    }
  ],
  "giftWrap": true,
  "giftWrapDetails": {
    "wrapType": "premium",
    "giftMessage": "Happy Birthday!",
    "hidePrice": true
  },
  "totalValue": 299.99
}
```

## Response Example

```json
{
  "pathId": "PP-abc12345",
  "orderId": "ORD-12345",
  "requirements": ["multi_item", "gift_wrap", "fragile"],
  "consolidationRequired": true,
  "giftWrapRequired": true,
  "specialHandling": ["fragile_packing"],
  "targetStationId": "PACK-GIFTWRAP-01"
}
```

## Domain Model

```go
type ProcessRequirement string
const (
    RequirementSingleItem ProcessRequirement = "single_item"
    RequirementMultiItem  ProcessRequirement = "multi_item"
    RequirementGiftWrap   ProcessRequirement = "gift_wrap"
    RequirementHighValue  ProcessRequirement = "high_value"
    RequirementFragile    ProcessRequirement = "fragile"
    RequirementOversized  ProcessRequirement = "oversized"
    RequirementHazmat     ProcessRequirement = "hazmat"
    RequirementColdChain  ProcessRequirement = "cold_chain"
)

type ProcessPath struct {
    ID                    string
    PathID                string
    OrderID               string
    Requirements          []ProcessRequirement
    ConsolidationRequired bool
    GiftWrapRequired      bool
    SpecialHandling       []string
    TargetStationID       string
    CreatedAt             time.Time
    UpdatedAt             time.Time
}
```

## Thresholds

| Threshold | Value | Description |
|-----------|-------|-------------|
| High Value | $500 | Orders >= $500 require verification |
| Oversized Weight | 25kg | Items >= 25kg require special handling |

## Running Locally

```bash
# From the service directory
go run cmd/api/main.go

# Or from the project root
make run-process-path-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDR` | HTTP server address | `:8015` |
| `MONGODB_URI` | MongoDB connection URI | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `process_path_db` |
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

- **order-service**: Provides order details for path determination
- **wes-service**: Uses process path to create execution routes
- **facility-service**: Provides station capabilities for routing
