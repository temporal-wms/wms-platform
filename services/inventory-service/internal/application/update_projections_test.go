package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/inventory-service/internal/domain"
	"github.com/wms-platform/inventory-service/internal/infrastructure/projections"
	"github.com/wms-platform/shared/pkg/logging"
)

type updateProjectionRepo struct {
	updates int
}

func (u *updateProjectionRepo) Upsert(ctx context.Context, projection *projections.InventoryListProjection) error {
	return nil
}

func (u *updateProjectionRepo) FindBySKU(ctx context.Context, sku string) (*projections.InventoryListProjection, error) {
	return &projections.InventoryListProjection{SKU: sku}, nil
}

func (u *updateProjectionRepo) FindWithFilter(ctx context.Context, filter projections.InventoryListFilter, page projections.Pagination) (*projections.PagedResult[projections.InventoryListProjection], error) {
	return nil, nil
}

func (u *updateProjectionRepo) UpdateFields(ctx context.Context, sku string, updates map[string]interface{}) error {
	u.updates++
	return nil
}

func (u *updateProjectionRepo) Delete(ctx context.Context, sku string) error {
	return nil
}

func (u *updateProjectionRepo) Count(ctx context.Context, filter projections.InventoryListFilter) (int64, error) {
	return 0, nil
}

type updateInventoryRepo struct {
	item *domain.InventoryItem
}

func (u *updateInventoryRepo) Save(ctx context.Context, item *domain.InventoryItem) error { return nil }
func (u *updateInventoryRepo) FindBySKU(ctx context.Context, sku string) (*domain.InventoryItem, error) {
	return u.item, nil
}
func (u *updateInventoryRepo) FindByLocation(ctx context.Context, locationID string) ([]*domain.InventoryItem, error) {
	return nil, nil
}
func (u *updateInventoryRepo) FindByZone(ctx context.Context, zone string) ([]*domain.InventoryItem, error) {
	return nil, nil
}
func (u *updateInventoryRepo) FindByOrderID(ctx context.Context, orderID string) ([]*domain.InventoryItem, error) {
	return nil, nil
}
func (u *updateInventoryRepo) FindLowStock(ctx context.Context) ([]*domain.InventoryItem, error) {
	return nil, nil
}
func (u *updateInventoryRepo) FindAll(ctx context.Context, limit, offset int) ([]*domain.InventoryItem, error) {
	return nil, nil
}
func (u *updateInventoryRepo) Delete(ctx context.Context, sku string) error {
	return nil
}

func TestInventoryApplicationService_UpdateProjections(t *testing.T) {
	item := domain.NewInventoryItem("SKU-1", "Widget", 5, 10)
	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 5, "PO-1", "user1"))

	projRepo := &updateProjectionRepo{}
	invRepo := &updateInventoryRepo{item: item}
	logger := logging.New(logging.DefaultConfig("test"))
	projector := projections.NewInventoryProjector(projRepo, invRepo, logger)

	svc := NewInventoryApplicationService(invRepo, nil, nil, projector, logger)
	events := []domain.DomainEvent{
		&domain.InventoryReceivedEvent{SKU: "SKU-1", ReceivedAt: time.Now()},
		&domain.InventoryAdjustedEvent{SKU: "SKU-1", AdjustedAt: time.Now()},
		&domain.LowStockAlertEvent{SKU: "SKU-1", AlertedAt: time.Now()},
		&domain.StockShortageEvent{SKU: "SKU-1", OccurredAt_: time.Now()},
		&domain.InventoryDiscrepancyEvent{SKU: "SKU-1", DetectedAt: time.Now()},
	}

	svc.updateProjections(context.Background(), "SKU-1", events)
	assert.GreaterOrEqual(t, projRepo.updates, 4)
}
