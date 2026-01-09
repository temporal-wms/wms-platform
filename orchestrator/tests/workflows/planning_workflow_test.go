package workflows_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	"github.com/wms-platform/orchestrator/internal/workflows"
)

// TestPlanningWorkflow_Success tests the happy path of the planning workflow
func TestPlanningWorkflow_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register workflow and activities
	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterActivity(DetermineProcessPath)
	env.RegisterActivity(AssignToWave)

	// Mock activity: DetermineProcessPath
	processPathResult := map[string]interface{}{
		"pathId":                "PATH-001",
		"requirements":          []string{"single_item"},
		"consolidationRequired": false,
		"giftWrapRequired":      false,
		"specialHandling":       []string{},
	}
	env.OnActivity(DetermineProcessPath, mock.Anything).Return(processPathResult, nil)

	// Mock activity: AssignToWave
	env.OnActivity(AssignToWave, mock.Anything, mock.Anything).Return(nil)

	// Prepare input
	input := workflows.PlanningWorkflowInput{
		OrderID:            "ORD-001",
		CustomerID:         "CUST-001",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 1, Weight: 2.5}},
		Priority:           "same_day",
		PromisedDeliveryAt: time.Now().Add(24 * time.Hour),
		IsMultiItem:        false,
		GiftWrap:           false,
		TotalValue:         100.0,
		UseUnitTracking:    false,
	}

	// Register signal to be sent after workflow initializes
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("waveAssigned", workflows.WaveAssignment{
			WaveID:         "WAVE-001",
			ScheduledStart: time.Now().Add(15 * time.Minute),
		})
	}, 0)

	// Execute workflow
	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	// Verify workflow completed successfully
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify result
	var result workflows.PlanningWorkflowResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, "WAVE-001", result.WaveID)
	assert.Empty(t, result.Error)
}

// TestPlanningWorkflow_ProcessPathFailed tests process path determination failure
func TestPlanningWorkflow_ProcessPathFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterActivity(DetermineProcessPath)
	env.RegisterActivity(AssignToWave)

	// Mock activity: DetermineProcessPath returns error
	env.OnActivity(DetermineProcessPath, mock.Anything).Return(
		map[string]interface{}{},
		errors.New("failed to determine process path"),
	)

	input := workflows.PlanningWorkflowInput{
		OrderID:            "ORD-002",
		CustomerID:         "CUST-002",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 1, Weight: 2.5}},
		Priority:           "standard",
		PromisedDeliveryAt: time.Now().Add(72 * time.Hour),
		IsMultiItem:        false,
		UseUnitTracking:    false,
	}

	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)
	assert.Contains(t, workflowErr.Error(), "failed to determine process path")
}

// TestPlanningWorkflow_WaveAssignmentTimeout tests wave assignment timeout
func TestPlanningWorkflow_WaveAssignmentTimeout(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterActivity(DetermineProcessPath)
	env.RegisterActivity(AssignToWave)

	// Mock activity: DetermineProcessPath succeeds
	processPathResult := map[string]interface{}{
		"pathId":                "PATH-002",
		"requirements":          []string{"single_item"},
		"consolidationRequired": false,
		"giftWrapRequired":      false,
		"specialHandling":       []string{},
	}
	env.OnActivity(DetermineProcessPath, mock.Anything).Return(processPathResult, nil)

	input := workflows.PlanningWorkflowInput{
		OrderID:            "ORD-003",
		CustomerID:         "CUST-003",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 1, Weight: 2.5}},
		Priority:           "same_day", // 30-minute timeout
		PromisedDeliveryAt: time.Now().Add(24 * time.Hour),
		IsMultiItem:        false,
		UseUnitTracking:    false,
	}

	// Don't send wave assignment signal - let it timeout
	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)
	assert.Contains(t, workflowErr.Error(), "wave assignment timeout")
	assert.Contains(t, workflowErr.Error(), "ORD-003")
}

