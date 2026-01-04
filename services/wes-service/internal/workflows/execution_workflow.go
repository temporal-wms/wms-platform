package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Task queue constants for cross-queue child workflow execution
const (
	PickingTaskQueue       = "picking-queue"
	ConsolidationTaskQueue = "consolidation-queue"
	PackingTaskQueue       = "packing-queue"
)

// Workflow name constants
const (
	PickingWorkflowName       = "OrchestratedPickingWorkflow"
	ConsolidationWorkflowName = "ConsolidationWorkflow"
	PackingWorkflowName       = "PackingWorkflow"
)

// WESExecutionInput represents the input for the WES execution workflow
type WESExecutionInput struct {
	OrderID         string          `json:"orderId"`
	WaveID          string          `json:"waveId"`
	Items           []ItemInfo      `json:"items"`
	MultiZone       bool            `json:"multiZone"`
	ProcessPathID   string          `json:"processPathId,omitempty"`
	SpecialHandling []string        `json:"specialHandling,omitempty"`
}

// ItemInfo represents an item in the order
type ItemInfo struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId,omitempty"`
	Zone       string `json:"zone,omitempty"`
}

// WESExecutionResult represents the result of the WES execution workflow
type WESExecutionResult struct {
	RouteID         string       `json:"routeId"`
	OrderID         string       `json:"orderId"`
	Status          string       `json:"status"`
	PathType        string       `json:"pathType"`
	StagesCompleted int          `json:"stagesCompleted"`
	TotalStages     int          `json:"totalStages"`
	PickResult      *StageResult `json:"pickResult,omitempty"`
	WallingResult   *StageResult `json:"wallingResult,omitempty"`
	PackingResult   *StageResult `json:"packingResult,omitempty"`
	CompletedAt     int64        `json:"completedAt,omitempty"`
	Error           string       `json:"error,omitempty"`
}

