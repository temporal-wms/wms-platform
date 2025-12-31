package application

import "time"

// StationDTO represents a station data transfer object
type StationDTO struct {
	StationID          string              `json:"stationId"`
	Name               string              `json:"name"`
	Zone               string              `json:"zone"`
	StationType        string              `json:"stationType"`
	Status             string              `json:"status"`
	Capabilities       []string            `json:"capabilities"`
	MaxConcurrentTasks int                 `json:"maxConcurrentTasks"`
	CurrentTasks       int                 `json:"currentTasks"`
	AvailableCapacity  int                 `json:"availableCapacity"`
	AssignedWorkerID   string              `json:"assignedWorkerId,omitempty"`
	Equipment          []StationEquipmentDTO `json:"equipment"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
}

// StationEquipmentDTO represents equipment at a station
type StationEquipmentDTO struct {
	EquipmentID   string `json:"equipmentId"`
	EquipmentType string `json:"equipmentType"`
	Status        string `json:"status"`
}
