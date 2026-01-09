# WMS Platform - Shared Libraries

Common libraries and utilities shared across all WMS Platform services.

## Overview

The shared module provides production-ready infrastructure components that ensure consistency across all microservices.

## Packages

### Core Infrastructure

#### `pkg/mongodb`
MongoDB client with circuit breaker integration.

```go
import "github.com/wms-platform/shared/pkg/mongodb"

// Create production client with circuit breaker
client, err := mongodb.NewProductionClient(ctx, uri, "mydb")
```

#### `pkg/kafka`
Kafka producer/consumer with circuit breaker and CloudEvents support.

```go
import "github.com/wms-platform/shared/pkg/kafka"

// Create producer with circuit breaker
producer := kafka.NewProductionProducer(brokers)

// Publish CloudEvent
producer.PublishEvent(ctx, "topic", event)
```

#### `pkg/temporal`
Temporal workflow client wrapper.

```go
import "github.com/wms-platform/shared/pkg/temporal"

// Create client
client, err := temporal.NewClient(host, namespace)
```

### Event System

#### `pkg/cloudevents`
CloudEvent types and builders for all domain events.

```go
import "github.com/wms-platform/shared/pkg/cloudevents"

// Create event
event := cloudevents.NewOrderReceivedEvent(orderID, customerID, items)
```

**Supported Events (58 types):**
- Order events: `OrderReceived`, `OrderValidated`, `OrderShipped`, etc.
- Wave events: `WaveCreated`, `WaveReleased`, `WaveCompleted`, etc.
- Picking events: `PickTaskCreated`, `ItemPicked`, `PickException`, etc.
- And many more...

### Resilience

#### `pkg/resilience`
Circuit breaker implementation using sony/gobreaker.

```go
import "github.com/wms-platform/shared/pkg/resilience"

// Create circuit breaker
cb := resilience.NewCircuitBreaker("my-service", config)

// Execute with protection
result, err := cb.Execute(func() (interface{}, error) {
    return doSomething()
})
```

#### `pkg/errors`
Standardized error handling with domain-specific error types.

```go
import "github.com/wms-platform/shared/pkg/errors"

// Create domain error
err := errors.NewValidationError("INVALID_ORDER", "Order is missing items")

// HTTP middleware handles conversion to proper status codes
```

### API Utilities

#### `pkg/api`
Pagination, validation, and request handling utilities.

```go
import "github.com/wms-platform/shared/pkg/api"

// Parse pagination from request
page := api.ParsePagination(c)

// Bind and validate request
var req CreateOrderRequest
if err := api.BindAndValidate(c, &req); err != nil {
    return
}
```

#### `pkg/middleware`
Gin middleware stack for logging, tracing, metrics, and error handling.

```go
import "github.com/wms-platform/shared/pkg/middleware"

// Apply middleware stack
router.Use(middleware.Logger())
router.Use(middleware.Metrics("order-service"))
router.Use(middleware.Tracing())
router.Use(middleware.ErrorHandler())
```

#### `pkg/idempotency`
Stripe-compliant idempotency for REST APIs and message deduplication for Kafka consumers.

```go
import "github.com/wms-platform/shared/pkg/idempotency"

// REST API idempotency
idempotencyKeyRepo := idempotency.NewMongoKeyRepository(db)
middlewareConfig.IdempotencyConfig = &idempotency.Config{
    ServiceName:     "order-service",
    Repository:      idempotencyKeyRepo,
    RequireKey:      false,  // Start with optional mode
    OnlyMutating:    true,   // Only POST/PUT/PATCH/DELETE
    RetentionPeriod: 24 * time.Hour,
}

// Kafka message deduplication
idempotencyMsgRepo := idempotency.NewMongoMessageRepository(db)
deduplicatedHandler := idempotency.DeduplicatingHandler(
    &idempotency.ConsumerConfig{
        ServiceName:   "order-service",
        Topic:         "orders.received",
        ConsumerGroup: "order-processor",
        Repository:    idempotencyMsgRepo,
    },
    originalHandler,
)
```

