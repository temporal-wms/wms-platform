package workflows_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"

	"github.com/wms-platform/orchestrator/internal/workflows"
)

// MockWESExecutionWorkflow is a mock workflow for WES execution
// This is needed to register the workflow type in the test environment
func MockWESExecutionWorkflow(ctx workflow.Context, input workflows.WESExecutionInput) (workflows.WESExecutionResult, error) {
	return workflows.WESExecutionResult{}, nil
}

// TestOrderFulfillmentWorkflow_Success tests the happy path of order fulfillment with WES
func TestOrderFulfillmentWorkflow_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register workflows
	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflow(workflows.SortationWorkflow)
	env.RegisterWorkflowWithOptions(MockWESExecutionWorkflow, workflow.RegisterOptions{Name: "WESExecutionWorkflow"})
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(ExecuteSLAM)

	// Mock activity: ValidateOrder
	env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

	// Mock child workflow: PlanningWorkflow
	planningResult := &workflows.PlanningWorkflowResult{
		ProcessPath: workflows.ProcessPathResult{
			PathID:                "PATH-001",
			Requirements:          []string{"multi_item"},
			ConsolidationRequired: true,
			GiftWrapRequired:      false,
		},
		WaveID:             "WAVE-001",
		WaveScheduledStart: time.Now().Add(15 * time.Minute),
		Success:            true,
	}
	env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(planningResult, nil)

	// Mock child workflow: WESExecutionWorkflow (replaces picking, walling, packing)
	wesResult := workflows.WESExecutionResult{
		RouteID:         "ROUTE-001",
		OrderID:         "ORD-001",
		Status:          "completed",
		PathType:        "pick_wall_pack",
		StagesCompleted: 3,
		TotalStages:     3,
		PickResult: &workflows.WESStageResult{
			StageType:   "picking",
			TaskID:      "PICK-001",
			WorkerID:    "WORKER-001",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		WallingResult: &workflows.WESStageResult{
			StageType:   "walling",
			TaskID:      "WALL-001",
			WorkerID:    "WORKER-002",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		PackingResult: &workflows.WESStageResult{
			StageType:   "packing",
			TaskID:      "PACK-001",
			WorkerID:    "WORKER-003",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		CompletedAt: time.Now().Unix(),
	}
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(wesResult, nil)

	// Mock SLAM activity
	env.OnActivity(ExecuteSLAM, mock.Anything).Return(workflows.SLAMResult{
		TrackingNumber: "TRACK-123456",
		ManifestID:     "MANIFEST-001",
		Success:        true,
		CarrierID:      "CARRIER-001",
		Destination:    "12345",
	}, nil)

	// Mock child workflow: SortationWorkflow
	sortationResult := &workflows.SortationWorkflowResult{
		BatchID: "BATCH-001",
		ChuteID: "CHUTE-001",
		Zone:    "ZONE-A",
		Success: true,
	}
	env.OnWorkflow("SortationWorkflow", mock.Anything, mock.Anything).Return(sortationResult, nil)

	// Mock child workflow: ShippingWorkflow
	env.OnWorkflow("ShippingWorkflow", mock.Anything, mock.Anything).Return(nil)

	// Prepare input
	input := workflows.OrderFulfillmentInput{
		OrderID:    "ORD-001",
		CustomerID: "CUST-001",
		Items: []workflows.Item{
			{SKU: "SKU-001", Quantity: 2, Weight: 2.5},
			{SKU: "SKU-002", Quantity: 1, Weight: 3.0},
		},
		Priority:           "same_day",
		PromisedDeliveryAt: time.Now().Add(24 * time.Hour),
		IsMultiItem:        true,
	}

	// Execute workflow
	env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)

	// Verify workflow completed successfully
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify result
	var result workflows.OrderFulfillmentResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.Equal(t, "ORD-001", result.OrderID)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "WAVE-001", result.WaveID)
	assert.Equal(t, "TRACK-123456", result.TrackingNumber)
	assert.Empty(t, result.Error)
}

