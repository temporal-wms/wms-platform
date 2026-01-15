package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
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

	"github.com/wms-platform/packing-service/internal/application"
	"github.com/wms-platform/packing-service/internal/domain"
	mongoRepo "github.com/wms-platform/packing-service/internal/infrastructure/mongodb"
)

const serviceName = "packing-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting packing-service API")

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
	eventFactory := cloudevents.NewEventFactory("/packing-service")

	// Initialize repositories with instrumented client and event factory
	repo := mongoRepo.NewPackTaskRepository(instrumentedMongo.Database(), eventFactory)

	// Initialize idempotency repository
	idempotencyKeyRepo := idempotency.NewMongoKeyRepository(instrumentedMongo.Database())
	logger.Info("Idempotency repositories initialized")

	// Initialize and start outbox publisher
	outboxPublisher := outbox.NewPublisher(
		repo.GetOutboxRepository(),
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
	packingService := application.NewPackingApplicationService(
		repo,
		instrumentedProducer,
		eventFactory,
		logger,
	)

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
		return instrumentedMongo.HealthCheck(ctx)
	}))

	// Metrics endpoint
	router.GET("/metrics", middleware.MetricsEndpoint(m))

	// API v1 routes with tenant context required
	api := router.Group("/api/v1/tasks")
	api.Use(middleware.RequireTenantAuth()) // All API routes require tenant headers
	{
		api.POST("", createPackTaskHandler(packingService, logger))
		api.GET("/:taskId", getPackTaskHandler(packingService, logger))
		api.POST("/:taskId/assign", assignPackTaskHandler(packingService, logger))
		api.POST("/:taskId/start", startPackTaskHandler(packingService, logger))
		api.POST("/:taskId/verify", verifyItemHandler(packingService, logger))
		api.POST("/:taskId/package", selectPackagingHandler(packingService, logger))
		api.POST("/:taskId/seal", sealPackageHandler(packingService, logger))
		api.POST("/:taskId/label", applyLabelHandler(packingService, logger))
		api.POST("/:taskId/complete", completePackTaskHandler(packingService, logger))
		api.GET("/order/:orderId", getByOrderHandler(packingService, logger))
		api.GET("/wave/:waveId", getByWaveHandler(packingService, logger))
		api.GET("/tracking/:trackingNumber", getByTrackingHandler(packingService, logger))
		api.GET("/pending", getPendingHandler(packingService, logger))
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
			logger.Error("Server error", "error", err)
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
		logger.Error("Server forced to shutdown", "error", err)
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
		ServerAddr: getEnv("SERVER_ADDR", ":8006"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "packing_db"),
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

// HTTP Handlers
func createPackTaskHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			TaskID  string              `json:"taskId" binding:"required"`
			OrderID string              `json:"orderId" binding:"required"`
			WaveID  string              `json:"waveId" binding:"required"`
			Items   []domain.PackItem   `json:"items" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id":  req.TaskID,
			"order.id": req.OrderID,
			"wave.id":  req.WaveID,
		})

		cmd := application.CreatePackTaskCommand{
			TaskID:  req.TaskID,
			OrderID: req.OrderID,
			WaveID:  req.WaveID,
			Items:   req.Items,
		}

		task, err := service.CreatePackTask(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusCreated, task)
	}
}

func getPackTaskHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		query := application.GetPackTaskQuery{TaskID: taskID}

		task, err := service.GetPackTask(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, task)
	}
}

func assignPackTaskHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		var req struct {
			PackerID string `json:"packerId" binding:"required"`
			Station  string `json:"station" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"packer.id": req.PackerID,
			"station":   req.Station,
		})

		cmd := application.AssignPackTaskCommand{
			TaskID:   taskID,
			PackerID: req.PackerID,
			Station:  req.Station,
		}

		task, err := service.AssignPackTask(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, task)
	}
}

func startPackTaskHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		cmd := application.StartPackTaskCommand{
			TaskID: taskID,
		}

		task, err := service.StartPackTask(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, task)
	}
}

func verifyItemHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		var req struct {
			SKU string `json:"sku" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"item.sku": req.SKU,
		})

		cmd := application.VerifyItemCommand{
			TaskID: taskID,
			SKU:    req.SKU,
		}

		task, err := service.VerifyItem(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, task)
	}
}

func selectPackagingHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		var req struct {
			PackageType string            `json:"packageType" binding:"required"`
			Dimensions  domain.Dimensions `json:"dimensions" binding:"required"`
			Materials   []string          `json:"materials"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"package.type": req.PackageType,
		})

		cmd := application.SelectPackagingCommand{
			TaskID:      taskID,
			PackageType: domain.PackageType(req.PackageType),
			Dimensions:  req.Dimensions,
			Materials:   req.Materials,
		}

		task, err := service.SelectPackaging(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, task)
	}
}

func sealPackageHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		cmd := application.SealPackageCommand{
			TaskID: taskID,
		}

		task, err := service.SealPackage(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, task)
	}
}

func applyLabelHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		var req struct {
			Label domain.ShippingLabel `json:"label" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"tracking.number": req.Label.TrackingNumber,
			"carrier":         req.Label.Carrier,
		})

		cmd := application.ApplyLabelCommand{
			TaskID: taskID,
			Label:  req.Label,
		}

		task, err := service.ApplyLabel(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, task)
	}
}

func completePackTaskHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		cmd := application.CompletePackTaskCommand{
			TaskID: taskID,
		}

		task, err := service.CompletePackTask(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, task)
	}
}

func getByOrderHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		orderID := c.Param("orderId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": orderID,
		})

		query := application.GetByOrderQuery{OrderID: orderID}

		task, err := service.GetByOrder(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, task)
	}
}

func getByWaveHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waveID := c.Param("waveId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.id": waveID,
		})

		query := application.GetByWaveQuery{WaveID: waveID}

		tasks, err := service.GetByWave(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, tasks)
	}
}

func getByTrackingHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		trackingNumber := c.Param("trackingNumber")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"tracking.number": trackingNumber,
		})

		query := application.GetByTrackingQuery{TrackingNumber: trackingNumber}

		task, err := service.GetByTracking(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, task)
	}
}

func getPendingHandler(service *application.PackingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		query := application.GetPendingQuery{Limit: 50}

		tasks, err := service.GetPending(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, tasks)
	}
}
