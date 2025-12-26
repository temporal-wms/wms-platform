package activities

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"
)

// ReleaseInventoryReservation releases inventory reservations for an order
func (a *InventoryActivities) ReleaseInventoryReservation(ctx context.Context, orderID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Releasing inventory reservation", "orderId", orderID)

	err := a.clients.ReleaseInventoryReservation(ctx, orderID)
	if err != nil {
		logger.Error("Failed to release inventory reservation", "orderId", orderID, "error", err)
		return fmt.Errorf("failed to release inventory reservation: %w", err)
	}

	logger.Info("Inventory reservation released successfully", "orderId", orderID)
	return nil
}
