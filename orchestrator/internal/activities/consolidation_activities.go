package activities

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// CreateConsolidationUnit creates a consolidation unit for picked items
func (a *ConsolidationActivities) CreateConsolidationUnit(ctx context.Context, input map[string]interface{}) (string, error) {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	waveID, _ := input["waveId"].(string)
	pickedItemsRaw, _ := input["pickedItems"].([]interface{})

	logger.Info("Creating consolidation unit", "orderId", orderID, "waveId", waveID)

	// Convert picked items to expected items format
	expectedItems := make([]clients.ExpectedItem, 0)
	for _, itemRaw := range pickedItemsRaw {
		if item, ok := itemRaw.(map[string]interface{}); ok {
			sku, _ := item["sku"].(string)
			quantity, _ := item["quantity"].(float64)
			toteID, _ := item["toteId"].(string)
			expectedItems = append(expectedItems, clients.ExpectedItem{
				SKU:          sku,
				Quantity:     int(quantity),
				SourceToteID: toteID,
			})
		}
	}

	// Generate consolidation ID
	consolidationID := "CN-" + uuid.New().String()[:8]

	// Call consolidation-service to create unit
	unit, err := a.clients.CreateConsolidation(ctx, &clients.CreateConsolidationRequest{
		ConsolidationID: consolidationID,
		OrderID:         orderID,
		WaveID:          waveID,
		Strategy:        "fifo",
		Items:           expectedItems,
	})
	if err != nil {
		logger.Error("Failed to create consolidation unit", "orderId", orderID, "error", err)
		return "", fmt.Errorf("failed to create consolidation unit: %w", err)
	}

	logger.Info("Consolidation unit created successfully",
		"orderId", orderID,
		"consolidationId", unit.ConsolidationID,
		"itemCount", len(expectedItems),
	)

	return unit.ConsolidationID, nil
}

// ConsolidateItems consolidates picked items from various totes
func (a *ConsolidationActivities) ConsolidateItems(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	consolidationID, _ := input["consolidationId"].(string)
	pickedItemsRaw, _ := input["pickedItems"].([]interface{})

	logger.Info("Consolidating items", "consolidationId", consolidationID, "itemCount", len(pickedItemsRaw))

	// Consolidate each item
	for i, itemRaw := range pickedItemsRaw {
		// Record heartbeat for long-running consolidation operations
		// This allows Temporal to detect if the activity is still making progress
		activity.RecordHeartbeat(ctx, fmt.Sprintf("Processing item %d/%d", i+1, len(pickedItemsRaw)))

		if item, ok := itemRaw.(map[string]interface{}); ok {
			sku, _ := item["sku"].(string)
			quantity, _ := item["quantity"].(float64)
			toteID, _ := item["toteId"].(string)

			err := a.clients.ConsolidateItem(ctx, consolidationID, &clients.ConsolidateItemRequest{
				SKU:          sku,
				Quantity:     int(quantity),
				SourceToteID: toteID,
				VerifiedBy:   "system",
			})
			if err != nil {
				logger.Error("Failed to consolidate item", "consolidationId", consolidationID, "sku", sku, "error", err)
				return fmt.Errorf("failed to consolidate item %s: %w", sku, err)
			}

			logger.Info("Item consolidated", "consolidationId", consolidationID, "sku", sku, "quantity", int(quantity), "progress", fmt.Sprintf("%d/%d", i+1, len(pickedItemsRaw)))
		}
	}

	logger.Info("All items consolidated successfully", "consolidationId", consolidationID)
	return nil
}

// VerifyConsolidation verifies all items have been consolidated
func (a *ConsolidationActivities) VerifyConsolidation(ctx context.Context, consolidationID string) (bool, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("Verifying consolidation", "consolidationId", consolidationID)

	// Get consolidation unit to verify
	unit, err := a.clients.GetConsolidation(ctx, consolidationID)
	if err != nil {
		logger.Error("Failed to get consolidation unit", "consolidationId", consolidationID, "error", err)
		return false, fmt.Errorf("failed to get consolidation unit: %w", err)
	}

	// Verify all expected items are consolidated
	expectedCount := len(unit.ExpectedItems)
	consolidatedCount := len(unit.ConsolidatedItems)

	verified := consolidatedCount >= expectedCount
	if !verified {
		logger.Warn("Consolidation incomplete",
			"consolidationId", consolidationID,
			"expected", expectedCount,
			"consolidated", consolidatedCount,
		)
	} else {
		logger.Info("Consolidation verified",
			"consolidationId", consolidationID,
			"itemCount", consolidatedCount,
		)
	}

	return verified, nil
}

// CompleteConsolidation marks the consolidation as complete
func (a *ConsolidationActivities) CompleteConsolidation(ctx context.Context, consolidationID string) error {
	logger := activity.GetLogger(ctx)

	logger.Info("Completing consolidation", "consolidationId", consolidationID)

	_, err := a.clients.CompleteConsolidation(ctx, consolidationID)
	if err != nil {
		// Handle idempotency: if already complete, treat as success
		errStr := err.Error()
		if strings.Contains(errStr, "already complete") {
			logger.Info("Consolidation already complete (idempotent)", "consolidationId", consolidationID)
			return nil
		}
		logger.Error("Failed to complete consolidation", "consolidationId", consolidationID, "error", err)
		return fmt.Errorf("failed to complete consolidation: %w", err)
	}

	logger.Info("Consolidation completed successfully", "consolidationId", consolidationID)
	return nil
}
