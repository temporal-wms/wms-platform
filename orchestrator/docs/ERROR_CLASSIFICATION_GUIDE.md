# Temporal Error Classification Guide

## Overview

Proper error classification is critical for workflow reliability. Temporal distinguishes between:
- **Retriable errors** (system/transient failures) → Regular Go errors
- **Non-retriable errors** (business/permanent failures) → `temporal.ApplicationError`

## Error Classification Rules

### Use `temporal.NewApplicationError` for Business Errors (NON-RETRYABLE)

Business errors represent permanent failures that won't succeed even with retries:

```go
import "go.temporal.io/sdk/temporal"

// Example: Order validation failure
if !result.Valid {
    return temporal.NewApplicationError(
        fmt.Sprintf("order validation failed: %v", result.Errors),
        "OrderValidationFailed",  // Error type for categorization
        result.Errors,             // Optional details
    )
}
```

**When to use**:
- ✅ Validation failures (invalid order, invalid SKU, negative quantity)
- ✅ Business rule violations (insufficient inventory, order already cancelled)
- ✅ Authorization/permission denied errors
- ✅ Resource not found (404) - permanent failure
- ✅ Duplicate operations (order already exists)
- ✅ Configuration errors (missing required settings)

### Use Regular `fmt.Errorf` for System Errors (RETRYABLE)

System errors are transient and may succeed on retry:

```go
// Example: Network timeout
if err != nil {
    return fmt.Errorf("failed to call order-service: %w", err)
}
```

**When to use**:
- ✅ Network timeouts
- ✅ Database connection failures
- ✅ Temporary service unavailability (503)
- ✅ Rate limiting (429) - will succeed later
- ✅ Deadlocks/conflicts (can retry)
- ✅ Transient cloud provider errors

## Activity-Specific Examples

### OrderActivities

```go
// ValidateOrder - Use ApplicationError for invalid orders
func (a *OrderActivities) ValidateOrder(ctx context.Context, input OrderInput) (bool, error) {
    result, err := a.clients.ValidateOrder(ctx, input.OrderID)
    if err != nil {
        // System error - retry
        return false, fmt.Errorf("order validation service error: %w", err)
    }

    if !result.Valid {
        // Business error - don't retry
        return false, temporal.NewApplicationError(
            fmt.Sprintf("invalid order: %v", result.Errors),
            "OrderValidationFailed",
            result.Errors,
        )
    }

    return true, nil
}
```

### InventoryActivities

```go
// ReserveInventory - Use ApplicationError for insufficient inventory
func (a *InventoryActivities) ReserveInventory(ctx context.Context, items []Item) error {
    result, err := a.clients.ReserveInventory(ctx, items)
    if err != nil {
        // System error - retry
        return fmt.Errorf("inventory service error: %w", err)
    }

    if result.InsufficientInventory {
        // Business error - inventory not available, don't retry
        return temporal.NewApplicationError(
            fmt.Sprintf("insufficient inventory for SKUs: %v", result.UnavailableSKUs),
            "InsufficientInventory",
            result.UnavailableSKUs,
        )
    }

    return nil
}
```

### PickingActivities

```go
// CreatePickTask - Distinguish between system and business errors
func (a *PickingActivities) CreatePickTask(ctx context.Context, input CreatePickTaskInput) (string, error) {
    // Validate input
    if len(input.Items) == 0 {
        // Business error - invalid input
        return "", temporal.NewApplicationError(
            "task must have at least one item",
            "InvalidPickTaskInput",
            nil,
        )
    }

    // Call service
    task, err := a.clients.CreatePickTask(ctx, &input)
    if err != nil {
        // Check if it's a business error from the service
        if isNotFoundError(err) {
            return "", temporal.NewApplicationError(
                fmt.Sprintf("location not found: %v", err),
                "LocationNotFound",
                err,
            )
        }

        // System error - retry
        return "", fmt.Errorf("failed to create pick task: %w", err)
    }

    return task.TaskID, nil
}
```

## Error Type Naming Conventions

Use PascalCase for error types to categorize failures:

- `OrderValidationFailed`
- `InsufficientInventory`
- `LocationNotFound`
- `OrderAlreadyCancelled`
- `InvalidSKU`
- `PaymentDeclined`
- `AddressInvalid`

## Workflow Error Handling

Workflows can catch and handle ApplicationErrors differently:

```go
err := workflow.ExecuteActivity(ctx, "ValidateOrder", input).Get(ctx, &result)
if err != nil {
    var applicationErr *temporal.ApplicationError
    if errors.As(err, &applicationErr) {
        // Business error - log and fail workflow
        logger.Error("Order validation failed permanently",
            "errorType", applicationErr.Type(),
            "message", applicationErr.Message())
        return nil, err  // Don't retry workflow
    }

    // System error - workflow will retry based on retry policy
    return nil, err
}
```

## Testing Error Classification

```go
func TestValidateOrder_InvalidOrder(t *testing.T) {
    // Test that business errors return ApplicationError
    result, err := activity.ValidateOrder(ctx, invalidInput)

    require.Error(t, err)

    var appErr *temporal.ApplicationError
    require.True(t, errors.As(err, &appErr), "Expected ApplicationError")
    require.Equal(t, "OrderValidationFailed", appErr.Type())
    require.False(t, result)
}
```

## Migration Checklist

When reviewing existing activities:

- [ ] Identify all error return paths
- [ ] Classify each error as business vs system
- [ ] Replace business errors with `temporal.NewApplicationError`
- [ ] Add descriptive error types
- [ ] Update tests to verify error types
- [ ] Document error scenarios in activity comments

## Common Pitfalls

### ❌ DON'T: Use ApplicationError for Transient Failures

```go
// WRONG - timeout is retryable
if err == context.DeadlineExceeded {
    return temporal.NewApplicationError("timeout", "Timeout", nil)
}
```

### ✅ DO: Let System Errors Retry

```go
// CORRECT - let Temporal retry on timeout
if err == context.DeadlineExceeded {
    return fmt.Errorf("operation timed out: %w", err)
}
```

### ❌ DON'T: Retry Business Logic Failures

```go
// WRONG - will retry even though order is fundamentally invalid
if order.Total < 0 {
    return fmt.Errorf("negative order total")
}
```

### ✅ DO: Use ApplicationError for Logic Violations

```go
// CORRECT - business rule violation, don't retry
if order.Total < 0 {
    return temporal.NewApplicationError(
        "order total cannot be negative",
        "InvalidOrderTotal",
        order.Total,
    )
}
```

## References

- [Temporal Error Handling Best Practices](https://docs.temporal.io/dev-guide/go/features#activity-errors)
- [ApplicationError API](https://pkg.go.dev/go.temporal.io/sdk/temporal#ApplicationError)
- Retry Policies: `/orchestrator/internal/workflows/retry_policies.go`

## Review Status

| Activity File | Reviewed | Business Errors Fixed | Notes |
|--------------|----------|----------------------|-------|
| order_activities.go | ✅ | ✅ ValidateOrder | Order validation now uses ApplicationError |
| inventory_activities.go | ⏸️ | - | Needs review |
| picking_activities.go | ⏸️ | - | Needs review |
| consolidation_activities.go | ⏸️ | - | Needs review |
| packing_activities.go | ⏸️ | - | Needs review |
| shipping_activities.go | ⏸️ | - | Needs review |
| routing_activities.go | ⏸️ | - | Needs review |
