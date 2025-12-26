package application

import "github.com/wms-platform/consolidation-service/internal/domain"

// ToConsolidationDTO converts a domain ConsolidationUnit to ConsolidationDTO
func ToConsolidationDTO(unit *domain.ConsolidationUnit) *ConsolidationDTO {
	if unit == nil {
		return nil
	}

	expectedItems := make([]ExpectedItemDTO, 0, len(unit.ExpectedItems))
	for _, item := range unit.ExpectedItems {
		expectedItems = append(expectedItems, ToExpectedItemDTO(item))
	}

	consolidatedItems := make([]ConsolidatedItemDTO, 0, len(unit.ConsolidatedItems))
	for _, item := range unit.ConsolidatedItems {
		consolidatedItems = append(consolidatedItems, ToConsolidatedItemDTO(item))
	}

	return &ConsolidationDTO{
		ConsolidationID:   unit.ConsolidationID,
		OrderID:           unit.OrderID,
		WaveID:            unit.WaveID,
		Status:            string(unit.Status),
		Strategy:          string(unit.Strategy),
		ExpectedItems:     expectedItems,
		ConsolidatedItems: consolidatedItems,
		SourceTotes:       unit.SourceTotes,
		DestinationBin:    unit.DestinationBin,
		Station:           unit.Station,
		WorkerID:          unit.WorkerID,
		TotalExpected:     unit.TotalExpected,
		TotalConsolidated: unit.TotalConsolidated,
		ReadyForPacking:   unit.ReadyForPacking,
		CreatedAt:         unit.CreatedAt,
		UpdatedAt:         unit.UpdatedAt,
		StartedAt:         unit.StartedAt,
		CompletedAt:       unit.CompletedAt,
	}
}

// ToExpectedItemDTO converts a domain ExpectedItem to ExpectedItemDTO
func ToExpectedItemDTO(item domain.ExpectedItem) ExpectedItemDTO {
	return ExpectedItemDTO{
		SKU:          item.SKU,
		ProductName:  item.ProductName,
		Quantity:     item.Quantity,
		SourceToteID: item.SourceToteID,
		Received:     item.Received,
		Status:       item.Status,
	}
}

// ToConsolidatedItemDTO converts a domain ConsolidatedItem to ConsolidatedItemDTO
func ToConsolidatedItemDTO(item domain.ConsolidatedItem) ConsolidatedItemDTO {
	return ConsolidatedItemDTO{
		SKU:          item.SKU,
		Quantity:     item.Quantity,
		SourceToteID: item.SourceToteID,
		ScannedAt:    item.ScannedAt,
		VerifiedBy:   item.VerifiedBy,
	}
}

// ToConsolidationDTOs converts a slice of domain ConsolidationUnits to ConsolidationDTOs
func ToConsolidationDTOs(units []*domain.ConsolidationUnit) []ConsolidationDTO {
	dtos := make([]ConsolidationDTO, 0, len(units))
	for _, unit := range units {
		if dto := ToConsolidationDTO(unit); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}
