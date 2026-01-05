# Idempotency Package

Stripe-compliant idempotency implementation for REST APIs and message deduplication for Kafka consumers in the WMS platform.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
  - [REST API Usage](#rest-api-usage)
  - [Kafka Consumer Usage](#kafka-consumer-usage)
- [Architecture](#architecture)
  - [Components](#components)
  - [Data Models](#data-models)
  - [MongoDB Collections](#mongodb-collections)
- [Configuration](#configuration)
  - [REST API Configuration](#rest-api-configuration)
  - [Consumer Configuration](#consumer-configuration)
- [Usage Examples](#usage-examples)
  - [REST API Integration](#rest-api-integration)
  - [Kafka Consumer Integration](#kafka-consumer-integration)
  - [Multi-Phase Operations](#multi-phase-operations)
- [Error Handling](#error-handling)
  - [HTTP Status Codes](#http-status-codes)
  - [Error Scenarios](#error-scenarios)
- [Observability](#observability)
  - [Metrics](#metrics)
  - [Logging](#logging)
  - [Monitoring](#monitoring)
- [Migration Guide](#migration-guide)
- [Performance Considerations](#performance-considerations)
- [Troubleshooting](#troubleshooting)
- [Testing](#testing)

## Overview

This package provides two idempotency patterns for the WMS platform:

### 1. REST API Idempotency (Stripe Pattern)

Implements Stripe's idempotency pattern using the `Idempotency-Key` header. This ensures that mutating operations (POST, PUT, PATCH, DELETE) can be safely retried without duplicating side effects.

**Key features:**
- Request fingerprinting (SHA256) to detect parameter changes
- Response caching for 24 hours
- Concurrent request detection and handling
- Automatic cleanup via MongoDB TTL indexes
- Comprehensive error responses

**Use cases:**
- Preventing duplicate order creation
- Safe API retries on network failures
- Ensuring exactly-once semantics for critical operations

### 2. Kafka Message Deduplication

Provides exactly-once message processing for Kafka consumers using CloudEvent IDs.

**Key features:**
- CloudEvent.id-based deduplication
- Per-consumer-group tracking
- Automatic duplicate detection
- 24-hour retention with TTL cleanup

**Use cases:**
- Preventing duplicate event processing
- Ensuring exactly-once workflow execution
- Safe message redelivery handling

## Quick Start

### REST API Usage

**1. Initialize repositories:**

```go
import "github.com/wms-platform/shared/pkg/idempotency"

// After establishing MongoDB connection
idempotencyKeyRepo := idempotency.NewMongoKeyRepository(db)
```

**2. Initialize indexes (on service startup):**

```go
if err := idempotency.InitializeIndexes(ctx, db); err != nil {
    logger.Warn("Failed to initialize idempotency indexes", "error", err)
}
```

**3. Configure middleware:**

```go
middlewareConfig.IdempotencyConfig = &idempotency.Config{
    ServiceName:     "order-service",
    Repository:      idempotencyKeyRepo,
    RequireKey:      false, // Start with optional mode
    OnlyMutating:    true,
    MaxKeyLength:    255,
    LockTimeout:     5 * time.Minute,
    RetentionPeriod: 24 * time.Hour,
    MaxResponseSize: 1024 * 1024, // 1MB
}
```

**4. Client usage:**

```bash
curl -X POST https://api.wms.example.com/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"customerId": "123", "items": [...]}'
```

### Kafka Consumer Usage

**1. Initialize repository:**

```go
idempotencyMsgRepo := idempotency.NewMongoMessageRepository(db)
```

**2. Wrap your handler:**

```go
originalHandler := func(ctx context.Context, event *cloudevents.WMSCloudEvent) error {
    // Your business logic here
    return processOrder(ctx, event)
}

deduplicatedHandler := idempotency.DeduplicatingHandler(
    &idempotency.ConsumerConfig{
        ServiceName:     "order-service",
        Topic:           "orders.received",
        ConsumerGroup:   "order-processor",
        Repository:      idempotencyMsgRepo,
        RetentionPeriod: 24 * time.Hour,
    },
    originalHandler,
)

// Use deduplicatedHandler in your Kafka consumer
```

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                    REST API Flow                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Client Request                                             │
│  (with Idempotency-Key header)                             │
│         │                                                   │
│         ▼                                                   │
│  ┌──────────────────┐                                      │
│  │ Idempotency      │                                      │
│  │ Middleware       │                                      │
│  └────────┬─────────┘                                      │
│           │                                                 │
│           ├─── Validate Key                                │
│           ├─── Compute Fingerprint (SHA256)                │
│           ├─── Check MongoDB (KeyRepository)               │
│           │                                                 │
│           ▼                                                 │
│    ┌─────────────┐                                         │
│    │ Key exists? │                                         │
│    └──────┬──────┘                                         │
│           │                                                 │
│      ┌────┴────┐                                           │
│      │         │                                           │
│     YES       NO                                           │
│      │         │                                           │
│      ▼         ▼                                           │
│  ┌────────┐  ┌─────────────┐                              │
│  │Completed│ │ Acquire Lock│                              │
│  │   ?     │  └──────┬──────┘                              │
│  └────┬───┘         │                                      │
│       │             ▼                                      │
│      YES     ┌──────────────┐                              │
│       │      │ Process      │                              │
│       │      │ Request      │                              │
│       │      └──────┬───────┘                              │
│       │             │                                      │
│       │             ▼                                      │
│       │      ┌──────────────┐                              │
│       │      │ Store        │                              │
│       │      │ Response     │                              │
│       │      └──────┬───────┘                              │
│       │             │                                      │
│       └─────────────┴──► Return Response                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                 Kafka Consumer Flow                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Kafka Message                                              │
│  (with CloudEvent.id)                                       │
│         │                                                   │
│         ▼                                                   │
│  ┌──────────────────┐                                      │
│  │ Deduplicating    │                                      │
│  │ Handler          │                                      │
│  └────────┬─────────┘                                      │
│           │                                                 │
│           ├─── Check MongoDB (MessageRepository)           │
│           │                                                 │
│           ▼                                                 │
│    ┌─────────────┐                                         │
│    │ Processed?  │                                         │
│    └──────┬──────┘                                         │
│           │                                                 │
│      ┌────┴────┐                                           │
│      │         │                                           │
│     YES       NO                                           │
│      │         │                                           │
│      ▼         ▼                                           │
│   Skip    ┌─────────────┐                                  │
│ (return)  │ Process     │                                  │
│           │ Event       │                                  │
│           └──────┬──────┘                                  │
│                  │                                          │
│                  ▼                                          │
│           ┌──────────────┐                                 │
│           │ Mark as      │                                 │
│           │ Processed    │                                 │
│           └──────────────┘                                 │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Data Models

#### IdempotencyKey

Represents a unique idempotency key for REST API requests.

```go
type IdempotencyKey struct {
    ID                 primitive.ObjectID `bson:"_id,omitempty"`
    Key                string             `bson:"key"`
    ServiceID          string             `bson:"serviceId"`
    UserID             string             `bson:"userId,omitempty"`
    RequestPath        string             `bson:"requestPath"`
    RequestMethod      string             `bson:"requestMethod"`
    RequestFingerprint string             `bson:"requestFingerprint"`

    // Lock management
    LockedAt           *time.Time         `bson:"lockedAt,omitempty"`

    // Recovery support
    RecoveryPoint      string             `bson:"recoveryPoint,omitempty"`

    // Response caching
    ResponseCode       int                `bson:"responseCode,omitempty"`
    ResponseBody       []byte             `bson:"responseBody,omitempty"`
    ResponseHeaders    map[string]string  `bson:"responseHeaders,omitempty"`

    // Timestamps
    CreatedAt          time.Time          `bson:"createdAt"`
    CompletedAt        *time.Time         `bson:"completedAt,omitempty"`
    ExpiresAt          time.Time          `bson:"expiresAt"`
}
```

**Field descriptions:**
- `Key`: The idempotency key from the header (max 255 chars)
- `ServiceID`: Service name (e.g., "order-service")
- `UserID`: Optional user identifier for user-scoped keys
- `RequestFingerprint`: SHA256 hash of request body
- `LockedAt`: Timestamp when lock was acquired (for concurrent detection)
- `RecoveryPoint`: Current phase for multi-phase operations
- `ResponseCode`, `ResponseBody`, `ResponseHeaders`: Cached response
- `ExpiresAt`: TTL for automatic cleanup (24 hours)

#### ProcessedMessage

Tracks processed Kafka messages for deduplication.

```go
type ProcessedMessage struct {
    ID            primitive.ObjectID `bson:"_id,omitempty"`
    MessageID     string             `bson:"messageId"`
    Topic         string             `bson:"topic"`
    EventType     string             `bson:"eventType"`
    ConsumerGroup string             `bson:"consumerGroup"`
    ServiceID     string             `bson:"serviceId"`
    ProcessedAt   time.Time          `bson:"processedAt"`
    ExpiresAt     time.Time          `bson:"expiresAt"`

    // Optional correlation
    CorrelationID string             `bson:"correlationId,omitempty"`
    WorkflowID    string             `bson:"workflowId,omitempty"`
}
```

**Field descriptions:**
- `MessageID`: CloudEvent.id (unique message identifier)
- `Topic`: Kafka topic name
- `ConsumerGroup`: Consumer group for scoping
- `ProcessedAt`: When the message was successfully processed
- `ExpiresAt`: TTL for automatic cleanup (24 hours)

### MongoDB Collections

#### `idempotency_keys`

Stores idempotency keys for REST API requests.

**Indexes:**
1. **idx_service_key** (unique): `{serviceId: 1, key: 1}`
   - Ensures one key per service
   - Used for lock acquisition

2. **idx_ttl**: `{expiresAt: 1}` with `expireAfterSeconds: 0`
   - Automatic cleanup after 24 hours

3. **idx_locked** (sparse): `{lockedAt: 1}`
   - Helps query locked/stale keys

**Storage:** ~1KB per key, 24-hour retention

#### `processed_messages`

Tracks processed Kafka messages.

**Indexes:**
1. **idx_msg_topic_group** (unique): `{messageId: 1, topic: 1, consumerGroup: 1}`
   - Ensures exactly-once processing per consumer group
   - Prevents duplicate message processing

2. **idx_ttl**: `{expiresAt: 1}` with `expireAfterSeconds: 0`
   - Automatic cleanup after 24 hours

3. **idx_processed_at**: `{processedAt: 1}`
   - Query optimization for monitoring

**Storage:** ~500B per message, 24-hour retention

## Configuration

### REST API Configuration

```go
type Config struct {
    // Required
    ServiceName  string          // e.g., "order-service"
    Repository   KeyRepository   // MongoDB repository

    // Behavior
    RequireKey   bool            // true = key required, false = optional
    OnlyMutating bool            // true = skip GET/HEAD/OPTIONS

    // Optional features
    UserIDExtractor func(*gin.Context) string  // Extract user ID from request

    // Limits
    MaxKeyLength    int           // Default: 255 characters
    LockTimeout     time.Duration // Default: 5 minutes
    RetentionPeriod time.Duration // Default: 24 hours
    MaxResponseSize int           // Default: 1MB

    // Observability
    Metrics *Metrics              // Prometheus metrics (optional)
}
```

**Default configuration:**

```go
config := &idempotency.Config{
    ServiceName:     "my-service",
    Repository:      repo,
    RequireKey:      false,
    OnlyMutating:    true,
    MaxKeyLength:    255,
    LockTimeout:     5 * time.Minute,
    RetentionPeriod: 24 * time.Hour,
    MaxResponseSize: 1024 * 1024,
}
```

**Configuration options explained:**

- **RequireKey**: Start with `false` for backward compatibility, switch to `true` after client migration
- **OnlyMutating**: Set to `true` to skip GET/HEAD/OPTIONS requests
- **UserIDExtractor**: Optional function to extract user ID for user-scoped keys
- **LockTimeout**: How long before a stale lock is considered abandoned
- **RetentionPeriod**: How long responses are cached (Stripe default: 24 hours)
- **MaxResponseSize**: Maximum response body size to cache (prevents memory issues)

### Consumer Configuration

```go
type ConsumerConfig struct {
    ServiceName     string              // e.g., "order-service"
    Topic           string              // Kafka topic name
    ConsumerGroup   string              // Consumer group ID
    Repository      MessageRepository   // MongoDB repository
    RetentionPeriod time.Duration       // Default: 24 hours
    Metrics         *Metrics            // Prometheus metrics (optional)
}
```

## Usage Examples

### REST API Integration

**Complete example for a new service:**

```go
package main

import (
    "context"
    "log/slog"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/wms-platform/shared/pkg/idempotency"
    "github.com/wms-platform/shared/pkg/middleware"
    "go.mongodb.org/mongo-driver/mongo"
)

func main() {
    ctx := context.Background()
    logger := slog.Default()

    // 1. Connect to MongoDB
    mongoClient, err := connectMongo(ctx)
    if err != nil {
        logger.Error("Failed to connect to MongoDB", "error", err)
        return
    }
    db := mongoClient.Database("wms")

    // 2. Initialize idempotency indexes
    if err := idempotency.InitializeIndexes(ctx, db); err != nil {
        logger.Warn("Failed to initialize idempotency indexes", "error", err)
    } else {
        logger.Info("Idempotency indexes initialized")
    }

    // 3. Create repositories
    idempotencyKeyRepo := idempotency.NewMongoKeyRepository(db)
    logger.Info("Idempotency repository initialized")

    // 4. Initialize metrics (optional)
    idempotencyMetrics := idempotency.NewMetrics(nil)

    // 5. Configure middleware
    middlewareConfig := middleware.DefaultConfig("order-service", logger)
    middlewareConfig.IdempotencyConfig = &idempotency.Config{
        ServiceName:     "order-service",
        Repository:      idempotencyKeyRepo,
        RequireKey:      false, // Start optional
        OnlyMutating:    true,
        MaxKeyLength:    255,
        LockTimeout:     5 * time.Minute,
        RetentionPeriod: 24 * time.Hour,
        MaxResponseSize: 1024 * 1024,
        Metrics:         idempotencyMetrics,
    }

    // 6. Setup router with middleware
    router := gin.New()
    middleware.Setup(router, middlewareConfig)

    // 7. Define routes
    router.POST("/api/v1/orders", createOrder)

    // 8. Start server
    router.Run(":8080")
}

func createOrder(c *gin.Context) {
    // Your business logic here
    c.JSON(201, gin.H{"orderId": "12345"})
}
```

### Kafka Consumer Integration

**Complete example:**

```go
package main

import (
    "context"
    "log/slog"
    "time"

    "github.com/wms-platform/shared/pkg/cloudevents"
    "github.com/wms-platform/shared/pkg/idempotency"
    "go.mongodb.org/mongo-driver/mongo"
)

func main() {
    ctx := context.Background()
    logger := slog.Default()

    // 1. Connect to MongoDB
    mongoClient, err := connectMongo(ctx)
    if err != nil {
        logger.Error("Failed to connect to MongoDB", "error", err)
        return
    }
    db := mongoClient.Database("wms")

    // 2. Initialize message repository
    idempotencyMsgRepo := idempotency.NewMongoMessageRepository(db)
    if err := idempotencyMsgRepo.EnsureIndexes(ctx); err != nil {
        logger.Warn("Failed to initialize message indexes", "error", err)
    }

    // 3. Define your business logic handler
    orderHandler := func(ctx context.Context, event *cloudevents.WMSCloudEvent) error {
        logger.Info("Processing order", "eventId", event.ID)

        // Your business logic here
        return processOrder(ctx, event)
    }

    // 4. Wrap with deduplication
    deduplicatedHandler := idempotency.DeduplicatingHandler(
        &idempotency.ConsumerConfig{
            ServiceName:     "order-service",
            Topic:           "orders.received",
            ConsumerGroup:   "order-processor",
            Repository:      idempotencyMsgRepo,
            RetentionPeriod: 24 * time.Hour,
        },
        orderHandler,
    )

    // 5. Use in your Kafka consumer
    consumer := NewKafkaConsumer(deduplicatedHandler)
    consumer.Start()
}

func processOrder(ctx context.Context, event *cloudevents.WMSCloudEvent) error {
    // Business logic implementation
    return nil
}
```

### Multi-Phase Operations

For complex operations with multiple steps, use the phase manager:

```go
func createOrderWithPhases(c *gin.Context) error {
    // Get phase manager from context (set by middleware)
    phaseManager, _ := c.Get("idempotency.phaseManager")
    pm := phaseManager.(*idempotency.PhaseManager)

    // Phase 1: Validate
    if err := validateOrder(c); err != nil {
        return err
    }
    pm.Checkpoint(c.Request.Context(), "validated")

    // Phase 2: Reserve inventory
    if err := reserveInventory(c); err != nil {
        return err
    }
    pm.Checkpoint(c.Request.Context(), "inventory_reserved")

    // Phase 3: Create order
    if err := createOrder(c); err != nil {
        return err
    }
    pm.Checkpoint(c.Request.Context(), "order_created")

    // Phase 4: Send notification
    if err := sendNotification(c); err != nil {
        return err
    }
    pm.Checkpoint(c.Request.Context(), "completed")

    return nil
}
```

**On retry, the system will resume from the last checkpoint.**

## Error Handling

### HTTP Status Codes

The middleware returns specific HTTP status codes for different scenarios:

| Status Code | Scenario | Description |
|-------------|----------|-------------|
| **200 OK** | Cached response | Previously completed request, returning cached response |
| **400 Bad Request** | Invalid/missing key | Key format invalid or missing (when required) |
| **409 Conflict** | Concurrent request | Another request with same key is being processed |
| **422 Unprocessable Entity** | Parameter mismatch | Request parameters differ from original |
| **503 Service Unavailable** | Storage failure | MongoDB temporarily unavailable |

### Error Scenarios

#### 1. Missing Idempotency-Key (Required Mode)

**Request:**
```bash
POST /api/v1/orders
Content-Type: application/json

{"customerId": "123"}
```

**Response:**
```json
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "Idempotency-Key header is required for this operation",
  "code": "IDEMPOTENCY_KEY_REQUIRED"
}
```

#### 2. Invalid Key Format

**Request:**
```bash
POST /api/v1/orders
Idempotency-Key: invalid key with spaces!
```

**Response:**
```json
HTTP/1.1 400 Bad Request

{
  "error": "Invalid idempotency key: key contains invalid characters",
  "code": "IDEMPOTENCY_KEY_INVALID"
}
```

**Valid formats:**
- Alphanumeric: `abc123`
- With hyphens: `550e8400-e29b-41d4-a716-446655440000`
- With underscores: `order_2024_01_03_abc`

#### 3. Concurrent Request

**Scenario:** Two requests with same key arrive simultaneously.

**First request:**
```bash
POST /api/v1/orders
Idempotency-Key: abc123
```
Response: Processing...

**Second request (concurrent):**
```bash
POST /api/v1/orders
Idempotency-Key: abc123
```

**Response:**
```json
HTTP/1.1 409 Conflict

{
  "error": "A request with this idempotency key is currently being processed",
  "code": "IDEMPOTENCY_CONCURRENT_REQUEST"
}
```

**Client action:** Implement exponential backoff and retry.

#### 4. Parameter Mismatch

**Original request:**
```bash
POST /api/v1/orders
Idempotency-Key: abc123
Content-Type: application/json

{"customerId": "123", "amount": 100}
```

**Retry with different parameters:**
```bash
POST /api/v1/orders
Idempotency-Key: abc123
Content-Type: application/json

{"customerId": "123", "amount": 200}  # Different amount!
```

**Response:**
```json
HTTP/1.1 422 Unprocessable Entity

{
  "error": "Request parameters differ from original request with this idempotency key",
  "code": "IDEMPOTENCY_PARAMETER_MISMATCH"
}
```

**Client action:** Use a new idempotency key if request parameters need to change.

#### 5. Storage Unavailable

**Scenario:** MongoDB connection failure.

**Response:**
```json
HTTP/1.1 503 Service Unavailable

{
  "error": "Idempotency storage temporarily unavailable",
  "code": "IDEMPOTENCY_STORAGE_UNAVAILABLE"
}
```

**Client action:** Retry with exponential backoff.

## Observability

### Metrics

The package exports 9 Prometheus metrics for monitoring:

#### REST API Metrics

**1. idempotency_hits_total**
- Type: Counter
- Labels: `service`, `endpoint`, `method`
- Description: Cached responses returned (successful idempotency)

**2. idempotency_misses_total**
- Type: Counter
- Labels: `service`, `endpoint`, `method`
- Description: New requests processed (first time seeing key)

**3. idempotency_parameter_mismatches_total**
- Type: Counter
- Labels: `service`, `endpoint`, `method`
- Description: 422 errors due to parameter changes

**4. idempotency_concurrent_collisions_total**
- Type: Counter
- Labels: `service`, `endpoint`, `method`
- Description: 409 errors due to concurrent requests

**5. idempotency_lock_acquisition_duration_seconds**
- Type: Histogram
- Labels: `service`, `endpoint`
- Buckets: 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0
- Description: Time to acquire lock in MongoDB

**6. idempotency_storage_errors_total**
- Type: Counter
- Labels: `service`, `operation`
- Description: MongoDB errors (503 responses)

#### Kafka Consumer Metrics

**7. message_deduplication_hits_total**
- Type: Counter
- Labels: `service`, `topic`, `consumer_group`, `event_type`
- Description: Duplicate messages skipped

**8. message_deduplication_misses_total**
- Type: Counter
- Labels: `service`, `topic`, `consumer_group`, `event_type`
- Description: New messages processed

**9. message_deduplication_errors_total**
- Type: Counter
- Labels: `service`, `topic`, `consumer_group`, `operation`
- Description: Deduplication check failures

### Logging

The package uses structured logging (slog) for all operations:

**Example logs:**

```
INFO Idempotency hit service=order-service endpoint=/api/v1/orders method=POST key=abc123
INFO Idempotency miss service=order-service endpoint=/api/v1/orders method=POST key=def456
WARN Idempotency parameter mismatch service=order-service key=abc123
WARN Concurrent request detected service=order-service key=abc123
ERROR Failed to acquire lock service=order-service error="connection timeout"
INFO Duplicate message skipped service=order-service topic=orders.received messageId=evt-123
```

### Monitoring

**Recommended alerts:**

```yaml
# High parameter mismatch rate
- alert: HighIdempotencyParameterMismatches
  expr: rate(idempotency_parameter_mismatches_total[5m]) > 10
  for: 5m
  annotations:
    summary: "High rate of idempotency parameter mismatches"
    description: "{{ $value }} mismatches/sec on {{ $labels.service }}"

# High storage error rate
- alert: IdempotencyStorageErrors
  expr: rate(idempotency_storage_errors_total[5m]) > 1
  for: 5m
  annotations:
    summary: "Idempotency storage errors detected"
    description: "MongoDB errors on {{ $labels.service }}"

# Slow lock acquisition
- alert: SlowIdempotencyLockAcquisition
  expr: histogram_quantile(0.99, rate(idempotency_lock_acquisition_duration_seconds_bucket[5m])) > 0.5
  for: 5m
  annotations:
    summary: "Slow idempotency lock acquisition"
    description: "P99 lock time: {{ $value }}s on {{ $labels.service }}"
```

**Grafana dashboard queries:**

```promql
# Idempotency hit rate by service
sum(rate(idempotency_hits_total[5m])) by (service)

# Cache effectiveness (hit ratio)
sum(rate(idempotency_hits_total[5m])) /
  (sum(rate(idempotency_hits_total[5m])) + sum(rate(idempotency_misses_total[5m])))

# Message deduplication rate
sum(rate(message_deduplication_hits_total[5m])) by (topic, consumer_group)

# Lock acquisition latency (P50, P95, P99)
histogram_quantile(0.50, rate(idempotency_lock_acquisition_duration_seconds_bucket[5m]))
histogram_quantile(0.95, rate(idempotency_lock_acquisition_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(idempotency_lock_acquisition_duration_seconds_bucket[5m]))
```

## Migration Guide

### Phase 1: Deploy with Optional Mode (Week 1-2)

**Goal:** Roll out idempotency infrastructure without breaking existing clients.

**1. Deploy services with RequireKey=false:**

```go
middlewareConfig.IdempotencyConfig = &idempotency.Config{
    ServiceName:     "order-service",
    Repository:      idempotencyKeyRepo,
    RequireKey:      false,  // Optional mode
    OnlyMutating:    true,
    // ... other config
}
```

**2. Monitor metrics:**
- Check `idempotency_misses_total` to see adoption
- Ensure no `idempotency_storage_errors_total`

**3. Update documentation:**
- API docs with `Idempotency-Key` header
- Client integration guides

### Phase 2: Client Migration (Week 3-6)

**Goal:** Update all API clients to send `Idempotency-Key`.

**1. Identify all API clients:**
- Frontend applications
- Mobile apps
- External integrations
- Internal service-to-service calls

**2. Update client code:**

**JavaScript/TypeScript:**
```javascript
import { v4 as uuidv4 } from 'uuid';

async function createOrder(orderData) {
  const response = await fetch('/api/v1/orders', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': uuidv4(),  // Add this
    },
    body: JSON.stringify(orderData),
  });
  return response.json();
}
```

**Go:**
```go
import "github.com/google/uuid"

func createOrder(client *http.Client, order Order) error {
    req, _ := http.NewRequest("POST", "/api/v1/orders", orderBody)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Idempotency-Key", uuid.New().String())  // Add this

    resp, err := client.Do(req)
    return err
}
```

**Python:**
```python
import uuid
import requests

def create_order(order_data):
    response = requests.post(
        '/api/v1/orders',
        json=order_data,
        headers={
            'Idempotency-Key': str(uuid.uuid4())  # Add this
        }
    )
    return response.json()
```

**3. Monitor adoption:**
```promql
# Percentage of requests with idempotency keys
sum(rate(idempotency_hits_total[5m]) + rate(idempotency_misses_total[5m])) /
  sum(rate(http_requests_total{method!="GET"}[5m]))
```

**4. Gradual rollout:**
- Start with non-production environments
- Roll out to 10% of production traffic
- Increase to 50%, then 100%
- Monitor error rates at each step

### Phase 3: Enforce Required Mode (Week 7+)

**Goal:** Make `Idempotency-Key` required for all mutating operations.

**1. Verify 100% adoption:**
- Check metrics show all clients sending keys
- Confirm no major clients are missing

**2. Update service configuration:**

```go
middlewareConfig.IdempotencyConfig = &idempotency.Config{
    ServiceName:     "order-service",
    Repository:      idempotencyKeyRepo,
    RequireKey:      true,  // Now required!
    OnlyMutating:    true,
    // ... other config
}
```

**3. Deploy with canary:**
- Deploy to 10% of instances
- Monitor for 400 errors (missing keys)
- If error rate < 0.1%, proceed
- Roll out to 100%

**4. Update API contracts:**
- Mark `Idempotency-Key` as required in OpenAPI specs
- Update client SDKs

### Rollback Plan

If issues arise during migration:

**1. Revert to optional mode:**
```bash
kubectl set env deployment/order-service IDEMPOTENCY_REQUIRE_KEY=false
```

**2. Disable idempotency entirely (emergency only):**
```go
// Remove IdempotencyConfig from middleware
middlewareConfig.IdempotencyConfig = nil
```

## Performance Considerations

### MongoDB Performance

**Index usage:**
- All queries use indexes (no collection scans)
- Lock acquisition: ~5-10ms (p99)
- Cached response lookup: ~2-5ms (p99)

**Optimization tips:**
1. **Use MongoDB replica sets** for high availability
2. **Monitor index usage:**
   ```javascript
   db.idempotency_keys.aggregate([{ $indexStats: {} }])
   ```
3. **Monitor collection size:**
   ```javascript
   db.idempotency_keys.stats()
   ```
4. **Tune TTL cleanup:**
   - Default runs every 60 seconds
   - Adjust with `setParameter: ttlMonitorSleepSecs`

### Response Body Caching

**Memory considerations:**
- Default max response size: 1MB
- Average response size: ~5KB
- With 10,000 keys: ~50MB MongoDB storage

**Large response handling:**
- Responses > MaxResponseSize are not cached
- Request still idempotent (returns same result)
- Consider increasing MaxResponseSize for specific services

**Optimization:**
```go
// For services with large responses
config.MaxResponseSize = 5 * 1024 * 1024  // 5MB
```

### Lock Contention

**Scenario:** Many clients retry with same key simultaneously.

**Mitigation strategies:**

1. **Client-side exponential backoff:**
```javascript
async function retryWithBackoff(fn, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (error.status === 409 && i < maxRetries - 1) {
        await sleep(Math.pow(2, i) * 100);  // 100ms, 200ms, 400ms
        continue;
      }
      throw error;
    }
  }
}
```

2. **Server-side retry with polling:**
```go
// Instead of immediately returning 409, wait briefly
if existingKey.LockedAt != nil && existingKey.CompletedAt == nil {
    // Wait 100ms and check again
    time.Sleep(100 * time.Millisecond)
    // Re-fetch key
    existingKey, err = repo.GetByID(ctx, existingKey.ID.Hex())
    if err == nil && existingKey.CompletedAt != nil {
        // Completed while we waited!
        return cached response
    }
    return 409 Conflict
}
```

### Kafka Consumer Performance

**Deduplication overhead:**
- MongoDB lookup: ~2-5ms per message
- Minimal impact on throughput
- Enables safe exactly-once processing

**Batch optimization:**
For high-throughput topics, consider batch deduplication checks:

```go
// Check multiple messages in one query
func (r *MongoMessageRepository) AreProcessed(ctx context.Context, messageIDs []string, topic, consumerGroup string) (map[string]bool, error) {
    filter := bson.M{
        "messageId":     bson.M{"$in": messageIDs},
        "topic":         topic,
        "consumerGroup": consumerGroup,
    }

    cursor, err := r.collection.Find(ctx, filter)
    // ... build map of processed IDs
}
```

## Troubleshooting

### Common Issues

#### Issue 1: High Parameter Mismatch Rate

**Symptom:**
```
idempotency_parameter_mismatches_total increasing rapidly
```

**Possible causes:**
1. Client generating new key for retries instead of reusing
2. Request body contains dynamic fields (timestamps, nonces)

**Solution:**
```javascript
// BAD: New key on each retry
function createOrder(data) {
  return fetch('/api/v1/orders', {
    headers: { 'Idempotency-Key': uuidv4() }  // Wrong!
  });
}

// GOOD: Store and reuse key
function createOrder(data) {
  if (!data.idempotencyKey) {
    data.idempotencyKey = uuidv4();
  }
  return fetch('/api/v1/orders', {
    headers: { 'Idempotency-Key': data.idempotencyKey }
  });
}
```

#### Issue 2: Stale Locks

**Symptom:**
```
409 Conflict errors for requests that should have completed
```

**Investigation:**
```javascript
// Find stale locks (locked > 5 minutes ago, not completed)
db.idempotency_keys.find({
  lockedAt: { $lt: new Date(Date.now() - 5 * 60 * 1000) },
  completedAt: null
})
```

**Solution:**
Locks are automatically released after `LockTimeout` (default 5 minutes). If seeing many stale locks, check for:
1. Long-running operations (increase LockTimeout)
2. Application crashes without cleanup
3. MongoDB connection issues

#### Issue 3: TTL Index Not Cleaning Up

**Symptom:**
```
idempotency_keys collection growing indefinitely
```

**Investigation:**
```javascript
// Check if TTL index exists
db.idempotency_keys.getIndexes()

// Check TTL monitor
db.serverStatus().metrics.ttl
```

**Solution:**
```javascript
// Manually trigger cleanup (testing only)
db.idempotency_keys.deleteMany({
  expiresAt: { $lt: new Date() }
})

// Recreate TTL index if missing
db.idempotency_keys.createIndex(
  { expiresAt: 1 },
  { expireAfterSeconds: 0 }
)
```

#### Issue 4: Duplicate Messages Still Processed

**Symptom:**
```
message_deduplication_hits_total not incrementing
CloudEvent.id duplicates being processed
```

**Investigation:**
1. Check if CloudEvent.id is actually unique:
```javascript
db.processed_messages.aggregate([
  { $group: { _id: "$messageId", count: { $sum: 1 } } },
  { $match: { count: { $gt: 1 } } }
])
```

2. Check consumer group configuration:
```go
// Ensure consumer group matches across instances
config := &idempotency.ConsumerConfig{
    ConsumerGroup: "order-processor",  // Must be consistent!
}
```

**Solution:**
- Ensure CloudEvent.id is globally unique (use UUID v4)
- Verify consumer group configuration is consistent
- Check MongoDB indexes are created

#### Issue 5: 503 Storage Errors

**Symptom:**
```
idempotency_storage_errors_total increasing
HTTP 503 responses
```

**Investigation:**
```bash
# Check MongoDB connection
kubectl logs deployment/order-service | grep -i mongodb

# Check MongoDB status
kubectl exec -it mongodb-0 -- mongo --eval "db.serverStatus()"
```

**Solutions:**
1. **Connection pool exhausted:**
```go
// Increase MongoDB connection pool
clientOptions := options.Client().
    SetMaxPoolSize(100).  // Increase from default 100
    SetMinPoolSize(10)
```

2. **Network issues:**
- Check MongoDB pod health
- Verify network policies
- Check DNS resolution

3. **MongoDB overloaded:**
- Scale MongoDB replica set
- Add read replicas
- Check slow query log

### Debug Mode

Enable verbose logging for troubleshooting:

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,  // Enable debug logs
}))

// Logs will include:
// - Lock acquisition details
// - Fingerprint comparisons
// - MongoDB query details
```

### Manual Key Inspection

**Check key details:**
```javascript
db.idempotency_keys.findOne({ key: "abc123", serviceId: "order-service" })
```

**Find all keys for a user:**
```javascript
db.idempotency_keys.find({ userId: "user-123" }).sort({ createdAt: -1 })
```

**Find concurrent collisions:**
```javascript
db.idempotency_keys.find({
  lockedAt: { $exists: true },
  completedAt: null,
  createdAt: { $gt: new Date(Date.now() - 60000) }  // Last minute
})
```

## Testing

### Unit Testing

The package includes comprehensive unit tests:

```bash
cd shared/pkg/idempotency
go test -v
```

**Test coverage:**
- Validator tests: Key validation, fingerprinting
- Middleware tests: All HTTP scenarios (400, 409, 422, 503)
- Repository tests: MongoDB operations

### Integration Testing

**Test idempotency end-to-end:**

```go
func TestIdempotencyIntegration(t *testing.T) {
    // 1. Make first request
    req1 := httptest.NewRequest("POST", "/api/v1/orders", body)
    req1.Header.Set("Idempotency-Key", "test-key-123")
    w1 := httptest.NewRecorder()
    router.ServeHTTP(w1, req1)

    assert.Equal(t, 201, w1.Code)
    response1 := w1.Body.String()

    // 2. Make retry with same key
    req2 := httptest.NewRequest("POST", "/api/v1/orders", body)
    req2.Header.Set("Idempotency-Key", "test-key-123")
    w2 := httptest.NewRecorder()
    router.ServeHTTP(w2, req2)

    // 3. Verify cached response
    assert.Equal(t, 200, w2.Code)  // Cached
    assert.Equal(t, response1, w2.Body.String())
}
```

### Load Testing

**Test concurrent requests:**

```bash
# Using Apache Bench
ab -n 1000 -c 100 \
   -H "Idempotency-Key: test-123" \
   -H "Content-Type: application/json" \
   -p order.json \
   http://localhost:8080/api/v1/orders

# Expected results:
# - First request: 201 Created
# - Remaining 999: 200 OK (cached) or 409 Conflict
```

**Test parameter mismatch:**

```bash
# Send 100 requests with same key but different bodies
for i in {1..100}; do
  curl -X POST http://localhost:8080/api/v1/orders \
    -H "Idempotency-Key: test-456" \
    -H "Content-Type: application/json" \
    -d "{\"amount\": $i}" &
done

# Expected: 1 success (201), 99 parameter mismatches (422)
```

### Kafka Consumer Testing

**Test message deduplication:**

```go
func TestMessageDeduplication(t *testing.T) {
    event := &cloudevents.WMSCloudEvent{
        ID:   "evt-123",
        Type: "OrderReceived",
        Data: orderData,
    }

    // Process first time
    err1 := deduplicatedHandler(ctx, event)
    assert.NoError(t, err1)

    // Process duplicate
    err2 := deduplicatedHandler(ctx, event)
    assert.NoError(t, err2)  // No error, silently skipped

    // Verify only processed once
    assert.Equal(t, 1, processedCount)
}
```

---

## Additional Resources

- **Stripe Idempotency Guide**: https://stripe.com/docs/api/idempotent_requests
- **CloudEvents Specification**: https://cloudevents.io/
- **MongoDB TTL Indexes**: https://docs.mongodb.com/manual/core/index-ttl/
- **Prometheus Best Practices**: https://prometheus.io/docs/practices/naming/

## Support

For questions or issues:
- File a GitHub issue in the `wms-platform` repository
- Contact the platform team on Slack: `#wms-platform-support`
- Refer to the integration guide: `/wms-platform/docs/guides/idempotency-integration.md`

## License

Copyright © 2026 WMS Platform Team. All rights reserved.
