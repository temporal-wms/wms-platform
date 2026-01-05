package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

	// Initialize idempotency indexes
	if err := idempotency.InitializeIndexes(ctx, instrumentedMongo.Database()); err != nil {
		logger.WithError(err).Warn("Failed to initialize idempotency indexes")
		// Continue - indexes might already exist
	} else {
		logger.Info("Idempotency indexes initialized")
	}

	// Initialize Kafka producer with instrumentation
	kafkaProducer := kafka.NewProducer(config.Kafka)
	instrumentedProducer := kafka.NewInstrumentedProducer(kafkaProducer, m, logger)
	defer instrumentedProducer.Close()
	logger.Info("Kafka producer initialized", "brokers", config.Kafka.Brokers)

	// Initialize CloudEvents factory
	eventFactory := cloudevents.NewEventFactory("/order-service")

	// Initialize repositories with instrumented client and event factory
	orderRepo := mongoRepo.NewOrderRepository(instrumentedMongo.Database(), eventFactory)

	// Initialize idempotency repositories
	idempotencyKeyRepo := idempotency.NewMongoKeyRepository(instrumentedMongo.Database())
	// Message repository will be used when integrating Kafka consumers
	// idempotencyMsgRepo := idempotency.NewMongoMessageRepository(instrumentedMongo.Database())
	logger.Info("Idempotency repositories initialized")

	// Initialize reprocessing repositories
	retryMetadataRepo := mongoRepo.NewRetryMetadataRepository(instrumentedMongo.Database())
	deadLetterRepo := mongoRepo.NewDeadLetterRepository(instrumentedMongo.Database())
	logger.Info("Reprocessing repositories initialized")

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

	// Initialize failure metrics helper
	failureMetrics := middleware.NewFailureMetrics(m)

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

	// Initialize reprocessing service
	reprocessingService := application.NewReprocessingService(
		orderRepo,
		retryMetadataRepo,
		deadLetterRepo,
		logger,
		failureMetrics,
	)

	// Setup Gin router with middleware
	router := gin.New()

	// Initialize idempotency metrics
	idempotencyMetrics := idempotency.NewMetrics(nil) // Uses default Prometheus registry

	// Apply standard middleware (includes recovery, request ID, correlation, logging, error handling)
	middlewareConfig := middleware.DefaultConfig(serviceName, logger.Logger)

	// Configure idempotency
	middlewareConfig.IdempotencyConfig = &idempotency.Config{
		ServiceName:     serviceName,
		Repository:      idempotencyKeyRepo,
		RequireKey:      false, // Start with optional mode for backward compatibility
		OnlyMutating:    true,  // Only POST/PUT/PATCH/DELETE
		MaxKeyLength:    255,
		LockTimeout:     5 * time.Minute,
		RetentionPeriod: 24 * time.Hour,
		MaxResponseSize: 1024 * 1024, // 1MB
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
	v1 := router.Group("/api/v1")
	{
		orders := v1.Group("/orders")
		{
			// Command handlers (write side)
			orders.POST("", createOrderHandler(orderService, logger))
			orders.PUT("/:orderId/validate", validateOrderHandler(orderService, logger))
			orders.PUT("/:orderId/cancel", cancelOrderHandler(orderService, logger))
			orders.PUT("/:orderId/assign-wave", assignWaveHandler(orderService, logger))
			orders.PUT("/:orderId/start-picking", startPickingHandler(orderService, logger))
			orders.PUT("/:orderId/mark-consolidated", markConsolidatedHandler(orderService, logger))
			orders.PUT("/:orderId/mark-packed", markPackedHandler(orderService, logger))

			// Query handlers (read side - CQRS)
			orders.GET("/:orderId", getOrderHandler(orderService, logger))
			orders.GET("", listOrdersHandler(orderQueryService, logger))
			orders.GET("/status/:status", listOrdersByStatusHandler(orderQueryService, logger))
			orders.GET("/customer/:customerId", listOrdersByCustomerHandler(orderQueryService, logger))
		}

		// Reprocessing endpoints (for orchestrator activities)
		reprocessing := v1.Group("/reprocessing")
		{
			// Get orders eligible for retry
			reprocessing.GET("/eligible", getEligibleOrdersHandler(reprocessingService, logger))

			// Order-specific retry operations
			reprocessing.GET("/orders/:orderId/retry-count", getRetryMetadataHandler(reprocessingService, logger))
			reprocessing.POST("/orders/:orderId/retry-count", incrementRetryCountHandler(reprocessingService, logger))
			reprocessing.POST("/orders/:orderId/reset", resetOrderForRetryHandler(reprocessingService, logger))
			reprocessing.POST("/orders/:orderId/dlq", moveToDeadLetterHandler(reprocessingService, logger))
		}

		// Dead Letter Queue endpoints
		dlq := v1.Group("/dead-letter-queue")
		{
			dlq.GET("", listDeadLetterQueueHandler(reprocessingService, logger))
			dlq.GET("/stats", getDLQStatsHandler(reprocessingService, logger))
			dlq.GET("/:orderId", getDeadLetterEntryHandler(reprocessingService, logger))
			dlq.PATCH("/:orderId/resolve", resolveDLQEntryHandler(reprocessingService, logger))
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

// AssignWaveRequest is the request body for assigning an order to a wave
type AssignWaveRequest struct {
	WaveID string `json:"waveId" binding:"required"`
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

func assignWaveHandler(service *application.OrderApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req AssignWaveRequest
		if appErr := middleware.BindAndValidate(c, &req); appErr != nil {
			responder.RespondWithAppError(appErr)
			return
		}

		cmd := application.AssignToWaveCommand{
			OrderID: c.Param("orderId"),
			WaveID:  req.WaveID,
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": cmd.OrderID,
			"wave.id":  cmd.WaveID,
		})

		order, err := service.AssignToWave(c.Request.Context(), cmd)
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

func startPickingHandler(service *application.OrderApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		cmd := application.StartPickingCommand{
			OrderID: c.Param("orderId"),
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": cmd.OrderID,
		})

		order, err := service.StartPicking(c.Request.Context(), cmd)
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

func markConsolidatedHandler(service *application.OrderApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		cmd := application.MarkConsolidatedCommand{
			OrderID: c.Param("orderId"),
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": cmd.OrderID,
		})

		order, err := service.MarkConsolidated(c.Request.Context(), cmd)
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

func markPackedHandler(service *application.OrderApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		cmd := application.MarkPackedCommand{
			OrderID: c.Param("orderId"),
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": cmd.OrderID,
		})

		order, err := service.MarkPacked(c.Request.Context(), cmd)
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

// --- Reprocessing Handlers ---

// IncrementRetryCountRequest is the request body for incrementing retry count
type IncrementRetryCountRequest struct {
	FailureStatus string `json:"failureStatus" binding:"required"`
	FailureReason string `json:"failureReason"`
	WorkflowID    string `json:"workflowId"`
	RunID         string `json:"runId"`
}

// MoveToDLQRequest is the request body for moving an order to DLQ
type MoveToDLQRequest struct {
	FailureStatus string `json:"failureStatus" binding:"required"`
	FailureReason string `json:"failureReason"`
	RetryCount    int    `json:"retryCount"`
	WorkflowID    string `json:"workflowId"`
	RunID         string `json:"runId"`
}

// ResolveDLQRequest is the request body for resolving a DLQ entry
type ResolveDLQRequest struct {
	Resolution string `json:"resolution" binding:"required,oneof=manual_retry cancelled escalated"`
	Notes      string `json:"notes"`
	ResolvedBy string `json:"resolvedBy"`
}

func getEligibleOrdersHandler(service *application.ReprocessingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		// Parse query parameters
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		maxRetries, _ := strconv.Atoi(c.DefaultQuery("maxRetries", "5"))

		// Parse status array
		var failureStatuses []string
		if statuses := c.QueryArray("status"); len(statuses) > 0 {
			failureStatuses = statuses
		}

		query := application.GetEligibleOrdersQuery{
			FailureStatuses: failureStatuses,
			MaxRetries:      maxRetries,
			Limit:           limit,
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"query.limit":      limit,
			"query.maxRetries": maxRetries,
		})

		result, err := service.GetEligibleOrders(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func getRetryMetadataHandler(service *application.ReprocessingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)
		orderID := c.Param("orderId")

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": orderID,
		})

		result, err := service.GetRetryMetadata(c.Request.Context(), orderID)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

func incrementRetryCountHandler(service *application.ReprocessingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)
		orderID := c.Param("orderId")

		var req IncrementRetryCountRequest
		if appErr := middleware.BindAndValidate(c, &req); appErr != nil {
			responder.RespondWithAppError(appErr)
			return
		}

		cmd := application.IncrementRetryCountCommand{
			OrderID:       orderID,
			FailureStatus: req.FailureStatus,
			FailureReason: req.FailureReason,
			WorkflowID:    req.WorkflowID,
			RunID:         req.RunID,
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id":       orderID,
			"failure.status": req.FailureStatus,
		})

		if err := service.IncrementRetryCount(c.Request.Context(), cmd); err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Retry count incremented",
		})
	}
}

