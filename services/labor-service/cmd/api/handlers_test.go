package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/labor-service/internal/application"
	"github.com/wms-platform/labor-service/internal/domain"
)

type stubWorkerRepo struct {
	SaveFn         func(ctx context.Context, worker *domain.Worker) error
	FindByIDFn     func(ctx context.Context, workerID string) (*domain.Worker, error)
	FindByStatusFn func(ctx context.Context, status domain.WorkerStatus) ([]*domain.Worker, error)
	FindByZoneFn   func(ctx context.Context, zone string) ([]*domain.Worker, error)
	FindAllFn      func(ctx context.Context, limit, offset int) ([]*domain.Worker, error)
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
	return nil, nil
}

func (s *stubWorkerRepo) FindAll(ctx context.Context, limit, offset int) ([]*domain.Worker, error) {
	if s.FindAllFn != nil {
		return s.FindAllFn(ctx, limit, offset)
	}
	return nil, nil
}

func (s *stubWorkerRepo) Delete(ctx context.Context, workerID string) error {
	return nil
}

func newTestService(repo domain.WorkerRepository) (*application.LaborApplicationService, *logging.Logger) {
	logger := logging.New(logging.DefaultConfig("test"))
	return application.NewLaborApplicationService(repo, nil, nil, logger), logger
}

func newWorkerWithShift(t *testing.T) *domain.Worker {
	t.Helper()
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	if err := worker.StartShift("shift-1", "morning", "A"); err != nil {
		t.Fatalf("start shift: %v", err)
	}
	return worker
}

func requestJSON(t *testing.T, router *gin.Engine, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(raw)
	} else {
		body = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_ENV_KEY", "value")
	if got := getEnv("TEST_ENV_KEY", "default"); got != "value" {
		t.Fatalf("expected env value, got %q", got)
	}
	if got := getEnv("MISSING_KEY", "default"); got != "default" {
		t.Fatalf("expected default, got %q", got)
	}
}

func TestLoadConfig(t *testing.T) {
	t.Setenv("SERVER_ADDR", ":9000")
	t.Setenv("MONGODB_URI", "mongodb://example:27017")
	t.Setenv("MONGODB_DATABASE", "labor_test")
	t.Setenv("KAFKA_BROKERS", "broker-1:9092")

	cfg := loadConfig()
	if cfg.ServerAddr != ":9000" {
		t.Fatalf("unexpected server addr: %q", cfg.ServerAddr)
	}
	if cfg.MongoDB.URI != "mongodb://example:27017" || cfg.MongoDB.Database != "labor_test" {
		t.Fatalf("unexpected mongo config: %#v", cfg.MongoDB)
	}
	if len(cfg.Kafka.Brokers) != 1 || cfg.Kafka.Brokers[0] != "broker-1:9092" {
		t.Fatalf("unexpected kafka brokers: %#v", cfg.Kafka.Brokers)
	}
}

func TestCreateWorkerHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubWorkerRepo{}
	service, logger := newTestService(repo)
	router := gin.New()
	router.POST("/workers", createWorkerHandler(service, logger))

	resp := requestJSON(t, router, http.MethodPost, "/workers", map[string]any{
		"workerId":   "worker-1",
		"employeeId": "emp-1",
		"name":       "Ada",
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
}

func TestCreateWorkerHandler_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubWorkerRepo{}
	service, logger := newTestService(repo)
	router := gin.New()
	router.POST("/workers", createWorkerHandler(service, logger))

	resp := requestJSON(t, router, http.MethodPost, "/workers", map[string]any{})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestCreateWorkerHandler_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubWorkerRepo{
		SaveFn: func(_ context.Context, _ *domain.Worker) error {
			return errors.New("save failed")
		},
	}
	service, logger := newTestService(repo)
	router := gin.New()
	router.POST("/workers", createWorkerHandler(service, logger))

	resp := requestJSON(t, router, http.MethodPost, "/workers", map[string]any{
		"workerId":   "worker-1",
		"employeeId": "emp-1",
		"name":       "Ada",
	})
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestGetWorkerHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return nil, nil
		},
	}
	service, logger := newTestService(repo)
	router := gin.New()
	router.GET("/workers/:workerId", getWorkerHandler(service, logger))

	resp := requestJSON(t, router, http.MethodGet, "/workers/worker-1", nil)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
}

