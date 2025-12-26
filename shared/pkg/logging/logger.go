package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"
)

// LogLevel represents logging levels
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Config holds logger configuration
type Config struct {
	Level       LogLevel
	ServiceName string
	Environment string
	Version     string
	Output      io.Writer
	AddSource   bool
}

// DefaultConfig returns a default logger configuration
func DefaultConfig(serviceName string) *Config {
	return &Config{
		Level:       LevelInfo,
		ServiceName: serviceName,
		Environment: getEnv("ENVIRONMENT", "development"),
		Version:     getEnv("VERSION", "unknown"),
		Output:      os.Stdout,
		AddSource:   false,
	}
}

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
	serviceName string
	environment string
	version     string
}

// New creates a new Logger instance
func New(config *Config) *Logger {
	level := slog.LevelInfo
	switch config.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	}

	output := config.Output
	if output == nil {
		output = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: config.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize time format
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.UTC().Format(time.RFC3339Nano))
				}
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(output, opts)

	// Add base attributes
	baseLogger := slog.New(handler).With(
		"service", config.ServiceName,
		"environment", config.Environment,
		"version", config.Version,
	)

	return &Logger{
		Logger:      baseLogger,
		serviceName: config.ServiceName,
		environment: config.Environment,
		version:     config.Version,
	}
}

// WithContext creates a logger with context attributes
func (l *Logger) WithContext(ctx context.Context) *Logger {
	attrs := extractContextAttrs(ctx)
	if len(attrs) == 0 {
		return l
	}

	return &Logger{
		Logger:      l.Logger.With(attrs...),
		serviceName: l.serviceName,
		environment: l.environment,
		version:     l.version,
	}
}

// WithRequestID adds a request ID to the logger
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger:      l.Logger.With("requestId", requestID),
		serviceName: l.serviceName,
		environment: l.environment,
		version:     l.version,
	}
}

// WithCorrelationID adds a correlation ID to the logger
func (l *Logger) WithCorrelationID(correlationID string) *Logger {
	return &Logger{
		Logger:      l.Logger.With("correlationId", correlationID),
		serviceName: l.serviceName,
		environment: l.environment,
		version:     l.version,
	}
}

// WithTraceID adds a trace ID to the logger
func (l *Logger) WithTraceID(traceID string) *Logger {
	return &Logger{
		Logger:      l.Logger.With("traceId", traceID),
		serviceName: l.serviceName,
		environment: l.environment,
		version:     l.version,
	}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]any) *Logger {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}

	return &Logger{
		Logger:      l.Logger.With(attrs...),
		serviceName: l.serviceName,
		environment: l.environment,
		version:     l.version,
	}
}

// WithError adds an error to the logger
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return &Logger{
		Logger:      l.Logger.With("error", err.Error()),
		serviceName: l.serviceName,
		environment: l.environment,
		version:     l.version,
	}
}

// WithComponent adds a component name to the logger
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger:      l.Logger.With("component", component),
		serviceName: l.serviceName,
		environment: l.environment,
		version:     l.version,
	}
}

// WithOperation adds an operation name to the logger
func (l *Logger) WithOperation(operation string) *Logger {
	return &Logger{
		Logger:      l.Logger.With("operation", operation),
		serviceName: l.serviceName,
		environment: l.environment,
		version:     l.version,
	}
}

// Event logs a business event with structured data
func (l *Logger) Event(ctx context.Context, eventType string, data map[string]any) {
	attrs := []any{
		"eventType", eventType,
		"timestamp", time.Now().UTC().Format(time.RFC3339Nano),
	}

	for k, v := range data {
		attrs = append(attrs, k, v)
	}

	l.WithContext(ctx).Info("Business event", attrs...)
}

// Audit logs an audit event
func (l *Logger) Audit(ctx context.Context, action string, resource string, resourceID string, userID string, details map[string]any) {
	attrs := []any{
		"auditAction", action,
		"resource", resource,
		"resourceId", resourceID,
		"userId", userID,
		"timestamp", time.Now().UTC().Format(time.RFC3339Nano),
	}

	for k, v := range details {
		attrs = append(attrs, k, v)
	}

	l.WithContext(ctx).Info("Audit event", attrs...)
}

// Performance logs performance metrics
func (l *Logger) Performance(ctx context.Context, operation string, duration time.Duration, success bool, details map[string]any) {
	attrs := []any{
		"operation", operation,
		"durationMs", duration.Milliseconds(),
		"durationNs", duration.Nanoseconds(),
		"success", success,
	}

	for k, v := range details {
		attrs = append(attrs, k, v)
	}

	l.WithContext(ctx).Info("Performance metric", attrs...)
}

