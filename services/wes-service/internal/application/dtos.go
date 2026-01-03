package application

import "github.com/wms-platform/wes-service/internal/domain"

// StageTemplateDTO represents a stage template for API responses
type StageTemplateDTO struct {
	TemplateID        string                  `json:"templateId"`
	PathType          string                  `json:"pathType"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description"`
	Stages            []StageDefinitionDTO    `json:"stages"`
	SelectionCriteria SelectionCriteriaDTO    `json:"selectionCriteria"`
	IsDefault         bool                    `json:"isDefault"`
	Active            bool                    `json:"active"`
}

// StageDefinitionDTO represents a stage definition
type StageDefinitionDTO struct {
	Order       int            `json:"order"`
	StageType   string         `json:"stageType"`
	TaskType    string         `json:"taskType"`
	Required    bool           `json:"required"`
	TimeoutMins int            `json:"timeoutMins"`
	Config      StageConfigDTO `json:"config,omitempty"`
}

// StageConfigDTO represents stage configuration
type StageConfigDTO struct {
	RequiresPutWall bool   `json:"requiresPutWall,omitempty"`
	PutWallZone     string `json:"putWallZone,omitempty"`
	StationID       string `json:"stationId,omitempty"`
}

// SelectionCriteriaDTO represents selection criteria
type SelectionCriteriaDTO struct {
	MinItems          *int     `json:"minItems,omitempty"`
	MaxItems          *int     `json:"maxItems,omitempty"`
	RequiresMultiZone bool     `json:"requiresMultiZone"`
	OrderTypes        []string `json:"orderTypes,omitempty"`
	Priority          int      `json:"priority"`
}

// TaskRouteDTO represents a task route for API responses
type TaskRouteDTO struct {
	RouteID         string           `json:"routeId"`
	OrderID         string           `json:"orderId"`
	WaveID          string           `json:"waveId"`
	PathTemplateID  string           `json:"pathTemplateId"`
	PathType        string           `json:"pathType"`
	CurrentStageIdx int              `json:"currentStageIdx"`
	Stages          []StageStatusDTO `json:"stages"`
	Status          string           `json:"status"`
	SpecialHandling []string         `json:"specialHandling"`
	ProcessPathID   string           `json:"processPathId,omitempty"`
	Progress        ProgressDTO      `json:"progress"`
	CreatedAt       int64            `json:"createdAt"`
	CompletedAt     *int64           `json:"completedAt,omitempty"`
}

// StageStatusDTO represents the status of a stage
type StageStatusDTO struct {
	StageType   string `json:"stageType"`
	TaskID      string `json:"taskId,omitempty"`
	WorkerID    string `json:"workerId,omitempty"`
	Status      string `json:"status"`
	StartedAt   *int64 `json:"startedAt,omitempty"`
	CompletedAt *int64 `json:"completedAt,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ProgressDTO represents progress information
type ProgressDTO struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
}

// ExecutionPlanDTO represents the resolved execution plan
type ExecutionPlanDTO struct {
	TemplateID      string               `json:"templateId"`
	PathType        string               `json:"pathType"`
	Stages          []StageDefinitionDTO `json:"stages"`
	SpecialHandling []string             `json:"specialHandling"`
	TargetStation   string               `json:"targetStation,omitempty"`
	ProcessPathID   string               `json:"processPathId"`
}

// ProcessPathResultDTO represents the result from process-path-service
type ProcessPathResultDTO struct {
	PathID                string   `json:"pathId"`
	Requirements          []string `json:"requirements"`
	ConsolidationRequired bool     `json:"consolidationRequired"`
	GiftWrapRequired      bool     `json:"giftWrapRequired"`
	SpecialHandling       []string `json:"specialHandling"`
	TargetStationID       string   `json:"targetStationId,omitempty"`
}

// WorkerAssignmentDTO represents a worker assignment result
type WorkerAssignmentDTO struct {
	WorkerID  string `json:"workerId"`
	TaskID    string `json:"taskId"`
	StageType string `json:"stageType"`
}

// MapTemplateToDTO maps a domain StageTemplate to DTO
func MapTemplateToDTO(template *domain.StageTemplate) *StageTemplateDTO {
	if template == nil {
		return nil
	}

	stages := make([]StageDefinitionDTO, len(template.Stages))
	for i, s := range template.Stages {
		stages[i] = StageDefinitionDTO{
			Order:       s.Order,
			StageType:   string(s.StageType),
			TaskType:    s.TaskType,
			Required:    s.Required,
			TimeoutMins: s.TimeoutMins,
			Config: StageConfigDTO{
				RequiresPutWall: s.Config.RequiresPutWall,
				PutWallZone:     s.Config.PutWallZone,
				StationID:       s.Config.StationID,
			},
		}
	}

	return &StageTemplateDTO{
		TemplateID:  template.TemplateID,
		PathType:    string(template.PathType),
		Name:        template.Name,
		Description: template.Description,
		Stages:      stages,
		SelectionCriteria: SelectionCriteriaDTO{
			MinItems:          template.SelectionCriteria.MinItems,
			MaxItems:          template.SelectionCriteria.MaxItems,
			RequiresMultiZone: template.SelectionCriteria.RequiresMultiZone,
			OrderTypes:        template.SelectionCriteria.OrderTypes,
			Priority:          template.SelectionCriteria.Priority,
		},
		IsDefault: template.IsDefault,
		Active:    template.Active,
	}
}

// MapRouteToDTO maps a domain TaskRoute to DTO
func MapRouteToDTO(route *domain.TaskRoute) *TaskRouteDTO {
	if route == nil {
		return nil
	}

	stages := make([]StageStatusDTO, len(route.Stages))
	for i, s := range route.Stages {
		stages[i] = StageStatusDTO{
			StageType:   string(s.StageType),
			TaskID:      s.TaskID,
			WorkerID:    s.WorkerID,
			Status:      s.Status,
			StartedAt:   s.StartedAt,
			CompletedAt: s.CompletedAt,
			Error:       s.Error,
		}
	}

	completed, total := route.GetProgress()

	var completedAt *int64
	if route.CompletedAt != nil {
		ts := route.CompletedAt.UnixMilli()
		completedAt = &ts
	}

	return &TaskRouteDTO{
		RouteID:         route.RouteID,
		OrderID:         route.OrderID,
		WaveID:          route.WaveID,
		PathTemplateID:  route.PathTemplateID,
		PathType:        string(route.PathType),
		CurrentStageIdx: route.CurrentStageIdx,
		Stages:          stages,
		Status:          string(route.Status),
		SpecialHandling: route.SpecialHandling,
		ProcessPathID:   route.ProcessPathID,
		Progress: ProgressDTO{
			Completed: completed,
			Total:     total,
		},
		CreatedAt:   route.CreatedAt.UnixMilli(),
		CompletedAt: completedAt,
	}
}
