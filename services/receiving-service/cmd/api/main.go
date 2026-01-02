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

	"github.com/wms-platform/services/receiving-service/internal/api/dto"
	"github.com/wms-platform/services/receiving-service/internal/application"
	"github.com/wms-platform/services/receiving-service/internal/domain"
	mongoRepo "github.com/wms-platform/services/receiving-service/internal/infrastructure/mongodb"
)

const serviceName = "receiving-service"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting receiving-service API")

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
	eventFactory := cloudevents.NewEventFactory("/receiving-service")

	// Initialize repositories with instrumented client and event factory
	repo := mongoRepo.NewInboundShipmentRepository(instrumentedMongo.Database(), eventFactory)

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
	receivingService := application.NewReceivingService(repo, logger)

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
	api := router.Group("/api/v1/shipments")
	{
		api.GET("", listShipmentsHandler(receivingService, logger))
		api.POST("", createShipmentHandler(receivingService, logger))
		api.GET("/status/:status", getShipmentsByStatusHandler(receivingService, logger))
		api.GET("/expected", getExpectedArrivalsHandler(receivingService, logger))
		api.GET("/:shipmentId", getShipmentHandler(receivingService, logger))
		api.POST("/:shipmentId/arrive", markArrivedHandler(receivingService, logger))
		api.POST("/:shipmentId/start", startReceivingHandler(receivingService, logger))
		api.POST("/:shipmentId/receive", receiveItemHandler(receivingService, logger))
		api.POST("/:shipmentId/complete", completeReceivingHandler(receivingService, logger))
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
		ServerAddr: getEnv("SERVER_ADDR", ":8010"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "receiving_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Kafka: &kafka.Config{
			Brokers:       []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			ConsumerGroup: "receiving-service",
			ClientID:      "receiving-service",
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

func createShipmentHandler(service *application.ReceivingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req dto.CreateShipmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.id": req.ShipmentID,
			"asn.id":      req.ASN.ASNID,
		})

		cmd := application.CreateShipmentCommand{
			ShipmentID:      req.ShipmentID,
			PurchaseOrderID: req.PurchaseOrderID,
			ASN: domain.AdvanceShippingNotice{
				ASNID:           req.ASN.ASNID,
				CarrierName:     req.ASN.CarrierName,
				TrackingNumber:  req.ASN.TrackingNumber,
				ExpectedArrival: req.ASN.ExpectedArrival,
				ContainerCount:  req.ASN.ContainerCount,
				TotalWeight:     req.ASN.TotalWeight,
				SpecialHandling: req.ASN.SpecialHandling,
			},
			Supplier: domain.Supplier{
				SupplierID:   req.Supplier.SupplierID,
				SupplierName: req.Supplier.SupplierName,
				ContactName:  req.Supplier.ContactName,
				ContactPhone: req.Supplier.ContactPhone,
				ContactEmail: req.Supplier.ContactEmail,
			},
		}

		for _, item := range req.ExpectedItems {
			cmd.ExpectedItems = append(cmd.ExpectedItems, domain.ExpectedItem{
				SKU:               item.SKU,
				ProductName:       item.ProductName,
				ExpectedQuantity:  item.ExpectedQuantity,
				UnitCost:          item.UnitCost,
				Weight:            item.Weight,
				IsHazmat:          item.IsHazmat,
				RequiresColdChain: item.RequiresColdChain,
			})
		}

		shipment, err := service.CreateShipment(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusCreated, toShipmentResponse(shipment))
	}
}

func getShipmentHandler(service *application.ReceivingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		shipmentID := c.Param("shipmentId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.id": shipmentID,
		})

		shipment, err := service.GetShipment(c.Request.Context(), shipmentID)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, toShipmentResponse(shipment))
	}
}

