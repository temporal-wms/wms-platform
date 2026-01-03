package workflows_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	"github.com/wms-platform/orchestrator/internal/workflows"
)

// TestOrderFulfillmentWorkflow_Success tests the happy path of order fulfillment
func TestOrderFulfillmentWorkflow_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register workflows (including child workflows)
	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.PickingWorkflow)
	env.RegisterWorkflow(workflows.ConsolidationWorkflow)
	env.RegisterWorkflow(workflows.PackingWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflow(workflows.SortationWorkflow)
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(CalculateMultiRoute)
	env.RegisterActivity(ReleaseInventoryReservation)
	env.RegisterActivity(StartPicking)
	env.RegisterActivity(MarkConsolidated)
	env.RegisterActivity(MarkPacked)
	env.RegisterActivity(FindCapableStation)
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

	// Mock activity: CalculateMultiRoute
	multiRouteResult := workflows.MultiRouteResult{
		OrderID:     "ORD-001",
		TotalRoutes: 1,
		Routes: []workflows.RouteResult{{
			RouteID: "ROUTE-001",
			Stops: []workflows.RouteStop{
				{LocationID: "LOC-A1", SKU: "SKU-001", Quantity: 2},
			},
			EstimatedDistance: 150.5,
			Strategy:          "shortest_path",
		}},
	}
	env.OnActivity(CalculateMultiRoute, mock.Anything).Return(multiRouteResult, nil)
	env.OnActivity(StartPicking, mock.Anything).Return(nil)
	env.OnActivity(MarkConsolidated, mock.Anything).Return(nil)
	env.OnActivity(MarkPacked, mock.Anything).Return(nil)
	env.OnActivity(FindCapableStation, mock.Anything).Return(map[string]interface{}{"stationId": "STATION-001"}, nil)
	env.OnActivity(ExecuteSLAM, mock.Anything).Return(workflows.SLAMResult{
		TrackingNumber: "TRACK-123456",
		ManifestID:     "MANIFEST-001",
		Success:        true,
		CarrierID:      "CARRIER-001",
		Destination:    "12345",
	}, nil)

	// Mock child workflow: PickingWorkflow
	pickResult := workflows.PickResult{
		TaskID: "PICK-001",
		PickedItems: []workflows.PickedItem{
			{SKU: "SKU-001", Quantity: 2, LocationID: "LOC-A1", ToteID: "TOTE-123"},
		},
		Success: true,
	}
	env.OnWorkflow("PickingWorkflow", mock.Anything, mock.Anything).Return(pickResult, nil)

	// Mock child workflow: ConsolidationWorkflow (for multi-item orders)
	env.OnWorkflow("ConsolidationWorkflow", mock.Anything, mock.Anything).Return(nil)

	// Mock child workflow: PackingWorkflow
	packResult := workflows.PackResult{
		PackageID:      "PKG-001",
		TrackingNumber: "TRACK-123456",
		Carrier:        "UPS",
		Weight:         5.5,
	}
	env.OnWorkflow("PackingWorkflow", mock.Anything, mock.Anything).Return(packResult, nil)

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
	env.RegisterWorkflow(workflows.PickingWorkflow)
	env.RegisterWorkflow(workflows.ConsolidationWorkflow)
	env.RegisterWorkflow(workflows.PackingWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(CalculateRoute)
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
	env.RegisterWorkflow(workflows.PickingWorkflow)
	env.RegisterWorkflow(workflows.ConsolidationWorkflow)
	env.RegisterWorkflow(workflows.PackingWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(CalculateRoute)
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

// TestOrderFulfillmentWorkflow_PickingFailed tests picking workflow failure with compensation
func TestOrderFulfillmentWorkflow_PickingFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.PickingWorkflow)
	env.RegisterWorkflow(workflows.ConsolidationWorkflow)
	env.RegisterWorkflow(workflows.PackingWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(CalculateMultiRoute)
	env.RegisterActivity(StartPicking)
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

	// Mock successful route calculation
	multiRouteResult := workflows.MultiRouteResult{
		OrderID:     "ORD-004",
		TotalRoutes: 1,
		Routes:      []workflows.RouteResult{{RouteID: "ROUTE-002", Stops: []workflows.RouteStop{{LocationID: "LOC-A1", SKU: "SKU-001", Quantity: 2}}}},
	}
	env.OnActivity(CalculateMultiRoute, mock.Anything).Return(multiRouteResult, nil)

	// Mock successful start picking
	env.OnActivity(StartPicking, mock.Anything).Return(nil)

	// Mock picking workflow failure
	env.OnWorkflow("PickingWorkflow", mock.Anything, mock.Anything).Return(
		workflows.PickResult{},
		errors.New("picker unavailable"),
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
	assert.Contains(t, workflowErr.Error(), "picker unavailable")

	// Verify compensation activity was called
	env.AssertExpectations(t)
}

// TestOrderFulfillmentWorkflow_SingleItemSkipsConsolidation tests that single-item orders skip consolidation
func TestOrderFulfillmentWorkflow_SingleItemSkipsConsolidation(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.PickingWorkflow)
	env.RegisterWorkflow(workflows.ConsolidationWorkflow)
	env.RegisterWorkflow(workflows.PackingWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(CalculateMultiRoute)
	env.RegisterActivity(StartPicking)
	env.RegisterActivity(MarkConsolidated)
	env.RegisterActivity(FindCapableStation)
	env.RegisterActivity(MarkPacked)
	env.RegisterActivity(ExecuteSLAM)
	env.RegisterActivity(ReleaseInventoryReservation)

	// Setup mocks
	env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

	// Mock child workflow: PlanningWorkflow returns single-item (no consolidation)
	planningResult := &workflows.PlanningWorkflowResult{
		ProcessPath: workflows.ProcessPathResult{
			PathID:                "PATH-003",
			Requirements:          []string{"single_item"},
			ConsolidationRequired: false, // Single item - no consolidation
			GiftWrapRequired:      false,
		},
		WaveID:             "WAVE-003",
		WaveScheduledStart: time.Now(),
		Success:            true,
	}
	env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(planningResult, nil)

	multiRouteResult := workflows.MultiRouteResult{
		OrderID:     "ORD-005",
		TotalRoutes: 1,
		Routes:      []workflows.RouteResult{{RouteID: "ROUTE-003"}},
	}
	env.OnActivity(CalculateMultiRoute, mock.Anything).Return(multiRouteResult, nil)
	env.OnActivity(StartPicking, mock.Anything).Return(nil)

	pickResult := workflows.PickResult{
		TaskID:      "PICK-002",
		PickedItems: []workflows.PickedItem{{SKU: "SKU-001", Quantity: 1}},
		Success:     true,
	}
	env.OnWorkflow("PickingWorkflow", mock.Anything, mock.Anything).Return(pickResult, nil)

	// Note: Single item order skips consolidation, so we don't mock ConsolidationWorkflow

	// Mock capable station finding
	env.OnActivity(FindCapableStation, mock.Anything).Return(map[string]interface{}{"stationId": "STATION-002"}, nil)

	packResult := workflows.PackResult{
		PackageID:      "PKG-002",
		TrackingNumber: "TRACK-789",
		Carrier:        "FedEx",
	}
	env.OnWorkflow("PackingWorkflow", mock.Anything, mock.Anything).Return(packResult, nil)
	env.OnActivity(MarkPacked, mock.Anything).Return(nil)

	// Mock SLAM
	slamResult := workflows.SLAMResult{
		TaskID:         "SLAM-002",
		TrackingNumber: "TRACK-789",
		ManifestID:     "MANIFEST-002",
		Success:        true,
	}
	env.OnActivity(ExecuteSLAM, mock.Anything).Return(slamResult, nil)

	// Mock sortation
	env.RegisterWorkflow(workflows.SortationWorkflow)
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

	// Verify expectations
	env.AssertExpectations(t)
}

// TestOrderFulfillmentWorkflow_PackingFailed tests packing workflow failure
func TestOrderFulfillmentWorkflow_PackingFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.PickingWorkflow)
	env.RegisterWorkflow(workflows.ConsolidationWorkflow)
	env.RegisterWorkflow(workflows.PackingWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(CalculateMultiRoute)
	env.RegisterActivity(StartPicking)
	env.RegisterActivity(MarkConsolidated)
	env.RegisterActivity(FindCapableStation)
	env.RegisterActivity(ReleaseInventoryReservation)

	// Setup successful activities up to packing
	env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

	// Mock child workflow: PlanningWorkflow
	planningResult := &workflows.PlanningWorkflowResult{
		ProcessPath: workflows.ProcessPathResult{
			PathID:                "PATH-004",
			Requirements:          []string{"single_item"},
			ConsolidationRequired: false,
			GiftWrapRequired:      false,
		},
		WaveID:             "WAVE-004",
		WaveScheduledStart: time.Now(),
		Success:            true,
	}
	env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(planningResult, nil)

	multiRouteResult := workflows.MultiRouteResult{
		OrderID:     "ORD-006",
		TotalRoutes: 1,
		Routes:      []workflows.RouteResult{{RouteID: "ROUTE-004"}},
	}
	env.OnActivity(CalculateMultiRoute, mock.Anything).Return(multiRouteResult, nil)
	env.OnActivity(StartPicking, mock.Anything).Return(nil)
	env.OnWorkflow("PickingWorkflow", mock.Anything, mock.Anything).Return(
		workflows.PickResult{TaskID: "PICK-003", Success: true},
		nil,
	)
	env.OnWorkflow("ConsolidationWorkflow", mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(MarkConsolidated, mock.Anything).Return(nil)
	env.OnActivity(FindCapableStation, mock.Anything).Return(map[string]interface{}{"stationId": "STATION-006"}, nil)

	// Packing fails
	env.OnWorkflow("PackingWorkflow", mock.Anything, mock.Anything).Return(
		workflows.PackResult{},
		errors.New("no packing materials available"),
	)

	input := workflows.OrderFulfillmentInput{
		OrderID:            "ORD-006",
		CustomerID:         "CUST-006",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 1, Weight: 2.5}},
		Priority:           "same_day",
		PromisedDeliveryAt: time.Now().Add(12 * time.Hour),
		IsMultiItem:        false,
	}

	env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)

	// When workflow returns an error, check the error message directly
	assert.Contains(t, workflowErr.Error(), "no packing materials available")
}

