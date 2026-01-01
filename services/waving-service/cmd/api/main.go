package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/temporal"
	"github.com/wms-platform/shared/pkg/tracing"

	"github.com/wms-platform/waving-service/internal/application"
	"github.com/wms-platform/waving-service/internal/domain"
	"github.com/wms-platform/waving-service/internal/infrastructure/clients"
	kafkaAdapter "github.com/wms-platform/waving-service/internal/infrastructure/kafka"
	mongoRepo "github.com/wms-platform/waving-service/internal/infrastructure/mongodb"
)

const serviceName = "waving-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting waving-service API")

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

	// Initialize Kafka producer with instrumentation
	kafkaProducer := kafka.NewProducer(config.Kafka)
	instrumentedProducer := kafka.NewInstrumentedProducer(kafkaProducer, m, logger)
	defer instrumentedProducer.Close()
	logger.Info("Kafka producer initialized", "brokers", config.Kafka.Brokers)

	// Initialize CloudEvents factory
	eventFactory := cloudevents.NewEventFactory("/waving-service")

	// Initialize repositories with instrumented client and event factory
	waveRepo := mongoRepo.NewWaveRepository(instrumentedMongo.Database(), eventFactory)

	// Initialize and start outbox publisher
	outboxPublisher := outbox.NewPublisher(
		waveRepo.GetOutboxRepository(),
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

	// Initialize Temporal client
	var temporalClient *temporal.Client
	temporalClient, err = temporal.NewClient(ctx, config.Temporal)
	if err != nil {
		logger.WithError(err).Warn("Failed to connect to Temporal - wave signaling will be disabled")
		// Don't exit - allow service to run without Temporal
	} else {
		defer temporalClient.Close()
		logger.Info("Connected to Temporal", "host", config.Temporal.HostPort)
	}

	// Initialize Order Service client (implements domain.OrderService)
	orderClient := clients.NewOrderServiceClient(config.OrderServiceURL, temporalClient)
	logger.Info("Order service client initialized", "url", config.OrderServiceURL)

	// Initialize Event Publisher (implements domain.EventPublisher)
	eventPublisher := kafkaAdapter.NewEventPublisher(instrumentedProducer, eventFactory, "wms.waves.events")
	logger.Info("Event publisher initialized")

	// Initialize application service
	wavingService := application.NewWavingApplicationService(
		waveRepo,
		instrumentedProducer,
		eventFactory,
		logger,
		orderClient,
		temporalClient,
	)

	// Initialize Continuous Waving Service (scheduler)
	var continuousWavingService *application.ContinuousWavingService
	if config.ContinuousWaving.Enabled {
		cwConfig := application.ContinuousWavingConfig{
			ReleaseInterval:     config.ContinuousWaving.ReleaseInterval,
			BatchSize:           config.ContinuousWaving.BatchSize,
			MinOrdersForRelease: config.ContinuousWaving.MinOrdersForRelease,
			MaxWaitTime:         config.ContinuousWaving.MaxWaitTime,
			PriorityThreshold:   config.ContinuousWaving.PriorityThreshold,
		}
		continuousWavingService = application.NewContinuousWavingService(
			waveRepo,
			orderClient,
			eventPublisher,
			cwConfig,
		)
		if err := continuousWavingService.Start(ctx); err != nil {
			logger.WithError(err).Error("Failed to start continuous waving service")
		} else {
			logger.Info("Continuous waving service started",
				"interval", cwConfig.ReleaseInterval,
				"batchSize", cwConfig.BatchSize,
				"minOrders", cwConfig.MinOrdersForRelease,
			)
		}
	} else {
		logger.Info("Continuous waving service disabled")
	}

	// Setup Gin router with middleware
	router := gin.New()

	// Add CORS middleware for frontend access
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:5174", "http://localhost:9080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-Correlation-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID", "X-Correlation-ID"},
		AllowCredentials: true,
	}))

	// Apply standard middleware (includes recovery, request ID, correlation, logging, error handling)
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

	// Wave API routes
	api := router.Group("/api/v1")
	{
		waves := api.Group("/waves")
		{
			waves.POST("", createWaveHandler(wavingService, logger))
			waves.POST("/from-orders", createWaveFromOrdersHandler(wavingService, logger))
			waves.GET("", listWavesHandler(wavingService, logger))
			waves.GET("/:waveId", getWaveHandler(wavingService, logger))
			waves.PUT("/:waveId", updateWaveHandler(wavingService, logger))
			waves.DELETE("/:waveId", deleteWaveHandler(wavingService, logger))

			// Wave operations
			waves.POST("/:waveId/orders", addOrderToWaveHandler(wavingService, logger))
			waves.DELETE("/:waveId/orders/:orderId", removeOrderFromWaveHandler(wavingService, logger))
			waves.POST("/:waveId/schedule", scheduleWaveHandler(wavingService, logger))
			waves.POST("/:waveId/release", releaseWaveHandler(wavingService, logger))
			waves.POST("/:waveId/cancel", cancelWaveHandler(wavingService, logger))

			// Wave queries
			waves.GET("/status/:status", getWavesByStatusHandler(wavingService, logger))
			waves.GET("/zone/:zone", getWavesByZoneHandler(wavingService, logger))
			waves.GET("/order/:orderId", getWaveByOrderHandler(wavingService, logger))
		}

		// Planning endpoints
		planning := api.Group("/planning")
		{
			planning.POST("/auto", autoPlanWaveHandler(logger))
			planning.POST("/optimize/:waveId", optimizeWaveHandler(logger))
			planning.GET("/ready-for-release", getReadyForReleaseHandler(wavingService, logger))
		}

		// Scheduler endpoints
		scheduler := api.Group("/scheduler")
		{
			scheduler.GET("/status", schedulerStatusHandler(continuousWavingService))
			scheduler.POST("/start", schedulerStartHandler(continuousWavingService, logger))
			scheduler.POST("/stop", schedulerStopHandler(continuousWavingService, logger))
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

	// Stop continuous waving service if running
	if continuousWavingService != nil && continuousWavingService.IsRunning() {
		continuousWavingService.Stop()
		logger.Info("Continuous waving service stopped")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	logger.Info("Server stopped")
}

// Scheduler handlers
func schedulerStatusHandler(service *application.ContinuousWavingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusOK, gin.H{
				"enabled": false,
				"running": false,
				"message": "Continuous waving service not configured",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"enabled": true,
			"running": service.IsRunning(),
		})
	}
}

