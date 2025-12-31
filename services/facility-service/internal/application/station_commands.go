package application

// Station Commands

// CreateStationCommand creates a new station
type CreateStationCommand struct {
	StationID          string   `json:"stationId"`
	Name               string   `json:"name"`
	Zone               string   `json:"zone"`
	StationType        string   `json:"stationType"`
	Capabilities       []string `json:"capabilities"`
	MaxConcurrentTasks int      `json:"maxConcurrentTasks"`
}

// UpdateStationCommand updates a station
type UpdateStationCommand struct {
	StationID          string `json:"stationId"`
	Name               string `json:"name"`
	Zone               string `json:"zone"`
	MaxConcurrentTasks int    `json:"maxConcurrentTasks"`
}

// AddCapabilityCommand adds a capability to a station
type AddCapabilityCommand struct {
	StationID  string `json:"stationId"`
	Capability string `json:"capability"`
}

// RemoveCapabilityCommand removes a capability from a station
type RemoveCapabilityCommand struct {
	StationID  string `json:"stationId"`
	Capability string `json:"capability"`
}

// SetCapabilitiesCommand sets all capabilities for a station
type SetCapabilitiesCommand struct {
	StationID    string   `json:"stationId"`
	Capabilities []string `json:"capabilities"`
}

// SetStationStatusCommand sets the station status
type SetStationStatusCommand struct {
	StationID string `json:"stationId"`
	Status    string `json:"status"`
}

// DeleteStationCommand deletes a station
type DeleteStationCommand struct {
	StationID string `json:"stationId"`
}

// Station Queries

// GetStationQuery retrieves a station by ID
type GetStationQuery struct {
	StationID string `json:"stationId"`
}

// ListStationsQuery lists all stations
type ListStationsQuery struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// FindCapableStationsQuery finds stations with required capabilities
type FindCapableStationsQuery struct {
	Requirements []string `json:"requirements"`
	StationType  string   `json:"stationType"`
	Zone         string   `json:"zone"`
}

// GetStationsByZoneQuery retrieves stations by zone
type GetStationsByZoneQuery struct {
	Zone string `json:"zone"`
}

// GetStationsByTypeQuery retrieves stations by type
type GetStationsByTypeQuery struct {
	StationType string `json:"stationType"`
}

// GetStationsByStatusQuery retrieves stations by status
type GetStationsByStatusQuery struct {
	Status string `json:"status"`
}
