package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wms-platform/walling-service/internal/api/http"
	"github.com/wms-platform/walling-service/internal/application"
	"github.com/wms-platform/walling-service/internal/infrastructure/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Walling Service")

	// Get configuration from environment
	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017")
	dbName := getEnv("MONGODB_DATABASE", "walling_db")
	serverAddr := getEnv("SERVER_ADDR", ":8017")

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetMinPoolSize(10).
		SetMaxPoolSize(100)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Error("Failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			logger.Error("Failed to disconnect from MongoDB", "error", err)
		}
	}()

	// Ping MongoDB
	if err := client.Ping(ctx, nil); err != nil {
		logger.Error("Failed to ping MongoDB", "error", err)
		os.Exit(1)
	}
	logger.Info("Connected to MongoDB", "database", dbName)

	// Get database
	db := client.Database(dbName)

	// Create repositories
	taskRepo := mongodb.NewWallingTaskRepository(db)

	// Create application service
	wallingService := application.NewWallingApplicationService(taskRepo, logger)

	// Create HTTP handlers
	handlers := http.NewHandlers(wallingService)

	// Setup Gin router
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(logger))

	// Setup routes
	http.SetupRoutes(router, handlers)

	// Start server
	logger.Info("Starting HTTP server", "addr", serverAddr)
	if err := router.Run(serverAddr); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// requestLogger returns a Gin middleware for logging requests
func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		logger.Info("HTTP request",
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"status", status,
			"latency", latency.String(),
			"clientIP", c.ClientIP(),
		)
	}
}
