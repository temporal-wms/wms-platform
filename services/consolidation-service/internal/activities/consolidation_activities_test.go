package activities

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/consolidation-service/internal/domain"
	"go.temporal.io/sdk/testsuite"
)

type mockRepo struct {
	saveFn     func(context.Context, *domain.ConsolidationUnit) error
	findByIDFn func(context.Context, string) (*domain.ConsolidationUnit, error)
}

func (m *mockRepo) Save(ctx context.Context, unit *domain.ConsolidationUnit) error {
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

func (m *mockRepo) FindByOrderID(context.Context, string) (*domain.ConsolidationUnit, error) {
	return nil, nil
}

func (m *mockRepo) FindByWaveID(context.Context, string) ([]*domain.ConsolidationUnit, error) {
	return nil, nil
}

func (m *mockRepo) FindByStatus(context.Context, domain.ConsolidationStatus) ([]*domain.ConsolidationUnit, error) {
	return nil, nil
}

func (m *mockRepo) FindByStation(context.Context, string) ([]*domain.ConsolidationUnit, error) {
	return nil, nil
}

func (m *mockRepo) FindPending(context.Context, int) ([]*domain.ConsolidationUnit, error) {
	return nil, nil
}

func (m *mockRepo) Delete(context.Context, string) error {
	return nil
}

func testSlog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCreateConsolidationUnitActivity(t *testing.T) {
	var saved *domain.ConsolidationUnit
	repo := &mockRepo{
		saveFn: func(ctx context.Context, unit *domain.ConsolidationUnit) error {
			saved = unit
			return nil
		},
	}
	acts := NewConsolidationActivities(repo, testSlog())

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(acts.CreateConsolidationUnit)

	input := map[string]interface{}{
		"orderId": "ORD-1",
		"pickedItems": []interface{}{
			map[string]interface{}{"sku": "SKU-1", "quantity": 2.0, "toteId": "TOTE-1"},
			map[string]interface{}{"sku": "SKU-2", "quantity": 1.0, "toteId": "TOTE-2"},
		},
	}

	blob, err := env.ExecuteActivity(acts.CreateConsolidationUnit, input)
	require.NoError(t, err)
	var consolidationID string
	require.NoError(t, blob.Get(&consolidationID))
	assert.True(t, strings.HasPrefix(consolidationID, "CONS-"))
	require.NotNil(t, saved)
	assert.Equal(t, "ORD-1", saved.OrderID)
	assert.Len(t, saved.ExpectedItems, 2)
}

func TestAssignStationActivityNotFound(t *testing.T) {
	repo := &mockRepo{}
	acts := NewConsolidationActivities(repo, testSlog())

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(acts.AssignStation)

	input := map[string]interface{}{
		"consolidationId": "CONS-404",
		"station":         "ST-1",
		"workerId":        "WK-1",
		"destinationBin":  "BIN-1",
	}

	_, err := env.ExecuteActivity(acts.AssignStation, input)
	assert.Error(t, err)
}

func TestConsolidateItemActivity(t *testing.T) {
	unit, err := domain.NewConsolidationUnit("CONS-2", "ORD-2", "WAVE-2", domain.StrategyOrderBased, []domain.ExpectedItem{
		{SKU: "SKU-1", Quantity: 2, SourceToteID: "TOTE-1"},
	})
	require.NoError(t, err)
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
			return unit, nil
		},
	}
	acts := NewConsolidationActivities(repo, testSlog())

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(acts.ConsolidateItem)

	_, err = env.ExecuteActivity(acts.ConsolidateItem, "CONS-2", "SKU-1", "TOTE-1", "WK-1", 2)
	require.NoError(t, err)
	assert.Equal(t, 2, unit.TotalConsolidated)
}

func TestCompleteConsolidationActivity(t *testing.T) {
	unit, err := domain.NewConsolidationUnit("CONS-3", "ORD-3", "WAVE-3", domain.StrategyOrderBased, []domain.ExpectedItem{
		{SKU: "SKU-1", Quantity: 1, SourceToteID: "TOTE-1"},
	})
	require.NoError(t, err)
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
			return unit, nil
		},
	}
	acts := NewConsolidationActivities(repo, testSlog())

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(acts.CompleteConsolidation)

	_, err = env.ExecuteActivity(acts.CompleteConsolidation, "CONS-3")
	require.NoError(t, err)
	assert.Equal(t, domain.ConsolidationStatusCompleted, unit.Status)
}

func TestMarkShortActivity(t *testing.T) {
	unit, err := domain.NewConsolidationUnit("CONS-4", "ORD-4", "WAVE-4", domain.StrategyOrderBased, []domain.ExpectedItem{
		{SKU: "SKU-1", Quantity: 2, SourceToteID: "TOTE-1"},
	})
	require.NoError(t, err)
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, consolidationID string) (*domain.ConsolidationUnit, error) {
			return unit, nil
		},
	}
	acts := NewConsolidationActivities(repo, testSlog())

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(acts.MarkShort)

	_, err = env.ExecuteActivity(acts.MarkShort, "CONS-4", "SKU-1", "TOTE-1", "damaged", 1)
	require.NoError(t, err)
	assert.Equal(t, "short", unit.ExpectedItems[0].Status)
}
