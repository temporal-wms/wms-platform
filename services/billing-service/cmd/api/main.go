package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/tracing"

	"github.com/wms-platform/services/billing-service/internal/api/handlers"
	"github.com/wms-platform/services/billing-service/internal/application"
	"github.com/wms-platform/services/billing-service/internal/domain"
	mongoRepo "github.com/wms-platform/services/billing-service/internal/infrastructure/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
)

const serviceName = "billing-service"

type instrumentedMongoClient interface {
	Database() *mongo.Database
	Close(context.Context) error
	HealthCheck(context.Context) error
}

type kafkaProducer interface {
	Close() error
}

type outboxPublisher interface {
	Start(context.Context) error
	Stop() error
}

type invoiceRepository interface {
	domain.InvoiceRepository
	GetOutboxRepository() outbox.Repository
}

var newInstrumentedMongoClient = func(ctx context.Context, cfg *mongodb.Config, m *metrics.Metrics, logger *logging.Logger) (instrumentedMongoClient, error) {
	client, err := mongodb.NewClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return mongodb.NewInstrumentedClient(client, m, logger), nil
}

var newInstrumentedKafkaProducer = func(cfg *kafka.Config, m *metrics.Metrics, logger *logging.Logger) kafkaProducer {
	producer := kafka.NewProducer(cfg)
	return kafka.NewInstrumentedProducer(producer, m, logger)
}

var newOutboxPublisher = func(repo outbox.Repository, producer kafkaProducer, logger *logging.Logger, m *metrics.Metrics, cfg *outbox.PublisherConfig) outboxPublisher {
	return outbox.NewPublisher(repo, producer.(*kafka.InstrumentedProducer), logger, m, cfg)
}

var newBillableActivityRepository = func(db *mongo.Database) domain.BillableActivityRepository {
	return mongoRepo.NewBillableActivityRepository(db)
}

var newInvoiceRepository = func(db *mongo.Database, eventFactory *cloudevents.EventFactory) invoiceRepository {
	return mongoRepo.NewInvoiceRepository(db, eventFactory)
}

var newStorageCalculationRepository = func(db *mongo.Database) domain.StorageCalculationRepository {
	return mongoRepo.NewStorageCalculationRepository(db)
}

var newBillingService = application.NewBillingService

var newBillingHandler = handlers.NewBillingHandler

var newMetrics = metrics.New

var initTracing = tracing.Initialize

var startHTTPServer = func(srv *http.Server) error {
	return srv.ListenAndServe()
}