// TestOrderFulfillmentWorkflow_ValidationFailed tests order validation failure
func TestOrderFulfillmentWorkflow_ValidationFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflowWithOptions(MockWESExecutionWorkflow, workflow.RegisterOptions{Name: "WESExecutionWorkflow"})
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(ReleaseInventoryReservation)

	// Mock activity: ValidateOrder returns error
	env.OnActivity(ValidateOrder, mock.Anything).Return(false, errors.New("insufficient inventory"))

	input := workflows.OrderFulfillmentInput{
		OrderID:            "ORD-002",
		CustomerID:         "CUST-002",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 100, Weight: 2.5}},
		Priority:           "standard",
		PromisedDeliveryAt: time.Now().Add(72 * time.Hour),
		IsMultiItem:        false,
	}

	env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)

	// When workflow returns an error, check the error message directly
	assert.Contains(t, workflowErr.Error(), "insufficient inventory")
}

// TestOrderFulfillmentWorkflow_PlanningFailed tests planning workflow failure (wave timeout)
func TestOrderFulfillmentWorkflow_PlanningFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflowWithOptions(MockWESExecutionWorkflow, workflow.RegisterOptions{Name: "WESExecutionWorkflow"})
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(ReleaseInventoryReservation)

	// Mock activity: ValidateOrder succeeds
	env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

	// Mock child workflow: PlanningWorkflow fails with wave timeout
	env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(
		&workflows.PlanningWorkflowResult{},
		errors.New("wave assignment timeout for order ORD-003"),
	)

	input := workflows.OrderFulfillmentInput{
		OrderID:            "ORD-003",
		CustomerID:         "CUST-003",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 2, Weight: 2.5}},
		Priority:           "same_day",
		PromisedDeliveryAt: time.Now().Add(24 * time.Hour),
		IsMultiItem:        false,
	}

	env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)

	// When workflow returns an error, check the error message directly
	assert.Contains(t, workflowErr.Error(), "wave assignment timeout")
	assert.Contains(t, workflowErr.Error(), "ORD-003")
}

// TestOrderFulfillmentWorkflow_WESExecutionFailed tests WES execution workflow failure with compensation
func TestOrderFulfillmentWorkflow_WESExecutionFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflowWithOptions(MockWESExecutionWorkflow, workflow.RegisterOptions{Name: "WESExecutionWorkflow"})
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(ReleaseInventoryReservation)

	// Mock successful validation
	env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

	// Mock child workflow: PlanningWorkflow
	planningResult := &workflows.PlanningWorkflowResult{
		ProcessPath: workflows.ProcessPathResult{
			PathID:                "PATH-002",
			Requirements:          []string{"single_item"},
			ConsolidationRequired: false,
			GiftWrapRequired:      false,
		},
		WaveID:             "WAVE-002",
		WaveScheduledStart: time.Now().Add(1 * time.Hour),
		Success:            true,
	}
	env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(planningResult, nil)

	// Mock WES execution failure
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(
		workflows.WESExecutionResult{},
		errors.New("worker unavailable for picking stage"),
	)

	// Mock compensation activity
	env.OnActivity(ReleaseInventoryReservation, mock.Anything).Return(nil)

	input := workflows.OrderFulfillmentInput{
		OrderID:            "ORD-004",
		CustomerID:         "CUST-004",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 2, Weight: 2.5}},
		Priority:           "next_day",
		PromisedDeliveryAt: time.Now().Add(48 * time.Hour),
		IsMultiItem:        false,
	}

	env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)

	// When workflow returns an error, check the error message directly
	assert.Contains(t, workflowErr.Error(), "worker unavailable")

	// Verify compensation activity was called
	env.AssertExpectations(t)
}

