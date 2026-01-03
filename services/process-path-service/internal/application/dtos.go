package application

import (
	"time"

	"github.com/wms-platform/process-path-service/internal/domain"
)

// DetermineProcessPathCommand represents the command to determine a process path
type DetermineProcessPathCommand struct {
	OrderID          string                    `json:"orderId" binding:"required"`
	Items            []domain.ProcessPathItem  `json:"items" binding:"required"`
	GiftWrap         bool                      `json:"giftWrap"`
	GiftWrapDetails  *domain.GiftWrapDetails   `json:"giftWrapDetails,omitempty"`
	HazmatDetails    *domain.HazmatDetails     `json:"hazmatDetails,omitempty"`
	ColdChainDetails *domain.ColdChainDetails  `json:"coldChainDetails,omitempty"`
	TotalValue       float64                   `json:"totalValue"`
}

// AssignStationCommand represents the command to assign a station
type AssignStationCommand struct {
	PathID    string `json:"pathId" binding:"required"`
	StationID string `json:"stationId" binding:"required"`
}

// ProcessPathDTO represents the response DTO for a process path
type ProcessPathDTO struct {
	PathID                string                       `json:"pathId"`
	OrderID               string                       `json:"orderId"`
	Requirements          []domain.ProcessRequirement  `json:"requirements"`
	ConsolidationRequired bool                         `json:"consolidationRequired"`
	GiftWrapRequired      bool                         `json:"giftWrapRequired"`
	SpecialHandling       []string                     `json:"specialHandling"`
	TargetStationID       string                       `json:"targetStationId,omitempty"`
	CreatedAt             time.Time                    `json:"createdAt"`
	UpdatedAt             time.Time                    `json:"updatedAt"`
}

// ToDTO converts a domain ProcessPath to a DTO
func ToDTO(p *domain.ProcessPath) *ProcessPathDTO {
	return &ProcessPathDTO{
		PathID:                p.PathID,
		OrderID:               p.OrderID,
		Requirements:          p.Requirements,
		ConsolidationRequired: p.ConsolidationRequired,
		GiftWrapRequired:      p.GiftWrapRequired,
		SpecialHandling:       p.SpecialHandling,
		TargetStationID:       p.TargetStationID,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}
}
