package workflows

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// TestOrderFulfillmentWorkflow_Success tests the happy path
func TestOrderFulfillmentWorkflow_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock all activities
	env.OnActivity("ValidateOrder", mock.Anything, mock.Anything).Return(true, nil)

	// Mock PlanningWorkflow child workflow
	planningResult := &PlanningWorkflowResult{
		WaveID:             "WAVE-001",
		PathID:             "PATH-001",
		WaveScheduledStart: time.Now().Add(1 * time.Hour),
		ProcessPath: ProcessPathResult{
			PathID:                "PATH-001",
			ConsolidationRequired: false,
			GiftWrapRequired:      false,
			SpecialHandling:       []string{},
		},
		ReservedUnitIDs: []string{},
	}
	env.OnWorkflow(PlanningWorkflow, mock.Anything, mock.Anything).Return(planningResult, nil)

	// Mock WES Execution child workflow
	wesResult := WESExecutionResult{
		OrderID:         "ORD-001",
		RouteID:         "ROUTE-001",
		Status:          "completed",
		PathType:        "standard",
		StagesCompleted: 3,
		TotalStages:     3,
	}
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(wesResult, nil)

	// Mock remaining activities
	env.OnActivity("MarkPacked", mock.Anything, mock.Anything).Return(nil)
	env.OnActivity("GenerateShippingLabel", mock.Anything, mock.Anything).Return("LABEL-001", nil)
	env.OnActivity("NotifyShipping", mock.Anything, mock.Anything).Return(nil)

	// Execute workflow
	input := OrderFulfillmentInput{
		OrderID:            "ORD-001",
		CustomerID:         "CUST-001",
		Items:              []Item{{SKU: "SKU-001", Quantity: 1}},
		Priority:           "standard",
		PromisedDeliveryAt: time.Now().Add(48 * time.Hour),
		IsMultiItem:        false,
	}

	env.ExecuteWorkflow(OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result OrderFulfillmentResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "ORD-001", result.OrderID)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "WAVE-001", result.WaveID)
}

// TestOrderFulfillmentWorkflow_ValidationFailed tests order validation failure
func TestOrderFulfillmentWorkflow_ValidationFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock validation failure with ApplicationError
	validationErr := temporal.NewApplicationError(
		"order validation failed: invalid SKU",
		"OrderValidationFailed",
		nil,
	)
	env.OnActivity("ValidateOrder", mock.Anything, mock.Anything).Return(false, validationErr)

	input := OrderFulfillmentInput{
		OrderID:    "ORD-002",
		CustomerID: "CUST-001",
		Items:      []Item{{SKU: "INVALID", Quantity: 1}},
	}

	env.ExecuteWorkflow(OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	var result OrderFulfillmentResult
	env.GetWorkflowResult(&result)
	require.Equal(t, "validation_failed", result.Status)
}

// TestOrderFulfillmentWorkflow_PlanningFailed tests planning workflow failure
func TestOrderFulfillmentWorkflow_PlanningFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity("ValidateOrder", mock.Anything, mock.Anything).Return(true, nil)

	// Mock planning failure
	planningErr := errors.New("inventory reservation failed")
	env.OnWorkflow(PlanningWorkflow, mock.Anything, mock.Anything).Return(nil, planningErr)

	input := OrderFulfillmentInput{
		OrderID:    "ORD-003",
		CustomerID: "CUST-001",
		Items:      []Item{{SKU: "SKU-001", Quantity: 100}}, // Large quantity
	}

	env.ExecuteWorkflow(OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	var result OrderFulfillmentResult
	env.GetWorkflowResult(&result)
	require.Equal(t, "planning_failed", result.Status)
	require.Contains(t, result.Error, "inventory reservation failed")
}

