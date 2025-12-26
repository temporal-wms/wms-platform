# WMS Platform Resilience & Error Handling

This document describes the error handling and resilience patterns implemented across the WMS platform.

## Overview

The platform implements a comprehensive resilience strategy with three layers:

1. **Error Handling** - Standardized error types and HTTP error responses
2. **Circuit Breakers** - Fail-fast protection for MongoDB and Kafka operations
3. **Retry Policies** - Configurable retry logic with exponential backoff

---

## 1. Error Handling

### Standardized Error Types

Location: `shared/pkg/errors/errors.go`

All errors in the platform use the `AppError` type, which includes:
- HTTP status code
- Error code (machine-readable)
- Human-readable message
- Optional details map
- Wrapped underlying error

```go
type AppError struct {
    Code       string            // e.g., "VALIDATION_ERROR"
    Message    string            // e.g., "Invalid order ID format"
    Details    map[string]string // Optional field details
    HTTPStatus int               // HTTP status code
    Err        error             // Wrapped error
}
```

### Error Codes

| Error Code | HTTP Status | Use Case |
|------------|-------------|----------|
| `VALIDATION_ERROR` | 400 | Invalid input data |
| `RESOURCE_NOT_FOUND` | 404 | Resource doesn't exist |
| `CONFLICT` | 409 | Duplicate or conflicting resource |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Permission denied |
| `INTERNAL_ERROR` | 500 | Unexpected server error |
| `BAD_REQUEST` | 400 | Malformed request |
| `SERVICE_UNAVAILABLE` | 503 | Downstream service unavailable |
| `TIMEOUT` | 504 | Operation timed out |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |

### Creating Errors

```go
import "github.com/wms-platform/shared/pkg/errors"

// Not found
err := errors.ErrNotFound("order")
err := errors.ErrNotFoundWithID("order", orderID)

// Validation
err := errors.ErrValidation("invalid order status")
err := errors.ErrValidationWithFields("validation failed", map[string]string{
    "orderId": "required",
    "customerId": "invalid format",
})

// Internal
err := errors.ErrInternal("database connection failed").Wrap(dbErr)

// Service unavailable
err := errors.ErrServiceUnavailable("order-service")
```

### Error Middleware

Location: `shared/pkg/middleware/error_handler.go`

The error middleware automatically:
- Catches errors from handlers
- Maps domain errors to HTTP responses
- Logs errors with context
- Returns standardized JSON responses

```go
// Add to Gin router
router.Use(middleware.ErrorHandler(logger))

// In handlers
func createOrderHandler(c *gin.Context) {
    responder := middleware.NewErrorResponder(c, logger)

    order, err := service.CreateOrder(ctx, req)
    if err != nil {
        responder.RespondWithError(err)
        return
    }

    c.JSON(http.StatusCreated, order)
}
```

### Error Response Format

```json
{
  "code": "VALIDATION_ERROR",
  "message": "Invalid order data",
  "details": {
    "orderId": "required",
    "customerId": "invalid format"
  },
  "requestId": "req-123",
  "timestamp": "2025-12-23T21:30:00Z",
  "path": "/api/v1/orders"
}
```

---

## 2. Circuit Breakers

### Overview

Circuit breakers protect against cascading failures by:
- **Closed State**: Normal operation, requests flow through
- **Open State**: Failures exceeded threshold, requests fail immediately
- **Half-Open State**: Testing if service recovered

Location: `shared/pkg/resilience/circuit_breaker.go`

### Configuration

```go
type CircuitBreakerConfig struct {
    Name                  string        // Circuit breaker name
    MaxRequests           uint32        // Max requests in half-open state (default: 3)
    Interval              time.Duration // Failure count reset interval (default: 60s)
    Timeout               time.Duration // Open -> Half-open timeout (default: 30s)
    FailureThreshold      uint32        // Consecutive failures to open (default: 5)
    SuccessThreshold      uint32        // Successes to close from half-open (default: 2)
    FailureRatioThreshold float64       // Failure ratio to open (default: 0.5)
    MinRequestsToTrip     uint32        // Min requests before evaluating ratio (default: 10)
}
```

### MongoDB Circuit Breaker

Location: `shared/pkg/mongodb/circuit_breaker_client.go`

Protects all MongoDB operations:

```go
// Create production-ready MongoDB client
client, err := mongodb.NewProductionClient(ctx, config, metrics, logger)
if err != nil {
    return nil, err
}

// All operations are automatically protected
collection := client.Collection("orders")
result, err := collection.InsertOne(ctx, order)
// If circuit is open, returns immediately with error
```

