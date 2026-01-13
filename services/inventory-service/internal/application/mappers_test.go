package application

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/inventory-service/internal/domain"
)

func TestToInventoryItemDTO(t *testing.T) {
	now := time.Now()
	item := &domain.InventoryItem{
		SKU:             "SKU-1",
		ProductName:     "Widget",
		TotalQuantity:   10,
		ReservedQuantity: 2,
		AvailableQuantity: 8,
		ReorderPoint:    5,
		ReorderQuantity: 20,
		Locations: []domain.StockLocation{
			{
				LocationID: "LOC-1",
				Zone:       "ZONE-A",
				Quantity:   10,
				Reserved:   2,
				Available:  8,
			},
		},
		Reservations: []domain.Reservation{
			{
				ReservationID: "RES-1",
				OrderID:       "ORD-1",
				Quantity:      2,
				LocationID:    "LOC-1",
				Status:        "active",
				CreatedAt:     now,
				ExpiresAt:     now.Add(1 * time.Hour),
			},
		},
		HardAllocations: []domain.HardAllocation{
			{
				AllocationID:     "ALLOC-1",
				ReservationID:    "RES-1",
				OrderID:          "ORD-1",
				Quantity:         2,
				SourceLocationID: "LOC-1",
				Status:           "staged",
				StagedBy:         "user1",
				CreatedAt:        now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	dto := ToInventoryItemDTO(item)
	require.NotNil(t, dto)
	assert.Equal(t, "SKU-1", dto.SKU)
	assert.Equal(t, 1, len(dto.Locations))
	assert.Equal(t, 1, len(dto.Reservations))
	assert.Equal(t, 1, len(dto.HardAllocations))
}

func TestToInventoryListDTO(t *testing.T) {
	item := &domain.InventoryItem{
		SKU:             "SKU-2",
		ProductName:     "Widget-2",
		TotalQuantity:   10,
		ReservedQuantity: 1,
		AvailableQuantity: 9,
		ReorderPoint:    3,
		Locations: []domain.StockLocation{
			{LocationID: "LOC-1"},
			{LocationID: "LOC-2"},
		},
	}

	dto := ToInventoryListDTO(item)
	require.NotNil(t, dto)
	assert.Equal(t, "SKU-2", dto.SKU)
	assert.Equal(t, 2, dto.LocationCount)
}

func TestToInventoryListDTOs(t *testing.T) {
	items := []*domain.InventoryItem{
		{SKU: "SKU-1"},
		nil,
		{SKU: "SKU-2"},
	}

	dtos := ToInventoryListDTOs(items)
	assert.Len(t, dtos, 2)
}
