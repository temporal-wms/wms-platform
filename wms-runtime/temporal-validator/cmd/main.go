package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wms-platform/wms-runtime/temporal-validator/internal/api"
	"github.com/wms-platform/wms-runtime/temporal-validator/internal/workflow"
	"go.temporal.io/sdk/client"
)

func main() {
	log.Println("Starting Temporal Validator Service...")

	// Configuration
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")
	serverPort := getEnv("SERVER_PORT", "9090")
	namespace := getEnv("TEMPORAL_NAMESPACE", "default")

	// Create Temporal client
	temporalClient, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: namespace,
	})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer temporalClient.Close()

	log.Printf("Connected to Temporal at %s (namespace: %s)", temporalHost, namespace)

	// Initialize workflow state monitor
	stateMonitor := workflow.NewStateMonitor(temporalClient)
	log.Println("Initialized workflow state monitor")

	// Initialize signal tracker
	signalTracker := workflow.NewSignalTracker(temporalClient)
	log.Println("Initialized signal tracker")

	// Initialize HTTP server
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "temporal-validator",
		})
	})

	// Initialize API handlers
	apiServer := api.NewServer(stateMonitor, signalTracker)

	// Register routes
	v1 := router.Group("/api/v1")
	{
		workflow := v1.Group("/workflow")
		{
			// Get workflow execution details
			workflow.GET("/describe/:workflowId", apiServer.DescribeWorkflow)

			// Get workflow history (including signals)
			workflow.GET("/history/:workflowId", apiServer.GetWorkflowHistory)

			// Assert signal was received
			workflow.POST("/assert-signal/:workflowId", apiServer.AssertSignal)

			// Get workflow status
			workflow.GET("/status/:workflowId", apiServer.GetWorkflowStatus)

			// Query workflow state
			workflow.POST("/query/:workflowId", apiServer.QueryWorkflow)
		}

		// Signal-specific endpoints
		signal := v1.Group("/signal")
		{
			// Get all signals for a workflow
			signal.GET("/list/:workflowId", apiServer.ListSignals)

			// Validate signal delivery
			signal.POST("/validate/:workflowId", apiServer.ValidateSignalDelivery)
		}

		// Statistics
		stats := v1.Group("/stats")
		{
			stats.GET("/summary", apiServer.GetStatsSummary)
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

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Temporal validator service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
