package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/idempotency"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/tracing"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/wms-platform/facility-service/internal/api/handlers"
	"github.com/wms-platform/facility-service/internal/application"
	mongoRepo "github.com/wms-platform/facility-service/internal/infrastructure/mongodb"
)

const serviceName = "facility-service"

func main() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	if err := run(context.Background(), loadConfig(), appDependencies{}, signalCh); err != nil {
		os.Exit(1)
	}
}

type tracerProvider interface {
	Shutdown(ctx context.Context) error
}

type instrumentedMongo interface {
	Database() *mongo.Database
	Close(ctx context.Context) error
	HealthCheck(ctx context.Context) error
}

type outboxPublisher interface {
	Start(ctx context.Context) error
	Stop() error
}

type httpServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

type stationRepository interface {
	application.StationRepository
	GetOutboxRepository() outbox.Repository
}

type appDependencies struct {
	initTracing             func(ctx context.Context, cfg *tracing.Config) (tracerProvider, error)
	newMetrics              func(cfg *metrics.Config) *metrics.Metrics
	newBusinessMetrics      func(m *metrics.Metrics) *middleware.BusinessMetrics
	newMongoClient          func(ctx context.Context, cfg *mongodb.Config) (*mongodb.Client, error)
	newInstrumentedMongo    func(client *mongodb.Client, m *metrics.Metrics, logger *logging.Logger) instrumentedMongo
	initIdempotencyIndexes  func(ctx context.Context, db *mongo.Database) error
	newKafkaProducer        func(cfg *kafka.Config) *kafka.Producer
	newInstrumentedProducer func(p *kafka.Producer, m *metrics.Metrics, logger *logging.Logger) *kafka.InstrumentedProducer
	closeInstrumentedProd   func(p *kafka.InstrumentedProducer) error
	newEventFactory         func(source string) *cloudevents.EventFactory
	newStationRepository    func(db *mongo.Database, factory *cloudevents.EventFactory) stationRepository
	newIdempotencyKeyRepo   func(db *mongo.Database) idempotency.KeyRepository
	newOutboxPublisher      func(repo outbox.Repository, producer *kafka.InstrumentedProducer, logger *logging.Logger, m *metrics.Metrics, cfg *outbox.PublisherConfig) outboxPublisher
	newStationService       func(repo application.StationRepository, producer *kafka.InstrumentedProducer, factory *cloudevents.EventFactory, logger *logging.Logger) handlers.StationService
	newHTTPServer           func(addr string, handler http.Handler) httpServer
}

func defaultDependencies() appDependencies {
	return appDependencies{
		initTracing: func(ctx context.Context, cfg *tracing.Config) (tracerProvider, error) {
			return tracing.Initialize(ctx, cfg)
		},
		newMetrics: metrics.New,
		newBusinessMetrics: func(m *metrics.Metrics) *middleware.BusinessMetrics {
			return middleware.NewBusinessMetrics(m)
		},
		newMongoClient: mongodb.NewClient,
		newInstrumentedMongo: func(client *mongodb.Client, m *metrics.Metrics, logger *logging.Logger) instrumentedMongo {
			return mongodb.NewInstrumentedClient(client, m, logger)
		},
		initIdempotencyIndexes: idempotency.InitializeIndexes,
		newKafkaProducer:       kafka.NewProducer,
		newInstrumentedProducer: func(p *kafka.Producer, m *metrics.Metrics, logger *logging.Logger) *kafka.InstrumentedProducer {
			return kafka.NewInstrumentedProducer(p, m, logger)
		},
		closeInstrumentedProd: func(p *kafka.InstrumentedProducer) error { return p.Close() },
		newEventFactory:       cloudevents.NewEventFactory,
		newStationRepository: func(db *mongo.Database, factory *cloudevents.EventFactory) stationRepository {
			return mongoRepo.NewStationRepository(db, factory)
		},
		newIdempotencyKeyRepo: func(db *mongo.Database) idempotency.KeyRepository {
			return idempotency.NewMongoKeyRepository(db)
		},
		newOutboxPublisher: func(repo outbox.Repository, producer *kafka.InstrumentedProducer, logger *logging.Logger, m *metrics.Metrics, cfg *outbox.PublisherConfig) outboxPublisher {
			return outbox.NewPublisher(repo, producer, logger, m, cfg)
		},
		newStationService: func(repo application.StationRepository, producer *kafka.InstrumentedProducer, factory *cloudevents.EventFactory, logger *logging.Logger) handlers.StationService {
			return application.NewStationApplicationService(repo, producer, factory, logger)
		},
		newHTTPServer: func(addr string, handler http.Handler) httpServer {
			return &http.Server{
				Addr:         addr,
				Handler:      handler,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 30 * time.Second,
			}
		},
	}
}

