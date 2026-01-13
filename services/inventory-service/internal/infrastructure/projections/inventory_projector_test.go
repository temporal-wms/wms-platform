package projections

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/inventory-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
)

type projectorProjectionRepo struct {
	bySKU      map[string]*InventoryListProjection
	updates    map[string]map[string]interface{}
	upserted   *InventoryListProjection
	updateCalls int
}

func (p *projectorProjectionRepo) Upsert(ctx context.Context, projection *InventoryListProjection) error {
	if p.bySKU == nil {
		p.bySKU = make(map[string]*InventoryListProjection)
	}
	p.bySKU[projection.SKU] = projection
	p.upserted = projection
	return nil
}

func (p *projectorProjectionRepo) FindBySKU(ctx context.Context, sku string) (*InventoryListProjection, error) {
	if proj, ok := p.bySKU[sku]; ok {
		return proj, nil
	}
	return nil, nil
}

func (p *projectorProjectionRepo) FindWithFilter(ctx context.Context, filter InventoryListFilter, page Pagination) (*PagedResult[InventoryListProjection], error) {
	return nil, nil
}

func (p *projectorProjectionRepo) UpdateFields(ctx context.Context, sku string, updates map[string]interface{}) error {
	if p.updates == nil {
		p.updates = make(map[string]map[string]interface{})
	}
	p.updateCalls++
	p.updates[sku] = updates
	return nil
}

func (p *projectorProjectionRepo) Delete(ctx context.Context, sku string) error { return nil }

func (p *projectorProjectionRepo) Count(ctx context.Context, filter InventoryListFilter) (int64, error) {
	return 0, nil
}

type projectorInventoryRepo struct {
	item *domain.InventoryItem
}

func (p *projectorInventoryRepo) Save(ctx context.Context, item *domain.InventoryItem) error {
	p.item = item
	return nil
}

func (p *projectorInventoryRepo) FindBySKU(ctx context.Context, sku string) (*domain.InventoryItem, error) {
	return p.item, nil
}

func (p *projectorInventoryRepo) FindByLocation(ctx context.Context, locationID string) ([]*domain.InventoryItem, error) {
	return nil, nil
}

func (p *projectorInventoryRepo) FindByZone(ctx context.Context, zone string) ([]*domain.InventoryItem, error) {
	return nil, nil
}

func (p *projectorInventoryRepo) FindByOrderID(ctx context.Context, orderID string) ([]*domain.InventoryItem, error) {
	return nil, nil
}

func (p *projectorInventoryRepo) FindLowStock(ctx context.Context) ([]*domain.InventoryItem, error) {
	return nil, nil
}

func (p *projectorInventoryRepo) FindAll(ctx context.Context, limit, offset int) ([]*domain.InventoryItem, error) {
	return nil, nil
}

func (p *projectorInventoryRepo) Delete(ctx context.Context, sku string) error {
	return nil
}

func TestInventoryProjector_OnInventoryReceived(t *testing.T) {
	item := domain.NewInventoryItem("SKU-1", "Widget", 5, 10)
	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 5, "PO-1", "user1"))

	projRepo := &projectorProjectionRepo{}
	invRepo := &projectorInventoryRepo{item: item}
	logger := logging.New(logging.DefaultConfig("test"))
	projector := NewInventoryProjector(projRepo, invRepo, logger)

	event := &domain.InventoryReceivedEvent{SKU: "SKU-1", ReceivedAt: time.Now()}
	require.NoError(t, projector.OnInventoryReceived(context.Background(), event))
	require.NotNil(t, projRepo.upserted)
	assert.Equal(t, "SKU-1", projRepo.upserted.SKU)

	projRepo.bySKU["SKU-1"] = projRepo.upserted
	require.NoError(t, projector.OnInventoryReceived(context.Background(), event))
	assert.GreaterOrEqual(t, projRepo.updateCalls, 1)
}

