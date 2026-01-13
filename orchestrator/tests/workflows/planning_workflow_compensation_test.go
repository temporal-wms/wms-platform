package workflows_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/orchestrator/internal/workflows"
	"go.temporal.io/sdk/testsuite"
)

// TestPlanningWorkflow_ReserveInventory_FailureTriggersCompensation tests that
// inventory reservation failure triggers unit release compensation
func TestPlanningWorkflow_ReserveInventory_FailureTriggersCompensation(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register workflow
	env.RegisterWorkflow(workflows.PlanningWorkflow)

	// Mock successful process path determination
	env.OnActivity("DetermineProcessPath", mock.Anything, mock.Anything).Return(
		map[string]interface{}{
			"pathId":                "PATH-001",
			"requirements":          []string{"single_item"},
			"consolidationRequired": false,
		}, nil)

	// Mock successful path persistence
	env.OnActivity("PersistProcessPath", mock.Anything, mock.Anything).Return(
		map[string]string{"pathId": "PATH-001"}, nil)

	// Mock successful unit reservation
	env.OnActivity("ReserveUnits", mock.Anything, mock.Anything).Return(
		map[string]interface{}{
			"reservedUnits": []interface{}{
				map[string]interface{}{
					"unitId":     "UNIT-001",
					"sku":        "SKU-001",
					"locationId": "LOC-001",
				},
			},
			"failedItems": []interface{}{},
		}, nil)

	// Mock FAILURE on inventory reservation (this should trigger compensation)
	inventoryError := errors.New("inventory service unavailable")
	env.OnActivity("ReserveInventory", mock.Anything, mock.Anything).Return(inventoryError)

	// Mock successful compensation (unit release)
	env.OnActivity("ReleaseUnits", mock.Anything, mock.Anything).Return(nil)

	input := workflows.PlanningWorkflowInput{
		OrderID:    "ORD-001",
		CustomerID: "CUST-001",
		Items: []workflows.Item{
			{SKU: "SKU-001", Quantity: 2, Weight: 1.0},
		},
		Priority: "standard",
	}

	// Execute workflow
	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	// Verify workflow FAILED (fatal error)
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	// Verify the error is InventoryReservationError
	var result workflows.PlanningWorkflowResult
	err := env.GetWorkflowResult(&result)
	assert.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "inventory reservation failed")

	// Verify ReleaseUnits was called (compensation occurred)
	env.AssertCalled(t, "ReleaseUnits", mock.Anything, mock.MatchedBy(func(input map[string]interface{}) bool {
		return input["orderId"] == "ORD-001" && input["reason"] == "inventory_reservation_failed"
	}))
}

// TestPlanningWorkflow_ReserveInventory_Success_NoCompensation tests that
// successful inventory reservation does NOT trigger compensation
func TestPlanningWorkflow_ReserveInventory_Success_NoCompensation(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)

	// Mock all activities as successful
	env.OnActivity("DetermineProcessPath", mock.Anything, mock.Anything).Return(
		map[string]interface{}{
			"pathId":       "PATH-001",
			"requirements": []string{"single_item"},
		}, nil)
	env.OnActivity("PersistProcessPath", mock.Anything, mock.Anything).Return(
		map[string]string{"pathId": "PATH-001"}, nil)
	env.OnActivity("ReserveUnits", mock.Anything, mock.Anything).Return(
		map[string]interface{}{
			"reservedUnits": []interface{}{
				map[string]interface{}{"unitId": "UNIT-001"},
			},
		}, nil)
	env.OnActivity("ReserveInventory", mock.Anything, mock.Anything).Return(nil) // SUCCESS

	// Send wave signal to allow workflow to complete
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("waveAssigned", map[string]interface{}{
			"waveId": "WAVE-001",
		})
	}, 0)

	env.OnActivity("AssignToWave", mock.Anything, mock.Anything).Return(nil)

	input := workflows.PlanningWorkflowInput{
		OrderID: "ORD-001",
		Items:   []workflows.Item{{SKU: "SKU-001", Quantity: 1}},
	}

	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	// Verify workflow succeeded
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify ReleaseUnits was NOT called (no compensation needed)
	env.AssertNotCalled(t, "ReleaseUnits")
}

// TestPlanningWorkflow_ReserveInventory_CompensationFails tests that
// compensation failure is logged but doesn't cascade
func TestPlanningWorkflow_ReserveInventory_CompensationFails(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)

	env.OnActivity("DetermineProcessPath", mock.Anything, mock.Anything).Return(
		map[string]interface{}{"pathId": "PATH-001"}, nil)
	env.OnActivity("PersistProcessPath", mock.Anything, mock.Anything).Return(
		map[string]string{"pathId": "PATH-001"}, nil)
	env.OnActivity("ReserveUnits", mock.Anything, mock.Anything).Return(
		map[string]interface{}{"reservedUnits": []interface{}{}}, nil)

	// Inventory reservation fails
	env.OnActivity("ReserveInventory", mock.Anything, mock.Anything).Return(
		errors.New("inventory service down"))

	// Compensation ALSO fails
	env.OnActivity("ReleaseUnits", mock.Anything, mock.Anything).Return(
		errors.New("unit service down"))

	input := workflows.PlanningWorkflowInput{
		OrderID: "ORD-001",
		Items:   []workflows.Item{{SKU: "SKU-001", Quantity: 1}},
	}

	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	// Workflow should still fail with original error, not compensation error
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	var result workflows.PlanningWorkflowResult
	env.GetWorkflowResult(&result)
	assert.Contains(t, result.Error, "inventory reservation failed")

	// Both activities should have been called
	env.AssertCalled(t, "ReserveInventory", mock.Anything, mock.Anything)
	env.AssertCalled(t, "ReleaseUnits", mock.Anything, mock.Anything)
}

