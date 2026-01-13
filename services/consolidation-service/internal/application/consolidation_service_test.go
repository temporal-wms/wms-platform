package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/consolidation-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
)

type mockRepo struct {
	saveFn         func(context.Context, *domain.ConsolidationUnit) error
	findByIDFn     func(context.Context, string) (*domain.ConsolidationUnit, error)
	findByOrderFn  func(context.Context, string) (*domain.ConsolidationUnit, error)
	findByWaveFn   func(context.Context, string) ([]*domain.ConsolidationUnit, error)
	findByStatusFn func(context.Context, domain.ConsolidationStatus) ([]*domain.ConsolidationUnit, error)
	findByStation  func(context.Context, string) ([]*domain.ConsolidationUnit, error)
	findPendingFn  func(context.Context, int) ([]*domain.ConsolidationUnit, error)
	deleteFn       func(context.Context, string) error

	lastSaved *domain.ConsolidationUnit
}

func (m *mockRepo) Save(ctx context.Context, unit *domain.ConsolidationUnit) error {
	m.lastSaved = unit
	if m.saveFn != nil {
		return m.saveFn(ctx, unit)
	}
	return nil
}

func (m *mockRepo) FindByID(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, consolidationID)
	}
	return nil, nil
}

func (m *mockRepo) FindByOrderID(ctx context.Context, orderID string) (*domain.ConsolidationUnit, error) {
	if m.findByOrderFn != nil {
		return m.findByOrderFn(ctx, orderID)
	}
	return nil, nil
}

func (m *mockRepo) FindByWaveID(ctx context.Context, waveID string) ([]*domain.ConsolidationUnit, error) {
	if m.findByWaveFn != nil {
		return m.findByWaveFn(ctx, waveID)
	}
	return nil, nil
}

func (m *mockRepo) FindByStatus(ctx context.Context, status domain.ConsolidationStatus) ([]*domain.ConsolidationUnit, error) {
	if m.findByStatusFn != nil {
		return m.findByStatusFn(ctx, status)
	}
	return nil, nil
}

func (m *mockRepo) FindByStation(ctx context.Context, station string) ([]*domain.ConsolidationUnit, error) {
	if m.findByStation != nil {
		return m.findByStation(ctx, station)
	}
	return nil, nil
}

func (m *mockRepo) FindPending(ctx context.Context, limit int) ([]*domain.ConsolidationUnit, error) {
	if m.findPendingFn != nil {
		return m.findPendingFn(ctx, limit)
	}
	return nil, nil
}

func (m *mockRepo) Delete(ctx context.Context, consolidationID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, consolidationID)
	}
	return nil
}

func testLogger() *logging.Logger {
	cfg := logging.DefaultConfig("consolidation-service-test")
	cfg.Level = logging.LogLevel("error")
	return logging.New(cfg)
}

func createUnit(t *testing.T, consolidationID string) *domain.ConsolidationUnit {
	t.Helper()
	unit, err := domain.NewConsolidationUnit(
		consolidationID,
		"ORD-1",
		"WAVE-1",
		domain.StrategyOrderBased,
		[]domain.ExpectedItem{
			{SKU: "SKU-1", Quantity: 2, SourceToteID: "TOTE-1"},
		},
	)
	require.NoError(t, err)
	return unit
}

func TestCreateConsolidation(t *testing.T) {
	repo := &mockRepo{}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	cmd := CreateConsolidationCommand{
		ConsolidationID: "CONS-1",
		OrderID:         "ORD-1",
		WaveID:          "WAVE-1",
		Items: []domain.ExpectedItem{
			{SKU: "SKU-1", Quantity: 2, SourceToteID: "TOTE-1"},
		},
	}

	dto, err := service.CreateConsolidation(context.Background(), cmd)
	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, "CONS-1", dto.ConsolidationID)
	assert.Equal(t, string(domain.StrategyOrderBased), dto.Strategy)
	assert.NotNil(t, repo.lastSaved)
}

func TestCreateConsolidationValidationError(t *testing.T) {
	service := NewConsolidationApplicationService(&mockRepo{}, nil, nil, testLogger())

	_, err := service.CreateConsolidation(context.Background(), CreateConsolidationCommand{
		ConsolidationID: "CONS-2",
		OrderID:         "ORD-2",
		WaveID:          "WAVE-2",
		Items:           []domain.ExpectedItem{},
	})
	assert.Error(t, err)
}

func TestCreateConsolidationSaveError(t *testing.T) {
	repo := &mockRepo{
		saveFn: func(ctx context.Context, unit *domain.ConsolidationUnit) error {
			return errors.New("save failed")
		},
	}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	_, err := service.CreateConsolidation(context.Background(), CreateConsolidationCommand{
		ConsolidationID: "CONS-3",
		OrderID:         "ORD-3",
		WaveID:          "WAVE-3",
		Items: []domain.ExpectedItem{
			{SKU: "SKU-1", Quantity: 1, SourceToteID: "TOTE-1"},
		},
	})
	assert.Error(t, err)
}

func TestGetConsolidation(t *testing.T) {
	unit := createUnit(t, "CONS-4")
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
			return unit, nil
		},
	}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	dto, err := service.GetConsolidation(context.Background(), GetConsolidationQuery{ConsolidationID: "CONS-4"})
	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, "CONS-4", dto.ConsolidationID)
}

