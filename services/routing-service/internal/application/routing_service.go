package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/routing-service/internal/domain"
)

// RoutingApplicationService handles routing-related use cases
type RoutingApplicationService struct {
	repo            domain.RouteRepository
	routeCalculator *RouteCalculator
	producer        *kafka.InstrumentedProducer
	eventFactory    *cloudevents.EventFactory
	logger          *logging.Logger
}

// NewRoutingApplicationService creates a new RoutingApplicationService
func NewRoutingApplicationService(
	repo domain.RouteRepository,
	routeCalculator *RouteCalculator,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	logger *logging.Logger,
) *RoutingApplicationService {
	return &RoutingApplicationService{
		repo:            repo,
		routeCalculator: routeCalculator,
		producer:        producer,
		eventFactory:    eventFactory,
		logger:          logger,
	}
}

// CalculateRoute calculates a new route
func (s *RoutingApplicationService) CalculateRoute(ctx context.Context, cmd CalculateRouteCommand) (*PickRouteDTO, error) {
	route, err := s.routeCalculator.CalculateRoute(ctx, cmd.RouteRequest)
	if err != nil {
		s.logger.WithError(err).Error("Failed to calculate route")
		return nil, fmt.Errorf("failed to calculate route: %w", err)
	}

	if err := s.repo.Save(ctx, route); err != nil {
		s.logger.WithError(err).Error("Failed to save route", "routeId", route.RouteID)
		return nil, fmt.Errorf("failed to save route: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	// Log business event: route created
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "route.created",
		EntityType: "route",
		EntityID:   route.RouteID,
		Action:     "created",
		RelatedIDs: map[string]string{
			"orderId": cmd.RouteRequest.OrderID,
			"waveId":  cmd.RouteRequest.WaveID,
		},
	})

	return ToPickRouteDTO(route), nil
}

// CalculateMultiRoute calculates multiple routes for an order with zone and capacity splitting
func (s *RoutingApplicationService) CalculateMultiRoute(ctx context.Context, cmd CalculateMultiRouteCommand) (*MultiRouteResultDTO, error) {
	result, err := s.routeCalculator.CalculateRoutes(ctx, cmd.RouteRequest)
	if err != nil {
		s.logger.WithError(err).Error("Failed to calculate multi-route")
		return nil, fmt.Errorf("failed to calculate multi-route: %w", err)
	}

	// Save all routes
	for _, route := range result.Routes {
		if err := s.repo.Save(ctx, route); err != nil {
			s.logger.WithError(err).Error("Failed to save route", "routeId", route.RouteID)
			return nil, fmt.Errorf("failed to save route %s: %w", route.RouteID, err)
		}
	}

	// Log business event: multi-route created
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "route.multi_created",
		EntityType: "route",
		EntityID:   cmd.RouteRequest.OrderID,
		Action:     "multi_route_created",
		RelatedIDs: map[string]string{
			"orderId":     cmd.RouteRequest.OrderID,
			"waveId":      cmd.RouteRequest.WaveID,
			"totalRoutes": fmt.Sprintf("%d", result.TotalRoutes),
			"splitReason": string(result.SplitReason),
		},
	})

	s.logger.Info("Multi-route calculated",
		"orderId", cmd.RouteRequest.OrderID,
		"totalRoutes", result.TotalRoutes,
		"splitReason", result.SplitReason,
		"totalItems", result.TotalItems,
	)

	return ToMultiRouteResultDTO(result), nil
}

