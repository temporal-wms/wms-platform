package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/services/receiving-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
)

// ReceivingService handles receiving operations
type ReceivingService struct {
	repo   domain.InboundShipmentRepository
	logger *logging.Logger
}

// NewReceivingService creates a new ReceivingService
func NewReceivingService(repo domain.InboundShipmentRepository, logger *logging.Logger) *ReceivingService {
	return &ReceivingService{
		repo:   repo,
		logger: logger,
	}
}

// CreateShipment creates a new expected inbound shipment
func (s *ReceivingService) CreateShipment(ctx context.Context, cmd CreateShipmentCommand) (*domain.InboundShipment, error) {
	// Generate shipment ID if not provided
	shipmentID := cmd.ShipmentID
	if shipmentID == "" {
		shipmentID = fmt.Sprintf("SHP-%s", time.Now().Format("20060102150405"))
	}

	// Create aggregate with provided ASN, supplier, and items
	shipment, err := domain.NewInboundShipment(shipmentID, cmd.ASN, cmd.Supplier, cmd.ExpectedItems, cmd.PurchaseOrderID)
	if err != nil {
		return nil, err
	}

	// Persist (repository handles event publishing via outbox)
	if err := s.repo.Save(ctx, shipment); err != nil {
		return nil, err
	}

	s.logger.Info("Created inbound shipment",
		"shipmentId", shipment.ShipmentID,
		"asnId", cmd.ASN.ASNID,
		"supplier", cmd.Supplier.SupplierName,
	)

	return shipment, nil
}

// MarkShipmentArrived marks a shipment as arrived at the dock
func (s *ReceivingService) MarkShipmentArrived(ctx context.Context, cmd MarkArrivedCommand) (*domain.InboundShipment, error) {
	shipment, err := s.repo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, fmt.Errorf("shipment not found: %s", cmd.ShipmentID)
	}

	if err := shipment.MarkArrived(cmd.DockID); err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		return nil, err
	}

	s.logger.Info("Shipment arrived",
		"shipmentId", cmd.ShipmentID,
		"dockId", cmd.DockID,
	)

	return shipment, nil
}

// StartReceiving starts the receiving process
func (s *ReceivingService) StartReceiving(ctx context.Context, cmd StartReceivingCommand) (*domain.InboundShipment, error) {
	shipment, err := s.repo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, fmt.Errorf("shipment not found: %s", cmd.ShipmentID)
	}

	if err := shipment.StartReceiving(cmd.WorkerID); err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		return nil, err
	}

	s.logger.Info("Started receiving",
		"shipmentId", cmd.ShipmentID,
		"workerId", cmd.WorkerID,
	)

	return shipment, nil
}

// ReceiveItem records the receipt of an item
func (s *ReceivingService) ReceiveItem(ctx context.Context, cmd ReceiveItemCommand) (*domain.InboundShipment, error) {
	shipment, err := s.repo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, fmt.Errorf("shipment not found: %s", cmd.ShipmentID)
	}

	if err := shipment.ReceiveItem(cmd.SKU, cmd.Quantity, cmd.Condition, cmd.ToteID, cmd.WorkerID, cmd.Notes); err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		return nil, err
	}

	s.logger.Info("Received item",
		"shipmentId", cmd.ShipmentID,
		"sku", cmd.SKU,
		"quantity", cmd.Quantity,
	)

	return shipment, nil
}

// CompleteReceiving completes the receiving process
func (s *ReceivingService) CompleteReceiving(ctx context.Context, cmd CompleteReceivingCommand) (*domain.InboundShipment, error) {
	shipment, err := s.repo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, fmt.Errorf("shipment not found: %s", cmd.ShipmentID)
	}

	if err := shipment.Complete(); err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		return nil, err
	}

	s.logger.Info("Completed receiving",
		"shipmentId", cmd.ShipmentID,
	)

	return shipment, nil
}

// BatchReceiveByCarton receives all items in a carton at once (batch ASN receive)
func (s *ReceivingService) BatchReceiveByCarton(ctx context.Context, cmd BatchReceiveByCartonCommand) (*domain.InboundShipment, error) {
	shipment, err := s.repo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, fmt.Errorf("shipment not found: %s", cmd.ShipmentID)
	}

	if err := shipment.BatchReceiveByCarton(cmd.CartonID, cmd.WorkerID, cmd.ToteID); err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		return nil, err
	}

	s.logger.Info("Batch received carton",
		"shipmentId", cmd.ShipmentID,
		"cartonId", cmd.CartonID,
		"workerId", cmd.WorkerID,
	)

	return shipment, nil
}

// MarkItemForPrep marks an item as needing prep (repackaging)
func (s *ReceivingService) MarkItemForPrep(ctx context.Context, cmd MarkItemForPrepCommand) (*domain.InboundShipment, error) {
	shipment, err := s.repo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, fmt.Errorf("shipment not found: %s", cmd.ShipmentID)
	}

	if err := shipment.MarkItemForPrep(cmd.SKU, cmd.Quantity, cmd.WorkerID, cmd.ToteID, cmd.Reason); err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		return nil, err
	}

	s.logger.Info("Marked item for prep",
		"shipmentId", cmd.ShipmentID,
		"sku", cmd.SKU,
		"quantity", cmd.Quantity,
		"reason", cmd.Reason,
	)

	return shipment, nil
}

// CompletePrep completes prep for an item
func (s *ReceivingService) CompletePrep(ctx context.Context, cmd CompletePrepCommand) (*domain.InboundShipment, error) {
	shipment, err := s.repo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, fmt.Errorf("shipment not found: %s", cmd.ShipmentID)
	}

	if err := shipment.CompletePrepForItem(cmd.SKU, cmd.Quantity, cmd.WorkerID, cmd.ToteID); err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		return nil, err
	}

	s.logger.Info("Completed prep for item",
		"shipmentId", cmd.ShipmentID,
		"sku", cmd.SKU,
		"quantity", cmd.Quantity,
	)

	return shipment, nil
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
func (s *ReceivingService) GetShipmentsByStatus(ctx context.Context, status domain.ShipmentStatus) ([]*domain.InboundShipment, error) {
	return s.repo.FindByStatus(ctx, status, domain.DefaultPagination())
}

// ListShipments retrieves all shipments up to the specified limit
func (s *ReceivingService) ListShipments(ctx context.Context, limit int) ([]*domain.InboundShipment, error) {
	return s.repo.FindAll(ctx, limit)
}

// GetExpectedArrivals retrieves shipments expected to arrive within the given time range
func (s *ReceivingService) GetExpectedArrivals(ctx context.Context, from, to time.Time) ([]*domain.InboundShipment, error) {
	return s.repo.FindExpectedArrivals(ctx, from, to)
}