**Configuration:**
- Failure Threshold: 5 consecutive failures
- Timeout: 30 seconds
- Success Threshold: 2 successes to close
- Failure Ratio: 50% over 10 requests

### Kafka Circuit Breaker

Location: `shared/pkg/kafka/circuit_breaker.go`

Protects Kafka producer and consumer operations:

```go
// Create production-ready Kafka producer
producer := kafka.NewProductionProducer(config, metrics, logger)

// Publish with circuit breaker protection
err := producer.PublishEvent(ctx, topic, event)
// Returns immediately if circuit is open

// Create production-ready Kafka consumer
consumer := kafka.NewProductionConsumer(config, metrics, logger)

// Subscribe with circuit breaker protection
consumer.Subscribe(topic, eventType, func(ctx context.Context, event *cloudevents.WMSCloudEvent) error {
    // Handler is protected by circuit breaker
    return processEvent(ctx, event)
})
```

**Producer Configuration:**
- Failure Threshold: 5 consecutive failures
- Timeout: 30 seconds
- Failure Ratio: 50% over 10 requests

**Consumer Configuration:**
- Failure Threshold: 10 consecutive failures (higher for consumers)
- Timeout: 30 seconds
- Failure Ratio: 70% over 20 requests (more tolerant)

### Circuit Breaker States

When circuit breaker opens:
```go
// Synchronous operations return error immediately
err := producer.PublishEvent(ctx, topic, event)
if err != nil {
    if errors.Is(err, resilience.ErrCircuitOpen) {
        // Handle gracefully - queue for later, use fallback, etc.
    }
}

// Asynchronous operations invoke callback with error
producer.PublishEventAsync(ctx, topic, event, func(err error) {
    if err != nil && errors.Is(err, resilience.ErrCircuitOpen) {
        // Circuit is open
    }
})
```

### Monitoring Circuit Breakers

```go
// Get circuit breaker status
registry := resilience.NewCircuitBreakerRegistry(logger)
cb := registry.Get("mongodb")

// Check state
state := cb.State() // Closed, Open, or Half-Open

// Get counts
counts := cb.Counts()
fmt.Printf("Requests: %d, Failures: %d, Successes: %d\n",
    counts.Requests, counts.TotalFailures, counts.TotalSuccesses)
```

---

## 3. Retry Policies

### Temporal Workflow Retry Policies

Location: `orchestrator/internal/workflows/retry_policies.go`

Four pre-configured retry policies:

#### StandardRetry (Default)
- **Use Case**: Normal operations
- **Attempts**: 3
- **Initial Interval**: 1 second
- **Backoff**: 2x exponential
- **Max Interval**: 1 minute
- **Non-Retryable**: ValidationError, NotFoundError

```go
ctx = workflow.WithActivityOptions(ctx, workflows.GetStandardActivityOptions())
err := workflow.ExecuteActivity(ctx, "ValidateOrder", input).Get(ctx, &result)
```

#### AggressiveRetry
- **Use Case**: Critical operations that must succeed
- **Attempts**: 5
- **Initial Interval**: 500ms
- **Backoff**: 2x exponential
- **Max Interval**: 30 seconds
- **Non-Retryable**: ValidationError, NotFoundError, ConflictError

```go
ctx = workflow.WithActivityOptions(ctx, workflows.GetCriticalActivityOptions())
err := workflow.ExecuteActivity(ctx, "CreateShipment", input).Get(ctx, &result)
```

#### ConservativeRetry
- **Use Case**: Expensive operations
- **Attempts**: 2
- **Initial Interval**: 2 seconds
- **Backoff**: 2x exponential
- **Max Interval**: 2 minutes
- **Non-Retryable**: ValidationError, NotFoundError

```go
ctx = workflow.WithActivityOptions(ctx, workflows.GetLongRunningActivityOptions())
err := workflow.ExecuteActivity(ctx, "GenerateReport", input).Get(ctx, &result)
```

#### NoRetry
- **Use Case**: Idempotent operations or operations that should fail fast
- **Attempts**: 1

```go
opts := workflow.ActivityOptions{
    StartToCloseTimeout: 5 * time.Minute,
    RetryPolicy:         workflows.GetRetryPolicy(workflows.NoRetry),
}
ctx = workflow.WithActivityOptions(ctx, opts)
```

### Custom Retry Policies

```go
// Custom activity options
opts := workflows.GetActivityOptions(workflows.ActivityOptionsConfig{
    StartToCloseTimeout: 10 * time.Minute,
    RetryPolicy:         workflows.AggressiveRetry,
    HeartbeatTimeout:    30 * time.Second,
})
ctx = workflow.WithActivityOptions(ctx, opts)

// Custom retry policy
customRetry := &temporal.RetryPolicy{
    InitialInterval:    500 * time.Millisecond,
    BackoffCoefficient: 1.5,
    MaximumInterval:    10 * time.Second,
    MaximumAttempts:    10,
    NonRetryableErrorTypes: []string{
        "ValidationError",
    },
}
```

