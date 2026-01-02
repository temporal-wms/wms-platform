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
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/tracing"

	"github.com/wms-platform/services/stow-service/internal/application"
	"github.com/wms-platform/services/stow-service/internal/domain"
	mongoRepo "github.com/wms-platform/services/stow-service/internal/infrastructure/mongodb"
)

const serviceName = "stow-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting stow-service API")

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

	// Initialize Kafka producer with instrumentation
	kafkaProducer := kafka.NewProducer(config.Kafka)
	instrumentedProducer := kafka.NewInstrumentedProducer(kafkaProducer, m, logger)
	defer instrumentedProducer.Close()
	logger.Info("Kafka producer initialized", "brokers", config.Kafka.Brokers)

	// Initialize CloudEvents factory
	eventFactory := cloudevents.NewEventFactory("/stow-service")

	// Initialize repositories with instrumented client and event factory
	taskRepo := mongoRepo.NewPutawayTaskRepository(instrumentedMongo.Database(), eventFactory)
	locationRepo := mongoRepo.NewStorageLocationRepository(instrumentedMongo.Database())

	// Initialize and start outbox publisher
	outboxPublisher := outbox.NewPublisher(
		taskRepo.GetOutboxRepository(),
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
	stowService := application.NewStowService(taskRepo, locationRepo, logger)

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
	api := router.Group("/api/v1/tasks")
	{
		api.GET("", listTasksHandler(stowService, logger))
		api.POST("", createTaskHandler(stowService, logger))
		api.GET("/pending", getPendingTasksHandler(stowService, logger))
		api.GET("/status/:status", getTasksByStatusHandler(stowService, logger))
		api.GET("/worker/:workerId", getTasksByWorkerHandler(stowService, logger))
		api.GET("/shipment/:shipmentId", getTasksByShipmentHandler(stowService, logger))
		api.GET("/:taskId", getTaskHandler(stowService, logger))
		api.POST("/:taskId/assign", assignTaskHandler(stowService, logger))
		api.POST("/:taskId/start", startTaskHandler(stowService, logger))
		api.POST("/:taskId/stow", recordStowHandler(stowService, logger))
		api.POST("/:taskId/complete", completeTaskHandler(stowService, logger))
		api.POST("/:taskId/fail", failTaskHandler(stowService, logger))
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
		ServerAddr: getEnv("SERVER_ADDR", ":8011"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "stow_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Kafka: &kafka.Config{
			Brokers:       []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			ConsumerGroup: "stow-service",
			ClientID:      "stow-service",
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

func createTaskHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			ShipmentID        string  `json:"shipmentId"`
			SKU               string  `json:"sku" binding:"required"`
			ProductName       string  `json:"productName" binding:"required"`
			Quantity          int     `json:"quantity" binding:"required,min=1"`
			SourceToteID      string  `json:"sourceToteId" binding:"required"`
			SourceLocationID  string  `json:"sourceLocationId"`
			IsHazmat          bool    `json:"isHazmat"`
			RequiresColdChain bool    `json:"requiresColdChain"`
			IsOversized       bool    `json:"isOversized"`
			IsFragile         bool    `json:"isFragile"`
			Weight            float64 `json:"weight"`
			Priority          int     `json:"priority"`
			Strategy          string  `json:"strategy"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"sku":      req.SKU,
			"quantity": req.Quantity,
		})

		cmd := application.CreatePutawayTaskCommand{
			ShipmentID:        req.ShipmentID,
			SKU:               req.SKU,
			ProductName:       req.ProductName,
			Quantity:          req.Quantity,
			SourceToteID:      req.SourceToteID,
			SourceLocationID:  req.SourceLocationID,
			IsHazmat:          req.IsHazmat,
			RequiresColdChain: req.RequiresColdChain,
			IsOversized:       req.IsOversized,
			IsFragile:         req.IsFragile,
			Weight:            req.Weight,
			Priority:          req.Priority,
			Strategy:          req.Strategy,
		}

		task, err := service.CreatePutawayTask(c.Request.Context(), cmd)
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

func getTaskHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
		})

		task, err := service.GetTask(c.Request.Context(), taskID)
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

func listTasksHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		limitStr := c.Query("limit")
		limit := 50
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
				if limit > 200 {
					limit = 200
				}
			}
		}

		tasks, err := service.GetPendingTasks(c.Request.Context(), limit)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"tasks": tasks, "total": len(tasks)})
	}
}

func getPendingTasksHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		limitStr := c.Query("limit")
		limit := 20
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		tasks, err := service.GetPendingTasks(c.Request.Context(), limit)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"tasks": tasks, "total": len(tasks)})
	}
}

func getTasksByStatusHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		status := domain.PutawayStatus(c.Param("status"))
		pagination := domain.DefaultPagination()

		tasks, err := service.GetTasksByStatus(c.Request.Context(), status, pagination)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"tasks": tasks, "total": len(tasks)})
	}
}

func getTasksByWorkerHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		pagination := domain.DefaultPagination()

		tasks, err := service.GetTasksByWorker(c.Request.Context(), workerID, pagination)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"tasks": tasks, "total": len(tasks)})
	}
}

func getTasksByShipmentHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		shipmentID := c.Param("shipmentId")

		tasks, err := service.GetTasksByShipment(c.Request.Context(), shipmentID)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"tasks": tasks, "total": len(tasks)})
	}
}

func assignTaskHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")

		var req struct {
			WorkerID string `json:"workerId" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id":   taskID,
			"worker.id": req.WorkerID,
		})

		cmd := application.AssignTaskCommand{
			TaskID:   taskID,
			WorkerID: req.WorkerID,
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

func startTaskHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
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

func recordStowHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")

		var req struct {
			Quantity int `json:"quantity" binding:"required,min=1"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id":  taskID,
			"quantity": req.Quantity,
		})

		cmd := application.RecordStowCommand{
			TaskID:   taskID,
			Quantity: req.Quantity,
		}

		task, err := service.RecordStow(c.Request.Context(), cmd)
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

func completeTaskHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
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

func failTaskHandler(service *application.StowService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		taskID := c.Param("taskId")

		var req struct {
			Reason string `json:"reason" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id": taskID,
			"reason":  req.Reason,
		})

		cmd := application.FailTaskCommand{
			TaskID: taskID,
			Reason: req.Reason,
		}

		task, err := service.FailTask(c.Request.Context(), cmd)
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
