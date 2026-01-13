package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/inventory-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
)

type fakeInventoryRepo struct {
	items           map[string]*domain.InventoryItem
	saveErr         error
	findErr         error
	findByOrderErr  error
	findByLocationErr error
	findByZoneErr   error
	findLowStockErr error
	findAllErr      error
	deleteErr       error
}

func (f *fakeInventoryRepo) Save(ctx context.Context, item *domain.InventoryItem) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if f.items == nil {
		f.items = make(map[string]*domain.InventoryItem)
	}
	f.items[item.SKU] = item
	return nil
}

func (f *fakeInventoryRepo) FindBySKU(ctx context.Context, sku string) (*domain.InventoryItem, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.items == nil {
		return nil, nil
	}
	return f.items[sku], nil
}

func (f *fakeInventoryRepo) FindByLocation(ctx context.Context, locationID string) ([]*domain.InventoryItem, error) {
	if f.findByLocationErr != nil {
		return nil, f.findByLocationErr
	}
	results := make([]*domain.InventoryItem, 0)
	for _, item := range f.items {
		for _, loc := range item.Locations {
			if loc.LocationID == locationID {
				results = append(results, item)
				break
			}
		}
	}
	return results, nil
}

func (f *fakeInventoryRepo) FindByZone(ctx context.Context, zone string) ([]*domain.InventoryItem, error) {
	if f.findByZoneErr != nil {
		return nil, f.findByZoneErr
	}
	results := make([]*domain.InventoryItem, 0)
	for _, item := range f.items {
		for _, loc := range item.Locations {
			if loc.Zone == zone {
				results = append(results, item)
				break
			}
		}
	}
	return results, nil
}

func (f *fakeInventoryRepo) FindByOrderID(ctx context.Context, orderID string) ([]*domain.InventoryItem, error) {
	if f.findByOrderErr != nil {
		return nil, f.findByOrderErr
	}
	results := make([]*domain.InventoryItem, 0)
	for _, item := range f.items {
		for _, res := range item.Reservations {
			if res.OrderID == orderID {
				results = append(results, item)
				break
			}
		}
	}
	return results, nil
}

func (f *fakeInventoryRepo) FindLowStock(ctx context.Context) ([]*domain.InventoryItem, error) {
	if f.findLowStockErr != nil {
		return nil, f.findLowStockErr
	}
	results := make([]*domain.InventoryItem, 0)
	for _, item := range f.items {
		if item.AvailableQuantity <= item.ReorderPoint {
			results = append(results, item)
		}
	}
	return results, nil
}

func (f *fakeInventoryRepo) FindAll(ctx context.Context, limit, offset int) ([]*domain.InventoryItem, error) {
	if f.findAllErr != nil {
		return nil, f.findAllErr
	}
	results := make([]*domain.InventoryItem, 0, len(f.items))
	for _, item := range f.items {
		results = append(results, item)
	}
	return results, nil
}

func (f *fakeInventoryRepo) Delete(ctx context.Context, sku string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.items, sku)
	return nil
}

func newTestService(repo *fakeInventoryRepo) *InventoryApplicationService {
	logger := logging.New(logging.DefaultConfig("test"))
	return NewInventoryApplicationService(repo, nil, nil, nil, logger)
}

func newItemWithStock(sku string, qty int) *domain.InventoryItem {
	item := domain.NewInventoryItem(sku, "Widget", 5, 10)
	_ = item.ReceiveStock("LOC-1", "ZONE-A", qty, "PO-1", "user1")
	return item
}

func TestInventoryApplicationService_CreateAndGet(t *testing.T) {
	repo := &fakeInventoryRepo{}
	svc := newTestService(repo)

	dto, err := svc.CreateItem(context.Background(), CreateItemCommand{
		SKU:            "SKU-1",
		ProductName:    "Widget",
		ReorderPoint:   5,
		ReorderQuantity: 10,
	})
	require.NoError(t, err)
	require.NotNil(t, dto)

	got, err := svc.GetItem(context.Background(), GetItemQuery{SKU: "SKU-1"})
	require.NoError(t, err)
	assert.Equal(t, "SKU-1", got.SKU)

	_, err = svc.GetItem(context.Background(), GetItemQuery{SKU: "missing"})
	assert.Error(t, err)
}

func TestInventoryApplicationService_ReceiveReservePickRelease(t *testing.T) {
	repo := &fakeInventoryRepo{items: map[string]*domain.InventoryItem{"SKU-1": newItemWithStock("SKU-1", 10)}}
	svc := newTestService(repo)

	_, err := svc.ReceiveStock(context.Background(), ReceiveStockCommand{
		SKU:        "SKU-1",
		LocationID: "LOC-1",
		Zone:       "ZONE-A",
		Quantity:   5,
		ReferenceID: "PO-2",
		CreatedBy:  "user1",
	})
	require.NoError(t, err)

	_, err = svc.Reserve(context.Background(), ReserveCommand{
		SKU:        "SKU-1",
		OrderID:    "ORD-1",
		LocationID: "LOC-1",
		Quantity:   5,
	})
	require.NoError(t, err)

	_, err = svc.Pick(context.Background(), PickCommand{
		SKU:        "SKU-1",
		OrderID:    "ORD-1",
		LocationID: "LOC-1",
		Quantity:   5,
		CreatedBy:  "user1",
	})
	require.NoError(t, err)

	_, err = svc.ReleaseReservation(context.Background(), ReleaseReservationCommand{
		SKU:     "SKU-1",
		OrderID: "ORD-1",
	})
	assert.Error(t, err)
}

