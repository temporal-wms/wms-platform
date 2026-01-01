package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig holds tracing middleware configuration
type TracingConfig struct {
	ServiceName  string
	SkipPaths    []string
	Propagators  propagation.TextMapPropagator
	TracerName   string
}

// DefaultTracingConfig returns default tracing configuration
func DefaultTracingConfig(serviceName string) *TracingConfig {
	return &TracingConfig{
		ServiceName: serviceName,
		SkipPaths:   []string{"/health", "/ready", "/metrics"},
		Propagators: otel.GetTextMapPropagator(),
		TracerName:  serviceName,
	}
}

// TracingMiddleware creates middleware that adds distributed tracing
func TracingMiddleware(config *TracingConfig) gin.HandlerFunc {
	tracer := otel.Tracer(config.TracerName)
	skipMap := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipMap[path] = true
	}

	return func(c *gin.Context) {
		// Skip tracing for excluded paths
		if skipMap[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Extract trace context from incoming request
		ctx := config.Propagators.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Determine span name
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		spanName := fmt.Sprintf("%s %s", c.Request.Method, path)

		// Start span
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethodKey.String(c.Request.Method),
				semconv.HTTPRouteKey.String(path),
				semconv.HTTPURLKey.String(c.Request.URL.String()),
				semconv.HTTPSchemeKey.String(c.Request.URL.Scheme),
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.client_ip", c.ClientIP()),
				attribute.String("service.name", config.ServiceName),
			),
		)
		defer span.End()

		// Add request ID if available
		if requestID, exists := c.Get(ContextKeyRequestID); exists {
			span.SetAttributes(attribute.String("request.id", requestID.(string)))
		}

		// Add correlation ID if available
		if correlationID, exists := c.Get(ContextKeyCorrelationID); exists {
			span.SetAttributes(attribute.String("correlation.id", correlationID.(string)))
		}

		// Add WMS CloudEvents extension attributes
		if wmsCorrelationID, exists := c.Get(ContextKeyWMSCorrelationID); exists {
			if id, ok := wmsCorrelationID.(string); ok && id != "" {
				span.SetAttributes(attribute.String("wms.correlation_id", id))
			}
		}
		if wmsWaveNumber, exists := c.Get(ContextKeyWMSWaveNumber); exists {
			if id, ok := wmsWaveNumber.(string); ok && id != "" {
				span.SetAttributes(attribute.String("wms.wave_number", id))
			}
		}
		if wmsWorkflowID, exists := c.Get(ContextKeyWMSWorkflowID); exists {
			if id, ok := wmsWorkflowID.(string); ok && id != "" {
				span.SetAttributes(attribute.String("wms.workflow_id", id))
			}
		}

		// Set trace ID in context for logging
		c.Set("traceId", span.SpanContext().TraceID().String())
		c.Set("spanId", span.SpanContext().SpanID().String())

		// Update request context with span
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Record response attributes
		status := c.Writer.Status()
		span.SetAttributes(
			semconv.HTTPStatusCodeKey.Int(status),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// Set span status based on HTTP status
		if status >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", status))
		} else if status >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", status))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Record errors if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
		}
	}
}

// SimpleTracingMiddleware creates a simpler tracing middleware using default config
func SimpleTracingMiddleware(serviceName string) gin.HandlerFunc {
	return TracingMiddleware(DefaultTracingConfig(serviceName))
}

// SpanFromGinContext extracts the span from a Gin context
func SpanFromGinContext(c *gin.Context) trace.Span {
	return trace.SpanFromContext(c.Request.Context())
}

// AddSpanAttributes adds attributes to the current span from Gin context
func AddSpanAttributes(c *gin.Context, attrs map[string]interface{}) {
	span := SpanFromGinContext(c)
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			span.SetAttributes(attribute.String(k, val))
		case int:
			span.SetAttributes(attribute.Int(k, val))
		case int64:
			span.SetAttributes(attribute.Int64(k, val))
		case float64:
			span.SetAttributes(attribute.Float64(k, val))
		case bool:
			span.SetAttributes(attribute.Bool(k, val))
		default:
			span.SetAttributes(attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
}

// AddSpanEvent adds an event to the current span from Gin context
func AddSpanEvent(c *gin.Context, name string, attrs map[string]interface{}) {
	span := SpanFromGinContext(c)
	var options []trace.EventOption
	if len(attrs) > 0 {
		attributes := make([]attribute.KeyValue, 0, len(attrs))
		for k, v := range attrs {
			switch val := v.(type) {
			case string:
				attributes = append(attributes, attribute.String(k, val))
			case int:
				attributes = append(attributes, attribute.Int(k, val))
			case int64:
				attributes = append(attributes, attribute.Int64(k, val))
			case float64:
				attributes = append(attributes, attribute.Float64(k, val))
			case bool:
				attributes = append(attributes, attribute.Bool(k, val))
			default:
				attributes = append(attributes, attribute.String(k, fmt.Sprintf("%v", val)))
			}
		}
		options = append(options, trace.WithAttributes(attributes...))
	}
	span.AddEvent(name, options...)
}

// SetSpanError records an error on the current span from Gin context
func SetSpanError(c *gin.Context, err error) {
	span := SpanFromGinContext(c)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// GetTraceIDFromGinContext returns the trace ID from the Gin context
func GetTraceIDFromGinContext(c *gin.Context) string {
	if traceID, exists := c.Get("traceId"); exists {
		return traceID.(string)
	}
	span := SpanFromGinContext(c)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanIDFromGinContext returns the span ID from the Gin context
func GetSpanIDFromGinContext(c *gin.Context) string {
	if spanID, exists := c.Get("spanId"); exists {
		return spanID.(string)
	}
	span := SpanFromGinContext(c)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// InjectTracingHeaders injects tracing headers for outgoing requests
func InjectTracingHeaders(c *gin.Context, headers map[string]string) {
	carrier := propagation.MapCarrier(headers)
	otel.GetTextMapPropagator().Inject(c.Request.Context(), carrier)
}

// TracingHeaderCarrier adapts http.Header to propagation.TextMapCarrier
type TracingHeaderCarrier struct {
	headers map[string]string
}

// NewTracingHeaderCarrier creates a new header carrier
func NewTracingHeaderCarrier() *TracingHeaderCarrier {
	return &TracingHeaderCarrier{
		headers: make(map[string]string),
	}
}

// Get returns the value for the given key
func (c *TracingHeaderCarrier) Get(key string) string {
	return c.headers[key]
}

// Set sets the value for the given key
func (c *TracingHeaderCarrier) Set(key, value string) {
	c.headers[key] = value
}

// Keys returns all keys
func (c *TracingHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for k := range c.headers {
		keys = append(keys, k)
	}
	return keys
}

// Headers returns the underlying headers map
func (c *TracingHeaderCarrier) Headers() map[string]string {
	return c.headers
}
