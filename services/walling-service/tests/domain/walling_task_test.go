package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/walling-service/internal/domain"
)

func createTestItems() []domain.ItemToSort {
	return []domain.ItemToSort{
		{SKU: "SKU-001", Quantity: 3, FromToteID: "TOTE-001"},
		{SKU: "SKU-002", Quantity: 2, FromToteID: "TOTE-001"},
		{SKU: "SKU-003", Quantity: 1, FromToteID: "TOTE-002"},
	}
}

func createTestTotes() []domain.SourceTote {
	return []domain.SourceTote{
		{ToteID: "TOTE-001", PickTaskID: "PICK-001", ItemCount: 5},
		{ToteID: "TOTE-002", PickTaskID: "PICK-002", ItemCount: 1},
	}
}

func TestNewWallingTask(t *testing.T) {
	items := createTestItems()
	totes := createTestTotes()

	task, err := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	require.NoError(t, err)
	require.NotNil(t, task)
	assert.NotEmpty(t, task.TaskID)
	assert.Equal(t, "ORD-001", task.OrderID)
	assert.Equal(t, "WAVE-001", task.WaveID)
	assert.Equal(t, "WALL-A", task.PutWallID)
	assert.Equal(t, "BIN-001", task.DestinationBin)
	assert.Equal(t, domain.WallingTaskStatusPending, task.Status)
	assert.Len(t, task.ItemsToSort, 3)
	assert.Len(t, task.SourceTotes, 2)
	assert.Empty(t, task.SortedItems)
	assert.Equal(t, 5, task.Priority)

	// Check domain event
	events := task.GetDomainEvents()
	assert.Len(t, events, 1)
}

func TestNewWallingTask_NoItems(t *testing.T) {
	totes := createTestTotes()
	items := []domain.ItemToSort{}

	task, err := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	assert.Nil(t, task)
	assert.Equal(t, domain.ErrNoItemsToSort, err)
}

func TestWallingTask_SetRouteID(t *testing.T) {
	items := createTestItems()
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.SetRouteID("ROUTE-001")

	assert.Equal(t, "ROUTE-001", task.RouteID)
}

func TestWallingTask_Assign(t *testing.T) {
	items := createTestItems()
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	err := task.Assign("WALLINER-001", "STATION-A")

	require.NoError(t, err)
	assert.Equal(t, "WALLINER-001", task.WallinerID)
	assert.Equal(t, "STATION-A", task.Station)
	assert.Equal(t, domain.WallingTaskStatusAssigned, task.Status)
	assert.NotNil(t, task.AssignedAt)

	// Check domain event
	events := task.GetDomainEvents()
	assert.Len(t, events, 2) // Created + Assigned
}

func TestWallingTask_Assign_AlreadyCompleted(t *testing.T) {
	items := []domain.ItemToSort{{SKU: "SKU-001", Quantity: 1, FromToteID: "TOTE-001"}}
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	// Complete the task
	task.Assign("WALLINER-001", "STATION-A")
	task.SortItem("SKU-001", 1, "TOTE-001")

	// Try to reassign
	err := task.Assign("WALLINER-002", "STATION-B")
	assert.Equal(t, domain.ErrWallingTaskCompleted, err)
}

func TestWallingTask_Start(t *testing.T) {
	items := createTestItems()
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Assign("WALLINER-001", "STATION-A")
	err := task.Start()

	require.NoError(t, err)
	assert.Equal(t, domain.WallingTaskStatusInProgress, task.Status)
	assert.NotNil(t, task.StartedAt)
}

func TestWallingTask_Start_NotAssigned(t *testing.T) {
	items := createTestItems()
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	err := task.Start()

	assert.Error(t, err)
	assert.Equal(t, domain.WallingTaskStatusPending, task.Status)
}

