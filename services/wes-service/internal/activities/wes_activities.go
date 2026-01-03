package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wms-platform/wes-service/internal/application"
	"github.com/wms-platform/wes-service/internal/workflows"
	"go.temporal.io/sdk/activity"
)

// WESActivities contains the WES workflow activities
type WESActivities struct {
	service       *application.WESApplicationService
	laborClient   *LaborServiceClient
	pickingClient *PickingServiceClient
	wallingClient *WallingServiceClient
	packingClient *PackingServiceClient
}

// NewWESActivities creates a new WESActivities instance
func NewWESActivities(
	service *application.WESApplicationService,
	laborClient *LaborServiceClient,
	pickingClient *PickingServiceClient,
	wallingClient *WallingServiceClient,
	packingClient *PackingServiceClient,
) *WESActivities {
	return &WESActivities{
		service:       service,
		laborClient:   laborClient,
		pickingClient: pickingClient,
		wallingClient: wallingClient,
		packingClient: packingClient,
	}
}

// ResolveExecutionPlan resolves the execution plan for an order
func (a *WESActivities) ResolveExecutionPlan(ctx context.Context, input map[string]interface{}) (*workflows.ExecutionPlan, error) {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	multiZone, _ := input["multiZone"].(bool)

	// Convert items
	var items []application.ItemInfo
	if itemsRaw, ok := input["items"].([]interface{}); ok {
		for _, itemRaw := range itemsRaw {
			if itemMap, ok := itemRaw.(map[string]interface{}); ok {
				items = append(items, application.ItemInfo{
					SKU:        getString(itemMap, "sku"),
					Quantity:   getInt(itemMap, "quantity"),
					LocationID: getString(itemMap, "locationId"),
					Zone:       getString(itemMap, "zone"),
				})
			}
		}
	}

	logger.Info("Resolving execution plan", "orderId", orderID, "itemCount", len(items))

	cmd := application.ResolveExecutionPlanCommand{
		OrderID:   orderID,
		Items:     items,
		MultiZone: multiZone,
	}

	// Get special handling if provided
	if handling, ok := input["specialHandling"].([]interface{}); ok {
		for _, h := range handling {
			if s, ok := h.(string); ok {
				cmd.SpecialHandling = append(cmd.SpecialHandling, s)
			}
		}
	}

	planDTO, err := a.service.ResolveExecutionPlan(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve execution plan: %w", err)
	}

	// Convert DTO to workflow struct
	stages := make([]workflows.StageDefinition, len(planDTO.Stages))
	for i, s := range planDTO.Stages {
		stages[i] = workflows.StageDefinition{
			Order:       s.Order,
			StageType:   s.StageType,
			TaskType:    s.TaskType,
			Required:    s.Required,
			TimeoutMins: s.TimeoutMins,
			Config: workflows.StageConfig{
				RequiresPutWall: s.Config.RequiresPutWall,
				PutWallZone:     s.Config.PutWallZone,
				StationID:       s.Config.StationID,
			},
		}
	}

	return &workflows.ExecutionPlan{
		TemplateID:      planDTO.TemplateID,
		PathType:        planDTO.PathType,
		Stages:          stages,
		SpecialHandling: planDTO.SpecialHandling,
		ProcessPathID:   planDTO.ProcessPathID,
	}, nil
}

// CreateTaskRoute creates a task route for an order
func (a *WESActivities) CreateTaskRoute(ctx context.Context, input map[string]interface{}) (*workflows.TaskRoute, error) {
	logger := activity.GetLogger(ctx)

	orderID := getString(input, "orderId")
	waveID := getString(input, "waveId")
	templateID := getString(input, "templateId")
	processPathID := getString(input, "processPathId")

	var specialHandling []string
	if handling, ok := input["specialHandling"].([]interface{}); ok {
		for _, h := range handling {
			if s, ok := h.(string); ok {
				specialHandling = append(specialHandling, s)
			}
		}
	}

	logger.Info("Creating task route", "orderId", orderID, "waveId", waveID, "templateId", templateID)

	cmd := application.CreateTaskRouteCommand{
		OrderID:         orderID,
		WaveID:          waveID,
		TemplateID:      templateID,
		SpecialHandling: specialHandling,
		ProcessPathID:   processPathID,
	}

	routeDTO, err := a.service.CreateTaskRoute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to create task route: %w", err)
	}

	return &workflows.TaskRoute{
		RouteID:        routeDTO.RouteID,
		OrderID:        routeDTO.OrderID,
		WaveID:         routeDTO.WaveID,
		PathTemplateID: routeDTO.PathTemplateID,
		PathType:       routeDTO.PathType,
		Status:         routeDTO.Status,
	}, nil
}

