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

	s.logger.Info("Adjusted inventory", "sku", cmd.SKU, "location", cmd.LocationID, "newQuantity", cmd.NewQuantity)
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
