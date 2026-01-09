package domain

import "context"

// UnitRepository defines the interface for unit persistence
type UnitRepository interface {
	// Save persists a unit
	Save(ctx context.Context, unit *Unit) error

	// FindByID retrieves a unit by its MongoDB ID
	FindByID(ctx context.Context, id string) (*Unit, error)

	// FindByUnitID retrieves a unit by its UUID
	FindByUnitID(ctx context.Context, unitID string) (*Unit, error)

	// FindByOrderID retrieves all units for an order
	FindByOrderID(ctx context.Context, orderID string) ([]*Unit, error)

	// FindBySKU retrieves all units for a SKU
	FindBySKU(ctx context.Context, sku string) ([]*Unit, error)

	// FindByShipmentID retrieves all units from an inbound shipment
	FindByShipmentID(ctx context.Context, shipmentID string) ([]*Unit, error)

	// FindByStatus retrieves all units with a specific status
	FindByStatus(ctx context.Context, status UnitStatus) ([]*Unit, error)

	// FindAvailableBySKU retrieves available (received) units for a SKU
	FindAvailableBySKU(ctx context.Context, sku string, limit int) ([]*Unit, error)

	// Update updates a unit
	Update(ctx context.Context, unit *Unit) error

	// Delete removes a unit
	Delete(ctx context.Context, unitID string) error
}

// UnitExceptionRepository defines the interface for unit exception persistence
type UnitExceptionRepository interface {
	// Save persists a unit exception
	Save(ctx context.Context, exception *UnitException) error

	// FindByID retrieves an exception by its ID
	FindByID(ctx context.Context, exceptionID string) (*UnitException, error)

	// FindByUnitID retrieves all exceptions for a unit
	FindByUnitID(ctx context.Context, unitID string) ([]*UnitException, error)

	// FindByOrderID retrieves all exceptions for an order
	FindByOrderID(ctx context.Context, orderID string) ([]*UnitException, error)

	// FindUnresolved retrieves all unresolved exceptions
	FindUnresolved(ctx context.Context, limit int) ([]*UnitException, error)

	// FindByStage retrieves exceptions by process stage
	FindByStage(ctx context.Context, stage ExceptionStage) ([]*UnitException, error)

	// Update updates an exception
	Update(ctx context.Context, exception *UnitException) error
}
