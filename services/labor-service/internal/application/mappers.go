package application

import "github.com/wms-platform/labor-service/internal/domain"

// ToWorkerDTO converts a domain Worker to WorkerDTO
func ToWorkerDTO(worker *domain.Worker) *WorkerDTO {
	if worker == nil {
		return nil
	}

	skills := make([]SkillDTO, 0, len(worker.Skills))
	for _, skill := range worker.Skills {
		skills = append(skills, ToSkillDTO(skill))
	}

	dto := &WorkerDTO{
		WorkerID:           worker.WorkerID,
		EmployeeID:         worker.EmployeeID,
		Name:               worker.Name,
		Status:             string(worker.Status),
		CurrentZone:        worker.CurrentZone,
		Skills:             skills,
		PerformanceMetrics: ToPerformanceMetricsDTO(worker.PerformanceMetrics),
		CreatedAt:          worker.CreatedAt,
		UpdatedAt:          worker.UpdatedAt,
	}

	if worker.CurrentShift != nil {
		dto.CurrentShift = ToShiftDTO(worker.CurrentShift)
	}

	if worker.CurrentTask != nil {
		dto.CurrentTask = ToTaskAssignmentDTO(worker.CurrentTask)
	}

	return dto
}

// ToSkillDTO converts a domain Skill to SkillDTO
func ToSkillDTO(skill domain.Skill) SkillDTO {
	return SkillDTO{
		Type:        string(skill.Type),
		Level:       skill.Level,
		Certified:   skill.Certified,
		CertifiedAt: skill.CertifiedAt,
	}
}

// ToShiftDTO converts a domain Shift to ShiftDTO
func ToShiftDTO(shift *domain.Shift) *ShiftDTO {
	if shift == nil {
		return nil
	}

	breaks := make([]BreakDTO, 0, len(shift.BreaksTaken))
	for _, brk := range shift.BreaksTaken {
		breaks = append(breaks, ToBreakDTO(brk))
	}

	return &ShiftDTO{
		ShiftID:        shift.ShiftID,
		ShiftType:      shift.ShiftType,
		Zone:           shift.Zone,
		StartTime:      shift.StartTime,
		EndTime:        shift.EndTime,
		BreaksTaken:    breaks,
		TasksCompleted: shift.TasksCompleted,
		ItemsProcessed: shift.ItemsProcessed,
	}
}

// ToBreakDTO converts a domain Break to BreakDTO
func ToBreakDTO(brk domain.Break) BreakDTO {
	return BreakDTO{
		Type:      brk.Type,
		StartTime: brk.StartTime,
		EndTime:   brk.EndTime,
	}
}

// ToTaskAssignmentDTO converts a domain TaskAssignment to TaskAssignmentDTO
func ToTaskAssignmentDTO(task *domain.TaskAssignment) *TaskAssignmentDTO {
	if task == nil {
		return nil
	}

	return &TaskAssignmentDTO{
		TaskID:      task.TaskID,
		TaskType:    string(task.TaskType),
		Priority:    task.Priority,
		AssignedAt:  task.AssignedAt,
		StartedAt:   task.StartedAt,
		CompletedAt: task.CompletedAt,
	}
}

// ToPerformanceMetricsDTO converts a domain PerformanceMetrics to PerformanceMetricsDTO
func ToPerformanceMetricsDTO(metrics domain.PerformanceMetrics) PerformanceMetricsDTO {
	return PerformanceMetricsDTO{
		TotalTasksCompleted:  metrics.TotalTasksCompleted,
		TotalItemsProcessed:  metrics.TotalItemsProcessed,
		AverageTaskTime:      metrics.AverageTaskTime,
		AverageItemsPerHour:  metrics.AverageItemsPerHour,
		AccuracyRate:         metrics.AccuracyRate,
		LastUpdated:          metrics.LastUpdated,
	}
}

// ToWorkerDTOs converts a slice of domain Workers to WorkerDTOs
func ToWorkerDTOs(workers []*domain.Worker) []WorkerDTO {
	dtos := make([]WorkerDTO, 0, len(workers))
	for _, worker := range workers {
		if dto := ToWorkerDTO(worker); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}

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