// TestOrderFulfillmentWorkflow_SingleItemPickPack tests single-item order goes through pick_pack path
func TestOrderFulfillmentWorkflow_SingleItemPickPack(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflow(workflows.SortationWorkflow)
	env.RegisterWorkflowWithOptions(MockWESExecutionWorkflow, workflow.RegisterOptions{Name: "WESExecutionWorkflow"})
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(ExecuteSLAM)
	env.RegisterActivity(ReleaseInventoryReservation)

	// Setup mocks
	env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

	// Mock child workflow: PlanningWorkflow returns single-item path (no walling)
	planningResult := &workflows.PlanningWorkflowResult{
		ProcessPath: workflows.ProcessPathResult{
			PathID:                "PATH-003",
			Requirements:          []string{"single_item"},
			ConsolidationRequired: false, // Single item - direct pick to pack
			GiftWrapRequired:      false,
		},
		WaveID:             "WAVE-003",
		WaveScheduledStart: time.Now(),
		Success:            true,
	}
	env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(planningResult, nil)

	// Mock WES execution with pick_pack path (no walling stage)
	wesResult := workflows.WESExecutionResult{
		RouteID:         "ROUTE-003",
		OrderID:         "ORD-005",
		Status:          "completed",
		PathType:        "pick_pack",
		StagesCompleted: 2,
		TotalStages:     2,
		PickResult: &workflows.WESStageResult{
			StageType:   "picking",
			TaskID:      "PICK-003",
			WorkerID:    "WORKER-001",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		WallingResult: nil, // No walling for single-item
		PackingResult: &workflows.WESStageResult{
			StageType:   "packing",
			TaskID:      "PACK-003",
			WorkerID:    "WORKER-002",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		CompletedAt: time.Now().Unix(),
	}
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(wesResult, nil)

	// Mock SLAM
	slamResult := workflows.SLAMResult{
		TaskID:         "SLAM-002",
		TrackingNumber: "TRACK-789",
		ManifestID:     "MANIFEST-002",
		Success:        true,
	}
	env.OnActivity(ExecuteSLAM, mock.Anything).Return(slamResult, nil)

	// Mock sortation
	sortationResult := &workflows.SortationWorkflowResult{
		BatchID: "BATCH-002",
		ChuteID: "CHUTE-002",
		Zone:    "ZONE-A",
		Success: true,
	}
	env.OnWorkflow("SortationWorkflow", mock.Anything, mock.Anything).Return(sortationResult, nil)

	env.OnWorkflow("ShippingWorkflow", mock.Anything, mock.Anything).Return(nil)

	// Single-item order
	input := workflows.OrderFulfillmentInput{
		OrderID:            "ORD-005",
		CustomerID:         "CUST-005",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 1, Weight: 2.5}},
		Priority:           "standard",
		PromisedDeliveryAt: time.Now().Add(72 * time.Hour),
		IsMultiItem:        false, // Single item
	}

	env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.OrderFulfillmentResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "TRACK-789", result.TrackingNumber)

	// Verify expectations
	env.AssertExpectations(t)
}

// TestOrderFulfillmentWorkflow_WESPartialFailure tests WES execution with partial stage failure
func TestOrderFulfillmentWorkflow_WESPartialFailure(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflowWithOptions(MockWESExecutionWorkflow, workflow.RegisterOptions{Name: "WESExecutionWorkflow"})
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(ReleaseInventoryReservation)

	// Setup successful activities up to WES
	env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

	// Mock child workflow: PlanningWorkflow
	planningResult := &workflows.PlanningWorkflowResult{
		ProcessPath: workflows.ProcessPathResult{
			PathID:                "PATH-004",
			Requirements:          []string{"multi_item"},
			ConsolidationRequired: true,
			GiftWrapRequired:      false,
		},
		WaveID:             "WAVE-004",
		WaveScheduledStart: time.Now(),
		Success:            true,
	}
	env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(planningResult, nil)

	// WES execution fails at packing stage
	wesResult := workflows.WESExecutionResult{
		RouteID:         "ROUTE-004",
		OrderID:         "ORD-006",
		Status:          "failed",
		PathType:        "pick_wall_pack",
		StagesCompleted: 2,
		TotalStages:     3,
		PickResult: &workflows.WESStageResult{
			StageType:   "picking",
			TaskID:      "PICK-004",
			WorkerID:    "WORKER-001",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		WallingResult: &workflows.WESStageResult{
			StageType:   "walling",
			TaskID:      "WALL-004",
			WorkerID:    "WORKER-002",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		PackingResult: &workflows.WESStageResult{
			StageType: "packing",
			TaskID:    "PACK-004",
			Success:   false,
			Error:     "no packing materials available",
		},
		Error: "packing stage failed: no packing materials available",
	}
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(
		wesResult,
		errors.New("packing stage failed: no packing materials available"),
	)

	input := workflows.OrderFulfillmentInput{
		OrderID:            "ORD-006",
		CustomerID:         "CUST-006",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 3, Weight: 2.5}},
		Priority:           "same_day",
		PromisedDeliveryAt: time.Now().Add(12 * time.Hour),
		IsMultiItem:        true,
	}

	env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)

	// When workflow returns an error, check the error message directly
	assert.Contains(t, workflowErr.Error(), "no packing materials available")
}