func main() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	if err := run(context.Background(), signalCh); err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context, signalCh <-chan os.Signal) error {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting billing-service API")

	// Load configuration
	config := loadConfig()

	// Initialize OpenTelemetry tracing
	tracingConfig := tracing.DefaultConfig(serviceName)
	tracingConfig.OTLPEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	tracingConfig.Environment = getEnv("ENVIRONMENT", "development")
	tracingConfig.Enabled = getEnv("TRACING_ENABLED", "true") == "true"

	tracerProvider, err := initTracing(ctx, tracingConfig)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize tracing")
	} else if tracerProvider != nil {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
				logger.WithError(err).Error("Failed to shutdown tracer")
			}
		}()
		logger.Info("Tracing initialized", "endpoint", tracingConfig.OTLPEndpoint)
	}

	// Initialize Prometheus metrics
	metricsConfig := metrics.DefaultConfig(serviceName)
	m := newMetrics(metricsConfig)
	logger.Info("Metrics initialized")

	// Initialize MongoDB with instrumentation
	instrumentedMongo, err := newInstrumentedMongoClient(ctx, config.MongoDB, m, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to MongoDB")
		return err
	}
	defer instrumentedMongo.Close(ctx)
	logger.Info("Connected to MongoDB", "database", config.MongoDB.Database)

	// Initialize Kafka producer with instrumentation
	instrumentedProducer := newInstrumentedKafkaProducer(config.Kafka, m, logger)
	defer instrumentedProducer.Close()
	logger.Info("Kafka producer initialized", "brokers", config.Kafka.Brokers)

	// Initialize CloudEvents factory
	eventFactory := cloudevents.NewEventFactory("/billing-service")

	// Initialize repositories
	activityRepo := newBillableActivityRepository(instrumentedMongo.Database())
	invoiceRepo := newInvoiceRepository(instrumentedMongo.Database(), eventFactory)
	storageRepo := newStorageCalculationRepository(instrumentedMongo.Database())

	// Initialize and start outbox publisher
	outboxPublisher := newOutboxPublisher(
		invoiceRepo.GetOutboxRepository(),
		instrumentedProducer,
		logger,
		m,
		&outbox.PublisherConfig{
			PollInterval: 1 * time.Second,
			BatchSize:    100,
		},
	)
	if err := outboxPublisher.Start(ctx); err != nil {
		logger.WithError(err).Error("Failed to start outbox publisher")
		return err
	}
	defer func() {
		if err := outboxPublisher.Stop(); err != nil {
			logger.WithError(err).Warn("Failed to stop outbox publisher")
		}
	}()
	logger.Info("Outbox publisher started")

	// Initialize application service
	billingService := newBillingService(
		activityRepo,
		invoiceRepo,
		storageRepo,
		logger,
	)

	// Initialize handlers
	billingHandler := newBillingHandler(billingService, logger)

	// Setup Gin router with middleware
	router := gin.New()

	// Apply standard middleware
	middlewareConfig := middleware.DefaultConfig(serviceName, logger.Logger)
	middleware.Setup(router, middlewareConfig)

	// Add metrics middleware
	router.Use(middleware.MetricsMiddleware(m))

	// Add tracing middleware
	router.Use(middleware.SimpleTracingMiddleware(serviceName))

	// Add tenant middleware
	router.Use(middleware.TenantAuth(middleware.DefaultTenantAuthConfig()))

	// Handle 404 and 405 errors
	router.NoRoute(middleware.NoRoute())
	router.NoMethod(middleware.NoMethod())

	// Health check endpoints
	router.GET("/health", middleware.HealthCheck(serviceName))
	router.GET("/ready", middleware.ReadinessCheck(serviceName, func() error {
		return instrumentedMongo.HealthCheck(ctx)
	}))

	// Metrics endpoint
	router.GET("/metrics", middleware.MetricsEndpoint(m))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Activities
		activities := v1.Group("/activities")
		{
			activities.POST("", billingHandler.RecordActivity)
			activities.POST("/batch", billingHandler.RecordActivities)
			activities.GET("/:activityId", billingHandler.GetActivity)
		}

		// Seller-scoped activities
		sellers := v1.Group("/sellers")
		{
			sellers.GET("/:sellerId/activities", billingHandler.ListActivities)
			sellers.GET("/:sellerId/activities/summary", billingHandler.GetActivitySummary)
			sellers.GET("/:sellerId/invoices", billingHandler.ListInvoices)
		}

		// Invoices
		invoices := v1.Group("/invoices")
		{
			invoices.POST("", billingHandler.CreateInvoice)
			invoices.GET("/:invoiceId", billingHandler.GetInvoice)
			invoices.PUT("/:invoiceId/finalize", billingHandler.FinalizeInvoice)
			invoices.PUT("/:invoiceId/pay", billingHandler.MarkInvoicePaid)
			invoices.PUT("/:invoiceId/void", billingHandler.VoidInvoice)
		}

		// Fee calculation
		fees := v1.Group("/fees")
		{
			fees.POST("/calculate", billingHandler.CalculateFees)
		}

		// Storage calculation
		storage := v1.Group("/storage")
		{
			storage.POST("/calculate", billingHandler.RecordStorage)
		}
	}

	// Start server
	srv := &http.Server{
		Addr:         config.ServerAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Graceful shutdown
	go func() {
		if err := startHTTPServer(srv); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.WithError(err).Error("Server error")
		}
	}()
	logger.Info("Server started", "addr", config.ServerAddr)

	// Wait for interrupt signal
	<-signalCh
	logger.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	logger.Info("Server stopped")
	return nil
}

// Config holds application configuration
type Config struct {
	ServerAddr string
	MongoDB    *mongodb.Config
	Kafka      *kafka.Config
}

func loadConfig() *Config {
	return &Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8018"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "billing_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Kafka: &kafka.Config{
			Brokers:       []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			ConsumerGroup: serviceName,
			ClientID:      serviceName,
			BatchSize:     100,
			BatchTimeout:  10 * time.Millisecond,
			RequiredAcks:  -1,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