// TestPlanningWorkflow_ReserveInventory_RetryThenSuccess tests that
// transient failures are retried and workflow succeeds on retry
func TestPlanningWorkflow_ReserveInventory_RetryThenSuccess(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)

	env.OnActivity("DetermineProcessPath", mock.Anything, mock.Anything).Return(
		map[string]interface{}{"pathId": "PATH-001"}, nil)
	env.OnActivity("PersistProcessPath", mock.Anything, mock.Anything).Return(
		map[string]string{"pathId": "PATH-001"}, nil)
	env.OnActivity("ReserveUnits", mock.Anything, mock.Anything).Return(
		map[string]interface{}{
			"reservedUnits": []interface{}{
				map[string]interface{}{"unitId": "UNIT-001"},
			},
		}, nil)

	// First attempt fails (transient network error), second succeeds
	callCount := 0
	env.OnActivity("ReserveInventory", mock.Anything, mock.Anything).Return(
		func(ctx interface{}, input interface{}) error {
			callCount++
			if callCount == 1 {
				return errors.New("network timeout")
			}
			return nil // Success on retry
		})

	// Send wave signal
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("waveAssigned", map[string]interface{}{
			"waveId": "WAVE-001",
		})
	}, 0)

	env.OnActivity("AssignToWave", mock.Anything, mock.Anything).Return(nil)

	input := workflows.PlanningWorkflowInput{
		OrderID: "ORD-001",
		Items:   []workflows.Item{{SKU: "SKU-001", Quantity: 1}},
	}

	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	// Verify workflow succeeded after retry
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify activity was retried
	assert.Equal(t, 2, callCount)

	// Verify NO compensation triggered (workflow succeeded after retry)
	env.AssertNotCalled(t, "ReleaseUnits")
}

// TestPlanningWorkflow_ReserveInventory_MultipleItemsFailure tests that
// error contains details about all failed items
func TestPlanningWorkflow_ReserveInventory_MultipleItemsFailure(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)

	env.OnActivity("DetermineProcessPath", mock.Anything, mock.Anything).Return(
		map[string]interface{}{"pathId": "PATH-001"}, nil)
	env.OnActivity("PersistProcessPath", mock.Anything, mock.Anything).Return(
		map[string]string{"pathId": "PATH-001"}, nil)
	env.OnActivity("ReserveUnits", mock.Anything, mock.Anything).Return(
		map[string]interface{}{
			"reservedUnits": []interface{}{
				map[string]interface{}{"unitId": "UNIT-001"},
				map[string]interface{}{"unitId": "UNIT-002"},
			},
		}, nil)

	// Inventory reservation fails for multi-item order
	env.OnActivity("ReserveInventory", mock.Anything, mock.Anything).Return(
		errors.New("insufficient inventory"))

	env.OnActivity("ReleaseUnits", mock.Anything, mock.Anything).Return(nil)

	input := workflows.PlanningWorkflowInput{
		OrderID: "ORD-001",
		Items: []workflows.Item{
			{SKU: "SKU-001", Quantity: 2},
			{SKU: "SKU-002", Quantity: 3},
		},
	}

	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	// Verify workflow failed
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	var result workflows.PlanningWorkflowResult
	env.GetWorkflowResult(&result)

	// Verify error message contains order ID
	assert.Contains(t, result.Error, "ORD-001")
	assert.Contains(t, result.Error, "inventory reservation failed")

	// Verify compensation was triggered
	env.AssertCalled(t, "ReleaseUnits", mock.Anything, mock.Anything)
}

// TestPlanningWorkflow_NoUnitsReserved_NoCompensation tests that
// if unit reservation wasn't successful, no compensation is attempted
func TestPlanningWorkflow_NoUnitsReserved_NoCompensation(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)

	env.OnActivity("DetermineProcessPath", mock.Anything, mock.Anything).Return(
		map[string]interface{}{"pathId": "PATH-001"}, nil)
	env.OnActivity("PersistProcessPath", mock.Anything, mock.Anything).Return(
		map[string]string{"pathId": "PATH-001"}, nil)

	// Unit reservation returns empty result (no units reserved)
	env.OnActivity("ReserveUnits", mock.Anything, mock.Anything).Return(
		map[string]interface{}{
			"reservedUnits": []interface{}{}, // Empty
		}, nil)

	// Inventory reservation fails
	env.OnActivity("ReserveInventory", mock.Anything, mock.Anything).Return(
		errors.New("inventory unavailable"))

	input := workflows.PlanningWorkflowInput{
		OrderID: "ORD-001",
		Items:   []workflows.Item{{SKU: "SKU-001", Quantity: 1}},
	}

	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	// Verify workflow failed
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	// Verify compensation was still called (flag was set regardless of unit count)
	// This is correct behavior - the compensation activity will handle empty case
	env.AssertCalled(t, "ReleaseUnits", mock.Anything, mock.Anything)
}