// TestOrderFulfillmentWorkflow_WESExecutionFailed tests WES execution failure
func TestOrderFulfillmentWorkflow_WESExecutionFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity("ValidateOrder", mock.Anything, mock.Anything).Return(true, nil)

	planningResult := &PlanningWorkflowResult{
		WaveID:             "WAVE-001",
		PathID:             "PATH-001",
		WaveScheduledStart: time.Now().Add(1 * time.Hour),
		ProcessPath: ProcessPathResult{
			PathID:                "PATH-001",
			ConsolidationRequired: false,
		},
	}
	env.OnWorkflow(PlanningWorkflow, mock.Anything, mock.Anything).Return(planningResult, nil)

	// Mock WES failure
	wesErr := errors.New("picking task creation failed")
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(nil, wesErr)

	input := OrderFulfillmentInput{
		OrderID:    "ORD-004",
		CustomerID: "CUST-001",
		Items:      []Item{{SKU: "SKU-001", Quantity: 1}},
	}

	env.ExecuteWorkflow(OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	var result OrderFulfillmentResult
	env.GetWorkflowResult(&result)
	require.Equal(t, "wes_execution_failed", result.Status)
}

// TestOrderFulfillmentWorkflow_QueryStatus tests the query handler
func TestOrderFulfillmentWorkflow_QueryStatus(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Set up mocks for successful workflow
	env.OnActivity("ValidateOrder", mock.Anything, mock.Anything).Return(true, nil)

	planningResult := &PlanningWorkflowResult{
		WaveID:             "WAVE-001",
		PathID:             "PATH-001",
		WaveScheduledStart: time.Now().Add(1 * time.Hour),
		ProcessPath: ProcessPathResult{
			PathID:                "PATH-001",
			ConsolidationRequired: false,
		},
	}
	env.OnWorkflow(PlanningWorkflow, mock.Anything, mock.Anything).Return(planningResult, nil)

	wesResult := WESExecutionResult{
		OrderID:         "ORD-005",
		Status:          "completed",
		StagesCompleted: 3,
	}
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(wesResult, nil)
	env.OnActivity("MarkPacked", mock.Anything, mock.Anything).Return(nil)
	env.OnActivity("GenerateShippingLabel", mock.Anything, mock.Anything).Return("LABEL-001", nil)
	env.OnActivity("NotifyShipping", mock.Anything, mock.Anything).Return(nil)

	// Register query handler
	env.RegisterDelayedCallback(func() {
		// Query status after validation completes
		val, err := env.QueryWorkflow("getStatus")
		require.NoError(t, err)

		var status OrderFulfillmentQueryStatus
		require.NoError(t, val.Get(&status))
		require.Equal(t, "ORD-005", status.OrderID)
		require.Equal(t, "in_progress", status.Status)
		require.Greater(t, status.CompletionPercent, 0)
	}, 1*time.Second)

	input := OrderFulfillmentInput{
		OrderID:    "ORD-005",
		CustomerID: "CUST-001",
		Items:      []Item{{SKU: "SKU-001", Quantity: 1}},
	}

	env.ExecuteWorkflow(OrderFulfillmentWorkflow, input)
	require.True(t, env.IsWorkflowCompleted())
}

// TestOrderFulfillmentWorkflow_MultiItemOrder tests multi-item order flow
func TestOrderFulfillmentWorkflow_MultiItemOrder(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity("ValidateOrder", mock.Anything, mock.Anything).Return(true, nil)

	planningResult := &PlanningWorkflowResult{
		WaveID:             "WAVE-001",
		PathID:             "PATH-001",
		WaveScheduledStart: time.Now().Add(1 * time.Hour),
		ProcessPath: ProcessPathResult{
			PathID:                "PATH-001",
			ConsolidationRequired: true, // Multi-item requires consolidation
		},
	}
	env.OnWorkflow(PlanningWorkflow, mock.Anything, mock.Anything).Return(planningResult, nil)

	wesResult := WESExecutionResult{
		OrderID:         "ORD-006",
		Status:          "completed",
		StagesCompleted: 3,
	}
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(wesResult, nil)
	env.OnActivity("MarkPacked", mock.Anything, mock.Anything).Return(nil)
	env.OnActivity("GenerateShippingLabel", mock.Anything, mock.Anything).Return("LABEL-001", nil)
	env.OnActivity("NotifyShipping", mock.Anything, mock.Anything).Return(nil)

	input := OrderFulfillmentInput{
		OrderID:     "ORD-006",
		CustomerID:  "CUST-001",
		Items:       []Item{{SKU: "SKU-001", Quantity: 1}, {SKU: "SKU-002", Quantity: 2}},
		IsMultiItem: true,
	}

	env.ExecuteWorkflow(OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result OrderFulfillmentResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "completed", result.Status)
}

