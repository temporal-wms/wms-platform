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

	"github.com/wms-platform/inventory-service/internal/application"
	mongoRepo "github.com/wms-platform/inventory-service/internal/infrastructure/mongodb"
	"github.com/wms-platform/inventory-service/internal/infrastructure/projections"
)

const serviceName = "inventory-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting inventory-service API")

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
	eventFactory := cloudevents.NewEventFactory("/inventory-service")

	// Initialize repositories with instrumented client and event factory
	repo := mongoRepo.NewInventoryRepository(instrumentedMongo.Database(), eventFactory)

	// Initialize idempotency repository
	idempotencyKeyRepo := idempotency.NewMongoKeyRepository(instrumentedMongo.Database())
	logger.Info("Idempotency repositories initialized")

	// Initialize CQRS projection repository
	projectionRepo := projections.NewMongoInventoryListProjectionRepository(instrumentedMongo.Database())
	logger.Info("Projection repository initialized")

	// Initialize inventory projector for CQRS read model
	projector := projections.NewInventoryProjector(projectionRepo, repo, logger)
	logger.Info("Inventory projector initialized")

	// Initialize query service for optimized reads
	queryService := application.NewInventoryQueryService(projectionRepo, logger)
	_ = queryService // TODO: Use query service in read handlers for better performance
	logger.Info("Query service initialized")

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

	// Initialize application service with projector for CQRS
	inventoryService := application.NewInventoryApplicationService(
		repo,
		instrumentedProducer,
		eventFactory,
		projector,
		logger,
	)

	// Initialize ledger repositories and service (optional feature)
	ledgerRepo := mongoRepo.NewInventoryLedgerRepository(instrumentedMongo.Database(), eventFactory)
	entryRepo := mongoRepo.NewLedgerEntryRepository(instrumentedMongo.Database(), eventFactory)
	ledgerService := application.NewLedgerApplicationService(ledgerRepo, entryRepo)

	// Set ledger service on inventory service to enable double-entry bookkeeping
	inventoryService.SetLedgerService(ledgerService)
	logger.Info("Ledger service initialized and integrated")

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
	api := router.Group("/api/v1/inventory")
	api.Use(middleware.RequireTenantAuth()) // All API routes require tenant headers
	{
		// Static routes first (must come before wildcard routes)
		api.POST("", createItemHandler(inventoryService, logger))
		api.GET("", listInventoryHandler(inventoryService, logger))
		api.GET("/location/:locationId", getByLocationHandler(inventoryService, logger))
		api.GET("/zone/:zone", getByZoneHandler(inventoryService, logger))
		api.GET("/low-stock", getLowStockHandler(inventoryService, logger))
		api.POST("/reserve", reserveBulkHandler(inventoryService, logger))
		api.POST("/release/:orderId", releaseByOrderHandler(inventoryService, logger))

		// Wildcard SKU routes (must come after static routes)
		api.GET("/:sku", getItemHandler(inventoryService, logger))
		api.POST("/:sku/receive", receiveStockHandler(inventoryService, logger))
		api.POST("/:sku/reserve", reserveHandler(inventoryService, logger))
		api.POST("/:sku/pick", pickHandler(inventoryService, logger))
		api.POST("/:sku/release", releaseReservationHandler(inventoryService, logger))
		api.POST("/:sku/adjust", adjustHandler(inventoryService, logger))

		// Hard allocation routes (physical staging lifecycle)
		api.POST("/:sku/stage", stageHandler(inventoryService, logger))
		api.POST("/:sku/pack", packHandler(inventoryService, logger))
		api.POST("/:sku/ship", shipHandler(inventoryService, logger))
		api.POST("/:sku/return-to-shelf", returnToShelfHandler(inventoryService, logger))

		// Shortage handling routes
		api.POST("/:sku/shortage", recordShortageHandler(inventoryService, logger))

		// Ledger routes (double-entry accounting)
		api.GET("/:sku/ledger", getLedgerHandler(ledgerService, logger))
		api.GET("/:sku/ledger/entries", getLedgerEntriesHandler(ledgerService, logger))
		api.GET("/ledger/transactions/:transactionId", getLedgerTransactionHandler(ledgerService, logger))
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
		ServerAddr: getEnv("SERVER_ADDR", ":8008"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "inventory_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Kafka: &kafka.Config{
			Brokers:       []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			ConsumerGroup: "inventory-service",
			ClientID:      "inventory-service",
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

func createItemHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			SKU             string `json:"sku" binding:"required"`
			ProductName     string `json:"productName" binding:"required"`
			ReorderPoint    int    `json:"reorderPoint"`
			ReorderQuantity int    `json:"reorderQuantity"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.CreateItemCommand{
			SKU:             req.SKU,
			ProductName:     req.ProductName,
			ReorderPoint:    req.ReorderPoint,
			ReorderQuantity: req.ReorderQuantity,
		}

		item, err := service.CreateItem(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusCreated, item)
	}
}

func getItemHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		query := application.GetItemQuery{SKU: c.Param("sku")}

		item, err := service.GetItem(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

func receiveStockHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			LocationID  string `json:"locationId" binding:"required"`
			Zone        string `json:"zone"`
			Quantity    int    `json:"quantity" binding:"required"`
			ReferenceID string `json:"referenceId"`
			CreatedBy   string `json:"createdBy" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.ReceiveStockCommand{
			SKU:         c.Param("sku"),
			LocationID:  req.LocationID,
			Zone:        req.Zone,
			Quantity:    req.Quantity,
			ReferenceID: req.ReferenceID,
			CreatedBy:   req.CreatedBy,
		}

		item, err := service.ReceiveStock(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

func reserveHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			OrderID    string `json:"orderId" binding:"required"`
			LocationID string `json:"locationId" binding:"required"`
			Quantity   int    `json:"quantity" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.ReserveCommand{
			SKU:        c.Param("sku"),
			OrderID:    req.OrderID,
			LocationID: req.LocationID,
			Quantity:   req.Quantity,
		}

		item, err := service.Reserve(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

func reserveBulkHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			OrderID string `json:"orderId" binding:"required"`
			Items   []struct {
				SKU        string `json:"sku" binding:"required"`
				Quantity   int    `json:"quantity" binding:"required"`
				LocationID string `json:"locationId"` // Optional - service will auto-select if not provided
			} `json:"items" binding:"required,min=1"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Convert to command
		items := make([]application.ReserveInventoryBulkItem, len(req.Items))
		for i, item := range req.Items {
			items[i] = application.ReserveInventoryBulkItem{
				SKU:        item.SKU,
				Quantity:   item.Quantity,
				LocationID: item.LocationID,
			}
		}

		cmd := application.ReserveInventoryBulkCommand{
			OrderID: req.OrderID,
			Items:   items,
		}

		err := service.ReserveBulk(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"orderId":   req.OrderID,
			"message":   "Inventory reserved successfully",
			"itemCount": len(items),
		})
	}
}

func pickHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			OrderID    string `json:"orderId" binding:"required"`
			LocationID string `json:"locationId" binding:"required"`
			Quantity   int    `json:"quantity" binding:"required"`
			CreatedBy  string `json:"createdBy" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.PickCommand{
			SKU:        c.Param("sku"),
			OrderID:    req.OrderID,
			LocationID: req.LocationID,
			Quantity:   req.Quantity,
			CreatedBy:  req.CreatedBy,
		}

		item, err := service.Pick(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

func releaseReservationHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			OrderID string `json:"orderId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.ReleaseReservationCommand{
			SKU:     c.Param("sku"),
			OrderID: req.OrderID,
		}

		item, err := service.ReleaseReservation(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

func adjustHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			LocationID  string `json:"locationId" binding:"required"`
			NewQuantity int    `json:"newQuantity" binding:"required"`
			Reason      string `json:"reason" binding:"required"`
			CreatedBy   string `json:"createdBy" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.AdjustCommand{
			SKU:         c.Param("sku"),
			LocationID:  req.LocationID,
			NewQuantity: req.NewQuantity,
			Reason:      req.Reason,
			CreatedBy:   req.CreatedBy,
		}

		item, err := service.Adjust(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

func getByLocationHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := application.GetByLocationQuery{LocationID: c.Param("locationId")}
		items, _ := service.GetByLocation(c.Request.Context(), query)
		c.JSON(http.StatusOK, items)
	}
}

func getByZoneHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := application.GetByZoneQuery{Zone: c.Param("zone")}
		items, _ := service.GetByZone(c.Request.Context(), query)
		c.JSON(http.StatusOK, items)
	}
}

func getLowStockHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, _ := service.GetLowStock(c.Request.Context())
		c.JSON(http.StatusOK, items)
	}
}

func listInventoryHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		query := application.ListInventoryQuery{
			Limit:  limit,
			Offset: offset,
		}

		items, _ := service.ListInventory(c.Request.Context(), query)
		c.JSON(http.StatusOK, items)
	}
}

func releaseByOrderHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		orderID := c.Param("orderId")
		if orderID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
			return
		}

		cmd := application.ReleaseByOrderCommand{
			OrderID: orderID,
		}

		releasedCount, err := service.ReleaseByOrder(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"orderId":       orderID,
			"releasedCount": releasedCount,
			"message":       "Reservations released successfully",
		})
	}
}

// stageHandler converts a soft reservation to hard allocation (physical staging)
func stageHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			ReservationID     string `json:"reservationId" binding:"required"`
			StagingLocationID string `json:"stagingLocationId" binding:"required"`
			StagedBy          string `json:"stagedBy" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.StageCommand{
			SKU:               c.Param("sku"),
			ReservationID:     req.ReservationID,
			StagingLocationID: req.StagingLocationID,
			StagedBy:          req.StagedBy,
		}

		item, err := service.Stage(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

// packHandler marks a hard allocation as packed
func packHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			AllocationID string `json:"allocationId" binding:"required"`
			PackedBy     string `json:"packedBy" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.PackCommand{
			SKU:          c.Param("sku"),
			AllocationID: req.AllocationID,
			PackedBy:     req.PackedBy,
		}

		item, err := service.Pack(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

// shipHandler ships a packed allocation (removes inventory from system)
func shipHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			AllocationID string `json:"allocationId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.ShipCommand{
			SKU:          c.Param("sku"),
			AllocationID: req.AllocationID,
		}

		item, err := service.Ship(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

// returnToShelfHandler returns hard allocated inventory back to shelf
func returnToShelfHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			AllocationID string `json:"allocationId" binding:"required"`
			ReturnedBy   string `json:"returnedBy" binding:"required"`
			Reason       string `json:"reason" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.ReturnToShelfCommand{
			SKU:          c.Param("sku"),
			AllocationID: req.AllocationID,
			ReturnedBy:   req.ReturnedBy,
			Reason:       req.Reason,
		}

		item, err := service.ReturnToShelf(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

// recordShortageHandler records a confirmed stock shortage discovered during picking
func recordShortageHandler(service *application.InventoryApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			LocationID  string `json:"locationId" binding:"required"`
			OrderID     string `json:"orderId" binding:"required"`
			ExpectedQty int    `json:"expectedQty" binding:"required"`
			ActualQty   int    `json:"actualQty" binding:"required"`
			Reason      string `json:"reason" binding:"required"` // not_found, damaged, quantity_mismatch
			ReportedBy  string `json:"reportedBy" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.RecordShortageCommand{
			SKU:         c.Param("sku"),
			LocationID:  req.LocationID,
			OrderID:     req.OrderID,
			ExpectedQty: req.ExpectedQty,
			ActualQty:   req.ActualQty,
			Reason:      req.Reason,
			ReportedBy:  req.ReportedBy,
		}

		item, err := service.RecordShortage(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

// Ledger Handlers

func getLedgerHandler(service *application.LedgerApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		sku := c.Param("sku")
		tenantID := c.GetString("tenantId")
		facilityID := c.GetString("facilityId")

		if tenantID == "" || facilityID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenantId and facilityId are required"})
			return
		}

		query := application.GetLedgerQuery{
			SKU:        sku,
			TenantID:   tenantID,
			FacilityID: facilityID,
		}

		ledger, err := service.GetLedger(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, ledger)
	}
}

func getLedgerEntriesHandler(service *application.LedgerApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		sku := c.Param("sku")
		tenantID := c.GetString("tenantId")
		facilityID := c.GetString("facilityId")

		if tenantID == "" || facilityID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenantId and facilityId are required"})
			return
		}

		limit := 50
		if limitStr := c.Query("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}

		query := application.GetLedgerEntriesQuery{
			SKU:        sku,
			TenantID:   tenantID,
			FacilityID: facilityID,
			Limit:      limit,
		}

		entries, err := service.GetLedgerEntries(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"entries": entries,
			"count":   len(entries),
		})
	}
}

func getLedgerTransactionHandler(service *application.LedgerApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		transactionID := c.Param("transactionId")
		tenantID := c.GetString("tenantId")

		if tenantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenantId is required"})
			return
		}

		query := application.GetLedgerByTransactionQuery{
			TransactionID: transactionID,
			TenantID:      tenantID,
		}

		transaction, err := service.GetTransactionEntries(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, transaction)
	}
}
