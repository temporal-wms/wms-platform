package domain

// ProcessRequirement represents a fulfillment requirement for process path routing
type ProcessRequirement string

const (
	// Item count requirements
	RequirementSingleItem ProcessRequirement = "single_item"
	RequirementMultiItem  ProcessRequirement = "multi_item"

	// Special handling requirements
	RequirementGiftWrap  ProcessRequirement = "gift_wrap"
	RequirementHazmat    ProcessRequirement = "hazmat"
	RequirementOversized ProcessRequirement = "oversized"
	RequirementFragile   ProcessRequirement = "fragile"
	RequirementColdChain ProcessRequirement = "cold_chain"
	RequirementHighValue ProcessRequirement = "high_value"
)

// IsValid checks if the requirement is a valid ProcessRequirement
func (r ProcessRequirement) IsValid() bool {
	switch r {
	case RequirementSingleItem, RequirementMultiItem,
		RequirementGiftWrap, RequirementHazmat,
		RequirementOversized, RequirementFragile,
		RequirementColdChain, RequirementHighValue:
		return true
	default:
		return false
	}
}

// GiftWrapDetails contains details for gift wrap processing
type GiftWrapDetails struct {
	WrapType    string `bson:"wrapType" json:"wrapType"`       // standard, premium, holiday
	GiftMessage string `bson:"giftMessage" json:"giftMessage"` // optional message to include
	HidePrice   bool   `bson:"hidePrice" json:"hidePrice"`     // hide price on packing slip
}

// HazmatDetails contains details for hazardous material handling
type HazmatDetails struct {
	Class              string `bson:"class" json:"class"`                           // UN hazmat class (1-9)
	UNNumber           string `bson:"unNumber" json:"unNumber"`                     // UN identification number
	PackingGroup       string `bson:"packingGroup" json:"packingGroup"`             // I, II, or III
	ProperShippingName string `bson:"properShippingName" json:"properShippingName"` // official shipping name
	LimitedQuantity    bool   `bson:"limitedQuantity" json:"limitedQuantity"`       // qualifies for LQ exemption
}

// ColdChainDetails contains details for temperature-controlled shipping
type ColdChainDetails struct {
	MinTempCelsius  float64 `bson:"minTempCelsius" json:"minTempCelsius"`   // minimum temperature
	MaxTempCelsius  float64 `bson:"maxTempCelsius" json:"maxTempCelsius"`   // maximum temperature
	RequiresDryIce  bool    `bson:"requiresDryIce" json:"requiresDryIce"`   // needs dry ice
	RequiresGelPack bool    `bson:"requiresGelPack" json:"requiresGelPack"` // needs gel packs
}

// OrderRequirements encapsulates all process path requirements for an order
type OrderRequirements struct {
	Requirements     []ProcessRequirement `bson:"requirements" json:"requirements"`
	GiftWrapDetails  *GiftWrapDetails     `bson:"giftWrapDetails,omitempty" json:"giftWrapDetails,omitempty"`
	HazmatDetails    *HazmatDetails       `bson:"hazmatDetails,omitempty" json:"hazmatDetails,omitempty"`
	ColdChainDetails *ColdChainDetails    `bson:"coldChainDetails,omitempty" json:"coldChainDetails,omitempty"`
}

// HasRequirement checks if the order has a specific requirement
func (r *OrderRequirements) HasRequirement(req ProcessRequirement) bool {
	for _, existing := range r.Requirements {
		if existing == req {
			return true
		}
	}
	return false
}

// AddRequirement adds a requirement if not already present
func (r *OrderRequirements) AddRequirement(req ProcessRequirement) {
	if !r.HasRequirement(req) {
		r.Requirements = append(r.Requirements, req)
	}
}

// RequiresConsolidation returns true if the order requires consolidation
func (r *OrderRequirements) RequiresConsolidation() bool {
	return r.HasRequirement(RequirementMultiItem)
}

// RequiresGiftWrap returns true if the order needs gift wrapping
func (r *OrderRequirements) RequiresGiftWrap() bool {
	return r.HasRequirement(RequirementGiftWrap)
}

// RequiresSpecialHandling returns true if any special handling is needed
func (r *OrderRequirements) RequiresSpecialHandling() bool {
	specialHandlingReqs := []ProcessRequirement{
		RequirementHazmat,
		RequirementOversized,
		RequirementFragile,
		RequirementColdChain,
		RequirementHighValue,
	}

	for _, req := range specialHandlingReqs {
		if r.HasRequirement(req) {
			return true
		}
	}
	return false
}

// GetSpecialHandlingTypes returns a list of special handling requirements
func (r *OrderRequirements) GetSpecialHandlingTypes() []ProcessRequirement {
	specialHandling := make([]ProcessRequirement, 0)
	specialHandlingReqs := []ProcessRequirement{
		RequirementHazmat,
		RequirementOversized,
		RequirementFragile,
		RequirementColdChain,
		RequirementHighValue,
	}

	for _, req := range specialHandlingReqs {
		if r.HasRequirement(req) {
			specialHandling = append(specialHandling, req)
		}
	}
	return specialHandling
}