func resetOrderForRetryHandler(service *application.ReprocessingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)
		orderID := c.Param("orderId")

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": orderID,
		})

		result, err := service.ResetOrderForRetry(c.Request.Context(), orderID)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    result,
		})
	}
}

func moveToDeadLetterHandler(service *application.ReprocessingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)
		orderID := c.Param("orderId")

		var req MoveToDLQRequest
		if appErr := middleware.BindAndValidate(c, &req); appErr != nil {
			responder.RespondWithAppError(appErr)
			return
		}

		cmd := application.MoveToDLQCommand{
			OrderID:       orderID,
			FailureStatus: req.FailureStatus,
			FailureReason: req.FailureReason,
			RetryCount:    req.RetryCount,
			WorkflowID:    req.WorkflowID,
			RunID:         req.RunID,
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id":       orderID,
			"failure.status": req.FailureStatus,
			"retry.count":    req.RetryCount,
		})

		if err := service.MoveToDeadLetterQueue(c.Request.Context(), cmd); err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "Order moved to dead letter queue",
		})
	}
}

// --- Dead Letter Queue Handlers ---

func listDeadLetterQueueHandler(service *application.ReprocessingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		// Parse query parameters
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		query := application.ListDLQQuery{
			Limit:  limit,
			Offset: offset,
		}

		// Optional filters
		if resolved := c.Query("resolved"); resolved != "" {
			r := strings.ToLower(resolved) == "true"
			query.Resolved = &r
		}
		if status := c.Query("failureStatus"); status != "" {
			query.FailureStatus = &status
		}
		if customerID := c.Query("customerId"); customerID != "" {
			query.CustomerID = &customerID
		}
		if hours := c.Query("olderThanHours"); hours != "" {
			if h, err := strconv.ParseFloat(hours, 64); err == nil {
				query.OlderThanHours = &h
			}
		}

		result, err := service.ListDeadLetterQueue(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func getDLQStatsHandler(service *application.ReprocessingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		stats, err := service.GetDLQStats(c.Request.Context())
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": stats})
	}
}

func getDeadLetterEntryHandler(service *application.ReprocessingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)
		orderID := c.Param("orderId")

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": orderID,
		})

		result, err := service.GetDeadLetterEntry(c.Request.Context(), orderID)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

func resolveDLQEntryHandler(service *application.ReprocessingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)
		orderID := c.Param("orderId")

		var req ResolveDLQRequest
		if appErr := middleware.BindAndValidate(c, &req); appErr != nil {
			responder.RespondWithAppError(appErr)
			return
		}

		cmd := application.ResolveDLQCommand{
			OrderID:    orderID,
			Resolution: req.Resolution,
			Notes:      req.Notes,
			ResolvedBy: req.ResolvedBy,
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id":   orderID,
			"resolution": req.Resolution,
		})

		if err := service.ResolveDLQEntry(c.Request.Context(), cmd); err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    "Dead letter entry resolved",
			"resolution": req.Resolution,
		})
	}
}
