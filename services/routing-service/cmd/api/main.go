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
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/tracing"

	"github.com/wms-platform/routing-service/internal/application"
	"github.com/wms-platform/routing-service/internal/domain"
	mongoRepo "github.com/wms-platform/routing-service/internal/infrastructure/mongodb"
)

const serviceName = "routing-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting routing-service API")

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
	eventFactory := cloudevents.NewEventFactory("/routing-service")

	// Initialize repositories with instrumented client and event factory
	repo := mongoRepo.NewRouteRepository(instrumentedMongo.Database(), eventFactory)

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

	// Initialize route calculator (nil for warehouse layout and inventory locator for now)
	routeCalculator := application.NewRouteCalculator(repo, nil, nil)

	// Initialize application service
	routingService := application.NewRoutingApplicationService(
		repo,
		routeCalculator,
		instrumentedProducer,
		eventFactory,
		logger,
	)

	// Setup Gin router with middleware
	router := gin.New()

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

	// API v1 routes
	api := router.Group("/api/v1")
	{
		routes := api.Group("/routes")
		{
			routes.POST("", calculateRouteHandler(routingService, logger))
			routes.GET("/:routeId", getRouteHandler(routingService, logger))
			routes.DELETE("/:routeId", deleteRouteHandler(routingService, logger))

			// Route operations
			routes.POST("/:routeId/start", startRouteHandler(routingService, logger))
			routes.POST("/:routeId/stops/:stopNumber/complete", completeStopHandler(routingService, logger))
			routes.POST("/:routeId/stops/:stopNumber/skip", skipStopHandler(routingService, logger))
			routes.POST("/:routeId/complete", completeRouteHandler(routingService, logger))
			routes.POST("/:routeId/pause", pauseRouteHandler(routingService, logger))
			routes.POST("/:routeId/cancel", cancelRouteHandler(routingService, logger))

			// Route queries
			routes.GET("/order/:orderId", getRoutesByOrderHandler(routingService, logger))
			routes.GET("/wave/:waveId", getRoutesByWaveHandler(routingService, logger))
			routes.GET("/picker/:pickerId", getRoutesByPickerHandler(routingService, logger))
			routes.GET("/picker/:pickerId/active", getActiveRouteHandler(routingService, logger))
			routes.GET("/status/:status", getRoutesByStatusHandler(routingService, logger))
			routes.GET("/pending", getPendingRoutesHandler(routingService, logger))
		}

		// Analysis endpoints
		analysis := api.Group("/analysis")
		{
			analysis.GET("/route/:routeId", analyzeRouteHandler(routingService, logger))
			analysis.POST("/suggest-strategy", suggestStrategyHandler(routingService, logger))
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
		ServerAddr: getEnv("SERVER_ADDR", ":8003"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "routing_db"),
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

// Handler implementations

func calculateRouteHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req domain.RouteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": req.OrderID,
			"wave.id":  req.WaveID,
		})

		cmd := application.CalculateRouteCommand{RouteRequest: req}

		route, err := service.CalculateRoute(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusCreated, route)
	}
}

func getRouteHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		routeID := c.Param("routeId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"route.id": routeID,
		})

		query := application.GetRouteQuery{RouteID: routeID}

		route, err := service.GetRoute(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, route)
	}
}

func deleteRouteHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		routeID := c.Param("routeId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"route.id": routeID,
		})

		cmd := application.DeleteRouteCommand{RouteID: routeID}

		if err := service.DeleteRoute(c.Request.Context(), cmd); err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusNoContent, nil)
	}
}

func startRouteHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		routeID := c.Param("routeId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"route.id": routeID,
		})

		var req struct {
			PickerID string `json:"pickerId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"picker.id": req.PickerID,
		})

		cmd := application.StartRouteCommand{
			RouteID:  routeID,
			PickerID: req.PickerID,
		}

		route, err := service.StartRoute(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, route)
	}
}

func completeStopHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		routeID := c.Param("routeId")
		stopNumber := 0
		if _, err := fmt.Sscanf(c.Param("stopNumber"), "%d", &stopNumber); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stop number"})
			return
		}

		var req struct {
			PickedQty int    `json:"pickedQty" binding:"required"`
			ToteID    string `json:"toteId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.CompleteStopCommand{
			RouteID:    routeID,
			StopNumber: stopNumber,
			PickedQty:  req.PickedQty,
			ToteID:     req.ToteID,
		}

		route, err := service.CompleteStop(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, route)
	}
}

func skipStopHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		routeID := c.Param("routeId")
		stopNumber := 0
		if _, err := fmt.Sscanf(c.Param("stopNumber"), "%d", &stopNumber); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stop number"})
			return
		}

		var req struct {
			Reason string `json:"reason" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.SkipStopCommand{
			RouteID:    routeID,
			StopNumber: stopNumber,
			Reason:     req.Reason,
		}

		route, err := service.SkipStop(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, route)
	}
}

func completeRouteHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		routeID := c.Param("routeId")

		cmd := application.CompleteRouteCommand{RouteID: routeID}

		route, err := service.CompleteRoute(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, route)
	}
}

func pauseRouteHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		routeID := c.Param("routeId")

		cmd := application.PauseRouteCommand{RouteID: routeID}

		route, err := service.PauseRoute(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, route)
	}
}

func cancelRouteHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		routeID := c.Param("routeId")

		var req struct {
			Reason string `json:"reason" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.CancelRouteCommand{
			RouteID: routeID,
			Reason:  req.Reason,
		}

		route, err := service.CancelRoute(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, route)
	}
}

func getRoutesByOrderHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		orderID := c.Param("orderId")

		query := application.GetRoutesByOrderQuery{OrderID: orderID}

		routes, err := service.GetRoutesByOrder(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, routes)
	}
}

func getRoutesByWaveHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		waveID := c.Param("waveId")

		query := application.GetRoutesByWaveQuery{WaveID: waveID}

		routes, err := service.GetRoutesByWave(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, routes)
	}
}

func getRoutesByPickerHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		pickerID := c.Param("pickerId")

		query := application.GetRoutesByPickerQuery{PickerID: pickerID}

		routes, err := service.GetRoutesByPicker(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, routes)
	}
}

func getActiveRouteHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		pickerID := c.Param("pickerId")

		query := application.GetActiveRouteQuery{PickerID: pickerID}

		route, err := service.GetActiveRoute(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, route)
	}
}

func getRoutesByStatusHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		status := domain.RouteStatus(c.Param("status"))

		query := application.GetRoutesByStatusQuery{Status: status}

		routes, err := service.GetRoutesByStatus(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, routes)
	}
}

func getPendingRoutesHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		zone := c.Query("zone")

		query := application.GetPendingRoutesQuery{
			Zone:  zone,
			Limit: 50,
		}

		routes, err := service.GetPendingRoutes(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, routes)
	}
}

func analyzeRouteHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		routeID := c.Param("routeId")

		query := application.AnalyzeRouteQuery{RouteID: routeID}

		analysis, err := service.AnalyzeRoute(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, analysis)
	}
}

func suggestStrategyHandler(service *application.RoutingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			Items []domain.RouteItem `json:"items" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		query := application.SuggestStrategyQuery{Items: req.Items}

		strategy, err := service.SuggestStrategy(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"strategy": strategy})
	}
}