// StageResult represents the result of a stage execution
type StageResult struct {
	StageType   string `json:"stageType"`
	TaskID      string `json:"taskId"`
	WorkerID    string `json:"workerId"`
	Success     bool   `json:"success"`
	CompletedAt int64  `json:"completedAt,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ExecutionPlan represents the resolved execution plan
type ExecutionPlan struct {
	TemplateID      string            `json:"templateId"`
	PathType        string            `json:"pathType"`
	Stages          []StageDefinition `json:"stages"`
	SpecialHandling []string          `json:"specialHandling"`
	ProcessPathID   string            `json:"processPathId"`
}

// StageDefinition represents a stage in the execution plan
type StageDefinition struct {
	Order       int         `json:"order"`
	StageType   string      `json:"stageType"`
	TaskType    string      `json:"taskType"`
	Required    bool        `json:"required"`
	TimeoutMins int         `json:"timeoutMins"`
	Config      StageConfig `json:"config,omitempty"`
}

// StageConfig represents stage-specific configuration
type StageConfig struct {
	RequiresPutWall bool   `json:"requiresPutWall,omitempty"`
	PutWallZone     string `json:"putWallZone,omitempty"`
	StationID       string `json:"stationId,omitempty"`
}

// TaskRoute represents a task route for tracking execution
type TaskRoute struct {
	RouteID        string `json:"routeId"`
	OrderID        string `json:"orderId"`
	WaveID         string `json:"waveId"`
	PathTemplateID string `json:"pathTemplateId"`
	PathType       string `json:"pathType"`
	Status         string `json:"status"`
}

// WorkerAssignment represents a worker assignment
type WorkerAssignment struct {
	WorkerID  string `json:"workerId"`
	TaskID    string `json:"taskId"`
	StageType string `json:"stageType"`
}

// WESExecutionWorkflow is the main workflow that orchestrates order execution through stages
func WESExecutionWorkflow(ctx workflow.Context, input WESExecutionInput) (*WESExecutionResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting WES execution workflow", "orderId", input.OrderID, "waveId", input.WaveID)

	result := &WESExecutionResult{
		OrderID: input.OrderID,
		Status:  "in_progress",
	}

	// Activity options for WES activities
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Resolve execution plan
	logger.Info("Resolving execution plan", "orderId", input.OrderID)
	var executionPlan ExecutionPlan
	err := workflow.ExecuteActivity(ctx, "ResolveExecutionPlan", map[string]interface{}{
		"orderId":         input.OrderID,
		"items":           input.Items,
		"multiZone":       input.MultiZone,
		"processPathId":   input.ProcessPathID,
		"specialHandling": input.SpecialHandling,
	}).Get(ctx, &executionPlan)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to resolve execution plan: %v", err)
		return result, err
	}

	result.PathType = executionPlan.PathType
	result.TotalStages = len(executionPlan.Stages)
	logger.Info("Execution plan resolved",
		"orderId", input.OrderID,
		"pathType", executionPlan.PathType,
		"stageCount", len(executionPlan.Stages),
	)

	// Step 2: Create task route
	logger.Info("Creating task route", "orderId", input.OrderID)
	var taskRoute TaskRoute
	err = workflow.ExecuteActivity(ctx, "CreateTaskRoute", map[string]interface{}{
		"orderId":         input.OrderID,
		"waveId":          input.WaveID,
		"templateId":      executionPlan.TemplateID,
		"specialHandling": executionPlan.SpecialHandling,
		"processPathId":   executionPlan.ProcessPathID,
	}).Get(ctx, &taskRoute)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to create task route: %v", err)
		return result, err
	}

	result.RouteID = taskRoute.RouteID
	logger.Info("Task route created", "routeId", taskRoute.RouteID, "orderId", input.OrderID)

	// Step 3: Execute each stage in sequence
	for i, stage := range executionPlan.Stages {
		logger.Info("Executing stage",
			"orderId", input.OrderID,
			"routeId", taskRoute.RouteID,
			"stageIndex", i,
			"stageType", stage.StageType,
		)

		stageResult, err := executeStage(ctx, taskRoute.RouteID, input, stage)
		if err != nil {
			// Mark stage as failed
			workflow.ExecuteActivity(ctx, "FailStage", map[string]interface{}{
				"routeId": taskRoute.RouteID,
				"error":   err.Error(),
			}).Get(ctx, nil)

			result.Status = "failed"
			result.Error = fmt.Sprintf("stage %s failed: %v", stage.StageType, err)
			result.StagesCompleted = i
			return result, err
		}

		// Store stage result
		switch stage.StageType {
		case "picking":
			result.PickResult = stageResult
		case "walling":
			result.WallingResult = stageResult
		case "packing":
			result.PackingResult = stageResult
		}

		// Mark stage as completed
		err = workflow.ExecuteActivity(ctx, "CompleteStage", map[string]interface{}{
			"routeId": taskRoute.RouteID,
		}).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to mark stage as completed, continuing", "error", err)
		}

		result.StagesCompleted = i + 1
		logger.Info("Stage completed",
			"orderId", input.OrderID,
			"stageType", stage.StageType,
			"stagesCompleted", result.StagesCompleted,
		)
	}

	result.Status = "completed"
	result.CompletedAt = workflow.Now(ctx).UnixMilli()

	logger.Info("WES execution completed successfully",
		"orderId", input.OrderID,
		"routeId", taskRoute.RouteID,
		"pathType", result.PathType,
	)

	return result, nil
}

// executeStage executes a single stage in the workflow
func executeStage(ctx workflow.Context, routeID string, input WESExecutionInput, stage StageDefinition) (*StageResult, error) {
	logger := workflow.GetLogger(ctx)

	// Stage-specific activity options with timeout from stage definition
	stageTimeout := time.Duration(stage.TimeoutMins) * time.Minute
	if stageTimeout == 0 {
		stageTimeout = 30 * time.Minute // default
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: stageTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	stageCtx := workflow.WithActivityOptions(ctx, ao)

	stageResult := &StageResult{
		StageType: stage.StageType,
	}

	// Assign worker to stage
	logger.Info("Assigning worker to stage", "stageType", stage.StageType, "routeId", routeID)
	var assignment WorkerAssignment
	err := workflow.ExecuteActivity(stageCtx, "AssignWorkerToStage", map[string]interface{}{
		"routeId":   routeID,
		"stageType": stage.StageType,
		"taskType":  stage.TaskType,
	}).Get(ctx, &assignment)
	if err != nil {
		stageResult.Success = false
		stageResult.Error = fmt.Sprintf("failed to assign worker: %v", err)
		return stageResult, err
	}

	stageResult.WorkerID = assignment.WorkerID
	stageResult.TaskID = assignment.TaskID

	// Start stage
	err = workflow.ExecuteActivity(stageCtx, "StartStage", map[string]interface{}{
		"routeId": routeID,
	}).Get(ctx, nil)
	if err != nil {
		stageResult.Success = false
		stageResult.Error = fmt.Sprintf("failed to start stage: %v", err)
		return stageResult, err
	}

	// Execute stage-specific child workflow
	switch stage.StageType {
	case "picking":
		err = executePickingStage(stageCtx, input, routeID, assignment, stageResult)
	case "walling":
		err = executeWallingStage(stageCtx, input, routeID, assignment, stage.Config, stageResult)
	case "consolidation":
		err = executeConsolidationStage(stageCtx, input, routeID, assignment, stageResult)
	case "packing":
		err = executePackingStage(stageCtx, input, routeID, assignment, stageResult)
	default:
		err = fmt.Errorf("unknown stage type: %s", stage.StageType)
	}

	if err != nil {
		stageResult.Success = false
		stageResult.Error = err.Error()
		return stageResult, err
	}

	stageResult.Success = true
	stageResult.CompletedAt = workflow.Now(ctx).UnixMilli()

	return stageResult, nil
}

// executePickingStage executes the picking stage via cross-queue child workflow
func executePickingStage(ctx workflow.Context, input WESExecutionInput, routeID string, assignment WorkerAssignment, result *StageResult) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing picking stage", "orderId", input.OrderID, "taskId", assignment.TaskID)

	// Build picking input
	pickingInput := map[string]interface{}{
		"orderId":  input.OrderID,
		"waveId":   input.WaveID,
		"routeId":  routeID,
		"taskId":   assignment.TaskID,
		"pickerId": assignment.WorkerID,
		"items":    input.Items,
	}

	// Configure child workflow options to route to picking-service worker
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		TaskQueue: PickingTaskQueue,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	childCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)

	// Execute PickingWorkflow as a child workflow on the picking-queue
	var pickingResult map[string]interface{}
	err := workflow.ExecuteChildWorkflow(childCtx, PickingWorkflowName, pickingInput).Get(ctx, &pickingResult)
	if err != nil {
		return fmt.Errorf("picking workflow failed: %w", err)
	}

	logger.Info("Picking stage completed", "orderId", input.OrderID, "taskId", assignment.TaskID)
	return nil
}

// executeWallingStage executes the walling (put-wall) stage
func executeWallingStage(ctx workflow.Context, input WESExecutionInput, routeID string, assignment WorkerAssignment, config StageConfig, result *StageResult) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing walling stage", "orderId", input.OrderID, "taskId", assignment.TaskID)

	// Build walling input
	wallingInput := map[string]interface{}{
		"orderId":   input.OrderID,
		"waveId":    input.WaveID,
		"routeId":   routeID,
		"taskId":    assignment.TaskID,
		"wallinerId": assignment.WorkerID,
		"putWallZone": config.PutWallZone,
	}

	// Execute walling activity (or child workflow if complex)
	err := workflow.ExecuteActivity(ctx, "ExecuteWallingTask", wallingInput).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("walling task failed: %w", err)
	}

	// Wait for walling completion signal
	wallingCompletedSignal := workflow.GetSignalChannel(ctx, "wallingCompleted")
	wallingTimeout := 15 * time.Minute

	selector := workflow.NewSelector(ctx)
	var wallingCompleted bool

	selector.AddReceive(wallingCompletedSignal, func(c workflow.ReceiveChannel, more bool) {
		var completion map[string]interface{}
		c.Receive(ctx, &completion)
		wallingCompleted = true
	})

	selector.AddFuture(workflow.NewTimer(ctx, wallingTimeout), func(f workflow.Future) {
		logger.Warn("Walling timeout", "orderId", input.OrderID)
	})

	selector.Select(ctx)

	if !wallingCompleted {
		return fmt.Errorf("walling timeout for order %s", input.OrderID)
	}

	logger.Info("Walling stage completed", "orderId", input.OrderID, "taskId", assignment.TaskID)
	return nil
}

// executeConsolidationStage executes the consolidation stage via cross-queue child workflow
func executeConsolidationStage(ctx workflow.Context, input WESExecutionInput, routeID string, assignment WorkerAssignment, result *StageResult) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing consolidation stage", "orderId", input.OrderID, "taskId", assignment.TaskID)

	// Build consolidation input
	consolidationInput := map[string]interface{}{
		"orderId": input.OrderID,
		"waveId":  input.WaveID,
		"routeId": routeID,
		"taskId":  assignment.TaskID,
		"items":   input.Items,
	}

	// Configure child workflow options to route to consolidation-service worker
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		TaskQueue: ConsolidationTaskQueue,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	childCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)

	// Execute ConsolidationWorkflow as a child workflow on the consolidation-queue
	var consolidationResult map[string]interface{}
	err := workflow.ExecuteChildWorkflow(childCtx, ConsolidationWorkflowName, consolidationInput).Get(ctx, &consolidationResult)
	if err != nil {
		return fmt.Errorf("consolidation workflow failed: %w", err)
	}

	logger.Info("Consolidation stage completed", "orderId", input.OrderID, "taskId", assignment.TaskID)
	return nil
}

// executePackingStage executes the packing stage via cross-queue child workflow
func executePackingStage(ctx workflow.Context, input WESExecutionInput, routeID string, assignment WorkerAssignment, result *StageResult) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing packing stage", "orderId", input.OrderID, "taskId", assignment.TaskID)

	// Build packing input
	packingInput := map[string]interface{}{
		"orderId":    input.OrderID,
		"waveId":     input.WaveID,
		"routeId":    routeID,
		"taskId":     assignment.TaskID,
		"packerId":   assignment.WorkerID,
		"sourceType": "tote",
		"items":      input.Items,
	}

	// Configure child workflow options to route to packing-service worker
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		TaskQueue: PackingTaskQueue,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	childCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)

	// Execute PackingWorkflow as a child workflow on the packing-queue
	var packingResult map[string]interface{}
	err := workflow.ExecuteChildWorkflow(childCtx, PackingWorkflowName, packingInput).Get(ctx, &packingResult)
	if err != nil {
		return fmt.Errorf("packing workflow failed: %w", err)
	}

	logger.Info("Packing stage completed", "orderId", input.OrderID, "taskId", assignment.TaskID)
	return nil
}
