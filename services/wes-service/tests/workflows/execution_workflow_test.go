package workflows_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"

	"github.com/wms-platform/wes-service/internal/workflows"
)

// Mock workflows for child workflows
func MockPickingWorkflow(ctx workflow.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

func MockPackingWorkflow(ctx workflow.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

func MockConsolidationWorkflow(ctx workflow.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

// Mock activity functions
func ResolveExecutionPlan(input map[string]interface{}) (*workflows.ExecutionPlan, error) {
	return &workflows.ExecutionPlan{
		TemplateID: "tpl-pick-pack",
		PathType:   "pick_pack",
		Stages: []workflows.StageDefinition{
			{Order: 1, StageType: "picking", TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: "packing", TaskType: "packing", Required: true, TimeoutMins: 15},
		},
	}, nil
}

func CreateTaskRoute(input map[string]interface{}) (*workflows.TaskRoute, error) {
	return &workflows.TaskRoute{
		RouteID:        "ROUTE-001",
		OrderID:        "ORD-001",
		WaveID:         "WAVE-001",
		PathTemplateID: "tpl-pick-pack",
		PathType:       "pick_pack",
		Status:         "pending",
	}, nil
}

func AssignWorkerToStage(input map[string]interface{}) (*workflows.WorkerAssignment, error) {
	stageType, _ := input["stageType"].(string)
	return &workflows.WorkerAssignment{
		WorkerID:  "WORKER-001",
		TaskID:    "TASK-" + stageType,
		StageType: stageType,
	}, nil
}

func StartStage(input map[string]interface{}) error {
	return nil
}

func CompleteStage(input map[string]interface{}) error {
	return nil
}

func FailStage(input map[string]interface{}) error {
	return nil
}

func ExecuteWallingTask(input map[string]interface{}) error {
	return nil
}

// TestWESExecutionWorkflow_PickPackPath tests the pick-pack path (simple 2-stage flow)
func TestWESExecutionWorkflow_PickPackPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register workflows
	env.RegisterWorkflow(workflows.WESExecutionWorkflow)
	env.RegisterWorkflowWithOptions(MockPickingWorkflow, workflow.RegisterOptions{Name: "PickingWorkflow"})
	env.RegisterWorkflowWithOptions(MockPackingWorkflow, workflow.RegisterOptions{Name: "PackingWorkflow"})

	// Register activities
	env.RegisterActivity(ResolveExecutionPlan)
	env.RegisterActivity(CreateTaskRoute)
	env.RegisterActivity(AssignWorkerToStage)
	env.RegisterActivity(StartStage)
	env.RegisterActivity(CompleteStage)
	env.RegisterActivity(FailStage)

	// Mock ResolveExecutionPlan
	env.OnActivity(ResolveExecutionPlan, mock.Anything).Return(&workflows.ExecutionPlan{
		TemplateID: "tpl-pick-pack",
		PathType:   "pick_pack",
		Stages: []workflows.StageDefinition{
			{Order: 1, StageType: "picking", TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: "packing", TaskType: "packing", Required: true, TimeoutMins: 15},
		},
	}, nil)

	// Mock CreateTaskRoute
	env.OnActivity(CreateTaskRoute, mock.Anything).Return(&workflows.TaskRoute{
		RouteID:        "ROUTE-001",
		OrderID:        "ORD-001",
		WaveID:         "WAVE-001",
		PathTemplateID: "tpl-pick-pack",
		PathType:       "pick_pack",
		Status:         "pending",
	}, nil)

	// Mock worker assignment
	env.OnActivity(AssignWorkerToStage, mock.Anything).Return(func(input map[string]interface{}) (*workflows.WorkerAssignment, error) {
		stageType, _ := input["stageType"].(string)
		return &workflows.WorkerAssignment{
			WorkerID:  "WORKER-001",
			TaskID:    "TASK-" + stageType,
			StageType: stageType,
		}, nil
	})

	// Mock StartStage and CompleteStage
	env.OnActivity(StartStage, mock.Anything).Return(nil)
	env.OnActivity(CompleteStage, mock.Anything).Return(nil)

	// Mock child workflows
	env.OnWorkflow("PickingWorkflow", mock.Anything, mock.Anything).Return(map[string]interface{}{"success": true}, nil)
	env.OnWorkflow("PackingWorkflow", mock.Anything, mock.Anything).Return(map[string]interface{}{"success": true}, nil)

	// Prepare input
	input := workflows.WESExecutionInput{
		OrderID: "ORD-001",
		WaveID:  "WAVE-001",
		Items: []workflows.ItemInfo{
			{SKU: "SKU-001", Quantity: 2},
		},
		MultiZone: false,
	}

	// Execute workflow
	env.ExecuteWorkflow(workflows.WESExecutionWorkflow, input)

	// Verify workflow completed successfully
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify result
	var result workflows.WESExecutionResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.Equal(t, "ORD-001", result.OrderID)
	assert.Equal(t, "ROUTE-001", result.RouteID)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "pick_pack", result.PathType)
	assert.Equal(t, 2, result.TotalStages)
	assert.Equal(t, 2, result.StagesCompleted)
	assert.NotNil(t, result.PickResult)
	assert.NotNil(t, result.PackingResult)
	assert.True(t, result.PickResult.Success)
	assert.True(t, result.PackingResult.Success)
}

// TestWESExecutionWorkflow_PickWallPackPath tests the pick-wall-pack path (3-stage flow with walling)
func TestWESExecutionWorkflow_PickWallPackPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register workflows
	env.RegisterWorkflow(workflows.WESExecutionWorkflow)
	env.RegisterWorkflowWithOptions(MockPickingWorkflow, workflow.RegisterOptions{Name: "PickingWorkflow"})
	env.RegisterWorkflowWithOptions(MockPackingWorkflow, workflow.RegisterOptions{Name: "PackingWorkflow"})

	// Register activities
	env.RegisterActivity(ResolveExecutionPlan)
	env.RegisterActivity(CreateTaskRoute)
	env.RegisterActivity(AssignWorkerToStage)
	env.RegisterActivity(StartStage)
	env.RegisterActivity(CompleteStage)
	env.RegisterActivity(FailStage)
	env.RegisterActivity(ExecuteWallingTask)

	// Mock ResolveExecutionPlan - pick_wall_pack path
	env.OnActivity(ResolveExecutionPlan, mock.Anything).Return(&workflows.ExecutionPlan{
		TemplateID: "tpl-pick-wall-pack",
		PathType:   "pick_wall_pack",
		Stages: []workflows.StageDefinition{
			{Order: 1, StageType: "picking", TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: "walling", TaskType: "walling", Required: true, TimeoutMins: 10, Config: workflows.StageConfig{RequiresPutWall: true}},
			{Order: 3, StageType: "packing", TaskType: "packing", Required: true, TimeoutMins: 15},
		},
	}, nil)

	// Mock CreateTaskRoute
	env.OnActivity(CreateTaskRoute, mock.Anything).Return(&workflows.TaskRoute{
		RouteID:        "ROUTE-002",
		OrderID:        "ORD-002",
		WaveID:         "WAVE-002",
		PathTemplateID: "tpl-pick-wall-pack",
		PathType:       "pick_wall_pack",
		Status:         "pending",
	}, nil)

	// Mock worker assignment
	env.OnActivity(AssignWorkerToStage, mock.Anything).Return(func(input map[string]interface{}) (*workflows.WorkerAssignment, error) {
		stageType, _ := input["stageType"].(string)
		return &workflows.WorkerAssignment{
			WorkerID:  "WORKER-002",
			TaskID:    "TASK-" + stageType,
			StageType: stageType,
		}, nil
	})

	// Mock other activities
	env.OnActivity(StartStage, mock.Anything).Return(nil)
	env.OnActivity(CompleteStage, mock.Anything).Return(nil)
	env.OnActivity(ExecuteWallingTask, mock.Anything).Return(nil)

	// Mock child workflows
	env.OnWorkflow("PickingWorkflow", mock.Anything, mock.Anything).Return(map[string]interface{}{"success": true}, nil)
	env.OnWorkflow("PackingWorkflow", mock.Anything, mock.Anything).Return(map[string]interface{}{"success": true}, nil)

	// For walling stage, we need to send a signal to complete
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("wallingCompleted", map[string]interface{}{"success": true})
	}, 0)

	// Prepare input - multi-item order needing walling
	input := workflows.WESExecutionInput{
		OrderID: "ORD-002",
		WaveID:  "WAVE-002",
		Items: []workflows.ItemInfo{
			{SKU: "SKU-001", Quantity: 2},
			{SKU: "SKU-002", Quantity: 3},
			{SKU: "SKU-003", Quantity: 1},
		},
		MultiZone: false,
	}

	// Execute workflow
	env.ExecuteWorkflow(workflows.WESExecutionWorkflow, input)

	// Verify workflow completed successfully
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify result
	var result workflows.WESExecutionResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.Equal(t, "ORD-002", result.OrderID)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "pick_wall_pack", result.PathType)
	assert.Equal(t, 3, result.TotalStages)
	assert.Equal(t, 3, result.StagesCompleted)
	assert.NotNil(t, result.PickResult)
	assert.NotNil(t, result.WallingResult)
	assert.NotNil(t, result.PackingResult)
}