// HTTPRequest logs an HTTP request with standard fields
func (l *Logger) HTTPRequest(ctx context.Context, method, path string, status int, duration time.Duration, clientIP, userAgent string) {
	level := slog.LevelInfo
	if status >= 500 {
		level = slog.LevelError
	} else if status >= 400 {
		level = slog.LevelWarn
	}

	l.WithContext(ctx).Log(ctx, level, "HTTP request",
		"method", method,
		"path", path,
		"status", status,
		"durationMs", duration.Milliseconds(),
		"clientIP", clientIP,
		"userAgent", userAgent,
	)
}

// DatabaseQuery logs a database query
func (l *Logger) DatabaseQuery(ctx context.Context, collection, operation string, duration time.Duration, success bool, rowsAffected int64) {
	level := slog.LevelDebug
	if !success {
		level = slog.LevelError
	}

	l.WithContext(ctx).Log(ctx, level, "Database query",
		"collection", collection,
		"operation", operation,
		"durationMs", duration.Milliseconds(),
		"success", success,
		"rowsAffected", rowsAffected,
	)
}

// KafkaPublish logs a Kafka publish event
func (l *Logger) KafkaPublish(ctx context.Context, topic, eventType string, success bool, duration time.Duration) {
	level := slog.LevelDebug
	if !success {
		level = slog.LevelError
	}

	l.WithContext(ctx).Log(ctx, level, "Kafka publish",
		"topic", topic,
		"eventType", eventType,
		"success", success,
		"durationMs", duration.Milliseconds(),
	)
}

// KafkaConsume logs a Kafka consume event
func (l *Logger) KafkaConsume(ctx context.Context, topic, eventType string, partition int, offset int64) {
	l.WithContext(ctx).Debug("Kafka consume",
		"topic", topic,
		"eventType", eventType,
		"partition", partition,
		"offset", offset,
	)
}

// WorkflowStart logs a Temporal workflow start
func (l *Logger) WorkflowStart(ctx context.Context, workflowType, workflowID string, input any) {
	l.WithContext(ctx).Info("Workflow started",
		"workflowType", workflowType,
		"workflowId", workflowID,
	)
}

// WorkflowComplete logs a Temporal workflow completion
func (l *Logger) WorkflowComplete(ctx context.Context, workflowType, workflowID string, duration time.Duration, success bool) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelError
	}

	l.WithContext(ctx).Log(ctx, level, "Workflow completed",
		"workflowType", workflowType,
		"workflowId", workflowID,
		"durationMs", duration.Milliseconds(),
		"success", success,
	)
}

// ActivityStart logs a Temporal activity start
func (l *Logger) ActivityStart(ctx context.Context, activityType string, input any) {
	l.WithContext(ctx).Debug("Activity started",
		"activityType", activityType,
	)
}

// ActivityComplete logs a Temporal activity completion
func (l *Logger) ActivityComplete(ctx context.Context, activityType string, duration time.Duration, success bool) {
	level := slog.LevelDebug
	if !success {
		level = slog.LevelError
	}

	l.WithContext(ctx).Log(ctx, level, "Activity completed",
		"activityType", activityType,
		"durationMs", duration.Milliseconds(),
		"success", success,
	)
}

// Panic logs a panic with stack trace
func (l *Logger) Panic(ctx context.Context, recovered any) {
	stack := make([]byte, 4096)
	n := runtime.Stack(stack, false)

	l.WithContext(ctx).Error("Panic recovered",
		"panic", recovered,
		"stack", string(stack[:n]),
	)
}

// SetDefault sets this logger as the default slog logger
func (l *Logger) SetDefault() {
	slog.SetDefault(l.Logger)
}

// Context keys for extracting attributes
type contextKey string

const (
	RequestIDKey     contextKey = "requestId"
	CorrelationIDKey contextKey = "correlationId"
	TraceIDKey       contextKey = "traceId"
	SpanIDKey        contextKey = "spanId"
	UserIDKey        contextKey = "userId"
)

// extractContextAttrs extracts logging attributes from context
func extractContextAttrs(ctx context.Context) []any {
	var attrs []any

	if v := ctx.Value(RequestIDKey); v != nil {
		attrs = append(attrs, "requestId", v)
	}
	if v := ctx.Value(CorrelationIDKey); v != nil {
		attrs = append(attrs, "correlationId", v)
	}
	if v := ctx.Value(TraceIDKey); v != nil {
		attrs = append(attrs, "traceId", v)
	}
	if v := ctx.Value(SpanIDKey); v != nil {
		attrs = append(attrs, "spanId", v)
	}
	if v := ctx.Value(UserIDKey); v != nil {
		attrs = append(attrs, "userId", v)
	}

	return attrs
}

// ContextWithRequestID adds request ID to context
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// ContextWithCorrelationID adds correlation ID to context
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// ContextWithTraceID adds trace ID to context
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// ContextWithUserID adds user ID to context
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