### Child Workflow Retry Policies

```go
// Standard child workflow options
childOpts := workflows.GetStandardChildWorkflowOptions()
childCtx := workflow.WithChildOptions(ctx, childOpts)

err := workflow.ExecuteChildWorkflow(childCtx, workflows.PickingWorkflow, input).Get(ctx, &result)

// Custom child workflow options
childOpts := workflows.GetChildWorkflowOptions(workflows.ChildWorkflowOptionsConfig{
    WorkflowExecutionTimeout: 2 * time.Hour,
    RetryPolicy:              workflows.AggressiveRetry,
    ParentClosePolicy:        temporal.ParentClosePolicyAbandon,
})
```

### Application-Level Retry

Location: `shared/pkg/resilience/circuit_breaker.go`

For non-Temporal operations:

```go
import "github.com/wms-platform/shared/pkg/resilience"

// Simple retry
config := resilience.DefaultRetryConfig()
config.MaxAttempts = 3
config.RetryableErrors = func(err error) bool {
    // Only retry on connection errors
    return strings.Contains(err.Error(), "connection")
}

err := resilience.Retry(ctx, config, func() error {
    return externalAPI.Call()
})

// Retry with result
result, err := resilience.RetryWithResult(ctx, config, func() (*Response, error) {
    return externalAPI.CallWithResult()
})
```

---

## 4. Error Classification

### Automatic Error Classification

The platform automatically classifies errors to determine retry behavior:

```go
classification := workflows.ClassifyError(err)

if classification.IsRetryable {
    // Retry with backoff
} else {
    // Fail immediately
}
```

### Error Categories

| Category | IsTransient | IsRetryable | Examples |
|----------|-------------|-------------|----------|
| `validation` | No | No | "invalid order ID" |
| `not_found` | No | No | "order not found" |
| `conflict` | No | No | "order already exists" |
| `timeout` | Yes | Yes | "request timeout", "deadline exceeded" |
| `connection` | Yes | Yes | "connection refused" |
| `circuit_breaker` | Yes | Yes | "circuit breaker open" |
| `unknown` | Yes | Yes | Other errors |

---

## 5. Best Practices

### Service Handlers

```go
func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
    // 1. Validate input
    if err := req.Validate(); err != nil {
        return nil, errors.ErrValidation(err.Error())
    }

    // 2. Check for conflicts
    existing, err := s.repo.FindByID(ctx, req.OrderID)
    if err == nil && existing != nil {
        return nil, errors.ErrConflict("order already exists").
            WithDetail("orderId", req.OrderID)
    }

    // 3. Perform operation (protected by circuit breaker automatically)
    order, err := s.repo.Create(ctx, req)
    if err != nil {
        // Wrap database errors
        return nil, errors.ErrInternal("failed to create order").Wrap(err)
    }

    // 4. Publish event (protected by circuit breaker automatically)
    event := createOrderCreatedEvent(order)
    if err := s.eventPublisher.Publish(ctx, event); err != nil {
        // Log but don't fail the operation
        s.logger.Error("failed to publish event", "error", err)
    }

    return order, nil
}
```

### HTTP Handlers

```go
func createOrderHandler(c *gin.Context) {
    responder := middleware.NewErrorResponder(c, logger)

    var req CreateOrderRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        responder.RespondBadRequest("invalid request body")
        return
    }

    order, err := service.CreateOrder(c.Request.Context(), &req)
    if err != nil {
        // Automatically converts domain errors to HTTP responses
        responder.RespondWithError(err)
        return
    }

    c.JSON(http.StatusCreated, order)
}
```

### Temporal Activities

```go
func (a *OrderActivities) ValidateOrder(ctx context.Context, input workflows.OrderFulfillmentInput) (bool, error) {
    logger := activity.GetLogger(ctx)

    // Activities are automatically retried based on workflow retry policy
    result, err := a.clients.ValidateOrder(ctx, input.OrderID)
    if err != nil {
        // Circuit breaker handles fail-fast
        // Temporal handles retries
        return false, fmt.Errorf("order validation failed: %w", err)
    }

    return true, nil
}
```

### Temporal Workflows

