package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/tracing"

	"github.com/wms-platform/services/channel-service/internal/api/handlers"
	"github.com/wms-platform/services/channel-service/internal/application"
	"github.com/wms-platform/services/channel-service/internal/domain"
	"github.com/wms-platform/services/channel-service/internal/infrastructure/adapters"
	mongoRepo "github.com/wms-platform/services/channel-service/internal/infrastructure/mongodb"
)

const serviceName = "channel-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting channel-service API")

	// Load configuration
	config := loadConfig()
	ctx := context.Background()

	// Initialize OpenTelemetry tracing
	tracingConfig := tracing.DefaultConfig(serviceName)
	tracingConfig.OTLPEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	tracingConfig.Environment = getEnv("ENVIRONMENT", "development")
	tracingConfig.Enabled = getEnv("TRACING_ENABLED", "true") == "true"

	tracerProvider, err := tracing.Initialize(ctx, tracingConfig)
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
	m := metrics.New(metricsConfig)
	logger.Info("Metrics initialized")

	// Initialize channel-specific metrics
	channelMetrics := NewChannelMetrics(m)

	// Initialize MongoDB with instrumentation
	mongoClient, err := mongodb.NewClient(ctx, config.MongoDB)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to MongoDB")
		os.Exit(1)
	}
	instrumentedMongo := mongodb.NewInstrumentedClient(mongoClient, m, logger)
	defer instrumentedMongo.Close(ctx)
	logger.Info("Connected to MongoDB", "database", config.MongoDB.Database)

	// Create repositories
	channelRepo := mongoRepo.NewChannelRepository(instrumentedMongo.Database())
	orderRepo := mongoRepo.NewChannelOrderRepository(instrumentedMongo.Database())
	syncJobRepo := mongoRepo.NewSyncJobRepository(instrumentedMongo.Database())
	outboxRepo := mongoRepo.NewOutboxRepository(instrumentedMongo.Database())

	// Initialize Kafka producer
	kafkaProducer := kafka.NewProducer(config.Kafka)
	instrumentedProducer := kafka.NewInstrumentedProducer(kafkaProducer, m, logger)
	defer instrumentedProducer.Close()
	logger.Info("Kafka producer initialized", "brokers", config.Kafka.Brokers)

	// Initialize and start outbox publisher
	outboxPublisher := outbox.NewPublisher(
		outboxRepo,
		instrumentedProducer,
		logger,
		m,
		outbox.DefaultPublisherConfig(),
	)
	if err := outboxPublisher.Start(ctx); err != nil {
		logger.WithError(err).Error("Failed to start outbox publisher")
	} else {
		defer outboxPublisher.Stop()
		logger.Info("Outbox publisher started")
	}

	// Create adapter factory and register adapters with instrumentation
	adapterFactory := domain.NewAdapterFactory()
	adapterFactory.Register(adapters.NewShopifyAdapter())
	adapterFactory.Register(adapters.NewAmazonAdapter())
	adapterFactory.Register(adapters.NewEbayAdapter())
	adapterFactory.Register(adapters.NewWooCommerceAdapter())
	logger.Info("Channel adapters registered", "adapters", []string{"shopify", "amazon", "ebay", "woocommerce"})

	// Create service
	channelService := application.NewChannelService(
		channelRepo,
		orderRepo,
		syncJobRepo,
		adapterFactory,
	)

	// Create handler with observability
	channelHandler := handlers.NewChannelHandler(channelService, logger, channelMetrics)

	// Setup Gin router with middleware
	router := gin.New()

	// Apply standard middleware
	middlewareConfig := middleware.DefaultConfig(serviceName, logger.Logger)
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
		return instrumentedMongo.HealthCheck(ctx)
	}))

	// Metrics endpoint
	router.GET("/metrics", middleware.MetricsEndpoint(m))

	// API v1 routes
	api := router.Group("/api/v1")
	channelHandler.RegisterRoutes(api)

	// Start server
	srv := &http.Server{
		Addr:         config.ServerAddr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("Server started", "addr", config.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Error("Server error")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	logger.Info("Server stopped")
}

// Config holds application configuration
type Config struct {
	ServerAddr string
	MongoDB    *mongodb.Config
	Kafka      *kafka.Config
}

func loadConfig() *Config {
	kafkaConfig := kafka.DefaultConfig()
	kafkaConfig.Brokers = strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	kafkaConfig.ClientID = serviceName

	return &Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8012"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "channel_service"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Kafka: kafkaConfig,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ChannelMetrics provides channel-specific metrics
type ChannelMetrics struct {
	metrics *metrics.Metrics
}

// NewChannelMetrics creates a new ChannelMetrics instance
func NewChannelMetrics(m *metrics.Metrics) *ChannelMetrics {
	return &ChannelMetrics{metrics: m}
}

// RecordSyncOperation records a channel sync operation
func (cm *ChannelMetrics) RecordSyncOperation(channel, syncType, status string, duration time.Duration) {
	// Use HTTP metrics to record sync operation
	statusCode := 200
	if status == "error" {
		statusCode = 500
	}
	cm.metrics.RecordHTTPRequest("POST", "/channel/"+channel+"/sync/"+syncType, statusCode, duration)
}

// RecordOrdersImported records the number of orders imported from a channel
func (cm *ChannelMetrics) RecordOrdersImported(channel string, count int) {
	// Use HTTP metrics to track imports
	for i := 0; i < count; i++ {
		cm.metrics.RecordHTTPRequest("POST", "/channel/"+channel+"/import", 200, 0)
	}
}

// RecordAPILatency records the latency of a channel API call
func (cm *ChannelMetrics) RecordAPILatency(channel, operation, status string, duration time.Duration) {
	// Use HTTP request duration to record API latency
	cm.metrics.HTTPRequestDuration.WithLabelValues(
		"channel-service",
		"GET",
		"/channel/"+channel+"/api/"+operation,
	).Observe(duration.Seconds())
}

// RecordWebhookReceived records a webhook received from a channel
func (cm *ChannelMetrics) RecordWebhookReceived(channel, topic, status string) {
	// Use HTTP metrics to track webhooks
	statusCode := 200
	if status == "error" {
		statusCode = 500
	}
	cm.metrics.RecordHTTPRequest("POST", "/channel/"+channel+"/webhook/"+topic, statusCode, 0)
}
