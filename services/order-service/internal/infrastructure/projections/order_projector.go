package projections

import (
	"context"
	"time"

	"github.com/wms-platform/services/order-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
)

// OrderProjector handles domain events and updates the order list projection
// This is the "event handler" in CQRS that keeps the read model in sync
type OrderProjector struct {
	projectionRepo OrderListProjectionRepository
	orderRepo      domain.OrderRepository // For fetching full aggregate when needed
	logger         *logging.Logger
}

// NewOrderProjector creates a new order projector
func NewOrderProjector(
	projectionRepo OrderListProjectionRepository,
	orderRepo domain.OrderRepository,
	logger *logging.Logger,
) *OrderProjector {
	return &OrderProjector{
		projectionRepo: projectionRepo,
		orderRepo:      orderRepo,
		logger:         logger,
	}
}

// OnOrderReceived handles OrderReceivedEvent
func (p *OrderProjector) OnOrderReceived(ctx context.Context, event *domain.OrderReceivedEvent) error {
	// Fetch the full order aggregate to build the initial projection
	order, err := p.orderRepo.FindByID(ctx, event.OrderID)
	if err != nil || order == nil {
		p.logger.Error("Failed to find order for projection", "orderId", event.OrderID, "error", err)
		return err
	}

	// Calculate derived fields
	daysUntilPromised := int(time.Until(order.PromisedDeliveryAt).Hours() / 24)
	isLate := time.Now().After(order.PromisedDeliveryAt) && order.Status != "delivered" && order.Status != "cancelled"
	isPriority := order.Priority == "same_day" || order.Priority == "next_day"

	// Create initial projection
	projection := &OrderListProjection{
		OrderID:           order.OrderID,
		CustomerID:        order.CustomerID,
		Status:            string(order.Status),
		Priority:          string(order.Priority),
		TotalItems:        order.TotalItems(),
		TotalWeight:       order.TotalWeight(),
		TotalValue:        calculateTotalValue(order),
		ShipToCity:        order.ShippingAddress.City,
		ShipToState:       order.ShippingAddress.State,
		ShipToZipCode:     order.ShippingAddress.ZipCode,
		ShipToCountry:     order.ShippingAddress.Country,
		ReceivedAt:        order.CreatedAt,
		PromisedDeliveryAt: order.PromisedDeliveryAt,
		DaysUntilPromised: daysUntilPromised,
		IsLate:            isLate,
		IsPriority:        isPriority,
		CreatedAt:         order.CreatedAt,
		UpdatedAt:         order.UpdatedAt,
	}

	if err := p.projectionRepo.Upsert(ctx, projection); err != nil {
		p.logger.Error("Failed to upsert order projection", "orderId", event.OrderID, "error", err)
		return err
	}

	p.logger.Info("Order projection created", "orderId", event.OrderID)
	return nil
}

// OnOrderValidated handles OrderValidatedEvent
func (p *OrderProjector) OnOrderValidated(ctx context.Context, event *domain.OrderValidatedEvent) error {
	updates := map[string]interface{}{
		"status": "validated",
	}

	if err := p.projectionRepo.UpdateFields(ctx, event.OrderID, updates); err != nil {
		p.logger.Error("Failed to update order projection", "orderId", event.OrderID, "error", err)
		return err
	}

	p.logger.Info("Order projection updated (validated)", "orderId", event.OrderID)
	return nil
}

// OnOrderAssignedToWave handles OrderAssignedToWaveEvent
func (p *OrderProjector) OnOrderAssignedToWave(ctx context.Context, event *domain.OrderAssignedToWaveEvent) error {
	updates := map[string]interface{}{
		"status": "wave_assigned",
		"waveId": event.WaveID,
		// Wave status and type would be denormalized from wave events
		// For now, we set placeholder values
		"waveStatus": "scheduled",
	}

	if err := p.projectionRepo.UpdateFields(ctx, event.OrderID, updates); err != nil {
		p.logger.Error("Failed to update order projection", "orderId", event.OrderID, "error", err)
		return err
	}

	p.logger.Info("Order projection updated (wave assigned)", "orderId", event.OrderID, "waveId", event.WaveID)
	return nil
}

// OnOrderShipped handles OrderShippedEvent
func (p *OrderProjector) OnOrderShipped(ctx context.Context, event *domain.OrderShippedEvent) error {
	updates := map[string]interface{}{
		"status":         "shipped",
		"trackingNumber": event.TrackingNumber,
		// Carrier info would come from shipping service events
	}

	if err := p.projectionRepo.UpdateFields(ctx, event.OrderID, updates); err != nil {
		p.logger.Error("Failed to update order projection", "orderId", event.OrderID, "error", err)
		return err
	}

	p.logger.Info("Order projection updated (shipped)", "orderId", event.OrderID, "trackingNumber", event.TrackingNumber)
	return nil
}

