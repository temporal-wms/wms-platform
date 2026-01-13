package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/process-path-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
)

// ProcessPathService handles process path business logic
type ProcessPathService struct {
	repo   domain.ProcessPathRepository
	logger *logging.Logger
}

// NewProcessPathService creates a new process path application service
func NewProcessPathService(
	repo domain.ProcessPathRepository,
	logger *logging.Logger,
) *ProcessPathService {
	return &ProcessPathService{
		repo:   repo,
		logger: logger,
	}
}

// DetermineProcessPath analyzes order characteristics and determines the process path
func (s *ProcessPathService) DetermineProcessPath(ctx context.Context, cmd DetermineProcessPathCommand) (*ProcessPathDTO, error) {
	s.logger.Info("Determining process path", "orderId", cmd.OrderID)

	// Check if process path already exists for this order
	existing, err := s.repo.FindByOrderID(ctx, cmd.OrderID)
	if err == nil && existing != nil {
		s.logger.Info("Process path already exists for order", "orderId", cmd.OrderID, "pathId", existing.PathID)
		return ToDTO(existing), nil
	}

	// Create domain input
	input := domain.DetermineProcessPathInput{
		OrderID:          cmd.OrderID,
		Items:            cmd.Items,
		GiftWrap:         cmd.GiftWrap,
		GiftWrapDetails:  cmd.GiftWrapDetails,
		HazmatDetails:    cmd.HazmatDetails,
		ColdChainDetails: cmd.ColdChainDetails,
		TotalValue:       cmd.TotalValue,
	}

	// Create process path using domain logic
	processPath := domain.NewProcessPath(input)

	// Persist the process path
	if err := s.repo.Save(ctx, processPath); err != nil {
		s.logger.WithError(err).Error("Failed to save process path", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to save process path: %w", err)
	}

	s.logger.Info("Process path determined",
		"orderId", cmd.OrderID,
		"pathId", processPath.PathID,
		"requirements", processPath.Requirements,
		"consolidationRequired", processPath.ConsolidationRequired,
		"giftWrapRequired", processPath.GiftWrapRequired,
	)

	return ToDTO(processPath), nil
}

// GetProcessPath retrieves a process path by pathId
func (s *ProcessPathService) GetProcessPath(ctx context.Context, pathID string) (*ProcessPathDTO, error) {
	s.logger.Info("Getting process path", "pathId", pathID)

	processPath, err := s.repo.FindByPathID(ctx, pathID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get process path", "pathId", pathID)
		return nil, err
	}

	return ToDTO(processPath), nil
}

// GetProcessPathByOrderID retrieves a process path by order ID
func (s *ProcessPathService) GetProcessPathByOrderID(ctx context.Context, orderID string) (*ProcessPathDTO, error) {
	s.logger.Info("Getting process path by order", "orderId", orderID)

	processPath, err := s.repo.FindByOrderID(ctx, orderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get process path by order", "orderId", orderID)
		return nil, err
	}

	return ToDTO(processPath), nil
}

// AssignStation assigns a target station to a process path
func (s *ProcessPathService) AssignStation(ctx context.Context, cmd AssignStationCommand) (*ProcessPathDTO, error) {
	s.logger.Info("Assigning station to process path", "pathId", cmd.PathID, "stationId", cmd.StationID)

	processPath, err := s.repo.FindByPathID(ctx, cmd.PathID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to find process path", "pathId", cmd.PathID)
		return nil, err
	}

	processPath.AssignStation(cmd.StationID)

	if err := s.repo.Update(ctx, processPath); err != nil {
		s.logger.WithError(err).Error("Failed to update process path", "pathId", cmd.PathID)
		return nil, fmt.Errorf("failed to update process path: %w", err)
	}

	s.logger.Info("Station assigned to process path", "pathId", cmd.PathID, "stationId", cmd.StationID)

	return ToDTO(processPath), nil
}

// OptimizeRouting performs ML-like routing optimization to select the best station
func (s *ProcessPathService) OptimizeRouting(ctx context.Context, cmd OptimizeRoutingCommand) (*RoutingDecisionDTO, error) {
	s.logger.Info("Optimizing routing for order", "orderId", cmd.OrderID, "priority", cmd.Priority)

	// Create routing optimizer with weighted scoring
	optimizer := domain.NewRoutingOptimizer()

	// Build routing context from command
	routingContext := domain.OrderRoutingContext{
		OrderID:            cmd.OrderID,
		Priority:           cmd.Priority,
		Requirements:       cmd.Requirements,
		SpecialHandling:    cmd.SpecialHandling,
		ItemCount:          cmd.ItemCount,
		TotalWeight:        cmd.TotalWeight,
		PromisedDeliveryAt: cmd.PromisedDeliveryAt,
		RequiredSkills:     cmd.RequiredSkills,
		RequiredEquipment:  cmd.RequiredEquipment,
	}

	// Get candidate stations (in real implementation, would query station repository)
	// For now, return a mock decision to demonstrate the structure
	candidates := s.buildStationCandidates(cmd.StationType, cmd.Zone)

	decision := optimizer.OptimizeStationRouting(routingContext, candidates)

	s.logger.Info("Routing optimization completed",
		"orderId", cmd.OrderID,
		"selectedStation", decision.SelectedStationID,
		"score", decision.Score,
		"confidence", decision.Confidence,
	)

	return ToRoutingDecisionDTO(decision), nil
}

// buildStationCandidates creates mock station candidates for routing optimization
func (s *ProcessPathService) buildStationCandidates(stationType, zone string) []domain.StationCandidate {
	// In real implementation, would query station repository
	// This is a simplified mock for demonstration
	return []domain.StationCandidate{
		{
			StationID:         "station-1",
			AvailableCapacity: 10,
			CurrentUtilization: 0.65,
			Capabilities:      []string{"packing", "gift_wrap"},
			DistanceMeters:    100.0,
			AverageThroughput: 50.0,
			CertifiedWorkers:  5,
		},
		{
			StationID:         "station-2",
			AvailableCapacity: 8,
			CurrentUtilization: 0.45,
			Capabilities:      []string{"packing", "hazmat"},
			DistanceMeters:    150.0,
			AverageThroughput: 45.0,
			CertifiedWorkers:  3,
		},
		{
			StationID:         "station-3",
			AvailableCapacity: 15,
			CurrentUtilization: 0.30,
			Capabilities:      []string{"packing"},
			DistanceMeters:    200.0,
			AverageThroughput: 60.0,
			CertifiedWorkers:  8,
		},
	}
}

// GetRoutingMetrics retrieves real-time routing performance metrics
func (s *ProcessPathService) GetRoutingMetrics(ctx context.Context, facilityID, zone, timeWindow string) (*RoutingMetricsDTO, error) {
	s.logger.Info("Getting routing metrics",
		"facilityId", facilityID,
		"zone", zone,
		"timeWindow", timeWindow,
	)

	// In real implementation, would query metrics from database or cache
	// This is a mock for demonstration
	metrics := domain.DynamicRoutingMetrics{
		TotalRoutingDecisions:   150,
		AverageDecisionTimeMs:   45,
		AverageConfidence:       0.82,
		StationUtilization: map[string]float64{
			"station-1": 0.85,
			"station-2": 0.65,
			"station-3": 0.45,
		},
		CapacityConstrainedRate: 0.12,
		RouteChanges:            8,
		RebalancingRecommended:  true,
	}

	s.logger.Info("Routing metrics retrieved",
		"totalDecisions", metrics.TotalRoutingDecisions,
		"averageConfidence", metrics.AverageConfidence,
	)

	return ToRoutingMetricsDTO(&metrics), nil
}

// RerouteOrder dynamically reroutes an order to a better station
func (s *ProcessPathService) RerouteOrder(ctx context.Context, cmd RerouteOrderCommand) (*ReroutingDecisionDTO, error) {
	s.logger.Info("Rerouting order",
		"orderId", cmd.OrderID,
		"currentPath", cmd.CurrentPath,
		"reason", cmd.Reason,
	)

	// Get current process path
	processPath, err := s.repo.FindByOrderID(ctx, cmd.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to find process path", "orderId", cmd.OrderID)
		return nil, err
	}

	// Create routing optimizer
	optimizer := domain.NewRoutingOptimizer()

	// Build routing context
	routingContext := domain.OrderRoutingContext{
		OrderID:         cmd.OrderID,
		Priority:        cmd.Priority,
		Requirements:    cmd.Requirements,
		SpecialHandling: processPath.SpecialHandling,
	}

	// Get alternative stations (excluding current)
	candidates := s.buildStationCandidates("", "")

	// Optimize new routing
	decision := optimizer.OptimizeStationRouting(routingContext, candidates)

	// Update process path with new station
	if decision.SelectedStationID != cmd.CurrentPath {
		processPath.AssignStation(decision.SelectedStationID)
		if err := s.repo.Update(ctx, processPath); err != nil {
			s.logger.WithError(err).Error("Failed to update process path", "orderId", cmd.OrderID)
			return nil, fmt.Errorf("failed to update process path: %w", err)
		}
	}

	s.logger.Info("Order rerouted",
		"orderId", cmd.OrderID,
		"newStation", decision.SelectedStationID,
		"confidence", decision.Confidence,
	)

	return &ReroutingDecisionDTO{
		OrderID:          cmd.OrderID,
		OldStationID:     cmd.CurrentPath,
		NewStationID:     decision.SelectedStationID,
		Score:            decision.Score,
		Confidence:       decision.Confidence,
		Reason:           cmd.Reason,
		ImprovementScore: decision.Score * decision.Confidence,
	}, nil
}

// EscalateProcessPath escalates a process path to a worse tier
func (s *ProcessPathService) EscalateProcessPath(ctx context.Context, cmd EscalateProcessPathCommand) (*ProcessPathDTO, error) {
	s.logger.Info("Escalating process path",
		"pathId", cmd.PathID,
		"toTier", cmd.ToTier,
		"trigger", cmd.Trigger,
	)

	processPath, err := s.repo.FindByPathID(ctx, cmd.PathID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to find process path", "pathId", cmd.PathID)
		return nil, err
	}

	// Convert string tier to domain type
	toTier := domain.ProcessPathTier(cmd.ToTier)
	trigger := domain.EscalationTrigger(cmd.Trigger)

	// Escalate using domain method
	processPath.Escalate(toTier, trigger, cmd.Reason, cmd.EscalatedBy)

	// Update in repository
	if err := s.repo.Update(ctx, processPath); err != nil {
		s.logger.WithError(err).Error("Failed to update process path", "pathId", cmd.PathID)
		return nil, fmt.Errorf("failed to update process path: %w", err)
	}

	s.logger.Info("Process path escalated",
		"pathId", cmd.PathID,
		"newTier", toTier,
		"escalationCount", processPath.GetEscalationCount(),
	)

	return ToDTO(processPath), nil
}

// DowngradeProcessPath downgrades (improves) a process path to a better tier
func (s *ProcessPathService) DowngradeProcessPath(ctx context.Context, cmd DowngradeProcessPathCommand) (*ProcessPathDTO, error) {
	s.logger.Info("Downgrading process path",
		"pathId", cmd.PathID,
		"toTier", cmd.ToTier,
	)

	processPath, err := s.repo.FindByPathID(ctx, cmd.PathID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to find process path", "pathId", cmd.PathID)
		return nil, err
	}

	// Convert string tier to domain type
	toTier := domain.ProcessPathTier(cmd.ToTier)

	// Downgrade using domain method
	processPath.Downgrade(toTier, cmd.Reason, cmd.DowngradedBy)

	// Update in repository
	if err := s.repo.Update(ctx, processPath); err != nil {
		s.logger.WithError(err).Error("Failed to update process path", "pathId", cmd.PathID)
		return nil, fmt.Errorf("failed to update process path: %w", err)
	}

	s.logger.Info("Process path downgraded",
		"pathId", cmd.PathID,
		"newTier", toTier,
	)

	return ToDTO(processPath), nil
}