// TestPlanningWorkflow_WithUnitTracking tests planning workflow with unit-level tracking enabled
func TestPlanningWorkflow_WithUnitTracking(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterActivity(DetermineProcessPath)
	env.RegisterActivity(PersistProcessPath)
	env.RegisterActivity(ReserveUnits)
	env.RegisterActivity(AssignToWave)

	// Mock activity: DetermineProcessPath
	processPathResult := map[string]interface{}{
		"pathId":                "PATH-003",
		"requirements":          []string{"multi_item"},
		"consolidationRequired": true,
		"giftWrapRequired":      false,
		"specialHandling":       []string{},
	}
	env.OnActivity(DetermineProcessPath, mock.Anything).Return(processPathResult, nil)

	// Mock activity: PersistProcessPath
	env.OnActivity(PersistProcessPath, mock.Anything).Return(map[string]string{"pathId": "PATH-003"}, nil)

	// Mock activity: ReserveUnits
	reserveResult := map[string]interface{}{
		"reservedUnits": []interface{}{
			map[string]interface{}{"unitId": "UNIT-001"},
			map[string]interface{}{"unitId": "UNIT-002"},
		},
		"failedItems": []interface{}{},
	}
	env.OnActivity(ReserveUnits, mock.Anything).Return(reserveResult, nil)

	// Mock activity: AssignToWave
	env.OnActivity(AssignToWave, mock.Anything, mock.Anything).Return(nil)

	input := workflows.PlanningWorkflowInput{
		OrderID:    "ORD-004",
		CustomerID: "CUST-004",
		Items: []workflows.Item{
			{SKU: "SKU-001", Quantity: 2, Weight: 2.5},
			{SKU: "SKU-002", Quantity: 1, Weight: 3.0},
		},
		Priority:           "next_day",
		PromisedDeliveryAt: time.Now().Add(48 * time.Hour),
		IsMultiItem:        true,
		UseUnitTracking:    true, // Enable unit tracking
	}

	// Register signal to be sent after workflow initializes
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("waveAssigned", workflows.WaveAssignment{
			WaveID:         "WAVE-004",
			ScheduledStart: time.Now().Add(1 * time.Hour),
		})
	}, 0)

	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.PlanningWorkflowResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, "WAVE-004", result.WaveID)
	assert.Equal(t, "PATH-003", result.PathID)
	assert.Len(t, result.ReservedUnitIDs, 2)
	assert.Contains(t, result.ReservedUnitIDs, "UNIT-001")
	assert.Contains(t, result.ReservedUnitIDs, "UNIT-002")
}

// TestPlanningWorkflow_UnitReservationFailed tests unit reservation failure
func TestPlanningWorkflow_UnitReservationFailed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterActivity(DetermineProcessPath)
	env.RegisterActivity(PersistProcessPath)
	env.RegisterActivity(ReserveUnits)
	env.RegisterActivity(AssignToWave)

	// Mock activity: DetermineProcessPath
	processPathResult := map[string]interface{}{
		"pathId":                "PATH-005",
		"requirements":          []string{"single_item"},
		"consolidationRequired": false,
		"giftWrapRequired":      false,
		"specialHandling":       []string{},
	}
	env.OnActivity(DetermineProcessPath, mock.Anything).Return(processPathResult, nil)

	// Mock activity: PersistProcessPath
	env.OnActivity(PersistProcessPath, mock.Anything).Return(map[string]string{"pathId": "PATH-005"}, nil)

	// Mock activity: ReserveUnits fails
	env.OnActivity(ReserveUnits, mock.Anything).Return(
		map[string]interface{}{},
		errors.New("insufficient inventory for reservation"),
	)

	input := workflows.PlanningWorkflowInput{
		OrderID:            "ORD-005",
		CustomerID:         "CUST-005",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 100, Weight: 2.5}},
		Priority:           "standard",
		PromisedDeliveryAt: time.Now().Add(72 * time.Hour),
		IsMultiItem:        false,
		UseUnitTracking:    true, // Enable unit tracking
	}

	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr)
	assert.Contains(t, workflowErr.Error(), "insufficient inventory for reservation")
}

