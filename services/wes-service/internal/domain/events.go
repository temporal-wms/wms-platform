package domain

import "time"

// DomainEvent represents a domain event
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// RouteCreatedEvent is emitted when a new task route is created
type RouteCreatedEvent struct {
	RouteID        string    `json:"routeId"`
	OrderID        string    `json:"orderId"`
	WaveID         string    `json:"waveId"`
	PathTemplateID string    `json:"pathTemplateId"`
	PathType       string    `json:"pathType"`
	StageCount     int       `json:"stageCount"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (e *RouteCreatedEvent) EventType() string     { return "wms.wes.route-created" }
func (e *RouteCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// StageAssignedEvent is emitted when a worker is assigned to a stage
type StageAssignedEvent struct {
	RouteID   string `json:"routeId"`
	OrderID   string `json:"orderId"`
	StageType string `json:"stageType"`
	WorkerID  string `json:"workerId"`
	TaskID    string `json:"taskId"`
	Timestamp int64  `json:"timestamp"`
}

func (e *StageAssignedEvent) EventType() string     { return "wms.wes.stage-assigned" }
func (e *StageAssignedEvent) OccurredAt() time.Time { return time.UnixMilli(e.Timestamp) }

// StageStartedEvent is emitted when a stage starts execution
type StageStartedEvent struct {
	RouteID   string `json:"routeId"`
	OrderID   string `json:"orderId"`
	StageType string `json:"stageType"`
	TaskID    string `json:"taskId"`
	WorkerID  string `json:"workerId"`
	Timestamp int64  `json:"timestamp"`
}

func (e *StageStartedEvent) EventType() string     { return "wms.wes.stage-started" }
func (e *StageStartedEvent) OccurredAt() time.Time { return time.UnixMilli(e.Timestamp) }

// StageCompletedEvent is emitted when a stage completes successfully
type StageCompletedEvent struct {
	RouteID   string `json:"routeId"`
	OrderID   string `json:"orderId"`
	StageType string `json:"stageType"`
	TaskID    string `json:"taskId"`
	WorkerID  string `json:"workerId"`
	Timestamp int64  `json:"timestamp"`
}

func (e *StageCompletedEvent) EventType() string     { return "wms.wes.stage-completed" }
func (e *StageCompletedEvent) OccurredAt() time.Time { return time.UnixMilli(e.Timestamp) }

// StageFailedEvent is emitted when a stage fails
type StageFailedEvent struct {
	RouteID   string `json:"routeId"`
	OrderID   string `json:"orderId"`
	StageType string `json:"stageType"`
	TaskID    string `json:"taskId"`
	Error     string `json:"error"`
	Timestamp int64  `json:"timestamp"`
}

func (e *StageFailedEvent) EventType() string     { return "wms.wes.stage-failed" }
func (e *StageFailedEvent) OccurredAt() time.Time { return time.UnixMilli(e.Timestamp) }

// RouteCompletedEvent is emitted when an entire route completes
type RouteCompletedEvent struct {
	RouteID     string    `json:"routeId"`
	OrderID     string    `json:"orderId"`
	PathType    string    `json:"pathType"`
	StageCount  int       `json:"stageCount"`
	CompletedAt time.Time `json:"completedAt"`
}

func (e *RouteCompletedEvent) EventType() string     { return "wms.wes.route-completed" }
func (e *RouteCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }

// TemplateCreatedEvent is emitted when a new stage template is created
type TemplateCreatedEvent struct {
	TemplateID string    `json:"templateId"`
	PathType   string    `json:"pathType"`
	Name       string    `json:"name"`
	StageCount int       `json:"stageCount"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (e *TemplateCreatedEvent) EventType() string     { return "wms.wes.template-created" }
func (e *TemplateCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }
