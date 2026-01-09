package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wms-platform/shared/pkg/errors"
)

// Context keys
const (
	ContextKeyRequestID     = "requestId"
	ContextKeyCorrelationID = "correlationId"
	ContextKeyTraceID       = "traceId"
	ContextKeySpanID        = "spanId"
)

// HTTP header names
const (
	HeaderRequestID     = "X-Request-ID"
	HeaderCorrelationID = "X-Correlation-ID"
	HeaderTraceID       = "X-Trace-ID"
	HeaderSpanID        = "X-Span-ID"
)

// RequestID middleware generates or propagates request IDs
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID exists in header
		requestID := c.GetHeader(HeaderRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set in context and response header
		c.Set(ContextKeyRequestID, requestID)
		c.Header(HeaderRequestID, requestID)

		c.Next()
	}
}

// CorrelationID middleware handles correlation ID propagation for distributed tracing
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get or generate correlation ID
		correlationID := c.GetHeader(HeaderCorrelationID)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Get or generate trace ID
		traceID := c.GetHeader(HeaderTraceID)
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Generate span ID for this request
		spanID := uuid.New().String()[:16]

		// Set in context
		c.Set(ContextKeyCorrelationID, correlationID)
		c.Set(ContextKeyTraceID, traceID)
		c.Set(ContextKeySpanID, spanID)

		// Set in response headers
		c.Header(HeaderCorrelationID, correlationID)
		c.Header(HeaderTraceID, traceID)
		c.Header(HeaderSpanID, spanID)

		c.Next()
	}
}

// LoggerConfig holds logger middleware configuration
type LoggerConfig struct {
	Logger       *slog.Logger
	ExcludePaths []string // Paths to exclude from logging (e.g., /health, /ready, /metrics)
}

// DefaultLoggerConfig returns default logger configuration with common health/metrics paths excluded
func DefaultLoggerConfig(logger *slog.Logger) *LoggerConfig {
	return &LoggerConfig{
		Logger:       logger,
		ExcludePaths: []string{"/health", "/ready", "/metrics"},
	}
}

// Logger middleware adds structured logging with correlation context
// Deprecated: Use LoggerWithConfig for path exclusion support
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return LoggerWithConfig(DefaultLoggerConfig(logger))
}

// LoggerWithConfig middleware adds structured logging with correlation context and path exclusion
func LoggerWithConfig(config *LoggerConfig) gin.HandlerFunc {
	// Build skip map for O(1) lookup
	skipMap := make(map[string]bool)
	for _, path := range config.ExcludePaths {
		skipMap[path] = true
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip logging for excluded paths
		if skipMap[path] {
			c.Next()
			return
		}

		start := time.Now()
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// Get correlation info from context
		requestID, _ := c.Get(ContextKeyRequestID)
		correlationID, _ := c.Get(ContextKeyCorrelationID)
		traceID, _ := c.Get(ContextKeyTraceID)

		attrs := []any{
			"status", status,
			"method", c.Request.Method,
			"path", path,
			"latency", latency.String(),
			"latencyMs", latency.Milliseconds(),
			"clientIP", c.ClientIP(),
			"userAgent", c.Request.UserAgent(),
		}

		if requestID != nil {
			attrs = append(attrs, "requestId", requestID)
		}
		if correlationID != nil {
			attrs = append(attrs, "correlationId", correlationID)
		}
		if traceID != nil {
			attrs = append(attrs, "traceId", traceID)
		}
		if query != "" {
			attrs = append(attrs, "query", query)
		}

		// Log level based on status code
		switch {
		case status >= 500:
			config.Logger.Error("HTTP request", attrs...)
		case status >= 400:
			config.Logger.Warn("HTTP request", attrs...)
		default:
			config.Logger.Info("HTTP request", attrs...)
		}
	}
}

// Recovery middleware handles panics and logs them properly
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get(ContextKeyRequestID)
				correlationID, _ := c.Get(ContextKeyCorrelationID)

				logger.Error("Panic recovered",
					"error", err,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"requestId", requestID,
					"correlationId", correlationID,
				)

				AbortWithAppError(c, &errors.AppError{
					Code:       "INTERNAL_ERROR",
					Message:    "An unexpected error occurred",
					HTTPStatus: 500,
				})
			}
		}()
		c.Next()
	}
}

// ContextLogger creates a logger with context information
func ContextLogger(ctx context.Context, logger *slog.Logger) *slog.Logger {
	attrs := []any{}

	if ginCtx, ok := ctx.(*gin.Context); ok {
		if requestID, exists := ginCtx.Get(ContextKeyRequestID); exists {
			attrs = append(attrs, "requestId", requestID)
		}
		if correlationID, exists := ginCtx.Get(ContextKeyCorrelationID); exists {
			attrs = append(attrs, "correlationId", correlationID)
		}
		if traceID, exists := ginCtx.Get(ContextKeyTraceID); exists {
			attrs = append(attrs, "traceId", traceID)
		}
	}

	return logger.With(attrs...)
}

// GetRequestID extracts request ID from context
func GetRequestID(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyRequestID); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// GetCorrelationID extracts correlation ID from context
func GetCorrelationID(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyCorrelationID); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// GetTraceID extracts trace ID from context
func GetTraceID(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyTraceID); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// PropagationHeaders returns headers that should be propagated to downstream services
func PropagationHeaders(c *gin.Context) map[string]string {
	headers := make(map[string]string)

	if id := GetRequestID(c); id != "" {
		headers[HeaderRequestID] = id
	}
	if id := GetCorrelationID(c); id != "" {
		headers[HeaderCorrelationID] = id
	}
	if id := GetTraceID(c); id != "" {
		headers[HeaderTraceID] = id
	}

	return headers
}