// TestWESExecutionWorkflow_ResolveExecutionPlanFailed tests failure when execution plan resolution fails
func TestWESExecutionWorkflow_ResolveExecutionPlanFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.WESExecutionWorkflow)
	env.RegisterActivity(ResolveExecutionPlan)

	// Mock ResolveExecutionPlan to fail
	env.OnActivity(ResolveExecutionPlan, mock.Anything).Return(nil, errors.New("template not found"))

	input := workflows.WESExecutionInput{
		OrderID: "ORD-FAIL-001",
		WaveID:  "WAVE-001",
		Items:   []workflows.ItemInfo{{SKU: "SKU-001", Quantity: 1}},
	}

	env.ExecuteWorkflow(workflows.WESExecutionWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)

	assert.Contains(t, workflowErr.Error(), "template not found")
}

// TestWESExecutionWorkflow_CreateTaskRouteFailed tests failure when task route creation fails
func TestWESExecutionWorkflow_CreateTaskRouteFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.WESExecutionWorkflow)
	env.RegisterActivity(ResolveExecutionPlan)
	env.RegisterActivity(CreateTaskRoute)

	// Mock successful execution plan
	env.OnActivity(ResolveExecutionPlan, mock.Anything).Return(&workflows.ExecutionPlan{
		TemplateID: "tpl-pick-pack",
		PathType:   "pick_pack",
		Stages:     []workflows.StageDefinition{{Order: 1, StageType: "picking", Required: true}},
	}, nil)

	// Mock CreateTaskRoute to fail
	env.OnActivity(CreateTaskRoute, mock.Anything).Return(nil, errors.New("database error"))

	input := workflows.WESExecutionInput{
		OrderID: "ORD-FAIL-002",
		WaveID:  "WAVE-001",
		Items:   []workflows.ItemInfo{{SKU: "SKU-001", Quantity: 1}},
	}

	env.ExecuteWorkflow(workflows.WESExecutionWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)

	assert.Contains(t, workflowErr.Error(), "database error")
}

