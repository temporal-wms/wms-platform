package workflows_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	"github.com/wms-platform/orchestrator/internal/workflows"
)

// TestOrderCancellationWorkflow_Success tests successful order cancellation
func TestOrderCancellationWorkflow_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderCancellationWorkflow)
	env.RegisterActivity(CancelOrder)
	env.RegisterActivity(ReleaseInventoryReservation)
	env.RegisterActivity(NotifyCustomerCancellation)

	// Mock activities
	env.OnActivity(CancelOrder, "ORD-CANCEL-001", "Customer requested").Return(nil)
	env.OnActivity(ReleaseInventoryReservation, "ORD-CANCEL-001").Return(nil)
	env.OnActivity(NotifyCustomerCancellation, "ORD-CANCEL-001", "Customer requested").Return(nil)

	env.ExecuteWorkflow(workflows.OrderCancellationWorkflow, "ORD-CANCEL-001", "Customer requested")

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify all activities were called
	env.AssertExpectations(t)
}

// TestOrderCancellationWorkflow_CancelOrderFailed tests when cancel order activity fails
func TestOrderCancellationWorkflow_CancelOrderFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderCancellationWorkflow)
	env.RegisterActivity(CancelOrder)
	env.RegisterActivity(ReleaseInventoryReservation)
	env.RegisterActivity(NotifyCustomerCancellation)

	// Mock CancelOrder activity to fail
	env.OnActivity(CancelOrder, "ORD-CANCEL-002", "Out of stock").Return(
		errors.New("order already cancelled"),
	)

	env.ExecuteWorkflow(workflows.OrderCancellationWorkflow, "ORD-CANCEL-002", "Out of stock")

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	assert.Contains(t, env.GetWorkflowError().Error(), "failed to cancel order")

	// Verify that subsequent activities were NOT called
	env.AssertExpectations(t)
}

// TestOrderCancellationWorkflow_InventoryReleaseFailed tests when inventory release fails but workflow continues
func TestOrderCancellationWorkflow_InventoryReleaseFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderCancellationWorkflow)
	env.RegisterActivity(CancelOrder)
	env.RegisterActivity(ReleaseInventoryReservation)
	env.RegisterActivity(NotifyCustomerCancellation)

	// CancelOrder succeeds
	env.OnActivity(CancelOrder, "ORD-CANCEL-003", "Address invalid").Return(nil)

	// ReleaseInventoryReservation fails (should be logged but workflow continues)
	env.OnActivity(ReleaseInventoryReservation, "ORD-CANCEL-003").Return(
		errors.New("inventory service unavailable"),
	)

	// NotifyCustomerCancellation should still be called
	env.OnActivity(NotifyCustomerCancellation, "ORD-CANCEL-003", "Address invalid").Return(nil)

	env.ExecuteWorkflow(workflows.OrderCancellationWorkflow, "ORD-CANCEL-003", "Address invalid")

	require.True(t, env.IsWorkflowCompleted())
	// Workflow should still complete successfully despite inventory release failure
	require.NoError(t, env.GetWorkflowError())

	// Verify all activities were attempted
	env.AssertExpectations(t)
}

// TestOrderCancellationWorkflow_CustomerNotificationFailed tests when customer notification fails
func TestOrderCancellationWorkflow_CustomerNotificationFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderCancellationWorkflow)
	env.RegisterActivity(CancelOrder)
	env.RegisterActivity(ReleaseInventoryReservation)
	env.RegisterActivity(NotifyCustomerCancellation)

	// CancelOrder and ReleaseInventoryReservation succeed
	env.OnActivity(CancelOrder, "ORD-CANCEL-004", "Payment failed").Return(nil)
	env.OnActivity(ReleaseInventoryReservation, "ORD-CANCEL-004").Return(nil)

	// NotifyCustomerCancellation fails (should be logged but workflow continues)
	env.OnActivity(NotifyCustomerCancellation, "ORD-CANCEL-004", "Payment failed").Return(
		errors.New("email service down"),
	)

	env.ExecuteWorkflow(workflows.OrderCancellationWorkflow, "ORD-CANCEL-004", "Payment failed")

	require.True(t, env.IsWorkflowCompleted())
	// Workflow should still complete successfully despite notification failure
	require.NoError(t, env.GetWorkflowError())

	env.AssertExpectations(t)
}

