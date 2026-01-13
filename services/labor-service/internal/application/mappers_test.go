package application

import (
	"testing"
	"time"

	"github.com/wms-platform/labor-service/internal/domain"
)

func TestToWorkerDTO(t *testing.T) {
	shiftStart := time.Now().Add(-2 * time.Hour)
	shiftEnd := time.Now().Add(-1 * time.Hour)
	breakStart := time.Now().Add(-90 * time.Minute)
	breakEnd := time.Now().Add(-80 * time.Minute)
	taskStart := time.Now().Add(-70 * time.Minute)
	taskEnd := time.Now().Add(-60 * time.Minute)
	certifiedAt := time.Now().Add(-24 * time.Hour)

	worker := &domain.Worker{
		WorkerID:    "worker-1",
		EmployeeID:  "emp-1",
		Name:        "Ada",
		Status:      domain.WorkerStatusAvailable,
		CurrentZone: "A",
		Skills: []domain.Skill{
			{Type: domain.TaskTypePicking, Level: 3, Certified: true, CertifiedAt: &certifiedAt},
		},
		CurrentShift: &domain.Shift{
			ShiftID:        "shift-1",
			ShiftType:      "morning",
			Zone:           "A",
			StartTime:      shiftStart,
			EndTime:        &shiftEnd,
			BreaksTaken:    []domain.Break{{Type: "lunch", StartTime: breakStart, EndTime: &breakEnd}},
			TasksCompleted: 2,
			ItemsProcessed: 10,
		},
		CurrentTask: &domain.TaskAssignment{
			TaskID:      "task-1",
			TaskType:    domain.TaskTypePicking,
			Priority:    1,
			AssignedAt:  shiftStart,
			StartedAt:   &taskStart,
			CompletedAt: &taskEnd,
		},
		PerformanceMetrics: domain.PerformanceMetrics{
			TotalTasksCompleted: 2,
			TotalItemsProcessed: 10,
			AverageTaskTime:     15.5,
			AverageItemsPerHour: 120.0,
			AccuracyRate:        99.5,
			LastUpdated:         time.Now(),
		},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	dto := ToWorkerDTO(worker)
	if dto == nil {
		t.Fatal("expected dto")
	}
	if dto.WorkerID != "worker-1" || dto.EmployeeID != "emp-1" || dto.Name != "Ada" {
		t.Fatalf("unexpected dto: %#v", dto)
	}
	if dto.CurrentShift == nil || dto.CurrentShift.ShiftID != "shift-1" {
		t.Fatalf("unexpected shift: %#v", dto.CurrentShift)
	}
	if dto.CurrentTask == nil || dto.CurrentTask.TaskID != "task-1" {
		t.Fatalf("unexpected task: %#v", dto.CurrentTask)
	}
	if len(dto.Skills) != 1 || dto.Skills[0].Type != string(domain.TaskTypePicking) {
		t.Fatalf("unexpected skills: %#v", dto.Skills)
	}
}

func TestToWorkerDTOs_SkipsNil(t *testing.T) {
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	dtos := ToWorkerDTOs([]*domain.Worker{nil, worker})
	if len(dtos) != 1 || dtos[0].WorkerID != "worker-1" {
		t.Fatalf("unexpected dtos: %#v", dtos)
	}
}

func TestToWorkerDTO_NilInput(t *testing.T) {
	if dto := ToWorkerDTO(nil); dto != nil {
		t.Fatalf("expected nil dto, got %#v", dto)
	}
}