// TestOrderFulfillmentWorkflow_WithPlanningResult tests that planning results are correctly used
func TestOrderFulfillmentWorkflow_WithPlanningResult(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.PickingWorkflow)
	env.RegisterWorkflow(workflows.ConsolidationWorkflow)
	env.RegisterWorkflow(workflows.PackingWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflow(workflows.SortationWorkflow)
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(CalculateMultiRoute)
	env.RegisterActivity(StartPicking)
	env.RegisterActivity(MarkConsolidated)
	env.RegisterActivity(FindCapableStation)
	env.RegisterActivity(MarkPacked)
	env.RegisterActivity(ExecuteSLAM)
	env.RegisterActivity(ReleaseInventoryReservation)

	// Setup successful path
	env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

	// Mock child workflow: PlanningWorkflow with specific wave
	planningResult := &workflows.PlanningWorkflowResult{
		ProcessPath: workflows.ProcessPathResult{
			PathID:                "PATH-005",
			Requirements:          []string{"single_item"},
			ConsolidationRequired: false,
			GiftWrapRequired:      false,
		},
		WaveID:             "WAVE-FIRST",
		WaveScheduledStart: time.Now(),
		Success:            true,
	}
	env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(planningResult, nil)

	multiRouteResult := workflows.MultiRouteResult{
		OrderID:     "ORD-007",
		TotalRoutes: 1,
		Routes:      []workflows.RouteResult{{RouteID: "ROUTE-005"}},
	}
	env.OnActivity(CalculateMultiRoute, mock.Anything).Return(multiRouteResult, nil)
	env.OnActivity(StartPicking, mock.Anything).Return(nil)
	env.OnWorkflow("PickingWorkflow", mock.Anything, mock.Anything).Return(workflows.PickResult{TaskID: "PICK-004", Success: true}, nil)
	env.OnWorkflow("ConsolidationWorkflow", mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(MarkConsolidated, mock.Anything).Return(nil)
	env.OnActivity(FindCapableStation, mock.Anything).Return(map[string]interface{}{"stationId": "STATION-007"}, nil)
	env.OnWorkflow("PackingWorkflow", mock.Anything, mock.Anything).Return(workflows.PackResult{PackageID: "PKG-005", TrackingNumber: "TRACK-999"}, nil)
	env.OnActivity(MarkPacked, mock.Anything).Return(nil)
	slamResult := workflows.SLAMResult{
		TaskID:         "SLAM-007",
		TrackingNumber: "TRACK-999",
		ManifestID:     "MANIFEST-007",
		Success:        true,
	}
	env.OnActivity(ExecuteSLAM, mock.Anything).Return(slamResult, nil)
	env.OnWorkflow("SortationWorkflow", mock.Anything, mock.Anything).Return(&workflows.SortationWorkflowResult{BatchID: "BATCH-007", ChuteID: "CHUTE-007", Zone: "ZONE-A", Success: true}, nil)
	env.OnWorkflow("ShippingWorkflow", mock.Anything, mock.Anything).Return(nil)

	input := workflows.OrderFulfillmentInput{
		OrderID:            "ORD-007",
		CustomerID:         "CUST-007",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 1, Weight: 2.5}},
		Priority:           "same_day",
		PromisedDeliveryAt: time.Now().Add(12 * time.Hour),
		IsMultiItem:        false,
	}

	env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.OrderFulfillmentResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	// Should use wave from planning result
	assert.Equal(t, "WAVE-FIRST", result.WaveID)
	assert.Equal(t, "completed", result.Status)
}

