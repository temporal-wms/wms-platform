package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/inventory-service/internal/infrastructure/projections"
	"github.com/wms-platform/shared/pkg/logging"
)

type fakeProjectionRepo struct {
	lastFilter projections.InventoryListFilter
	lastPage   projections.Pagination
	result     *projections.PagedResult[projections.InventoryListProjection]
	findBySKU  *projections.InventoryListProjection
	count      int64
	err        error
}

func (f *fakeProjectionRepo) Upsert(ctx context.Context, projection *projections.InventoryListProjection) error {
	return f.err
}

func (f *fakeProjectionRepo) FindBySKU(ctx context.Context, sku string) (*projections.InventoryListProjection, error) {
	return f.findBySKU, f.err
}

func (f *fakeProjectionRepo) FindWithFilter(ctx context.Context, filter projections.InventoryListFilter, page projections.Pagination) (*projections.PagedResult[projections.InventoryListProjection], error) {
	f.lastFilter = filter
	f.lastPage = page
	return f.result, f.err
}

func (f *fakeProjectionRepo) UpdateFields(ctx context.Context, sku string, updates map[string]interface{}) error {
	return f.err
}

func (f *fakeProjectionRepo) Delete(ctx context.Context, sku string) error {
	return f.err
}

func (f *fakeProjectionRepo) Count(ctx context.Context, filter projections.InventoryListFilter) (int64, error) {
	f.lastFilter = filter
	return f.count, f.err
}

func TestInventoryQueryService_ListInventoryDefaults(t *testing.T) {
	repo := &fakeProjectionRepo{
		result: &projections.PagedResult[projections.InventoryListProjection]{
			Items: []projections.InventoryListProjection{
				{SKU: "SKU-1", ProductName: "Widget"},
			},
			Total:   1,
			Limit:   50,
			Offset:  0,
			HasMore: false,
		},
	}
	logger := logging.New(logging.DefaultConfig("test"))
	svc := NewInventoryQueryService(repo, logger)

	sku := "SKU-1"
	product := "Widget"
	zone := "ZONE-A"
	isLow := true
	query := ListInventoryQuery{
		SKU:         &sku,
		ProductName: &product,
		SearchTerm:  "wid",
		IsLowStock:  &isLow,
		Zone:        &zone,
	}

	result, err := svc.ListInventory(context.Background(), query)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 50, repo.lastPage.Limit)
	assert.Equal(t, "updatedAt", repo.lastPage.SortBy)
	assert.Equal(t, "desc", repo.lastPage.SortOrder)
	assert.Equal(t, sku, *repo.lastFilter.SKU)
	assert.Equal(t, product, *repo.lastFilter.ProductName)
	assert.Equal(t, "wid", repo.lastFilter.SearchTerm)
	assert.Equal(t, zone, *repo.lastFilter.Zone)
}

func TestInventoryQueryService_GetInventorySummary(t *testing.T) {
	repo := &fakeProjectionRepo{
		findBySKU: &projections.InventoryListProjection{
			SKU:           "SKU-1",
			ProductName:   "Widget",
			TotalQuantity: 10,
		},
	}
	logger := logging.New(logging.DefaultConfig("test"))
	svc := NewInventoryQueryService(repo, logger)

	dto, err := svc.GetInventorySummary(context.Background(), "SKU-1")
	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, "SKU-1", dto.SKU)

	repo.findBySKU = nil
	dto, err = svc.GetInventorySummary(context.Background(), "SKU-2")
	require.NoError(t, err)
	assert.Nil(t, dto)
}

func TestInventoryQueryService_GetLowStockAndOutOfStock(t *testing.T) {
	repo := &fakeProjectionRepo{
		result: &projections.PagedResult[projections.InventoryListProjection]{
			Items: []projections.InventoryListProjection{},
			Total: 0,
		},
	}
	logger := logging.New(logging.DefaultConfig("test"))
	svc := NewInventoryQueryService(repo, logger)

	_, err := svc.GetLowStockItems(context.Background(), 25, 10)
	require.NoError(t, err)
	require.NotNil(t, repo.lastFilter.IsLowStock)
	assert.True(t, *repo.lastFilter.IsLowStock)
	assert.Equal(t, "availableQuantity", repo.lastPage.SortBy)
	assert.Equal(t, "asc", repo.lastPage.SortOrder)

	_, err = svc.GetOutOfStockItems(context.Background(), 5, 0)
	require.NoError(t, err)
	require.NotNil(t, repo.lastFilter.IsOutOfStock)
	assert.True(t, *repo.lastFilter.IsOutOfStock)
}

func TestInventoryQueryService_CountInventoryByStatus(t *testing.T) {
	repo := &fakeProjectionRepo{count: 12}
	logger := logging.New(logging.DefaultConfig("test"))
	svc := NewInventoryQueryService(repo, logger)

	isLow := true
	count, err := svc.CountInventoryByStatus(context.Background(), &isLow, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(12), count)
	assert.NotNil(t, repo.lastFilter.IsLowStock)
}