// TestPlanningWorkflow_WithPreReservedUnits tests planning with pre-reserved unit IDs
func TestPlanningWorkflow_WithPreReservedUnits(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterActivity(DetermineProcessPath)
	env.RegisterActivity(PersistProcessPath)
	env.RegisterActivity(AssignToWave)

	// Mock activity: DetermineProcessPath
	processPathResult := map[string]interface{}{
		"pathId":                "PATH-006",
		"requirements":          []string{"single_item"},
		"consolidationRequired": false,
		"giftWrapRequired":      false,
		"specialHandling":       []string{},
	}
	env.OnActivity(DetermineProcessPath, mock.Anything).Return(processPathResult, nil)

	// Mock activity: PersistProcessPath
	env.OnActivity(PersistProcessPath, mock.Anything).Return(map[string]string{"pathId": "PATH-006"}, nil)

	// Note: ReserveUnits should NOT be called since we have pre-reserved units

	// Mock activity: AssignToWave
	env.OnActivity(AssignToWave, mock.Anything, mock.Anything).Return(nil)

	input := workflows.PlanningWorkflowInput{
		OrderID:            "ORD-006",
		CustomerID:         "CUST-006",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 2, Weight: 2.5}},
		Priority:           "same_day",
		PromisedDeliveryAt: time.Now().Add(24 * time.Hour),
		IsMultiItem:        false,
		UseUnitTracking:    true,
		UnitIDs:            []string{"PRE-UNIT-001", "PRE-UNIT-002"}, // Pre-reserved units
	}

	// Register signal to be sent after workflow initializes
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("waveAssigned", workflows.WaveAssignment{
			WaveID:         "WAVE-006",
			ScheduledStart: time.Now().Add(15 * time.Minute),
		})
	}, 0)

	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.PlanningWorkflowResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, "WAVE-006", result.WaveID)
	// Should use pre-reserved units
	assert.Len(t, result.ReservedUnitIDs, 2)
	assert.Contains(t, result.ReservedUnitIDs, "PRE-UNIT-001")
	assert.Contains(t, result.ReservedUnitIDs, "PRE-UNIT-002")
}

// TestPlanningWorkflow_GiftWrapRequired tests planning with gift wrap requirements
func TestPlanningWorkflow_GiftWrapRequired(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterActivity(DetermineProcessPath)
	env.RegisterActivity(AssignToWave)

	// Mock activity: DetermineProcessPath returns gift wrap required
	processPathResult := map[string]interface{}{
		"pathId":                "PATH-007",
		"requirements":          []string{"single_item", "gift_wrap"},
		"consolidationRequired": false,
		"giftWrapRequired":      true,
		"specialHandling":       []string{"premium_packaging"},
	}
	env.OnActivity(DetermineProcessPath, mock.Anything).Return(processPathResult, nil)

	// Mock activity: AssignToWave
	env.OnActivity(AssignToWave, mock.Anything, mock.Anything).Return(nil)

	input := workflows.PlanningWorkflowInput{
		OrderID:            "ORD-007",
		CustomerID:         "CUST-007",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 1, Weight: 2.5}},
		Priority:           "next_day",
		PromisedDeliveryAt: time.Now().Add(48 * time.Hour),
		IsMultiItem:        false,
		GiftWrap:           true,
		GiftWrapDetails: &workflows.GiftWrapDetailsInput{
			WrapType:    "premium",
			GiftMessage: "Happy Birthday!",
			HidePrice:   true,
		},
		TotalValue:      150.0,
		UseUnitTracking: false,
	}

	// Register signal to be sent after workflow initializes
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("waveAssigned", workflows.WaveAssignment{
			WaveID:         "WAVE-007",
			ScheduledStart: time.Now().Add(1 * time.Hour),
		})
	}, 0)

	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.PlanningWorkflowResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.True(t, result.ProcessPath.GiftWrapRequired)
}