**Features:**
- Stripe-compliant `Idempotency-Key` header pattern for REST APIs
- CloudEvent ID-based deduplication for Kafka consumers
- Request fingerprinting (SHA256) to detect parameter changes
- Response caching for 24 hours with automatic cleanup
- Concurrent request detection (409 Conflict)
- Parameter mismatch detection (422 Unprocessable Entity)
- Comprehensive Prometheus metrics for observability
- MongoDB TTL indexes for automatic cleanup

**Use cases:**
- Preventing duplicate order creation on API retries
- Ensuring exactly-once message processing in Kafka consumers
- Safe network retry scenarios
- Idempotent workflow executions

See [Idempotency Package Documentation](pkg/idempotency/README.md) for complete usage guide.

### Observability

#### `pkg/logging`
Structured logging with slog.

```go
import "github.com/wms-platform/shared/pkg/logging"

logger := logging.NewLogger("order-service", os.Getenv("LOG_LEVEL"))
logger.Info("Order created", "orderId", order.ID)
```

#### `pkg/metrics`
Prometheus metrics helpers.

```go
import "github.com/wms-platform/shared/pkg/metrics"

// Register custom counter
counter := metrics.NewCounter("orders_created_total", "Total orders created")
counter.Inc()
```

#### `pkg/tracing`
OpenTelemetry tracing integration.

```go
import "github.com/wms-platform/shared/pkg/tracing"

// Initialize tracer
shutdown := tracing.InitTracer("order-service", oltpEndpoint)
defer shutdown()

// Create span
ctx, span := tracer.Start(ctx, "CreateOrder")
defer span.End()
```

### Contract Testing

#### `pkg/contracts/openapi`
OpenAPI request/response validation.

```go
import "github.com/wms-platform/shared/pkg/contracts/openapi"

validator, _ := openapi.NewValidator("openapi.yaml")
err := validator.ValidateRequest(req)
err := validator.ValidateResponse(req, resp)
```

#### `pkg/contracts/asyncapi`
AsyncAPI/CloudEvents schema validation.

```go
import "github.com/wms-platform/shared/pkg/contracts/asyncapi"

validator, _ := asyncapi.NewEventValidator("asyncapi.yaml")
err := validator.ValidateEvent(event)
```

### Testing

#### `pkg/testing`
Testcontainers helpers for integration testing.

```go
import "github.com/wms-platform/shared/pkg/testing"

// Start MongoDB container
container, err := testing.StartMongoDBContainer(ctx)
defer container.Terminate(ctx)

// Get connection string
uri := container.ConnectionString()
```

## Configuration

All packages support environment-based configuration:

| Variable | Package | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | logging | Logging level (debug, info, warn, error) |
| `MONGODB_URI` | mongodb | MongoDB connection string |
| `KAFKA_BROKERS` | kafka | Comma-separated broker addresses |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | tracing | OpenTelemetry endpoint |
| `TRACING_ENABLED` | tracing | Enable/disable tracing |
| `IDEMPOTENCY_REQUIRE_KEY` | idempotency | Require Idempotency-Key header (default: false) |
| `IDEMPOTENCY_RETENTION_HOURS` | idempotency | Response cache retention in hours (default: 24) |
| `IDEMPOTENCY_LOCK_TIMEOUT_MINUTES` | idempotency | Lock timeout in minutes (default: 5) |
| `IDEMPOTENCY_MAX_RESPONSE_SIZE_MB` | idempotency | Max cached response size in MB (default: 1) |

## Best Practices

1. **Always use production factories** for MongoDB and Kafka clients to get circuit breaker protection
2. **Use CloudEvents** for all inter-service communication
3. **Apply the middleware stack** in the correct order: Logger → Metrics → Tracing → Idempotency → ErrorHandler
4. **Use standardized errors** from `pkg/errors` for consistent API responses
5. **Enable idempotency** for all services to prevent duplicate operations and ensure exactly-once processing
6. **Use UUID v4** for `Idempotency-Key` headers in client applications

## Documentation

- [Resilience Guide](pkg/RESILIENCE.md) - Detailed circuit breaker and retry documentation
- [Idempotency Guide](pkg/idempotency/README.md) - Comprehensive idempotency implementation guide
