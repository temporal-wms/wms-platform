package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// EscalationActivities handles process path escalation logic
type EscalationActivities struct {
	clients *clients.ServiceClients
}

// NewEscalationActivities creates a new EscalationActivities instance
func NewEscalationActivities(clients *clients.ServiceClients) *EscalationActivities {
	return &EscalationActivities{
		clients: clients,
	}
}

// EscalateProcessPathInput represents input for escalating a process path
type EscalateProcessPathInput struct {
	PathID      string `json:"pathId"`
	OrderID     string `json:"orderId"`
	FromTier    string `json:"fromTier"`
	ToTier      string `json:"toTier"`
	Trigger     string `json:"trigger"`
	Reason      string `json:"reason"`
	EscalatedBy string `json:"escalatedBy,omitempty"`
}

// EscalateProcessPathResult represents the result of escalation
type EscalateProcessPathResult struct {
	PathID           string    `json:"pathId"`
	NewTier          string    `json:"newTier"`
	EscalatedAt      time.Time `json:"escalatedAt"`
	FallbackStations []string  `json:"fallbackStations,omitempty"`
	Success          bool      `json:"success"`
}

// EscalateProcessPath escalates a process path to a worse tier
func (a *EscalationActivities) EscalateProcessPath(ctx context.Context, input EscalateProcessPathInput) (*EscalateProcessPathResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Escalating process path",
		"pathId", input.PathID,
		"orderId", input.OrderID,
		"fromTier", input.FromTier,
		"toTier", input.ToTier,
		"trigger", input.Trigger,
	)

	// Call process-path-service to escalate
	req := &clients.EscalateProcessPathRequest{
		PathID:      input.PathID,
		ToTier:      input.ToTier,
		Trigger:     input.Trigger,
		Reason:      input.Reason,
		EscalatedBy: input.EscalatedBy,
	}

	result, err := a.clients.EscalateProcessPath(ctx, req)
	if err != nil {
		logger.Error("Failed to escalate process path",
			"pathId", input.PathID,
			"error", err,
		)
		return &EscalateProcessPathResult{
			Success: false,
		}, fmt.Errorf("failed to escalate process path: %w", err)
	}

	logger.Info("Process path escalated successfully",
		"pathId", input.PathID,
		"newTier", result.NewTier,
		"fallbackStationsCount", len(result.FallbackStations),
	)

	return &EscalateProcessPathResult{
		PathID:           result.PathID,
		NewTier:          result.NewTier,
		EscalatedAt:      result.EscalatedAt,
		FallbackStations: result.FallbackStations,
		Success:          true,
	}, nil
}

// DetermineEscalationTierInput represents input for determining escalation tier
type DetermineEscalationTierInput struct {
	PathID               string   `json:"pathId"`
	OrderID              string   `json:"orderId"`
	StationUnavailable   bool     `json:"stationUnavailable"`
	CapacityExceeded     bool     `json:"capacityExceeded"`
	EquipmentUnavailable bool     `json:"equipmentUnavailable"`
	WorkerUnavailable    bool     `json:"workerUnavailable"`
	Timeout              bool     `json:"timeout"`
	Requirements         []string `json:"requirements"`
}

// DetermineEscalationTierResult represents the recommended escalation tier
type DetermineEscalationTierResult struct {
	RecommendedTier string   `json:"recommendedTier"`
	Trigger         string   `json:"trigger"`
	Reason          string   `json:"reason"`
	Priority        int      `json:"priority"` // 1=high, 2=medium, 3=low
	FallbackStations []string `json:"fallbackStations,omitempty"`
}

// DetermineEscalationTier analyzes the situation and recommends an escalation tier
func (a *EscalationActivities) DetermineEscalationTier(ctx context.Context, input DetermineEscalationTierInput) (*DetermineEscalationTierResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Determining escalation tier",
		"orderId", input.OrderID,
		"stationUnavailable", input.StationUnavailable,
		"capacityExceeded", input.CapacityExceeded,
		"equipmentUnavailable", input.EquipmentUnavailable,
		"workerUnavailable", input.WorkerUnavailable,
	)

	// Priority escalation logic
	if input.StationUnavailable {
		// Most severe - station is completely unavailable
		return &DetermineEscalationTierResult{
			RecommendedTier: "degraded",
			Trigger:         "station_unavailable",
			Reason:          "Target station is unavailable, attempting fallback stations",
			Priority:        1,
		}, nil
	}

	if input.EquipmentUnavailable {
		// High severity - cannot process without equipment
		hasSpecialHandling := false
		for _, req := range input.Requirements {
			if req == "hazmat" || req == "cold_chain" {
				hasSpecialHandling = true
				break
			}
		}

		if hasSpecialHandling {
			// Critical equipment for hazmat/cold chain
			return &DetermineEscalationTierResult{
				RecommendedTier: "manual",
				Trigger:         "equipment_unavailable",
				Reason:          "Critical equipment unavailable for special handling requirements",
				Priority:        1,
			}, nil
		}

		return &DetermineEscalationTierResult{
			RecommendedTier: "degraded",
			Trigger:         "equipment_unavailable",
			Reason:          "Equipment unavailable, manual workaround required",
			Priority:        2,
		}, nil
	}

	if input.WorkerUnavailable {
		// Medium severity - can potentially queue
		return &DetermineEscalationTierResult{
			RecommendedTier: "standard",
			Trigger:         "worker_unavailable",
			Reason:          "No certified workers available, queuing for next available",
			Priority:        2,
		}, nil
	}

	if input.CapacityExceeded {
		// Medium severity - can queue or route to alternate
		return &DetermineEscalationTierResult{
			RecommendedTier: "standard",
			Trigger:         "capacity_exceeded",
			Reason:          "Station capacity exceeded, attempting alternate routing",
			Priority:        2,
		}, nil
	}

	if input.Timeout {
		// Lower severity - retry or escalate
		return &DetermineEscalationTierResult{
			RecommendedTier: "degraded",
			Trigger:         "timeout",
			Reason:          "Timeout waiting for resource allocation",
			Priority:        3,
		}, nil
	}

	// No escalation needed
	logger.Info("No escalation needed", "orderId", input.OrderID)
	return &DetermineEscalationTierResult{
		RecommendedTier: "optimal",
		Trigger:         "",
		Reason:          "No constraints detected",
		Priority:        0,
	}, nil
}

