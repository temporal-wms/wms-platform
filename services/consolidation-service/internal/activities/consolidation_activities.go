package activities

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/wms-platform/consolidation-service/internal/domain"
	"go.temporal.io/sdk/activity"
)

// ConsolidationActivities contains activities for the consolidation workflow
type ConsolidationActivities struct {
	repo   domain.ConsolidationRepository
	logger *slog.Logger
}

// NewConsolidationActivities creates a new ConsolidationActivities instance
func NewConsolidationActivities(repo domain.ConsolidationRepository, logger *slog.Logger) *ConsolidationActivities {
	return &ConsolidationActivities{
		repo:   repo,
		logger: logger,
	}
}

// CreateConsolidationUnit creates a new consolidation unit
func (a *ConsolidationActivities) CreateConsolidationUnit(ctx context.Context, input map[string]interface{}) (string, error) {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	pickedItemsRaw, _ := input["pickedItems"].([]interface{})

	logger.Info("Creating consolidation unit", "orderId", orderID)

	// Generate consolidation ID
	consolidationID := "CONS-" + uuid.New().String()[:8]

	// Convert picked items to expected items
	items := make([]domain.ExpectedItem, 0)
	for _, itemRaw := range pickedItemsRaw {
		if item, ok := itemRaw.(map[string]interface{}); ok {
			sku, _ := item["sku"].(string)
			quantity, _ := item["quantity"].(float64)
			toteID, _ := item["toteId"].(string)

			items = append(items, domain.ExpectedItem{
				SKU:          sku,
				Quantity:     int(quantity),
				SourceToteID: toteID,
				Received:     0,
				Status:       "pending",
			})
		}
	}

	// Get waveID from context or generate a placeholder
	waveID := "WAVE-" + uuid.New().String()[:8]

	// Create the consolidation unit
	unit, err := domain.NewConsolidationUnit(consolidationID, orderID, waveID, domain.StrategyOrderBased, items)
	if err != nil {
		logger.Error("Failed to create consolidation unit", "error", err)
		return "", fmt.Errorf("failed to create consolidation unit: %w", err)
	}

	// Save to repository
	if err := a.repo.Save(ctx, unit); err != nil {
		logger.Error("Failed to save consolidation unit", "error", err)
		return "", fmt.Errorf("failed to save consolidation unit: %w", err)
	}

	logger.Info("Consolidation unit created", "consolidationId", consolidationID, "itemCount", len(items))
	return consolidationID, nil
}

// AssignStation assigns a station to a consolidation unit
func (a *ConsolidationActivities) AssignStation(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	consolidationID, _ := input["consolidationId"].(string)
	station, _ := input["station"].(string)
	workerID, _ := input["workerId"].(string)
	destinationBin, _ := input["destinationBin"].(string)

	logger.Info("Assigning station", "consolidationId", consolidationID, "station", station)

	unit, err := a.repo.FindByID(ctx, consolidationID)
	if err != nil {
		return fmt.Errorf("failed to find consolidation unit: %w", err)
	}

	if unit == nil {
		return fmt.Errorf("consolidation unit not found: %s", consolidationID)
	}

	if err := unit.AssignStation(station, workerID, destinationBin); err != nil {
		return fmt.Errorf("failed to assign station: %w", err)
	}

	if err := a.repo.Save(ctx, unit); err != nil {
		return fmt.Errorf("failed to save consolidation unit: %w", err)
	}

	logger.Info("Station assigned", "consolidationId", consolidationID, "station", station)
	return nil
}

// ConsolidateItem consolidates an item
func (a *ConsolidationActivities) ConsolidateItem(ctx context.Context, consolidationID, sku, sourceToteID, verifiedBy string, quantity int) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Consolidating item", "consolidationId", consolidationID, "sku", sku, "quantity", quantity)

	unit, err := a.repo.FindByID(ctx, consolidationID)
	if err != nil {
		return fmt.Errorf("failed to find consolidation unit: %w", err)
	}

	if unit == nil {
		return fmt.Errorf("consolidation unit not found: %s", consolidationID)
	}

	if err := unit.ConsolidateItem(sku, quantity, sourceToteID, verifiedBy); err != nil {
		return fmt.Errorf("failed to consolidate item: %w", err)
	}

	if err := a.repo.Save(ctx, unit); err != nil {
		return fmt.Errorf("failed to save consolidation unit: %w", err)
	}

	logger.Info("Item consolidated", "consolidationId", consolidationID, "sku", sku, "quantity", quantity)
	return nil
}

// CompleteConsolidation marks a consolidation unit as complete
func (a *ConsolidationActivities) CompleteConsolidation(ctx context.Context, consolidationID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Completing consolidation", "consolidationId", consolidationID)

	unit, err := a.repo.FindByID(ctx, consolidationID)
	if err != nil {
		return fmt.Errorf("failed to find consolidation unit: %w", err)
	}

	if unit == nil {
		return fmt.Errorf("consolidation unit not found: %s", consolidationID)
	}

	// If not already complete, complete it
	if unit.Status != domain.ConsolidationStatusCompleted {
		if err := unit.Complete(); err != nil {
			logger.Warn("Failed to complete consolidation", "consolidationId", consolidationID, "error", err)
		}
	}

	if err := a.repo.Save(ctx, unit); err != nil {
		return fmt.Errorf("failed to save consolidation unit: %w", err)
	}

	logger.Info("Consolidation completed", "consolidationId", consolidationID)
	return nil
}

// MarkShort marks items as short
func (a *ConsolidationActivities) MarkShort(ctx context.Context, consolidationID, sku, sourceToteID, reason string, shortQty int) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Marking item short", "consolidationId", consolidationID, "sku", sku)

	unit, err := a.repo.FindByID(ctx, consolidationID)
	if err != nil {
		return fmt.Errorf("failed to find consolidation unit: %w", err)
	}

	if unit == nil {
		return fmt.Errorf("consolidation unit not found: %s", consolidationID)
	}

	if err := unit.MarkShort(sku, sourceToteID, shortQty, reason); err != nil {
		return fmt.Errorf("failed to mark short: %w", err)
	}

	if err := a.repo.Save(ctx, unit); err != nil {
		return fmt.Errorf("failed to save consolidation unit: %w", err)
	}

	logger.Info("Item marked short", "consolidationId", consolidationID, "sku", sku)
	return nil
}
