package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/wes-service/internal/domain"
)

// WESApplicationService is the main application service for the Warehouse Execution System
type WESApplicationService struct {
	templateRepo      domain.StageTemplateRepository
	routeRepo         domain.TaskRouteRepository
	templateSelector  *TemplateSelector
	processPathClient *ProcessPathClient
	producer          *kafka.InstrumentedProducer
	eventFactory      *cloudevents.EventFactory
	logger            *slog.Logger
}

// NewWESApplicationService creates a new WES application service
func NewWESApplicationService(
	templateRepo domain.StageTemplateRepository,
	routeRepo domain.TaskRouteRepository,
	processPathClient *ProcessPathClient,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	logger *slog.Logger,
) *WESApplicationService {
	return &WESApplicationService{
		templateRepo:      templateRepo,
		routeRepo:         routeRepo,
		templateSelector:  NewTemplateSelector(templateRepo),
		processPathClient: processPathClient,
		producer:          producer,
		eventFactory:      eventFactory,
		logger:            logger,
	}
}

// ResolveExecutionPlan determines the execution plan for an order
func (s *WESApplicationService) ResolveExecutionPlan(ctx context.Context, cmd ResolveExecutionPlanCommand) (*ExecutionPlanDTO, error) {
	s.logger.Info("Resolving execution plan", "orderId", cmd.OrderID, "itemCount", len(cmd.Items))

	// Get process path from process-path-service if not provided
	var processPath *ProcessPathResultDTO
	if cmd.ProcessPathResult != nil {
		processPath = cmd.ProcessPathResult
	} else if s.processPathClient != nil {
		var err error
		processPath, err = s.processPathClient.DetermineProcessPath(ctx, cmd.OrderID, cmd.Items)
		if err != nil {
			s.logger.Warn("Failed to get process path, using defaults", "error", err)
			// Continue without process path - will use default template
		}
	}

	// Calculate total items
	itemCount := 0
	for _, item := range cmd.Items {
		itemCount += item.Quantity
	}

	// Select appropriate template
	template, err := s.templateSelector.SelectTemplateForProcessPath(ctx, processPath, itemCount, cmd.MultiZone)
	if err != nil {
		return nil, fmt.Errorf("failed to select template: %w", err)
	}

	// Build execution plan
	stages := make([]StageDefinitionDTO, len(template.Stages))
	for i, stage := range template.Stages {
		stages[i] = StageDefinitionDTO{
			Order:       stage.Order,
			StageType:   string(stage.StageType),
			TaskType:    stage.TaskType,
			Required:    stage.Required,
			TimeoutMins: stage.TimeoutMins,
			Config: StageConfigDTO{
				RequiresPutWall: stage.Config.RequiresPutWall,
				PutWallZone:     stage.Config.PutWallZone,
				StationID:       stage.Config.StationID,
			},
		}
	}

	plan := &ExecutionPlanDTO{
		TemplateID: template.TemplateID,
		PathType:   string(template.PathType),
		Stages:     stages,
	}

	// Add special handling and process path info if available
	if processPath != nil {
		plan.SpecialHandling = processPath.SpecialHandling
		plan.TargetStation = processPath.TargetStationID
		plan.ProcessPathID = processPath.PathID
	}

	s.logger.Info("Execution plan resolved",
		"orderId", cmd.OrderID,
		"templateId", template.TemplateID,
		"pathType", template.PathType,
		"stageCount", len(stages),
	)

	return plan, nil
}

// CreateTaskRoute creates a new task route for an order
func (s *WESApplicationService) CreateTaskRoute(ctx context.Context, cmd CreateTaskRouteCommand) (*TaskRouteDTO, error) {
	s.logger.Info("Creating task route", "orderId", cmd.OrderID, "waveId", cmd.WaveID, "templateId", cmd.TemplateID)

	// Get the template
	template, err := s.templateRepo.FindByTemplateID(ctx, cmd.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to find template: %w", err)
	}
	if template == nil {
		return nil, fmt.Errorf("template not found: %s", cmd.TemplateID)
	}

	// Create the task route
	route := domain.NewTaskRoute(cmd.OrderID, cmd.WaveID, template, cmd.SpecialHandling, cmd.ProcessPathID)

	// Save to repository
	if err := s.routeRepo.Save(ctx, route); err != nil {
		return nil, fmt.Errorf("failed to save task route: %w", err)
	}

	s.logger.Info("Task route created",
		"routeId", route.RouteID,
		"orderId", cmd.OrderID,
		"pathType", template.PathType,
	)

	return MapRouteToDTO(route), nil
}

// AssignWorkerToStage assigns a worker to the current stage of a route
func (s *WESApplicationService) AssignWorkerToStage(ctx context.Context, cmd AssignWorkerCommand) (*TaskRouteDTO, error) {
	s.logger.Info("Assigning worker to stage", "routeId", cmd.RouteID, "workerId", cmd.WorkerID)

	route, err := s.routeRepo.FindByRouteID(ctx, cmd.RouteID)
	if err != nil {
		return nil, fmt.Errorf("failed to find route: %w", err)
	}
	if route == nil {
		return nil, fmt.Errorf("route not found: %s", cmd.RouteID)
	}

	if err := route.AssignWorkerToCurrentStage(cmd.WorkerID, cmd.TaskID); err != nil {
		return nil, fmt.Errorf("failed to assign worker: %w", err)
	}

	if err := s.routeRepo.Update(ctx, route); err != nil {
		return nil, fmt.Errorf("failed to update route: %w", err)
	}

	s.logger.Info("Worker assigned to stage",
		"routeId", cmd.RouteID,
		"workerId", cmd.WorkerID,
		"stageType", route.GetCurrentStage().StageType,
	)

	return MapRouteToDTO(route), nil
}

