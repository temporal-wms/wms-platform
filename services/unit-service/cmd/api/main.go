package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/idempotency"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/tracing"

	"github.com/wms-platform/services/unit-service/internal/api/dto"
	"github.com/wms-platform/services/unit-service/internal/application"
	mongoRepo "github.com/wms-platform/services/unit-service/internal/infrastructure/mongodb"
)

const serviceName = "unit-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting unit-service API")

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

	// Initialize repositories
	unitRepo := mongoRepo.NewUnitRepository(instrumentedMongo.Database())
	exceptionRepo := mongoRepo.NewUnitExceptionRepository(instrumentedMongo.Database())

	// Initialize idempotency repository
	idempotencyKeyRepo := idempotency.NewMongoKeyRepository(instrumentedMongo.Database())
	logger.Info("Idempotency repositories initialized")

	// Initialize application service (no publisher for now)
	unitService := application.NewUnitService(unitRepo, exceptionRepo, nil)

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

	// API v1 routes
	api := router.Group("/api/v1/units")
	{
		api.POST("", createUnitsHandler(unitService, logger))
		api.POST("/reserve", reserveUnitsHandler(unitService, logger))
		api.GET("/order/:orderId", getUnitsForOrderHandler(unitService, logger))
		api.GET("/:unitId", getUnitHandler(unitService, logger))
		api.GET("/:unitId/audit", getAuditTrailHandler(unitService, logger))
		api.POST("/:unitId/pick", confirmPickHandler(unitService, logger))
		api.POST("/:unitId/consolidate", confirmConsolidationHandler(unitService, logger))
		api.POST("/:unitId/pack", confirmPackedHandler(unitService, logger))
		api.POST("/:unitId/ship", confirmShippedHandler(unitService, logger))
		api.POST("/:unitId/exception", createExceptionHandler(unitService, logger))
	}

	// Exception routes
	exceptions := router.Group("/api/v1/exceptions")
	{
		exceptions.GET("/order/:orderId", getExceptionsForOrderHandler(unitService, logger))
		exceptions.GET("/unresolved", getUnresolvedExceptionsHandler(unitService, logger))
		exceptions.POST("/:exceptionId/resolve", resolveExceptionHandler(unitService, logger))
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
}

func loadConfig() *Config {
	return &Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8014"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "unit_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
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

func createUnitsHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateUnitsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := service.CreateUnits(c.Request.Context(), application.CreateUnitsCommand{
			SKU:        req.SKU,
			ShipmentID: req.ShipmentID,
			LocationID: req.LocationID,
			Quantity:   req.Quantity,
			CreatedBy:  req.CreatedBy,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, dto.CreateUnitsResponse{
			UnitIDs: result.UnitIDs,
			SKU:     result.SKU,
			Count:   result.Count,
		})
	}
}

func reserveUnitsHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ReserveUnitsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		items := make([]application.ReserveItemSpec, len(req.Items))
		for i, item := range req.Items {
			items[i] = application.ReserveItemSpec{
				SKU:      item.SKU,
				Quantity: item.Quantity,
			}
		}

		result, err := service.ReserveUnits(c.Request.Context(), application.ReserveUnitsCommand{
			OrderID:   req.OrderID,
			PathID:    req.PathID,
			Items:     items,
			HandlerID: req.HandlerID,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := dto.ReserveUnitsResponse{
			ReservedUnits: make([]dto.ReservedUnitInfo, len(result.ReservedUnits)),
			FailedItems:   make([]dto.FailedReserve, len(result.FailedItems)),
		}
		for i, u := range result.ReservedUnits {
			resp.ReservedUnits[i] = dto.ReservedUnitInfo{
				UnitID:     u.UnitID,
				SKU:        u.SKU,
				LocationID: u.LocationID,
			}
		}
		for i, f := range result.FailedItems {
			resp.FailedItems[i] = dto.FailedReserve{
				SKU:       f.SKU,
				Requested: f.Requested,
				Available: f.Available,
				Reason:    f.Reason,
			}
		}

		c.JSON(http.StatusOK, resp)
	}
}

func getUnitsForOrderHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("orderId")

		units, err := service.GetUnitsForOrder(c.Request.Context(), orderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := dto.UnitListResponse{
			Units: make([]dto.UnitSummary, len(units)),
			Total: len(units),
		}
		for i, u := range units {
			resp.Units[i] = dto.UnitSummary{
				UnitID:   u.UnitID,
				SKU:      u.SKU,
				OrderID:  u.OrderID,
				Status:   string(u.Status),
				Location: u.CurrentLocationID,
			}
		}

		c.JSON(http.StatusOK, resp)
	}
}

func getUnitHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		unitID := c.Param("unitId")

		unit, err := service.GetUnit(c.Request.Context(), unitID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
			return
		}

		c.JSON(http.StatusOK, dto.ToUnitResponse(unit))
	}
}

func getAuditTrailHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		unitID := c.Param("unitId")

		movements, err := service.GetUnitAuditTrail(c.Request.Context(), unitID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
			return
		}

		movementDTOs := make([]dto.UnitMovementDTO, len(movements))
		for i, m := range movements {
			movementDTOs[i] = dto.UnitMovementDTO{
				MovementID:     m.MovementID,
				FromLocationID: m.FromLocationID,
				ToLocationID:   m.ToLocationID,
				FromStatus:     string(m.FromStatus),
				ToStatus:       string(m.ToStatus),
				StationID:      m.StationID,
				HandlerID:      m.HandlerID,
				Timestamp:      m.Timestamp,
				Notes:          m.Notes,
			}
		}

		c.JSON(http.StatusOK, dto.AuditTrailResponse{
			UnitID:    unitID,
			Movements: movementDTOs,
		})
	}
}

func confirmPickHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		unitID := c.Param("unitId")

		var req dto.ConfirmPickRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := service.ConfirmPick(c.Request.Context(), application.ConfirmPickCommand{
			UnitID:    unitID,
			ToteID:    req.ToteID,
			PickerID:  req.PickerID,
			StationID: req.StationID,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "picked"})
	}
}

func confirmConsolidationHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		unitID := c.Param("unitId")

		var req dto.ConfirmConsolidationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := service.ConfirmConsolidation(c.Request.Context(), application.ConfirmConsolidationCommand{
			UnitID:         unitID,
			DestinationBin: req.DestinationBin,
			WorkerID:       req.WorkerID,
			StationID:      req.StationID,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "consolidated"})
	}
}

func confirmPackedHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		unitID := c.Param("unitId")

		var req dto.ConfirmPackedRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := service.ConfirmPacked(c.Request.Context(), application.ConfirmPackedCommand{
			UnitID:    unitID,
			PackageID: req.PackageID,
			PackerID:  req.PackerID,
			StationID: req.StationID,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "packed"})
	}
}

func confirmShippedHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		unitID := c.Param("unitId")

		var req dto.ConfirmShippedRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := service.ConfirmShipped(c.Request.Context(), application.ConfirmShippedCommand{
			UnitID:         unitID,
			ShipmentID:     req.ShipmentID,
			TrackingNumber: req.TrackingNumber,
			HandlerID:      req.HandlerID,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "shipped"})
	}
}

func createExceptionHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		unitID := c.Param("unitId")

		var req dto.CreateExceptionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		exception, err := service.CreateException(c.Request.Context(), application.CreateExceptionCommand{
			UnitID:        unitID,
			ExceptionType: req.ToExceptionType(),
			Stage:         req.ToExceptionStage(),
			Description:   req.Description,
			StationID:     req.StationID,
			ReportedBy:    req.ReportedBy,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, dto.ToExceptionResponse(exception))
	}
}

func getExceptionsForOrderHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("orderId")

		exceptions, err := service.GetExceptionsForOrder(c.Request.Context(), orderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := dto.ExceptionListResponse{
			Exceptions: make([]dto.ExceptionResponse, len(exceptions)),
			Total:      len(exceptions),
		}
		for i, e := range exceptions {
			resp.Exceptions[i] = dto.ToExceptionResponse(e)
		}

		c.JSON(http.StatusOK, resp)
	}
}

func getUnresolvedExceptionsHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		exceptions, err := service.GetUnresolvedExceptions(c.Request.Context(), 100)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := dto.ExceptionListResponse{
			Exceptions: make([]dto.ExceptionResponse, len(exceptions)),
			Total:      len(exceptions),
		}
		for i, e := range exceptions {
			resp.Exceptions[i] = dto.ToExceptionResponse(e)
		}

		c.JSON(http.StatusOK, resp)
	}
}

func resolveExceptionHandler(service *application.UnitService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		exceptionID := c.Param("exceptionId")

		var req dto.ResolveExceptionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := service.ResolveException(c.Request.Context(), application.ResolveExceptionCommand{
			ExceptionID: exceptionID,
			Resolution:  req.Resolution,
			ResolvedBy:  req.ResolvedBy,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "resolved"})
	}
}
