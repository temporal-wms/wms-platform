package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/inventory-service/internal/domain"
	"github.com/wms-platform/inventory-service/internal/infrastructure/projections"
)

// InventoryApplicationService handles inventory-related use cases
type InventoryApplicationService struct {
	repo         domain.InventoryRepository
	producer     *kafka.InstrumentedProducer
	eventFactory *cloudevents.EventFactory
	projector    *projections.InventoryProjector // CQRS projector for read model
	logger       *logging.Logger
}

// NewInventoryApplicationService creates a new InventoryApplicationService
func NewInventoryApplicationService(
	repo domain.InventoryRepository,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	projector *projections.InventoryProjector,
	logger *logging.Logger,
) *InventoryApplicationService {
	return &InventoryApplicationService{
		repo:         repo,
		producer:     producer,
		eventFactory: eventFactory,
		projector:    projector,
		logger:       logger,
	}
}

// CreateItem creates a new inventory item
func (s *InventoryApplicationService) CreateItem(ctx context.Context, cmd CreateItemCommand) (*InventoryItemDTO, error) {
	item := domain.NewInventoryItem(cmd.SKU, cmd.ProductName, cmd.ReorderPoint, cmd.ReorderQuantity)

	// Capture events before save
	events := item.GetDomainEvents()

	if err := s.repo.Save(ctx, item); err != nil {
		s.logger.Error("Failed to create item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, cmd.SKU, events)

	s.logger.Info("Created inventory item", "sku", cmd.SKU)
	return ToInventoryItemDTO(item), nil
}

// GetItem retrieves an inventory item by SKU
func (s *InventoryApplicationService) GetItem(ctx context.Context, query GetItemQuery) (*InventoryItemDTO, error) {
	item, err := s.repo.FindBySKU(ctx, query.SKU)
	if err != nil {
		s.logger.Error("Failed to get item", "sku", query.SKU, "error", err)
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if item == nil {
		return nil, errors.ErrNotFound("item")
	}

	return ToInventoryItemDTO(item), nil
}

// ReceiveStock receives stock into a location
func (s *InventoryApplicationService) ReceiveStock(ctx context.Context, cmd ReceiveStockCommand) (*InventoryItemDTO, error) {
	item, err := s.repo.FindBySKU(ctx, cmd.SKU)
	if err != nil {
		s.logger.Error("Failed to get item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if item == nil {
		return nil, errors.ErrNotFound("item")
	}

	// Receive stock (domain logic)
	if err := item.ReceiveStock(cmd.LocationID, cmd.Zone, cmd.Quantity, cmd.ReferenceID, cmd.CreatedBy); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Capture events before save
	events := item.GetDomainEvents()

	// Save the updated item
	if err := s.repo.Save(ctx, item); err != nil {
		s.logger.Error("Failed to save item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to save item: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, cmd.SKU, events)

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Received stock", "sku", cmd.SKU, "quantity", cmd.Quantity, "location", cmd.LocationID)
	return ToInventoryItemDTO(item), nil
}

// Reserve reserves stock for an order
func (s *InventoryApplicationService) Reserve(ctx context.Context, cmd ReserveCommand) (*InventoryItemDTO, error) {
	item, err := s.repo.FindBySKU(ctx, cmd.SKU)
	if err != nil {
		s.logger.Error("Failed to get item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if item == nil {
		return nil, errors.ErrNotFound("item")
	}

	// Reserve stock (domain logic)
	if err := item.Reserve(cmd.OrderID, cmd.LocationID, cmd.Quantity); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Note: Reserve doesn't generate domain events, but we still need to update projections
	// Call projector directly for reservation updates
	if s.projector != nil {
		_ = s.projector.OnInventoryReserved(ctx, cmd.SKU, cmd.OrderID)
	}

	// Save the updated item
	if err := s.repo.Save(ctx, item); err != nil {
		s.logger.Error("Failed to save item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to save item: %w", err)
	}

	s.logger.Info("Reserved stock", "sku", cmd.SKU, "orderId", cmd.OrderID, "quantity", cmd.Quantity)
	return ToInventoryItemDTO(item), nil
}

// Pick picks stock (reduces quantity)
func (s *InventoryApplicationService) Pick(ctx context.Context, cmd PickCommand) (*InventoryItemDTO, error) {
	item, err := s.repo.FindBySKU(ctx, cmd.SKU)
	if err != nil {
		s.logger.Error("Failed to get item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if item == nil {
		return nil, errors.ErrNotFound("item")
	}

	// Pick stock (domain logic)
	if err := item.Pick(cmd.OrderID, cmd.LocationID, cmd.Quantity, cmd.CreatedBy); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Capture events before save
	events := item.GetDomainEvents()

	// Save the updated item
	if err := s.repo.Save(ctx, item); err != nil {
		s.logger.Error("Failed to save item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to save item: %w", err)
	}

	// Update CQRS projections (including pick-specific updates)
	s.updateProjections(ctx, cmd.SKU, events)
	if s.projector != nil {
		_ = s.projector.OnInventoryPicked(ctx, cmd.SKU)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Picked stock", "sku", cmd.SKU, "orderId", cmd.OrderID, "quantity", cmd.Quantity)
	return ToInventoryItemDTO(item), nil
}

// ReleaseReservation releases a reservation
func (s *InventoryApplicationService) ReleaseReservation(ctx context.Context, cmd ReleaseReservationCommand) (*InventoryItemDTO, error) {
	item, err := s.repo.FindBySKU(ctx, cmd.SKU)
	if err != nil {
		s.logger.Error("Failed to get item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if item == nil {
		return nil, errors.ErrNotFound("item")
	}

	// Release reservation (domain logic)
	if err := item.ReleaseReservation(cmd.OrderID); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Save the updated item
	if err := s.repo.Save(ctx, item); err != nil {
		s.logger.Error("Failed to save item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to save item: %w", err)
	}

	// Update projections for reservation release
	if s.projector != nil {
		_ = s.projector.OnInventoryReserved(ctx, cmd.SKU, cmd.OrderID)
	}

	s.logger.Info("Released reservation", "sku", cmd.SKU, "orderId", cmd.OrderID)
	return ToInventoryItemDTO(item), nil
}

// ReleaseByOrder releases all reservations for an order across all SKUs
func (s *InventoryApplicationService) ReleaseByOrder(ctx context.Context, cmd ReleaseByOrderCommand) (int, error) {
	// Find all inventory items with reservations for this order
	items, err := s.repo.FindByOrderID(ctx, cmd.OrderID)
	if err != nil {
		s.logger.Error("Failed to find items by order", "orderId", cmd.OrderID, "error", err)
		return 0, fmt.Errorf("failed to find items by order: %w", err)
	}

	releasedCount := 0
	for _, item := range items {
		// Release reservation for this order
		if err := item.ReleaseReservation(cmd.OrderID); err != nil {
			s.logger.Warn("Failed to release reservation", "sku", item.SKU, "orderId", cmd.OrderID, "error", err)
			continue // Continue with other items
		}

		// Save the updated item
		if err := s.repo.Save(ctx, item); err != nil {
			s.logger.Error("Failed to save item after release", "sku", item.SKU, "error", err)
			continue
		}

		// Update projections
		if s.projector != nil {
			_ = s.projector.OnInventoryReserved(ctx, item.SKU, cmd.OrderID)
		}

		releasedCount++
	}

	s.logger.Info("Released reservations by order", "orderId", cmd.OrderID, "releasedCount", releasedCount)
	return releasedCount, nil
}

// Adjust adjusts inventory quantity
func (s *InventoryApplicationService) Adjust(ctx context.Context, cmd AdjustCommand) (*InventoryItemDTO, error) {
	item, err := s.repo.FindBySKU(ctx, cmd.SKU)
	if err != nil {
		s.logger.Error("Failed to get item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if item == nil {
		return nil, errors.ErrNotFound("item")
	}

	// Adjust inventory (domain logic)
	if err := item.Adjust(cmd.LocationID, cmd.NewQuantity, cmd.Reason, cmd.CreatedBy); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Capture events before save
	events := item.GetDomainEvents()

	// Save the updated item
	if err := s.repo.Save(ctx, item); err != nil {
		s.logger.Error("Failed to save item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to save item: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, cmd.SKU, events)

	// Events are saved to outbox by repository in transaction

	// Log business event: inventory adjusted
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "inventory.adjusted",
		EntityType: "inventory",
		EntityID:   cmd.SKU,
		Action:     "adjusted",
		RelatedIDs: map[string]string{
			"locationId":  cmd.LocationID,
			"newQuantity": fmt.Sprintf("%d", cmd.NewQuantity),
			"reason":      cmd.Reason,
		},
	})

	return ToInventoryItemDTO(item), nil
}

// GetByLocation retrieves items by location
func (s *InventoryApplicationService) GetByLocation(ctx context.Context, query GetByLocationQuery) ([]InventoryListDTO, error) {
	items, err := s.repo.FindByLocation(ctx, query.LocationID)
	if err != nil {
		s.logger.Error("Failed to get items by location", "locationId", query.LocationID, "error", err)
		return nil, fmt.Errorf("failed to get items by location: %w", err)
	}

	return ToInventoryListDTOs(items), nil
}

// GetByZone retrieves items by zone
func (s *InventoryApplicationService) GetByZone(ctx context.Context, query GetByZoneQuery) ([]InventoryListDTO, error) {
	items, err := s.repo.FindByZone(ctx, query.Zone)
	if err != nil {
		s.logger.Error("Failed to get items by zone", "zone", query.Zone, "error", err)
		return nil, fmt.Errorf("failed to get items by zone: %w", err)
	}

	return ToInventoryListDTOs(items), nil
}

// GetLowStock retrieves low stock items
func (s *InventoryApplicationService) GetLowStock(ctx context.Context) ([]InventoryListDTO, error) {
	items, err := s.repo.FindLowStock(ctx)
	if err != nil {
		s.logger.Error("Failed to get low stock items", "error", err)
		return nil, fmt.Errorf("failed to get low stock items: %w", err)
	}

	return ToInventoryListDTOs(items), nil
}

// ListInventory lists inventory with pagination
func (s *InventoryApplicationService) ListInventory(ctx context.Context, query ListInventoryQuery) ([]InventoryListDTO, error) {
	items, err := s.repo.FindAll(ctx, query.Limit, query.Offset)
	if err != nil {
		s.logger.Error("Failed to list inventory", "error", err)
		return nil, fmt.Errorf("failed to list inventory: %w", err)
	}

	return ToInventoryListDTOs(items), nil
}

// Stage converts a soft reservation to hard allocation (physical staging)
func (s *InventoryApplicationService) Stage(ctx context.Context, cmd StageCommand) (*InventoryItemDTO, error) {
	item, err := s.repo.FindBySKU(ctx, cmd.SKU)
	if err != nil {
		s.logger.Error("Failed to get item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if item == nil {
		return nil, errors.ErrNotFound("item")
	}

	// Stage inventory (domain logic - converts soft to hard allocation)
	if err := item.Stage(cmd.ReservationID, cmd.StagingLocationID, cmd.StagedBy); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Capture events before save
	events := item.GetDomainEvents()

	// Save the updated item
	if err := s.repo.Save(ctx, item); err != nil {
		s.logger.Error("Failed to save item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to save item: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, cmd.SKU, events)

	// Log business event
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "inventory.staged",
		EntityType: "inventory",
		EntityID:   cmd.SKU,
		Action:     "staged",
		RelatedIDs: map[string]string{
			"reservationId":     cmd.ReservationID,
			"stagingLocationId": cmd.StagingLocationID,
			"stagedBy":          cmd.StagedBy,
		},
	})

	s.logger.Info("Staged inventory", "sku", cmd.SKU, "reservationId", cmd.ReservationID, "stagingLocation", cmd.StagingLocationID)
	return ToInventoryItemDTO(item), nil
}

// Pack marks a hard allocation as packed
func (s *InventoryApplicationService) Pack(ctx context.Context, cmd PackCommand) (*InventoryItemDTO, error) {
	item, err := s.repo.FindBySKU(ctx, cmd.SKU)
	if err != nil {
		s.logger.Error("Failed to get item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if item == nil {
		return nil, errors.ErrNotFound("item")
	}

	// Pack inventory (domain logic)
	if err := item.Pack(cmd.AllocationID, cmd.PackedBy); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Capture events before save
	events := item.GetDomainEvents()

	// Save the updated item
	if err := s.repo.Save(ctx, item); err != nil {
		s.logger.Error("Failed to save item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to save item: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, cmd.SKU, events)

	// Log business event
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "inventory.packed",
		EntityType: "inventory",
		EntityID:   cmd.SKU,
		Action:     "packed",
		RelatedIDs: map[string]string{
			"allocationId": cmd.AllocationID,
			"packedBy":     cmd.PackedBy,
		},
	})

	s.logger.Info("Packed inventory", "sku", cmd.SKU, "allocationId", cmd.AllocationID)
	return ToInventoryItemDTO(item), nil
}

// Ship ships a packed allocation (removes inventory from system)
func (s *InventoryApplicationService) Ship(ctx context.Context, cmd ShipCommand) (*InventoryItemDTO, error) {
	item, err := s.repo.FindBySKU(ctx, cmd.SKU)
	if err != nil {
		s.logger.Error("Failed to get item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if item == nil {
		return nil, errors.ErrNotFound("item")
	}

	// Ship inventory (domain logic - removes from system)
	if err := item.Ship(cmd.AllocationID); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Capture events before save
	events := item.GetDomainEvents()

	// Save the updated item
	if err := s.repo.Save(ctx, item); err != nil {
		s.logger.Error("Failed to save item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to save item: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, cmd.SKU, events)

	// Log business event
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "inventory.shipped",
		EntityType: "inventory",
		EntityID:   cmd.SKU,
		Action:     "shipped",
		RelatedIDs: map[string]string{
			"allocationId": cmd.AllocationID,
		},
	})

	s.logger.Info("Shipped inventory", "sku", cmd.SKU, "allocationId", cmd.AllocationID)
	return ToInventoryItemDTO(item), nil
}

// ReturnToShelf returns hard allocated inventory back to shelf
func (s *InventoryApplicationService) ReturnToShelf(ctx context.Context, cmd ReturnToShelfCommand) (*InventoryItemDTO, error) {
	item, err := s.repo.FindBySKU(ctx, cmd.SKU)
	if err != nil {
		s.logger.Error("Failed to get item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if item == nil {
		return nil, errors.ErrNotFound("item")
	}

	// Return to shelf (domain logic - moves from hard allocation back to available)
	if err := item.ReturnToShelf(cmd.AllocationID, cmd.ReturnedBy, cmd.Reason); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Capture events before save
	events := item.GetDomainEvents()

	// Save the updated item
	if err := s.repo.Save(ctx, item); err != nil {
		s.logger.Error("Failed to save item", "sku", cmd.SKU, "error", err)
		return nil, fmt.Errorf("failed to save item: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, cmd.SKU, events)

	// Log business event
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "inventory.returned_to_shelf",
		EntityType: "inventory",
		EntityID:   cmd.SKU,
		Action:     "returned_to_shelf",
		RelatedIDs: map[string]string{
			"allocationId": cmd.AllocationID,
			"returnedBy":   cmd.ReturnedBy,
			"reason":       cmd.Reason,
		},
	})

	s.logger.Info("Returned inventory to shelf", "sku", cmd.SKU, "allocationId", cmd.AllocationID, "reason", cmd.Reason)
	return ToInventoryItemDTO(item), nil
}

// updateProjections updates the CQRS read model based on domain events
// Call this after successfully saving an inventory item to keep projections in sync
func (s *InventoryApplicationService) updateProjections(ctx context.Context, sku string, events []domain.DomainEvent) {
	if s.projector == nil {
		return // Projector not configured (e.g., in tests)
	}

	for _, event := range events {
		var err error
		switch e := event.(type) {
		case *domain.InventoryReceivedEvent:
			err = s.projector.OnInventoryReceived(ctx, e)
		case *domain.InventoryAdjustedEvent:
			err = s.projector.OnInventoryAdjusted(ctx, e)
		case *domain.LowStockAlertEvent:
			err = s.projector.OnLowStockAlert(ctx, e)
		}

		if err != nil {
			// Log error but don't fail the operation - projection updates are eventually consistent
			s.logger.Error("Failed to update projection", "eventType", fmt.Sprintf("%T", event), "error", err)
		}
	}
}