func (d appDependencies) withDefaults() appDependencies {
	def := defaultDependencies()
	if d.initTracing == nil {
		d.initTracing = def.initTracing
	}
	if d.newMetrics == nil {
		d.newMetrics = def.newMetrics
	}
	if d.newBusinessMetrics == nil {
		d.newBusinessMetrics = def.newBusinessMetrics
	}
	if d.newMongoClient == nil {
		d.newMongoClient = def.newMongoClient
	}
	if d.newInstrumentedMongo == nil {
		d.newInstrumentedMongo = def.newInstrumentedMongo
	}
	if d.initIdempotencyIndexes == nil {
		d.initIdempotencyIndexes = def.initIdempotencyIndexes
	}
	if d.newKafkaProducer == nil {
		d.newKafkaProducer = def.newKafkaProducer
	}
	if d.newInstrumentedProducer == nil {
		d.newInstrumentedProducer = def.newInstrumentedProducer
	}
	if d.closeInstrumentedProd == nil {
		d.closeInstrumentedProd = def.closeInstrumentedProd
	}
	if d.newEventFactory == nil {
		d.newEventFactory = def.newEventFactory
	}
	if d.newStationRepository == nil {
		d.newStationRepository = def.newStationRepository
	}
	if d.newIdempotencyKeyRepo == nil {
		d.newIdempotencyKeyRepo = def.newIdempotencyKeyRepo
	}
	if d.newOutboxPublisher == nil {
		d.newOutboxPublisher = def.newOutboxPublisher
	}
	if d.newStationService == nil {
		d.newStationService = def.newStationService
	}
	if d.newHTTPServer == nil {
		d.newHTTPServer = def.newHTTPServer
	}
	return d
}

