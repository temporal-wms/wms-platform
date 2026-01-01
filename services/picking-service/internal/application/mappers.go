package application

import "github.com/wms-platform/picking-service/internal/domain"

// ToPickTaskDTO converts a domain PickTask to PickTaskDTO
func ToPickTaskDTO(task *domain.PickTask) *PickTaskDTO {
	if task == nil {
		return nil
	}

	items := make([]PickItemDTO, 0, len(task.Items))
	pickedItems := make([]PickedItemDTO, 0)
	for _, item := range task.Items {
		items = append(items, ToPickItemDTO(item))
		// Extract picked items (items with PickedQty > 0)
		if item.PickedQty > 0 && item.PickedAt != nil {
			pickedItems = append(pickedItems, PickedItemDTO{
				SKU:        item.SKU,
				Quantity:   item.PickedQty,
				LocationID: item.Location.LocationID,
				ToteID:     item.ToteID,
				PickedAt:   *item.PickedAt,
			})
		}
	}

	exceptions := make([]PickExceptionDTO, 0, len(task.Exceptions))
	for _, exception := range task.Exceptions {
		exceptions = append(exceptions, ToPickExceptionDTO(exception))
	}

	return &PickTaskDTO{
		TaskID:           task.TaskID,
		OrderID:          task.OrderID,
		WaveID:           task.WaveID,
		RouteID:          task.RouteID,
		PickerID:         task.PickerID,
		Status:           string(task.Status),
		Method:           string(task.Method),
		Items:            items,
		ToteID:           task.ToteID,
		Zone:             task.Zone,
		Priority:         task.Priority,
		TotalItems:       task.TotalItems,
		PickedItemsCount: task.PickedItems,
		PickedItems:      pickedItems,
		Exceptions:       exceptions,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
		AssignedAt:       task.AssignedAt,
		StartedAt:        task.StartedAt,
		CompletedAt:      task.CompletedAt,
	}
}

// ToPickItemDTO converts a domain PickItem to PickItemDTO
func ToPickItemDTO(item domain.PickItem) PickItemDTO {
	return PickItemDTO{
		SKU:         item.SKU,
		ProductName: item.ProductName,
		Quantity:    item.Quantity,
		PickedQty:   item.PickedQty,
		Location:    ToLocationDTO(item.Location),
		Status:      item.Status,
		ToteID:      item.ToteID,
		PickedAt:    item.PickedAt,
		VerifiedAt:  item.VerifiedAt,
		Notes:       item.Notes,
	}
}

// ToLocationDTO converts a domain Location to LocationDTO
func ToLocationDTO(location domain.Location) LocationDTO {
	return LocationDTO{
		LocationID: location.LocationID,
		Aisle:      location.Aisle,
		Rack:       location.Rack,
		Level:      location.Level,
		Position:   location.Position,
		Zone:       location.Zone,
	}
}

// ToPickExceptionDTO converts a domain PickException to PickExceptionDTO
func ToPickExceptionDTO(exception domain.PickException) PickExceptionDTO {
	return PickExceptionDTO{
		ExceptionID:  exception.ExceptionID,
		SKU:          exception.SKU,
		LocationID:   exception.LocationID,
		Reason:       exception.Reason,
		RequestedQty: exception.RequestedQty,
		AvailableQty: exception.AvailableQty,
		Resolution:   exception.Resolution,
		ResolvedAt:   exception.ResolvedAt,
		CreatedAt:    exception.CreatedAt,
	}
}

// ToPickTaskDTOs converts a slice of domain PickTasks to PickTaskDTOs
func ToPickTaskDTOs(tasks []*domain.PickTask) []PickTaskDTO {
	dtos := make([]PickTaskDTO, 0, len(tasks))
	for _, task := range tasks {
		if dto := ToPickTaskDTO(task); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}
