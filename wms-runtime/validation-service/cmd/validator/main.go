package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wms-platform/wms-runtime/validation-service/internal/api"
	"github.com/wms-platform/wms-runtime/validation-service/internal/eventcapture"
	"github.com/wms-platform/wms-runtime/validation-service/internal/validation"
)

func main() {
	log.Println("Starting WMS Validation Service...")

	// Configuration
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	serverPort := getEnv("SERVER_PORT", "8080")
	eventTTLMinutes := getEnvInt("EVENT_TTL_MINUTES", 30)

	// Initialize event store
	eventStore := eventcapture.NewEventStore(time.Duration(eventTTLMinutes) * time.Minute)
	log.Printf("Initialized event store with TTL: %d minutes", eventTTLMinutes)

	// Initialize event validator
	eventValidator := validation.NewEventValidator()
	log.Println("Initialized event validator")

	// Initialize sequence validator
	sequenceValidator := validation.NewSequenceValidator()
	log.Println("Initialized sequence validator")

	// Initialize correlation tracker
	correlationTracker := validation.NewCorrelationTracker(eventStore)
	log.Println("Initialized correlation tracker")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize and start Kafka subscriber
	kafkaSubscriber := eventcapture.NewKafkaSubscriber(
		kafkaBrokers,
		eventStore,
		eventValidator,
	)

	// Start Kafka consumer in background
	go func() {
		if err := kafkaSubscriber.Start(ctx); err != nil {
			log.Fatalf("Failed to start Kafka subscriber: %v", err)
		}
	}()

	log.Printf("Started Kafka subscriber (brokers: %s)", kafkaBrokers)

	// Initialize HTTP server
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "validation-service",
		})
	})

	// Initialize API handlers
	apiServer := api.NewServer(
		eventStore,
		eventValidator,
		sequenceValidator,
		correlationTracker,
	)

	// Register routes
	v1 := router.Group("/api/v1")
	{
		validation := v1.Group("/validation")
		{
			// Start tracking an order
			validation.POST("/start-tracking/:orderId", apiServer.StartTracking)

			// Get captured events for an order
			validation.GET("/events/:orderId", apiServer.GetEvents)

			// Assert expected events were received
			validation.POST("/assert/:orderId", apiServer.AssertEvents)

			// Get validation status
			validation.GET("/status/:orderId", apiServer.GetStatus)

			// Clear tracking data
			validation.DELETE("/clear/:orderId", apiServer.ClearTracking)

			// Validate event sequence
			validation.POST("/sequence/:orderId", apiServer.ValidateSequence)

			// Get validation report
			validation.GET("/report/:orderId", apiServer.GetReport)
		}

		// Statistics endpoints
		stats := v1.Group("/stats")
		{
			stats.GET("/summary", apiServer.GetStatsSummary)
			stats.GET("/events-by-type", apiServer.GetEventsByType)
		}
	}

	// Start HTTP server
	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: router,
	}

	go func() {
		log.Printf("Starting HTTP server on port %s", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")

	// Cancel context to stop Kafka consumer
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Close Kafka subscriber
	if err := kafkaSubscriber.Close(); err != nil {
		log.Printf("Kafka subscriber close error: %v", err)
	}

	log.Println("Validation service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}
