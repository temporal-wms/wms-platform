package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/wes-service/internal/domain"
)

// createTestTemplate creates a test stage template
func createTestTemplate(pathType domain.ProcessPathType, stageCount int) *domain.StageTemplate {
	stages := make([]domain.StageDefinition, 0)

	switch pathType {
	case domain.PathPickPack:
		stages = []domain.StageDefinition{
			{Order: 1, StageType: domain.StagePicking, TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: domain.StagePacking, TaskType: "packing", Required: true, TimeoutMins: 15},
		}
	case domain.PathPickWallPack:
		stages = []domain.StageDefinition{
			{Order: 1, StageType: domain.StagePicking, TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: domain.StageWalling, TaskType: "walling", Required: true, TimeoutMins: 10},
			{Order: 3, StageType: domain.StagePacking, TaskType: "packing", Required: true, TimeoutMins: 15},
		}
	case domain.PathPickConsolidatePack:
		stages = []domain.StageDefinition{
			{Order: 1, StageType: domain.StagePicking, TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: domain.StageConsolidation, TaskType: "consolidation", Required: true, TimeoutMins: 20},
			{Order: 3, StageType: domain.StagePacking, TaskType: "packing", Required: true, TimeoutMins: 15},
		}
	}

	maxItems := 100
	return domain.NewStageTemplate(
		"tpl-test-"+string(pathType),
		pathType,
		"Test Template",
		"Test template for "+string(pathType),
		stages,
		domain.SelectionCriteria{MaxItems: &maxItems, Priority: 1},
	)
}

func TestNewTaskRoute(t *testing.T) {
	template := createTestTemplate(domain.PathPickPack, 2)

	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, []string{"fragile"}, "PATH-001")

	require.NotNil(t, route)
	assert.NotEmpty(t, route.RouteID)
	assert.Equal(t, "ORD-001", route.OrderID)
	assert.Equal(t, "WAVE-001", route.WaveID)
	assert.Equal(t, "tpl-test-pick_pack", route.PathTemplateID)
	assert.Equal(t, domain.PathPickPack, route.PathType)
	assert.Equal(t, domain.RouteStatusPending, route.Status)
	assert.Equal(t, 0, route.CurrentStageIdx)
	assert.Len(t, route.Stages, 2)
	assert.Equal(t, []string{"fragile"}, route.SpecialHandling)
	assert.Equal(t, "PATH-001", route.ProcessPathID)

	// Check stages are initialized properly
	assert.Equal(t, domain.StagePicking, route.Stages[0].StageType)
	assert.Equal(t, domain.StageStatusPending, route.Stages[0].Status)
	assert.Equal(t, domain.StagePacking, route.Stages[1].StageType)
	assert.Equal(t, domain.StageStatusPending, route.Stages[1].Status)

	// Check domain event was created
	assert.Len(t, route.DomainEvents, 1)
}

func TestTaskRoute_GetCurrentStage(t *testing.T) {
	template := createTestTemplate(domain.PathPickPack, 2)
	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, nil, "")

	stage := route.GetCurrentStage()

	require.NotNil(t, stage)
	assert.Equal(t, domain.StagePicking, stage.StageType)
	assert.Equal(t, domain.StageStatusPending, stage.Status)
}

func TestTaskRoute_AssignWorkerToCurrentStage(t *testing.T) {
	template := createTestTemplate(domain.PathPickPack, 2)
	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, nil, "")

	err := route.AssignWorkerToCurrentStage("WORKER-001", "TASK-001")

	require.NoError(t, err)
	assert.Equal(t, domain.RouteStatusInProgress, route.Status)

	stage := route.GetCurrentStage()
	assert.Equal(t, "WORKER-001", stage.WorkerID)
	assert.Equal(t, "TASK-001", stage.TaskID)
	assert.Equal(t, domain.StageStatusAssigned, stage.Status)

	// Check domain event
	assert.Len(t, route.DomainEvents, 2) // RouteCreated + StageAssigned
}

func TestTaskRoute_AssignWorkerToCurrentStage_AlreadyAssigned(t *testing.T) {
	template := createTestTemplate(domain.PathPickPack, 2)
	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, nil, "")

	// First assignment
	err := route.AssignWorkerToCurrentStage("WORKER-001", "TASK-001")
	require.NoError(t, err)

	// Second assignment should fail
	err = route.AssignWorkerToCurrentStage("WORKER-002", "TASK-002")
	assert.Error(t, err)
	assert.Equal(t, domain.ErrStageNotPending, err)
}

func TestTaskRoute_StartCurrentStage(t *testing.T) {
	template := createTestTemplate(domain.PathPickPack, 2)
	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, nil, "")

	// Must assign before starting
	err := route.AssignWorkerToCurrentStage("WORKER-001", "TASK-001")
	require.NoError(t, err)

	// Now start the stage
	err = route.StartCurrentStage()
	require.NoError(t, err)

	stage := route.GetCurrentStage()
	assert.Equal(t, domain.StageStatusInProgress, stage.Status)
	assert.NotNil(t, stage.StartedAt)

	// Check domain event
	events := route.DomainEvents
	assert.Len(t, events, 3) // RouteCreated + StageAssigned + StageStarted
}

func TestTaskRoute_StartCurrentStage_NotAssigned(t *testing.T) {
	template := createTestTemplate(domain.PathPickPack, 2)
	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, nil, "")

	// Try to start without assigning
	err := route.StartCurrentStage()
	assert.Error(t, err)
}

