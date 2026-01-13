---
sidebar_position: 10
slug: /architecture/shared-packages
---

# Shared Packages

Common packages used across all WMS platform services, providing consistent patterns and reducing code duplication.

## Package Overview

| Package | Import Path | Purpose |
|---------|-------------|---------|
| `idempotency` | `shared/pkg/idempotency` | Request deduplication for APIs and Kafka |
| `outbox` | `shared/pkg/outbox` | Transactional outbox pattern |
| `cloudevents` | `shared/pkg/cloudevents` | 58 CloudEvent type definitions |
| `domain` | `shared/pkg/domain` | Shared value objects and enums |
| `middleware` | `shared/pkg/middleware` | HTTP middleware components |
| `kafka` | `shared/pkg/kafka` | Kafka producer/consumer wrappers |
| `mongodb` | `shared/pkg/mongodb` | MongoDB client and helpers |
| `temporal` | `shared/pkg/temporal` | Temporal client configuration |
| `tenant` | `shared/pkg/tenant` | Multi-tenancy support |
| `resilience` | `shared/pkg/resilience` | Circuit breakers |
| `tracing` | `shared/pkg/tracing` | OpenTelemetry integration |
| `logging` | `shared/pkg/logging` | Structured logging |
| `metrics` | `shared/pkg/metrics` | Prometheus metrics |
| `testing` | `shared/pkg/testing` | Test utilities and testcontainers |
| `errors` | `shared/pkg/errors` | Standardized error handling |
| `api` | `shared/pkg/api` | API pagination and validation |
| `contracts` | `shared/pkg/contracts` | Pact contract testing |

---

## Idempotency Package

Ensures exactly-once semantics for REST API requests and Kafka messages using the Stripe idempotency pattern.

### REST API Middleware

```go
import "github.com/wms-platform/shared/pkg/idempotency"

// Configure idempotency middleware
config := &idempotency.Config{
    Repository:    mongoRepo,
    TTL:           24 * time.Hour,
    RequireKey:    true,              // Require Idempotency-Key header
    OnlyMutating:  true,              // Only POST/PUT/PATCH/DELETE
    MaxKeyLength:  255,
    UserIDExtractor: func(c *gin.Context) string {
        return c.GetHeader("X-User-ID")
    },
}

router.Use(idempotency.Middleware(config))
```

### HTTP Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Idempotency-Key` | Configurable | Unique request identifier (UUID recommended) |

### Response Behavior

| Scenario | HTTP Status | Behavior |
|----------|-------------|----------|
| New request | Original | Execute and cache response |
| Duplicate (same body) | Cached | Return cached response |
| Duplicate (different body) | 422 | Return conflict error |
| Expired key | Original | Treat as new request |

### Kafka Consumer Middleware

```go
import "github.com/wms-platform/shared/pkg/idempotency"

// Wrap Kafka message handler
handler := idempotency.ConsumerMiddleware(
    repo,
    24*time.Hour,
    func(ctx context.Context, msg *kafka.Message) error {
        // Process message
        return nil
    },
)
```

---

## Outbox Package

Implements the transactional outbox pattern for reliable event publishing with MongoDB.

### Usage

```go
import "github.com/wms-platform/shared/pkg/outbox"

// Create outbox event within transaction
event, err := outbox.NewOutboxEvent(
    order.ID,           // Aggregate ID
    "Order",            // Aggregate type
    "wms.orders.events", // Kafka topic
    orderCreatedEvent,  // Domain event
)

// Store in same transaction as aggregate
session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) error {
    // Save aggregate
    _, err := ordersCollection.InsertOne(sessCtx, order)
    if err != nil {
        return err
    }

    // Save outbox event
    _, err = outboxCollection.InsertOne(sessCtx, event)
    return err
})
```

### Outbox Event Structure

```go
type OutboxEvent struct {
    ID            string          `bson:"_id"`
    AggregateID   string          `bson:"aggregateId"`
    AggregateType string          `bson:"aggregateType"`
    EventType     string          `bson:"eventType"`
    Topic         string          `bson:"topic"`
    Payload       json.RawMessage `bson:"payload"`
    CreatedAt     time.Time       `bson:"createdAt"`
    PublishedAt   *time.Time      `bson:"publishedAt"`
    RetryCount    int             `bson:"retryCount"`
    LastError     string          `bson:"lastError"`
    MaxRetries    int             `bson:"maxRetries"` // Default: 10
}
```

### Outbox Publisher

```go
// Background publisher polls and publishes events
publisher := outbox.NewPublisher(
    outboxRepo,
    kafkaProducer,
    &outbox.PublisherConfig{
        PollInterval:  time.Second,
        BatchSize:     100,
        MaxRetries:    10,
    },
)

go publisher.Start(ctx)
```

---

## CloudEvents Package

Defines 58 CloudEvent types used across the platform following the CloudEvents 1.0 specification.

### Event Categories