func TestWallingTask_SortItem(t *testing.T) {
	items := createTestItems()
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Assign("WALLINER-001", "STATION-A")

	// Sort first item
	err := task.SortItem("SKU-001", 2, "TOTE-001")
	require.NoError(t, err)

	// Check sorted items
	assert.Len(t, task.SortedItems, 1)
	assert.Equal(t, "SKU-001", task.SortedItems[0].SKU)
	assert.Equal(t, 2, task.SortedItems[0].Quantity)
	assert.Equal(t, "TOTE-001", task.SortedItems[0].FromToteID)
	assert.Equal(t, "BIN-001", task.SortedItems[0].ToBinID)
	assert.True(t, task.SortedItems[0].Verified)

	// Check item progress
	assert.Equal(t, 2, task.ItemsToSort[0].SortedQty)

	// Task should be in progress
	assert.Equal(t, domain.WallingTaskStatusInProgress, task.Status)

	// Check domain events
	events := task.GetDomainEvents()
	assert.Len(t, events, 3) // Created + Assigned + ItemSorted
}

func TestWallingTask_SortItem_ItemNotFound(t *testing.T) {
	items := createTestItems()
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Assign("WALLINER-001", "STATION-A")

	// Try to sort non-existing item
	err := task.SortItem("SKU-NOTFOUND", 1, "TOTE-001")

	assert.Equal(t, domain.ErrItemNotFound, err)
}

func TestWallingTask_SortItem_LimitToRemaining(t *testing.T) {
	items := []domain.ItemToSort{{SKU: "SKU-001", Quantity: 3, FromToteID: "TOTE-001"}}
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Assign("WALLINER-001", "STATION-A")

	// Sort more than remaining
	err := task.SortItem("SKU-001", 10, "TOTE-001")
	require.NoError(t, err)

	// Should only sort 3 (the max remaining)
	assert.Len(t, task.SortedItems, 1)
	assert.Equal(t, 3, task.SortedItems[0].Quantity)
}

func TestWallingTask_SortItem_AutoComplete(t *testing.T) {
	items := []domain.ItemToSort{{SKU: "SKU-001", Quantity: 2, FromToteID: "TOTE-001"}}
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Assign("WALLINER-001", "STATION-A")

	// Sort all items
	err := task.SortItem("SKU-001", 2, "TOTE-001")
	require.NoError(t, err)

	// Task should be completed automatically
	assert.Equal(t, domain.WallingTaskStatusCompleted, task.Status)
	assert.NotNil(t, task.CompletedAt)
}

func TestWallingTask_SortAllItems(t *testing.T) {
	items := createTestItems() // 3 + 2 + 1 = 6 items
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Assign("WALLINER-001", "STATION-A")

	// Sort all items from all totes
	task.SortItem("SKU-001", 3, "TOTE-001")
	task.SortItem("SKU-002", 2, "TOTE-001")
	err := task.SortItem("SKU-003", 1, "TOTE-002")
	require.NoError(t, err)

	// All items sorted
	assert.True(t, task.AllItemsSorted())
	assert.Equal(t, domain.WallingTaskStatusCompleted, task.Status)

	sorted, total := task.GetProgress()
	assert.Equal(t, 6, sorted)
	assert.Equal(t, 6, total)
}

func TestWallingTask_AllItemsSorted(t *testing.T) {
	items := []domain.ItemToSort{
		{SKU: "SKU-001", Quantity: 2, FromToteID: "TOTE-001"},
		{SKU: "SKU-002", Quantity: 1, FromToteID: "TOTE-001"},
	}
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Assign("WALLINER-001", "STATION-A")

	// Initially not all sorted
	assert.False(t, task.AllItemsSorted())

	// Sort first item
	task.SortItem("SKU-001", 2, "TOTE-001")
	assert.False(t, task.AllItemsSorted())

	// Sort second item
	task.SortItem("SKU-002", 1, "TOTE-001")
	assert.True(t, task.AllItemsSorted())
}

func TestWallingTask_Complete(t *testing.T) {
	items := []domain.ItemToSort{{SKU: "SKU-001", Quantity: 1, FromToteID: "TOTE-001"}}
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Assign("WALLINER-001", "STATION-A")
	task.SortItem("SKU-001", 1, "TOTE-001")

	// Manual complete (though it auto-completes in this case)
	// Let's test with partial sorting then manual complete
	items2 := []domain.ItemToSort{{SKU: "SKU-001", Quantity: 5, FromToteID: "TOTE-001"}}
	task2, _ := domain.NewWallingTask("ORD-002", "WAVE-001", "WALL-A", "BIN-002", totes, items2)
	task2.Assign("WALLINER-001", "STATION-A")
	task2.SortItem("SKU-001", 3, "TOTE-001") // Only sort 3 of 5

	err := task2.Complete()
	require.NoError(t, err)
	assert.Equal(t, domain.WallingTaskStatusCompleted, task2.Status)
}

