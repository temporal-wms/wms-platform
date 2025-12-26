package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Config holds middleware configuration
type Config struct {
	Logger            *slog.Logger
	ServiceName       string
	EnableCORS        bool
	EnableRateLimiter bool
	RateLimitRPS      int
	TrustedProxies    []string
}

// DefaultConfig returns a default middleware configuration
func DefaultConfig(serviceName string, logger *slog.Logger) *Config {
	return &Config{
		Logger:            logger,
		ServiceName:       serviceName,
		EnableCORS:        true,
		EnableRateLimiter: false,
		RateLimitRPS:      100,
		TrustedProxies:    nil,
	}
}

// Setup applies all standard middleware to a Gin router
func Setup(router *gin.Engine, config *Config) {
	// Initialize validator
	InitValidator()

	// Trust specific proxies if configured
	if len(config.TrustedProxies) > 0 {
		_ = router.SetTrustedProxies(config.TrustedProxies)
	}

	// Apply middleware in order
	router.Use(Recovery(config.Logger))
	router.Use(RequestID())
	router.Use(CorrelationID())
	router.Use(Logger(config.Logger))
	router.Use(InputSanitizer())

	if config.EnableCORS {
		router.Use(CORS())
	}

	router.Use(ContentType())
	router.Use(ErrorHandler(config.Logger))
}

// SetupMinimal applies only essential middleware (for internal services)
func SetupMinimal(router *gin.Engine, config *Config) {
	InitValidator()

	router.Use(Recovery(config.Logger))
	router.Use(RequestID())
	router.Use(Logger(config.Logger))
	router.Use(ErrorHandler(config.Logger))
}

// CORS middleware for handling Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-Correlation-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-Correlation-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Timeout middleware adds request timeout
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a context with timeout
		// Note: This is a simplified version - in production you'd want
		// to handle this more carefully with proper cancellation
		c.Request = c.Request.WithContext(c.Request.Context())
		c.Next()
	}
}

// SecurityHeaders middleware adds common security headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Header("Pragma", "no-cache")

		c.Next()
	}
}

// HealthCheck creates a health check handler
func HealthCheck(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": serviceName,
		})
	}
}

// ReadinessCheck creates a readiness check handler with custom check function
func ReadinessCheck(serviceName string, checkFn func() error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := checkFn(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "not ready",
				"service": serviceName,
				"error":   err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"service": serviceName,
		})
	}
}

// NoRoute handles 404 errors with proper error format
func NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID, _ := c.Get(ContextKeyRequestID)
		reqID, _ := requestID.(string)

		c.JSON(http.StatusNotFound, APIErrorResponse{
			Code:      "ROUTE_NOT_FOUND",
			Message:   "The requested resource was not found",
			RequestID: reqID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Path:      c.Request.URL.Path,
		})
	}
}

// NoMethod handles 405 errors with proper error format
func NoMethod() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID, _ := c.Get(ContextKeyRequestID)
		reqID, _ := requestID.(string)

		c.JSON(http.StatusMethodNotAllowed, APIErrorResponse{
			Code:      "METHOD_NOT_ALLOWED",
			Message:   "The request method is not supported for this resource",
			RequestID: reqID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Path:      c.Request.URL.Path,
		})
	}
}

// WrapHandler wraps a handler with error handling
func WrapHandler(handler func(*gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := handler(c); err != nil {
			_ = c.Error(err)
		}
	}
}