// TestOrderFulfillmentWorkflow_WithSpecialHandling tests WES with special handling requirements
func TestOrderFulfillmentWorkflow_WithSpecialHandling(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflow(workflows.SortationWorkflow)
	env.RegisterWorkflowWithOptions(MockWESExecutionWorkflow, workflow.RegisterOptions{Name: "WESExecutionWorkflow"})
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(ExecuteSLAM)
	env.RegisterActivity(ReleaseInventoryReservation)

	// Setup successful path
	env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

	// Mock child workflow: PlanningWorkflow with special handling
	planningResult := &workflows.PlanningWorkflowResult{
		ProcessPath: workflows.ProcessPathResult{
			PathID:                "PATH-005",
			Requirements:          []string{"fragile", "hazmat"},
			ConsolidationRequired: false,
			GiftWrapRequired:      false,
			SpecialHandling:       []string{"fragile", "hazmat"},
		},
		WaveID:             "WAVE-SPECIAL",
		WaveScheduledStart: time.Now(),
		Success:            true,
	}
	env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(planningResult, nil)

	// Mock WES execution with special handling
	wesResult := workflows.WESExecutionResult{
		RouteID:         "ROUTE-005",
		OrderID:         "ORD-007",
		Status:          "completed",
		PathType:        "pick_pack",
		StagesCompleted: 2,
		TotalStages:     2,
		PickResult: &workflows.WESStageResult{
			StageType:   "picking",
			TaskID:      "PICK-005",
			WorkerID:    "WORKER-CERTIFIED",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		PackingResult: &workflows.WESStageResult{
			StageType:   "packing",
			TaskID:      "PACK-005",
			WorkerID:    "WORKER-HAZMAT",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		CompletedAt: time.Now().Unix(),
	}
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(wesResult, nil)

	// Mock SLAM
	slamResult := workflows.SLAMResult{
		TaskID:         "SLAM-007",
		TrackingNumber: "TRACK-HAZMAT-001",
		ManifestID:     "MANIFEST-007",
		Success:        true,
	}
	env.OnActivity(ExecuteSLAM, mock.Anything).Return(slamResult, nil)

	env.OnWorkflow("SortationWorkflow", mock.Anything, mock.Anything).Return(&workflows.SortationWorkflowResult{
		BatchID: "BATCH-007",
		ChuteID: "CHUTE-HAZMAT",
		Zone:    "ZONE-HAZMAT",
		Success: true,
	}, nil)
	env.OnWorkflow("ShippingWorkflow", mock.Anything, mock.Anything).Return(nil)

	input := workflows.OrderFulfillmentInput{
		OrderID:    "ORD-007",
		CustomerID: "CUST-007",
		Items: []workflows.Item{
			{SKU: "SKU-HAZMAT", Quantity: 1, Weight: 2.5, IsHazmat: true, IsFragile: true},
		},
		Priority:           "same_day",
		PromisedDeliveryAt: time.Now().Add(12 * time.Hour),
		IsMultiItem:        false,
		HazmatDetails: &workflows.HazmatDetailsInput{
			Class:              "3",
			UNNumber:           "UN1234",
			PackingGroup:       "II",
			ProperShippingName: "Flammable Liquid",
		},
	}

	env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.OrderFulfillmentResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	// Should use wave from planning result
	assert.Equal(t, "WAVE-SPECIAL", result.WaveID)
	assert.Equal(t, "completed", result.Status)
}

// TestOrderFulfillmentWorkflow_SLAMFailed tests SLAM process failure
func TestOrderFulfillmentWorkflow_SLAMFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflowWithOptions(MockWESExecutionWorkflow, workflow.RegisterOptions{Name: "WESExecutionWorkflow"})
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(ExecuteSLAM)
	env.RegisterActivity(ReleaseInventoryReservation)

	// Setup successful path up to SLAM
	env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

	planningResult := &workflows.PlanningWorkflowResult{
		ProcessPath: workflows.ProcessPathResult{
			PathID:       "PATH-008",
			Requirements: []string{"single_item"},
		},
		WaveID:  "WAVE-008",
		Success: true,
	}
	env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(planningResult, nil)

	wesResult := workflows.WESExecutionResult{
		RouteID:         "ROUTE-008",
		OrderID:         "ORD-008",
		Status:          "completed",
		PathType:        "pick_pack",
		StagesCompleted: 2,
		TotalStages:     2,
		CompletedAt:     time.Now().Unix(),
	}
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(wesResult, nil)

	// SLAM fails
	env.OnActivity(ExecuteSLAM, mock.Anything).Return(workflows.SLAMResult{
		Success: false,
	}, errors.New("label printer offline"))

	input := workflows.OrderFulfillmentInput{
		OrderID:            "ORD-008",
		CustomerID:         "CUST-008",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 1, Weight: 2.5}},
		Priority:           "standard",
		PromisedDeliveryAt: time.Now().Add(72 * time.Hour),
		IsMultiItem:        false,
	}

	env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)

	assert.Contains(t, workflowErr.Error(), "label printer offline")
}

