package domain

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ProcessPath represents a persisted process path for an order
// This ensures all units of an order follow the same warehouse path
type ProcessPath struct {
	ID                    primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	PathID                string               `bson:"pathId" json:"pathId"`
	OrderID               string               `bson:"orderId" json:"orderId"`
	Requirements          []ProcessRequirement `bson:"requirements" json:"requirements"`
	ConsolidationRequired bool                 `bson:"consolidationRequired" json:"consolidationRequired"`
	GiftWrapRequired      bool                 `bson:"giftWrapRequired" json:"giftWrapRequired"`
	SpecialHandling       []string             `bson:"specialHandling" json:"specialHandling"`
	TargetStationID       string               `bson:"targetStationId,omitempty" json:"targetStationId,omitempty"`
	Version               int                  `bson:"version" json:"version"`
	CreatedAt             time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt             time.Time            `bson:"updatedAt" json:"updatedAt"`
}

// NewProcessPath creates a new process path for an order
func NewProcessPath(orderID string, requirements []ProcessRequirement, consolidationRequired, giftWrapRequired bool, specialHandling []string) *ProcessPath {
	now := time.Now()
	return &ProcessPath{
		PathID:                uuid.New().String(),
		OrderID:               orderID,
		Requirements:          requirements,
		ConsolidationRequired: consolidationRequired,
		GiftWrapRequired:      giftWrapRequired,
		SpecialHandling:       specialHandling,
		Version:               1,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

// SetTargetStation sets the target station for this path
func (p *ProcessPath) SetTargetStation(stationID string) {
	p.TargetStationID = stationID
	p.UpdatedAt = time.Now()
}

// HasRequirement checks if the path has a specific requirement
func (p *ProcessPath) HasRequirement(req ProcessRequirement) bool {
	for _, r := range p.Requirements {
		if r == req {
			return true
		}
	}
	return false
}

// GetRequirementsAsStrings returns requirements as string slice (for API compatibility)
func (p *ProcessPath) GetRequirementsAsStrings() []string {
	result := make([]string, len(p.Requirements))
	for i, r := range p.Requirements {
		result[i] = string(r)
	}
	return result
}