func schedulerStartHandler(service *application.ContinuousWavingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Continuous waving service not configured"})
			return
		}
		if service.IsRunning() {
			c.JSON(http.StatusOK, gin.H{"message": "Scheduler already running"})
			return
		}
		if err := service.Start(c.Request.Context()); err != nil {
			logger.WithError(err).Error("Failed to start scheduler")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		logger.Info("Scheduler started via API")
		c.JSON(http.StatusOK, gin.H{"message": "Scheduler started"})
	}
}

func schedulerStopHandler(service *application.ContinuousWavingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Continuous waving service not configured"})
			return
		}
		if !service.IsRunning() {
			c.JSON(http.StatusOK, gin.H{"message": "Scheduler already stopped"})
			return
		}
		service.Stop()
		logger.Info("Scheduler stopped via API")
		c.JSON(http.StatusOK, gin.H{"message": "Scheduler stopped"})
	}
}

// Config holds application configuration
type Config struct {
	ServerAddr       string
	MongoDB          *mongodb.Config
	Kafka            *kafka.Config
	Temporal         *temporal.Config
	OrderServiceURL  string
	ContinuousWaving *ContinuousWavingConfig
}

// ContinuousWavingConfig holds configuration for continuous waving scheduler
type ContinuousWavingConfig struct {
	Enabled             bool
	ReleaseInterval     time.Duration
	BatchSize           int
	MinOrdersForRelease int
	MaxWaitTime         time.Duration
	PriorityThreshold   int
}

func loadConfig() *Config {
	return &Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8002"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "waves_db"),
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
		Temporal: &temporal.Config{
			HostPort:  getEnv("TEMPORAL_HOST", "localhost:7233"),
			Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
			Identity:  serviceName,
		},
		OrderServiceURL: getEnv("ORDER_SERVICE_URL", "http://localhost:8001"),
		ContinuousWaving: &ContinuousWavingConfig{
			Enabled:             getEnv("CONTINUOUS_WAVING_ENABLED", "false") == "true",
			ReleaseInterval:     parseDuration(getEnv("CONTINUOUS_WAVING_INTERVAL", "60s")),
			BatchSize:           parseInt(getEnv("CONTINUOUS_WAVING_BATCH_SIZE", "50")),
			MinOrdersForRelease: parseInt(getEnv("CONTINUOUS_WAVING_MIN_ORDERS", "5")),
			MaxWaitTime:         parseDuration(getEnv("CONTINUOUS_WAVING_MAX_WAIT", "15m")),
			PriorityThreshold:   parseInt(getEnv("CONTINUOUS_WAVING_PRIORITY_THRESHOLD", "2")),
		},
	}
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Minute
	}
	return d
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// HTTP Handlers
func createWaveHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			WaveType        string                   `json:"waveType" binding:"required"`
			FulfillmentMode string                   `json:"fulfillmentMode"`
			Zone            string                   `json:"zone"`
			Configuration   domain.WaveConfiguration `json:"configuration"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.CreateWaveCommand{
			WaveType:        req.WaveType,
			FulfillmentMode: req.FulfillmentMode,
			Zone:            req.Zone,
			Configuration:   req.Configuration,
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.type": req.WaveType,
			"wave.zone": req.Zone,
		})

		wave, err := service.CreateWave(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusCreated, wave)
	}
}

func listWavesHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waves, err := service.ListActiveWaves(c.Request.Context())
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, waves)
	}
}

func getWaveHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		query := application.GetWaveQuery{WaveID: c.Param("waveId")}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.id": query.WaveID,
		})

		wave, err := service.GetWave(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, wave)
	}
}

func updateWaveHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waveID := c.Param("waveId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.id": waveID,
		})

		var req struct {
			Priority int    `json:"priority"`
			Zone     string `json:"zone"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.UpdateWaveCommand{WaveID: waveID}
		if req.Priority > 0 {
			cmd.Priority = &req.Priority
		}
		if req.Zone != "" {
			cmd.Zone = &req.Zone
		}

		wave, err := service.UpdateWave(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, wave)
	}
}

func deleteWaveHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waveID := c.Param("waveId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.id": waveID,
		})

		cmd := application.DeleteWaveCommand{WaveID: waveID}

		if err := service.DeleteWave(c.Request.Context(), cmd); err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusNoContent, nil)
	}
}

func addOrderToWaveHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waveID := c.Param("waveId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.id": waveID,
		})

		var order domain.WaveOrder
		if err := c.ShouldBindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": order.OrderID,
		})

		cmd := application.AddOrderToWaveCommand{
			WaveID: waveID,
			Order:  order,
		}

		wave, err := service.AddOrderToWave(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, wave)
	}
}

func removeOrderFromWaveHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waveID := c.Param("waveId")
		orderID := c.Param("orderId")

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.id":  waveID,
			"order.id": orderID,
		})

		cmd := application.RemoveOrderFromWaveCommand{
			WaveID:  waveID,
			OrderID: orderID,
		}

		wave, err := service.RemoveOrderFromWave(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, wave)
	}
}

func scheduleWaveHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waveID := c.Param("waveId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.id": waveID,
		})

		var req struct {
			ScheduledStart time.Time `json:"scheduledStart" binding:"required"`
			ScheduledEnd   time.Time `json:"scheduledEnd" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.scheduledStart": req.ScheduledStart,
			"wave.scheduledEnd":   req.ScheduledEnd,
		})

		cmd := application.ScheduleWaveCommand{
			WaveID:         waveID,
			ScheduledStart: req.ScheduledStart,
			ScheduledEnd:   req.ScheduledEnd,
		}

		wave, err := service.ScheduleWave(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, wave)
	}
}

func releaseWaveHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waveID := c.Param("waveId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.id": waveID,
		})

		cmd := application.ReleaseWaveCommand{WaveID: waveID}

		wave, err := service.ReleaseWave(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, wave)
	}
}

func cancelWaveHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waveID := c.Param("waveId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.id": waveID,
		})

		var req struct {
			Reason string `json:"reason" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.cancelReason": req.Reason,
		})

		cmd := application.CancelWaveCommand{
			WaveID: waveID,
			Reason: req.Reason,
		}

		wave, err := service.CancelWave(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, wave)
	}
}

func getWavesByStatusHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		status := c.Param("status")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.status": status,
		})

		query := application.GetWavesByStatusQuery{Status: status}

		waves, err := service.GetWavesByStatus(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, waves)
	}
}

func getWavesByZoneHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		zone := c.Param("zone")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.zone": zone,
		})

		query := application.GetWavesByZoneQuery{Zone: zone}

		waves, err := service.GetWavesByZone(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, waves)
	}
}

func getWaveByOrderHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		orderID := c.Param("orderId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": orderID,
		})

		query := application.GetWaveByOrderQuery{OrderID: orderID}

		wave, err := service.GetWaveByOrder(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, wave)
	}
}

func autoPlanWaveHandler(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Auto-planning not yet implemented"})
	}
}

func optimizeWaveHandler(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Optimization not yet implemented"})
	}
}

func getReadyForReleaseHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waves, err := service.GetReadyForRelease(c.Request.Context())
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, waves)
	}
}

func createWaveFromOrdersHandler(service *application.WavingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req application.CreateWaveFromOrdersRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.CreateWaveFromOrdersCommand{
			OrderIDs:        req.OrderIDs,
			WaveType:        req.WaveType,
			FulfillmentMode: req.FulfillmentMode,
			Zone:            req.Zone,
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"wave.type":   req.WaveType,
			"order.count": len(req.OrderIDs),
		})

		result, err := service.CreateWaveFromOrders(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusCreated, result)
	}
}
