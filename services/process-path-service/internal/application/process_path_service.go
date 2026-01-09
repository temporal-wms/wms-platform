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