// TestOrderFulfillmentWorkflow_Versioning tests workflow versioning
func TestOrderFulfillmentWorkflow_Versioning(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock GetVersion to return specific version
	env.OnGetVersion("OrderFulfillmentWorkflow", workflow.DefaultVersion, OrderFulfillmentWorkflowVersion).
		Return(OrderFulfillmentWorkflowVersion)

	env.OnActivity("ValidateOrder", mock.Anything, mock.Anything).Return(true, nil)

	planningResult := &PlanningWorkflowResult{
		WaveID:             "WAVE-001",
		PathID:             "PATH-001",
		WaveScheduledStart: time.Now().Add(1 * time.Hour),
		ProcessPath:        ProcessPathResult{PathID: "PATH-001"},
	}
	env.OnWorkflow(PlanningWorkflow, mock.Anything, mock.Anything).Return(planningResult, nil)

	wesResult := WESExecutionResult{OrderID: "ORD-007", Status: "completed"}
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(wesResult, nil)
	env.OnActivity("MarkPacked", mock.Anything, mock.Anything).Return(nil)
	env.OnActivity("GenerateShippingLabel", mock.Anything, mock.Anything).Return("LABEL-001", nil)
	env.OnActivity("NotifyShipping", mock.Anything, mock.Anything).Return(nil)

	input := OrderFulfillmentInput{
		OrderID:    "ORD-007",
		CustomerID: "CUST-001",
		Items:      []Item{{SKU: "SKU-001", Quantity: 1}},
	}

	env.ExecuteWorkflow(OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestOrderFulfillmentWorkflow_ActivityRetry tests activity retry logic
func TestOrderFulfillmentWorkflow_ActivityRetry(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock validation to fail twice, then succeed (testing retry)
	callCount := 0
	env.OnActivity("ValidateOrder", mock.Anything, mock.Anything).Return(
		func(ctx interface{}, input interface{}) (bool, error) {
			callCount++
			if callCount < 3 {
				return false, errors.New("temporary service error")
			}
			return true, nil
		},
	)

	planningResult := &PlanningWorkflowResult{
		WaveID:             "WAVE-001",
		PathID:             "PATH-001",
		WaveScheduledStart: time.Now().Add(1 * time.Hour),
		ProcessPath:        ProcessPathResult{PathID: "PATH-001"},
	}
	env.OnWorkflow(PlanningWorkflow, mock.Anything, mock.Anything).Return(planningResult, nil)

	wesResult := WESExecutionResult{OrderID: "ORD-008", Status: "completed"}
	env.OnWorkflow("WESExecutionWorkflow", mock.Anything, mock.Anything).Return(wesResult, nil)
	env.OnActivity("MarkPacked", mock.Anything, mock.Anything).Return(nil)
	env.OnActivity("GenerateShippingLabel", mock.Anything, mock.Anything).Return("LABEL-001", nil)
	env.OnActivity("NotifyShipping", mock.Anything, mock.Anything).Return(nil)

	input := OrderFulfillmentInput{
		OrderID:    "ORD-008",
		CustomerID: "CUST-001",
		Items:      []Item{{SKU: "SKU-001", Quantity: 1}},
	}

	env.ExecuteWorkflow(OrderFulfillmentWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 3, callCount, "Activity should have been retried 3 times")
}
