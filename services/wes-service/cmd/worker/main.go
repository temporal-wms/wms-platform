package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/temporal"
	"github.com/wms-platform/wes-service/internal/activities"
	"github.com/wms-platform/wes-service/internal/application"
	"github.com/wms-platform/wes-service/internal/domain"
	"github.com/wms-platform/wes-service/internal/infrastructure/mongodb"
	"github.com/wms-platform/wes-service/internal/workflows"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Setup logger with shared logging package
	logConfig := logging.DefaultConfig("wes-worker")
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	loggerWrapper := logging.New(logConfig)
	logger := loggerWrapper.Logger
	slog.SetDefault(logger)

	logger.Info("Starting WES Temporal Worker")

	// Get configuration from environment
	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017")
	dbName := getEnv("MONGODB_DATABASE", "wes_db")
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")
	temporalNamespace := getEnv("TEMPORAL_NAMESPACE", "default")
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")

	// Service URLs for clients
	laborServiceURL := getEnv("LABOR_SERVICE_URL", "http://localhost:8010")
	pickingServiceURL := getEnv("PICKING_SERVICE_URL", "http://localhost:8012")
	wallingServiceURL := getEnv("WALLING_SERVICE_URL", "http://localhost:8017")
	packingServiceURL := getEnv("PACKING_SERVICE_URL", "http://localhost:8014")
	processPathServiceURL := getEnv("PROCESS_PATH_SERVICE_URL", "http://localhost:8011")

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetMinPoolSize(5).
		SetMaxPoolSize(50)

	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Error("Failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			logger.Error("Failed to disconnect from MongoDB", "error", err)
		}
	}()

	// Ping MongoDB
	if err := mongoClient.Ping(ctx, nil); err != nil {
		logger.Error("Failed to ping MongoDB", "error", err)
		os.Exit(1)
	}
	logger.Info("Connected to MongoDB", "database", dbName)

	// Get database
	db := mongoClient.Database(dbName)

	// Initialize metrics
	metricsConfig := metrics.DefaultConfig("wes-worker")
	m := metrics.New(metricsConfig)

	// Initialize Kafka producer with instrumentation
	kafkaConfig := &kafka.Config{
		Brokers:       []string{kafkaBrokers},
		ConsumerGroup: "wes-worker",
		ClientID:      "wes-worker",
		BatchSize:     100,
		BatchTimeout:  10 * time.Millisecond,
		RequiredAcks:  -1,
	}
	kafkaProducer := kafka.NewProducer(kafkaConfig)
	instrumentedProducer := kafka.NewInstrumentedProducer(kafkaProducer, m, loggerWrapper)
	defer instrumentedProducer.Close()
	logger.Info("Kafka producer initialized", "brokers", kafkaBrokers)

	// Initialize CloudEvents factory
	eventFactory := cloudevents.NewEventFactory("/wes-service")

	// Create repositories
	templateRepo := mongodb.NewStageTemplateRepository(db, eventFactory)
	routeRepo := mongodb.NewTaskRouteRepository(db, eventFactory)

	// Seed default templates if needed
	if err := seedDefaultTemplates(ctx, templateRepo, logger); err != nil {
		logger.Warn("Failed to seed default templates", "error", err)
	}

	// Create service clients
	processPathClient := application.NewProcessPathClient(processPathServiceURL)
	laborClient := activities.NewLaborServiceClient(laborServiceURL)
	pickingClient := activities.NewPickingServiceClient(pickingServiceURL)
	wallingClient := activities.NewWallingServiceClient(wallingServiceURL)
	packingClient := activities.NewPackingServiceClient(packingServiceURL)

	// Create application service
	wesService := application.NewWESApplicationService(
		templateRepo,
		routeRepo,
		processPathClient,
		instrumentedProducer,
		eventFactory,
		logger,
	)

	// Create WES activities
	wesActivities := activities.NewWESActivities(
		wesService,
		laborClient,
		pickingClient,
		wallingClient,
		packingClient,
	)

	// Create Temporal client
	temporalClient, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: temporalNamespace,
	})
	if err != nil {
		logger.Error("Failed to create Temporal client", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()

	logger.Info("Connected to Temporal", "host", temporalHost, "namespace", temporalNamespace)

	// Create worker
	w := worker.New(temporalClient, temporal.TaskQueues.WESExecution, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflows.WESExecutionWorkflow)

	// Register activities
	w.RegisterActivity(wesActivities.ResolveExecutionPlan)
	w.RegisterActivity(wesActivities.CreateTaskRoute)
	w.RegisterActivity(wesActivities.AssignWorkerToStage)
	w.RegisterActivity(wesActivities.StartStage)
	w.RegisterActivity(wesActivities.CompleteStage)
	w.RegisterActivity(wesActivities.FailStage)
	w.RegisterActivity(wesActivities.ExecuteWallingTask)
	w.RegisterActivity(wesActivities.ExecutePickingTask)
	w.RegisterActivity(wesActivities.ExecuteConsolidationTask)
	w.RegisterActivity(wesActivities.ExecutePackingTask)

	// Start worker
	logger.Info("Starting WES worker", "taskQueue", temporal.TaskQueues.WESExecution)
	if err := w.Run(worker.InterruptCh()); err != nil {
		logger.Error("Failed to start worker", "error", err)
		os.Exit(1)
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// seedDefaultTemplates seeds default stage templates if they don't exist
func seedDefaultTemplates(ctx context.Context, repo *mongodb.StageTemplateRepository, logger *slog.Logger) error {
	// Check if templates already exist
	templates, err := repo.FindActive(ctx)
	if err != nil {
		return err
	}
	if len(templates) > 0 {
		logger.Info("Stage templates already exist", "count", len(templates))
		return nil
	}

	logger.Info("Seeding default stage templates")

	// Create default templates
	defaults := []*domain.StageTemplate{
		domain.DefaultPickPackTemplate(),
		domain.DefaultPickWallPackTemplate(),
		domain.DefaultPickConsolidatePackTemplate(),
	}

	for _, t := range defaults {
		if err := repo.Save(ctx, t); err != nil {
			return err
		}
		logger.Info("Created default template", "templateId", t.TemplateID, "pathType", t.PathType)
	}

	// Mark the first one as default
	defaults[0].SetDefault()
	if err := repo.Update(ctx, defaults[0]); err != nil {
		return err
	}

	logger.Info("Default stage templates seeded successfully")
	return nil
}
