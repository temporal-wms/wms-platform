package projections

import (
	"context"
	"time"

	"github.com/wms-platform/inventory-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
)

// InventoryProjector handles domain events and updates the inventory list projection
// This is the "event handler" in CQRS that keeps the read model in sync
type InventoryProjector struct {
	projectionRepo InventoryListProjectionRepository
	inventoryRepo  domain.InventoryRepository // For fetching full aggregate when needed
	logger         *logging.Logger
}

// NewInventoryProjector creates a new inventory projector
func NewInventoryProjector(
	projectionRepo InventoryListProjectionRepository,
	inventoryRepo domain.InventoryRepository,
	logger *logging.Logger,
) *InventoryProjector {
	return &InventoryProjector{
		projectionRepo: projectionRepo,
		inventoryRepo:  inventoryRepo,
		logger:         logger,
	}
}

// OnInventoryReceived handles InventoryReceivedEvent
func (p *InventoryProjector) OnInventoryReceived(ctx context.Context, event *domain.InventoryReceivedEvent) error {
	// Fetch the full inventory aggregate to build/update the projection
	item, err := p.inventoryRepo.FindBySKU(ctx, event.SKU)
	if err != nil || item == nil {
		p.logger.Error("Failed to find inventory for projection", "sku", event.SKU, "error", err)
		return err
	}

	// Check if projection exists
	existing, _ := p.projectionRepo.FindBySKU(ctx, event.SKU)

	if existing == nil {
		// Create new projection
		projection := p.buildProjectionFromAggregate(item)
		projection.LastReceived = &event.ReceivedAt
		return p.projectionRepo.Upsert(ctx, projection)
	}

	// Update existing projection
	updates := map[string]interface{}{
		"totalQuantity":     item.TotalQuantity,
		"reservedQuantity":  item.ReservedQuantity,
		"availableQuantity": item.AvailableQuantity,
		"isLowStock":        item.AvailableQuantity <= item.ReorderPoint,
		"isOutOfStock":      item.AvailableQuantity == 0,
		"lastReceived":      event.ReceivedAt,
		"locationCount":     len(item.Locations),
		"availableLocations": p.extractAvailableLocations(item),
		"primaryLocation":    p.findPrimaryLocation(item),
	}

	return p.projectionRepo.UpdateFields(ctx, event.SKU, updates)
}

// OnInventoryAdjusted handles InventoryAdjustedEvent
func (p *InventoryProjector) OnInventoryAdjusted(ctx context.Context, event *domain.InventoryAdjustedEvent) error {
	// Fetch the full inventory aggregate
	item, err := p.inventoryRepo.FindBySKU(ctx, event.SKU)
	if err != nil || item == nil {
		p.logger.Error("Failed to find inventory for projection", "sku", event.SKU, "error", err)
		return err
	}

	// Update projection
	updates := map[string]interface{}{
		"totalQuantity":      item.TotalQuantity,
		"availableQuantity":  item.AvailableQuantity,
		"isLowStock":         item.AvailableQuantity <= item.ReorderPoint,
		"isOutOfStock":       item.AvailableQuantity == 0,
		"lastAdjusted":       event.AdjustedAt,
		"availableLocations": p.extractAvailableLocations(item),
	}

	return p.projectionRepo.UpdateFields(ctx, event.SKU, updates)
}

// OnLowStockAlert handles LowStockAlertEvent
func (p *InventoryProjector) OnLowStockAlert(ctx context.Context, event *domain.LowStockAlertEvent) error {
	// Update low stock flag
	updates := map[string]interface{}{
		"isLowStock":         true,
		"availableQuantity":  event.CurrentQuantity,
	}

	return p.projectionRepo.UpdateFields(ctx, event.SKU, updates)
}

// OnInventoryReserved handles inventory reservation (not a domain event yet, but useful)
func (p *InventoryProjector) OnInventoryReserved(ctx context.Context, sku, orderID string) error {
	// Fetch the full inventory aggregate
	item, err := p.inventoryRepo.FindBySKU(ctx, sku)
	if err != nil || item == nil {
		p.logger.Error("Failed to find inventory for projection", "sku", sku, "error", err)
		return err
	}

	// Extract active reservation order IDs
	reservedOrders := make([]string, 0)
	activeReservations := 0
	for _, res := range item.Reservations {
		if res.Status == "active" {
			activeReservations++
			reservedOrders = append(reservedOrders, res.OrderID)
		}
	}

	// Update projection
	updates := map[string]interface{}{
		"reservedQuantity":    item.ReservedQuantity,
		"availableQuantity":   item.AvailableQuantity,
		"isLowStock":          item.AvailableQuantity <= item.ReorderPoint,
		"isOutOfStock":        item.AvailableQuantity == 0,
		"activeReservations":  activeReservations,
		"reservedOrders":      reservedOrders,
		"availableLocations":  p.extractAvailableLocations(item),
	}

	return p.projectionRepo.UpdateFields(ctx, sku, updates)
}

