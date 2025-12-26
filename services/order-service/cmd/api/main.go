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
	"github.com/wms-platform/shared/pkg/temporal"
	"github.com/wms-platform/shared/pkg/tracing"

	"github.com/wms-platform/services/order-service/internal/application"
	"github.com/wms-platform/services/order-service/internal/domain"
	mongoRepo "github.com/wms-platform/services/order-service/internal/infrastructure/mongodb"
	"github.com/wms-platform/services/order-service/internal/infrastructure/projections"
)

const serviceName = "order-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting order-service API")

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
	eventFactory := cloudevents.NewEventFactory("/order-service")

	// Initialize repositories with instrumented client and event factory
	orderRepo := mongoRepo.NewOrderRepository(instrumentedMongo.Database(), eventFactory)

	// Initialize CQRS read model repository and projector
	projectionRepo := projections.NewMongoOrderListProjectionRepository(instrumentedMongo.Database())
	orderProjector := projections.NewOrderProjector(projectionRepo, orderRepo, logger)
	logger.Info("CQRS projections initialized")

	// Initialize and start outbox publisher
	outboxPublisher := outbox.NewPublisher(
		orderRepo.GetOutboxRepository(),
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
	temporalClient, err := temporal.NewClient(ctx, config.Temporal)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to Temporal")
		os.Exit(1)
	}
	defer temporalClient.Close()
	logger.Info("Connected to Temporal", "namespace", config.Temporal.Namespace)

	// Initialize business metrics helper
	businessMetrics := middleware.NewBusinessMetrics(m)

	// Initialize application service (write side)
	orderService := application.NewOrderApplicationService(
		orderRepo,
		instrumentedProducer,
		eventFactory,
		temporalClient,
		orderProjector,
		logger,
		businessMetrics,
	)

	// Initialize query service (read side - CQRS)
	orderQueryService := application.NewOrderQueryService(
		projectionRepo,
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
	v1 := router.Group("/api/v1")
	{
		orders := v1.Group("/orders")
		{
			// Command handlers (write side)
			orders.POST("", createOrderHandler(orderService, logger))
			orders.PUT("/:orderId/validate", validateOrderHandler(orderService, logger))
			orders.PUT("/:orderId/cancel", cancelOrderHandler(orderService, logger))

			// Query handlers (read side - CQRS)
			orders.GET("/:orderId", getOrderHandler(orderService, logger))
			orders.GET("", listOrdersHandler(orderQueryService, logger))
			orders.GET("/status/:status", listOrdersByStatusHandler(orderQueryService, logger))
			orders.GET("/customer/:customerId", listOrdersByCustomerHandler(orderQueryService, logger))
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
	Temporal   *temporal.Config
}

func loadConfig() *Config {
	return &Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8001"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "orders_db"),
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
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// CreateOrderRequest is the request body for creating an order
type CreateOrderRequest struct {
	CustomerID         string             `json:"customerId" binding:"required"`
	Items              []domain.OrderItem `json:"items" binding:"required,min=1"`
	ShippingAddress    domain.Address     `json:"shippingAddress" binding:"required"`
	Priority           string             `json:"priority" binding:"required"`
	PromisedDeliveryAt time.Time          `json:"promisedDeliveryAt" binding:"required"`
}

// CancelOrderRequest is the request body for cancelling an order
type CancelOrderRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func createOrderHandler(service *application.OrderApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req CreateOrderRequest
		if appErr := middleware.BindAndValidate(c, &req); appErr != nil {
			responder.RespondWithAppError(appErr)
			return
		}

		// Map request to command
		cmd := application.CreateOrderCommand{
			CustomerID:         req.CustomerID,
			Items:              toOrderItemInputs(req.Items),
			ShippingAddress:    toAddressInput(req.ShippingAddress),
			Priority:           req.Priority,
			PromisedDeliveryAt: req.PromisedDeliveryAt,
		}

		// Add span attributes for tracing
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"customer.id":    req.CustomerID,
			"order.items":    len(req.Items),
			"order.priority": req.Priority,
		})

		// Execute use case
		result, err := service.CreateOrder(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondWithAppError(err.(*errors.AppError))
			return
		}

		// Add workflow info to span
		if result.WorkflowID != "" {
			middleware.AddSpanEvent(c, "workflow_started", map[string]interface{}{
				"workflow_id": result.WorkflowID,
			})
		}

		c.JSON(http.StatusCreated, result)
	}
}

