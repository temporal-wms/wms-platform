package domain

import (
	"time"

	"github.com/google/uuid"
)

// ProcessPath represents the determined process path for an order
type ProcessPath struct {
	ID                    string               `json:"id" bson:"_id,omitempty"`
	PathID                string               `json:"pathId" bson:"pathId"`
	OrderID               string               `json:"orderId" bson:"orderId"`
	Requirements          []ProcessRequirement `json:"requirements" bson:"requirements"`
	ConsolidationRequired bool                 `json:"consolidationRequired" bson:"consolidationRequired"`
	GiftWrapRequired      bool                 `json:"giftWrapRequired" bson:"giftWrapRequired"`
	SpecialHandling       []string             `json:"specialHandling" bson:"specialHandling"`
	TargetStationID       string               `json:"targetStationId,omitempty" bson:"targetStationId,omitempty"`
	CreatedAt             time.Time            `json:"createdAt" bson:"createdAt"`
	UpdatedAt             time.Time            `json:"updatedAt" bson:"updatedAt"`
}

// ProcessPathItem represents an item for process path determination
type ProcessPathItem struct {
	SKU               string  `json:"sku"`
	Quantity          int     `json:"quantity"`
	Weight            float64 `json:"weight"`
	IsFragile         bool    `json:"isFragile"`
	IsHazmat          bool    `json:"isHazmat"`
	RequiresColdChain bool    `json:"requiresColdChain"`
}

// GiftWrapDetails contains details for gift wrap processing
type GiftWrapDetails struct {
	WrapType    string `json:"wrapType"`
	GiftMessage string `json:"giftMessage"`
	HidePrice   bool   `json:"hidePrice"`
}

// HazmatDetails contains details for hazardous material handling
type HazmatDetails struct {
	Class              string `json:"class"`
	UNNumber           string `json:"unNumber"`
	PackingGroup       string `json:"packingGroup"`
	ProperShippingName string `json:"properShippingName"`
	LimitedQuantity    bool   `json:"limitedQuantity"`
}

// ColdChainDetails contains details for temperature-controlled shipping
type ColdChainDetails struct {
	MinTempCelsius  float64 `json:"minTempCelsius"`
	MaxTempCelsius  float64 `json:"maxTempCelsius"`
	RequiresDryIce  bool    `json:"requiresDryIce"`
	RequiresGelPack bool    `json:"requiresGelPack"`
}

// DetermineProcessPathInput represents input for determining process path
type DetermineProcessPathInput struct {
	OrderID          string            `json:"orderId"`
	Items            []ProcessPathItem `json:"items"`
	GiftWrap         bool              `json:"giftWrap"`
	GiftWrapDetails  *GiftWrapDetails  `json:"giftWrapDetails,omitempty"`
	HazmatDetails    *HazmatDetails    `json:"hazmatDetails,omitempty"`
	ColdChainDetails *ColdChainDetails `json:"coldChainDetails,omitempty"`
	TotalValue       float64           `json:"totalValue"`
}

// NewProcessPath creates a new ProcessPath based on order characteristics
func NewProcessPath(input DetermineProcessPathInput) *ProcessPath {
	now := time.Now()
	path := &ProcessPath{
		PathID:          uuid.New().String(),
		OrderID:         input.OrderID,
		Requirements:    make([]ProcessRequirement, 0),
		SpecialHandling: make([]string, 0),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Determine single vs multi-item
	totalItems := 0
	for _, item := range input.Items {
		totalItems += item.Quantity
	}

	if totalItems == 1 && len(input.Items) == 1 {
		path.Requirements = append(path.Requirements, RequirementSingleItem)
		path.ConsolidationRequired = false
	} else {
		path.Requirements = append(path.Requirements, RequirementMultiItem)
		path.ConsolidationRequired = true
	}

	// Check for gift wrap
	if input.GiftWrap {
		path.Requirements = append(path.Requirements, RequirementGiftWrap)
		path.GiftWrapRequired = true
	}

	// Check for high value
	if input.TotalValue >= HighValueThreshold {
		path.Requirements = append(path.Requirements, RequirementHighValue)
		path.SpecialHandling = append(path.SpecialHandling, "high_value_verification")
	}

	// Check for fragile items
	hasFragile := false
	for _, item := range input.Items {
		if item.IsFragile {
			hasFragile = true
			break
		}
	}
	if hasFragile {
		path.Requirements = append(path.Requirements, RequirementFragile)
		path.SpecialHandling = append(path.SpecialHandling, "fragile_packing")
	}

	// Check for oversized items
	hasOversized := false
	for _, item := range input.Items {
		if item.Weight >= OversizedWeightThreshold {
			hasOversized = true
			break
		}
	}
	if hasOversized {
		path.Requirements = append(path.Requirements, RequirementOversized)
		path.SpecialHandling = append(path.SpecialHandling, "oversized_handling")
	}

	// Check for hazmat items
	hasHazmat := false
	for _, item := range input.Items {
		if item.IsHazmat {
			hasHazmat = true
			break
		}
	}
	if hasHazmat || input.HazmatDetails != nil {
		path.Requirements = append(path.Requirements, RequirementHazmat)
		path.SpecialHandling = append(path.SpecialHandling, "hazmat_compliance")
	}

	// Check for cold chain items
	hasColdChain := false
	for _, item := range input.Items {
		if item.RequiresColdChain {
			hasColdChain = true
			break
		}
	}
	if hasColdChain || input.ColdChainDetails != nil {
		path.Requirements = append(path.Requirements, RequirementColdChain)
		path.SpecialHandling = append(path.SpecialHandling, "cold_chain_packaging")
	}

	return path
}

// AssignStation assigns a target station to the process path
func (p *ProcessPath) AssignStation(stationID string) {
	p.TargetStationID = stationID
	p.UpdatedAt = time.Now()
}

// HasRequirement checks if the process path has a specific requirement
func (p *ProcessPath) HasRequirement(req ProcessRequirement) bool {
	for _, r := range p.Requirements {
		if r == req {
			return true
		}
	}
	return false
}
