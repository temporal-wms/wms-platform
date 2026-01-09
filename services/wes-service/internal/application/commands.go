package application

// ResolveExecutionPlanCommand represents a request to resolve an execution plan
type ResolveExecutionPlanCommand struct {
	OrderID         string     `json:"orderId"`
	Items           []ItemInfo `json:"items"`
	MultiZone       bool       `json:"multiZone"`
	OrderType       string     `json:"orderType,omitempty"`
	SpecialHandling []string   `json:"specialHandling,omitempty"`
	// Optional: pre-fetched process path result
	ProcessPathResult *ProcessPathResultDTO `json:"processPathResult,omitempty"`
}

// ItemInfo represents item information for execution plan resolution
type ItemInfo struct {
	SKU               string  `json:"sku"`
	Quantity          int     `json:"quantity"`
	Weight            float64 `json:"weight"`
	LocationID        string  `json:"locationId,omitempty"`
	Zone              string  `json:"zone,omitempty"`
	IsFragile         bool    `json:"isFragile"`
	IsHazmat          bool    `json:"isHazmat"`
	RequiresColdChain bool    `json:"requiresColdChain"`
}

// CreateTaskRouteCommand represents a request to create a task route
type CreateTaskRouteCommand struct {
	OrderID         string   `json:"orderId"`
	WaveID          string   `json:"waveId"`
	TemplateID      string   `json:"templateId"`
	SpecialHandling []string `json:"specialHandling"`
	ProcessPathID   string   `json:"processPathId,omitempty"`
}

// AssignWorkerCommand represents a request to assign a worker to a stage
type AssignWorkerCommand struct {
	RouteID  string `json:"routeId"`
	WorkerID string `json:"workerId"`
	TaskID   string `json:"taskId"`
}

// StartStageCommand represents a request to start a stage
type StartStageCommand struct {
	RouteID string `json:"routeId"`
}

// CompleteStageCommand represents a request to complete a stage
type CompleteStageCommand struct {
	RouteID string `json:"routeId"`
}

// FailStageCommand represents a request to fail a stage
type FailStageCommand struct {
	RouteID string `json:"routeId"`
	Error   string `json:"error"`
}

// GetTaskRouteQuery represents a query for a task route
type GetTaskRouteQuery struct {
	RouteID string `json:"routeId"`
}

// GetRouteByOrderQuery represents a query for a task route by order
type GetRouteByOrderQuery struct {
	OrderID string `json:"orderId"`
}

// GetTemplateQuery represents a query for a stage template
type GetTemplateQuery struct {
	TemplateID string `json:"templateId"`
}

// ListTemplatesQuery represents a query to list templates
type ListTemplatesQuery struct {
	ActiveOnly bool `json:"activeOnly"`
}

// GetAvailableWorkersQuery represents a query for available workers
type GetAvailableWorkersQuery struct {
	TaskType string `json:"taskType"`
	Zone     string `json:"zone,omitempty"`
}