// TestWESExecutionWorkflow_PickingFailed tests failure during picking stage
func TestWESExecutionWorkflow_PickingFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.WESExecutionWorkflow)
	env.RegisterWorkflowWithOptions(MockPickingWorkflow, workflow.RegisterOptions{Name: "PickingWorkflow"})
	env.RegisterActivity(ResolveExecutionPlan)
	env.RegisterActivity(CreateTaskRoute)
	env.RegisterActivity(AssignWorkerToStage)
	env.RegisterActivity(StartStage)
	env.RegisterActivity(CompleteStage)
	env.RegisterActivity(FailStage)

	// Mock successful setup
	env.OnActivity(ResolveExecutionPlan, mock.Anything).Return(&workflows.ExecutionPlan{
		TemplateID: "tpl-pick-pack",
		PathType:   "pick_pack",
		Stages: []workflows.StageDefinition{
			{Order: 1, StageType: "picking", TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: "packing", TaskType: "packing", Required: true, TimeoutMins: 15},
		},
	}, nil)

	env.OnActivity(CreateTaskRoute, mock.Anything).Return(&workflows.TaskRoute{
		RouteID:  "ROUTE-FAIL",
		OrderID:  "ORD-FAIL-003",
		WaveID:   "WAVE-001",
		PathType: "pick_pack",
		Status:   "pending",
	}, nil)

	env.OnActivity(AssignWorkerToStage, mock.Anything).Return(&workflows.WorkerAssignment{
		WorkerID:  "WORKER-001",
		TaskID:    "TASK-picking",
		StageType: "picking",
	}, nil)

	env.OnActivity(StartStage, mock.Anything).Return(nil)
	env.OnActivity(FailStage, mock.Anything).Return(nil)

	// Mock picking workflow to fail
	env.OnWorkflow("PickingWorkflow", mock.Anything, mock.Anything).Return(nil, errors.New("no items at location"))

	input := workflows.WESExecutionInput{
		OrderID: "ORD-FAIL-003",
		WaveID:  "WAVE-001",
		Items:   []workflows.ItemInfo{{SKU: "SKU-001", Quantity: 1}},
	}

	env.ExecuteWorkflow(workflows.WESExecutionWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)

	assert.Contains(t, workflowErr.Error(), "no items at location")
}