// TestOrderCancellationWorkflow_AllCompensationsFailed tests when all compensation activities fail
func TestOrderCancellationWorkflow_AllCompensationsFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderCancellationWorkflow)
	env.RegisterActivity(CancelOrder)
	env.RegisterActivity(ReleaseInventoryReservation)
	env.RegisterActivity(NotifyCustomerCancellation)

	// Only CancelOrder succeeds
	env.OnActivity(CancelOrder, "ORD-CANCEL-005", "Duplicate order").Return(nil)

	// Both compensation activities fail
	env.OnActivity(ReleaseInventoryReservation, "ORD-CANCEL-005").Return(
		errors.New("inventory service error"),
	)
	env.OnActivity(NotifyCustomerCancellation, "ORD-CANCEL-005", "Duplicate order").Return(
		errors.New("notification service error"),
	)

	env.ExecuteWorkflow(workflows.OrderCancellationWorkflow, "ORD-CANCEL-005", "Duplicate order")

	require.True(t, env.IsWorkflowCompleted())
	// Workflow should complete successfully despite compensation failures
	// (failures are logged but don't fail the workflow)
	require.NoError(t, env.GetWorkflowError())

	env.AssertExpectations(t)
}

// TestOrderCancellationWorkflow_EmptyReason tests cancellation with empty reason
func TestOrderCancellationWorkflow_EmptyReason(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderCancellationWorkflow)
	env.RegisterActivity(CancelOrder)
	env.RegisterActivity(ReleaseInventoryReservation)
	env.RegisterActivity(NotifyCustomerCancellation)

	// Setup mocks with empty reason
	env.OnActivity(CancelOrder, "ORD-CANCEL-006", "").Return(nil)
	env.OnActivity(ReleaseInventoryReservation, "ORD-CANCEL-006").Return(nil)
	env.OnActivity(NotifyCustomerCancellation, "ORD-CANCEL-006", "").Return(nil)

	env.ExecuteWorkflow(workflows.OrderCancellationWorkflow, "ORD-CANCEL-006", "")

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	env.AssertExpectations(t)
}

// TestOrderCancellationWorkflow_Retries tests activity retry behavior
func TestOrderCancellationWorkflow_Retries(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.OrderCancellationWorkflow)
	env.RegisterActivity(CancelOrder)
	env.RegisterActivity(ReleaseInventoryReservation)
	env.RegisterActivity(NotifyCustomerCancellation)

	// Mock CancelOrder to fail twice, then succeed on third attempt
	callCount := 0
	env.OnActivity(CancelOrder, "ORD-CANCEL-007", "Test retry").Return(
		func(orderID string, reason string) error {
			callCount++
			if callCount < 3 {
				return errors.New("transient error")
			}
			return nil
		},
	)

	env.OnActivity(ReleaseInventoryReservation, "ORD-CANCEL-007").Return(nil)
	env.OnActivity(NotifyCustomerCancellation, "ORD-CANCEL-007", "Test retry").Return(nil)

	env.ExecuteWorkflow(workflows.OrderCancellationWorkflow, "ORD-CANCEL-007", "Test retry")

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify the activity was retried
	assert.Equal(t, 3, callCount)

	env.AssertExpectations(t)
}

// BenchmarkOrderCancellationWorkflow benchmarks cancellation workflow execution
func BenchmarkOrderCancellationWorkflow(b *testing.B) {
	testSuite := &testsuite.WorkflowTestSuite{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := testSuite.NewTestWorkflowEnvironment()
		env.RegisterWorkflow(workflows.OrderCancellationWorkflow)
	env.RegisterActivity(CancelOrder)
	env.RegisterActivity(ReleaseInventoryReservation)
	env.RegisterActivity(NotifyCustomerCancellation)

		env.OnActivity(CancelOrder, mock.Anything, mock.Anything).Return(nil)
		env.OnActivity(ReleaseInventoryReservation, mock.Anything).Return(nil)
		env.OnActivity(NotifyCustomerCancellation, mock.Anything, mock.Anything).Return(nil)

		env.ExecuteWorkflow(workflows.OrderCancellationWorkflow, "ORD-BENCH", "benchmark test")
	}
}
