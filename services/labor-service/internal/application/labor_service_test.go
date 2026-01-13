package application

import (
	"context"
	"errors"
	"testing"

	sharedErrors "github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/labor-service/internal/domain"
)

type stubWorkerRepo struct {
	SaveFn                 func(ctx context.Context, worker *domain.Worker) error
	FindByIDFn             func(ctx context.Context, workerID string) (*domain.Worker, error)
	FindByEmployeeIDFn     func(ctx context.Context, employeeID string) (*domain.Worker, error)
	FindByStatusFn         func(ctx context.Context, status domain.WorkerStatus) ([]*domain.Worker, error)
	FindByZoneFn           func(ctx context.Context, zone string) ([]*domain.Worker, error)
	FindAvailableBySkillFn func(ctx context.Context, taskType domain.TaskType, zone string) ([]*domain.Worker, error)
	FindAllFn              func(ctx context.Context, limit, offset int) ([]*domain.Worker, error)
	DeleteFn               func(ctx context.Context, workerID string) error
}

func (s *stubWorkerRepo) Save(ctx context.Context, worker *domain.Worker) error {
	if s.SaveFn != nil {
		return s.SaveFn(ctx, worker)
	}
	return nil
}

func (s *stubWorkerRepo) FindByID(ctx context.Context, workerID string) (*domain.Worker, error) {
	if s.FindByIDFn != nil {
		return s.FindByIDFn(ctx, workerID)
	}
	return nil, nil
}

func (s *stubWorkerRepo) FindByEmployeeID(ctx context.Context, employeeID string) (*domain.Worker, error) {
	if s.FindByEmployeeIDFn != nil {
		return s.FindByEmployeeIDFn(ctx, employeeID)
	}
	return nil, nil
}

func (s *stubWorkerRepo) FindByStatus(ctx context.Context, status domain.WorkerStatus) ([]*domain.Worker, error) {
	if s.FindByStatusFn != nil {
		return s.FindByStatusFn(ctx, status)
	}
	return nil, nil
}

func (s *stubWorkerRepo) FindByZone(ctx context.Context, zone string) ([]*domain.Worker, error) {
	if s.FindByZoneFn != nil {
		return s.FindByZoneFn(ctx, zone)
	}
	return nil, nil
}

func (s *stubWorkerRepo) FindAvailableBySkill(ctx context.Context, taskType domain.TaskType, zone string) ([]*domain.Worker, error) {
	if s.FindAvailableBySkillFn != nil {
		return s.FindAvailableBySkillFn(ctx, taskType, zone)
	}
	return nil, nil
}

func (s *stubWorkerRepo) FindAll(ctx context.Context, limit, offset int) ([]*domain.Worker, error) {
	if s.FindAllFn != nil {
		return s.FindAllFn(ctx, limit, offset)
	}
	return nil, nil
}

func (s *stubWorkerRepo) Delete(ctx context.Context, workerID string) error {
	if s.DeleteFn != nil {
		return s.DeleteFn(ctx, workerID)
	}
	return nil
}

func newTestService(repo domain.WorkerRepository) *LaborApplicationService {
	logger := logging.New(logging.DefaultConfig("test"))
	return NewLaborApplicationService(repo, nil, nil, logger)
}

func workerWithShift(t *testing.T) *domain.Worker {
	t.Helper()
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	if err := worker.StartShift("shift-1", "morning", "A"); err != nil {
		t.Fatalf("unexpected start shift err: %v", err)
	}
	return worker
}

func workerWithTask(t *testing.T) *domain.Worker {
	t.Helper()
	worker := workerWithShift(t)
	if err := worker.AssignTask("task-1", domain.TaskTypePicking, 1); err != nil {
		t.Fatalf("unexpected assign task err: %v", err)
	}
	return worker
}