// BenchmarkOrderFulfillmentWorkflow benchmarks the workflow execution
func BenchmarkOrderFulfillmentWorkflow(b *testing.B) {
	testSuite := &testsuite.WorkflowTestSuite{}

	input := workflows.OrderFulfillmentInput{
		OrderID:            "ORD-BENCH",
		CustomerID:         "CUST-BENCH",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 1, Weight: 2.5}},
		Priority:           "standard",
		PromisedDeliveryAt: time.Now().Add(72 * time.Hour),
		IsMultiItem:        false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := testSuite.NewTestWorkflowEnvironment()
		env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
		env.RegisterWorkflow(workflows.PlanningWorkflow)
		env.RegisterWorkflow(workflows.PickingWorkflow)
		env.RegisterWorkflow(workflows.ConsolidationWorkflow)
		env.RegisterWorkflow(workflows.PackingWorkflow)
		env.RegisterWorkflow(workflows.ShippingWorkflow)
		env.RegisterActivity(ValidateOrder)
		env.RegisterActivity(CalculateRoute)
		env.RegisterActivity(ReleaseInventoryReservation)

		env.OnActivity(ValidateOrder, mock.Anything).Return(true, nil)

		// Mock PlanningWorkflow
		waveID := fmt.Sprintf("WAVE-%d", i)
		planningResult := &workflows.PlanningWorkflowResult{
			ProcessPath: workflows.ProcessPathResult{
				PathID:                "PATH-BENCH",
				Requirements:          []string{"single_item"},
				ConsolidationRequired: false,
				GiftWrapRequired:      false,
			},
			WaveID:             waveID,
			WaveScheduledStart: time.Now(),
			Success:            true,
		}
		env.OnWorkflow("PlanningWorkflow", mock.Anything, mock.Anything).Return(planningResult, nil)

		env.OnActivity(CalculateRoute, mock.Anything).Return(workflows.RouteResult{RouteID: fmt.Sprintf("ROUTE-%d", i)}, nil)
		env.OnWorkflow("PickingWorkflow", mock.Anything, mock.Anything).Return(workflows.PickResult{Success: true}, nil)
		env.OnWorkflow("PackingWorkflow", mock.Anything, mock.Anything).Return(workflows.PackResult{TrackingNumber: "TRACK-123"}, nil)
		env.OnWorkflow("ShippingWorkflow", mock.Anything, mock.Anything).Return(nil)

		env.ExecuteWorkflow(workflows.OrderFulfillmentWorkflow, input)
	}
}
