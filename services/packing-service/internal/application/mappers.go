package application

import "github.com/wms-platform/packing-service/internal/domain"

// ToPackTaskDTO converts a domain PackTask to PackTaskDTO
func ToPackTaskDTO(task *domain.PackTask) *PackTaskDTO {
	if task == nil {
		return nil
	}

	items := make([]PackItemDTO, 0, len(task.Items))
	for _, item := range task.Items {
		items = append(items, ToPackItemDTO(item))
	}

	dto := &PackTaskDTO{
		TaskID:          task.TaskID,
		OrderID:         task.OrderID,
		WaveID:          task.WaveID,
		ConsolidationID: task.ConsolidationID,
		Status:          string(task.Status),
		Items:           items,
		Package:         ToPackageDTO(task.Package),
		PackerID:        task.PackerID,
		Station:         task.Station,
		Priority:        task.Priority,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
		StartedAt:       task.StartedAt,
		PackedAt:        task.PackedAt,
		LabeledAt:       task.LabeledAt,
		CompletedAt:     task.CompletedAt,
	}

	if task.ShippingLabel != nil {
		dto.ShippingLabel = ToShippingLabelDTO(*task.ShippingLabel)
	}

	return dto
}

// ToPackItemDTO converts a domain PackItem to PackItemDTO
func ToPackItemDTO(item domain.PackItem) PackItemDTO {
	return PackItemDTO{
		SKU:         item.SKU,
		ProductName: item.ProductName,
		Quantity:    item.Quantity,
		Weight:      item.Weight,
		Fragile:     item.Fragile,
		Verified:    item.Verified,
	}
}

// ToPackageDTO converts domain Package to PackageDTO
func ToPackageDTO(pkg domain.Package) PackageDTO {
	return PackageDTO{
		PackageID:     pkg.PackageID,
		Type:          string(pkg.Type),
		SuggestedType: string(pkg.SuggestedType),
		Dimensions:    ToDimensionsDTO(pkg.Dimensions),
		Weight:        pkg.Weight,
		TotalWeight:   pkg.TotalWeight,
		Materials:     pkg.Materials,
		Sealed:        pkg.Sealed,
		SealedAt:      pkg.SealedAt,
	}
}

// ToDimensionsDTO converts domain Dimensions to DimensionsDTO
func ToDimensionsDTO(dims domain.Dimensions) DimensionsDTO {
	return DimensionsDTO{
		Length: dims.Length,
		Width:  dims.Width,
		Height: dims.Height,
	}
}

// ToShippingLabelDTO converts domain ShippingLabel to DTO
func ToShippingLabelDTO(label domain.ShippingLabel) *ShippingLabelDTO {
	return &ShippingLabelDTO{
		TrackingNumber: label.TrackingNumber,
		Carrier:        label.Carrier,
		ServiceType:    label.ServiceType,
		LabelURL:       label.LabelURL,
		LabelData:      label.LabelData,
		GeneratedAt:    label.GeneratedAt,
		AppliedAt:      label.AppliedAt,
	}
}

// ToPackTaskDTOs converts a slice of domain PackTasks to PackTaskDTOs
func ToPackTaskDTOs(tasks []*domain.PackTask) []PackTaskDTO {
	dtos := make([]PackTaskDTO, 0, len(tasks))
	for _, task := range tasks {
		if dto := ToPackTaskDTO(task); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}
