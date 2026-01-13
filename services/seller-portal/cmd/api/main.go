package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/tracing"

	"github.com/wms-platform/services/seller-portal/internal/api/handlers"
	"github.com/wms-platform/services/seller-portal/internal/application"
	"github.com/wms-platform/services/seller-portal/internal/infrastructure/clients"
)

const serviceName = "seller-portal"

func main() {
	// Setup enhanced logger
	logConfig := logging.DefaultConfig(serviceName)
	logConfig.Level = logging.LogLevel(getEnv("LOG_LEVEL", "info"))
	logger := logging.New(logConfig)
	logger.SetDefault()

	logger.Info("Starting seller-portal API")

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

	// Initialize downstream metrics helper
	downstreamMetrics := NewDownstreamMetrics(m)

	// Create instrumented service clients
	sellerClient := clients.NewInstrumentedSellerClient(config.SellerServiceURL, logger, downstreamMetrics)
	orderClient := clients.NewInstrumentedOrderClient(config.OrderServiceURL, logger, downstreamMetrics)
	inventoryClient := clients.NewInstrumentedInventoryClient(config.InventoryServiceURL, logger, downstreamMetrics)
	billingClient := clients.NewInstrumentedBillingClient(config.BillingServiceURL, logger, downstreamMetrics)
	channelClient := clients.NewInstrumentedChannelClient(config.ChannelServiceURL, logger, downstreamMetrics)

	logger.Info("Service clients initialized",
		"seller_service", config.SellerServiceURL,
		"order_service", config.OrderServiceURL,
		"inventory_service", config.InventoryServiceURL,
		"billing_service", config.BillingServiceURL,
		"channel_service", config.ChannelServiceURL,
	)

	// Create dashboard service
	dashboardService := application.NewDashboardService(
		sellerClient,
		orderClient,
		inventoryClient,
		billingClient,
		channelClient,
	)

	// Create handler with observability
	dashboardHandler := handlers.NewDashboardHandler(dashboardService, logger, m)

	// Setup Gin router with middleware
	router := gin.New()

	// Apply standard middleware
	middlewareConfig := middleware.DefaultConfig(serviceName, logger.Logger)
	middleware.Setup(router, middlewareConfig)

	// Add metrics middleware
	router.Use(middleware.MetricsMiddleware(m))

	// Add tracing middleware
	router.Use(middleware.SimpleTracingMiddleware(serviceName))

	// CORS middleware for seller portal frontend
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-WMS-Seller-ID, X-WMS-Tenant-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Handle 404 and 405 errors
	router.NoRoute(middleware.NoRoute())
	router.NoMethod(middleware.NoMethod())

	// Health check endpoints
	router.GET("/health", middleware.HealthCheck(serviceName))
	router.GET("/ready", middleware.ReadinessCheck(serviceName, func() error {
		// Check if downstream services are reachable
		// For now, just return nil - could add ping checks
		return nil
	}))

	// Metrics endpoint
	router.GET("/metrics", middleware.MetricsEndpoint(m))

	// API v1 routes
	api := router.Group("/api/v1")
	dashboardHandler.RegisterRoutes(api)

	// Start server
	srv := &http.Server{
		Addr:         config.ServerAddr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("Server started", "addr", config.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Error("Server error")
		}
	}()

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
	ServerAddr          string
	SellerServiceURL    string
	OrderServiceURL     string
	InventoryServiceURL string
	BillingServiceURL   string
	ChannelServiceURL   string
}

func loadConfig() *Config {
	return &Config{
		ServerAddr:          getEnv("SERVER_ADDR", ":8021"),
		SellerServiceURL:    getEnv("SELLER_SERVICE_URL", "http://localhost:8010"),
		OrderServiceURL:     getEnv("ORDER_SERVICE_URL", "http://localhost:8001"),
		InventoryServiceURL: getEnv("INVENTORY_SERVICE_URL", "http://localhost:8002"),
		BillingServiceURL:   getEnv("BILLING_SERVICE_URL", "http://localhost:8011"),
		ChannelServiceURL:   getEnv("CHANNEL_SERVICE_URL", "http://localhost:8012"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// DownstreamMetrics provides metrics for downstream service calls
type DownstreamMetrics struct {
	metrics *metrics.Metrics
}

// NewDownstreamMetrics creates a new DownstreamMetrics instance
func NewDownstreamMetrics(m *metrics.Metrics) *DownstreamMetrics {
	return &DownstreamMetrics{metrics: m}
}

// RecordRequest records a downstream service request
func (dm *DownstreamMetrics) RecordRequest(service, operation, status string, duration time.Duration) {
	// Use HTTP metrics to record downstream service requests
	statusCode := 200
	if status == "error" {
		statusCode = 500
	}
	dm.metrics.RecordHTTPRequest("GET", "/downstream/"+service+"/"+operation, statusCode, duration)
}

// RecordDashboardAssembly records dashboard assembly time
func (dm *DownstreamMetrics) RecordDashboardAssembly(dashboard string, duration time.Duration) {
	// Use HTTP metrics to record dashboard assembly duration
	dm.metrics.HTTPRequestDuration.WithLabelValues(
		"seller-portal",
		"GET",
		"/dashboard/"+dashboard,
	).Observe(duration.Seconds())
}
