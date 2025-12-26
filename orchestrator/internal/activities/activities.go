package activities

import (
	"log/slog"

	"github.com/wms-platform/orchestrator/internal/activities/clients"
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
