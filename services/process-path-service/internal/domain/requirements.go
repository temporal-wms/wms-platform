package domain

// ProcessRequirement represents a fulfillment requirement
type ProcessRequirement string

const (
	RequirementSingleItem ProcessRequirement = "single_item"
	RequirementMultiItem  ProcessRequirement = "multi_item"
	RequirementGiftWrap   ProcessRequirement = "gift_wrap"
	RequirementHazmat     ProcessRequirement = "hazmat"
	RequirementOversized  ProcessRequirement = "oversized"
	RequirementFragile    ProcessRequirement = "fragile"
	RequirementColdChain  ProcessRequirement = "cold_chain"
	RequirementHighValue  ProcessRequirement = "high_value"
)

// Thresholds for process path determination
const (
	// HighValueThreshold is the threshold for high-value orders (in dollars)
	HighValueThreshold float64 = 500.0

	// OversizedWeightThreshold is the threshold weight for oversized items (in kg)
	OversizedWeightThreshold float64 = 30.0
)
