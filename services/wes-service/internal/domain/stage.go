package domain

// StageType represents the type of warehouse execution stage
type StageType string

const (
	StagePicking       StageType = "picking"
	StageWalling       StageType = "walling"
	StageConsolidation StageType = "consolidation"
	StagePacking       StageType = "packing"
)

// StageDefinition represents a stage in a process path template
type StageDefinition struct {
	Order       int         `bson:"order" json:"order"`
	StageType   StageType   `bson:"stageType" json:"stageType"`
	TaskType    string      `bson:"taskType" json:"taskType"`
	Required    bool        `bson:"required" json:"required"`
	TimeoutMins int         `bson:"timeoutMins" json:"timeoutMins"`
	Config      StageConfig `bson:"config,omitempty" json:"config,omitempty"`
}

// StageConfig contains stage-specific configuration
type StageConfig struct {
	RequiresPutWall bool   `bson:"requiresPutWall,omitempty" json:"requiresPutWall,omitempty"`
	PutWallZone     string `bson:"putWallZone,omitempty" json:"putWallZone,omitempty"`
	StationID       string `bson:"stationId,omitempty" json:"stationId,omitempty"`
}

// StageStatus represents the status of a stage within a task route
type StageStatus struct {
	StageType   StageType `bson:"stageType" json:"stageType"`
	TaskID      string    `bson:"taskId,omitempty" json:"taskId,omitempty"`
	WorkerID    string    `bson:"workerId,omitempty" json:"workerId,omitempty"`
	Status      string    `bson:"status" json:"status"` // pending, assigned, in_progress, completed, failed
	StartedAt   *int64    `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt *int64    `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	Error       string    `bson:"error,omitempty" json:"error,omitempty"`
}

// Stage status constants
const (
	StageStatusPending    = "pending"
	StageStatusAssigned   = "assigned"
	StageStatusInProgress = "in_progress"
	StageStatusCompleted  = "completed"
	StageStatusFailed     = "failed"
)