func listShipmentsHandler(service *application.ReceivingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		limitStr := c.Query("limit")
		limit := 100
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
				if limit > 500 {
					limit = 500
				}
			}
		}

		shipments, err := service.ListShipments(c.Request.Context(), limit)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		response := dto.ShipmentListResponse{
			Shipments: make([]dto.ShipmentSummary, len(shipments)),
			Total:     len(shipments),
		}

		for i, s := range shipments {
			response.Shipments[i] = dto.ShipmentSummary{
				ID:              s.ID.Hex(),
				ShipmentID:      s.ShipmentID,
				ASNID:           s.ASN.ASNID,
				SupplierName:    s.Supplier.SupplierName,
				Status:          string(s.Status),
				ExpectedArrival: s.ASN.ExpectedArrival,
				TotalExpected:   s.TotalExpectedQuantity(),
				TotalReceived:   s.TotalReceivedQuantity(),
				CreatedAt:       s.CreatedAt,
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

func getShipmentsByStatusHandler(service *application.ReceivingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		status := c.Param("status")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"status": status,
		})

		shipments, err := service.GetShipmentsByStatus(c.Request.Context(), domain.ShipmentStatus(status))
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		response := dto.ShipmentListResponse{
			Shipments: make([]dto.ShipmentSummary, len(shipments)),
			Total:     len(shipments),
		}

		for i, s := range shipments {
			response.Shipments[i] = dto.ShipmentSummary{
				ID:              s.ID.Hex(),
				ShipmentID:      s.ShipmentID,
				ASNID:           s.ASN.ASNID,
				SupplierName:    s.Supplier.SupplierName,
				Status:          string(s.Status),
				ExpectedArrival: s.ASN.ExpectedArrival,
				TotalExpected:   s.TotalExpectedQuantity(),
				TotalReceived:   s.TotalReceivedQuantity(),
				CreatedAt:       s.CreatedAt,
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

func getExpectedArrivalsHandler(service *application.ReceivingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		// Default to next 24 hours
		from := time.Now()
		to := from.Add(24 * time.Hour)

		if fromStr := c.Query("from"); fromStr != "" {
			if parsed, err := time.Parse(time.RFC3339, fromStr); err == nil {
				from = parsed
			}
		}
		if toStr := c.Query("to"); toStr != "" {
			if parsed, err := time.Parse(time.RFC3339, toStr); err == nil {
				to = parsed
			}
		}

		shipments, err := service.GetExpectedArrivals(c.Request.Context(), from, to)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		response := dto.ShipmentListResponse{
			Shipments: make([]dto.ShipmentSummary, len(shipments)),
			Total:     len(shipments),
		}

		for i, s := range shipments {
			response.Shipments[i] = dto.ShipmentSummary{
				ID:              s.ID.Hex(),
				ShipmentID:      s.ShipmentID,
				ASNID:           s.ASN.ASNID,
				SupplierName:    s.Supplier.SupplierName,
				Status:          string(s.Status),
				ExpectedArrival: s.ASN.ExpectedArrival,
				TotalExpected:   s.TotalExpectedQuantity(),
				TotalReceived:   s.TotalReceivedQuantity(),
				CreatedAt:       s.CreatedAt,
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

func markArrivedHandler(service *application.ReceivingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		shipmentID := c.Param("shipmentId")

		var req dto.MarkArrivedRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.id": shipmentID,
			"dock.id":     req.DockID,
		})

		cmd := application.MarkArrivedCommand{
			ShipmentID: shipmentID,
			DockID:     req.DockID,
		}

		shipment, err := service.MarkShipmentArrived(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, toShipmentResponse(shipment))
	}
}

func startReceivingHandler(service *application.ReceivingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		shipmentID := c.Param("shipmentId")

		var req dto.StartReceivingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.id": shipmentID,
			"worker.id":   req.WorkerID,
		})

		cmd := application.StartReceivingCommand{
			ShipmentID: shipmentID,
			WorkerID:   req.WorkerID,
		}

		shipment, err := service.StartReceiving(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, toShipmentResponse(shipment))
	}
}

func receiveItemHandler(service *application.ReceivingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		shipmentID := c.Param("shipmentId")

		var req dto.ReceiveItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.id": shipmentID,
			"sku":         req.SKU,
			"quantity":    req.Quantity,
		})

		cmd := application.ReceiveItemCommand{
			ShipmentID: shipmentID,
			SKU:        req.SKU,
			Quantity:   req.Quantity,
			Condition:  req.Condition,
			ToteID:     req.ToteID,
			WorkerID:   req.WorkerID,
			Notes:      req.Notes,
		}

		shipment, err := service.ReceiveItem(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, toShipmentResponse(shipment))
	}
}