| Category | Event Count | Example Types |
|----------|-------------|---------------|
| Order | 12 | `OrderCreated`, `OrderValidated`, `OrderCancelled` |
| Wave | 6 | `WaveCreated`, `WaveReleased`, `WaveCompleted` |
| Picking | 8 | `PickTaskCreated`, `PickCompleted`, `PickException` |
| Packing | 5 | `PackTaskCreated`, `PackCompleted` |
| Shipping | 7 | `ShipmentCreated`, `LabelGenerated`, `ShipmentConfirmed` |
| Inventory | 8 | `StockReserved`, `StockPicked`, `StockAdjusted` |
| Others | 12 | Various service-specific events |

### Usage

```go
import "github.com/wms-platform/shared/pkg/cloudevents"

event := cloudevents.NewOrderCreatedEvent(order)
// event.Type = "com.wms.order.created"
// event.Source = "order-service"
```

---

## Domain Package

Shared value objects, enums, and domain primitives.

### Value Objects

| Type | File | Description |
|------|------|-------------|
| `Priority` | `priority.go` | Order priority (same_day, next_day, standard) |
| `OrderStatus` | `status.go` | Order lifecycle status |
| `WaveType` | `wave_type.go` | Wave types (standard, priority, express) |
| `WaveStatus` | `wave_status.go` | Wave lifecycle status |
| `Location` | `location.go` | Warehouse location representation |
| `Carrier` | `carrier.go` | Shipping carrier information |
| `TrackingNumber` | `tracking_number.go` | Carrier tracking number |
| `Tote` | `tote.go` | Tote container information |

---

## Middleware Package

HTTP middleware components for Gin framework.

### Available Middleware

| Middleware | Purpose |
|------------|---------|
| `ErrorHandler` | Standardized error response formatting |
| `Validation` | Request validation with detailed errors |
| `Metrics` | Prometheus request metrics |
| `Correlation` | Request correlation ID propagation |
| `Tracing` | OpenTelemetry span creation |
| `CloudEvents` | CloudEvent header extraction |
| `TenantAuth` | Multi-tenant authentication |

### Usage

```go
import "github.com/wms-platform/shared/pkg/middleware"

router := gin.New()
router.Use(
    middleware.Correlation(),
    middleware.Tracing(tracer),
    middleware.Metrics(metricsRegistry),
    middleware.ErrorHandler(),
)
```

---

## Tenant Package

Multi-tenancy support for context propagation and isolation.

### Extracting Tenant Context

```go
import "github.com/wms-platform/shared/pkg/tenant"

// From HTTP context
ctx := tenant.FromGinContext(c)
tenantID := tenant.GetTenantID(ctx)
facilityID := tenant.GetFacilityID(ctx)
warehouseID := tenant.GetWarehouseID(ctx)

// Set in context
ctx = tenant.WithTenant(ctx, "TENANT-001", "FAC-001", "WH-001")
```

---

## Resilience Package

Circuit breakers and resilience patterns for service-to-service communication.

### Circuit Breaker

```go
import "github.com/wms-platform/shared/pkg/resilience"

cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
    MaxFailures:     5,
    Timeout:         30 * time.Second,
    HalfOpenTimeout: 10 * time.Second,
})

result, err := cb.Execute(func() (interface{}, error) {
    return httpClient.Get(url)
})
```

---

## Testing Package

Test utilities including testcontainers integration.

### MongoDB Testcontainer

```go
import "github.com/wms-platform/shared/pkg/testing"

func TestIntegration(t *testing.T) {
    ctx := context.Background()

    // Start MongoDB container
    mongoContainer, err := testing.StartMongoContainer(ctx)
    if err != nil {
        t.Fatal(err)
    }
    defer mongoContainer.Terminate(ctx)

    // Get connection string
    uri, _ := mongoContainer.ConnectionString(ctx)
    client, _ := mongo.Connect(ctx, options.Client().ApplyURI(uri))
}
```

### Kafka Testcontainer

```go
kafkaContainer, err := testing.StartKafkaContainer(ctx)
brokers, _ := kafkaContainer.Brokers(ctx)
```

---

## Tracing Package

OpenTelemetry integration for distributed tracing.

### Setup

```go
import "github.com/wms-platform/shared/pkg/tracing"

shutdown, err := tracing.InitTracer(tracing.Config{
    ServiceName:    "order-service",
    JaegerEndpoint: "http://jaeger:14268/api/traces",
    Environment:    "production",
})
defer shutdown(ctx)
```

### Creating Spans

```go
ctx, span := tracing.StartSpan(ctx, "ProcessOrder")
defer span.End()

span.SetAttributes(
    attribute.String("order.id", orderID),
    attribute.String("order.status", status),
)
```

---

## Import Paths

All packages are imported from:

```go
import "github.com/wms-platform/shared/pkg/{package-name}"
```

## Related Documentation

- [Architecture Overview](/architecture/overview)
- [Infrastructure - MongoDB](/infrastructure/mongodb)
- [Infrastructure - Kafka](/infrastructure/kafka)
- [Infrastructure - Observability](/infrastructure/observability)
