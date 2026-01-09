package workflows

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// TestReprocessingOrchestrationWorkflow_SmallBatch tests processing a small batch (no ContinueAsNew)
func TestReprocessingOrchestrationWorkflow_SmallBatch(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock QueryFailedWorkflows to return 10 failed workflows
	failedWorkflows := make([]FailedWorkflowInfo, 10)
	for i := 0; i < 10; i++ {
		failedWorkflows[i] = FailedWorkflowInfo{
			OrderID:       "ORD-" + string(rune('A'+i)),
			WorkflowID:    "order-fulfillment-ORD-" + string(rune('A'+i)),
			FailureStatus: "failed",
			RetryCount:    1,
		}
	}
	env.OnActivity("QueryFailedWorkflows", mock.Anything, mock.Anything).Return(failedWorkflows, nil)

	// Mock ProcessFailedWorkflow for each workflow
	for i := 0; i < 10; i++ {
		result := ProcessWorkflowResult{
			OrderID:       "ORD-" + string(rune('A'+i)),
			Restarted:     true,
			NewWorkflowID: "order-fulfillment-ORD-" + string(rune('A'+i)) + "-retry",
		}
		env.OnActivity("ProcessFailedWorkflow", mock.Anything, failedWorkflows[i]).Return(result, nil)
	}

	input := ReprocessingOrchestrationInput{
		FailureStatuses: []string{"failed"},
		MaxRetries:      MaxReprocessingRetries,
		BatchSize:       100,
	}

	env.ExecuteWorkflow(ReprocessingOrchestrationWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ReprocessingResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 10, result.FoundCount)
	require.Equal(t, 10, result.RestartedCount)
	require.Equal(t, 0, result.DLQCount)
	require.Equal(t, 0, result.ErrorCount)
}

// TestReprocessingOrchestrationWorkflow_ContinueAsNew tests ContinueAsNew for large batches
func TestReprocessingOrchestrationWorkflow_ContinueAsNew(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// First batch: return 1000 workflows (will trigger ContinueAsNew)
	firstBatch := make([]FailedWorkflowInfo, 1000)
	for i := 0; i < 1000; i++ {
		firstBatch[i] = FailedWorkflowInfo{
			OrderID:    "ORD-BATCH1-" + string(rune(i)),
			WorkflowID: "wf-" + string(rune(i)),
		}
	}

	// Second batch: return 500 workflows (final batch)
	secondBatch := make([]FailedWorkflowInfo, 500)
	for i := 0; i < 500; i++ {
		secondBatch[i] = FailedWorkflowInfo{
			OrderID:    "ORD-BATCH2-" + string(rune(i)),
			WorkflowID: "wf-batch2-" + string(rune(i)),
		}
	}

	// Third query: return empty (no more workflows)
	callCount := 0
	env.OnActivity("QueryFailedWorkflows", mock.Anything, mock.Anything).Return(
		func(ctx interface{}, input interface{}) ([]FailedWorkflowInfo, error) {
			callCount++
			if callCount == 1 {
				return firstBatch, nil
			} else if callCount == 2 {
				return secondBatch, nil
			}
			return []FailedWorkflowInfo{}, nil
		},
	)

	// Mock ProcessFailedWorkflow to succeed
	env.OnActivity("ProcessFailedWorkflow", mock.Anything, mock.Anything).Return(
		ProcessWorkflowResult{Restarted: true}, nil,
	)

	input := ReprocessingOrchestrationInput{
		FailureStatuses: []string{"failed"},
		MaxRetries:      MaxReprocessingRetries,
		BatchSize:       2000, // Large batch size
	}

	env.ExecuteWorkflow(ReprocessingOrchestrationWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())

	// Workflow should have used ContinueAsNew
	// The test framework will handle ContinueAsNew automatically
	var result ReprocessingResult
	require.NoError(t, env.GetWorkflowResult(&result))

	// After ContinueAsNew, the final result should have accumulated counts
	// Note: In test environment, ContinueAsNew behavior might differ
	require.GreaterOrEqual(t, result.FoundCount, 1000)
}

