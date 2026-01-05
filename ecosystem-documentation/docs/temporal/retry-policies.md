---
sidebar_position: 22
slug: /temporal/retry-policies
---

# Retry Policies

Reference for retry configuration and timeout settings in the WMS Platform.

## Default Retry Policy

The standard retry policy used across most activities:

```go
RetryPolicy: &temporal.RetryPolicy{
    InitialInterval:    time.Second,
    BackoffCoefficient: 2.0,
    MaximumInterval:    time.Minute,
    MaximumAttempts:    3,
}
```

| Parameter | Value | Description |
|-----------|-------|-------------|
| InitialInterval | 1 second | First retry delay |
| BackoffCoefficient | 2.0 | Exponential backoff multiplier |
| MaximumInterval | 1 minute | Cap on retry delay |
| MaximumAttempts | 3 | Total attempts (initial + 2 retries) |

## Retry Timeline

```
Attempt 1: Immediate
Attempt 2: After 1s
Attempt 3: After 2s (1s × 2.0)
```

---

## Activity Timeouts

### Timeout Types

| Timeout | Purpose | Typical Value |
|---------|---------|---------------|
| `StartToCloseTimeout` | Max time for single attempt | 5-15 minutes |
| `ScheduleToCloseTimeout` | Max time including retries | 30 minutes |
| `HeartbeatTimeout` | Detect stuck workers | 30 seconds |

### Configuration Example

```go
ao := workflow.ActivityOptions{
    ScheduleToCloseTimeout: 30 * time.Minute, // Total time including retries
    StartToCloseTimeout:    10 * time.Minute, // Single attempt time
    HeartbeatTimeout:       30 * time.Second, // Detect stuck workers
    RetryPolicy: &temporal.RetryPolicy{
        InitialInterval:    time.Second,
        BackoffCoefficient: 2.0,
        MaximumInterval:    time.Minute,
        MaximumAttempts:    3,
    },
}
ctx = workflow.WithActivityOptions(ctx, ao)
```

---

## Workflow-Specific Timeouts

### OrderFulfillmentWorkflow

```go
{
    TaskQueue:                "orchestrator",
    WorkflowExecutionTimeout: 24 * time.Hour,
    ActivityTimeout:          5 * time.Minute, // 30 min with retries
    HeartbeatTimeout:         30 * time.Second,
}
```

### PlanningWorkflow

```go
{
    WorkflowExecutionTimeout: 5 * time.Hour,
    ActivityTimeout:          2 * time.Minute,
}
```

**Wave Assignment Timeouts (Priority-based):**

| Priority | Timeout |
|----------|---------|
| `same_day` | 30 minutes |
| `next_day` | 2 hours |
| `standard` | 4 hours |

### WESExecutionWorkflow

```go
{
    WorkflowExecutionTimeout: 4 * time.Hour,
    ActivityTimeout:          5 * time.Minute,
}
```

**Stage-specific timeouts from execution plan:**

```go
// Default if not specified
stageTimeout := 30 * time.Minute

// Or from stage definition
stageTimeout = time.Duration(stage.TimeoutMins) * time.Minute
```

### Picking Workflow

```go
{
    ActivityTimeout:   10 * time.Minute,
    HeartbeatTimeout:  30 * time.Second,
    PickingTimeout:    30 * time.Minute, // Signal wait
}
```

### Packing Workflow

```go
{
    ActivityTimeout:  15 * time.Minute,
    PackingTimeout:   1 * time.Hour, // Signal wait
}
```

### Consolidation Workflow

```go
{
    ActivityTimeout:       15 * time.Minute,
    ToteArrivalTimeout:    30 * time.Minute,
    ConsolidationTimeout:  1 * time.Hour,
}
```

---

## Non-Retryable Errors

Errors that should NOT be retried:

```go
RetryPolicy: &temporal.RetryPolicy{
    NonRetryableErrorTypes: []string{
        "ValidationError",
        "NotFoundError",
        "ConflictError",
        "BusinessRuleError",
    },
}
```

### Creating Non-Retryable Errors

```go
// Activity returns non-retryable error
return temporal.NewApplicationError(
    fmt.Sprintf("order validation failed: %v", result.Errors),
    "ValidationError",  // Error type
    result.Errors,       // Details
)
```

### Common Non-Retryable Scenarios

| Error Type | Scenario | Example |
|------------|----------|---------|
| `ValidationError` | Invalid input data | Missing required field |
| `NotFoundError` | Resource doesn't exist | Order not found |
| `ConflictError` | State conflict | Order already shipped |
| `BusinessRuleError` | Business rule violation | Insufficient inventory |

---

## Child Workflow Retry

Child workflows should NOT have retry policies (they have their own built-in retry):

```go
// Good: No retry policy on child workflow
childOpts := workflow.ChildWorkflowOptions{
    WorkflowID:               fmt.Sprintf("child-%s", orderID),
    WorkflowExecutionTimeout: 4 * time.Hour,
    // No RetryPolicy - child workflows handle retries internally
}

// Avoid: Retry policy on child workflow (can cause duplicate executions)
childOpts := workflow.ChildWorkflowOptions{
    RetryPolicy: &temporal.RetryPolicy{...}, // Don't do this
}
```

---

## Heartbeat Usage

Use heartbeats for activities that:
- Process multiple items
- Take longer than 30 seconds
- Need progress tracking

```go
func (a *InventoryActivities) ConfirmInventoryPick(ctx context.Context, input Input) error {
    for i, item := range input.PickedItems {
        // Record progress for long operations
        activity.RecordHeartbeat(ctx, fmt.Sprintf("Item %d/%d", i+1, len(items)))

        // Process item...
    }
    return nil
}
```

---

## Constants Reference

From `constants.go`:

```go
const (
    // Default timeouts
    DefaultActivityTimeout        = 5 * time.Minute
    DefaultChildWorkflowTimeout   = 4 * time.Hour
    DefaultWorkflowTimeout        = 24 * time.Hour

    // Retry configuration
    DefaultRetryInitialInterval    = time.Second
    DefaultRetryBackoffCoefficient = 2.0
    DefaultRetryMaxInterval        = time.Minute
    DefaultMaxRetryAttempts        = 3

    // Signal timeouts
    DefaultSignalTimeout = 30 * time.Minute
)
```

---

## Timeout Selection Guide

| Operation Type | StartToClose | ScheduleToClose | Heartbeat |
|----------------|--------------|-----------------|-----------|
| Quick API call | 30s-1m | 2-5m | Not needed |
| Database operation | 1-5m | 10-15m | Not needed |
| Multi-item processing | 5-10m | 30m | 30s |
| Long-running task | 15-30m | 1h | 30-60s |
| Signal wait | N/A | N/A | Not needed |

---

## Best Practices

1. **Use ScheduleToCloseTimeout**: Set the total time budget including retries
2. **Set HeartbeatTimeout**: For activities > 30 seconds
3. **Avoid retry on child workflows**: They handle retries internally
4. **Use non-retryable errors**: For business failures that won't succeed on retry
5. **Log retry attempts**: Track retry behavior for debugging

## Related Documentation

- [Activities Overview](./activities/overview) - Activity patterns
- [Task Queues](./task-queues) - Queue configuration
- [Order Fulfillment Workflow](./workflows/order-fulfillment) - Timeout examples