func getOrderHandler(service *application.OrderApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		query := application.GetOrderQuery{
			OrderID: c.Param("orderId"),
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": query.OrderID,
		})

		order, err := service.GetOrder(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, order)
	}
}

func validateOrderHandler(service *application.OrderApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		cmd := application.ValidateOrderCommand{
			OrderID: c.Param("orderId"),
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": cmd.OrderID,
		})

		order, err := service.ValidateOrder(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, order)
	}
}

func cancelOrderHandler(service *application.OrderApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req CancelOrderRequest
		if appErr := middleware.BindAndValidate(c, &req); appErr != nil {
			responder.RespondWithAppError(appErr)
			return
		}

		cmd := application.CancelOrderCommand{
			OrderID: c.Param("orderId"),
			Reason:  req.Reason,
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id":      cmd.OrderID,
			"cancel.reason": cmd.Reason,
		})

		order, err := service.CancelOrder(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, order)
	}
}

func listOrdersHandler(queryService *application.OrderQueryService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		// Parse pagination parameters (CQRS uses limit/offset instead of page/pageSize)
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		if limit < 1 || limit > 100 {
			limit = 50
		}
		if offset < 0 {
			offset = 0
		}

		// Build query from query params (extended CQRS filters)
		query := application.ListOrdersQuery{
			Limit:     limit,
			Offset:    offset,
			SortBy:    c.DefaultQuery("sortBy", "receivedAt"),
			SortOrder: c.DefaultQuery("sortOrder", "desc"),
		}

		// Basic filters
		if customerID := c.Query("customerId"); customerID != "" {
			query.CustomerID = &customerID
		}
		if status := c.Query("status"); status != "" {
			query.Status = &status
		}
		if priority := c.Query("priority"); priority != "" {
			query.Priority = &priority
		}

		// Extended CQRS filters
		if waveID := c.Query("waveId"); waveID != "" {
			query.WaveID = &waveID
		}
		if picker := c.Query("assignedPicker"); picker != "" {
			query.AssignedPicker = &picker
		}
		if state := c.Query("shipToState"); state != "" {
			query.ShipToState = &state
		}
		if isLate := c.Query("isLate"); isLate == "true" {
			late := true
			query.IsLate = &late
		}
		if isPriority := c.Query("isPriority"); isPriority == "true" {
			priority := true
			query.IsPriority = &priority
		}
		if search := c.Query("search"); search != "" {
			query.SearchTerm = search
		}

		result, err := queryService.ListOrders(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func listOrdersByStatusHandler(queryService *application.OrderQueryService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		status := c.Param("status")

		// Parse pagination parameters
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		if limit < 1 || limit > 100 {
			limit = 50
		}
		if offset < 0 {
			offset = 0
		}

		query := application.ListOrdersQuery{
			Status:    &status,
			Limit:     limit,
			Offset:    offset,
			SortBy:    c.DefaultQuery("sortBy", "receivedAt"),
			SortOrder: c.DefaultQuery("sortOrder", "desc"),
		}

		result, err := queryService.ListOrders(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func listOrdersByCustomerHandler(queryService *application.OrderQueryService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		customerID := c.Param("customerId")

		// Parse pagination parameters
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		if limit < 1 || limit > 100 {
			limit = 50
		}
		if offset < 0 {
			offset = 0
		}

		query := application.ListOrdersQuery{
			CustomerID: &customerID,
			Limit:      limit,
			Offset:     offset,
			SortBy:     c.DefaultQuery("sortBy", "receivedAt"),
			SortOrder:  c.DefaultQuery("sortOrder", "desc"),
		}

		result, err := queryService.ListOrders(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// Helper functions to convert domain types to application inputs
func toOrderItemInputs(items []domain.OrderItem) []application.OrderItemInput {
	inputs := make([]application.OrderItemInput, 0, len(items))
	for _, item := range items {
		inputs = append(inputs, application.OrderItemInput{
			SKU:      item.SKU,
			Quantity: item.Quantity,
			Weight:   item.Weight,
		})
	}
	return inputs
}

func toAddressInput(address domain.Address) application.AddressInput {
	return application.AddressInput{
		Street:  address.Street,
		City:    address.City,
		State:   address.State,
		ZipCode: address.ZipCode,
		Country: address.Country,
	}
}
