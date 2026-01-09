package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds tracing configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	SampleRate     float64
	Enabled        bool
}

// DefaultConfig returns default tracing configuration
func DefaultConfig(serviceName string) *Config {
	return &Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Environment:    "development",
		OTLPEndpoint:   "localhost:4317",
		SampleRate:     1.0, // Sample all traces in development
		Enabled:        true,
	}
}

// TracerProvider wraps the OpenTelemetry tracer provider
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	config   *Config
}

// Initialize sets up the OpenTelemetry tracing infrastructure
func Initialize(ctx context.Context, config *Config) (*TracerProvider, error) {
	if !config.Enabled {
		return &TracerProvider{
			tracer: otel.Tracer(config.ServiceName),
			config: config,
		}, nil
	}

	// Create OTLP exporter
	conn, err := grpc.NewClient(config.OTLPEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(otlptracegrpc.WithGRPCConn(conn)))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
			attribute.String("service.namespace", "wms"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler
	var sampler sdktrace.Sampler
	if config.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SampleRate)
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider and propagator
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{
		provider: provider,
		tracer:   provider.Tracer(config.ServiceName),
		config:   config,
	}, nil
}

// Shutdown gracefully shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider != nil {
		return tp.provider.Shutdown(ctx)
	}
	return nil
}

// Tracer returns the tracer instance
func (tp *TracerProvider) Tracer() trace.Tracer {
	return tp.tracer
}

// StartSpan starts a new span with the given name
func (tp *TracerProvider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tp.tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan returns a context with the given span
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}

// GetTraceID extracts the trace ID from context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID extracts the span ID from context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// SpanHelper provides convenience methods for span operations
type SpanHelper struct {
	span trace.Span
}

// NewSpanHelper creates a new span helper
func NewSpanHelper(span trace.Span) *SpanHelper {
	return &SpanHelper{span: span}
}

// SetAttribute sets a single attribute on the span
func (h *SpanHelper) SetAttribute(key string, value interface{}) {
	switch v := value.(type) {
	case string:
		h.span.SetAttributes(attribute.String(key, v))
	case int:
		h.span.SetAttributes(attribute.Int(key, v))
	case int64:
		h.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		h.span.SetAttributes(attribute.Float64(key, v))
	case bool:
		h.span.SetAttributes(attribute.Bool(key, v))
	case []string:
		h.span.SetAttributes(attribute.StringSlice(key, v))
	default:
		h.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

// SetAttributes sets multiple attributes on the span
func (h *SpanHelper) SetAttributes(attrs map[string]interface{}) {
	for k, v := range attrs {
		h.SetAttribute(k, v)
	}
}

// SetError records an error on the span
func (h *SpanHelper) SetError(err error) {
	h.span.RecordError(err)
	h.span.SetStatus(codes.Error, err.Error())
}

// SetOK sets the span status to OK
func (h *SpanHelper) SetOK() {
	h.span.SetStatus(codes.Ok, "")
}

// AddEvent adds an event to the span
func (h *SpanHelper) AddEvent(name string, attrs map[string]interface{}) {
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
	h.span.AddEvent(name, options...)
}

// End ends the span
func (h *SpanHelper) End() {
	h.span.End()
}

// HTTPSpanAttributes returns common HTTP span attributes
func HTTPSpanAttributes(method, path string, statusCode int) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.HTTPMethodKey.String(method),
		semconv.HTTPRouteKey.String(path),
		semconv.HTTPStatusCodeKey.Int(statusCode),
	}
}

// DatabaseSpanAttributes returns common database span attributes
func DatabaseSpanAttributes(dbSystem, dbName, operation, collection string) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.DBSystemKey.String(dbSystem),
		semconv.DBNameKey.String(dbName),
		semconv.DBOperationKey.String(operation),
		attribute.String("db.collection", collection),
	}
}

// MessagingSpanAttributes returns common messaging span attributes
func MessagingSpanAttributes(system, destination, operation string) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.MessagingSystemKey.String(system),
		semconv.MessagingDestinationNameKey.String(destination),
		semconv.MessagingOperationKey.String(operation),
	}
}

// WorkflowSpanAttributes returns Temporal workflow span attributes
func WorkflowSpanAttributes(workflowType, workflowID, runID string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("temporal.workflow.type", workflowType),
		attribute.String("temporal.workflow.id", workflowID),
		attribute.String("temporal.run.id", runID),
	}
}

// ActivitySpanAttributes returns Temporal activity span attributes
func ActivitySpanAttributes(activityType, activityID string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("temporal.activity.type", activityType),
		attribute.String("temporal.activity.id", activityID),
	}
}

// TracedOperation wraps an operation with tracing
func TracedOperation[T any](ctx context.Context, tracer trace.Tracer, spanName string, operation func(context.Context) (T, error)) (T, error) {
	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()

	result, err := operation(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return result, err
}

// TracedVoidOperation wraps a void operation with tracing
func TracedVoidOperation(ctx context.Context, tracer trace.Tracer, spanName string, operation func(context.Context) error) error {
	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()

	err := operation(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// InjectTraceContext injects trace context into a carrier for propagation
func InjectTraceContext(ctx context.Context, carrier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// ExtractTraceContext extracts trace context from a carrier
func ExtractTraceContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// MapCarrier adapts a map to the TextMapCarrier interface
type MapCarrier map[string]string

// Get returns the value for the key
func (c MapCarrier) Get(key string) string {
	return c[key]
}

// Set sets the value for the key
func (c MapCarrier) Set(key, value string) {
	c[key] = value
}

// Keys returns all keys in the carrier
func (c MapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// TimedSpan is a helper for timing operations within a span
type TimedSpan struct {
	span      trace.Span
	startTime time.Time
	name      string
}

// StartTimedSpan starts a new timed span
func StartTimedSpan(ctx context.Context, tracer trace.Tracer, name string) (context.Context, *TimedSpan) {
	ctx, span := tracer.Start(ctx, name)
	return ctx, &TimedSpan{
		span:      span,
		startTime: time.Now(),
		name:      name,
	}
}

// End ends the timed span and records duration
func (ts *TimedSpan) End() time.Duration {
	duration := time.Since(ts.startTime)
	ts.span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))
	ts.span.End()
	return duration
}

// EndWithError ends the span with an error
func (ts *TimedSpan) EndWithError(err error) time.Duration {
	if err != nil {
		ts.span.RecordError(err)
		ts.span.SetStatus(codes.Error, err.Error())
	} else {
		ts.span.SetStatus(codes.Ok, "")
	}
	return ts.End()
}

// SetAttribute sets an attribute on the timed span
func (ts *TimedSpan) SetAttribute(key string, value interface{}) {
	helper := NewSpanHelper(ts.span)
	helper.SetAttribute(key, value)
}
