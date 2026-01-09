# WMS E2E Validation Framework

Comprehensive automated testing and validation framework for the WMS Platform.

## Overview

This validation framework provides:
- **Event Validation**: Captures and validates Kafka events against expected sequences
- **Signal Validation**: Verifies Temporal workflow signal delivery and state progression
- **Failure Scenario Testing**: Tests order cancellations, timeouts, and compensation workflows
- **Multi-Order Orchestration**: Validates concurrent order processing

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│        K6 Simulators (Enhanced)                         │
│  full-flow-simulator.js + validation.js library         │
└────────────┬────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────┐
│         Validation Framework                             │
│  ┌──────────────────┐  ┌──────────────────────┐        │
│  │ validation-service│  │ temporal-validator   │        │
│  │ (Port 8080)       │  │ (Port 9090)          │        │
│  └──────────────────┘  └──────────────────────┘        │
└────────────┬────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────┐
│          Infrastructure                                  │
│  Kafka • MongoDB • Temporal • PostgreSQL                │
└─────────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Start Infrastructure

```bash
cd /Users/claudioed/development/github/temporal-war/wms-platform

# Start all services including validation framework
docker-compose -f docker-compose.test.yml up -d

# Wait for services to be healthy
docker-compose -f docker-compose.test.yml ps
```

### 2. Run Validation Tests

```bash
cd wms-runtime

# Run with validation enabled
k6 run -e ENABLE_EVENT_VALIDATION=true -e ENABLE_SIGNAL_VALIDATION=true \
  scripts/scenarios/full-flow-simulator.js
```

### 3. Check Validation Results

```bash
# Get validation report for an order
curl http://localhost:8080/api/v1/validation/report/ORD-001

# Check workflow signals
curl http://localhost:9090/api/v1/signal/list/planning-ORD-001

# Get overall statistics
curl http://localhost:8080/api/v1/stats/summary
```

## Services

### Validation Service (Port 8080)

Captures Kafka events and validates them against expected sequences.

**API Endpoints:**
```
POST   /api/v1/validation/start-tracking/:orderId
GET    /api/v1/validation/events/:orderId
POST   /api/v1/validation/assert/:orderId
POST   /api/v1/validation/sequence/:orderId
GET    /api/v1/validation/report/:orderId
DELETE /api/v1/validation/clear/:orderId
GET    /api/v1/stats/summary
```

### Temporal Validator (Port 9090)

Queries Temporal workflows to validate signal delivery.

**API Endpoints:**
```
GET  /api/v1/workflow/describe/:workflowId
GET  /api/v1/workflow/history/:workflowId
POST /api/v1/workflow/assert-signal/:workflowId
GET  /api/v1/workflow/status/:workflowId
GET  /api/v1/signal/list/:workflowId
```

## K6 Validation Library Usage

### Event Validation

```javascript
import {
  startEventTracking,
  assertEventsReceived,
  validateEventSequence,
} from '../lib/validation.js';

// Start tracking
startEventTracking(orderId);

// Assert events
assertEventsReceived(orderId, [
  'wms.order.received',
  'wms.order.validated',
]);

// Validate sequence
const result = validateEventSequence(orderId, 'standard_flow');
console.log(`Valid: ${result.isValid}`);
```

### Signal Validation

```javascript
import {
  validateSignalDelivered,
  getWorkflowStatus,
} from '../lib/validation.js';

// Validate signal
const delivered = validateSignalDelivered(
  'planning-ORD-001',
  'waveAssigned'
);

// Check status
const status = getWorkflowStatus('planning-ORD-001');
```

## Expected Flow Types

Defined in `validation-service/pkg/testdata/expected_sequences.json`:

- `standard_flow` - Single-item pick-pack
- `multi_item_flow` - Multi-item with consolidation
- `pick_wall_pack_flow` - Multi-zone with put-wall
- `gift_wrap_flow` - Order with gift wrap
- `cancellation_flow` - Order cancellation
- `wave_timeout_flow` - Wave timeout
- `multi_route_flow` - Large multi-route order

## Environment Variables

```bash
# Validation service URLs
export VALIDATION_SERVICE_URL=http://localhost:8080
export TEMPORAL_VALIDATOR_URL=http://localhost:9090

# Enable/disable validation
export ENABLE_EVENT_VALIDATION=true
export ENABLE_SIGNAL_VALIDATION=true
```

## Development

### Build Services

```bash
# Build validation-service
cd wms-runtime/validation-service
go build -o validator ./cmd/validator

# Build temporal-validator
cd ../temporal-validator
go build -o temporal-validator ./cmd
```

### Run Locally

```bash
# Start validation-service
cd wms-runtime/validation-service
export KAFKA_BROKERS=localhost:9092
./validator

# Start temporal-validator
cd ../temporal-validator
export TEMPORAL_HOST=localhost:7233
./temporal-validator
```

## Metrics

Custom k6 metrics:
- `event_validation_success` - Event validation rate
- `signal_delivery_success` - Signal delivery rate

## Troubleshooting

### Events Not Captured

```bash
# Check tracking
curl http://localhost:8080/api/v1/validation/status/ORD-001

# Start tracking
curl -X POST http://localhost:8080/api/v1/validation/start-tracking/ORD-001

# Check logs
docker-compose -f docker-compose.test.yml logs validation-service
```

### Temporal Connection Issues

```bash
# Check Temporal
curl http://localhost:9090/health

# View Temporal UI
open http://localhost:8088
```

## Next Steps

See the full plan at: `/Users/claudioed/.claude/plans/reflective-hopping-frog.md`

Phases completed:
- ✅ Phase 1: Foundation
- ✅ Phase 2: Event Validation
- ✅ Phase 3: Signal Validation

Remaining phases:
- Phase 4: Failure Scenarios
- Phase 5: Multi-Order Orchestration
- Phase 6: CI/CD Integration
