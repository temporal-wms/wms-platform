package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

type stationSignal struct {
	Station        string `json:"station"`
	WorkerID       string `json:"workerId"`
	DestinationBin string `json:"destinationBin"`
}

type itemSignal struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
	ToteID   string `json:"toteId"`
}

type completeSignal struct {
	Success           bool `json:"success"`
	TotalConsolidated int  `json:"totalConsolidated"`
}

func TestConsolidationWorkflowSignals(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(ConsolidationWorkflow)
	env.RegisterActivityWithOptions(func(ctx context.Context, input map[string]interface{}) (string, error) {
		return "CONS-1", nil
	}, activity.RegisterOptions{Name: "CreateConsolidationUnit"})
	env.RegisterActivityWithOptions(func(ctx context.Context, input map[string]interface{}) error {
		return nil
	}, activity.RegisterOptions{Name: "AssignStation"})
	env.RegisterActivityWithOptions(func(ctx context.Context, consolidationID string) error {
		return nil
	}, activity.RegisterOptions{Name: "CompleteConsolidation"})

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("stationAssigned", stationSignal{
			Station:        "ST-1",
			WorkerID:       "WK-1",
			DestinationBin: "BIN-1",
		})
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("itemConsolidated", itemSignal{SKU: "SKU-1", Quantity: 2, ToteID: "TOTE-1"})
	}, 2*time.Second)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("itemConsolidated", itemSignal{SKU: "SKU-2", Quantity: 1, ToteID: "TOTE-2"})
	}, 3*time.Second)

	input := map[string]interface{}{
		"orderId": "ORD-1",
		"pickedItems": []interface{}{
			map[string]interface{}{"sku": "SKU-1", "quantity": 2, "toteId": "TOTE-1"},
			map[string]interface{}{"sku": "SKU-2", "quantity": 1, "toteId": "TOTE-2"},
		},
	}

	env.ExecuteWorkflow(ConsolidationWorkflow, input)
	require.NoError(t, env.GetWorkflowError())

	var result ConsolidationWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.True(t, result.Success)
	assert.Equal(t, 3, result.TotalConsolidated)
	assert.Equal(t, "CONS-1", result.ConsolidationID)
}

func TestConsolidationWorkflowCompleteSignal(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(ConsolidationWorkflow)
	env.RegisterActivityWithOptions(func(ctx context.Context, input map[string]interface{}) (string, error) {
		return "CONS-2", nil
	}, activity.RegisterOptions{Name: "CreateConsolidationUnit"})
	env.RegisterActivityWithOptions(func(ctx context.Context, input map[string]interface{}) error {
		return nil
	}, activity.RegisterOptions{Name: "AssignStation"})
	env.RegisterActivityWithOptions(func(ctx context.Context, consolidationID string) error {
		return nil
	}, activity.RegisterOptions{Name: "CompleteConsolidation"})

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("stationAssigned", stationSignal{
			Station:        "ST-2",
			WorkerID:       "WK-2",
			DestinationBin: "BIN-2",
		})
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("consolidationComplete", completeSignal{Success: true, TotalConsolidated: 5})
	}, 2*time.Second)

	input := map[string]interface{}{
		"orderId": "ORD-2",
		"pickedItems": []interface{}{
			map[string]interface{}{"sku": "SKU-1", "quantity": 2, "toteId": "TOTE-1"},
			map[string]interface{}{"sku": "SKU-2", "quantity": 3, "toteId": "TOTE-2"},
		},
	}

	env.ExecuteWorkflow(ConsolidationWorkflow, input)
	require.NoError(t, env.GetWorkflowError())

	var result ConsolidationWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.True(t, result.Success)
	assert.Equal(t, 5, result.TotalConsolidated)
	assert.Equal(t, "CONS-2", result.ConsolidationID)
}

func TestConsolidationWorkflowStationTimeout(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(ConsolidationWorkflow)
	env.RegisterActivityWithOptions(func(ctx context.Context, input map[string]interface{}) (string, error) {
		return "CONS-3", nil
	}, activity.RegisterOptions{Name: "CreateConsolidationUnit"})

	input := map[string]interface{}{
		"orderId": "ORD-3",
		"pickedItems": []interface{}{
			map[string]interface{}{"sku": "SKU-1", "quantity": 1, "toteId": "TOTE-1"},
		},
	}

	env.ExecuteWorkflow(ConsolidationWorkflow, input)
	err := env.GetWorkflowError()
	assert.Error(t, err)
}
