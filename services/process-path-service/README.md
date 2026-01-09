# Process Path Service

The Process Path Service determines the optimal fulfillment process path for orders based on their characteristics, handling requirements, and item properties. It evaluates each order to identify which special handling procedures are required and whether items need consolidation before packing.

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

## How Process Path Determination Works

When an order is submitted, the service evaluates item characteristics sequentially:

```
Order Received
    ↓
Check Item Count → single_item OR multi_item
    ↓
Check Gift Wrap Flag → gift_wrap (if requested)
    ↓
Check Total Value → high_value (if >= $500)
    ↓
Check Fragile Items → fragile (if any item is fragile)
    ↓
Check Item Weights → oversized (if any >= 30kg)
    ↓
Check Hazmat Flags → hazmat (if any hazmat items)
    ↓
Check Cold Chain → cold_chain (if temperature sensitive)
    ↓
Process Path Complete
```

## Process Requirements

| Requirement | Trigger | Special Handling | Business Reason |
|-------------|---------|------------------|-----------------|
| `single_item` | 1 item, qty 1 | Direct pick-pack | Fastest path - no consolidation needed |
| `multi_item` | >1 item or qty>1 | Consolidation required | Items must be gathered before packing |
| `gift_wrap` | GiftWrap flag | Gift wrap station | Customer experience - professional presentation |
| `high_value` | Value >= $500 | `high_value_verification` | Loss prevention - dual verification required |
| `fragile` | Fragile item | `fragile_packing` | Damage prevention - special materials |
| `oversized` | Weight >= 30kg | `oversized_handling` | Safety - requires equipment/multiple workers |
| `hazmat` | Hazardous material | `hazmat_compliance` | Regulatory - DOT certified handlers only |
| `cold_chain` | Temperature sensitive | `cold_chain_packaging` | Product integrity - maintain temperature range |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/process-paths/determine` | Determine process path for order |
| GET | `/api/v1/process-paths/:pathId` | Get process path by ID |
| GET | `/api/v1/process-paths/order/:orderId` | Get process path for order |
| PUT | `/api/v1/process-paths/:pathId/station` | Assign target station |

## API Examples

### Example 1: Simple Single-Item Order

A customer orders a single item - fastest fulfillment path.

**Request:**
```json
POST /api/v1/process-paths/determine
Content-Type: application/json
X-WMS-Correlation-ID: corr-12345

{
  "orderId": "ORD-2026-0108-001",
  "items": [
    {
      "sku": "ELEC-HDMI-CBL-6FT",
      "productName": "HDMI Cable 6ft",
      "quantity": 1,
      "price": 12.99,
      "weight": 0.15,
      "isFragile": false,
      "isHazmat": false,
      "requiresColdChain": false
    }
  ],
  "giftWrap": false,
  "totalValue": 12.99
}
```

**Response:**
```json
{
  "pathId": "PP-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "orderId": "ORD-2026-0108-001",
  "requirements": ["single_item"],
  "consolidationRequired": false,
  "giftWrapRequired": false,
  "specialHandling": [],
  "createdAt": "2026-01-08T10:30:00Z"
}
```

### Example 2: High-Value Fragile Electronics

A customer orders an expensive TV requiring verification and fragile handling.

**Request:**
```json
POST /api/v1/process-paths/determine
Content-Type: application/json

{
  "orderId": "ORD-2026-0108-002",
  "items": [
    {
      "sku": "ELEC-TV-65IN-OLED",
      "productName": "65-inch OLED Smart TV 4K",
      "quantity": 1,
      "price": 1499.99,
      "weight": 22.0,
      "isFragile": true,
      "isHazmat": false,
      "requiresColdChain": false
    }
  ],
  "giftWrap": false,
  "totalValue": 1499.99
}
```

**Response:**
```json
{
  "pathId": "PP-b2c3d4e5-f6a7-8901-bcde-f23456789012",
  "orderId": "ORD-2026-0108-002",
  "requirements": ["single_item", "high_value", "fragile"],
  "consolidationRequired": false,
  "giftWrapRequired": false,
  "specialHandling": ["high_value_verification", "fragile_packing"],
  "createdAt": "2026-01-08T10:35:00Z"
}
```

### Example 3: Multi-Item Order with Consolidation

Multiple items from different zones require consolidation.

**Request:**
```json
POST /api/v1/process-paths/determine
Content-Type: application/json