func TestGetConsolidationNotFound(t *testing.T) {
	repo := &mockRepo{}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	_, err := service.GetConsolidation(context.Background(), GetConsolidationQuery{ConsolidationID: "CONS-404"})
	assert.Error(t, err)
}

func TestGetConsolidationRepoError(t *testing.T) {
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
			return nil, errors.New("db error")
		},
	}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	_, err := service.GetConsolidation(context.Background(), GetConsolidationQuery{ConsolidationID: "CONS-500"})
	assert.Error(t, err)
}

func TestAssignStation(t *testing.T) {
	unit := createUnit(t, "CONS-5")
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
			return unit, nil
		},
	}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	dto, err := service.AssignStation(context.Background(), AssignStationCommand{
		ConsolidationID: "CONS-5",
		Station:         "ST-1",
		WorkerID:        "WK-1",
		DestinationBin:  "BIN-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "ST-1", dto.Station)
	assert.NotNil(t, repo.lastSaved)
}

func TestAssignStationNotFound(t *testing.T) {
	service := NewConsolidationApplicationService(&mockRepo{}, nil, nil, testLogger())

	_, err := service.AssignStation(context.Background(), AssignStationCommand{ConsolidationID: "CONS-6"})
	assert.Error(t, err)
}

func TestAssignStationValidationError(t *testing.T) {
	unit := createUnit(t, "CONS-7")
	unit.Status = domain.ConsolidationStatusCompleted
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
			return unit, nil
		},
	}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	_, err := service.AssignStation(context.Background(), AssignStationCommand{ConsolidationID: "CONS-7", Station: "ST-2"})
	assert.Error(t, err)
}

func TestConsolidateItem(t *testing.T) {
	unit := createUnit(t, "CONS-8")
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
			return unit, nil
		},
	}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	dto, err := service.ConsolidateItem(context.Background(), ConsolidateItemCommand{
		ConsolidationID: "CONS-8",
		SKU:             "SKU-1",
		Quantity:        2,
		SourceToteID:    "TOTE-1",
		VerifiedBy:      "WK-2",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.ConsolidationStatusCompleted, unit.Status)
	assert.Equal(t, "CONS-8", dto.ConsolidationID)
}

func TestConsolidateItemNotFound(t *testing.T) {
	service := NewConsolidationApplicationService(&mockRepo{}, nil, nil, testLogger())

	_, err := service.ConsolidateItem(context.Background(), ConsolidateItemCommand{ConsolidationID: "CONS-9"})
	assert.Error(t, err)
}

func TestCompleteConsolidation(t *testing.T) {
	unit := createUnit(t, "CONS-10")
	unit.Start()
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
			return unit, nil
		},
	}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	dto, err := service.CompleteConsolidation(context.Background(), CompleteConsolidationCommand{ConsolidationID: "CONS-10"})
	require.NoError(t, err)
	assert.Equal(t, domain.ConsolidationStatusCompleted, unit.Status)
	assert.True(t, dto.ReadyForPacking)
}

func TestCompleteConsolidationValidationError(t *testing.T) {
	unit := createUnit(t, "CONS-11")
	unit.Status = domain.ConsolidationStatusCompleted
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
			return unit, nil
		},
	}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	_, err := service.CompleteConsolidation(context.Background(), CompleteConsolidationCommand{ConsolidationID: "CONS-11"})
	assert.Error(t, err)
}

func TestGetByOrderAndWaveAndStation(t *testing.T) {
	unit := createUnit(t, "CONS-12")
	repo := &mockRepo{
		findByOrderFn: func(ctx context.Context, orderID string) (*domain.ConsolidationUnit, error) {
			return unit, nil
		},
		findByWaveFn: func(ctx context.Context, waveID string) ([]*domain.ConsolidationUnit, error) {
			return []*domain.ConsolidationUnit{unit}, nil
		},
		findByStation: func(ctx context.Context, station string) ([]*domain.ConsolidationUnit, error) {
			return []*domain.ConsolidationUnit{unit}, nil
		},
	}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	byOrder, err := service.GetByOrder(context.Background(), GetByOrderQuery{OrderID: "ORD-1"})
	require.NoError(t, err)
	assert.Equal(t, "CONS-12", byOrder.ConsolidationID)

	byWave, err := service.GetByWave(context.Background(), GetByWaveQuery{WaveID: "WAVE-1"})
	require.NoError(t, err)
	assert.Len(t, byWave, 1)

	byStation, err := service.GetByStation(context.Background(), GetByStationQuery{Station: "ST-1"})
	require.NoError(t, err)
	assert.Len(t, byStation, 1)
}

func TestGetPendingDefaultLimit(t *testing.T) {
	repo := &mockRepo{
		findPendingFn: func(ctx context.Context, limit int) ([]*domain.ConsolidationUnit, error) {
			assert.Equal(t, 50, limit)
			return []*domain.ConsolidationUnit{createUnit(t, "CONS-13")}, nil
		},
	}
	service := NewConsolidationApplicationService(repo, nil, nil, testLogger())

	result, err := service.GetPending(context.Background(), GetPendingQuery{Limit: 0})
	require.NoError(t, err)
	assert.Len(t, result, 1)
}
