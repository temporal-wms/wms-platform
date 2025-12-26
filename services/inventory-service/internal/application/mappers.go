package application

import "github.com/wms-platform/inventory-service/internal/domain"

// ToInventoryItemDTO converts a domain InventoryItem to InventoryItemDTO
func ToInventoryItemDTO(item *domain.InventoryItem) *InventoryItemDTO {
	if item == nil {
		return nil
	}

	locations := make([]StockLocationDTO, 0, len(item.Locations))
	for _, loc := range item.Locations {
		locations = append(locations, StockLocationDTO{
			LocationID: loc.LocationID,
			Zone:       loc.Zone,
			Aisle:      loc.Aisle,
			Rack:       loc.Rack,
			Level:      loc.Level,
			Quantity:   loc.Quantity,
			Reserved:   loc.Reserved,
			Available:  loc.Available,
		})
	}

	reservations := make([]ReservationDTO, 0, len(item.Reservations))
	for _, res := range item.Reservations {
		reservations = append(reservations, ReservationDTO{
			ReservationID: res.ReservationID,
			OrderID:       res.OrderID,
			Quantity:      res.Quantity,
			LocationID:    res.LocationID,
			Status:        res.Status,
			CreatedAt:     res.CreatedAt,
			ExpiresAt:     res.ExpiresAt,
		})
	}

	return &InventoryItemDTO{
		SKU:               item.SKU,
		ProductName:       item.ProductName,
		Locations:         locations,
		TotalQuantity:     item.TotalQuantity,
		ReservedQuantity:  item.ReservedQuantity,
		AvailableQuantity: item.AvailableQuantity,
		ReorderPoint:      item.ReorderPoint,
		ReorderQuantity:   item.ReorderQuantity,
		Reservations:      reservations,
		LastCycleCount:    item.LastCycleCount,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

// ToInventoryListDTO converts a domain InventoryItem to InventoryListDTO (simplified)
func ToInventoryListDTO(item *domain.InventoryItem) *InventoryListDTO {
	if item == nil {
		return nil
	}

	return &InventoryListDTO{
		SKU:               item.SKU,
		ProductName:       item.ProductName,
		TotalQuantity:     item.TotalQuantity,
		ReservedQuantity:  item.ReservedQuantity,
		AvailableQuantity: item.AvailableQuantity,
		ReorderPoint:      item.ReorderPoint,
		LocationCount:     len(item.Locations),
		UpdatedAt:         item.UpdatedAt,
	}
}

// ToInventoryListDTOs converts a slice of domain InventoryItems to InventoryListDTOs
func ToInventoryListDTOs(items []*domain.InventoryItem) []InventoryListDTO {
	dtos := make([]InventoryListDTO, 0, len(items))
	for _, item := range items {
		if dto := ToInventoryListDTO(item); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}
