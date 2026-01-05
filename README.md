# WMS Platform

[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Temporal](https://img.shields.io/badge/Temporal-Workflow%20Orchestration-5C4EE5?style=flat)](https://temporal.io/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![CI](https://github.com/temporal-wms/wms-platform/actions/workflows/ci.yml/badge.svg)](https://github.com/temporal-wms/wms-platform/actions)

A production-ready **Warehouse Management System (WMS)** built with microservices architecture, featuring workflow orchestration with Temporal, event-driven communication with Kafka, and comprehensive observability.

## Overview

The WMS Platform manages the complete order fulfillment lifecycle in warehouse operations:

```
Order → Validation → Waving → Routing → Picking → Consolidation → Packing → Shipping
```

### Key Features

- **9 Domain-Driven Microservices** - Each service owns its domain with rich aggregate models
- **Temporal Workflow Orchestration** - Reliable, durable execution of complex business processes
- **Event-Driven Architecture** - 58 CloudEvent types across 11 Kafka topics
- **Saga Pattern** - Distributed transactions with automatic compensation
- **Circuit Breakers** - Resilient inter-service communication
- **Full Observability** - OpenTelemetry tracing, Prometheus metrics, structured logging
- **Contract Testing** - Pact consumer-driven contracts + OpenAPI validation
- **Idempotency** - Stripe-compliant REST API pattern and Kafka message deduplication

## Idempotency

The platform implements comprehensive idempotency across all services to ensure exactly-once semantics for both REST APIs and Kafka message processing.

### REST API Idempotency

All services support the `Idempotency-Key` header following [Stripe's pattern](https://stripe.com/docs/api/idempotent_requests):

```bash
curl -X POST http://localhost:8001/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"customerId": "CUST-001", ...}'
```

**Features:**
- **Request fingerprinting**: SHA256 hash detects parameter changes
- **Response caching**: 24-hour retention with automatic cleanup
- **Concurrent protection**: 409 Conflict for simultaneous requests
- **Parameter validation**: 422 error on parameter mismatch
- **Automatic retries**: Safe network retry scenarios

**Use cases:**
- Prevent duplicate orders on API retries
- Safe payment processing
- Reliable integration with external systems

### Kafka Message Deduplication

All Kafka consumers implement exactly-once message processing using CloudEvent IDs:

```go
// Messages with duplicate IDs are automatically skipped
event := &cloudevents.WMSCloudEvent{
    ID: "evt-550e8400-e29b-41d4-a716-446655440000",
    Type: "com.wms.OrderReceived",
    Data: orderData,
}
```

**Features:**
- **CloudEvent ID-based**: Unique message identification
- **Per-consumer-group**: Independent deduplication per consumer
- **Automatic cleanup**: 24-hour TTL with MongoDB indexes
- **Exactly-once**: Guaranteed single processing per message

**Use cases:**
- Prevent duplicate workflow executions
- Safe event reprocessing
- Kafka consumer group rebalancing

### Storage & Performance

- **MongoDB collections**: `idempotency_keys`, `processed_messages`
- **TTL indexes**: Automatic cleanup after 24 hours
- **Lock acquisition**: ~5-10ms (p99)
- **Storage overhead**: ~1KB per key, ~500B per message

See [Idempotency Package Documentation](shared/pkg/idempotency/README.md) for implementation details.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              WMS Platform                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐                   │
│  │ Order        │    │ Waving       │    │ Routing      │                   │
│  │ Service      │───▶│ Service      │───▶│ Service      │                   │
│  │ :8001        │    │ :8002        │    │ :8003        │                   │
│  └──────────────┘    └──────────────┘    └──────────────┘                   │
│         │                   │                   │                            │
│         ▼                   ▼                   ▼                            │
│  ┌──────────────────────────────────────────────────────────┐               │
│  │                    Orchestrator                           │               │
│  │              (Temporal Workflows)                         │               │
│  └──────────────────────────────────────────────────────────┘               │
│         │                   │                   │                            │
│         ▼                   ▼                   ▼                            │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐                   │
│  │ Picking      │    │ Consolidation│    │ Packing      │                   │
│  │ Service      │───▶│ Service      │───▶│ Service      │                   │
│  │ :8004        │    │ :8005        │    │ :8006        │                   │
│  └──────────────┘    └──────────────┘    └──────────────┘                   │
│         │                   │                   │                            │
│         ▼                   ▼                   ▼                            │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐                   │
│  │ Shipping     │    │ Inventory    │    │ Labor        │                   │
│  │ Service      │    │ Service      │    │ Service      │                   │
│  │ :8007        │    │ :8008        │    │ :8009        │                   │
│  └──────────────┘    └──────────────┘    └──────────────┘                   │
│                                                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│  Infrastructure: MongoDB │ Kafka │ Temporal │ Jaeger │ Prometheus           │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Services

| Service | Port | Description | Aggregate Root |
|---------|------|-------------|----------------|
| [order-service](services/order-service/) | 8001 | Order lifecycle management | Order |
| [waving-service](services/waving-service/) | 8002 | Batch order grouping for picking | Wave |
| [routing-service](services/routing-service/) | 8003 | Pick path optimization | PickRoute |
| [picking-service](services/picking-service/) | 8004 | Warehouse picking operations | PickTask |
| [consolidation-service](services/consolidation-service/) | 8005 | Multi-item order combining | ConsolidationUnit |
| [packing-service](services/packing-service/) | 8006 | Package preparation and labeling | PackTask |
| [shipping-service](services/shipping-service/) | 8007 | Carrier integration and SLAM | Shipment |
| [inventory-service](services/inventory-service/) | 8008 | Stock management | InventoryItem |
| [labor-service](services/labor-service/) | 8009 | Workforce management | Worker |

## Technology Stack

| Category | Technology |
|----------|------------|
| **Language** | Go 1.24 |
| **Web Framework** | Gin |
| **Database** | MongoDB 6.0+ |
| **Message Broker** | Apache Kafka 3.0+ |
| **Workflow Engine** | Temporal |
| **Tracing** | OpenTelemetry, Jaeger |
| **Metrics** | Prometheus |
| **Container** | Docker, Kubernetes |
| **Testing** | Testcontainers, Pact |

## Quick Start

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- Make

### 1. Clone the Repository

```bash
git clone https://github.com/temporal-wms/wms-platform.git
cd wms-platform
```

### 2. Start Infrastructure

```bash
# Start MongoDB, Kafka, Temporal, and observability stack
make docker-infra

# Create Kafka topics
make kafka-create-topics

# Create Temporal namespace
make temporal-namespace
```

### 3. Run Services

```bash
# Build all services
make build

# Run order service
make run-order-service

# Run orchestrator (Temporal worker)
make run-orchestrator
```

### 4. Test the API

```bash
# Create an order
curl -X POST http://localhost:8001/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customerId": "CUST-001",
    "priority": "standard",
    "items": [
      {"sku": "SKU-001", "quantity": 2, "price": 29.99}
    ],
    "shippingAddress": {
      "street": "123 Main St",
      "city": "New York",
      "state": "NY",
      "zipCode": "10001",
      "country": "US"
    }
  }'
```

## Project Structure

```
wms-platform/
├── services/                    # Microservices
│   ├── order-service/
│   ├── waving-service/
│   ├── routing-service/
│   ├── picking-service/
│   ├── consolidation-service/
│   ├── packing-service/
│   ├── shipping-service/
│   ├── inventory-service/
│   └── labor-service/
├── orchestrator/                # Temporal workflows
│   ├── internal/
│   │   ├── workflows/           # Workflow definitions
│   │   └── activities/          # Activity implementations
│   └── tests/
├── shared/                      # Shared libraries
│   └── pkg/
│       ├── cloudevents/         # Event types
│       ├── kafka/               # Kafka client
│       ├── mongodb/             # MongoDB client
│       ├── resilience/          # Circuit breakers
│       ├── errors/              # Error handling
│       └── contracts/           # Contract testing
├── contracts/                   # Pact contracts
├── deployments/
│   ├── docker-compose.yml
│   ├── kubernetes/
│   └── helm/
├── docs/
│   ├── asyncapi.yaml            # Event documentation
│   └── diagrams/                # Architecture diagrams
└── Makefile
```

## Workflows

### Order Fulfillment Workflow

The main workflow orchestrates the complete order fulfillment process:

```go
// Workflow execution
workflow.Go(ctx, "OrderFulfillmentWorkflow", OrderFulfillmentWorkflowParams{
    OrderID: "ORD-12345",
})
```

**Steps:**
1. Validate Order
2. Reserve Inventory
3. Wait for Wave Assignment (signal)
4. Calculate Pick Route
5. Execute Picking (child workflow)
6. Execute Consolidation (child workflow)
7. Execute Packing (child workflow)
8. Execute Shipping (child workflow)
9. Complete Order

### Compensation (Saga Pattern)

On failure, the workflow automatically compensates:
- Release inventory reservations
- Cancel order
- Notify customer

## Testing

```bash
# Run all tests
make test

# Run unit tests only
go test ./... -short

# Run integration tests (requires Docker)
go test ./... -tags=integration

# Run contract tests
make test-contracts
```

## Observability

### Endpoints

All services expose:
- `GET /health` - Liveness probe
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics

### Dashboards

- **Jaeger UI**: http://localhost:16686 - Distributed tracing
- **Prometheus**: http://localhost:9090 - Metrics
- **Temporal UI**: http://localhost:8080 - Workflow monitoring
- **Kafka UI**: http://localhost:8081 - Event monitoring

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LOG_LEVEL` | Logging level | `info` |
| `MONGODB_URI` | MongoDB connection string | `mongodb://localhost:27017` |
| `KAFKA_BROKERS` | Kafka broker addresses | `localhost:9092` |
| `TEMPORAL_HOST` | Temporal server address | `localhost:7233` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry endpoint | `localhost:4317` |
| `IDEMPOTENCY_REQUIRE_KEY` | Require Idempotency-Key header | `false` |
| `IDEMPOTENCY_RETENTION_HOURS` | Response cache retention | `24` |
| `IDEMPOTENCY_LOCK_TIMEOUT_MINUTES` | Lock timeout for concurrent requests | `5` |
| `IDEMPOTENCY_MAX_RESPONSE_SIZE_MB` | Max cached response size | `1` |

## Documentation

- [API Documentation](docs/API_DOCUMENTATION.md)
- [AsyncAPI Specification](docs/asyncapi.yaml)
- [Idempotency Guide](shared/pkg/idempotency/README.md)
- [Resilience Guide](shared/pkg/RESILIENCE.md)
- [Kubernetes Deployment](deployments/kubernetes/README.md)
- [Architecture Diagrams](docs/diagrams/)

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Temporal](https://temporal.io/) - Workflow orchestration
- [CloudEvents](https://cloudevents.io/) - Event specification
- [Testcontainers](https://testcontainers.com/) - Integration testing