func TestInventoryProjector_OnInventoryAdjustedAndLowStock(t *testing.T) {
	item := domain.NewInventoryItem("SKU-2", "Widget", 5, 10)
	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 3, "PO-1", "user1"))

	projRepo := &projectorProjectionRepo{bySKU: map[string]*InventoryListProjection{"SKU-2": {SKU: "SKU-2"}}}
	invRepo := &projectorInventoryRepo{item: item}
	logger := logging.New(logging.DefaultConfig("test"))
	projector := NewInventoryProjector(projRepo, invRepo, logger)

	require.NoError(t, projector.OnInventoryAdjusted(context.Background(), &domain.InventoryAdjustedEvent{SKU: "SKU-2", AdjustedAt: time.Now()}))
	require.Contains(t, projRepo.updates["SKU-2"], "availableQuantity")

	require.NoError(t, projector.OnLowStockAlert(context.Background(), &domain.LowStockAlertEvent{SKU: "SKU-2", CurrentQuantity: 1}))
	require.Contains(t, projRepo.updates["SKU-2"], "isLowStock")
}

func TestInventoryProjector_OnInventoryReservedAndPicked(t *testing.T) {
	item := domain.NewInventoryItem("SKU-3", "Widget", 5, 10)
	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 10, "PO-1", "user1"))
	require.NoError(t, item.Reserve("ORD-1", "LOC-1", 2))
	item.Reservations = append(item.Reservations, domain.Reservation{
		ReservationID: "RES-2",
		OrderID:       "ORD-2",
		LocationID:    "LOC-1",
		Status:        "cancelled",
	})

	projRepo := &projectorProjectionRepo{bySKU: map[string]*InventoryListProjection{"SKU-3": {SKU: "SKU-3"}}}
	invRepo := &projectorInventoryRepo{item: item}
	logger := logging.New(logging.DefaultConfig("test"))
	projector := NewInventoryProjector(projRepo, invRepo, logger)

	require.NoError(t, projector.OnInventoryReserved(context.Background(), "SKU-3", "ORD-1"))
	require.Contains(t, projRepo.updates["SKU-3"], "activeReservations")

	require.NoError(t, projector.OnInventoryPicked(context.Background(), "SKU-3"))
	require.Contains(t, projRepo.updates["SKU-3"], "lastPicked")
}

func TestInventoryProjector_OnShortageAndDiscrepancy(t *testing.T) {
	item := domain.NewInventoryItem("SKU-4", "Widget", 5, 10)
	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 10, "PO-1", "user1"))

	projRepo := &projectorProjectionRepo{bySKU: map[string]*InventoryListProjection{"SKU-4": {SKU: "SKU-4"}}}
	invRepo := &projectorInventoryRepo{item: item}
	logger := logging.New(logging.DefaultConfig("test"))
	projector := NewInventoryProjector(projRepo, invRepo, logger)

	require.NoError(t, projector.OnStockShortage(context.Background(), &domain.StockShortageEvent{SKU: "SKU-4"}))
	require.Contains(t, projRepo.updates["SKU-4"], "lastShortage")

	require.NoError(t, projector.OnInventoryDiscrepancy(context.Background(), &domain.InventoryDiscrepancyEvent{SKU: "SKU-4", DiscrepancyType: "shortage"}))
	require.Contains(t, projRepo.updates["SKU-4"], "lastDiscrepancy")
}

func TestInventoryProjector_Helpers(t *testing.T) {
	item := domain.NewInventoryItem("SKU-5", "Widget", 5, 10)
	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 5, "PO-1", "user1"))
	require.NoError(t, item.ReceiveStock("LOC-2", "ZONE-B", 8, "PO-2", "user1"))

	projector := NewInventoryProjector(&projectorProjectionRepo{}, &projectorInventoryRepo{item: item}, logging.New(logging.DefaultConfig("test")))
	available := projector.extractAvailableLocations(item)
	assert.Len(t, available, 2)

	primary := projector.findPrimaryLocation(item)
	assert.Equal(t, "LOC-2", primary)

	projection := projector.buildProjectionFromAggregate(item)
	assert.Equal(t, "SKU-5", projection.SKU)
}