// GetRoute retrieves a route by ID
func (s *RoutingApplicationService) GetRoute(ctx context.Context, query GetRouteQuery) (*PickRouteDTO, error) {
	route, err := s.repo.FindByID(ctx, query.RouteID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get route", "routeId", query.RouteID)
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	if route == nil {
		return nil, errors.ErrNotFound("route")
	}

	return ToPickRouteDTO(route), nil
}

// DeleteRoute deletes a route
func (s *RoutingApplicationService) DeleteRoute(ctx context.Context, cmd DeleteRouteCommand) error {
	if err := s.repo.Delete(ctx, cmd.RouteID); err != nil {
		s.logger.WithError(err).Error("Failed to delete route", "routeId", cmd.RouteID)
		return fmt.Errorf("failed to delete route: %w", err)
	}

	s.logger.Info("Deleted route", "routeId", cmd.RouteID)
	return nil
}

// StartRoute starts a route
func (s *RoutingApplicationService) StartRoute(ctx context.Context, cmd StartRouteCommand) (*PickRouteDTO, error) {
	route, err := s.repo.FindByID(ctx, cmd.RouteID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	if route == nil {
		return nil, errors.ErrNotFound("route")
	}

	if err := route.Start(cmd.PickerID); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, route); err != nil {
		s.logger.WithError(err).Error("Failed to save route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to save route: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Started route", "routeId", cmd.RouteID, "pickerId", cmd.PickerID)
	return ToPickRouteDTO(route), nil
}

// CompleteStop completes a stop in the route
func (s *RoutingApplicationService) CompleteStop(ctx context.Context, cmd CompleteStopCommand) (*PickRouteDTO, error) {
	route, err := s.repo.FindByID(ctx, cmd.RouteID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	if route == nil {
		return nil, errors.ErrNotFound("route")
	}

	if err := route.CompleteStop(cmd.StopNumber, cmd.PickedQty, cmd.ToteID); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, route); err != nil {
		s.logger.WithError(err).Error("Failed to save route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to save route: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Completed stop", "routeId", cmd.RouteID, "stopNumber", cmd.StopNumber)
	return ToPickRouteDTO(route), nil
}

// SkipStop skips a stop in the route
func (s *RoutingApplicationService) SkipStop(ctx context.Context, cmd SkipStopCommand) (*PickRouteDTO, error) {
	route, err := s.repo.FindByID(ctx, cmd.RouteID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	if route == nil {
		return nil, errors.ErrNotFound("route")
	}

	if err := route.SkipStop(cmd.StopNumber, cmd.Reason); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, route); err != nil {
		s.logger.WithError(err).Error("Failed to save route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to save route: %w", err)
	}

	s.logger.Info("Skipped stop", "routeId", cmd.RouteID, "stopNumber", cmd.StopNumber, "reason", cmd.Reason)
	return ToPickRouteDTO(route), nil
}

// CompleteRoute completes a route
func (s *RoutingApplicationService) CompleteRoute(ctx context.Context, cmd CompleteRouteCommand) (*PickRouteDTO, error) {
	route, err := s.repo.FindByID(ctx, cmd.RouteID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	if route == nil {
		return nil, errors.ErrNotFound("route")
	}

	if err := route.Complete(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, route); err != nil {
		s.logger.WithError(err).Error("Failed to save route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to save route: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	// Log business event: route completed
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "route.completed",
		EntityType: "route",
		EntityID:   cmd.RouteID,
		Action:     "completed",
		RelatedIDs: map[string]string{
			"orderId": route.OrderID,
		},
	})

	return ToPickRouteDTO(route), nil
}

// PauseRoute pauses a route
func (s *RoutingApplicationService) PauseRoute(ctx context.Context, cmd PauseRouteCommand) (*PickRouteDTO, error) {
	route, err := s.repo.FindByID(ctx, cmd.RouteID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	if route == nil {
		return nil, errors.ErrNotFound("route")
	}

	if err := route.Pause(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, route); err != nil {
		s.logger.WithError(err).Error("Failed to save route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to save route: %w", err)
	}

	s.logger.Info("Paused route", "routeId", cmd.RouteID)
	return ToPickRouteDTO(route), nil
}

// CancelRoute cancels a route
func (s *RoutingApplicationService) CancelRoute(ctx context.Context, cmd CancelRouteCommand) (*PickRouteDTO, error) {
	route, err := s.repo.FindByID(ctx, cmd.RouteID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	if route == nil {
		return nil, errors.ErrNotFound("route")
	}

	if err := route.Cancel(cmd.Reason); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, route); err != nil {
		s.logger.WithError(err).Error("Failed to save route", "routeId", cmd.RouteID)
		return nil, fmt.Errorf("failed to save route: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Cancelled route", "routeId", cmd.RouteID, "reason", cmd.Reason)
	return ToPickRouteDTO(route), nil
}

// GetRoutesByOrder retrieves routes by order ID
func (s *RoutingApplicationService) GetRoutesByOrder(ctx context.Context, query GetRoutesByOrderQuery) ([]PickRouteDTO, error) {
	routes, err := s.repo.FindByOrderID(ctx, query.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get routes by order", "orderId", query.OrderID)
		return nil, fmt.Errorf("failed to get routes by order: %w", err)
	}

	return ToPickRouteDTOs(routes), nil
}

// GetRoutesByWave retrieves routes by wave ID
func (s *RoutingApplicationService) GetRoutesByWave(ctx context.Context, query GetRoutesByWaveQuery) ([]PickRouteDTO, error) {
	routes, err := s.repo.FindByWaveID(ctx, query.WaveID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get routes by wave", "waveId", query.WaveID)
		return nil, fmt.Errorf("failed to get routes by wave: %w", err)
	}

	return ToPickRouteDTOs(routes), nil
}

// GetRoutesByPicker retrieves routes by picker ID
func (s *RoutingApplicationService) GetRoutesByPicker(ctx context.Context, query GetRoutesByPickerQuery) ([]PickRouteDTO, error) {
	routes, err := s.repo.FindByPickerID(ctx, query.PickerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get routes by picker", "pickerId", query.PickerID)
		return nil, fmt.Errorf("failed to get routes by picker: %w", err)
	}

	return ToPickRouteDTOs(routes), nil
}

// GetActiveRoute retrieves active route for a picker
func (s *RoutingApplicationService) GetActiveRoute(ctx context.Context, query GetActiveRouteQuery) (*PickRouteDTO, error) {
	route, err := s.repo.FindActiveByPicker(ctx, query.PickerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get active route", "pickerId", query.PickerID)
		return nil, fmt.Errorf("failed to get active route: %w", err)
	}

	if route == nil {
		return nil, errors.ErrNotFound("active route")
	}

	return ToPickRouteDTO(route), nil
}

// GetRoutesByStatus retrieves routes by status
func (s *RoutingApplicationService) GetRoutesByStatus(ctx context.Context, query GetRoutesByStatusQuery) ([]PickRouteDTO, error) {
	routes, err := s.repo.FindByStatus(ctx, query.Status)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get routes by status", "status", query.Status)
		return nil, fmt.Errorf("failed to get routes by status: %w", err)
	}

	return ToPickRouteDTOs(routes), nil
}

// GetPendingRoutes retrieves pending routes
func (s *RoutingApplicationService) GetPendingRoutes(ctx context.Context, query GetPendingRoutesQuery) ([]PickRouteDTO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50 // Default limit
	}

	routes, err := s.repo.FindPendingRoutes(ctx, query.Zone, limit)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pending routes")
		return nil, fmt.Errorf("failed to get pending routes: %w", err)
	}

	return ToPickRouteDTOs(routes), nil
}

// AnalyzeRoute analyzes route efficiency
func (s *RoutingApplicationService) AnalyzeRoute(ctx context.Context, query AnalyzeRouteQuery) (*RouteAnalysis, error) {
	route, err := s.repo.FindByID(ctx, query.RouteID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get route", "routeId", query.RouteID)
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	if route == nil {
		return nil, errors.ErrNotFound("route")
	}

	analysis, err := s.routeCalculator.AnalyzeRouteEfficiency(ctx, route)
	if err != nil {
		s.logger.WithError(err).Error("Failed to analyze route", "routeId", query.RouteID)
		return nil, fmt.Errorf("failed to analyze route: %w", err)
	}

	return analysis, nil
}

// SuggestStrategy suggests routing strategy for items
func (s *RoutingApplicationService) SuggestStrategy(ctx context.Context, query SuggestStrategyQuery) (domain.RoutingStrategy, error) {
	strategy, err := s.routeCalculator.SuggestStrategy(ctx, query.Items)
	if err != nil {
		s.logger.WithError(err).Error("Failed to suggest strategy")
		return "", fmt.Errorf("failed to suggest strategy: %w", err)
	}

	return strategy, nil
}