// OnOrderCancelled handles OrderCancelledEvent
func (p *OrderProjector) OnOrderCancelled(ctx context.Context, event *domain.OrderCancelledEvent) error {
	updates := map[string]interface{}{
		"status": "cancelled",
		"isLate": false, // No longer late if cancelled
	}

	if err := p.projectionRepo.UpdateFields(ctx, event.OrderID, updates); err != nil {
		p.logger.Error("Failed to update order projection", "orderId", event.OrderID, "error", err)
		return err
	}

	p.logger.Info("Order projection updated (cancelled)", "orderId", event.OrderID)
	return nil
}

// OnPickingStarted handles picking started event (from picking service)
// This would be triggered by external events via Kafka consumer
func (p *OrderProjector) OnPickingStarted(ctx context.Context, orderID, pickerID string, startedAt time.Time) error {
	updates := map[string]interface{}{
		"status":           "picking",
		"assignedPicker":   pickerID,
		"pickingStartedAt": startedAt,
	}

	if err := p.projectionRepo.UpdateFields(ctx, orderID, updates); err != nil {
		p.logger.Error("Failed to update order projection", "orderId", orderID, "error", err)
		return err
	}

	p.logger.Info("Order projection updated (picking started)", "orderId", orderID, "picker", pickerID)
	return nil
}

// OnPickingCompleted handles picking completed event
func (p *OrderProjector) OnPickingCompleted(ctx context.Context, orderID string, completedAt time.Time) error {
	updates := map[string]interface{}{
		"pickingCompletedAt": completedAt,
	}

	if err := p.projectionRepo.UpdateFields(ctx, orderID, updates); err != nil {
		p.logger.Error("Failed to update order projection", "orderId", orderID, "error", err)
		return err
	}

	p.logger.Info("Order projection updated (picking completed)", "orderId", orderID)
	return nil
}

// OnWaveReleased handles wave released event (denormalize wave status)
func (p *OrderProjector) OnWaveReleased(ctx context.Context, waveID string) error {
	// This would require a batch update for all orders with this waveID
	// For simplicity, we'll leave this as a TODO
	// In production, you'd want to:
	// 1. Find all projections with this waveID
	// 2. Update them in batch
	// 3. Or use MongoDB updateMany with filter: {waveId: waveID}
	// Example:
	// updates := map[string]interface{}{"waveStatus": "released"}
	// r.projectionRepo.collection.UpdateMany(ctx, bson.M{"waveId": waveID}, bson.M{"$set": updates})

	p.logger.Info("Wave released - projections need bulk update", "waveId", waveID)
	return nil
}

// RebuildProjection rebuilds a projection from the current aggregate state
// Useful for fixing inconsistencies or initial population
func (p *OrderProjector) RebuildProjection(ctx context.Context, orderID string) error {
	order, err := p.orderRepo.FindByID(ctx, orderID)
	if err != nil || order == nil {
		return err
	}

	// Rebuild from current state (similar to OnOrderReceived)
	daysUntilPromised := int(time.Until(order.PromisedDeliveryAt).Hours() / 24)
	isLate := time.Now().After(order.PromisedDeliveryAt) && order.Status != "delivered" && order.Status != "cancelled"
	isPriority := order.Priority == "same_day" || order.Priority == "next_day"

	projection := &OrderListProjection{
		OrderID:           order.OrderID,
		CustomerID:        order.CustomerID,
		Status:            string(order.Status),
		Priority:          string(order.Priority),
		TotalItems:        order.TotalItems(),
		TotalWeight:       order.TotalWeight(),
		TotalValue:        calculateTotalValue(order),
		WaveID:            order.WaveID,
		TrackingNumber:    order.TrackingNumber,
		ShipToCity:        order.ShippingAddress.City,
		ShipToState:       order.ShippingAddress.State,
		ShipToZipCode:     order.ShippingAddress.ZipCode,
		ShipToCountry:     order.ShippingAddress.Country,
		ReceivedAt:        order.CreatedAt,
		PromisedDeliveryAt: order.PromisedDeliveryAt,
		DaysUntilPromised: daysUntilPromised,
		IsLate:            isLate,
		IsPriority:        isPriority,
		CreatedAt:         order.CreatedAt,
		UpdatedAt:         order.UpdatedAt,
	}

	return p.projectionRepo.Upsert(ctx, projection)
}

// RebuildAllProjections rebuilds all projections from scratch
// Useful for initial population or complete rebuild
func (p *OrderProjector) RebuildAllProjections(ctx context.Context) error {
	// This would:
	// 1. Find all orders
	// 2. Rebuild projection for each
	// 3. Track progress
	// This is left as a TODO for production implementation
	p.logger.Info("Rebuild all projections requested - not yet implemented")
	return nil
}

// Helper functions

func calculateTotalValue(order *domain.Order) float64 {
	total := 0.0
	for _, item := range order.Items {
		total += item.UnitPrice * float64(item.Quantity)
	}
	return total
}
