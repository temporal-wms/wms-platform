package domain

import "errors"

// Ledger-specific domain errors
var (
	// ErrInvalidAccountType is returned when an invalid account type is used
	ErrInvalidAccountType = errors.New("invalid account type")

	// ErrInvalidValuationMethod is returned when an invalid valuation method is used
	ErrInvalidValuationMethod = errors.New("invalid valuation method")

	// ErrInsufficientCostLayers is returned when there are not enough cost layers to consume
	ErrInsufficientCostLayers = errors.New("insufficient cost layers to fulfill consumption")

	// ErrNoCostLayers is returned when trying to consume from empty cost layers
	ErrNoCostLayers = errors.New("no cost layers available")

	// ErrLedgerNotFound is returned when a ledger cannot be found
	ErrLedgerNotFound = errors.New("ledger not found")

	// ErrLedgerAlreadyExists is returned when trying to create a duplicate ledger
	ErrLedgerAlreadyExists = errors.New("ledger already exists for this SKU")

	// ErrInvalidLedgerBalance is returned when a ledger balance becomes invalid
	ErrInvalidLedgerBalance = errors.New("invalid ledger balance")

	// ErrNegativeLedgerBalance is returned when a ledger balance would become negative
	ErrNegativeLedgerBalance = errors.New("ledger balance cannot be negative")

	// ErrInvalidEntryAmount is returned when an entry amount is invalid
	ErrInvalidEntryAmount = errors.New("entry amount must be positive")

	// ErrUnbalancedTransaction is returned when debit and credit don't match
	ErrUnbalancedTransaction = errors.New("transaction is unbalanced: debits must equal credits")

	// ErrZeroUnitCost is returned when unit cost is zero
	ErrZeroUnitCost = errors.New("unit cost cannot be zero")

	// ErrMissingReferenceID is returned when a required reference ID is missing
	ErrMissingReferenceID = errors.New("reference ID is required")

	// ErrInvalidReconciliation is returned when reconciliation data is invalid
	ErrInvalidReconciliation = errors.New("invalid reconciliation data")
)
