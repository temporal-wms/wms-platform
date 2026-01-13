package domain

import (
	"time"

	"github.com/google/uuid"
)

// ProcessPathTier represents the quality tier of a process path
type ProcessPathTier string

const (
	// TierOptimal - All automation, optimal routing, full capabilities
	TierOptimal ProcessPathTier = "optimal"

	// TierStandard - Standard routing with all requirements met
	TierStandard ProcessPathTier = "standard"

	// TierDegraded - Degraded path due to capacity/resource constraints
	TierDegraded ProcessPathTier = "degraded"

	// TierManual - Manual intervention required
	TierManual ProcessPathTier = "manual"
)

// EscalationTrigger represents reasons for path escalation
type EscalationTrigger string

const (
	TriggerStationUnavailable     EscalationTrigger = "station_unavailable"
	TriggerCapacityExceeded       EscalationTrigger = "capacity_exceeded"
	TriggerEquipmentUnavailable   EscalationTrigger = "equipment_unavailable"
	TriggerWorkerUnavailable      EscalationTrigger = "worker_unavailable"
	TriggerTimeout                EscalationTrigger = "timeout"
	TriggerQualityIssue           EscalationTrigger = "quality_issue"
)

// ProcessPath represents the determined process path for an order
type ProcessPath struct {
	ID                    string               `json:"id" bson:"_id,omitempty"`
	PathID                string               `json:"pathId" bson:"pathId"`
	TenantID    string `json:"tenantId" bson:"tenantId"`
	FacilityID  string `json:"facilityId" bson:"facilityId"`
	WarehouseID string `json:"warehouseId" bson:"warehouseId"`
	OrderID               string               `json:"orderId" bson:"orderId"`
	Requirements          []ProcessRequirement `json:"requirements" bson:"requirements"`
	ConsolidationRequired bool                 `json:"consolidationRequired" bson:"consolidationRequired"`
	GiftWrapRequired      bool                 `json:"giftWrapRequired" bson:"giftWrapRequired"`
	SpecialHandling       []string             `json:"specialHandling" bson:"specialHandling"`
	TargetStationID       string               `json:"targetStationId,omitempty" bson:"targetStationId,omitempty"`
	// Conditional Path Escalation
	Tier                  ProcessPathTier      `json:"tier" bson:"tier"`
	EscalationHistory     []EscalationEvent    `json:"escalationHistory,omitempty" bson:"escalationHistory,omitempty"`
	FallbackStationIDs    []string             `json:"fallbackStationIds,omitempty" bson:"fallbackStationIds,omitempty"`
	CreatedAt             time.Time            `json:"createdAt" bson:"createdAt"`
	UpdatedAt             time.Time            `json:"updatedAt" bson:"updatedAt"`
}

// EscalationEvent records a path tier change
type EscalationEvent struct {
	FromTier      ProcessPathTier   `json:"fromTier" bson:"fromTier"`
	ToTier        ProcessPathTier   `json:"toTier" bson:"toTier"`
	Trigger       EscalationTrigger `json:"trigger" bson:"trigger"`
	Reason        string            `json:"reason" bson:"reason"`
	EscalatedAt   time.Time         `json:"escalatedAt" bson:"escalatedAt"`
	EscalatedBy   string            `json:"escalatedBy,omitempty" bson:"escalatedBy,omitempty"`
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
		PathID:            uuid.New().String(),
		OrderID:           input.OrderID,
		Requirements:      make([]ProcessRequirement, 0),
		SpecialHandling:   make([]string, 0),
		Tier:              TierOptimal, // Start at optimal tier
		EscalationHistory: make([]EscalationEvent, 0),
		FallbackStationIDs: make([]string, 0),
		CreatedAt:         now,
		UpdatedAt:         now,
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

// Escalate escalates the process path to a worse tier
func (p *ProcessPath) Escalate(toTier ProcessPathTier, trigger EscalationTrigger, reason string, escalatedBy string) {
	if p.Tier == toTier {
		return // Already at target tier
	}

	event := EscalationEvent{
		FromTier:    p.Tier,
		ToTier:      toTier,
		Trigger:     trigger,
		Reason:      reason,
		EscalatedAt: time.Now(),
		EscalatedBy: escalatedBy,
	}

	p.Tier = toTier
	p.EscalationHistory = append(p.EscalationHistory, event)
	p.UpdatedAt = time.Now()
}

// CanDowngrade checks if the path can be downgraded to a better tier
func (p *ProcessPath) CanDowngrade() bool {
	// Can only downgrade if not at optimal tier
	return p.Tier != TierOptimal
}

// Downgrade attempts to improve the path tier (opposite of escalate)
func (p *ProcessPath) Downgrade(toTier ProcessPathTier, reason string, downgradedBy string) {
	if p.Tier == toTier {
		return
	}

	// Validate it's actually a downgrade (improvement)
	tierOrder := map[ProcessPathTier]int{
		TierOptimal:  0,
		TierStandard: 1,
		TierDegraded: 2,
		TierManual:   3,
	}

	if tierOrder[toTier] >= tierOrder[p.Tier] {
		return // Not an improvement
	}

	event := EscalationEvent{
		FromTier:    p.Tier,
		ToTier:      toTier,
		Trigger:     "", // No trigger for improvements
		Reason:      reason,
		EscalatedAt: time.Now(),
		EscalatedBy: downgradedBy,
	}

	p.Tier = toTier
	p.EscalationHistory = append(p.EscalationHistory, event)
	p.UpdatedAt = time.Now()
}

// IsOptimal checks if the path is at optimal tier
func (p *ProcessPath) IsOptimal() bool {
	return p.Tier == TierOptimal
}

// IsManual checks if the path requires manual intervention
func (p *ProcessPath) IsManual() bool {
	return p.Tier == TierManual
}

// GetEscalationCount returns the number of times the path has been escalated
func (p *ProcessPath) GetEscalationCount() int {
	return len(p.EscalationHistory)
}

// AddFallbackStation adds a fallback station to the list
func (p *ProcessPath) AddFallbackStation(stationID string) {
	// Check if already exists
	for _, id := range p.FallbackStationIDs {
		if id == stationID {
			return
		}
	}
	p.FallbackStationIDs = append(p.FallbackStationIDs, stationID)
	p.UpdatedAt = time.Now()
}

// GetNextFallbackStation returns the next available fallback station
func (p *ProcessPath) GetNextFallbackStation() string {
	if len(p.FallbackStationIDs) == 0 {
		return ""
	}
	return p.FallbackStationIDs[0]
}

// RemoveFallbackStation removes a fallback station from the list
func (p *ProcessPath) RemoveFallbackStation(stationID string) {
	newList := make([]string, 0, len(p.FallbackStationIDs))
	for _, id := range p.FallbackStationIDs {
		if id != stationID {
			newList = append(newList, id)
		}
	}
	p.FallbackStationIDs = newList
	p.UpdatedAt = time.Now()
}
