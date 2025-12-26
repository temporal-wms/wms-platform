package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wms-platform/picking-service/internal/activities"
	mongoRepo "github.com/wms-platform/picking-service/internal/infrastructure/mongodb"
	"github.com/wms-platform/picking-service/internal/workflows"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/temporal"
)

func main() {
	// Setup JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting picking-service worker")

	// Load configuration
	config := loadConfig()

	// Initialize MongoDB
	ctx := context.Background()
	mongoClient, err := mongodb.NewClient(ctx, config.MongoDB)
	if err != nil {
		logger.Error("Failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer mongoClient.Close(ctx)
	logger.Info("Connected to MongoDB", "database", config.MongoDB.Database)

	// Initialize CloudEvents factory
	eventFactory := cloudevents.NewEventFactory("/picking-service")

	// Initialize repository
	repo := mongoRepo.NewPickTaskRepository(mongoClient.Database(), eventFactory)

	// Initialize Temporal client
	temporalClient, err := temporal.NewClient(ctx, config.Temporal)
	if err != nil {
		logger.Error("Failed to create Temporal client", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()
	logger.Info("Connected to Temporal", "hostPort", config.Temporal.HostPort)

	// Create activities
	pickingActivities := activities.NewPickingActivities(repo, logger)

	// Create worker
	workerOpts := temporal.DefaultWorkerOptions(temporal.TaskQueues.Picking)
	w := temporalClient.NewWorker(workerOpts)

	// Register workflow
	w.RegisterWorkflow(workflows.PickingWorkflow)
	logger.Info("Registered workflow", "workflow", "PickingWorkflow")

	// Register activities
	w.RegisterActivity(pickingActivities.CreatePickTask)
	w.RegisterActivity(pickingActivities.CompletePickTask)
	w.RegisterActivity(pickingActivities.AssignWorker)
	w.RegisterActivity(pickingActivities.StartPicking)
	w.RegisterActivity(pickingActivities.ConfirmPick)
	w.RegisterActivity(pickingActivities.ReportException)
	logger.Info("Registered activities")

	// Start worker in background
	go func() {
		if err := w.Run(nil); err != nil {
			logger.Error("Worker failed", "error", err)
			os.Exit(1)
		}
	}()
	logger.Info("Worker started", "taskQueue", temporal.TaskQueues.Picking)

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
	MongoDB  *mongodb.Config
	Temporal *temporal.Config
}

func loadConfig() *Config {
	return &Config{
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "picking_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Temporal: &temporal.Config{
			HostPort:  getEnv("TEMPORAL_HOST", "localhost:7233"),
			Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
			Identity:  "picking-worker",
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
