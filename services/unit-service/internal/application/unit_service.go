package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/services/unit-service/internal/domain"
)

// UnitService handles unit-related business operations
type UnitService struct {
	unitRepo      domain.UnitRepository
	exceptionRepo domain.UnitExceptionRepository
	publisher     EventPublisher
}

// EventPublisher interface for publishing domain events
type EventPublisher interface {
	Publish(ctx context.Context, events []domain.DomainEvent) error
}

// NewUnitService creates a new unit service
func NewUnitService(unitRepo domain.UnitRepository, exceptionRepo domain.UnitExceptionRepository, publisher EventPublisher) *UnitService {
	return &UnitService{
		unitRepo:      unitRepo,
		exceptionRepo: exceptionRepo,
		publisher:     publisher,
	}
}

// CreateUnits creates new units at receiving
func (s *UnitService) CreateUnits(ctx context.Context, cmd CreateUnitsCommand) (*CreateUnitsResult, error) {
	unitIDs := make([]string, 0, cmd.Quantity)

	for i := 0; i < cmd.Quantity; i++ {
		unit := domain.NewUnit(cmd.SKU, cmd.ShipmentID, cmd.LocationID, cmd.CreatedBy)

		if err := s.unitRepo.Save(ctx, unit); err != nil {
			return nil, fmt.Errorf("failed to save unit: %w", err)
		}

		if s.publisher != nil {
			s.publisher.Publish(ctx, unit.Events())
		}

		unitIDs = append(unitIDs, unit.UnitID)
	}

	return &CreateUnitsResult{
		UnitIDs: unitIDs,
		SKU:     cmd.SKU,
		Count:   len(unitIDs),
	}, nil
}

// ReserveUnits reserves units for an order
func (s *UnitService) ReserveUnits(ctx context.Context, cmd ReserveUnitsCommand) (*ReserveUnitsResult, error) {
	result := &ReserveUnitsResult{
		ReservedUnits: make([]ReservedUnitInfo, 0),
		FailedItems:   make([]FailedReserve, 0),
	}

	for _, item := range cmd.Items {
		// Find available units for this SKU
		units, err := s.unitRepo.FindAvailableBySKU(ctx, item.SKU, item.Quantity)
		if err != nil {
			result.FailedItems = append(result.FailedItems, FailedReserve{
				SKU:       item.SKU,
				Requested: item.Quantity,
				Available: 0,
				Reason:    fmt.Sprintf("error finding units: %v", err),
			})
			continue
		}

		if len(units) < item.Quantity {
			result.FailedItems = append(result.FailedItems, FailedReserve{
				SKU:       item.SKU,
				Requested: item.Quantity,
				Available: len(units),
				Reason:    "insufficient available units",
			})
			continue
		}

		// Reserve each unit
		reservationID := fmt.Sprintf("%s-%s", cmd.OrderID, item.SKU)
		for i := 0; i < item.Quantity; i++ {
			unit := units[i]
			if err := unit.Reserve(cmd.OrderID, cmd.PathID, reservationID, cmd.HandlerID); err != nil {
				result.FailedItems = append(result.FailedItems, FailedReserve{
					SKU:       item.SKU,
					Requested: 1,
					Available: 0,
					Reason:    fmt.Sprintf("failed to reserve unit %s: %v", unit.UnitID, err),
				})
				continue
			}

			if err := s.unitRepo.Update(ctx, unit); err != nil {
				result.FailedItems = append(result.FailedItems, FailedReserve{
					SKU:       item.SKU,
					Requested: 1,
					Available: 0,
					Reason:    fmt.Sprintf("failed to update unit %s: %v", unit.UnitID, err),
				})
				continue
			}

			if s.publisher != nil {
				s.publisher.Publish(ctx, unit.Events())
			}

			result.ReservedUnits = append(result.ReservedUnits, ReservedUnitInfo{
				UnitID:     unit.UnitID,
				SKU:        unit.SKU,
				LocationID: unit.CurrentLocationID,
			})
		}
	}

	return result, nil
}

// GetUnitsForOrder retrieves all units for an order
func (s *UnitService) GetUnitsForOrder(ctx context.Context, orderID string) ([]*domain.Unit, error) {
	return s.unitRepo.FindByOrderID(ctx, orderID)
}

// GetUnit retrieves a single unit by its ID
func (s *UnitService) GetUnit(ctx context.Context, unitID string) (*domain.Unit, error) {
	return s.unitRepo.FindByUnitID(ctx, unitID)
}

// ConfirmPick confirms a unit has been picked
func (s *UnitService) ConfirmPick(ctx context.Context, cmd ConfirmPickCommand) error {
	unit, err := s.unitRepo.FindByUnitID(ctx, cmd.UnitID)
	if err != nil {
		return fmt.Errorf("unit not found: %w", err)
	}

	if err := unit.Pick(cmd.ToteID, cmd.PickerID, cmd.StationID); err != nil {
		return fmt.Errorf("failed to mark unit as picked: %w", err)
	}

	if err := s.unitRepo.Update(ctx, unit); err != nil {
		return fmt.Errorf("failed to update unit: %w", err)
	}

	if s.publisher != nil {
		s.publisher.Publish(ctx, unit.Events())
	}

	return nil
}