// AssignWorkerToStage assigns a worker to a stage
func (a *WESActivities) AssignWorkerToStage(ctx context.Context, input map[string]interface{}) (*workflows.WorkerAssignment, error) {
	logger := activity.GetLogger(ctx)

	routeID := getString(input, "routeId")
	stageType := getString(input, "stageType")
	taskType := getString(input, "taskType")

	logger.Info("Assigning worker to stage", "routeId", routeID, "stageType", stageType)

	// Get available worker from labor service
	var workerID string
	if a.laborClient != nil {
		workers, err := a.laborClient.GetAvailableWorkers(ctx, taskType, "")
		if err != nil {
			logger.Warn("Failed to get available workers, using default", "error", err)
		} else if len(workers) > 0 {
			workerID = workers[0].WorkerID
		}
	}

	// Generate default worker ID if none available
	if workerID == "" {
		prefix := "WK"
		switch stageType {
		case "picking":
			prefix = "PK"
		case "walling":
			prefix = "WL"
		case "packing":
			prefix = "PC"
		}
		workerID = fmt.Sprintf("%s-%s", prefix, uuid.New().String()[:8])
	}

	// Generate task ID
	taskID := fmt.Sprintf("%s-%s-%s", strings.ToUpper(stageType[:2]), routeID, uuid.New().String()[:4])

	// Assign worker in WES service
	cmd := application.AssignWorkerCommand{
		RouteID:  routeID,
		WorkerID: workerID,
		TaskID:   taskID,
	}

	_, err := a.service.AssignWorkerToStage(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to assign worker: %w", err)
	}

	logger.Info("Worker assigned to stage", "routeId", routeID, "stageType", stageType, "workerId", workerID, "taskId", taskID)

	return &workflows.WorkerAssignment{
		WorkerID:  workerID,
		TaskID:    taskID,
		StageType: stageType,
	}, nil
}

// StartStage starts the current stage of a route
func (a *WESActivities) StartStage(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	routeID := getString(input, "routeId")

	logger.Info("Starting stage", "routeId", routeID)

	cmd := application.StartStageCommand{
		RouteID: routeID,
	}

	_, err := a.service.StartStage(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to start stage: %w", err)
	}

	return nil
}

// CompleteStage completes the current stage of a route
func (a *WESActivities) CompleteStage(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	routeID := getString(input, "routeId")

	logger.Info("Completing stage", "routeId", routeID)

	cmd := application.CompleteStageCommand{
		RouteID: routeID,
	}

	_, err := a.service.CompleteStage(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to complete stage: %w", err)
	}

	return nil
}

// FailStage marks the current stage as failed
func (a *WESActivities) FailStage(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	routeID := getString(input, "routeId")
	errorMsg := getString(input, "error")

	logger.Info("Failing stage", "routeId", routeID, "error", errorMsg)

	cmd := application.FailStageCommand{
		RouteID: routeID,
		Error:   errorMsg,
	}

	_, err := a.service.FailStage(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to fail stage: %w", err)
	}

	return nil
}

// ExecuteWallingTask creates and monitors a walling task
func (a *WESActivities) ExecuteWallingTask(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	orderID := getString(input, "orderId")
	waveID := getString(input, "waveId")
	routeID := getString(input, "routeId")
	taskID := getString(input, "taskId")
	wallinerID := getString(input, "wallinerId")
	putWallZone := getString(input, "putWallZone")

	logger.Info("Executing walling task", "orderId", orderID, "taskId", taskID, "wallinerId", wallinerID)

	if a.wallingClient != nil {
		// Create walling task via walling service
		err := a.wallingClient.CreateWallingTask(ctx, &CreateWallingTaskRequest{
			OrderID:        orderID,
			WaveID:         waveID,
			RouteID:        routeID,
			TaskID:         taskID,
			WallinerID:     wallinerID,
			PutWallID:      putWallZone,
			DestinationBin: fmt.Sprintf("BIN-%s", orderID[:8]),
		})
		if err != nil {
			return fmt.Errorf("failed to create walling task: %w", err)
		}
	}

	logger.Info("Walling task created", "taskId", taskID)
	return nil
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}

// Service clients

// LaborServiceClient is a client for the labor service
type LaborServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewLaborServiceClient creates a new LaborServiceClient
func NewLaborServiceClient(baseURL string) *LaborServiceClient {
	return &LaborServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Worker represents a worker from the labor service
type Worker struct {
	WorkerID   string `json:"workerId"`
	Name       string `json:"name"`
	TaskType   string `json:"taskType"`
	Zone       string `json:"zone"`
	Status     string `json:"status"`
}

// GetAvailableWorkers gets available workers for a task type
func (c *LaborServiceClient) GetAvailableWorkers(ctx context.Context, taskType, zone string) ([]Worker, error) {
	url := fmt.Sprintf("%s/api/v1/workers?taskType=%s&status=available", c.baseURL, taskType)
	if zone != "" {
		url += "&zone=" + zone
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("labor service returned status %d", resp.StatusCode)
	}

	var workers []Worker
	if err := json.NewDecoder(resp.Body).Decode(&workers); err != nil {
		return nil, err
	}

	return workers, nil
}

// PickingServiceClient is a client for the picking service
type PickingServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPickingServiceClient creates a new PickingServiceClient
func NewPickingServiceClient(baseURL string) *PickingServiceClient {
	return &PickingServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// WallingServiceClient is a client for the walling service
type WallingServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewWallingServiceClient creates a new WallingServiceClient
func NewWallingServiceClient(baseURL string) *WallingServiceClient {
	return &WallingServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreateWallingTaskRequest represents a request to create a walling task
type CreateWallingTaskRequest struct {
	OrderID        string `json:"orderId"`
	WaveID         string `json:"waveId"`
	RouteID        string `json:"routeId"`
	TaskID         string `json:"taskId"`
	WallinerID     string `json:"wallinerId"`
	PutWallID      string `json:"putWallId"`
	DestinationBin string `json:"destinationBin"`
}

// CreateWallingTask creates a walling task
func (c *WallingServiceClient) CreateWallingTask(ctx context.Context, req *CreateWallingTaskRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/tasks", nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Use body in actual implementation
	_ = body

	// For now, return nil (mock implementation)
	return nil
}

// PackingServiceClient is a client for the packing service
type PackingServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPackingServiceClient creates a new PackingServiceClient
func NewPackingServiceClient(baseURL string) *PackingServiceClient {
	return &PackingServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
