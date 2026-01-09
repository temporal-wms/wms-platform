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

	"github.com/wms-platform/picking-service/internal/application"
	"github.com/wms-platform/picking-service/internal/domain"
	mongoRepo "github.com/wms-platform/picking-service/internal/infrastructure/mongodb"
)

const serviceName = "picking-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting picking-service API")

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
	eventFactory := cloudevents.NewEventFactory("/picking-service")

	// Initialize repositories with instrumented client and event factory
	repo := mongoRepo.NewPickTaskRepository(instrumentedMongo.Database(), eventFactory)

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
	pickingService := application.NewPickingApplicationService(
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

	// API v1 routes
	api := router.Group("/api/v1/tasks")
	{
		// List endpoint first (before :taskId wildcard)
		api.GET("", listTasksHandler(pickingService, logger))
		api.POST("", createTaskHandler(pickingService, logger))
		// Static routes before wildcard
		api.GET("/pending", getPendingTasksHandler(pickingService, logger))
		api.GET("/order/:orderId", getTasksByOrderHandler(pickingService, logger))
		api.GET("/wave/:waveId", getTasksByWaveHandler(pickingService, logger))
		api.GET("/picker/:pickerId", getTasksByPickerHandler(pickingService, logger))
		api.GET("/picker/:pickerId/active", getActiveTaskHandler(pickingService, logger))
		// Wildcard routes after static routes
		api.GET("/:taskId", getTaskHandler(pickingService, logger))
		api.POST("/:taskId/assign", assignTaskHandler(pickingService, logger))
		api.POST("/:taskId/start", startTaskHandler(pickingService, logger))
		api.POST("/:taskId/pick", confirmPickHandler(pickingService, logger))
		api.POST("/:taskId/exception", reportExceptionHandler(pickingService, logger))
		api.POST("/:taskId/complete", completeTaskHandler(pickingService, logger))
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
		ServerAddr: getEnv("SERVER_ADDR", ":8004"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "picking_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Kafka: &kafka.Config{
			Brokers:       []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			ConsumerGroup: "picking-service",
			ClientID:      "picking-service",
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
func createTaskHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			TaskID  string            `json:"taskId" binding:"required"`
			OrderID string            `json:"orderId" binding:"required"`
			WaveID  string            `json:"waveId" binding:"required"`
			RouteID string            `json:"routeId"`
			Method  string            `json:"method"`
			Items   []domain.PickItem `json:"items" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id":    req.TaskID,
			"order.id":   req.OrderID,
			"wave.id":    req.WaveID,
			"task.items": len(req.Items),
		})

		cmd := application.CreatePickTaskCommand{
			TaskID:  req.TaskID,
			OrderID: req.OrderID,
			WaveID:  req.WaveID,
			RouteID: req.RouteID,
			Method:  req.Method,
			Items:   req.Items,
		}

		task, err := service.CreatePickTask(c.Request.Context(), cmd)
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

func getTaskHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		query := application.GetPickTaskQuery{TaskID: taskID}

		task, err := service.GetPickTask(c.Request.Context(), query)
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

func assignTaskHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		var req struct {
			PickerID string `json:"pickerId" binding:"required"`
			ToteID   string `json:"toteId" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"picker.id": req.PickerID,
			"tote.id":   req.ToteID,
		})

		cmd := application.AssignTaskCommand{
			TaskID:   taskID,
			PickerID: req.PickerID,
			ToteID:   req.ToteID,
		}

		task, err := service.AssignTask(c.Request.Context(), cmd)
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

func startTaskHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		cmd := application.StartTaskCommand{TaskID: taskID}

		task, err := service.StartTask(c.Request.Context(), cmd)
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

func confirmPickHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		var req struct {
			SKU        string `json:"sku" binding:"required"`
			LocationID string `json:"locationId"` // Optional - may be empty for tasks without location info
			PickedQty  int    `json:"pickedQty" binding:"required"`
			ToteID     string `json:"toteId" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"item.sku":    req.SKU,
			"location.id": req.LocationID,
			"picked.qty":  req.PickedQty,
		})

		cmd := application.ConfirmPickCommand{
			TaskID:     taskID,
			SKU:        req.SKU,
			LocationID: req.LocationID,
			PickedQty:  req.PickedQty,
			ToteID:     req.ToteID,
		}

		task, err := service.ConfirmPick(c.Request.Context(), cmd)
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

func reportExceptionHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		var req struct {
			SKU          string `json:"sku" binding:"required"`
			LocationID   string `json:"locationId" binding:"required"`
			Reason       string `json:"reason" binding:"required"`
			RequestedQty int    `json:"requestedQty"`
			AvailableQty int    `json:"availableQty"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"exception.sku":    req.SKU,
			"exception.reason": req.Reason,
		})

		cmd := application.ReportExceptionCommand{
			TaskID:       taskID,
			SKU:          req.SKU,
			LocationID:   req.LocationID,
			Reason:       req.Reason,
			RequestedQty: req.RequestedQty,
			AvailableQty: req.AvailableQty,
		}

		task, err := service.ReportException(c.Request.Context(), cmd)
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

func completeTaskHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		cmd := application.CompleteTaskCommand{TaskID: taskID}

		task, err := service.CompleteTask(c.Request.Context(), cmd)
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

func getTasksByOrderHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		orderID := c.Param("orderId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": orderID,
		})

		query := application.GetTasksByOrderQuery{OrderID: orderID}

		tasks, err := service.GetTasksByOrder(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, tasks)
	}
}

func getTasksByWaveHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waveID := c.Param("waveId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.id": waveID,
		})

		query := application.GetTasksByWaveQuery{WaveID: waveID}

		tasks, err := service.GetTasksByWave(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, tasks)
	}
}

func getTasksByPickerHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		pickerID := c.Param("pickerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"picker.id": pickerID,
		})

		query := application.GetTasksByPickerQuery{PickerID: pickerID}

		tasks, err := service.GetTasksByPicker(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, tasks)
	}
}

func getActiveTaskHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		pickerID := c.Param("pickerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"picker.id": pickerID,
		})

		query := application.GetActiveTaskQuery{PickerID: pickerID}

		task, err := service.GetActiveTask(c.Request.Context(), query)
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

func getPendingTasksHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		zone := c.Query("zone")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"zone": zone,
		})

		query := application.GetPendingTasksQuery{
			Zone:  zone,
			Limit: 50,
		}

		tasks, err := service.GetPendingTasks(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, tasks)
	}
}

func listTasksHandler(service *application.PickingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		status := c.Query("status")
		zone := c.Query("zone")
		limitStr := c.Query("limit")

		// Default limit is 200, max is 1000
		limit := 200
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
				if limit > 1000 {
					limit = 1000
				}
			}
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"status": status,
			"zone":   zone,
			"limit":  limit,
		})

		query := application.ListTasksQuery{
			Status: status,
			Zone:   zone,
			Limit:  limit,
		}

		tasks, err := service.ListTasks(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, tasks)
	}
}
