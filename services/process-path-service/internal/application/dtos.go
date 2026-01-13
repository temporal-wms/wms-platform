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

// OptimizeRoutingCommand represents the command for routing optimization
type OptimizeRoutingCommand struct {
	OrderID            string    `json:"orderId" binding:"required"`
	Priority           string    `json:"priority" binding:"required"`
	Requirements       []string  `json:"requirements"`
	SpecialHandling    []string  `json:"specialHandling"`
	ItemCount          int       `json:"itemCount"`
	TotalWeight        float64   `json:"totalWeight"`
	PromisedDeliveryAt time.Time `json:"promisedDeliveryAt"`
	RequiredSkills     []string  `json:"requiredSkills"`
	RequiredEquipment  []string  `json:"requiredEquipment"`
	Zone               string    `json:"zone,omitempty"`
	StationType        string    `json:"stationType"`
}

// RoutingDecisionDTO represents a routing optimization decision
type RoutingDecisionDTO struct {
	SelectedStationID string                     `json:"selectedStationId"`
	Score             float64                    `json:"score"`
	Reasoning         map[string]float64         `json:"reasoning"`
	AlternateStations []AlternateStationResponse `json:"alternateStations"`
	Confidence        float64                    `json:"confidence"`
	DecisionTime      time.Time                  `json:"decisionTime"`
}

// AlternateStationResponse represents an alternate station option
type AlternateStationResponse struct {
	StationID string  `json:"stationId"`
	Score     float64 `json:"score"`
	Rank      int     `json:"rank"`
}

// ToRoutingDecisionDTO converts domain RoutingDecision to DTO
func ToRoutingDecisionDTO(d *domain.RoutingDecision) *RoutingDecisionDTO {
	alternates := make([]AlternateStationResponse, len(d.AlternateStations))
	for i, alt := range d.AlternateStations {
		alternates[i] = AlternateStationResponse{
			StationID: alt.StationID,
			Score:     alt.Score,
			Rank:      alt.Rank,
		}
	}

	return &RoutingDecisionDTO{
		SelectedStationID: d.SelectedStationID,
		Score:             d.Score,
		Reasoning:         d.Reasoning,
		AlternateStations: alternates,
		Confidence:        d.Confidence,
		DecisionTime:      d.DecisionTime,
	}
}

// RoutingMetricsDTO represents routing performance metrics
type RoutingMetricsDTO struct {
	TotalRoutingDecisions   int                `json:"totalRoutingDecisions"`
	AverageDecisionTimeMs   int64              `json:"averageDecisionTimeMs"`
	AverageConfidence       float64            `json:"averageConfidence"`
	StationUtilization      map[string]float64 `json:"stationUtilization"`
	CapacityConstrainedRate float64            `json:"capacityConstrainedRate"`
	RouteChanges            int                `json:"routeChanges"`
	RebalancingRecommended  bool               `json:"rebalancingRecommended"`
	LastUpdated             time.Time          `json:"lastUpdated"`
}

// ToRoutingMetricsDTO converts domain DynamicRoutingMetrics to DTO
func ToRoutingMetricsDTO(m *domain.DynamicRoutingMetrics) *RoutingMetricsDTO {
	return &RoutingMetricsDTO{
		TotalRoutingDecisions:   m.TotalRoutingDecisions,
		AverageDecisionTimeMs:   m.AverageDecisionTimeMs,
		AverageConfidence:       m.AverageConfidence,
		StationUtilization:      m.StationUtilization,
		CapacityConstrainedRate: m.CapacityConstrainedRate,
		RouteChanges:            m.RouteChanges,
		RebalancingRecommended:  m.RebalancingRecommended,
		LastUpdated:             time.Now(),
	}
}

// RerouteOrderCommand represents the command to reroute an order
type RerouteOrderCommand struct {
	OrderID      string   `json:"orderId" binding:"required"`
	CurrentPath  string   `json:"currentPath" binding:"required"`
	Reason       string   `json:"reason" binding:"required"`
	Requirements []string `json:"requirements"`
	Priority     string   `json:"priority"`
	ForceReroute bool     `json:"forceReroute"`
}

// ReroutingDecisionDTO represents a rerouting decision
type ReroutingDecisionDTO struct {
	OrderID          string  `json:"orderId"`
	OldStationID     string  `json:"oldStationId"`
	NewStationID     string  `json:"newStationId"`
	Score            float64 `json:"score"`
	Confidence       float64 `json:"confidence"`
	Reason           string  `json:"reason"`
	ImprovementScore float64 `json:"improvementScore"`
}

// EscalateProcessPathCommand represents the command to escalate a process path
type EscalateProcessPathCommand struct {
	PathID      string `json:"pathId" binding:"required"`
	ToTier      string `json:"toTier" binding:"required"`
	Trigger     string `json:"trigger" binding:"required"`
	Reason      string `json:"reason" binding:"required"`
	EscalatedBy string `json:"escalatedBy"`
}

// DowngradeProcessPathCommand represents the command to downgrade a process path
type DowngradeProcessPathCommand struct {
	PathID       string `json:"pathId" binding:"required"`
	ToTier       string `json:"toTier" binding:"required"`
	Reason       string `json:"reason" binding:"required"`
	DowngradedBy string `json:"downgradedBy"`
}