func TestLaborApplicationService_CreateWorker(t *testing.T) {
	var saved *domain.Worker
	repo := &stubWorkerRepo{
		SaveFn: func(_ context.Context, worker *domain.Worker) error {
			saved = worker
			return nil
		},
	}
	service := newTestService(repo)

	dto, err := service.CreateWorker(context.Background(), CreateWorkerCommand{
		WorkerID:   "worker-1",
		EmployeeID: "emp-1",
		Name:       "Ada",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if saved == nil {
		t.Fatal("expected worker to be saved")
	}
	if dto == nil || dto.WorkerID != "worker-1" || dto.EmployeeID != "emp-1" {
		t.Fatalf("unexpected dto: %#v", dto)
	}
}

func TestLaborApplicationService_CreateWorker_SaveError(t *testing.T) {
	repo := &stubWorkerRepo{
		SaveFn: func(_ context.Context, _ *domain.Worker) error {
			return errors.New("save failed")
		},
	}
	service := newTestService(repo)

	dto, err := service.CreateWorker(context.Background(), CreateWorkerCommand{
		WorkerID:   "worker-1",
		EmployeeID: "emp-1",
		Name:       "Ada",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if dto != nil {
		t.Fatalf("expected nil dto, got %#v", dto)
	}
}

func TestLaborApplicationService_GetWorker_NotFound(t *testing.T) {
	service := newTestService(&stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return nil, nil
		},
	})

	_, err := service.GetWorker(context.Background(), GetWorkerQuery{WorkerID: "missing"})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*sharedErrors.AppError)
	if !ok || appErr.Code != sharedErrors.CodeNotFound {
		t.Fatalf("expected not found AppError, got %#v", err)
	}
}

func TestLaborApplicationService_GetWorker_Success(t *testing.T) {
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service := newTestService(repo)

	dto, err := service.GetWorker(context.Background(), GetWorkerQuery{WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if dto == nil || dto.WorkerID != "worker-1" {
		t.Fatalf("unexpected dto: %#v", dto)
	}
}

func TestLaborApplicationService_StartShift_Success(t *testing.T) {
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	saved := false
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
		SaveFn: func(_ context.Context, _ *domain.Worker) error {
			saved = true
			return nil
		},
	}
	service := newTestService(repo)

	dto, err := service.StartShift(context.Background(), StartShiftCommand{
		WorkerID:  "worker-1",
		ShiftID:   "shift-1",
		ShiftType: "morning",
		Zone:      "A",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !saved {
		t.Fatal("expected worker to be saved")
	}
	if dto == nil || dto.CurrentShift == nil || dto.CurrentShift.ShiftID != "shift-1" {
		t.Fatalf("unexpected dto: %#v", dto)
	}
}

func TestLaborApplicationService_StartShift_ValidationError(t *testing.T) {
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	if err := worker.StartShift("shift-1", "morning", "A"); err != nil {
		t.Fatalf("unexpected start shift err: %v", err)
	}
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service := newTestService(repo)

	_, err := service.StartShift(context.Background(), StartShiftCommand{
		WorkerID:  "worker-1",
		ShiftID:   "shift-2",
		ShiftType: "night",
		Zone:      "B",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*sharedErrors.AppError)
	if !ok || appErr.Code != sharedErrors.CodeValidationError {
		t.Fatalf("expected validation AppError, got %#v", err)
	}
}

func TestLaborApplicationService_EndShift_ValidationError(t *testing.T) {
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service := newTestService(repo)

	_, err := service.EndShift(context.Background(), EndShiftCommand{WorkerID: "worker-1"})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*sharedErrors.AppError)
	if !ok || appErr.Code != sharedErrors.CodeValidationError {
		t.Fatalf("expected validation AppError, got %#v", err)
	}
}

func TestLaborApplicationService_StartBreak_ValidationError(t *testing.T) {
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service := newTestService(repo)

	_, err := service.StartBreak(context.Background(), StartBreakCommand{WorkerID: "worker-1", BreakType: "lunch"})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*sharedErrors.AppError)
	if !ok || appErr.Code != sharedErrors.CodeValidationError {
		t.Fatalf("expected validation AppError, got %#v", err)
	}
}

func TestLaborApplicationService_StartBreak_Success(t *testing.T) {
	worker := workerWithShift(t)
	saved := false
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
		SaveFn: func(_ context.Context, _ *domain.Worker) error {
			saved = true
			return nil
		},
	}
	service := newTestService(repo)

	dto, err := service.StartBreak(context.Background(), StartBreakCommand{WorkerID: "worker-1", BreakType: "lunch"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !saved {
		t.Fatal("expected worker to be saved")
	}
	if dto == nil || dto.Status != string(domain.WorkerStatusOnBreak) {
		t.Fatalf("unexpected dto: %#v", dto)
	}
}

func TestLaborApplicationService_EndBreak_ValidationError(t *testing.T) {
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	if err := worker.StartShift("shift-1", "morning", "A"); err != nil {
		t.Fatalf("unexpected start shift err: %v", err)
	}
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service := newTestService(repo)

	_, err := service.EndBreak(context.Background(), EndBreakCommand{WorkerID: "worker-1"})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*sharedErrors.AppError)
	if !ok || appErr.Code != sharedErrors.CodeValidationError {
		t.Fatalf("expected validation AppError, got %#v", err)
	}
}

func TestLaborApplicationService_EndBreak_Success(t *testing.T) {
	worker := workerWithShift(t)
	if err := worker.StartBreak("lunch"); err != nil {
		t.Fatalf("unexpected start break err: %v", err)
	}
	saved := false
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
		SaveFn: func(_ context.Context, _ *domain.Worker) error {
			saved = true
			return nil
		},
	}
	service := newTestService(repo)

	dto, err := service.EndBreak(context.Background(), EndBreakCommand{WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !saved {
		t.Fatal("expected worker to be saved")
	}
	if dto == nil || dto.Status != string(domain.WorkerStatusAvailable) {
		t.Fatalf("unexpected dto: %#v", dto)
	}
}

func TestLaborApplicationService_AssignTask_ValidationError(t *testing.T) {
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service := newTestService(repo)

	_, err := service.AssignTask(context.Background(), AssignTaskCommand{
		WorkerID: "worker-1",
		TaskID:   "task-1",
		TaskType: domain.TaskTypePicking,
		Priority: 1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*sharedErrors.AppError)
	if !ok || appErr.Code != sharedErrors.CodeValidationError {
		t.Fatalf("expected validation AppError, got %#v", err)
	}
}

func TestLaborApplicationService_AssignTask_Success(t *testing.T) {
	worker := workerWithShift(t)
	saved := false
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
		SaveFn: func(_ context.Context, _ *domain.Worker) error {
			saved = true
			return nil
		},
	}
	service := newTestService(repo)

	dto, err := service.AssignTask(context.Background(), AssignTaskCommand{
		WorkerID: "worker-1",
		TaskID:   "task-1",
		TaskType: domain.TaskTypePicking,
		Priority: 2,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !saved {
		t.Fatal("expected worker to be saved")
	}
	if dto == nil || dto.CurrentTask == nil || dto.CurrentTask.TaskID != "task-1" {
		t.Fatalf("unexpected dto: %#v", dto)
	}
}

func TestLaborApplicationService_StartTask_ValidationError(t *testing.T) {
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service := newTestService(repo)

	_, err := service.StartTask(context.Background(), StartTaskCommand{WorkerID: "worker-1"})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*sharedErrors.AppError)
	if !ok || appErr.Code != sharedErrors.CodeValidationError {
		t.Fatalf("expected validation AppError, got %#v", err)
	}
}

func TestLaborApplicationService_StartTask_Success(t *testing.T) {
	worker := workerWithTask(t)
	saved := false
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
		SaveFn: func(_ context.Context, _ *domain.Worker) error {
			saved = true
			return nil
		},
	}
	service := newTestService(repo)

	dto, err := service.StartTask(context.Background(), StartTaskCommand{WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !saved {
		t.Fatal("expected worker to be saved")
	}
	if dto == nil || dto.CurrentTask == nil || dto.CurrentTask.StartedAt == nil {
		t.Fatalf("unexpected dto: %#v", dto)
	}
}

func TestLaborApplicationService_CompleteTask_ValidationError(t *testing.T) {
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service := newTestService(repo)

	_, err := service.CompleteTask(context.Background(), CompleteTaskCommand{WorkerID: "worker-1", ItemsProcessed: 10})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*sharedErrors.AppError)
	if !ok || appErr.Code != sharedErrors.CodeValidationError {
		t.Fatalf("expected validation AppError, got %#v", err)
	}
}

func TestLaborApplicationService_CompleteTask_Success(t *testing.T) {
	worker := workerWithTask(t)
	saved := false
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
		SaveFn: func(_ context.Context, _ *domain.Worker) error {
			saved = true
			return nil
		},
	}
	service := newTestService(repo)

	dto, err := service.CompleteTask(context.Background(), CompleteTaskCommand{WorkerID: "worker-1", ItemsProcessed: 5})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !saved {
		t.Fatal("expected worker to be saved")
	}
	if dto == nil || dto.Status != string(domain.WorkerStatusAvailable) {
		t.Fatalf("unexpected dto: %#v", dto)
	}
}

func TestLaborApplicationService_AddSkill(t *testing.T) {
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service := newTestService(repo)

	dto, err := service.AddSkill(context.Background(), AddSkillCommand{
		WorkerID:  "worker-1",
		TaskType:  domain.TaskTypePacking,
		Level:     3,
		Certified: true,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if dto == nil || len(dto.Skills) != 1 {
		t.Fatalf("expected skill to be added, got %#v", dto)
	}
}

func TestLaborApplicationService_GetAvailable_Filtered(t *testing.T) {
	repo := &stubWorkerRepo{
		FindByStatusFn: func(_ context.Context, _ domain.WorkerStatus) ([]*domain.Worker, error) {
			workerA := domain.NewWorker("worker-1", "emp-1", "Ada")
			workerA.Status = domain.WorkerStatusAvailable
			workerA.CurrentZone = "A"
			workerB := domain.NewWorker("worker-2", "emp-2", "Bob")
			workerB.Status = domain.WorkerStatusAvailable
			workerB.CurrentZone = "B"
			return []*domain.Worker{workerA, workerB}, nil
		},
	}
	service := newTestService(repo)

	dtos, err := service.GetAvailable(context.Background(), GetAvailableQuery{Zone: "A"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(dtos) != 1 || dtos[0].WorkerID != "worker-1" {
		t.Fatalf("unexpected available result: %#v", dtos)
	}
}

func TestLaborApplicationService_GetByStatus(t *testing.T) {
	repo := &stubWorkerRepo{
		FindByStatusFn: func(_ context.Context, _ domain.WorkerStatus) ([]*domain.Worker, error) {
			worker := domain.NewWorker("worker-1", "emp-1", "Ada")
			worker.Status = domain.WorkerStatusAvailable
			return []*domain.Worker{worker}, nil
		},
	}
	service := newTestService(repo)

	dtos, err := service.GetByStatus(context.Background(), GetByStatusQuery{Status: domain.WorkerStatusAvailable})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(dtos) != 1 || dtos[0].WorkerID != "worker-1" {
		t.Fatalf("unexpected result: %#v", dtos)
	}
}

func TestLaborApplicationService_GetByZone(t *testing.T) {
	repo := &stubWorkerRepo{
		FindByZoneFn: func(_ context.Context, _ string) ([]*domain.Worker, error) {
			worker := domain.NewWorker("worker-1", "emp-1", "Ada")
			worker.CurrentZone = "A"
			return []*domain.Worker{worker}, nil
		},
	}
	service := newTestService(repo)

	dtos, err := service.GetByZone(context.Background(), GetByZoneQuery{Zone: "A"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(dtos) != 1 || dtos[0].CurrentZone != "A" {
		t.Fatalf("unexpected result: %#v", dtos)
	}
}

func TestLaborApplicationService_ListWorkers_DefaultLimit(t *testing.T) {
	called := false
	repo := &stubWorkerRepo{
		FindAllFn: func(_ context.Context, limit, offset int) ([]*domain.Worker, error) {
			called = true
			if limit != 50 || offset != 0 {
				t.Fatalf("unexpected limit/offset: %d/%d", limit, offset)
			}
			return []*domain.Worker{}, nil
		},
	}
	service := newTestService(repo)

	_, err := service.ListWorkers(context.Background(), ListWorkersQuery{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !called {
		t.Fatal("expected FindAll to be called")
	}
}