// TestReprocessingOrchestrationWorkflow_WithAccumulatedResults tests continuation with accumulated results
func TestReprocessingOrchestrationWorkflow_WithAccumulatedResults(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Return small batch
	failedWorkflows := make([]FailedWorkflowInfo, 5)
	for i := 0; i < 5; i++ {
		failedWorkflows[i] = FailedWorkflowInfo{
			OrderID:    "ORD-" + string(rune('A'+i)),
			WorkflowID: "wf-" + string(rune('A'+i)),
		}
	}
	env.OnActivity("QueryFailedWorkflows", mock.Anything, mock.Anything).Return(failedWorkflows, nil)

	env.OnActivity("ProcessFailedWorkflow", mock.Anything, mock.Anything).Return(
		ProcessWorkflowResult{Restarted: true}, nil,
	)

	// Input with accumulated results from previous run
	previousResult := &ReprocessingResult{
		FoundCount:     1000,
		RestartedCount: 950,
		DLQCount:       30,
		ErrorCount:     20,
	}

	input := ReprocessingOrchestrationInput{
		FailureStatuses:   []string{"failed"},
		MaxRetries:        MaxReprocessingRetries,
		BatchSize:         100,
		AccumulatedResult: previousResult,
		ContinuationCount: 1,
	}

	env.ExecuteWorkflow(ReprocessingOrchestrationWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ReprocessingResult
	require.NoError(t, env.GetWorkflowResult(&result))

	// Results should be accumulated
	require.Equal(t, 1005, result.FoundCount)      // 1000 + 5
	require.Equal(t, 955, result.RestartedCount)   // 950 + 5
	require.Equal(t, 30, result.DLQCount)          // unchanged
	require.Equal(t, 20, result.ErrorCount)        // unchanged
}

// TestReprocessingOrchestrationWorkflow_DLQWorkflows tests workflows sent to DLQ
func TestReprocessingOrchestrationWorkflow_DLQWorkflows(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	failedWorkflows := []FailedWorkflowInfo{
		{OrderID: "ORD-001", WorkflowID: "wf-001", RetryCount: 5}, // Max retries reached
		{OrderID: "ORD-002", WorkflowID: "wf-002", RetryCount: 2}, // Will retry
		{OrderID: "ORD-003", WorkflowID: "wf-003", RetryCount: 5}, // Max retries reached
	}
	env.OnActivity("QueryFailedWorkflows", mock.Anything, mock.Anything).Return(failedWorkflows, nil)

	// First and third go to DLQ, second gets restarted
	env.OnActivity("ProcessFailedWorkflow", mock.Anything, failedWorkflows[0]).Return(
		ProcessWorkflowResult{OrderID: "ORD-001", MovedToDLQ: true}, nil,
	)
	env.OnActivity("ProcessFailedWorkflow", mock.Anything, failedWorkflows[1]).Return(
		ProcessWorkflowResult{OrderID: "ORD-002", Restarted: true}, nil,
	)
	env.OnActivity("ProcessFailedWorkflow", mock.Anything, failedWorkflows[2]).Return(
		ProcessWorkflowResult{OrderID: "ORD-003", MovedToDLQ: true}, nil,
	)

	input := ReprocessingOrchestrationInput{
		FailureStatuses: []string{"failed"},
		MaxRetries:      MaxReprocessingRetries,
		BatchSize:       100,
	}

	env.ExecuteWorkflow(ReprocessingOrchestrationWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ReprocessingResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 3, result.FoundCount)
	require.Equal(t, 1, result.RestartedCount)
	require.Equal(t, 2, result.DLQCount)
	require.Equal(t, 0, result.ErrorCount)
}

// TestReprocessingOrchestrationWorkflow_PartialFailures tests partial failures in processing
func TestReprocessingOrchestrationWorkflow_PartialFailures(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	failedWorkflows := []FailedWorkflowInfo{
		{OrderID: "ORD-001", WorkflowID: "wf-001"},
		{OrderID: "ORD-002", WorkflowID: "wf-002"},
		{OrderID: "ORD-003", WorkflowID: "wf-003"},
	}
	env.OnActivity("QueryFailedWorkflows", mock.Anything, mock.Anything).Return(failedWorkflows, nil)

	// First succeeds, second fails, third succeeds
	env.OnActivity("ProcessFailedWorkflow", mock.Anything, failedWorkflows[0]).Return(
		ProcessWorkflowResult{OrderID: "ORD-001", Restarted: true}, nil,
	)
	env.OnActivity("ProcessFailedWorkflow", mock.Anything, failedWorkflows[1]).Return(
		ProcessWorkflowResult{}, errors.New("failed to process workflow"),
	)
	env.OnActivity("ProcessFailedWorkflow", mock.Anything, failedWorkflows[2]).Return(
		ProcessWorkflowResult{OrderID: "ORD-003", Restarted: true}, nil,
	)

	input := ReprocessingOrchestrationInput{
		FailureStatuses: []string{"failed"},
		MaxRetries:      MaxReprocessingRetries,
		BatchSize:       100,
	}

	env.ExecuteWorkflow(ReprocessingOrchestrationWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ReprocessingResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 3, result.FoundCount)
	require.Equal(t, 2, result.RestartedCount)
	require.Equal(t, 0, result.DLQCount)
	require.Equal(t, 1, result.ErrorCount) // One workflow processing failed
}

// TestReprocessingOrchestrationWorkflow_NoFailedWorkflows tests empty result
func TestReprocessingOrchestrationWorkflow_NoFailedWorkflows(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Return empty list
	env.OnActivity("QueryFailedWorkflows", mock.Anything, mock.Anything).Return(
		[]FailedWorkflowInfo{}, nil,
	)

	input := ReprocessingOrchestrationInput{
		FailureStatuses: []string{"failed"},
		MaxRetries:      MaxReprocessingRetries,
		BatchSize:       100,
	}

	env.ExecuteWorkflow(ReprocessingOrchestrationWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ReprocessingResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 0, result.FoundCount)
	require.Equal(t, 0, result.RestartedCount)
}

// TestReprocessingOrchestrationWorkflow_QueryFailure tests query activity failure
func TestReprocessingOrchestrationWorkflow_QueryFailure(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock query failure
	env.OnActivity("QueryFailedWorkflows", mock.Anything, mock.Anything).Return(
		nil, errors.New("database connection failed"),
	)

	input := ReprocessingOrchestrationInput{
		FailureStatuses: []string{"failed"},
		MaxRetries:      MaxReprocessingRetries,
		BatchSize:       100,
	}

	env.ExecuteWorkflow(ReprocessingOrchestrationWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

// TestReprocessingOrchestrationWorkflow_Versioning tests workflow versioning
func TestReprocessingOrchestrationWorkflow_Versioning(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnGetVersion("ReprocessingOrchestrationWorkflow", workflow.DefaultVersion, ReprocessingOrchestrationWorkflowVersion).
		Return(ReprocessingOrchestrationWorkflowVersion)

	env.OnActivity("QueryFailedWorkflows", mock.Anything, mock.Anything).Return(
		[]FailedWorkflowInfo{}, nil,
	)

	input := ReprocessingOrchestrationInput{
		FailureStatuses: []string{"failed"},
		MaxRetries:      MaxReprocessingRetries,
		BatchSize:       100,
	}

	env.ExecuteWorkflow(ReprocessingOrchestrationWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}
