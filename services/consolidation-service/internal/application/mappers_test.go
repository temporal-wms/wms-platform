package application

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/consolidation-service/internal/domain"
)

func TestToConsolidationDTONil(t *testing.T) {
	assert.Nil(t, ToConsolidationDTO(nil))
}

func TestToConsolidationDTO(t *testing.T) {
	now := time.Now()
	unit := &domain.ConsolidationUnit{
		ConsolidationID: "CONS-20",
		OrderID:         "ORD-20",
		WaveID:          "WAVE-20",
		Status:          domain.ConsolidationStatusInProgress,
		Strategy:        domain.StrategyOrderBased,
		ExpectedItems: []domain.ExpectedItem{
			{SKU: "SKU-1", ProductName: "Item 1", Quantity: 2, SourceToteID: "TOTE-1", Received: 1, Status: "partial"},
		},
		ConsolidatedItems: []domain.ConsolidatedItem{
			{SKU: "SKU-1", Quantity: 1, SourceToteID: "TOTE-1", ScannedAt: now, VerifiedBy: "WK-1"},
		},
		SourceTotes:       []string{"TOTE-1"},
		DestinationBin:    "BIN-1",
		Station:           "ST-1",
		WorkerID:          "WK-1",
		TotalExpected:     2,
		TotalConsolidated: 1,
		ReadyForPacking:   false,
		CreatedAt:         now,
		UpdatedAt:         now,
		StartedAt:         &now,
	}

	dto := ToConsolidationDTO(unit)
	require.NotNil(t, dto)
	assert.Equal(t, "CONS-20", dto.ConsolidationID)
	assert.Equal(t, "in_progress", dto.Status)
	assert.Equal(t, "order", dto.Strategy)
	assert.Len(t, dto.ExpectedItems, 1)
	assert.Len(t, dto.ConsolidatedItems, 1)
}

func TestToExpectedItemDTO(t *testing.T) {
	item := domain.ExpectedItem{
		SKU:          "SKU-2",
		ProductName:  "Item 2",
		Quantity:     3,
		SourceToteID: "TOTE-2",
		Received:     1,
		Status:       "partial",
	}
	dto := ToExpectedItemDTO(item)
	assert.Equal(t, "SKU-2", dto.SKU)
	assert.Equal(t, "Item 2", dto.ProductName)
	assert.Equal(t, 3, dto.Quantity)
	assert.Equal(t, "TOTE-2", dto.SourceToteID)
	assert.Equal(t, 1, dto.Received)
	assert.Equal(t, "partial", dto.Status)
}

func TestToConsolidatedItemDTO(t *testing.T) {
	now := time.Now()
	item := domain.ConsolidatedItem{
		SKU:          "SKU-3",
		Quantity:     2,
		SourceToteID: "TOTE-3",
		ScannedAt:    now,
		VerifiedBy:   "WK-3",
	}
	dto := ToConsolidatedItemDTO(item)
	assert.Equal(t, "SKU-3", dto.SKU)
	assert.Equal(t, 2, dto.Quantity)
	assert.Equal(t, "TOTE-3", dto.SourceToteID)
	assert.Equal(t, now, dto.ScannedAt)
	assert.Equal(t, "WK-3", dto.VerifiedBy)
}

func TestToConsolidationDTOs(t *testing.T) {
	unit := &domain.ConsolidationUnit{ConsolidationID: "CONS-30"}
	dtos := ToConsolidationDTOs([]*domain.ConsolidationUnit{nil, unit})
	require.Len(t, dtos, 1)
	assert.Equal(t, "CONS-30", dtos[0].ConsolidationID)
}
