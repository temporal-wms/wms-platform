package projections

import (
	"context"
	"log/slog"
	"time"

	"github.com/wms-platform/receiving-service/internal/domain"
)

// ShipmentProjector handles domain events and updates the shipment list projection
type ShipmentProjector struct {
	projectionRepo ShipmentListProjectionRepository
	shipmentRepo   domain.InboundShipmentRepository
	logger         *slog.Logger
}

// NewShipmentProjector creates a new shipment projector
func NewShipmentProjector(
	projectionRepo ShipmentListProjectionRepository,
	shipmentRepo domain.InboundShipmentRepository,
	logger *slog.Logger,
) *ShipmentProjector {
	return &ShipmentProjector{
		projectionRepo: projectionRepo,
		shipmentRepo:   shipmentRepo,
		logger:         logger,
	}
}

// OnShipmentExpected handles ShipmentExpectedEvent
func (p *ShipmentProjector) OnShipmentExpected(ctx context.Context, event *domain.ShipmentExpectedEvent) error {
	// Fetch the full shipment aggregate to build the initial projection
	shipment, err := p.shipmentRepo.FindByID(ctx, event.ShipmentID)
	if err != nil || shipment == nil {
		p.logger.Error("Failed to find shipment for projection", "shipmentId", event.ShipmentID, "error", err)
		return err
	}

	now := time.Now().UTC()
	isLate := now.After(event.ExpectedArrival)

	// Create initial projection
	projection := &ShipmentListProjection{
		ShipmentID:         shipment.ShipmentID,
		ASNID:              shipment.ASNID,
		SupplierID:         shipment.SupplierID,
		SupplierName:       shipment.SupplierName,
		Status:             string(shipment.Status),
		TotalItemsExpected: shipment.TotalItemsExpected(),
		TotalItemsReceived: 0,
		TotalDamaged:       0,
		DiscrepancyCount:   0,
		ReceivingProgress:  0,
		ExpectedArrival:    shipment.ExpectedArrival,
		CreatedAt:          shipment.CreatedAt,
		UpdatedAt:          now,
		IsOnTime:           !isLate,
		IsLate:             isLate,
		HasIssues:          false,
	}

	if err := p.projectionRepo.Upsert(ctx, projection); err != nil {
		p.logger.Error("Failed to upsert shipment projection", "shipmentId", event.ShipmentID, "error", err)
		return err
	}

	p.logger.Info("Shipment projection created", "shipmentId", event.ShipmentID)
	return nil
}

// OnShipmentArrived handles ShipmentArrivedEvent
func (p *ShipmentProjector) OnShipmentArrived(ctx context.Context, event *domain.ShipmentArrivedEvent) error {
	updates := map[string]interface{}{
		"status":    "arrived",
		"dockId":    event.DockID,
		"arrivedAt": event.ArrivedAt,
		"isOnTime":  event.IsOnTime,
		"isLate":    !event.IsOnTime,
	}

	if err := p.projectionRepo.UpdateFields(ctx, event.ShipmentID, updates); err != nil {
		p.logger.Error("Failed to update shipment projection", "shipmentId", event.ShipmentID, "error", err)
		return err
	}

	p.logger.Info("Shipment projection updated (arrived)", "shipmentId", event.ShipmentID, "dockId", event.DockID)
	return nil
}

// OnItemReceived handles ItemReceivedEvent
func (p *ShipmentProjector) OnItemReceived(ctx context.Context, event *domain.ItemReceivedEvent) error {
	// Get current projection to update counts
	projection, err := p.projectionRepo.FindByID(ctx, event.ShipmentID)
	if err != nil || projection == nil {
		p.logger.Error("Failed to find projection", "shipmentId", event.ShipmentID, "error", err)
		return err
	}

	newReceived := projection.TotalItemsReceived + event.Quantity
	newDamaged := projection.TotalDamaged
	if event.Condition == "damaged" {
		newDamaged += event.Quantity
	}

	// Calculate progress
	progress := float64(0)
	if projection.TotalItemsExpected > 0 {
		progress = float64(newReceived) / float64(projection.TotalItemsExpected) * 100
	}

	updates := map[string]interface{}{
		"status":             "receiving",
		"totalItemsReceived": newReceived,
		"totalDamaged":       newDamaged,
		"receivingProgress":  progress,
		"hasIssues":          newDamaged > 0 || projection.DiscrepancyCount > 0,
	}

	if err := p.projectionRepo.UpdateFields(ctx, event.ShipmentID, updates); err != nil {
		p.logger.Error("Failed to update shipment projection", "shipmentId", event.ShipmentID, "error", err)
		return err
	}

	p.logger.Info("Shipment projection updated (item received)",
		"shipmentId", event.ShipmentID,
		"sku", event.SKU,
		"progress", progress,
	)
	return nil
}