func TestTaskRoute_CompleteCurrentStage(t *testing.T) {
	template := createTestTemplate(domain.PathPickPack, 2)
	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, nil, "")

	// Setup: assign and start
	route.AssignWorkerToCurrentStage("WORKER-001", "TASK-001")
	route.StartCurrentStage()

	// Complete the stage
	err := route.CompleteCurrentStage()
	require.NoError(t, err)

	// Check first stage is completed
	assert.Equal(t, domain.StageStatusCompleted, route.Stages[0].Status)
	assert.NotNil(t, route.Stages[0].CompletedAt)

	// Check we moved to next stage
	assert.Equal(t, 1, route.CurrentStageIdx)
	stage := route.GetCurrentStage()
	assert.Equal(t, domain.StagePacking, stage.StageType)
	assert.Equal(t, domain.StageStatusPending, stage.Status)

	// Route should still be in progress
	assert.Equal(t, domain.RouteStatusInProgress, route.Status)
}

func TestTaskRoute_CompleteAllStages(t *testing.T) {
	template := createTestTemplate(domain.PathPickPack, 2)
	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, nil, "")

	// Complete first stage (picking)
	route.AssignWorkerToCurrentStage("WORKER-001", "TASK-001")
	route.StartCurrentStage()
	route.CompleteCurrentStage()

	// Complete second stage (packing)
	route.AssignWorkerToCurrentStage("WORKER-002", "TASK-002")
	route.StartCurrentStage()
	err := route.CompleteCurrentStage()
	require.NoError(t, err)

	// Route should be completed
	assert.Equal(t, domain.RouteStatusCompleted, route.Status)
	assert.True(t, route.IsCompleted())
	assert.NotNil(t, route.CompletedAt)

	// Verify progress
	completed, total := route.GetProgress()
	assert.Equal(t, 2, completed)
	assert.Equal(t, 2, total)
}

func TestTaskRoute_FailCurrentStage(t *testing.T) {
	template := createTestTemplate(domain.PathPickPack, 2)
	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, nil, "")

	// Assign worker
	route.AssignWorkerToCurrentStage("WORKER-001", "TASK-001")
	route.StartCurrentStage()

	// Fail the stage
	err := route.FailCurrentStage("item not found at location")
	require.NoError(t, err)

	// Check stage is failed
	stage := route.GetCurrentStage()
	assert.Equal(t, domain.StageStatusFailed, stage.Status)
	assert.Equal(t, "item not found at location", stage.Error)

	// Check route is failed
	assert.Equal(t, domain.RouteStatusFailed, route.Status)
	assert.True(t, route.IsFailed())
}

func TestTaskRoute_FailCurrentStage_AlreadyCompleted(t *testing.T) {
	template := createTestTemplate(domain.PathPickPack, 2)
	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, nil, "")

	// Complete all stages
	route.AssignWorkerToCurrentStage("WORKER-001", "TASK-001")
	route.StartCurrentStage()
	route.CompleteCurrentStage()
	route.AssignWorkerToCurrentStage("WORKER-002", "TASK-002")
	route.StartCurrentStage()
	route.CompleteCurrentStage()

	// Try to fail a stage after route is completed
	err := route.FailCurrentStage("some error")
	assert.Error(t, err)
	assert.Equal(t, domain.ErrRouteAlreadyCompleted, err)
}

func TestTaskRoute_PickWallPackPath(t *testing.T) {
	template := createTestTemplate(domain.PathPickWallPack, 3)
	route := domain.NewTaskRoute("ORD-002", "WAVE-002", template, nil, "")

	assert.Len(t, route.Stages, 3)
	assert.Equal(t, domain.StagePicking, route.Stages[0].StageType)
	assert.Equal(t, domain.StageWalling, route.Stages[1].StageType)
	assert.Equal(t, domain.StagePacking, route.Stages[2].StageType)

	// Complete all 3 stages
	for i := 0; i < 3; i++ {
		route.AssignWorkerToCurrentStage("WORKER-"+string(rune('A'+i)), "TASK-"+string(rune('A'+i)))
		route.StartCurrentStage()
		err := route.CompleteCurrentStage()
		require.NoError(t, err)
	}

	assert.True(t, route.IsCompleted())
	completed, total := route.GetProgress()
	assert.Equal(t, 3, completed)
	assert.Equal(t, 3, total)
}

func TestTaskRoute_ClearDomainEvents(t *testing.T) {
	template := createTestTemplate(domain.PathPickPack, 2)
	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, nil, "")

	// Verify we have events
	assert.NotEmpty(t, route.GetDomainEvents())

	// Clear events
	route.ClearDomainEvents()
	assert.Empty(t, route.GetDomainEvents())
}

func TestTaskRoute_GetProgress(t *testing.T) {
	template := createTestTemplate(domain.PathPickWallPack, 3)
	route := domain.NewTaskRoute("ORD-001", "WAVE-001", template, nil, "")

	// Initial progress
	completed, total := route.GetProgress()
	assert.Equal(t, 0, completed)
	assert.Equal(t, 3, total)

	// Complete first stage
	route.AssignWorkerToCurrentStage("WORKER-001", "TASK-001")
	route.StartCurrentStage()
	route.CompleteCurrentStage()

	completed, total = route.GetProgress()
	assert.Equal(t, 1, completed)
	assert.Equal(t, 3, total)
}
