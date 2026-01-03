package domain

import "context"

// StageTemplateRepository defines the interface for stage template persistence
type StageTemplateRepository interface {
	// Save saves a stage template
	Save(ctx context.Context, template *StageTemplate) error

	// FindByID finds a template by its MongoDB ObjectID
	FindByID(ctx context.Context, id string) (*StageTemplate, error)

	// FindByTemplateID finds a template by its template ID
	FindByTemplateID(ctx context.Context, templateID string) (*StageTemplate, error)

	// FindByPathType finds templates by path type
	FindByPathType(ctx context.Context, pathType ProcessPathType) ([]*StageTemplate, error)

	// FindActive finds all active templates
	FindActive(ctx context.Context) ([]*StageTemplate, error)

	// FindDefault finds the default template
	FindDefault(ctx context.Context) (*StageTemplate, error)

	// Update updates a stage template
	Update(ctx context.Context, template *StageTemplate) error
}

// TaskRouteRepository defines the interface for task route persistence
type TaskRouteRepository interface {
	// Save saves a task route
	Save(ctx context.Context, route *TaskRoute) error

	// FindByID finds a route by its MongoDB ObjectID
	FindByID(ctx context.Context, id string) (*TaskRoute, error)

	// FindByRouteID finds a route by its route ID
	FindByRouteID(ctx context.Context, routeID string) (*TaskRoute, error)

	// FindByOrderID finds a route by order ID
	FindByOrderID(ctx context.Context, orderID string) (*TaskRoute, error)

	// FindByWaveID finds routes by wave ID
	FindByWaveID(ctx context.Context, waveID string) ([]*TaskRoute, error)

	// FindByStatus finds routes by status
	FindByStatus(ctx context.Context, status RouteStatus) ([]*TaskRoute, error)

	// FindInProgress finds all in-progress routes
	FindInProgress(ctx context.Context) ([]*TaskRoute, error)

	// Update updates a task route
	Update(ctx context.Context, route *TaskRoute) error
}
