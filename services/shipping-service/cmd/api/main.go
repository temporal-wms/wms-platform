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
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/tracing"

	"github.com/wms-platform/shipping-service/internal/application"
	"github.com/wms-platform/shipping-service/internal/domain"
	mongoRepo "github.com/wms-platform/shipping-service/internal/infrastructure/mongodb"
)

const serviceName = "shipping-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting shipping-service API")

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
	eventFactory := cloudevents.NewEventFactory("/shipping-service")

	// Initialize repositories with instrumented client and event factory
	repo := mongoRepo.NewShipmentRepository(instrumentedMongo.Database(), eventFactory)
	manifestRepo := mongoRepo.NewManifestRepository(instrumentedMongo.Database(), eventFactory)

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
	shippingService := application.NewShippingApplicationService(
		repo,
		instrumentedProducer,
		eventFactory,
		logger,
	)
	manifestService := application.NewManifestApplicationService(
		manifestRepo,
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

	// API v1 routes - Shipments
	api := router.Group("/api/v1/shipments")
	{
		api.POST("", createShipmentHandler(shippingService, logger))
		api.GET("/:shipmentId", getShipmentHandler(shippingService, logger))
		api.POST("/:shipmentId/label", generateLabelHandler(shippingService, logger))
		api.POST("/:shipmentId/manifest", addToManifestHandler(shippingService, logger))
		api.POST("/:shipmentId/ship", confirmShipmentHandler(shippingService, logger))
		api.GET("/order/:orderId", getByOrderHandler(shippingService, logger))
		api.GET("/tracking/:trackingNumber", getByTrackingHandler(shippingService, logger))
		api.GET("/status/:status", getByStatusHandler(shippingService, logger))
		api.GET("/carrier/:carrierCode", getByCarrierHandler(shippingService, logger))
		api.GET("/carrier/:carrierCode/pending", getPendingForManifestHandler(shippingService, logger))
	}

	// API v1 routes - Manifests
	manifestAPI := router.Group("/api/v1/manifests")
	{
		manifestAPI.POST("", createManifestHandler(manifestService, logger))
		manifestAPI.GET("/:manifestId", getManifestHandler(manifestService, logger))
		manifestAPI.POST("/:manifestId/packages", addPackageToManifestHandler(manifestService, logger))
		manifestAPI.POST("/:manifestId/close", closeManifestHandler(manifestService, logger))
		manifestAPI.POST("/:manifestId/trailer", assignTrailerHandler(manifestService, logger))
		manifestAPI.POST("/:manifestId/dispatch", dispatchManifestHandler(manifestService, logger))
		manifestAPI.GET("/carrier/:carrierId", getManifestsByCarrierHandler(manifestService, logger))
		manifestAPI.GET("/carrier/:carrierId/closed", getClosedManifestsByCarrierHandler(manifestService, logger))
		manifestAPI.GET("/status/:status", getManifestsByStatusHandler(manifestService, logger))
		manifestAPI.GET("/dispatched/today", getDispatchedTodayHandler(manifestService, logger))
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
		ServerAddr: getEnv("SERVER_ADDR", ":8007"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "shipping_db"),
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
func createShipmentHandler(service *application.ShippingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			ShipmentID string            `json:"shipmentId" binding:"required"`
			OrderID    string            `json:"orderId" binding:"required"`
			PackageID  string            `json:"packageId" binding:"required"`
			WaveID     string            `json:"waveId"`
			Carrier    domain.Carrier    `json:"carrier" binding:"required"`
			Package    domain.PackageInfo `json:"package" binding:"required"`
			Recipient  domain.Address    `json:"recipient" binding:"required"`
			Shipper    domain.Address    `json:"shipper" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.id": req.ShipmentID,
			"order.id":    req.OrderID,
			"package.id":  req.PackageID,
		})

		cmd := application.CreateShipmentCommand{
			ShipmentID: req.ShipmentID,
			OrderID:    req.OrderID,
			PackageID:  req.PackageID,
			WaveID:     req.WaveID,
			Carrier:    req.Carrier,
			Package:    req.Package,
			Recipient:  req.Recipient,
			Shipper:    req.Shipper,
		}

		shipment, err := service.CreateShipment(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusCreated, shipment)
	}
}

func getShipmentHandler(service *application.ShippingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		query := application.GetShipmentQuery{ShipmentID: c.Param("shipmentId")}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.id": query.ShipmentID,
		})

		shipment, err := service.GetShipment(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, shipment)
	}
}

func generateLabelHandler(service *application.ShippingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		shipmentID := c.Param("shipmentId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.id": shipmentID,
		})

		var label domain.ShippingLabel
		if err := c.ShouldBindJSON(&label); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"tracking.number": label.TrackingNumber,
		})

		cmd := application.GenerateLabelCommand{
			ShipmentID: shipmentID,
			Label:      label,
		}

		shipment, err := service.GenerateLabel(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, shipment)
	}
}

func addToManifestHandler(service *application.ShippingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		shipmentID := c.Param("shipmentId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.id": shipmentID,
		})

		var manifest domain.Manifest
		if err := c.ShouldBindJSON(&manifest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"manifest.id": manifest.ManifestID,
		})

		cmd := application.AddToManifestCommand{
			ShipmentID: shipmentID,
			Manifest:   manifest,
		}

		shipment, err := service.AddToManifest(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, shipment)
	}
}

func confirmShipmentHandler(service *application.ShippingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		shipmentID := c.Param("shipmentId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.id": shipmentID,
		})

		var req struct {
			EstimatedDelivery *time.Time `json:"estimatedDelivery"`
		}
		c.ShouldBindJSON(&req)

		cmd := application.ConfirmShipmentCommand{
			ShipmentID:        shipmentID,
			EstimatedDelivery: req.EstimatedDelivery,
		}

		shipment, err := service.ConfirmShipment(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, shipment)
	}
}

func getByOrderHandler(service *application.ShippingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		orderID := c.Param("orderId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": orderID,
		})

		query := application.GetByOrderQuery{OrderID: orderID}

		shipment, err := service.GetByOrder(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, shipment)
	}
}

func getByTrackingHandler(service *application.ShippingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		trackingNumber := c.Param("trackingNumber")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"tracking.number": trackingNumber,
		})

		query := application.GetByTrackingQuery{TrackingNumber: trackingNumber}

		shipment, err := service.GetByTracking(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, shipment)
	}
}

func getByStatusHandler(service *application.ShippingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		status := c.Param("status")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.status": status,
		})

		query := application.GetByStatusQuery{Status: status}

		shipments, err := service.GetByStatus(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, shipments)
	}
}

func getByCarrierHandler(service *application.ShippingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		carrierCode := c.Param("carrierCode")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"carrier.code": carrierCode,
		})

		query := application.GetByCarrierQuery{CarrierCode: carrierCode}

		shipments, err := service.GetByCarrier(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, shipments)
	}
}

func getPendingForManifestHandler(service *application.ShippingApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		carrierCode := c.Param("carrierCode")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"carrier.code": carrierCode,
		})

		query := application.GetPendingForManifestQuery{CarrierCode: carrierCode}

		shipments, err := service.GetPendingForManifest(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, shipments)
	}
}

// Manifest Handlers

func createManifestHandler(service *application.ManifestApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var cmd application.CreateManifestCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"carrier.id": cmd.CarrierID,
		})

		manifest, err := service.CreateManifest(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusCreated, manifest)
	}
}

func getManifestHandler(service *application.ManifestApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		manifestID := c.Param("manifestId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"manifest.id": manifestID,
		})

		query := application.GetManifestQuery{ManifestID: manifestID}
		manifest, err := service.GetManifest(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, manifest)
	}
}

func addPackageToManifestHandler(service *application.ManifestApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		manifestID := c.Param("manifestId")

		var cmd application.AddPackageCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cmd.ManifestID = manifestID

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"manifest.id": manifestID,
			"package.id":  cmd.PackageID,
		})

		manifest, err := service.AddPackage(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, manifest)
	}
}

func closeManifestHandler(service *application.ManifestApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		manifestID := c.Param("manifestId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"manifest.id": manifestID,
		})

		cmd := application.CloseManifestCommand{ManifestID: manifestID}
		manifest, err := service.CloseManifest(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, manifest)
	}
}

func assignTrailerHandler(service *application.ManifestApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		manifestID := c.Param("manifestId")

		var req struct {
			TrailerID    string `json:"trailerId" binding:"required"`
			DispatchDock string `json:"dispatchDock" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"manifest.id":   manifestID,
			"trailer.id":    req.TrailerID,
			"dispatch.dock": req.DispatchDock,
		})

		cmd := application.AssignTrailerCommand{
			ManifestID:   manifestID,
			TrailerID:    req.TrailerID,
			DispatchDock: req.DispatchDock,
		}

		manifest, err := service.AssignTrailer(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, manifest)
	}
}

func dispatchManifestHandler(service *application.ManifestApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		manifestID := c.Param("manifestId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"manifest.id": manifestID,
		})

		cmd := application.DispatchManifestCommand{ManifestID: manifestID}
		manifest, err := service.DispatchManifest(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, manifest)
	}
}

func getManifestsByCarrierHandler(service *application.ManifestApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		carrierID := c.Param("carrierId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"carrier.id": carrierID,
		})

		query := application.GetManifestsByCarrierQuery{CarrierID: carrierID}
		manifests, err := service.GetByCarrier(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, manifests)
	}
}

func getClosedManifestsByCarrierHandler(service *application.ManifestApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		carrierID := c.Param("carrierId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"carrier.id": carrierID,
		})

		manifests, err := service.GetClosedByCarrier(c.Request.Context(), carrierID)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, manifests)
	}
}

func getManifestsByStatusHandler(service *application.ManifestApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		status := c.Param("status")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"manifest.status": status,
		})

		query := application.GetManifestsByStatusQuery{Status: status}
		manifests, err := service.GetByStatus(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, manifests)
	}
}

func getDispatchedTodayHandler(service *application.ManifestApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		manifests, err := service.GetDispatchedToday(c.Request.Context())
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, manifests)
	}
}