{
  "orderId": "ORD-2026-0108-003",
  "items": [
    {
      "sku": "APPAREL-TSHIRT-BLK-M",
      "productName": "Classic T-Shirt Black Medium",
      "quantity": 2,
      "price": 24.99,
      "weight": 0.25
    },
    {
      "sku": "APPAREL-JEANS-BLU-32",
      "productName": "Slim Fit Jeans Blue 32x30",
      "quantity": 1,
      "price": 49.99,
      "weight": 0.6
    },
    {
      "sku": "FOOTWEAR-SNEAKER-WHT-10",
      "productName": "Running Sneakers White Size 10",
      "quantity": 1,
      "price": 89.99,
      "weight": 0.8
    }
  ],
  "giftWrap": false,
  "totalValue": 189.96
}
```

**Response:**
```json
{
  "pathId": "PP-c3d4e5f6-a7b8-9012-cdef-345678901234",
  "orderId": "ORD-2026-0108-003",
  "requirements": ["multi_item"],
  "consolidationRequired": true,
  "giftWrapRequired": false,
  "specialHandling": [],
  "createdAt": "2026-01-08T10:40:00Z"
}
```

### Example 4: Hazmat Order (Car Battery)

Order containing hazardous materials requiring compliance handling.

**Request:**
```json
POST /api/v1/process-paths/determine
Content-Type: application/json

{
  "orderId": "ORD-2026-0108-004",
  "items": [
    {
      "sku": "AUTO-BATT-12V-750CCA",
      "productName": "Car Battery 12V 750 CCA",
      "quantity": 1,
      "price": 149.99,
      "weight": 18.5,
      "isFragile": false,
      "isHazmat": true,
      "hazmatDetails": {
        "class": "8",
        "unNumber": "UN2794",
        "packingGroup": "III",
        "properShippingName": "Batteries, wet, filled with acid",
        "limitedQuantity": false
      },
      "requiresColdChain": false
    }
  ],
  "giftWrap": false,
  "totalValue": 149.99
}
```

**Response:**
```json
{
  "pathId": "PP-d4e5f6a7-b8c9-0123-def4-567890123456",
  "orderId": "ORD-2026-0108-004",
  "requirements": ["single_item", "hazmat"],
  "consolidationRequired": false,
  "giftWrapRequired": false,
  "specialHandling": ["hazmat_compliance"],
  "createdAt": "2026-01-08T10:45:00Z"
}
```

### Example 5: Cold Chain Gift Order

Premium frozen food gift requiring multiple special handling procedures.

**Request:**
```json
POST /api/v1/process-paths/determine
Content-Type: application/json

{
  "orderId": "ORD-2026-0108-005",
  "items": [
    {
      "sku": "FOOD-STEAK-WAGYU-8OZ",
      "productName": "Premium Wagyu Beef Steak 8oz",
      "quantity": 4,
      "price": 89.99,
      "weight": 0.25,
      "isFragile": false,
      "isHazmat": false,
      "requiresColdChain": true,
      "coldChainDetails": {
        "minTempCelsius": -18.0,
        "maxTempCelsius": -12.0,
        "requiresDryIce": true,
        "requiresGelPack": false
      }
    },
    {
      "sku": "FOOD-LOBSTER-TAIL-2PK",
      "productName": "Maine Lobster Tails (2-pack)",
      "quantity": 2,
      "price": 79.99,
      "weight": 0.5,
      "isFragile": false,
      "isHazmat": false,
      "requiresColdChain": true,
      "coldChainDetails": {
        "minTempCelsius": -18.0,
        "maxTempCelsius": -12.0,
        "requiresDryIce": true,
        "requiresGelPack": false
      }
    }
  ],
  "giftWrap": true,
  "giftWrapDetails": {
    "wrapType": "premium",
    "giftMessage": "Happy Birthday! Enjoy this special dinner.",
    "hidePrice": true
  },
  "totalValue": 519.94
}
```

**Response:**
```json
{
  "pathId": "PP-e5f6a7b8-c9d0-1234-ef56-789012345678",
  "orderId": "ORD-2026-0108-005",
  "requirements": ["multi_item", "gift_wrap", "high_value", "cold_chain"],
  "consolidationRequired": true,
  "giftWrapRequired": true,
  "specialHandling": ["high_value_verification", "cold_chain_packaging"],
  "createdAt": "2026-01-08T10:50:00Z"
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

## Configuration Thresholds

| Threshold | Value | Description | Configurable |
|-----------|-------|-------------|--------------|
| High Value | $500.00 USD | Orders >= $500 require verification | Yes |
| Oversized Weight | 30.0 kg | Items >= 30kg require special handling | Yes |

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
| `HIGH_VALUE_THRESHOLD` | High value order threshold | `500.0` |
| `OVERSIZED_WEIGHT_THRESHOLD` | Oversized weight threshold (kg) | `30.0` |

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
- **walling-service**: Handles consolidation for multi-item orders

## Related Documentation

- [ProcessPath Aggregate](/docs/domain-driven-design/aggregates/process-path) - Domain model documentation
- [Planning Workflow](/docs/temporal/workflows/planning) - Uses process path determination
- [Order Fulfillment Workflow](/docs/temporal/workflows/order-fulfillment) - Parent workflow
