package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/wms-platform/orchestrator/internal/activities"
	"github.com/wms-platform/orchestrator/internal/workflows"
	"github.com/wms-platform/shared/pkg/temporal"
)

func main() {
	// Setup JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting orchestrator worker")

	// Load configuration
	config := loadConfig()

	// Initialize Temporal client
	ctx := context.Background()
	temporalClient, err := temporal.NewClient(ctx, config.Temporal)
	if err != nil {
		logger.Error("Failed to create Temporal client", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()
	logger.Info("Connected to Temporal", "hostPort", config.Temporal.HostPort, "namespace", config.Temporal.Namespace)

	// Initialize HTTP clients for service communication
	serviceClients := activities.NewServiceClients(&activities.ServiceClientsConfig{
		OrderServiceURL:         config.OrderServiceURL,
		InventoryServiceURL:     config.InventoryServiceURL,
		RoutingServiceURL:       config.RoutingServiceURL,
		PickingServiceURL:       config.PickingServiceURL,
		ConsolidationServiceURL: config.ConsolidationServiceURL,
		PackingServiceURL:       config.PackingServiceURL,
		ShippingServiceURL:      config.ShippingServiceURL,
		LaborServiceURL:         config.LaborServiceURL,
		WavingServiceURL:        config.WavingServiceURL,
	})

	// Create activities with service clients
	orderActivities := activities.NewOrderActivities(serviceClients, logger)
	inventoryActivities := activities.NewInventoryActivities(serviceClients, logger)
	routingActivities := activities.NewRoutingActivities(serviceClients, logger)

	// Create worker
	workerOpts := temporal.DefaultWorkerOptions(temporal.TaskQueues.Orchestrator)
	w := temporalClient.NewWorker(workerOpts)

	// Register workflows
	w.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	w.RegisterWorkflow(workflows.OrderCancellationWorkflow)
	w.RegisterWorkflow(workflows.PickingWorkflow)
	w.RegisterWorkflow(workflows.ConsolidationWorkflow)
	w.RegisterWorkflow(workflows.PackingWorkflow)
	w.RegisterWorkflow(workflows.ShippingWorkflow)
	logger.Info("Registered workflows", "workflows", []string{
		"OrderFulfillmentWorkflow",
		"OrderCancellationWorkflow",
		"PickingWorkflow",
		"ConsolidationWorkflow",
		"PackingWorkflow",
		"ShippingWorkflow",
	})

	// Register activities
	w.RegisterActivity(orderActivities.ValidateOrder)
	w.RegisterActivity(orderActivities.CancelOrder)
	w.RegisterActivity(orderActivities.NotifyCustomerCancellation)
	w.RegisterActivity(inventoryActivities.ReleaseInventoryReservation)
	w.RegisterActivity(routingActivities.CalculateRoute)
	logger.Info("Registered activities", "activities", []string{
		"ValidateOrder",
		"CancelOrder",
		"NotifyCustomerCancellation",
		"ReleaseInventoryReservation",
		"CalculateRoute",
	})

	// Start worker in background
	go func() {
		if err := w.Run(nil); err != nil {
			logger.Error("Worker failed", "error", err)
			os.Exit(1)
		}
	}()
	logger.Info("Worker started", "taskQueue", temporal.TaskQueues.Orchestrator)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down worker...")

	w.Stop()
	logger.Info("Worker stopped")
}

// Config holds application configuration
type Config struct {
	Temporal                *temporal.Config
	OrderServiceURL         string
	InventoryServiceURL     string
	RoutingServiceURL       string
	PickingServiceURL       string
	ConsolidationServiceURL string
	PackingServiceURL       string
	ShippingServiceURL      string
	LaborServiceURL         string
	WavingServiceURL        string
}

func loadConfig() *Config {
	return &Config{
		Temporal: &temporal.Config{
			HostPort:  getEnv("TEMPORAL_HOST", "localhost:7233"),
			Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
			Identity:  "orchestrator-worker",
		},
		OrderServiceURL:         getEnv("ORDER_SERVICE_URL", "http://localhost:8001"),
		InventoryServiceURL:     getEnv("INVENTORY_SERVICE_URL", "http://localhost:8008"),
		RoutingServiceURL:       getEnv("ROUTING_SERVICE_URL", "http://localhost:8003"),
		PickingServiceURL:       getEnv("PICKING_SERVICE_URL", "http://localhost:8004"),
		ConsolidationServiceURL: getEnv("CONSOLIDATION_SERVICE_URL", "http://localhost:8005"),
		PackingServiceURL:       getEnv("PACKING_SERVICE_URL", "http://localhost:8006"),
		ShippingServiceURL:      getEnv("SHIPPING_SERVICE_URL", "http://localhost:8007"),
		LaborServiceURL:         getEnv("LABOR_SERVICE_URL", "http://localhost:8009"),
		WavingServiceURL:        getEnv("WAVING_SERVICE_URL", "http://localhost:8002"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
