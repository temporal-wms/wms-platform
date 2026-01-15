package domain

// ValuationMethod represents the inventory cost valuation method
type ValuationMethod string

const (
	// ValuationFIFO - First-In-First-Out: oldest inventory costs are consumed first
	// Most common for perishables and standard accounting
	ValuationFIFO ValuationMethod = "FIFO"

	// ValuationLIFO - Last-In-First-Out: newest inventory costs are consumed first
	// Less common, has tax implications
	ValuationLIFO ValuationMethod = "LIFO"

	// ValuationWeightedAverage - Average cost recalculated after each receipt
	// Simpler to implement, good for fungible goods
	ValuationWeightedAverage ValuationMethod = "WEIGHTED_AVERAGE"
)

// DefaultValuationMethod is the default valuation method for new ledgers
const DefaultValuationMethod = ValuationFIFO

// IsValid checks if the valuation method is valid
func (v ValuationMethod) IsValid() bool {
	switch v {
	case ValuationFIFO, ValuationLIFO, ValuationWeightedAverage:
		return true
	default:
		return false
	}
}

// String returns the string representation of the valuation method
func (v ValuationMethod) String() string {
	return string(v)
}

// UsesLayers returns true if this valuation method uses cost layers (FIFO/LIFO)
func (v ValuationMethod) UsesLayers() bool {
	return v == ValuationFIFO || v == ValuationLIFO
}

// Description returns a human-readable description of the valuation method
func (v ValuationMethod) Description() string {
	switch v {
	case ValuationFIFO:
		return "First-In-First-Out: oldest inventory costs are consumed first"
	case ValuationLIFO:
		return "Last-In-First-Out: newest inventory costs are consumed first"
	case ValuationWeightedAverage:
		return "Weighted Average: average cost recalculated after each receipt"
	default:
		return "Unknown valuation method"
	}
}