```go
func OrderFulfillmentWorkflow(ctx workflow.Context, input OrderFulfillmentInput) (*OrderFulfillmentResult, error) {
    logger := workflow.GetLogger(ctx)

    // Set activity options with appropriate retry policy
    ctx = workflow.WithActivityOptions(ctx, workflows.GetStandardActivityOptions())

    // Execute activity - automatically retried on transient failures
    var orderValidated bool
    err := workflow.ExecuteActivity(ctx, "ValidateOrder", input).Get(ctx, &orderValidated)
    if err != nil {
        // Classification determines if error is retryable
        classification := workflows.ClassifyError(err)
        if !classification.IsRetryable {
            // Fail workflow immediately for non-retryable errors
            return nil, err
        }
        // Temporal will retry for retryable errors
        return nil, err
    }

    return &result, nil
}
```

---

## 6. Monitoring & Alerting

### Circuit Breaker Metrics

The following metrics are automatically collected:

- `wms_circuit_breaker_state{name}` - Current state (0=Closed, 1=HalfOpen, 2=Open)
- `wms_circuit_breaker_requests_total{name}` - Total requests
- `wms_circuit_breaker_failures_total{name}` - Total failures
- `wms_circuit_breaker_successes_total{name}` - Total successes

### Error Metrics

- `wms_http_errors_total{service, code, status}` - HTTP errors by code
- `wms_activity_errors_total{activity_type, error_category}` - Temporal activity errors
- `wms_workflow_failures_total{workflow_type, reason}` - Workflow failures

### Alerting Rules

```yaml
# Circuit breaker open for >5 minutes
- alert: CircuitBreakerOpen
  expr: wms_circuit_breaker_state{name="mongodb"} == 2
  for: 5m
  annotations:
    summary: "MongoDB circuit breaker has been open for >5 minutes"

# High error rate
- alert: HighErrorRate
  expr: rate(wms_http_errors_total[5m]) > 10
  for: 2m
  annotations:
    summary: "High HTTP error rate detected"

# Workflow failure rate
- alert: WorkflowFailureRate
  expr: rate(wms_workflow_failures_total[10m]) > 1
  for: 5m
  annotations:
    summary: "High workflow failure rate"
```

---

## 7. Testing

### Testing Error Handling

```go
func TestCreateOrder_ValidationError(t *testing.T) {
    service := NewOrderService(repo, publisher, logger)

    req := &CreateOrderRequest{} // Invalid request
    _, err := service.CreateOrder(ctx, req)

    require.Error(t, err)

    var appErr *errors.AppError
    require.True(t, errors.As(err, &appErr))
    assert.Equal(t, errors.CodeValidationError, appErr.Code)
    assert.Equal(t, http.StatusBadRequest, appErr.HTTPStatus)
}
```

### Testing Circuit Breaker

```go
func TestCircuitBreaker_OpensOnFailures(t *testing.T) {
    cb := resilience.NewCircuitBreaker(config, logger)

    // Trigger failures
    for i := 0; i < 5; i++ {
        _, err := cb.Execute(ctx, func() (interface{}, error) {
            return nil, errors.New("service error")
        })
        require.Error(t, err)
    }

    // Circuit should be open
    assert.Equal(t, gobreaker.StateOpen, cb.State())

    // Next request should fail immediately
    _, err := cb.Execute(ctx, func() (interface{}, error) {
        return nil, nil
    })
    require.Error(t, err)
    assert.Contains(t, err.Error(), "circuit breaker open")
}
```

### Testing Temporal Retry

```go
func TestWorkflow_RetryOnTransientError(t *testing.T) {
    testSuite := &testsuite.WorkflowTestSuite{}
    env := testSuite.NewTestWorkflowEnvironment()

    // Mock activity with transient failure then success
    attemptCount := 0
    env.OnActivity("ValidateOrder", mock.Anything, mock.Anything).
        Return(func(ctx context.Context, input interface{}) (bool, error) {
            attemptCount++
            if attemptCount < 3 {
                return false, errors.New("connection timeout")
            }
            return true, nil
        })

    env.ExecuteWorkflow(OrderFulfillmentWorkflow, input)

    require.True(t, env.IsWorkflowCompleted())
    require.NoError(t, env.GetWorkflowError())
    assert.Equal(t, 3, attemptCount) // Should have retried twice
}
```

---

## Summary

The WMS platform implements a comprehensive resilience strategy:

✅ **Standardized Error Handling**
- Consistent error types across all services
- Automatic HTTP error mapping
- Detailed error context and logging

✅ **Circuit Breakers**
- MongoDB operations protected
- Kafka operations protected
- Fail-fast on cascading failures
- Automatic recovery detection

✅ **Retry Policies**
- Configurable retry strategies
- Exponential backoff
- Non-retryable error classification
- Temporal workflow-level retries

This ensures the platform is resilient to:
- Transient network failures
- Downstream service outages
- Database connection issues
- Message broker unavailability
- High load conditions