func completeReceivingHandler(service *application.ReceivingService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		shipmentID := c.Param("shipmentId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shipment.id": shipmentID,
		})

		cmd := application.CompleteReceivingCommand{
			ShipmentID: shipmentID,
		}

		shipment, err := service.CompleteReceiving(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, toShipmentResponse(shipment))
	}
}

// Helper function to convert domain to response
func toShipmentResponse(s *domain.InboundShipment) dto.ShipmentResponse {
	resp := dto.ShipmentResponse{
		ID:               s.ID.Hex(),
		ShipmentID:       s.ShipmentID,
		PurchaseOrderID:  s.PurchaseOrderID,
		Status:           string(s.Status),
		ReceivingDockID:  s.ReceivingDockID,
		AssignedWorkerID: s.AssignedWorkerID,
		ArrivedAt:        s.ArrivedAt,
		CompletedAt:      s.CompletedAt,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
		TotalExpected:    s.TotalExpectedQuantity(),
		TotalReceived:    s.TotalReceivedQuantity(),
		TotalDamaged:     s.TotalDamagedQuantity(),
		IsFullyReceived:  s.IsFullyReceived(),
		ASN: dto.ASNResponse{
			ASNID:           s.ASN.ASNID,
			CarrierName:     s.ASN.CarrierName,
			TrackingNumber:  s.ASN.TrackingNumber,
			ExpectedArrival: s.ASN.ExpectedArrival,
			ContainerCount:  s.ASN.ContainerCount,
			TotalWeight:     s.ASN.TotalWeight,
			SpecialHandling: s.ASN.SpecialHandling,
		},
		Supplier: dto.SupplierResponse{
			SupplierID:   s.Supplier.SupplierID,
			SupplierName: s.Supplier.SupplierName,
			ContactName:  s.Supplier.ContactName,
			ContactPhone: s.Supplier.ContactPhone,
			ContactEmail: s.Supplier.ContactEmail,
		},
	}

	resp.ExpectedItems = make([]dto.ExpectedItemResponse, len(s.ExpectedItems))
	for i, item := range s.ExpectedItems {
		resp.ExpectedItems[i] = dto.ExpectedItemResponse{
			SKU:               item.SKU,
			ProductName:       item.ProductName,
			ExpectedQuantity:  item.ExpectedQuantity,
			ReceivedQuantity:  item.ReceivedQuantity,
			DamagedQuantity:   item.DamagedQuantity,
			RemainingQuantity: item.RemainingQuantity(),
			UnitCost:          item.UnitCost,
			Weight:            item.Weight,
			IsHazmat:          item.IsHazmat,
			RequiresColdChain: item.RequiresColdChain,
			IsFullyReceived:   item.IsFullyReceived(),
		}
	}

	resp.ReceiptRecords = make([]dto.ReceiptRecordResponse, len(s.ReceiptRecords))
	for i, record := range s.ReceiptRecords {
		resp.ReceiptRecords[i] = dto.ReceiptRecordResponse{
			ReceiptID:  record.ReceiptID,
			SKU:        record.SKU,
			Quantity:   record.Quantity,
			ToteID:     record.ToteID,
			Condition:  record.Condition,
			ReceivedBy: record.ReceivedBy,
			ReceivedAt: record.ReceivedAt,
			Notes:      record.Notes,
		}
	}

	resp.Discrepancies = make([]dto.DiscrepancyResponse, len(s.Discrepancies))
	for i, disc := range s.Discrepancies {
		resp.Discrepancies[i] = dto.DiscrepancyResponse{
			SKU:              disc.SKU,
			ExpectedQuantity: disc.ExpectedQuantity,
			ReceivedQuantity: disc.ReceivedQuantity,
			DamagedQuantity:  disc.DamagedQuantity,
			DiscrepancyType:  disc.DiscrepancyType,
			RecordedAt:       disc.RecordedAt,
			Notes:            disc.Notes,
		}
	}

	return resp
}
