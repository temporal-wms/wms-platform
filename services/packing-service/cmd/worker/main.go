package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wms-platform/packing-service/internal/activities"
	mongoRepo "github.com/wms-platform/packing-service/internal/infrastructure/mongodb"
	"github.com/wms-platform/packing-service/internal/workflows"
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

	logger.Info("Starting packing-service worker")

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
	eventFactory := cloudevents.NewEventFactory("/packing-service")

	// Initialize repository
	repo := mongoRepo.NewPackTaskRepository(mongoClient.Database(), eventFactory)

	// Initialize Temporal client
	temporalClient, err := temporal.NewClient(ctx, config.Temporal)
	if err != nil {
		logger.Error("Failed to create Temporal client", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()
	logger.Info("Connected to Temporal", "hostPort", config.Temporal.HostPort)

	// Create activities
	packingActivities := activities.NewPackingActivities(repo, logger)

	// Create worker
	workerOpts := temporal.DefaultWorkerOptions(temporal.TaskQueues.Packing)
	w := temporalClient.NewWorker(workerOpts)

	// Register workflow
	w.RegisterWorkflow(workflows.PackingWorkflow)
	logger.Info("Registered workflow", "workflow", "PackingWorkflow")

	// Register activities
	w.RegisterActivity(packingActivities.CreatePackTask)
	w.RegisterActivity(packingActivities.AssignPacker)
	w.RegisterActivity(packingActivities.StartPacking)
	w.RegisterActivity(packingActivities.VerifyItem)
	w.RegisterActivity(packingActivities.SelectPackaging)
	w.RegisterActivity(packingActivities.SealPackage)
	w.RegisterActivity(packingActivities.ApplyLabel)
	w.RegisterActivity(packingActivities.CompletePackTask)
	logger.Info("Registered activities")

	// Start worker in background
	go func() {
		if err := w.Run(nil); err != nil {
			logger.Error("Worker failed", "error", err)
			os.Exit(1)
		}
	}()
	logger.Info("Worker started", "taskQueue", temporal.TaskQueues.Packing)

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
			Database:       getEnv("MONGODB_DATABASE", "packing_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Temporal: &temporal.Config{
			HostPort:  getEnv("TEMPORAL_HOST", "localhost:7233"),
			Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
			Identity:  "packing-worker",
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