// TestWESExecutionWorkflow_MultiZoneOrder tests multi-zone order using pick_consolidate_pack path
func TestWESExecutionWorkflow_MultiZoneOrder(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.WESExecutionWorkflow)
	env.RegisterWorkflowWithOptions(MockPickingWorkflow, workflow.RegisterOptions{Name: "PickingWorkflow"})
	env.RegisterWorkflowWithOptions(MockConsolidationWorkflow, workflow.RegisterOptions{Name: "ConsolidationWorkflow"})
	env.RegisterWorkflowWithOptions(MockPackingWorkflow, workflow.RegisterOptions{Name: "PackingWorkflow"})

	env.RegisterActivity(ResolveExecutionPlan)
	env.RegisterActivity(CreateTaskRoute)
	env.RegisterActivity(AssignWorkerToStage)
	env.RegisterActivity(StartStage)
	env.RegisterActivity(CompleteStage)
	env.RegisterActivity(FailStage)

	// Mock ResolveExecutionPlan - pick_consolidate_pack path
	env.OnActivity(ResolveExecutionPlan, mock.Anything).Return(&workflows.ExecutionPlan{
		TemplateID: "tpl-pick-consolidate-pack",
		PathType:   "pick_consolidate_pack",
		Stages: []workflows.StageDefinition{
			{Order: 1, StageType: "picking", TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: "consolidation", TaskType: "consolidation", Required: true, TimeoutMins: 20},
			{Order: 3, StageType: "packing", TaskType: "packing", Required: true, TimeoutMins: 15},
		},
	}, nil)

	env.OnActivity(CreateTaskRoute, mock.Anything).Return(&workflows.TaskRoute{
		RouteID:  "ROUTE-MZ",
		OrderID:  "ORD-MZ-001",
		WaveID:   "WAVE-MZ",
		PathType: "pick_consolidate_pack",
		Status:   "pending",
	}, nil)

	env.OnActivity(AssignWorkerToStage, mock.Anything).Return(func(input map[string]interface{}) (*workflows.WorkerAssignment, error) {
		stageType, _ := input["stageType"].(string)
		return &workflows.WorkerAssignment{
			WorkerID:  "WORKER-MZ",
			TaskID:    "TASK-" + stageType,
			StageType: stageType,
		}, nil
	})

	env.OnActivity(StartStage, mock.Anything).Return(nil)
	env.OnActivity(CompleteStage, mock.Anything).Return(nil)

	env.OnWorkflow("PickingWorkflow", mock.Anything, mock.Anything).Return(map[string]interface{}{"success": true}, nil)
	env.OnWorkflow("ConsolidationWorkflow", mock.Anything, mock.Anything).Return(map[string]interface{}{"success": true}, nil)
	env.OnWorkflow("PackingWorkflow", mock.Anything, mock.Anything).Return(map[string]interface{}{"success": true}, nil)

	input := workflows.WESExecutionInput{
		OrderID: "ORD-MZ-001",
		WaveID:  "WAVE-MZ",
		Items: []workflows.ItemInfo{
			{SKU: "SKU-ZONE-A", Quantity: 2, Zone: "ZONE-A"},
			{SKU: "SKU-ZONE-B", Quantity: 1, Zone: "ZONE-B"},
		},
		MultiZone: true,
	}

	env.ExecuteWorkflow(workflows.WESExecutionWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.WESExecutionResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "pick_consolidate_pack", result.PathType)
	assert.Equal(t, 3, result.TotalStages)
	assert.Equal(t, 3, result.StagesCompleted)
}

