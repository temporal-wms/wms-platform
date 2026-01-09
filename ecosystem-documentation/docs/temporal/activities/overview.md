---
sidebar_position: 1
slug: /temporal/activities/overview
---

# Activities Overview

Activities are the building blocks of Temporal workflows, executing the actual business logic.

## Activity Patterns

### Activity Structure

All activities in the orchestrator follow this pattern:

```go
type XxxActivities struct {
    clients *ServiceClients  // HTTP clients to downstream services
    logger  *slog.Logger     // Structured logging
}

func NewXxxActivities(clients *ServiceClients, logger *slog.Logger) *XxxActivities {
    return &XxxActivities{clients: clients, logger: logger}
}
```

### Activity Categories

| Category | Struct | Purpose |
|----------|--------|---------|
| [Order Activities](./order-activities) | `OrderActivities` | Order lifecycle management |
| [Inventory Activities](./inventory-activities) | `InventoryActivities` | Inventory operations |
| [Picking Activities](./picking-activities) | `PickingActivities` | Picking task management |
| [Packing Activities](./packing-activities) | `PackingActivities` | Packing task management |
| [Consolidation Activities](./consolidation-activities) | `ConsolidationActivities` | Consolidation operations |
| [Shipping Activities](./shipping-activities) | `ShippingActivities` | Shipping and SLAM |
| [Receiving Activities](./receiving-activities) | `ReceivingActivities` | Inbound receiving |
| [Sortation Activities](./sortation-activities) | `SortationActivities` | Sortation and batching |
| [SLAM Activities](./slam-activities) | `SLAMActivities` | Scan, Label, Apply, Manifest |
| [Unit Activities](./unit-activities) | `UnitActivities` | Unit-level tracking |
| [Process Path Activities](./process-path-activities) | `ProcessPathActivities` | Process path routing |

## Error Handling

### Retryable vs Non-Retryable Errors

Activities use Temporal's retry mechanism with non-retryable error types:

```go
// Non-retryable business error
return temporal.NewApplicationError(
    fmt.Sprintf("order validation failed: %v", result.Errors),
    "OrderValidationFailed",  // Error type
    result.Errors,             // Additional details
)
```

### Common Non-Retryable Error Types

| Error Type | Description | Example |
|------------|-------------|---------|
| `ValidationError` | Input validation failed | Invalid order ID format |
| `NotFoundError` | Resource not found | Order does not exist |
| `ConflictError` | State conflict | Order already cancelled |
| `BusinessRuleError` | Business rule violation | Insufficient inventory |

### Retry Policy Configuration

Standard retry policy used across activities:

```go
RetryPolicy: &temporal.RetryPolicy{
    InitialInterval:    time.Second,
    BackoffCoefficient: 2.0,
    MaximumInterval:    time.Minute,
    MaximumAttempts:    3,
    NonRetryableErrorTypes: []string{
        "ValidationError",
        "NotFoundError",
        "ConflictError",
    },
}
```

## Heartbeating

Long-running activities should record heartbeats:

```go
func (a *InventoryActivities) ConfirmInventoryPick(ctx context.Context, input Input) error {
    for i, item := range input.PickedItems {
        // Record heartbeat for long-running operations
        activity.RecordHeartbeat(ctx, fmt.Sprintf("Processing item %d/%d", i+1, len(items)))

        // Process item...
    }
    return nil
}
```

### Heartbeat Timeout Configuration

```go
ao := workflow.ActivityOptions{
    StartToCloseTimeout: 10 * time.Minute,
    HeartbeatTimeout:    30 * time.Second,  // Detect stuck workers
}
```

## Logging Patterns

Activities use structured logging:

```go
func (a *OrderActivities) ValidateOrder(ctx context.Context, input Input) error {
    logger := activity.GetLogger(ctx)  // Get Temporal activity logger

    logger.Info("Validating order", "orderId", input.OrderID)

    // ... business logic ...

    if err != nil {
        logger.Error("Failed to validate order",
            "orderId", input.OrderID,
            "error", err,
        )
        return err
    }

    logger.Info("Order validated successfully", "orderId", input.OrderID)
    return nil
}
```

## Input/Output Patterns

### Typed Structs

Activities use typed structs for inputs and outputs:

```go
// Input struct
type ConfirmInventoryPickInput struct {
    OrderID     string                     `json:"orderId"`
    PickedItems []ConfirmInventoryPickItem `json:"pickedItems"`
}

// Output struct
type StageInventoryOutput struct {
    StagedItems   []StagedItem `json:"stagedItems"`
    FailedItems   []string     `json:"failedItems,omitempty"`
    AllocationIDs []string     `json:"allocationIds"`
}
```

### Map-Based Inputs

Some activities accept `map[string]interface{}` for flexibility:

```go
func (a *PickingActivities) CreatePickTask(ctx context.Context, input map[string]interface{}) (string, error) {
    orderID, _ := input["orderId"].(string)
    waveID, _ := input["waveId"].(string)
    // ...
}
```

## Service Client Pattern

Activities call downstream services via HTTP clients:

```go
type ServiceClients struct {
    *clients.ServiceClients  // Wrapped HTTP clients
}

func (a *OrderActivities) ValidateOrder(ctx context.Context, input Input) (bool, error) {
    // Call order-service via HTTP
    result, err := a.clients.ValidateOrder(ctx, input.OrderID)
    // ...
}
```

## Registration

Activities are registered with the Temporal worker:

```go
// Create activity instances
orderActivities := activities.NewOrderActivities(clients, logger)
inventoryActivities := activities.NewInventoryActivities(clients, logger)

// Register with worker
worker.RegisterActivity(orderActivities.ValidateOrder)
worker.RegisterActivity(orderActivities.CancelOrder)
worker.RegisterActivity(inventoryActivities.ConfirmInventoryPick)
// ...
```

## Best Practices

1. **Idempotency**: Activities should be idempotent - safe to retry
2. **Heartbeating**: Use heartbeats for operations > 30 seconds
3. **Logging**: Log at start and end of each activity
4. **Error Types**: Use non-retryable errors for business failures
5. **Timeouts**: Set appropriate ScheduleToCloseTimeout and StartToCloseTimeout
6. **Typed Inputs**: Prefer typed structs over map[string]interface{}

## Related Documentation

- [Retry Policies](../retry-policies) - Retry configuration details
- [Task Queues](../task-queues) - Task queue assignments
- [Order Fulfillment Workflow](../workflows/order-fulfillment) - Main workflow
