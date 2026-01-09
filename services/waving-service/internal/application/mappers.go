package application

import "github.com/wms-platform/waving-service/internal/domain"

// ToWaveDTO converts a domain Wave to WaveDTO
func ToWaveDTO(wave *domain.Wave) *WaveDTO {
	if wave == nil {
		return nil
	}

	orders := make([]WaveOrderDTO, 0, len(wave.Orders))
	totalItems := 0
	totalWeight := 0.0
	for _, order := range wave.Orders {
		orders = append(orders, WaveOrderDTO{
			OrderID:            order.OrderID,
			CustomerID:         order.CustomerID,
			Priority:           order.Priority,
			ItemCount:          order.ItemCount,
			TotalWeight:        order.TotalWeight,
			PromisedDeliveryAt: order.PromisedDeliveryAt,
			CarrierCutoff:      order.CarrierCutoff,
			Zone:               order.Zone,
			Status:             order.Status,
			AddedAt:            order.AddedAt,
		})
		totalItems += order.ItemCount
		totalWeight += order.TotalWeight
	}

	estDuration := ""
	if wave.EstimatedDuration > 0 {
		estDuration = wave.EstimatedDuration.String()
	}

	releaseDelay := ""
	if wave.Configuration.ReleaseDelay > 0 {
		releaseDelay = wave.Configuration.ReleaseDelay.String()
	}

	return &WaveDTO{
		WaveID:          wave.WaveID,
		WaveType:        string(wave.WaveType),
		Status:          string(wave.Status),
		FulfillmentMode: string(wave.FulfillmentMode),
		Orders:          orders,
		Configuration: WaveConfigurationDTO{
			MaxOrders:           wave.Configuration.MaxOrders,
			MaxItems:            wave.Configuration.MaxItems,
			MaxWeight:           wave.Configuration.MaxWeight,
			CarrierFilter:       wave.Configuration.CarrierFilter,
			PriorityFilter:      wave.Configuration.PriorityFilter,
			ZoneFilter:          wave.Configuration.ZoneFilter,
			CutoffTime:          wave.Configuration.CutoffTime,
			ReleaseDelay:        releaseDelay,
			AutoRelease:         wave.Configuration.AutoRelease,
			OptimizeForCarrier:  wave.Configuration.OptimizeForCarrier,
			OptimizeForZone:     wave.Configuration.OptimizeForZone,
			OptimizeForPriority: wave.Configuration.OptimizeForPriority,
		},
		LaborAllocation: LaborAllocationDTO{
			PickersRequired:   wave.LaborAllocation.PickersRequired,
			PickersAssigned:   wave.LaborAllocation.PickersAssigned,
			PackersRequired:   wave.LaborAllocation.PackersRequired,
			PackersAssigned:   wave.LaborAllocation.PackersAssigned,
			AssignedWorkerIDs: wave.LaborAllocation.AssignedWorkerIDs,
		},
		ScheduledStart:    wave.ScheduledStart,
		ScheduledEnd:      wave.ScheduledEnd,
		ActualStart:       wave.ActualStart,
		ActualEnd:         wave.ActualEnd,
		EstimatedDuration: estDuration,
		Priority:          wave.Priority,
		Zone:              wave.Zone,
		CreatedAt:         wave.CreatedAt,
		UpdatedAt:         wave.UpdatedAt,
		ReleasedAt:        wave.ReleasedAt,
		CompletedAt:       wave.CompletedAt,
		OrderCount:        len(wave.Orders),
		TotalItems:        totalItems,
		TotalWeight:       totalWeight,
	}
}

// ToWaveListDTO converts a domain Wave to WaveListDTO (simplified)
func ToWaveListDTO(wave *domain.Wave) *WaveListDTO {
	if wave == nil {
		return nil
	}

	return &WaveListDTO{
		WaveID:          wave.WaveID,
		WaveType:        string(wave.WaveType),
		Status:          string(wave.Status),
		FulfillmentMode: string(wave.FulfillmentMode),
		OrderCount:      len(wave.Orders),
		Priority:        wave.Priority,
		Zone:            wave.Zone,
		ScheduledStart:  wave.ScheduledStart,
		ScheduledEnd:    wave.ScheduledEnd,
		ReleasedAt:      wave.ReleasedAt,
		CreatedAt:       wave.CreatedAt,
		UpdatedAt:       wave.UpdatedAt,
	}
}

// ToWaveListDTOs converts a slice of domain Waves to WaveListDTOs
func ToWaveListDTOs(waves []*domain.Wave) []WaveListDTO {
	dtos := make([]WaveListDTO, 0, len(waves))
	for _, wave := range waves {
		if dto := ToWaveListDTO(wave); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}

// ToWaveDTOs converts a slice of domain Waves to WaveDTOs
func ToWaveDTOs(waves []*domain.Wave) []WaveDTO {
	dtos := make([]WaveDTO, 0, len(waves))
	for _, wave := range waves {
		if dto := ToWaveDTO(wave); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}