// TestWESExecutionWorkflow_WithSpecialHandling tests order with special handling requirements
func TestWESExecutionWorkflow_WithSpecialHandling(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.WESExecutionWorkflow)
	env.RegisterWorkflowWithOptions(MockPickingWorkflow, workflow.RegisterOptions{Name: "PickingWorkflow"})
	env.RegisterWorkflowWithOptions(MockPackingWorkflow, workflow.RegisterOptions{Name: "PackingWorkflow"})

	env.RegisterActivity(ResolveExecutionPlan)
	env.RegisterActivity(CreateTaskRoute)
	env.RegisterActivity(AssignWorkerToStage)
	env.RegisterActivity(StartStage)
	env.RegisterActivity(CompleteStage)
	env.RegisterActivity(FailStage)

	// Mock ResolveExecutionPlan with special handling
	env.OnActivity(ResolveExecutionPlan, mock.Anything).Return(&workflows.ExecutionPlan{
		TemplateID:      "tpl-pick-pack",
		PathType:        "pick_pack",
		SpecialHandling: []string{"fragile", "hazmat"},
		Stages: []workflows.StageDefinition{
			{Order: 1, StageType: "picking", TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: "packing", TaskType: "packing", Required: true, TimeoutMins: 15},
		},
	}, nil)

	env.OnActivity(CreateTaskRoute, mock.Anything).Return(&workflows.TaskRoute{
		RouteID:  "ROUTE-SPECIAL",
		OrderID:  "ORD-SPECIAL",
		WaveID:   "WAVE-SPECIAL",
		PathType: "pick_pack",
		Status:   "pending",
	}, nil)

	env.OnActivity(AssignWorkerToStage, mock.Anything).Return(func(input map[string]interface{}) (*workflows.WorkerAssignment, error) {
		stageType, _ := input["stageType"].(string)
		return &workflows.WorkerAssignment{
			WorkerID:  "WORKER-CERTIFIED",
			TaskID:    "TASK-" + stageType,
			StageType: stageType,
		}, nil
	})

	env.OnActivity(StartStage, mock.Anything).Return(nil)
	env.OnActivity(CompleteStage, mock.Anything).Return(nil)

	env.OnWorkflow("PickingWorkflow", mock.Anything, mock.Anything).Return(map[string]interface{}{"success": true}, nil)
	env.OnWorkflow("PackingWorkflow", mock.Anything, mock.Anything).Return(map[string]interface{}{"success": true}, nil)

	input := workflows.WESExecutionInput{
		OrderID: "ORD-SPECIAL",
		WaveID:  "WAVE-SPECIAL",
		Items: []workflows.ItemInfo{
			{SKU: "SKU-HAZMAT", Quantity: 1},
		},
		SpecialHandling: []string{"fragile", "hazmat"},
	}

	env.ExecuteWorkflow(workflows.WESExecutionWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.WESExecutionResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.Equal(t, "completed", result.Status)
}
