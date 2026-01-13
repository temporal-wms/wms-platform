package workflows

import "fmt"

// InventoryReservationError represents a specific error for inventory reservation failures
type InventoryReservationError struct {
	OrderID       string
	Items         []ItemError
	UnderlyingErr error
	Message       string
}

// ItemError contains details about a failed item reservation
type ItemError struct {
	SKU      string
	Quantity int
	Reason   string
}

// Error implements the error interface
func (e *InventoryReservationError) Error() string {
	return fmt.Sprintf("inventory reservation failed for order %s: %s (underlying: %v)",
		e.OrderID, e.Message, e.UnderlyingErr)
}

// Unwrap returns the underlying error for error chain support
func (e *InventoryReservationError) Unwrap() error {
	return e.UnderlyingErr
}

// NewInventoryReservationError creates a new InventoryReservationError
func NewInventoryReservationError(orderID string, items []ItemError, underlying error, message string) *InventoryReservationError {
	return &InventoryReservationError{
		OrderID:       orderID,
		Items:         items,
		UnderlyingErr: underlying,
		Message:       message,
	}
}
