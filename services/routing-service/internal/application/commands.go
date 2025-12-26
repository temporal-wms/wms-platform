package application

import "github.com/wms-platform/routing-service/internal/domain"

// CalculateRouteCommand calculates a new route
type CalculateRouteCommand struct {
	RouteRequest domain.RouteRequest
}

// StartRouteCommand starts a route
type StartRouteCommand struct {
	RouteID  string
	PickerID string
}

// CompleteStopCommand completes a stop in the route
type CompleteStopCommand struct {
	RouteID    string
	StopNumber int
	PickedQty  int
	ToteID     string
}

// SkipStopCommand skips a stop in the route
type SkipStopCommand struct {
	RouteID    string
	StopNumber int
	Reason     string
}

// CompleteRouteCommand completes a route
type CompleteRouteCommand struct {
	RouteID string
}

// PauseRouteCommand pauses a route
type PauseRouteCommand struct {
	RouteID string
}

// CancelRouteCommand cancels a route
type CancelRouteCommand struct {
	RouteID string
	Reason  string
}

// DeleteRouteCommand deletes a route
type DeleteRouteCommand struct {
	RouteID string
}

// GetRouteQuery retrieves a route by ID
type GetRouteQuery struct {
	RouteID string
}

// GetRoutesByOrderQuery retrieves routes by order ID
type GetRoutesByOrderQuery struct {
	OrderID string
}

// GetRoutesByWaveQuery retrieves routes by wave ID
type GetRoutesByWaveQuery struct {
	WaveID string
}

// GetRoutesByPickerQuery retrieves routes by picker ID
type GetRoutesByPickerQuery struct {
	PickerID string
}

// GetActiveRouteQuery retrieves active route for a picker
type GetActiveRouteQuery struct {
	PickerID string
}

// GetRoutesByStatusQuery retrieves routes by status
type GetRoutesByStatusQuery struct {
	Status domain.RouteStatus
}

// GetPendingRoutesQuery retrieves pending routes
type GetPendingRoutesQuery struct {
	Zone  string
	Limit int
}

// AnalyzeRouteQuery analyzes route efficiency
type AnalyzeRouteQuery struct {
	RouteID string
}

// SuggestStrategyQuery suggests routing strategy
type SuggestStrategyQuery struct {
	Items []domain.RouteItem
}