// TestPlanningWorkflow_HighValueOrder tests planning with high-value order
func TestPlanningWorkflow_HighValueOrder(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.PlanningWorkflow)
	env.RegisterActivity(DetermineProcessPath)
	env.RegisterActivity(AssignToWave)

	// Mock activity: DetermineProcessPath returns high value requirements
	processPathResult := map[string]interface{}{
		"pathId":                "PATH-008",
		"requirements":          []string{"single_item", "high_value"},
		"consolidationRequired": false,
		"giftWrapRequired":      false,
		"specialHandling":       []string{"high_value_verification"},
	}
	env.OnActivity(DetermineProcessPath, mock.Anything).Return(processPathResult, nil)

	// Mock activity: AssignToWave
	env.OnActivity(AssignToWave, mock.Anything, mock.Anything).Return(nil)

	input := workflows.PlanningWorkflowInput{
		OrderID:            "ORD-008",
		CustomerID:         "CUST-008",
		Items:              []workflows.Item{{SKU: "SKU-HIGH", Quantity: 1, Weight: 0.5}},
		Priority:           "same_day",
		PromisedDeliveryAt: time.Now().Add(12 * time.Hour),
		IsMultiItem:        false,
		TotalValue:         1500.0, // High value order (>$500)
		UseUnitTracking:    false,
	}

	// Register signal to be sent after workflow initializes
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("waveAssigned", workflows.WaveAssignment{
			WaveID:         "WAVE-008",
			ScheduledStart: time.Now().Add(15 * time.Minute),
		})
	}, 0)

	env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.PlanningWorkflowResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Contains(t, result.ProcessPath.SpecialHandling, "high_value_verification")
}

// TestPlanningWorkflow_DifferentPriorityTimeouts tests different timeout values based on priority
func TestPlanningWorkflow_DifferentPriorityTimeouts(t *testing.T) {
	priorities := []struct {
		name     string
		priority string
	}{
		{"same_day", "same_day"},
		{"next_day", "next_day"},
		{"standard", "standard"},
	}

	for _, tc := range priorities {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()

			env.RegisterWorkflow(workflows.PlanningWorkflow)
			env.RegisterActivity(DetermineProcessPath)
			env.RegisterActivity(AssignToWave)

			processPathResult := map[string]interface{}{
				"pathId":                "PATH-TIMEOUT",
				"requirements":          []string{"single_item"},
				"consolidationRequired": false,
				"giftWrapRequired":      false,
				"specialHandling":       []string{},
			}
			env.OnActivity(DetermineProcessPath, mock.Anything).Return(processPathResult, nil)

			input := workflows.PlanningWorkflowInput{
				OrderID:            "ORD-TIMEOUT-" + tc.priority,
				CustomerID:         "CUST-TIMEOUT",
				Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 1, Weight: 2.5}},
				Priority:           tc.priority,
				PromisedDeliveryAt: time.Now().Add(72 * time.Hour),
				IsMultiItem:        false,
				UseUnitTracking:    false,
			}

			// Don't send wave assignment signal - let it timeout
			env.ExecuteWorkflow(workflows.PlanningWorkflow, input)

			require.True(t, env.IsWorkflowCompleted())
			workflowErr := env.GetWorkflowError()
			require.Error(t, workflowErr)
			assert.Contains(t, workflowErr.Error(), "wave assignment timeout")
		})
	}
}

// BenchmarkPlanningWorkflow benchmarks the planning workflow execution
func BenchmarkPlanningWorkflow(b *testing.B) {
	testSuite := &testsuite.WorkflowTestSuite{}

	input := workflows.PlanningWorkflowInput{
		OrderID:            "ORD-BENCH",
		CustomerID:         "CUST-BENCH",
		Items:              []workflows.Item{{SKU: "SKU-001", Quantity: 1, Weight: 2.5}},
		Priority:           "standard",
		PromisedDeliveryAt: time.Now().Add(72 * time.Hour),
		IsMultiItem:        false,
		UseUnitTracking:    false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := testSuite.NewTestWorkflowEnvironment()
		env.RegisterWorkflow(workflows.PlanningWorkflow)
		env.RegisterActivity(DetermineProcessPath)
		env.RegisterActivity(AssignToWave)

		processPathResult := map[string]interface{}{
			"pathId":                "PATH-BENCH",
			"requirements":          []string{"single_item"},
			"consolidationRequired": false,
			"giftWrapRequired":      false,
			"specialHandling":       []string{},
		}
		env.OnActivity(DetermineProcessPath, mock.Anything).Return(processPathResult, nil)
		env.OnActivity(AssignToWave, mock.Anything, mock.Anything).Return(nil)

		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow("waveAssigned", workflows.WaveAssignment{
				WaveID:         "WAVE-BENCH",
				ScheduledStart: time.Now(),
			})
		}, 0)

		env.ExecuteWorkflow(workflows.PlanningWorkflow, input)
	}
}