// TestOrderFulfillmentWorkflow_MultiZoneOrder tests multi-zone order handling through WES
func TestOrderFulfillmentWorkflow_MultiZoneOrder(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflow(workflows.SortationWorkflow)
	env.RegisterWorkflowWithOptions(MockWESExecutionWorkflow, workflow.RegisterOptions{Name: "WESExecutionWorkflow"})
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(ExecuteSLAM)
	env.RegisterActivity(ReleaseInventoryReservation)

	env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

	// Multi-zone order requires consolidation through walling
	planningResult := &workflows.PlanningWorkflowResult{
		ProcessPath: workflows.ProcessPathResult{
			PathID:                "PATH-MULTIZONE",
			Requirements:          []string{"multi_zone", "multi_item"},
			ConsolidationRequired: true,
			GiftWrapRequired:      false,
		},
		WaveID:             "WAVE-MULTIZONE",
		WaveScheduledStart: time.Now(),
		Success:            true,
	}
	env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(planningResult, nil)

	// WES handles multi-zone with pick_wall_pack path
	wesResult := workflows.WESExecutionResult{
		RouteID:         "ROUTE-MULTIZONE",
		OrderID:         "ORD-MULTIZONE",
		Status:          "completed",
		PathType:        "pick_wall_pack",
		StagesCompleted: 3,
		TotalStages:     3,
		PickResult: &workflows.WESStageResult{
			StageType:   "picking",
			TaskID:      "PICK-MULTIZONE",
			WorkerID:    "WORKER-ZONE-A",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		WallingResult: &workflows.WESStageResult{
			StageType:   "walling",
			TaskID:      "WALL-MULTIZONE",
			WorkerID:    "WORKER-CONSOLIDATE",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		PackingResult: &workflows.WESStageResult{
			StageType:   "packing",
			TaskID:      "PACK-MULTIZONE",
			WorkerID:    "WORKER-PACK",
			Success:     true,
			CompletedAt: time.Now().Unix(),
		},
		CompletedAt: time.Now().Unix(),
	}
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(wesResult, nil)

	env.OnActivity(ExecuteSLAM, mock.Anything).Return(workflows.SLAMResult{
		TrackingNumber: "TRACK-MULTIZONE",
		ManifestID:     "MANIFEST-MULTIZONE",
		Success:        true,
	}, nil)

	env.OnWorkflow("SortationWorkflow", mock.Anything, mock.Anything).Return(&workflows.SortationWorkflowResult{
		BatchID: "BATCH-MULTIZONE",
		Success: true,
	}, nil)
	env.OnWorkflow("ShippingWorkflow", mock.Anything, mock.Anything).Return(nil)

	input := workflows.OrderFulfillmentInput{
		OrderID:    "ORD-MULTIZONE",
		CustomerID: "CUST-MULTIZONE",
		Items: []workflows.Item{
			{SKU: "SKU-ZONE-A", Quantity: 2, Weight: 1.5},
			{SKU: "SKU-ZONE-B", Quantity: 1, Weight: 2.0},
			{SKU: "SKU-ZONE-C", Quantity: 3, Weight: 0.5},
		},
		Priority:           "same_day",
		PromisedDeliveryAt: time.Now().Add(8 * time.Hour),
		IsMultiItem:        true,
	}

	env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.OrderFulfillmentResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "TRACK-MULTIZONE", result.TrackingNumber)
	assert.Equal(t, "WAVE-MULTIZONE", result.WaveID)
}
