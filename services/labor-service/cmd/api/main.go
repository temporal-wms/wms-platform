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

	"github.com/wms-platform/labor-service/internal/application"
	"github.com/wms-platform/labor-service/internal/domain"
	mongoRepo "github.com/wms-platform/labor-service/internal/infrastructure/mongodb"
)

const serviceName = "labor-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting labor-service API")

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
	eventFactory := cloudevents.NewEventFactory("/labor-service")

	// Initialize repositories with instrumented client and event factory
	repo := mongoRepo.NewWorkerRepository(instrumentedMongo.Database(), eventFactory)

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

	// Initialize application services
	laborService := application.NewLaborApplicationService(
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
	apiV1 := router.Group("/api/v1")

	// Worker routes
	workers := apiV1.Group("/workers")
	{
		workers.POST("", createWorkerHandler(laborService, logger))
		workers.GET("/:workerId", getWorkerHandler(laborService, logger))
		workers.POST("/:workerId/shift/start", startShiftHandler(laborService, logger))
		workers.POST("/:workerId/shift/end", endShiftHandler(laborService, logger))
		workers.POST("/:workerId/break/start", startBreakHandler(laborService, logger))
		workers.POST("/:workerId/break/end", endBreakHandler(laborService, logger))
		workers.POST("/:workerId/task/assign", assignTaskHandler(laborService, logger))
		workers.POST("/:workerId/task/start", startTaskHandler(laborService, logger))
		workers.POST("/:workerId/task/complete", completeTaskHandler(laborService, logger))
		workers.POST("/:workerId/skills", addSkillHandler(laborService, logger))
		workers.GET("/status/:status", getByStatusHandler(laborService, logger))
		workers.GET("/zone/:zone", getByZoneHandler(laborService, logger))
		workers.GET("/available", getAvailableHandler(laborService, logger))
		workers.GET("", listWorkersHandler(laborService, logger))
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
		ServerAddr: getEnv("SERVER_ADDR", ":8009"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "labor_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Kafka: &kafka.Config{
			Brokers:       []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			ConsumerGroup: "labor-service",
			ClientID:      "labor-service",
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

func createWorkerHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			WorkerID   string `json:"workerId" binding:"required"`
			EmployeeID string `json:"employeeId" binding:"required"`
			Name       string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": req.WorkerID,
		})

		cmd := application.CreateWorkerCommand{
			WorkerID:   req.WorkerID,
			EmployeeID: req.EmployeeID,
			Name:       req.Name,
		}

		worker, err := service.CreateWorker(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusCreated, worker)
	}
}

func getWorkerHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		query := application.GetWorkerQuery{WorkerID: workerID}

		worker, err := service.GetWorker(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func startShiftHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		var req struct {
			ShiftID   string `json:"shiftId" binding:"required"`
			ShiftType string `json:"shiftType" binding:"required"`
			Zone      string `json:"zone" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shift.id": req.ShiftID,
			"zone":     req.Zone,
		})

		cmd := application.StartShiftCommand{
			WorkerID:  workerID,
			ShiftID:   req.ShiftID,
			ShiftType: req.ShiftType,
			Zone:      req.Zone,
		}

		worker, err := service.StartShift(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func endShiftHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		cmd := application.EndShiftCommand{WorkerID: workerID}

		worker, err := service.EndShift(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func startBreakHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		var req struct {
			BreakType string `json:"breakType" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.StartBreakCommand{
			WorkerID:  workerID,
			BreakType: req.BreakType,
		}

		worker, err := service.StartBreak(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func endBreakHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		cmd := application.EndBreakCommand{WorkerID: workerID}

		worker, err := service.EndBreak(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func assignTaskHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		var req struct {
			TaskID   string `json:"taskId" binding:"required"`
			TaskType string `json:"taskType" binding:"required"`
			Priority int    `json:"priority"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id":   req.TaskID,
			"task.type": req.TaskType,
		})

		cmd := application.AssignTaskCommand{
			WorkerID: workerID,
			TaskID:   req.TaskID,
			TaskType: domain.TaskType(req.TaskType),
			Priority: req.Priority,
		}

		worker, err := service.AssignTask(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func startTaskHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		cmd := application.StartTaskCommand{WorkerID: workerID}

		worker, err := service.StartTask(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func completeTaskHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		var req struct {
			ItemsProcessed int `json:"itemsProcessed"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.CompleteTaskCommand{
			WorkerID:       workerID,
			ItemsProcessed: req.ItemsProcessed,
		}

		worker, err := service.CompleteTask(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func addSkillHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		var req struct {
			TaskType  string `json:"taskType" binding:"required"`
			Level     int    `json:"level" binding:"required"`
			Certified bool   `json:"certified"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.AddSkillCommand{
			WorkerID:  workerID,
			TaskType:  domain.TaskType(req.TaskType),
			Level:     req.Level,
			Certified: req.Certified,
		}

		worker, err := service.AddSkill(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func getByStatusHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		status := domain.WorkerStatus(c.Param("status"))

		query := application.GetByStatusQuery{Status: status}

		workers, err := service.GetByStatus(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, workers)
	}
}

func getByZoneHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		zone := c.Param("zone")

		query := application.GetByZoneQuery{Zone: zone}

		workers, err := service.GetByZone(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, workers)
	}
}

func getAvailableHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		zone := c.Query("zone")

		query := application.GetAvailableQuery{Zone: zone}

		workers, err := service.GetAvailable(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, workers)
	}
}

func listWorkersHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		query := application.ListWorkersQuery{
			Limit:  limit,
			Offset: offset,
		}

		workers, err := service.ListWorkers(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, workers)
	}
}
