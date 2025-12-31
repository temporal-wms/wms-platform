package application

import "github.com/wms-platform/facility-service/internal/domain"

// ToStationDTO converts a domain Station to StationDTO
func ToStationDTO(station *domain.Station) *StationDTO {
	if station == nil {
		return nil
	}

	capabilities := make([]string, len(station.Capabilities))
	for i, cap := range station.Capabilities {
		capabilities[i] = string(cap)
	}

	equipment := make([]StationEquipmentDTO, len(station.Equipment))
	for i, eq := range station.Equipment {
		equipment[i] = StationEquipmentDTO{
			EquipmentID:   eq.EquipmentID,
			EquipmentType: eq.EquipmentType,
			Status:        eq.Status,
		}
	}

	return &StationDTO{
		StationID:          station.StationID,
		Name:               station.Name,
		Zone:               station.Zone,
		StationType:        string(station.StationType),
		Status:             string(station.Status),
		Capabilities:       capabilities,
		MaxConcurrentTasks: station.MaxConcurrentTasks,
		CurrentTasks:       station.CurrentTasks,
		AvailableCapacity:  station.GetAvailableCapacity(),
		AssignedWorkerID:   station.AssignedWorkerID,
		Equipment:          equipment,
		CreatedAt:          station.CreatedAt,
		UpdatedAt:          station.UpdatedAt,
	}
}

// ToStationDTOs converts a slice of domain Stations to StationDTOs
func ToStationDTOs(stations []*domain.Station) []StationDTO {
	dtos := make([]StationDTO, 0, len(stations))
	for _, station := range stations {
		if dto := ToStationDTO(station); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}
