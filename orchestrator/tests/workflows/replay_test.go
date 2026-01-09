package workflows_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/wms-platform/orchestrator/internal/workflows"
)

// Unused import suppression
var _ = testsuite.WorkflowTestSuite{}

// TestOrderFulfillmentWorkflow_Replay tests that workflow code changes maintain determinism
// by replaying historical workflow executions against current code.
//
// This test helps prevent non-determinism errors when deploying workflow updates:
// - Verifies that new code paths are compatible with running workflows
// - Catches breaking changes before they reach production
// - Ensures workflow versioning is properly implemented
//
// To add new replay histories:
// 1. Export history from Temporal UI/CLI: temporal workflow show -w <workflow-id> -o json > history.json
// 2. Place the JSON file in testdata/ directory
// 3. The test will automatically pick it up
func TestOrderFulfillmentWorkflow_Replay(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register all workflows that may be replayed
	env.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterWorkflow(workflows.SortationWorkflow)
	env.RegisterWorkflow(workflows.ShippingWorkflow)
	env.RegisterWorkflowWithOptions(MockWESExecutionWorkflow, workflow.RegisterOptions{Name: "WESExecutionWorkflow"})

	// Register activities (mocks)
	env.RegisterActivity(ValidateOrder)
	env.RegisterActivity(ExecuteSLAM)
	env.RegisterActivity(ReleaseInventoryReservation)

	// Find all history files in testdata directory
	testdataDir := filepath.Join("testdata")
	historyFiles, err := filepath.Glob(filepath.Join(testdataDir, "order_fulfillment_*.json"))
	if err != nil {
		t.Logf("Warning: Could not read testdata directory: %v", err)
		t.Skip("No history files found in testdata directory")
		return
	}

	if len(historyFiles) == 0 {
		t.Skip("No history files found in testdata directory. To add replay tests:\n" +
			"1. Export workflow history: temporal workflow show -w <workflow-id> -o json > testdata/order_fulfillment_<name>.json\n" +
			"2. Re-run this test")
		return
	}

	for _, historyFile := range historyFiles {
		t.Run(filepath.Base(historyFile), func(t *testing.T) {
			historyData, err := os.ReadFile(historyFile)
			require.NoError(t, err, "Failed to read history file: %s", historyFile)

			// Parse the history JSON
			var history map[string]interface{}
			err = json.Unmarshal(historyData, &history)
			require.NoError(t, err, "Failed to parse history JSON: %s", historyFile)

			// Replay the workflow history using worker.ReplayWorkflowHistoryFromJSONFile
			// Note: This will fail if current workflow code is non-deterministic
			// compared to the recorded history
			replayer := worker.NewWorkflowReplayer()
			replayer.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
			replayer.RegisterWorkflow(workflows.PlanningWorkflow)
			replayer.RegisterWorkflow(workflows.SortationWorkflow)
			replayer.RegisterWorkflow(workflows.ShippingWorkflow)

			err = replayer.ReplayWorkflowHistoryFromJSONFile(nil, historyFile)
			require.NoError(t, err, "Workflow replay failed - this indicates a non-determinism error.\n"+
				"The current workflow code is not compatible with the recorded history.\n"+
				"Check for:\n"+
				"- Removed or reordered activities\n"+
				"- Changed activity/workflow inputs\n"+
				"- Missing workflow.GetVersion() calls for breaking changes\n"+
				"File: %s", historyFile)
		})
	}
}

// TestPlanningWorkflow_Replay tests planning workflow determinism
func TestPlanningWorkflow_Replay(t *testing.T) {
	testdataDir := filepath.Join("testdata")
	historyFiles, err := filepath.Glob(filepath.Join(testdataDir, "planning_*.json"))
	if err != nil || len(historyFiles) == 0 {
		t.Skip("No planning workflow history files found in testdata directory")
		return
	}

	for _, historyFile := range historyFiles {
		t.Run(filepath.Base(historyFile), func(t *testing.T) {
			replayer := worker.NewWorkflowReplayer()
			replayer.RegisterWorkflow(workflows.PlanningWorkflow)

			err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, historyFile)
			require.NoError(t, err, "Planning workflow replay failed: %s", historyFile)
		})
	}
}

// TestReprocessingWorkflow_Replay tests reprocessing workflow determinism
func TestReprocessingWorkflow_Replay(t *testing.T) {
	testdataDir := filepath.Join("testdata")
	historyFiles, err := filepath.Glob(filepath.Join(testdataDir, "reprocessing_*.json"))
	if err != nil || len(historyFiles) == 0 {
		t.Skip("No reprocessing workflow history files found in testdata directory")
		return
	}

	for _, historyFile := range historyFiles {
		t.Run(filepath.Base(historyFile), func(t *testing.T) {
			replayer := worker.NewWorkflowReplayer()
			replayer.RegisterWorkflow(workflows.ReprocessingOrchestrationWorkflow)

			err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, historyFile)
			require.NoError(t, err, "Reprocessing workflow replay failed: %s", historyFile)
		})
	}
}

/*
WorkflowHistoryExport documents how to export workflow history for replay tests

To export a workflow history for replay testing:

1. Using Temporal CLI:

	temporal workflow show -w order-fulfillment-ORD-123 -o json > testdata/order_fulfillment_ord123.json

2. Using Temporal UI:
  - Navigate to the workflow in Temporal UI
  - Click "Download" to export the history as JSON
  - Save to testdata/ directory

3. Programmatically (in Go):

	history, _ := client.GetWorkflowHistory(ctx, workflowID, runID, false, ...)
	json.Marshal(history)

Best practices for replay testing:
  - Export histories before major workflow changes
  - Include histories for different execution paths (success, failure, compensation)
  - Include histories with different workflow versions
  - Run replay tests in CI/CD before deployment
*/
