package activities

import (
	"context"
	"fmt"

	"github.com/wms-platform/orchestrator/internal/workflows"
	"go.temporal.io/sdk/activity"
)

// ValidateOrder validates an order by calling order-service
func (a *OrderActivities) ValidateOrder(ctx context.Context, input workflows.OrderFulfillmentInput) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Validating order", "orderId", input.OrderID)

	// Call order-service to validate the order
	result, err := a.clients.ValidateOrder(ctx, input.OrderID)
	if err != nil {
		logger.Error("Failed to validate order", "orderId", input.OrderID, "error", err)
		return false, fmt.Errorf("order validation failed: %w", err)
	}

	if !result.Valid {
		logger.Warn("Order validation failed", "orderId", input.OrderID, "errors", result.Errors)
		return false, fmt.Errorf("order validation failed: %v", result.Errors)
	}

	logger.Info("Order validated successfully", "orderId", input.OrderID)
	return true, nil
}

// CancelOrder cancels an order
func (a *OrderActivities) CancelOrder(ctx context.Context, orderID string, reason string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Cancelling order", "orderId", orderID, "reason", reason)

	err := a.clients.CancelOrder(ctx, orderID, reason)
	if err != nil {
		logger.Error("Failed to cancel order", "orderId", orderID, "error", err)
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	logger.Info("Order cancelled successfully", "orderId", orderID)
	return nil
}

// NotifyCustomerCancellation notifies the customer about order cancellation
func (a *OrderActivities) NotifyCustomerCancellation(ctx context.Context, orderID string, reason string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Notifying customer about cancellation", "orderId", orderID, "reason", reason)

	// Get order details to find customer info
	order, err := a.clients.GetOrder(ctx, orderID)
	if err != nil {
		logger.Warn("Failed to get order for notification", "orderId", orderID, "error", err)
		// Don't fail the activity - notification is best-effort
		return nil
	}

	// In a real implementation, this would call a notification service
	// For now, we just log the notification
	logger.Info("Customer notified about cancellation",
		"orderId", orderID,
		"customerId", order.CustomerID,
		"reason", reason,
	)

	return nil
}