// FindFallbackStationsInput represents input for finding fallback stations
type FindFallbackStationsInput struct {
	PathID            string   `json:"pathId"`
	OrderID           string   `json:"orderId"`
	FailedStationID   string   `json:"failedStationId"`
	Requirements      []string `json:"requirements"`
	SpecialHandling   []string `json:"specialHandling"`
	FacilityID        string   `json:"facilityId"`
	MaxAlternates     int      `json:"maxAlternates"` // Max number of fallback stations to find
}

// FindFallbackStationsResult represents fallback station options
type FindFallbackStationsResult struct {
	FallbackStations []FallbackStationInfo `json:"fallbackStations"`
	Success          bool                  `json:"success"`
}

// FallbackStationInfo represents a fallback station option
type FallbackStationInfo struct {
	StationID  string  `json:"stationId"`
	Score      float64 `json:"score"`
	Rank       int     `json:"rank"`
	Confidence float64 `json:"confidence"`
	Distance   float64 `json:"distance,omitempty"`
}

// FindFallbackStations finds alternative stations when primary station fails
func (a *EscalationActivities) FindFallbackStations(ctx context.Context, input FindFallbackStationsInput) (*FindFallbackStationsResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Finding fallback stations",
		"orderId", input.OrderID,
		"failedStationId", input.FailedStationID,
		"maxAlternates", input.MaxAlternates,
	)

	// Use routing optimizer to find alternate stations
	req := &clients.OptimizeRoutingRequest{
		OrderID:         input.OrderID,
		Requirements:    input.Requirements,
		SpecialHandling: input.SpecialHandling,
		Priority:        "high", // Escalated orders are high priority
		StationType:     "", // Open to any capable station
		// Exclude the failed station
	}

	decision, err := a.clients.OptimizeRouting(ctx, req)
	if err != nil {
		logger.Error("Failed to find fallback stations",
			"orderId", input.OrderID,
			"error", err,
		)
		return &FindFallbackStationsResult{
			Success: false,
		}, fmt.Errorf("failed to optimize routing for fallbacks: %w", err)
	}

	// Convert alternates to fallback info
	maxAlternates := input.MaxAlternates
	if maxAlternates == 0 {
		maxAlternates = 3 // Default to 3 fallback stations
	}

	fallbacks := make([]FallbackStationInfo, 0, maxAlternates)

	// Exclude the failed station from results
	for _, alt := range decision.AlternateStations {
		if alt.StationID == input.FailedStationID {
			continue
		}

		if len(fallbacks) >= maxAlternates {
			break
		}

		fallbacks = append(fallbacks, FallbackStationInfo{
			StationID:  alt.StationID,
			Score:      alt.Score,
			Rank:       alt.Rank,
			Confidence: decision.Confidence,
		})
	}

	logger.Info("Found fallback stations",
		"orderId", input.OrderID,
		"fallbackCount", len(fallbacks),
	)

	return &FindFallbackStationsResult{
		FallbackStations: fallbacks,
		Success:          len(fallbacks) > 0,
	}, nil
}

// DowngradeProcessPathInput represents input for downgrading a process path
type DowngradeProcessPathInput struct {
	PathID        string `json:"pathId"`
	OrderID       string `json:"orderId"`
	ToTier        string `json:"toTier"`
	Reason        string `json:"reason"`
	DowngradedBy  string `json:"downgradedBy,omitempty"`
}

// DowngradeProcessPathResult represents the result of downgrade
type DowngradeProcessPathResult struct {
	PathID      string    `json:"pathId"`
	NewTier     string    `json:"newTier"`
	DowngradedAt time.Time `json:"downgradedAt"`
	Success     bool      `json:"success"`
}

// DowngradeProcessPath improves a process path to a better tier (opposite of escalate)
func (a *EscalationActivities) DowngradeProcessPath(ctx context.Context, input DowngradeProcessPathInput) (*DowngradeProcessPathResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Downgrading process path to better tier",
		"pathId", input.PathID,
		"orderId", input.OrderID,
		"toTier", input.ToTier,
	)

	// Call process-path-service to downgrade
	req := &clients.DowngradeProcessPathRequest{
		PathID:       input.PathID,
		ToTier:       input.ToTier,
		Reason:       input.Reason,
		DowngradedBy: input.DowngradedBy,
	}

	result, err := a.clients.DowngradeProcessPath(ctx, req)
	if err != nil {
		logger.Error("Failed to downgrade process path",
			"pathId", input.PathID,
			"error", err,
		)
		return &DowngradeProcessPathResult{
			Success: false,
		}, fmt.Errorf("failed to downgrade process path: %w", err)
	}

	logger.Info("Process path downgraded successfully",
		"pathId", input.PathID,
		"newTier", result.NewTier,
	)

	return &DowngradeProcessPathResult{
		PathID:       result.PathID,
		NewTier:      result.NewTier,
		DowngradedAt: result.DowngradedAt,
		Success:      true,
	}, nil
}