// OnReceivingCompleted handles ReceivingCompletedEvent
func (p *ShipmentProjector) OnReceivingCompleted(ctx context.Context, event *domain.ReceivingCompletedEvent) error {
	updates := map[string]interface{}{
		"status":             "completed",
		"totalItemsReceived": event.TotalItemsReceived,
		"totalDamaged":       event.TotalDamaged,
		"discrepancyCount":   event.DiscrepancyCount,
		"completedAt":        event.CompletedAt,
		"receivingProgress":  100,
		"hasIssues":          event.TotalDamaged > 0 || event.DiscrepancyCount > 0,
	}

	if err := p.projectionRepo.UpdateFields(ctx, event.ShipmentID, updates); err != nil {
		p.logger.Error("Failed to update shipment projection", "shipmentId", event.ShipmentID, "error", err)
		return err
	}

	p.logger.Info("Shipment projection updated (completed)", "shipmentId", event.ShipmentID)
	return nil
}

// OnReceivingDiscrepancy handles ReceivingDiscrepancyEvent
func (p *ShipmentProjector) OnReceivingDiscrepancy(ctx context.Context, event *domain.ReceivingDiscrepancyEvent) error {
	// Get current projection to increment discrepancy count
	projection, err := p.projectionRepo.FindByID(ctx, event.ShipmentID)
	if err != nil || projection == nil {
		p.logger.Error("Failed to find projection", "shipmentId", event.ShipmentID, "error", err)
		return err
	}

	updates := map[string]interface{}{
		"discrepancyCount": projection.DiscrepancyCount + 1,
		"hasIssues":        true,
	}

	if err := p.projectionRepo.UpdateFields(ctx, event.ShipmentID, updates); err != nil {
		p.logger.Error("Failed to update shipment projection", "shipmentId", event.ShipmentID, "error", err)
		return err
	}

	p.logger.Info("Shipment projection updated (discrepancy)",
		"shipmentId", event.ShipmentID,
		"sku", event.SKU,
		"type", event.DiscrepancyType,
	)
	return nil
}

// RebuildProjection rebuilds a projection from the current aggregate state
func (p *ShipmentProjector) RebuildProjection(ctx context.Context, shipmentID string) error {
	shipment, err := p.shipmentRepo.FindByID(ctx, shipmentID)
	if err != nil || shipment == nil {
		return err
	}

	now := time.Now().UTC()
	isLate := shipment.ArrivedAt == nil && now.After(shipment.ExpectedArrival)
	if shipment.ArrivedAt != nil {
		isLate = shipment.ArrivedAt.After(shipment.ExpectedArrival)
	}

	progress := float64(0)
	totalExpected := shipment.TotalItemsExpected()
	totalReceived := shipment.TotalItemsReceived()
	if totalExpected > 0 {
		progress = float64(totalReceived) / float64(totalExpected) * 100
	}

	projection := &ShipmentListProjection{
		ShipmentID:         shipment.ShipmentID,
		ASNID:              shipment.ASNID,
		SupplierID:         shipment.SupplierID,
		SupplierName:       shipment.SupplierName,
		Status:             string(shipment.Status),
		DockID:             shipment.DockID,
		TotalItemsExpected: totalExpected,
		TotalItemsReceived: totalReceived,
		TotalDamaged:       shipment.TotalDamaged(),
		DiscrepancyCount:   len(shipment.Discrepancies),
		ReceivingProgress:  progress,
		ExpectedArrival:    shipment.ExpectedArrival,
		ArrivedAt:          shipment.ArrivedAt,
		CompletedAt:        shipment.CompletedAt,
		CreatedAt:          shipment.CreatedAt,
		UpdatedAt:          now,
		IsOnTime:           !isLate,
		IsLate:             isLate,
		HasIssues:          shipment.TotalDamaged() > 0 || len(shipment.Discrepancies) > 0,
	}

	return p.projectionRepo.Upsert(ctx, projection)
}