func run(ctx context.Context, config *Config, deps appDependencies, signalCh <-chan os.Signal) error {
	deps = deps.withDefaults()
	if config == nil {
		config = loadConfig()
	}

	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting facility-service API")

	// Initialize OpenTelemetry tracing
	tracingConfig := tracing.DefaultConfig(serviceName)
	tracingConfig.OTLPEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	tracingConfig.Environment = getEnv("ENVIRONMENT", "development")
	tracingConfig.Enabled = getEnv("TRACING_ENABLED", "true") == "true"

	tracerProvider, err := deps.initTracing(ctx, tracingConfig)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize tracing")
		// Continue without tracing - don't exit
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
	m := deps.newMetrics(metricsConfig)
	logger.Info("Metrics initialized")

	// Initialize business metrics helper (like order-service)
	businessMetrics := deps.newBusinessMetrics(m)

	// Initialize MongoDB with instrumentation
	mongoClient, err := deps.newMongoClient(ctx, config.MongoDB)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to MongoDB")
		return fmt.Errorf("failed to connect to mongodb: %w", err)
	}
	instrumentedMongo := deps.newInstrumentedMongo(mongoClient, m, logger)
	if instrumentedMongo != nil {
		defer instrumentedMongo.Close(ctx)
	}
	logger.Info("Connected to MongoDB", "database", config.MongoDB.Database)

	// Initialize idempotency indexes
	if instrumentedMongo != nil {
		if err := deps.initIdempotencyIndexes(ctx, instrumentedMongo.Database()); err != nil {
			logger.WithError(err).Warn("Failed to initialize idempotency indexes")
		} else {
			logger.Info("Idempotency indexes initialized")
		}
	}

	// Initialize Kafka producer with instrumentation
	kafkaProducer := deps.newKafkaProducer(config.Kafka)
	instrumentedProducer := deps.newInstrumentedProducer(kafkaProducer, m, logger)
	if instrumentedProducer != nil {
		defer func() {
			_ = deps.closeInstrumentedProd(instrumentedProducer)
		}()
	}
	logger.Info("Kafka producer initialized", "brokers", config.Kafka.Brokers)

	// Initialize CloudEvents factory
	eventFactory := deps.newEventFactory("/facility-service")

	// Initialize repositories with instrumented client and event factory
	var db *mongo.Database
	if instrumentedMongo != nil {
		db = instrumentedMongo.Database()
	}
	stationRepo := deps.newStationRepository(db, eventFactory)

	// Initialize idempotency repository
	idempotencyKeyRepo := deps.newIdempotencyKeyRepo(db)
	logger.Info("Idempotency repositories initialized")

	// Initialize and start outbox publisher
	outboxPublisher := deps.newOutboxPublisher(
		stationRepo.GetOutboxRepository(),
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
		return fmt.Errorf("failed to start outbox publisher: %w", err)
	}
	defer func() {
		_ = outboxPublisher.Stop()
	}()
	logger.Info("Outbox publisher started")

	// Initialize application services
	stationService := deps.newStationService(stationRepo, instrumentedProducer, eventFactory, logger)

	// Setup Gin router with middleware
	router := gin.New()

	// Apply standard middleware (includes recovery, request ID, correlation, logging, error handling)
	middlewareConfig := middleware.DefaultConfig(serviceName, logger.Logger)

	// Initialize idempotency metrics
	idempotencyMetrics := idempotency.NewMetrics(nil)

	// Configure idempotency middleware
	middlewareConfig.IdempotencyConfig = &idempotency.Config{
		ServiceName:     serviceName,
		Repository:      idempotencyKeyRepo,
		RequireKey:      false,
		OnlyMutating:    true,
		MaxKeyLength:    255,
		LockTimeout:     5 * time.Minute,
		RetentionPeriod: 24 * time.Hour,
		MaxResponseSize: 1024 * 1024,
		Metrics:         idempotencyMetrics,
	}

	middleware.Setup(router, middlewareConfig)

	// Add metrics middleware
	router.Use(middleware.MetricsMiddleware(m))

	// Add tracing middleware
	router.Use(middleware.SimpleTracingMiddleware(serviceName))

	// Handle 404 and 405 errors
	router.NoRoute(middleware.NoRoute())
	router.NoMethod(middleware.NoMethod())

	// Health check endpoints
	router.GET("/health", middleware.HealthCheck(serviceName))
	router.GET("/ready", middleware.ReadinessCheck(serviceName, func() error {
		if instrumentedMongo == nil {
			return fmt.Errorf("mongo client unavailable")
		}
		return instrumentedMongo.HealthCheck(ctx)
	}))

	// Metrics endpoint
	router.GET("/metrics", middleware.MetricsEndpoint(m))

	// API v1 routes
	apiV1 := router.Group("/api/v1")

	// Station routes (for process path routing and facility management)
	stationHandlers := handlers.NewStationHandlers(stationService, logger, businessMetrics)
	stationHandlers.RegisterRoutes(apiV1)

	// Start server
	srv := deps.newHTTPServer(config.ServerAddr, router)

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
		}
	}()
	logger.Info("Server started", "addr", config.ServerAddr)

	// Wait for interrupt signal
	if signalCh == nil {
		signalCh = make(chan os.Signal, 1)
	}
	select {
	case <-signalCh:
	case <-ctx.Done():
	}
	logger.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
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
		ServerAddr: getEnv("SERVER_ADDR", ":8010"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "facility_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Kafka: &kafka.Config{
			Brokers:       []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			ConsumerGroup: "facility-service",
			ClientID:      "facility-service",
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