// StartStage starts the current stage of a route
func (s *WESApplicationService) StartStage(ctx context.Context, cmd StartStageCommand) (*TaskRouteDTO, error) {
	s.logger.Info("Starting stage", "routeId", cmd.RouteID)

	route, err := s.routeRepo.FindByRouteID(ctx, cmd.RouteID)
	if err != nil {
		return nil, fmt.Errorf("failed to find route: %w", err)
	}
	if route == nil {
		return nil, fmt.Errorf("route not found: %s", cmd.RouteID)
	}

	if err := route.StartCurrentStage(); err != nil {
		return nil, fmt.Errorf("failed to start stage: %w", err)
	}

	if err := s.routeRepo.Update(ctx, route); err != nil {
		return nil, fmt.Errorf("failed to update route: %w", err)
	}

	return MapRouteToDTO(route), nil
}

// CompleteStage completes the current stage of a route
func (s *WESApplicationService) CompleteStage(ctx context.Context, cmd CompleteStageCommand) (*TaskRouteDTO, error) {
	s.logger.Info("Completing stage", "routeId", cmd.RouteID)

	route, err := s.routeRepo.FindByRouteID(ctx, cmd.RouteID)
	if err != nil {
		return nil, fmt.Errorf("failed to find route: %w", err)
	}
	if route == nil {
		return nil, fmt.Errorf("route not found: %s", cmd.RouteID)
	}

	currentStage := route.GetCurrentStage()
	if err := route.CompleteCurrentStage(); err != nil {
		return nil, fmt.Errorf("failed to complete stage: %w", err)
	}

	if err := s.routeRepo.Update(ctx, route); err != nil {
		return nil, fmt.Errorf("failed to update route: %w", err)
	}

	s.logger.Info("Stage completed",
		"routeId", cmd.RouteID,
		"stageType", currentStage.StageType,
		"routeStatus", route.Status,
	)

	return MapRouteToDTO(route), nil
}

// FailStage marks the current stage as failed
func (s *WESApplicationService) FailStage(ctx context.Context, cmd FailStageCommand) (*TaskRouteDTO, error) {
	s.logger.Info("Failing stage", "routeId", cmd.RouteID, "error", cmd.Error)

	route, err := s.routeRepo.FindByRouteID(ctx, cmd.RouteID)
	if err != nil {
		return nil, fmt.Errorf("failed to find route: %w", err)
	}
	if route == nil {
		return nil, fmt.Errorf("route not found: %s", cmd.RouteID)
	}

	if err := route.FailCurrentStage(cmd.Error); err != nil {
		return nil, fmt.Errorf("failed to fail stage: %w", err)
	}

	if err := s.routeRepo.Update(ctx, route); err != nil {
		return nil, fmt.Errorf("failed to update route: %w", err)
	}

	return MapRouteToDTO(route), nil
}

// GetTaskRoute gets a task route by ID
func (s *WESApplicationService) GetTaskRoute(ctx context.Context, query GetTaskRouteQuery) (*TaskRouteDTO, error) {
	route, err := s.routeRepo.FindByRouteID(ctx, query.RouteID)
	if err != nil {
		return nil, fmt.Errorf("failed to find route: %w", err)
	}
	return MapRouteToDTO(route), nil
}

// GetTaskRouteByOrder gets a task route by order ID
func (s *WESApplicationService) GetTaskRouteByOrder(ctx context.Context, query GetRouteByOrderQuery) (*TaskRouteDTO, error) {
	route, err := s.routeRepo.FindByOrderID(ctx, query.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to find route: %w", err)
	}
	return MapRouteToDTO(route), nil
}

// GetTemplate gets a stage template by ID
func (s *WESApplicationService) GetTemplate(ctx context.Context, query GetTemplateQuery) (*StageTemplateDTO, error) {
	template, err := s.templateRepo.FindByTemplateID(ctx, query.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to find template: %w", err)
	}
	return MapTemplateToDTO(template), nil
}

// ListTemplates lists all templates
func (s *WESApplicationService) ListTemplates(ctx context.Context, query ListTemplatesQuery) ([]*StageTemplateDTO, error) {
	var templates []*domain.StageTemplate
	var err error

	if query.ActiveOnly {
		templates, err = s.templateRepo.FindActive(ctx)
	} else {
		templates, err = s.templateRepo.FindActive(ctx) // TODO: add FindAll method
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	dtos := make([]*StageTemplateDTO, len(templates))
	for i, t := range templates {
		dtos[i] = MapTemplateToDTO(t)
	}
	return dtos, nil
}

// ProcessPathClient is a client for the process-path-service
type ProcessPathClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewProcessPathClient creates a new process path client
func NewProcessPathClient(baseURL string) *ProcessPathClient {
	return &ProcessPathClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DetermineProcessPath calls process-path-service to determine the process path
func (c *ProcessPathClient) DetermineProcessPath(ctx context.Context, orderID string, items []ItemInfo) (*ProcessPathResultDTO, error) {
	// Build request body
	reqBody := map[string]interface{}{
		"orderId": orderID,
		"items":   items,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/process-paths/determine", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// For now, return nil to indicate we should use defaults
	// In production, this would make an actual HTTP call
	_ = req
	return nil, nil
}
