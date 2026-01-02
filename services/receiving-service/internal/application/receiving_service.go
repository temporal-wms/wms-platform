package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/services/receiving-service/internal/domain"
)

// ReceivingService handles receiving operations
type ReceivingService struct {
	repo      domain.InboundShipmentRepository
	publisher domain.EventPublisher
}

// NewReceivingService creates a new ReceivingService
func NewReceivingService(repo domain.InboundShipmentRepository, publisher domain.EventPublisher) *ReceivingService {
	return &ReceivingService{
		repo:      repo,
		publisher: publisher,
	}
}

// CreateShipment creates a new expected inbound shipment
func (s *ReceivingService) CreateShipment(ctx context.Context, cmd CreateShipmentCommand) (*domain.InboundShipment, error) {
	// Build ASN
	asn := domain.AdvanceShippingNotice{
		ASNID:           cmd.ASNID,
		CarrierName:     cmd.CarrierName,
		TrackingNumber:  cmd.TrackingNumber,
		ExpectedArrival: cmd.ExpectedArrival,
		ContainerCount:  cmd.ContainerCount,
		TotalWeight:     cmd.TotalWeight,
		SpecialHandling: cmd.SpecialHandling,
	}

	// Build supplier
	supplier := domain.Supplier{
		SupplierID:   cmd.SupplierID,
		SupplierName: cmd.SupplierName,
		ContactName:  cmd.ContactName,
		ContactPhone: cmd.ContactPhone,
		ContactEmail: cmd.ContactEmail,
	}

	// Build expected items
	expectedItems := make([]domain.ExpectedItem, len(cmd.ExpectedItems))
	for i, item := range cmd.ExpectedItems {
		expectedItems[i] = domain.ExpectedItem{
			SKU:               item.SKU,
			ProductName:       item.ProductName,
			ExpectedQuantity:  item.ExpectedQuantity,
			UnitCost:          item.UnitCost,
			Weight:            item.Weight,
			IsHazmat:          item.IsHazmat,
			RequiresColdChain: item.RequiresColdChain,
		}
	}

	// Generate shipment ID
	shipmentID := fmt.Sprintf("SHP-%s", time.Now().Format("20060102150405"))

	// Create aggregate
	shipment, err := domain.NewInboundShipment(shipmentID, asn, supplier, expectedItems, cmd.PurchaseOrderID)
	if err != nil {
		return nil, err
	}

	// Persist
	if err := s.repo.Save(ctx, shipment); err != nil {
		return nil, err
	}

	// Publish events
	if s.publisher != nil {
		if err := s.publisher.PublishAll(ctx, shipment.GetDomainEvents()); err != nil {
			// Log but don't fail
			fmt.Printf("Failed to publish events: %v\n", err)
		}
	}
	shipment.ClearDomainEvents()

	return shipment, nil
}

// MarkShipmentArrived marks a shipment as arrived at the dock
func (s *ReceivingService) MarkShipmentArrived(ctx context.Context, shipmentID, dockID string) error {
	shipment, err := s.repo.FindByID(ctx, shipmentID)
	if err != nil {
		return err
	}

	if err := shipment.MarkArrived(dockID); err != nil {
		return err
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		return err
	}

	// Publish events
	if s.publisher != nil {
		if err := s.publisher.PublishAll(ctx, shipment.GetDomainEvents()); err != nil {
			fmt.Printf("Failed to publish events: %v\n", err)
		}
	}
	shipment.ClearDomainEvents()

	return nil
}

// StartReceiving starts the receiving process
func (s *ReceivingService) StartReceiving(ctx context.Context, shipmentID, workerID string) error {
	shipment, err := s.repo.FindByID(ctx, shipmentID)
	if err != nil {
		return err
	}

	if err := shipment.StartReceiving(workerID); err != nil {
		return err
	}

	return s.repo.Save(ctx, shipment)
}

// ReceiveItem records the receipt of an item
func (s *ReceivingService) ReceiveItem(ctx context.Context, cmd ReceiveItemCommand) error {
	shipment, err := s.repo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		return err
	}

	if err := shipment.ReceiveItem(cmd.SKU, cmd.Quantity, cmd.Condition, cmd.ToteID, cmd.WorkerID, cmd.Notes); err != nil {
		return err
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		return err
	}

	// Publish events
	if s.publisher != nil {
		if err := s.publisher.PublishAll(ctx, shipment.GetDomainEvents()); err != nil {
			fmt.Printf("Failed to publish events: %v\n", err)
		}
	}
	shipment.ClearDomainEvents()

	return nil
}

// CompleteReceiving completes the receiving process
func (s *ReceivingService) CompleteReceiving(ctx context.Context, shipmentID string) error {
	shipment, err := s.repo.FindByID(ctx, shipmentID)
	if err != nil {
		return err
	}

	if err := shipment.Complete(); err != nil {
		return err
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		return err
	}

	// Publish events
	if s.publisher != nil {
		if err := s.publisher.PublishAll(ctx, shipment.GetDomainEvents()); err != nil {
			fmt.Printf("Failed to publish events: %v\n", err)
		}
	}
	shipment.ClearDomainEvents()

	return nil
}

// GetShipment retrieves a shipment by ID
func (s *ReceivingService) GetShipment(ctx context.Context, shipmentID string) (*domain.InboundShipment, error) {
	return s.repo.FindByID(ctx, shipmentID)
}

// GetExpectedToday retrieves shipments expected today
func (s *ReceivingService) GetExpectedToday(ctx context.Context) ([]*domain.InboundShipment, error) {
	return s.repo.FindExpectedToday(ctx)
}

// GetPendingReceiving retrieves shipments pending receiving
func (s *ReceivingService) GetPendingReceiving(ctx context.Context, limit int) ([]*domain.InboundShipment, error) {
	return s.repo.FindPendingReceiving(ctx, limit)
}

// GetShipmentsByStatus retrieves shipments by status
func (s *ReceivingService) GetShipmentsByStatus(ctx context.Context, status domain.ShipmentStatus, pagination domain.Pagination) ([]*domain.InboundShipment, error) {
	return s.repo.FindByStatus(ctx, status, pagination)
}