func TestInventoryApplicationService_ReleaseByOrderAndAdjust(t *testing.T) {
	itemA := newItemWithStock("SKU-A", 10)
	_ = itemA.Reserve("ORD-1", "LOC-1", 2)
	itemB := newItemWithStock("SKU-B", 10)
	_ = itemB.Reserve("ORD-1", "LOC-1", 3)

	repo := &fakeInventoryRepo{items: map[string]*domain.InventoryItem{
		"SKU-A": itemA,
		"SKU-B": itemB,
	}}
	svc := newTestService(repo)

	released, err := svc.ReleaseByOrder(context.Background(), ReleaseByOrderCommand{OrderID: "ORD-1"})
	require.NoError(t, err)
	assert.Equal(t, 2, released)

	_, err = svc.Adjust(context.Background(), AdjustCommand{
		SKU:         "SKU-A",
		LocationID:  "LOC-1",
		NewQuantity: 20,
		Reason:      "count",
		CreatedBy:   "user1",
	})
	require.NoError(t, err)
	assert.Equal(t, 20, repo.items["SKU-A"].TotalQuantity)
}

func TestInventoryApplicationService_QueryHelpers(t *testing.T) {
	itemA := newItemWithStock("SKU-A", 2)
	itemB := newItemWithStock("SKU-B", 10)
	itemB.ReorderPoint = 2

	repo := &fakeInventoryRepo{items: map[string]*domain.InventoryItem{
		"SKU-A": itemA,
		"SKU-B": itemB,
	}}
	svc := newTestService(repo)

	items, err := svc.GetByLocation(context.Background(), GetByLocationQuery{LocationID: "LOC-1"})
	require.NoError(t, err)
	assert.Len(t, items, 2)

	items, err = svc.GetByZone(context.Background(), GetByZoneQuery{Zone: "ZONE-A"})
	require.NoError(t, err)
	assert.Len(t, items, 2)

	items, err = svc.GetLowStock(context.Background())
	require.NoError(t, err)
	assert.Len(t, items, 1)

	list, err := svc.ListInventory(context.Background(), ListInventoryQuery{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestInventoryApplicationService_StagePackShipReturn(t *testing.T) {
	item := newItemWithStock("SKU-1", 10)
	require.NoError(t, item.Reserve("ORD-1", "LOC-1", 4))
	reservationID := item.Reservations[0].ReservationID

	repo := &fakeInventoryRepo{items: map[string]*domain.InventoryItem{"SKU-1": item}}
	svc := newTestService(repo)

	_, err := svc.Stage(context.Background(), StageCommand{
		SKU:              "SKU-1",
		ReservationID:    reservationID,
		StagingLocationID: "STAGE-1",
		StagedBy:         "user1",
	})
	require.NoError(t, err)
	require.Len(t, repo.items["SKU-1"].HardAllocations, 1)
	allocationID := repo.items["SKU-1"].HardAllocations[0].AllocationID

	_, err = svc.Pack(context.Background(), PackCommand{
		SKU:         "SKU-1",
		AllocationID: allocationID,
		PackedBy:    "user2",
	})
	require.NoError(t, err)
	assert.Equal(t, "packed", repo.items["SKU-1"].HardAllocations[0].Status)

	_, err = svc.Ship(context.Background(), ShipCommand{
		SKU:         "SKU-1",
		AllocationID: allocationID,
	})
	require.NoError(t, err)
	assert.Equal(t, "shipped", repo.items["SKU-1"].HardAllocations[0].Status)

	item2 := newItemWithStock("SKU-2", 10)
	require.NoError(t, item2.Reserve("ORD-2", "LOC-1", 3))
	reservationID2 := item2.Reservations[0].ReservationID
	require.NoError(t, item2.Stage(reservationID2, "STAGE-1", "user1"))
	allocationID2 := item2.HardAllocations[0].AllocationID

	repo.items["SKU-2"] = item2
	_, err = svc.ReturnToShelf(context.Background(), ReturnToShelfCommand{
		SKU:         "SKU-2",
		AllocationID: allocationID2,
		ReturnedBy:  "user3",
		Reason:      "damaged",
	})
	require.NoError(t, err)
	assert.Equal(t, "returned", repo.items["SKU-2"].HardAllocations[0].Status)
}

func TestInventoryApplicationService_RecordShortage(t *testing.T) {
	item := newItemWithStock("SKU-1", 5)
	require.NoError(t, item.Reserve("ORD-1", "LOC-1", 4))

	repo := &fakeInventoryRepo{items: map[string]*domain.InventoryItem{"SKU-1": item}}
	svc := newTestService(repo)

	_, err := svc.RecordShortage(context.Background(), RecordShortageCommand{
		SKU:         "SKU-1",
		LocationID:  "LOC-1",
		OrderID:     "ORD-1",
		ExpectedQty: 4,
		ActualQty:   1,
		Reason:      "not_found",
		ReportedBy:  "user1",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.items["SKU-1"].TotalQuantity)
}