// OnInventoryPicked handles inventory pick (update after picking)
func (p *InventoryProjector) OnInventoryPicked(ctx context.Context, sku string) error {
	// Fetch the full inventory aggregate
	item, err := p.inventoryRepo.FindBySKU(ctx, sku)
	if err != nil || item == nil {
		p.logger.Error("Failed to find inventory for projection", "sku", sku, "error", err)
		return err
	}

	now := time.Now()

	// Extract active reservation order IDs
	reservedOrders := make([]string, 0)
	activeReservations := 0
	for _, res := range item.Reservations {
		if res.Status == "active" {
			activeReservations++
			reservedOrders = append(reservedOrders, res.OrderID)
		}
	}

	// Update projection
	updates := map[string]interface{}{
		"totalQuantity":       item.TotalQuantity,
		"reservedQuantity":    item.ReservedQuantity,
		"availableQuantity":   item.AvailableQuantity,
		"isLowStock":          item.AvailableQuantity <= item.ReorderPoint,
		"isOutOfStock":        item.AvailableQuantity == 0,
		"activeReservations":  activeReservations,
		"reservedOrders":      reservedOrders,
		"lastPicked":          now,
		"locationCount":       len(item.Locations),
		"availableLocations":  p.extractAvailableLocations(item),
		"primaryLocation":     p.findPrimaryLocation(item),
	}

	return p.projectionRepo.UpdateFields(ctx, sku, updates)
}

// OnStockShortage handles StockShortageEvent
func (p *InventoryProjector) OnStockShortage(ctx context.Context, event *domain.StockShortageEvent) error {
	// Fetch the full inventory aggregate
	item, err := p.inventoryRepo.FindBySKU(ctx, event.SKU)
	if err != nil || item == nil {
		p.logger.Error("Failed to find inventory for projection", "sku", event.SKU, "error", err)
		return err
	}

	now := time.Now()

	// Update projection with adjusted quantities
	updates := map[string]interface{}{
		"totalQuantity":      item.TotalQuantity,
		"availableQuantity":  item.AvailableQuantity,
		"reservedQuantity":   item.ReservedQuantity,
		"isLowStock":         item.AvailableQuantity <= item.ReorderPoint,
		"isOutOfStock":       item.AvailableQuantity == 0,
		"lastShortage":       now,
		"availableLocations": p.extractAvailableLocations(item),
		"primaryLocation":    p.findPrimaryLocation(item),
	}

	return p.projectionRepo.UpdateFields(ctx, event.SKU, updates)
}

// OnInventoryDiscrepancy handles InventoryDiscrepancyEvent
func (p *InventoryProjector) OnInventoryDiscrepancy(ctx context.Context, event *domain.InventoryDiscrepancyEvent) error {
	// Fetch the full inventory aggregate
	item, err := p.inventoryRepo.FindBySKU(ctx, event.SKU)
	if err != nil || item == nil {
		p.logger.Error("Failed to find inventory for projection", "sku", event.SKU, "error", err)
		return err
	}

	now := time.Now()

	// Update projection with adjusted quantities
	updates := map[string]interface{}{
		"totalQuantity":       item.TotalQuantity,
		"availableQuantity":   item.AvailableQuantity,
		"isLowStock":          item.AvailableQuantity <= item.ReorderPoint,
		"isOutOfStock":        item.AvailableQuantity == 0,
		"lastDiscrepancy":     now,
		"lastDiscrepancyType": event.DiscrepancyType,
		"availableLocations":  p.extractAvailableLocations(item),
	}

	return p.projectionRepo.UpdateFields(ctx, event.SKU, updates)
}

// buildProjectionFromAggregate creates a new projection from a full aggregate
func (p *InventoryProjector) buildProjectionFromAggregate(item *domain.InventoryItem) *InventoryListProjection {
	// Extract active reservation order IDs
	reservedOrders := make([]string, 0)
	activeReservations := 0
	for _, res := range item.Reservations {
		if res.Status == "active" {
			activeReservations++
			reservedOrders = append(reservedOrders, res.OrderID)
		}
	}

	projection := &InventoryListProjection{
		SKU:                item.SKU,
		ProductName:        item.ProductName,
		TotalQuantity:      item.TotalQuantity,
		ReservedQuantity:   item.ReservedQuantity,
		AvailableQuantity:  item.AvailableQuantity,
		ReorderPoint:       item.ReorderPoint,
		ReorderQuantity:    item.ReorderQuantity,
		IsLowStock:         item.AvailableQuantity <= item.ReorderPoint,
		IsOutOfStock:       item.AvailableQuantity == 0,
		LocationCount:      len(item.Locations),
		PrimaryLocation:    p.findPrimaryLocation(item),
		AvailableLocations: p.extractAvailableLocations(item),
		ActiveReservations: activeReservations,
		ReservedOrders:     reservedOrders,
		LastCycleCount:     item.LastCycleCount,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}

	return projection
}

// extractAvailableLocations extracts location IDs with available stock
func (p *InventoryProjector) extractAvailableLocations(item *domain.InventoryItem) []string {
	locations := make([]string, 0)
	for _, loc := range item.Locations {
		if loc.Available > 0 {
			locations = append(locations, loc.LocationID)
		}
	}
	return locations
}

// findPrimaryLocation finds the location with the most available stock
func (p *InventoryProjector) findPrimaryLocation(item *domain.InventoryItem) string {
	if len(item.Locations) == 0 {
		return ""
	}

	maxLocation := ""
	maxQty := 0
	for _, loc := range item.Locations {
		if loc.Available > maxQty {
			maxQty = loc.Available
			maxLocation = loc.LocationID
		}
	}

	return maxLocation
}
