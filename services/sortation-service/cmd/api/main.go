package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/idempotency"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/tracing"

	"github.com/wms-platform/services/sortation-service/internal/application"
	"github.com/wms-platform/services/sortation-service/internal/domain"
	mongoRepo "github.com/wms-platform/services/sortation-service/internal/infrastructure/mongodb"
)

const serviceName = "sortation-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting sortation-service API")

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

	// Initialize MongoDB with instrumentation
	mongoClient, err := mongodb.NewClient(ctx, config.MongoDB)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to MongoDB")
		os.Exit(1)
	}
	instrumentedMongo := mongodb.NewInstrumentedClient(mongoClient, m, logger)
	defer instrumentedMongo.Close(ctx)
	logger.Info("Connected to MongoDB", "database", config.MongoDB.Database)

	// Initialize idempotency indexes
	if err := idempotency.InitializeIndexes(ctx, instrumentedMongo.Database()); err != nil {
		logger.WithError(err).Warn("Failed to initialize idempotency indexes")
	} else {
		logger.Info("Idempotency indexes initialized")
	}

	// Initialize Kafka producer with instrumentation
	kafkaProducer := kafka.NewProducer(config.Kafka)
	instrumentedProducer := kafka.NewInstrumentedProducer(kafkaProducer, m, logger)
	defer instrumentedProducer.Close()
	logger.Info("Kafka producer initialized", "brokers", config.Kafka.Brokers)

	// Initialize CloudEvents factory
	eventFactory := cloudevents.NewEventFactory("/sortation-service")

	// Initialize repositories
	batchRepo := mongoRepo.NewSortationBatchRepository(instrumentedMongo.Database(), eventFactory)

	// Initialize idempotency repository
	idempotencyKeyRepo := idempotency.NewMongoKeyRepository(instrumentedMongo.Database())
	logger.Info("Idempotency repositories initialized")

	// Initialize and start outbox publisher
	outboxPublisher := outbox.NewPublisher(
		batchRepo.GetOutboxRepository(),
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
		os.Exit(1)
	}
	defer outboxPublisher.Stop()
	logger.Info("Outbox publisher started")

	// Initialize application service
	sortationService := application.NewSortationService(batchRepo, logger)

	// Setup Gin router with middleware
	router := gin.New()

	// Initialize idempotency metrics
	idempotencyMetrics := idempotency.NewMetrics(nil)

	// Apply standard middleware
	middlewareConfig := middleware.DefaultConfig(serviceName, logger.Logger)

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
		return instrumentedMongo.HealthCheck(ctx)
	}))

	// Metrics endpoint
	router.GET("/metrics", middleware.MetricsEndpoint(m))

	// API v1 routes with tenant context required
	api := router.Group("/api/v1/batches")
	api.Use(middleware.RequireTenantAuth()) // All API routes require tenant headers
	{
		api.GET("", listBatchesHandler(sortationService, logger))
		api.POST("", createBatchHandler(sortationService, logger))
		api.GET("/status/:status", getBatchesByStatusHandler(sortationService, logger))
		api.GET("/ready", getReadyBatchesHandler(sortationService, logger))
		api.GET("/:batchId", getBatchHandler(sortationService, logger))
		api.POST("/:batchId/packages", addPackageHandler(sortationService, logger))
		api.POST("/:batchId/sort", sortPackageHandler(sortationService, logger))
		api.POST("/:batchId/ready", markReadyHandler(sortationService, logger))
		api.POST("/:batchId/dispatch", dispatchBatchHandler(sortationService, logger))
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
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Error("Server error")
		}
	}()
	logger.Info("Server started", "addr", config.ServerAddr)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

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
	return &Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8012"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "sortation_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Kafka: &kafka.Config{
			Brokers:       []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			ConsumerGroup: "sortation-service",
			ClientID:      "sortation-service",
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

// HTTP Handlers

func createBatchHandler(service *application.SortationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			SortationCenter  string `json:"sortationCenter" binding:"required"`
			DestinationGroup string `json:"destinationGroup" binding:"required"`
			CarrierID        string `json:"carrierId" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.CreateBatchCommand{
			SortationCenter:  req.SortationCenter,
			DestinationGroup: req.DestinationGroup,
			CarrierID:        req.CarrierID,
		}

		batch, err := service.CreateBatch(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusCreated, batch)
	}
}

func getBatchHandler(service *application.SortationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		batchID := c.Param("batchId")

		batch, err := service.GetBatch(c.Request.Context(), batchID)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, batch)
	}
}

func listBatchesHandler(service *application.SortationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		limitStr := c.Query("limit")
		limit := 50
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		batches, err := service.ListBatches(c.Request.Context(), limit)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"batches": batches, "total": len(batches)})
	}
}

func getBatchesByStatusHandler(service *application.SortationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		status := domain.SortationStatus(c.Param("status"))

		batches, err := service.GetBatchesByStatus(c.Request.Context(), status)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"batches": batches, "total": len(batches)})
	}
}

func getReadyBatchesHandler(service *application.SortationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		carrierID := c.Query("carrierId")
		limitStr := c.Query("limit")
		limit := 20
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		batches, err := service.GetReadyBatches(c.Request.Context(), carrierID, limit)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"batches": batches, "total": len(batches)})
	}
}

func addPackageHandler(service *application.SortationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		batchID := c.Param("batchId")

		var req struct {
			PackageID      string  `json:"packageId" binding:"required"`
			OrderID        string  `json:"orderId" binding:"required"`
			TrackingNumber string  `json:"trackingNumber"`
			Destination    string  `json:"destination" binding:"required"`
			CarrierID      string  `json:"carrierId"`
			Weight         float64 `json:"weight"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.AddPackageCommand{
			BatchID:        batchID,
			PackageID:      req.PackageID,
			OrderID:        req.OrderID,
			TrackingNumber: req.TrackingNumber,
			Destination:    req.Destination,
			CarrierID:      req.CarrierID,
			Weight:         req.Weight,
		}

		batch, err := service.AddPackage(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, batch)
	}
}

func sortPackageHandler(service *application.SortationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		batchID := c.Param("batchId")

		var req struct {
			PackageID string `json:"packageId" binding:"required"`
			ChuteID   string `json:"chuteId" binding:"required"`
			WorkerID  string `json:"workerId" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.SortPackageCommand{
			BatchID:   batchID,
			PackageID: req.PackageID,
			ChuteID:   req.ChuteID,
			WorkerID:  req.WorkerID,
		}

		batch, err := service.SortPackage(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, batch)
	}
}

func markReadyHandler(service *application.SortationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		batchID := c.Param("batchId")

		cmd := application.MarkReadyCommand{BatchID: batchID}

		batch, err := service.MarkReady(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, batch)
	}
}

func dispatchBatchHandler(service *application.SortationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		batchID := c.Param("batchId")

		var req struct {
			TrailerID    string `json:"trailerId" binding:"required"`
			DispatchDock string `json:"dispatchDock" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.DispatchBatchCommand{
			BatchID:      batchID,
			TrailerID:    req.TrailerID,
			DispatchDock: req.DispatchDock,
		}

		batch, err := service.DispatchBatch(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, batch)
	}
}
