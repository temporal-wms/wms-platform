# Idempotency Integration Guide

Complete guide for integrating idempotency into WMS services and client applications.

## Table of Contents

- [For Service Developers](#for-service-developers)
  - [Adding Idempotency to a New Service](#adding-idempotency-to-a-new-service)
  - [Adding Kafka Consumer Deduplication](#adding-kafka-consumer-deduplication)
  - [Multi-Phase Operations](#multi-phase-operations)
- [For Client Developers](#for-client-developers)
  - [REST API Clients](#rest-api-clients)
  - [Best Practices](#best-practices)
  - [Error Handling](#error-handling)
- [Testing](#testing)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

---

## For Service Developers

### Adding Idempotency to a New Service

Follow these steps to add idempotency support to a new or existing WMS service.

#### Step 1: Initialize Repositories

After establishing MongoDB connection, create the idempotency repositories:

```go
package main

import (
    "context"
    "log/slog"

    "github.com/wms-platform/shared/pkg/idempotency"
    "go.mongodb.org/mongo-driver/mongo"
)

func initializeIdempotency(ctx context.Context, db *mongo.Database, logger *slog.Logger) (*idempotency.MongoKeyRepository, *idempotency.MongoMessageRepository, error) {
    // Create repositories
    idempotencyKeyRepo := idempotency.NewMongoKeyRepository(db)
    idempotencyMsgRepo := idempotency.NewMongoMessageRepository(db)

    // Initialize indexes (creates MongoDB indexes if they don't exist)
    if err := idempotency.InitializeIndexes(ctx, db); err != nil {
        logger.Warn("Failed to initialize idempotency indexes", "error", err)
        return nil, nil, err
    }

    logger.Info("Idempotency repositories and indexes initialized")
    return idempotencyKeyRepo, idempotencyMsgRepo, nil
}
```

#### Step 2: Initialize Metrics

Create Prometheus metrics for observability:

```go
import "github.com/wms-platform/shared/pkg/idempotency"

func setupMetrics() *idempotency.Metrics {
    // Use default Prometheus registry
    idempotencyMetrics := idempotency.NewMetrics(nil)
    return idempotencyMetrics
}

// Or with custom registry
func setupMetricsWithRegistry(registry prometheus.Registerer) *idempotency.Metrics {
    idempotencyMetrics := idempotency.NewMetrics(registry)
    return idempotencyMetrics
}
```

#### Step 3: Configure Middleware

Add idempotency configuration to your middleware setup:

```go
import (
    "time"

    "github.com/wms-platform/shared/pkg/middleware"
    "github.com/wms-platform/shared/pkg/idempotency"
)

func setupMiddleware(serviceName string, logger *slog.Logger, idempotencyKeyRepo *idempotency.MongoKeyRepository, metrics *idempotency.Metrics) *middleware.Config {
    middlewareConfig := middleware.DefaultConfig(serviceName, logger)

    // Configure idempotency
    middlewareConfig.IdempotencyConfig = &idempotency.Config{
        ServiceName:     serviceName,
        Repository:      idempotencyKeyRepo,
        RequireKey:      false,  // Start with optional mode for backward compatibility
        OnlyMutating:    true,   // Only apply to POST/PUT/PATCH/DELETE
        MaxKeyLength:    255,
        LockTimeout:     5 * time.Minute,
        RetentionPeriod: 24 * time.Hour,
        MaxResponseSize: 1024 * 1024, // 1MB
        Metrics:         metrics,
    }

    return middlewareConfig
}
```

#### Step 4: Apply Middleware

Apply the middleware to your Gin router:

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/wms-platform/shared/pkg/middleware"
)

func setupRouter(middlewareConfig *middleware.Config) *gin.Engine {
    router := gin.New()

    // Apply all middleware (including idempotency)
    middleware.Setup(router, middlewareConfig)

    // Define routes
    router.POST("/api/v1/orders", createOrder)
    router.PUT("/api/v1/orders/:id", updateOrder)
    router.DELETE("/api/v1/orders/:id", deleteOrder)
    router.GET("/api/v1/orders", listOrders)  // Idempotency skipped for GET

    return router
}
```

#### Complete Example

Here's a complete `main.go` integrating all steps:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/wms-platform/shared/pkg/idempotency"
    "github.com/wms-platform/shared/pkg/middleware"
    "github.com/wms-platform/shared/pkg/mongodb"
)

func main() {
    ctx := context.Background()
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    serviceName := "my-service"

    // 1. Connect to MongoDB
    mongoClient, err := mongodb.NewProductionClient(ctx, os.Getenv("MONGODB_URI"), "wms")
    if err != nil {
        logger.Error("Failed to connect to MongoDB", "error", err)
        os.Exit(1)
    }
    defer mongoClient.Disconnect(ctx)

    db := mongoClient.Database("wms")

    // 2. Initialize idempotency
    idempotencyKeyRepo, idempotencyMsgRepo, err := initializeIdempotency(ctx, db, logger)
    if err != nil {
        logger.Error("Failed to initialize idempotency", "error", err)
        os.Exit(1)
    }

    // 3. Setup metrics
    idempotencyMetrics := setupMetrics()

    // 4. Configure middleware
    middlewareConfig := setupMiddleware(serviceName, logger, idempotencyKeyRepo, idempotencyMetrics)

    // 5. Setup router
    router := setupRouter(middlewareConfig)

    // 6. Start server
    logger.Info("Starting server", "port", 8080)
    if err := router.Run(":8080"); err != nil {
        logger.Error("Server failed", "error", err)
        os.Exit(1)
    }
}

func createOrder(c *gin.Context) {
    // Your business logic here
    c.JSON(201, gin.H{"orderId": "ORD-12345"})
}
```

---

### Adding Kafka Consumer Deduplication

For services that consume Kafka events, wrap your handlers with deduplication logic.

#### Step 1: Wrap Event Handler

```go
import (
    "context"

    "github.com/wms-platform/shared/pkg/cloudevents"
    "github.com/wms-platform/shared/pkg/idempotency"
)

func setupKafkaConsumer(idempotencyMsgRepo *idempotency.MongoMessageRepository) {
    // Your original business logic handler
    originalHandler := func(ctx context.Context, event *cloudevents.WMSCloudEvent) error {
        logger.Info("Processing order", "eventId", event.ID)
        return processOrder(ctx, event)
    }

    // Wrap with deduplication
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
    consumer := NewKafkaConsumer(deduplicatedHandler)
    consumer.Start()
}

func processOrder(ctx context.Context, event *cloudevents.WMSCloudEvent) error {
    // Business logic implementation
    // This will only be called once per unique event.ID
    return nil
}
```

#### Step 2: Ensure CloudEvent IDs are Unique

Make sure your event publishers generate unique IDs:

```go
import (
    "github.com/google/uuid"
    "github.com/wms-platform/shared/pkg/cloudevents"
)

func publishOrderEvent(producer *kafka.Producer) error {
    event := &cloudevents.WMSCloudEvent{
        ID:              uuid.New().String(),  // ← Unique ID!
        Type:            "com.wms.OrderReceived",
        Source:          "/order-service",
        Time:            time.Now().UTC(),
        DataContentType: "application/json",
        Data:            orderData,
    }

    return producer.PublishEvent(ctx, "orders.received", event)
}
```

---

### Multi-Phase Operations

For complex operations with multiple steps, use phase checkpoints for recovery:

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/wms-platform/shared/pkg/idempotency"
)

func createOrderWithPhases(c *gin.Context) error {
    // Get phase manager from context (set by middleware)
    phaseManager, exists := c.Get("idempotency.phaseManager")
    if !exists {
        // No idempotency key provided, proceed normally
        return createOrderNormal(c)
    }

    pm := phaseManager.(*idempotency.PhaseManager)
    ctx := c.Request.Context()

    // Phase 1: Validate order
    if err := validateOrder(c); err != nil {
        return err
    }
    pm.Checkpoint(ctx, "validated")

    // Phase 2: Reserve inventory
    if err := reserveInventory(c); err != nil {
        return err
    }
    pm.Checkpoint(ctx, "inventory_reserved")

    // Phase 3: Create order record
    if err := createOrderRecord(c); err != nil {
        return err
    }
    pm.Checkpoint(ctx, "order_created")

    // Phase 4: Send notification
    if err := sendNotification(c); err != nil {
        return err
    }
    pm.Checkpoint(ctx, "notification_sent")

    c.JSON(201, gin.H{"orderId": "ORD-12345"})
    return nil
}
```

**How it works:**
- On retry with same idempotency key, the system resumes from the last checkpoint
- Each checkpoint is atomic - phases are not re-executed on retry
- Useful for operations that can't be fully rolled back

---

## For Client Developers

### REST API Clients

#### JavaScript/TypeScript

**Using fetch API:**

```javascript
import { v4 as uuidv4 } from 'uuid';

class WMSClient {
  constructor(baseURL) {
    this.baseURL = baseURL;
  }

  async createOrder(orderData, options = {}) {
    // Generate idempotency key (or use provided one for retry)
    const idempotencyKey = options.idempotencyKey || uuidv4();

    const response = await fetch(`${this.baseURL}/api/v1/orders`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': idempotencyKey,
      },
      body: JSON.stringify(orderData),
    });

    if (response.ok) {
      return await response.json();
    }

    // Handle idempotency errors
    if (response.status === 409) {
      // Concurrent request - retry with exponential backoff
      throw new ConcurrentRequestError('Request is being processed', idempotencyKey);
    }

    if (response.status === 422) {
      // Parameter mismatch - don't retry with same key
      throw new ParameterMismatchError('Parameters changed', idempotencyKey);
    }

    throw new Error(`Request failed: ${response.status}`);
  }

  // Retry with exponential backoff
  async createOrderWithRetry(orderData, maxRetries = 3) {
    const idempotencyKey = uuidv4();

    for (let attempt = 0; attempt < maxRetries; attempt++) {
      try {
        return await this.createOrder(orderData, { idempotencyKey });
      } catch (error) {
        if (error instanceof ConcurrentRequestError && attempt < maxRetries - 1) {
          // Exponential backoff: 100ms, 200ms, 400ms
          const delay = Math.pow(2, attempt) * 100;
          await sleep(delay);
          continue;
        }
        throw error;
      }
    }
  }
}

// Usage
const client = new WMSClient('https://api.wms.example.com');

const order = await client.createOrderWithRetry({
  customerId: 'CUST-001',
  priority: 'standard',
  items: [
    { sku: 'SKU-001', quantity: 2, price: 29.99 }
  ],
  shippingAddress: {
    street: '123 Main St',
    city: 'New York',
    state: 'NY',
    zipCode: '10001',
    country: 'US'
  }
});

console.log('Order created:', order.orderId);
```

#### Python

```python
import uuid
import requests
import time
from typing import Dict, Any, Optional

class WMSClient:
    def __init__(self, base_url: str):
        self.base_url = base_url

    def create_order(
        self,
        order_data: Dict[str, Any],
        idempotency_key: Optional[str] = None,
        max_retries: int = 3
    ) -> Dict[str, Any]:
        """Create an order with automatic retry on 409 Conflict."""

        if idempotency_key is None:
            idempotency_key = str(uuid.uuid4())

        for attempt in range(max_retries):
            try:
                response = requests.post(
                    f'{self.base_url}/api/v1/orders',
                    json=order_data,
                    headers={
                        'Content-Type': 'application/json',
                        'Idempotency-Key': idempotency_key
                    }
                )

                if response.status_code in [200, 201]:
                    return response.json()

                if response.status_code == 409:
                    # Concurrent request - retry with backoff
                    if attempt < max_retries - 1:
                        time.sleep((2 ** attempt) * 0.1)  # 100ms, 200ms, 400ms
                        continue
                    raise ConcurrentRequestError(response.json())

                if response.status_code == 422:
                    raise ParameterMismatchError(response.json())

                response.raise_for_status()

            except requests.RequestException as e:
                if attempt < max_retries - 1:
                    time.sleep((2 ** attempt) * 0.1)
                    continue
                raise

        raise MaxRetriesExceededError(f'Failed after {max_retries} attempts')

# Usage
client = WMSClient('https://api.wms.example.com')

order = client.create_order({
    'customerId': 'CUST-001',
    'priority': 'standard',
    'items': [
        {'sku': 'SKU-001', 'quantity': 2, 'price': 29.99}
    ],
    'shippingAddress': {
        'street': '123 Main St',
        'city': 'New York',
        'state': 'NY',
        'zipCode': '10001',
        'country': 'US'
    }
})

print(f"Order created: {order['orderId']}")
```

#### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/google/uuid"
)

type WMSClient struct {
    BaseURL    string
    HTTPClient *http.Client
}

func NewWMSClient(baseURL string) *WMSClient {
    return &WMSClient{
        BaseURL: baseURL,
        HTTPClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *WMSClient) CreateOrder(orderData map[string]interface{}) (map[string]interface{}, error) {
    return c.CreateOrderWithRetry(orderData, 3)
}

func (c *WMSClient) CreateOrderWithRetry(orderData map[string]interface{}, maxRetries int) (map[string]interface{}, error) {
    idempotencyKey := uuid.New().String()

    for attempt := 0; attempt < maxRetries; attempt++ {
        result, err := c.createOrderAttempt(orderData, idempotencyKey)

        if err == nil {
            return result, nil
        }

        // Check if it's a concurrent request error (409)
        if httpErr, ok := err.(*HTTPError); ok && httpErr.StatusCode == 409 {
            if attempt < maxRetries-1 {
                // Exponential backoff
                backoff := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
                time.Sleep(backoff)
                continue
            }
        }

        return nil, err
    }

    return nil, fmt.Errorf("max retries exceeded")
}

func (c *WMSClient) createOrderAttempt(orderData map[string]interface{}, idempotencyKey string) (map[string]interface{}, error) {
    bodyBytes, err := json.Marshal(orderData)
    if err != nil {
        return nil, err
    }

    req, err := http.NewRequest("POST", c.BaseURL+"/api/v1/orders", bytes.NewBuffer(bodyBytes))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Idempotency-Key", idempotencyKey)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        var result map[string]interface{}
        if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
            return nil, err
        }
        return result, nil
    }

    return nil, &HTTPError{
        StatusCode: resp.StatusCode,
        Message:    fmt.Sprintf("Request failed with status %d", resp.StatusCode),
    }
}

type HTTPError struct {
    StatusCode int
    Message    string
}

func (e *HTTPError) Error() string {
    return e.Message
}

// Usage
func main() {
    client := NewWMSClient("https://api.wms.example.com")

    order, err := client.CreateOrder(map[string]interface{}{
        "customerId": "CUST-001",
        "priority":   "standard",
        "items": []map[string]interface{}{
            {"sku": "SKU-001", "quantity": 2, "price": 29.99},
        },
        "shippingAddress": map[string]string{
            "street":  "123 Main St",
            "city":    "New York",
            "state":   "NY",
            "zipCode": "10001",
            "country": "US",
        },
    })

    if err != nil {
        panic(err)
    }

    fmt.Printf("Order created: %s\n", order["orderId"])
}
```

---

### Best Practices

1. **Always Use UUIDs**: Generate UUID v4 for maximum uniqueness
   ```javascript
   const idempotencyKey = uuidv4(); // ✓ Good
   const idempotencyKey = Date.now().toString(); // ✗ Bad (not unique enough)
   ```

2. **Store Keys Client-Side**: Keep keys for retry scenarios
   ```javascript
   // Store in database, localStorage, or in-memory cache
   const operation = {
     id: 'op-123',
     idempotencyKey: uuidv4(),
     payload: orderData,
     status: 'pending'
   };
   await db.operations.insert(operation);
   ```

3. **Implement Exponential Backoff**: For 409 Conflict errors
   ```javascript
   const backoff = Math.pow(2, attempt) * 100; // 100ms, 200ms, 400ms, ...
   await sleep(backoff);
   ```

4. **Don't Reuse Keys Across Operations**: Each unique operation gets a new key
   ```javascript
   // ✗ Bad - reusing same key
   await client.createOrder(orderA, { idempotencyKey: 'my-key' });
   await client.createOrder(orderB, { idempotencyKey: 'my-key' }); // Will fail with 422!

   // ✓ Good - new key per operation
   await client.createOrder(orderA, { idempotencyKey: uuidv4() });
   await client.createOrder(orderB, { idempotencyKey: uuidv4() });
   ```

5. **Handle All Error Codes Appropriately**:
   - `400`: Fix key format and retry with new key
   - `409`: Retry with same key after backoff
   - `422`: Don't retry - parameters changed, use new key
   - `503`: Retry with same key after backoff

---

### Error Handling

#### Error Code Reference

| Code | Status | Meaning | Client Action |
|------|--------|---------|---------------|
| `IDEMPOTENCY_KEY_REQUIRED` | 400 | Key missing when required | Add header with UUID v4 |
| `IDEMPOTENCY_KEY_INVALID` | 400 | Invalid key format | Fix format (alphanumeric, hyphens, underscores only) |
| `IDEMPOTENCY_CONCURRENT_REQUEST` | 409 | Another request processing | Retry with same key after backoff |
| `IDEMPOTENCY_PARAMETER_MISMATCH` | 422 | Parameters changed | Use new key if changing parameters |
| `IDEMPOTENCY_STORAGE_UNAVAILABLE` | 503 | MongoDB unavailable | Retry with same key after backoff |

#### Example Error Handling

```javascript
async function handleIdempotencyError(error, idempotencyKey, orderData) {
  const errorData = await error.response.json();

  switch (errorData.code) {
    case 'IDEMPOTENCY_KEY_REQUIRED':
      // Service now requires idempotency keys
      return createOrder(orderData, { idempotencyKey: uuidv4() });

    case 'IDEMPOTENCY_KEY_INVALID':
      // Our key format was invalid
      console.error('Invalid idempotency key format:', idempotencyKey);
      return createOrder(orderData, { idempotencyKey: uuidv4() });

    case 'IDEMPOTENCY_CONCURRENT_REQUEST':
      // Another request is processing - wait and retry
      const retryAfter = errorData.details?.retryAfter || 2;
      await sleep(retryAfter * 1000);
      return createOrder(orderData, { idempotencyKey }); // Same key!

    case 'IDEMPOTENCY_PARAMETER_MISMATCH':
      // We changed parameters - use new key or keep old parameters
      console.error('Parameters changed for key:', idempotencyKey);
      throw new Error('Cannot change parameters with same idempotency key');

    case 'IDEMPOTENCY_STORAGE_UNAVAILABLE':
      // MongoDB issue - retry after delay
      const delay = errorData.details?.retryAfter || 5;
      await sleep(delay * 1000);
      return createOrder(orderData, { idempotencyKey }); // Same key!

    default:
      throw error;
  }
}
```

---

## Testing

### Unit Testing Idempotency

```go
func TestIdempotencyMiddleware(t *testing.T) {
    gin.SetMode(gin.TestMode)

    // Setup mock repository
    repo := &mockKeyRepository{
        acquireLockFunc: func(ctx context.Context, key *idempotency.IdempotencyKey) (*idempotency.IdempotencyKey, bool, error) {
            key.ID = primitive.NewObjectID()
            return key, true, nil
        },
    }

    config := &idempotency.Config{
        ServiceName:     "test-service",
        Repository:      repo,
        RequireKey:      false,
        OnlyMutating:    true,
        MaxKeyLength:    255,
        LockTimeout:     5 * time.Minute,
        RetentionPeriod: 24 * time.Hour,
        MaxResponseSize: 1024 * 1024,
    }

    router := gin.New()
    router.Use(idempotency.Middleware(config))
    router.POST("/test", func(c *gin.Context) {
        c.JSON(http.StatusCreated, gin.H{"message": "created"})
    })

    // Test 1: First request
    req1 := httptest.NewRequest("POST", "/test", bytes.NewBufferString(`{"data":"test"}`))
    req1.Header.Set("Content-Type", "application/json")
    req1.Header.Set("Idempotency-Key", "test-key-123")
    w1 := httptest.NewRecorder()
    router.ServeHTTP(w1, req1)

    assert.Equal(t, 201, w1.Code)

    // Test 2: Retry with same key (should return cached response)
    req2 := httptest.NewRequest("POST", "/test", bytes.NewBufferString(`{"data":"test"}`))
    req2.Header.Set("Content-Type", "application/json")
    req2.Header.Set("Idempotency-Key", "test-key-123")
    w2 := httptest.NewRecorder()
    router.ServeHTTP(w2, req2)

    assert.Equal(t, 200, w2.Code) // Cached
    assert.Equal(t, w1.Body.String(), w2.Body.String())
}
```

### Integration Testing

```go
func TestIdempotencyIntegration(t *testing.T) {
    // Start MongoDB testcontainer
    ctx := context.Background()
    mongoContainer, err := testcontainers.StartMongoDBContainer(ctx)
    require.NoError(t, err)
    defer mongoContainer.Terminate(ctx)

    // Connect to MongoDB
    mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoContainer.ConnectionString()))
    require.NoError(t, err)
    db := mongoClient.Database("test")

    // Initialize idempotency
    idempotencyKeyRepo := idempotency.NewMongoKeyRepository(db)
    err = idempotencyKeyRepo.EnsureIndexes(ctx)
    require.NoError(t, err)

    // Test idempotency flow
    key := &idempotency.IdempotencyKey{
        Key:                "test-123",
        ServiceID:          "test-service",
        RequestPath:        "/test",
        RequestMethod:      "POST",
        RequestFingerprint: "abc123",
        CreatedAt:          time.Now().UTC(),
        ExpiresAt:          time.Now().UTC().Add(24 * time.Hour),
    }

    // First acquisition - should succeed
    result1, isNew1, err := idempotencyKeyRepo.AcquireLock(ctx, key)
    require.NoError(t, err)
    assert.True(t, isNew1)

    // Second acquisition - should return existing
    result2, isNew2, err := idempotencyKeyRepo.AcquireLock(ctx, key)
    require.NoError(t, err)
    assert.False(t, isNew2)
    assert.Equal(t, result1.ID, result2.ID)
}
```

---

## Monitoring

### Key Metrics to Monitor

```promql
# Idempotency hit rate (cache effectiveness)
sum(rate(idempotency_hits_total[5m])) /
  (sum(rate(idempotency_hits_total[5m])) + sum(rate(idempotency_misses_total[5m])))

# Parameter mismatches (client issues)
rate(idempotency_parameter_mismatches_total[5m])

# Concurrent collisions (client retry behavior)
rate(idempotency_concurrent_collisions_total[5m])

# Storage errors (MongoDB health)
rate(idempotency_storage_errors_total[5m])

# Lock acquisition latency (P99)
histogram_quantile(0.99, rate(idempotency_lock_acquisition_duration_seconds_bucket[5m]))

# Message deduplication rate
sum(rate(message_deduplication_hits_total[5m])) by (topic, consumer_group)
```

### Recommended Alerts

```yaml
groups:
  - name: idempotency
    rules:
      - alert: HighParameterMismatchRate
        expr: rate(idempotency_parameter_mismatches_total[5m]) > 10
        for: 5m
        annotations:
          summary: "High rate of parameter mismatches detected"
          description: "{{ $value }} mismatches/sec - clients may be incorrectly reusing keys"

      - alert: IdempotencyStorageErrors
        expr: rate(idempotency_storage_errors_total[5m]) > 1
        for: 5m
        annotations:
          summary: "Idempotency storage errors detected"
          description: "MongoDB connection issues on {{ $labels.service }}"

      - alert: SlowLockAcquisition
        expr: histogram_quantile(0.99, rate(idempotency_lock_acquisition_duration_seconds_bucket[5m])) > 0.5
        for: 5m
        annotations:
          summary: "Slow idempotency lock acquisition"
          description: "P99 lock time: {{ $value }}s on {{ $labels.service }}"
```

---

## Troubleshooting

### Common Issues

#### Issue 1: Clients Getting 422 Errors

**Symptoms:**
- High `idempotency_parameter_mismatches_total`
- Client reports "parameters differ" errors

**Diagnosis:**
```bash
# Check recent mismatches
curl http://localhost:9090/api/v1/query?query=rate(idempotency_parameter_mismatches_total[5m])
```

**Common causes:**
1. Client reusing idempotency key with different parameters
2. Request body contains dynamic fields (timestamps, nonces)
3. Client not storing keys properly for retries

**Solution:**
```javascript
// BAD: Reusing key with different data
const key = uuidv4();
await createOrder({customerId: 'A', amount: 100}, {idempotencyKey: key});
await createOrder({customerId: 'A', amount: 200}, {idempotencyKey: key}); // 422 error!

// GOOD: New key for different operation
await createOrder({customerId: 'A', amount: 100}, {idempotencyKey: uuidv4()});
await createOrder({customerId: 'A', amount: 200}, {idempotencyKey: uuidv4()});

// GOOD: Same key for true retry
const key = uuidv4();
const data = {customerId: 'A', amount: 100};
try {
  await createOrder(data, {idempotencyKey: key});
} catch (error) {
  await createOrder(data, {idempotencyKey: key}); // Same key, same data
}
```

#### Issue 2: Service Returning 503 Errors

**Symptoms:**
- High `idempotency_storage_errors_total`
- HTTP 503 responses

**Diagnosis:**
```bash
# Check MongoDB health
kubectl exec -it mongodb-0 -- mongo --eval "db.serverStatus()"

# Check service logs
kubectl logs deployment/order-service | grep -i idempotency
```

**Solutions:**
1. Check MongoDB connection: `MONGODB_URI` configuration
2. Scale MongoDB replica set if under load
3. Check network policies between service and MongoDB

#### Issue 3: Idempotency Not Working

**Symptoms:**
- Duplicate orders being created
- No `idempotency_hits_total` metrics

**Diagnosis:**
```bash
# Check if indexes exist
mongosh --eval "db.idempotency_keys.getIndexes()"

# Check if middleware is configured
curl http://localhost:8001/api/v1/orders \
  -H "Idempotency-Key: test" \
  -X POST \
  -d '{}' \
  -v # Look for idempotency-related headers in response
```

**Solutions:**
1. Verify indexes: Run `idempotency.InitializeIndexes()`
2. Check middleware order: Idempotency must be after logger, before handlers
3. Verify client is sending `Idempotency-Key` header

---

## Migration Checklist

When rolling out idempotency to an existing service:

- [ ] Add idempotency repositories initialization
- [ ] Configure idempotency middleware with `RequireKey: false`
- [ ] Deploy services
- [ ] Initialize MongoDB indexes
- [ ] Update API documentation with `Idempotency-Key` header
- [ ] Update client SDKs to send `Idempotency-Key`
- [ ] Monitor metrics for adoption
- [ ] After 100% client migration, switch to `RequireKey: true`
- [ ] Update OpenAPI specs to mark header as required

---

## Additional Resources

- [Idempotency Package README](../../shared/pkg/idempotency/README.md)
- [Stripe Idempotency Guide](https://stripe.com/docs/api/idempotent_requests)
- [CloudEvents Specification](https://cloudevents.io/)
- [WMS Platform Documentation](../../README.md)

## Support

For questions or issues:
- File a GitHub issue in the `wms-platform` repository
- Contact the platform team on Slack: `#wms-platform-support`
- Review the troubleshooting section above