func TestWallingTask_Complete_AlreadyCompleted(t *testing.T) {
	items := []domain.ItemToSort{{SKU: "SKU-001", Quantity: 1, FromToteID: "TOTE-001"}}
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Assign("WALLINER-001", "STATION-A")
	task.SortItem("SKU-001", 1, "TOTE-001")

	// Try to complete again
	err := task.Complete()
	assert.Equal(t, domain.ErrWallingTaskCompleted, err)
}

func TestWallingTask_Cancel(t *testing.T) {
	items := createTestItems()
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	err := task.Cancel("order cancelled")

	require.NoError(t, err)
	assert.Equal(t, domain.WallingTaskStatusCancelled, task.Status)
}

func TestWallingTask_Cancel_AlreadyCompleted(t *testing.T) {
	items := []domain.ItemToSort{{SKU: "SKU-001", Quantity: 1, FromToteID: "TOTE-001"}}
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Assign("WALLINER-001", "STATION-A")
	task.SortItem("SKU-001", 1, "TOTE-001")

	// Try to cancel after completion
	err := task.Cancel("too late")
	assert.Equal(t, domain.ErrWallingTaskCompleted, err)
}

func TestWallingTask_SortItem_Cancelled(t *testing.T) {
	items := createTestItems()
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Cancel("order cancelled")

	task.Assign("WALLINER-001", "STATION-A") // This should fail
	err := task.SortItem("SKU-001", 1, "TOTE-001")

	assert.Equal(t, domain.ErrWallingTaskCancelled, err)
}

func TestWallingTask_GetProgress(t *testing.T) {
	items := createTestItems() // 3 + 2 + 1 = 6 items
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	sorted, total := task.GetProgress()
	assert.Equal(t, 0, sorted)
	assert.Equal(t, 6, total)

	task.Assign("WALLINER-001", "STATION-A")
	task.SortItem("SKU-001", 2, "TOTE-001")

	sorted, total = task.GetProgress()
	assert.Equal(t, 2, sorted)
	assert.Equal(t, 6, total)

	task.SortItem("SKU-002", 2, "TOTE-001")

	sorted, total = task.GetProgress()
	assert.Equal(t, 4, sorted)
	assert.Equal(t, 6, total)
}

func TestWallingTask_ClearDomainEvents(t *testing.T) {
	items := createTestItems()
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	assert.NotEmpty(t, task.GetDomainEvents())

	task.ClearDomainEvents()

	assert.Empty(t, task.GetDomainEvents())
}

func TestWallingTask_IncrementalSorting(t *testing.T) {
	// Test sorting items in multiple steps
	items := []domain.ItemToSort{{SKU: "SKU-001", Quantity: 5, FromToteID: "TOTE-001"}}
	totes := createTestTotes()
	task, _ := domain.NewWallingTask("ORD-001", "WAVE-001", "WALL-A", "BIN-001", totes, items)

	task.Assign("WALLINER-001", "STATION-A")

	// Sort 2 items
	err := task.SortItem("SKU-001", 2, "TOTE-001")
	require.NoError(t, err)
	assert.Len(t, task.SortedItems, 1)
	assert.Equal(t, 2, task.ItemsToSort[0].SortedQty)

	// Sort 2 more
	err = task.SortItem("SKU-001", 2, "TOTE-001")
	require.NoError(t, err)
	assert.Len(t, task.SortedItems, 2)
	assert.Equal(t, 4, task.ItemsToSort[0].SortedQty)

	// Sort remaining 1
	err = task.SortItem("SKU-001", 1, "TOTE-001")
	require.NoError(t, err)
	assert.Len(t, task.SortedItems, 3)
	assert.Equal(t, 5, task.ItemsToSort[0].SortedQty)

	// Task should be completed
	assert.Equal(t, domain.WallingTaskStatusCompleted, task.Status)
}
