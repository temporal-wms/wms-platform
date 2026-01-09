package activities

import (
	"log/slog"

	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"github.com/wms-platform/shared/pkg/middleware"
)

// ServiceClients wraps all service clients
type ServiceClients struct {
	*clients.ServiceClients
}

// ServiceClientsConfig holds configuration for service clients
type ServiceClientsConfig = clients.Config

// NewServiceClients creates a new ServiceClients wrapper
func NewServiceClients(config *ServiceClientsConfig) *ServiceClients {
	return &ServiceClients{
		ServiceClients: clients.NewServiceClients(config),
	}
}

// OrderActivities contains activities related to order operations
type OrderActivities struct {
	clients *ServiceClients
	logger  *slog.Logger
}

// NewOrderActivities creates a new OrderActivities instance
func NewOrderActivities(clients *ServiceClients, logger *slog.Logger) *OrderActivities {
	return &OrderActivities{
		clients: clients,
		logger:  logger,
	}
}

// InventoryActivities contains activities related to inventory operations
type InventoryActivities struct {
	clients *ServiceClients
	logger  *slog.Logger
}

// NewInventoryActivities creates a new InventoryActivities instance
func NewInventoryActivities(clients *ServiceClients, logger *slog.Logger) *InventoryActivities {
	return &InventoryActivities{
		clients: clients,
		logger:  logger,
	}
}

// RoutingActivities contains activities related to routing operations
type RoutingActivities struct {
	clients *ServiceClients
	logger  *slog.Logger
}

// NewRoutingActivities creates a new RoutingActivities instance
func NewRoutingActivities(clients *ServiceClients, logger *slog.Logger) *RoutingActivities {
	return &RoutingActivities{
		clients: clients,
		logger:  logger,
	}
}

// PickingActivities contains activities related to picking operations
type PickingActivities struct {
	clients *ServiceClients
	logger  *slog.Logger
}

// NewPickingActivities creates a new PickingActivities instance
func NewPickingActivities(clients *ServiceClients, logger *slog.Logger) *PickingActivities {
	return &PickingActivities{
		clients: clients,
		logger:  logger,
	}
}

// ConsolidationActivities contains activities related to consolidation operations
type ConsolidationActivities struct {
	clients *ServiceClients
	logger  *slog.Logger
}

// NewConsolidationActivities creates a new ConsolidationActivities instance
func NewConsolidationActivities(clients *ServiceClients, logger *slog.Logger) *ConsolidationActivities {
	return &ConsolidationActivities{
		clients: clients,
		logger:  logger,
	}
}

// PackingActivities contains activities related to packing operations
type PackingActivities struct {
	clients *ServiceClients
	logger  *slog.Logger
}

// NewPackingActivities creates a new PackingActivities instance
func NewPackingActivities(clients *ServiceClients, logger *slog.Logger) *PackingActivities {
	return &PackingActivities{
		clients: clients,
		logger:  logger,
	}
}

// ShippingActivities contains activities related to shipping operations
type ShippingActivities struct {
	clients *ServiceClients
	logger  *slog.Logger
}

// NewShippingActivities creates a new ShippingActivities instance
func NewShippingActivities(clients *ServiceClients, logger *slog.Logger) *ShippingActivities {
	return &ShippingActivities{
		clients: clients,
		logger:  logger,
	}
}

// ReprocessingActivities contains activities related to workflow reprocessing
type ReprocessingActivities struct {
	clients        *ServiceClients
	temporalClient interface{} // Temporal client for workflow operations
	logger         *slog.Logger
	failureMetrics *middleware.FailureMetrics
}

// NewReprocessingActivities creates a new ReprocessingActivities instance
func NewReprocessingActivities(clients *ServiceClients, temporalClient interface{}, logger *slog.Logger, failureMetrics *middleware.FailureMetrics) *ReprocessingActivities {
	return &ReprocessingActivities{
		clients:        clients,
		temporalClient: temporalClient,
		logger:         logger,
		failureMetrics: failureMetrics,
	}
}

// ProcessPathActivities contains activities related to process path routing
type ProcessPathActivities struct {
	clients *ServiceClients
	logger  *slog.Logger
}

// NewProcessPathActivities creates a new ProcessPathActivities instance
func NewProcessPathActivities(clients *ServiceClients, logger *slog.Logger) *ProcessPathActivities {
	return &ProcessPathActivities{
		clients: clients,
		logger:  logger,
	}
}

// UnitActivities contains activities related to unit-level tracking
type UnitActivities struct {
	clients *ServiceClients
	logger  *slog.Logger
}

// NewUnitActivities creates a new UnitActivities instance
func NewUnitActivities(clients *ServiceClients, logger *slog.Logger) *UnitActivities {
	return &UnitActivities{
		clients: clients,
		logger:  logger,
	}
}
