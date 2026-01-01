package application

import "github.com/wms-platform/picking-service/internal/domain"

// CreatePickTaskCommand represents the command to create a new pick task
type CreatePickTaskCommand struct {
	TaskID  string
	OrderID string
	WaveID  string
	RouteID string
	Method  string
	Items   []domain.PickItem
}

// AssignTaskCommand represents the command to assign a task to a picker
type AssignTaskCommand struct {
	TaskID   string
	PickerID string
	ToteID   string
}

// StartTaskCommand represents the command to start a pick task
type StartTaskCommand struct {
	TaskID string
}

// ConfirmPickCommand represents the command to confirm an item was picked
type ConfirmPickCommand struct {
	TaskID     string
	SKU        string
	LocationID string
	PickedQty  int
	ToteID     string
}

// ReportExceptionCommand represents the command to report a pick exception
type ReportExceptionCommand struct {
	TaskID       string
	SKU          string
	LocationID   string
	Reason       string
	RequestedQty int
	AvailableQty int
}

// CompleteTaskCommand represents the command to complete a pick task
type CompleteTaskCommand struct {
	TaskID string
}

// GetPickTaskQuery represents the query to get a pick task by ID
type GetPickTaskQuery struct {
	TaskID string
}

// GetTasksByOrderQuery represents the query to get tasks by order ID
type GetTasksByOrderQuery struct {
	OrderID string
}

// GetTasksByWaveQuery represents the query to get tasks by wave ID
type GetTasksByWaveQuery struct {
	WaveID string
}

// GetTasksByPickerQuery represents the query to get tasks by picker ID
type GetTasksByPickerQuery struct {
	PickerID string
}

// GetActiveTaskQuery represents the query to get active task for a picker
type GetActiveTaskQuery struct {
	PickerID string
}

// GetPendingTasksQuery represents the query to get pending tasks
type GetPendingTasksQuery struct {
	Zone  string
	Limit int
}

// ListTasksQuery represents the query to list tasks with optional filters
type ListTasksQuery struct {
	Status string
	Zone   string
	Limit  int
	Offset int
}