func TestShiftHandlers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service, logger := newTestService(repo)
	router := gin.New()
	router.POST("/workers/:workerId/shift/start", startShiftHandler(service, logger))
	router.POST("/workers/:workerId/shift/end", endShiftHandler(service, logger))

	startResp := requestJSON(t, router, http.MethodPost, "/workers/worker-1/shift/start", map[string]any{
		"shiftId":   "shift-1",
		"shiftType": "morning",
		"zone":      "A",
	})
	if startResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", startResp.Code)
	}

	endResp := requestJSON(t, router, http.MethodPost, "/workers/worker-1/shift/end", map[string]any{})
	if endResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", endResp.Code)
	}
}

func TestBreakHandlers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	worker := newWorkerWithShift(t)
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service, logger := newTestService(repo)
	router := gin.New()
	router.POST("/workers/:workerId/break/start", startBreakHandler(service, logger))
	router.POST("/workers/:workerId/break/end", endBreakHandler(service, logger))

	startResp := requestJSON(t, router, http.MethodPost, "/workers/worker-1/break/start", map[string]any{
		"breakType": "lunch",
	})
	if startResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", startResp.Code)
	}

	endResp := requestJSON(t, router, http.MethodPost, "/workers/worker-1/break/end", map[string]any{})
	if endResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", endResp.Code)
	}
}

func TestTaskHandlers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	worker := newWorkerWithShift(t)
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service, logger := newTestService(repo)
	router := gin.New()
	router.POST("/workers/:workerId/task/assign", assignTaskHandler(service, logger))
	router.POST("/workers/:workerId/task/start", startTaskHandler(service, logger))
	router.POST("/workers/:workerId/task/complete", completeTaskHandler(service, logger))

	assignResp := requestJSON(t, router, http.MethodPost, "/workers/worker-1/task/assign", map[string]any{
		"taskId":   "task-1",
		"taskType": "picking",
		"priority": 1,
	})
	if assignResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", assignResp.Code)
	}

	startResp := requestJSON(t, router, http.MethodPost, "/workers/worker-1/task/start", map[string]any{})
	if startResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", startResp.Code)
	}

	completeResp := requestJSON(t, router, http.MethodPost, "/workers/worker-1/task/complete", map[string]any{
		"itemsProcessed": 5,
	})
	if completeResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", completeResp.Code)
	}
}

func TestAddSkillHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	worker := domain.NewWorker("worker-1", "emp-1", "Ada")
	repo := &stubWorkerRepo{
		FindByIDFn: func(_ context.Context, _ string) (*domain.Worker, error) {
			return worker, nil
		},
	}
	service, logger := newTestService(repo)
	router := gin.New()
	router.POST("/workers/:workerId/skills", addSkillHandler(service, logger))

	resp := requestJSON(t, router, http.MethodPost, "/workers/worker-1/skills", map[string]any{
		"taskType":  "packing",
		"level":     3,
		"certified": true,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestQueryHandlers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubWorkerRepo{
		FindByStatusFn: func(_ context.Context, _ domain.WorkerStatus) ([]*domain.Worker, error) {
			worker := domain.NewWorker("worker-1", "emp-1", "Ada")
			worker.Status = domain.WorkerStatusAvailable
			worker.CurrentZone = "A"
			return []*domain.Worker{worker}, nil
		},
		FindByZoneFn: func(_ context.Context, _ string) ([]*domain.Worker, error) {
			worker := domain.NewWorker("worker-1", "emp-1", "Ada")
			worker.CurrentZone = "A"
			return []*domain.Worker{worker}, nil
		},
		FindAllFn: func(_ context.Context, _ int, _ int) ([]*domain.Worker, error) {
			worker := domain.NewWorker("worker-1", "emp-1", "Ada")
			return []*domain.Worker{worker}, nil
		},
	}
	service, logger := newTestService(repo)
	router := gin.New()
	router.GET("/workers/status/:status", getByStatusHandler(service, logger))
	router.GET("/workers/zone/:zone", getByZoneHandler(service, logger))
	router.GET("/workers/available", getAvailableHandler(service, logger))
	router.GET("/workers", listWorkersHandler(service, logger))

	statusResp := requestJSON(t, router, http.MethodGet, "/workers/status/available", nil)
	if statusResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", statusResp.Code)
	}

	zoneResp := requestJSON(t, router, http.MethodGet, "/workers/zone/A", nil)
	if zoneResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", zoneResp.Code)
	}

	availableResp := requestJSON(t, router, http.MethodGet, "/workers/available?zone=A", nil)
	if availableResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", availableResp.Code)
	}

	listResp := requestJSON(t, router, http.MethodGet, "/workers?limit=10&offset=0", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.Code)
	}
}