// ConfirmConsolidation confirms a unit has been consolidated
func (s *UnitService) ConfirmConsolidation(ctx context.Context, cmd ConfirmConsolidationCommand) error {
	unit, err := s.unitRepo.FindByUnitID(ctx, cmd.UnitID)
	if err != nil {
		return fmt.Errorf("unit not found: %w", err)
	}

	if err := unit.Consolidate(cmd.DestinationBin, cmd.WorkerID, cmd.StationID); err != nil {
		return fmt.Errorf("failed to mark unit as consolidated: %w", err)
	}

	if err := s.unitRepo.Update(ctx, unit); err != nil {
		return fmt.Errorf("failed to update unit: %w", err)
	}

	if s.publisher != nil {
		s.publisher.Publish(ctx, unit.Events())
	}

	return nil
}

// ConfirmPacked confirms a unit has been packed
func (s *UnitService) ConfirmPacked(ctx context.Context, cmd ConfirmPackedCommand) error {
	unit, err := s.unitRepo.FindByUnitID(ctx, cmd.UnitID)
	if err != nil {
		return fmt.Errorf("unit not found: %w", err)
	}

	if err := unit.Pack(cmd.PackageID, cmd.PackerID, cmd.StationID); err != nil {
		return fmt.Errorf("failed to mark unit as packed: %w", err)
	}

	if err := s.unitRepo.Update(ctx, unit); err != nil {
		return fmt.Errorf("failed to update unit: %w", err)
	}

	if s.publisher != nil {
		s.publisher.Publish(ctx, unit.Events())
	}

	return nil
}

// ConfirmShipped confirms a unit has been shipped
func (s *UnitService) ConfirmShipped(ctx context.Context, cmd ConfirmShippedCommand) error {
	unit, err := s.unitRepo.FindByUnitID(ctx, cmd.UnitID)
	if err != nil {
		return fmt.Errorf("unit not found: %w", err)
	}

	if err := unit.Ship(cmd.ShipmentID, cmd.TrackingNumber, cmd.HandlerID); err != nil {
		return fmt.Errorf("failed to mark unit as shipped: %w", err)
	}

	if err := s.unitRepo.Update(ctx, unit); err != nil {
		return fmt.Errorf("failed to update unit: %w", err)
	}

	if s.publisher != nil {
		s.publisher.Publish(ctx, unit.Events())
	}

	return nil
}

// CreateException creates an exception for a unit
func (s *UnitService) CreateException(ctx context.Context, cmd CreateExceptionCommand) (*domain.UnitException, error) {
	// Get the unit to extract order info
	unit, err := s.unitRepo.FindByUnitID(ctx, cmd.UnitID)
	if err != nil {
		return nil, fmt.Errorf("unit not found: %w", err)
	}

	// Create the exception
	exception := domain.NewUnitException(
		cmd.UnitID,
		unit.OrderID,
		unit.SKU,
		cmd.ExceptionType,
		cmd.Stage,
		cmd.Description,
		cmd.StationID,
		cmd.ReportedBy,
	)

	if err := s.exceptionRepo.Save(ctx, exception); err != nil {
		return nil, fmt.Errorf("failed to save exception: %w", err)
	}

	// Mark the unit as having an exception
	if err := unit.MarkException(exception.ExceptionID, cmd.Description, cmd.ReportedBy, cmd.StationID); err != nil {
		return nil, fmt.Errorf("failed to mark unit exception: %w", err)
	}

	if err := s.unitRepo.Update(ctx, unit); err != nil {
		return nil, fmt.Errorf("failed to update unit: %w", err)
	}

	if s.publisher != nil {
		s.publisher.Publish(ctx, unit.Events())
	}

	return exception, nil
}

// ResolveException resolves a unit exception
func (s *UnitService) ResolveException(ctx context.Context, cmd ResolveExceptionCommand) error {
	exception, err := s.exceptionRepo.FindByID(ctx, cmd.ExceptionID)
	if err != nil {
		return fmt.Errorf("exception not found: %w", err)
	}

	exception.Resolve(cmd.Resolution, cmd.ResolvedBy)

	if err := s.exceptionRepo.Update(ctx, exception); err != nil {
		return fmt.Errorf("failed to update exception: %w", err)
	}

	return nil
}

// GetUnitAuditTrail retrieves the full movement history for a unit
func (s *UnitService) GetUnitAuditTrail(ctx context.Context, unitID string) ([]domain.UnitMovement, error) {
	unit, err := s.unitRepo.FindByUnitID(ctx, unitID)
	if err != nil {
		return nil, fmt.Errorf("unit not found: %w", err)
	}

	return unit.GetAuditTrail(), nil
}

// GetExceptionsForOrder retrieves all exceptions for an order
func (s *UnitService) GetExceptionsForOrder(ctx context.Context, orderID string) ([]*domain.UnitException, error) {
	return s.exceptionRepo.FindByOrderID(ctx, orderID)
}

// GetUnresolvedExceptions retrieves unresolved exceptions
func (s *UnitService) GetUnresolvedExceptions(ctx context.Context, limit int) ([]*domain.UnitException, error) {
	return s.exceptionRepo.FindUnresolved(ctx, limit)
}
