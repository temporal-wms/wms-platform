package kafka

import (
	"context"
	"time"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// addWMSCloudEventAttributes adds WMS extension attributes to a span
func addWMSCloudEventAttributes(span trace.Span, event *cloudevents.WMSCloudEvent) {
	if event.CorrelationID != "" {
		span.SetAttributes(attribute.String("wms.correlation_id", event.CorrelationID))
	}
	if event.WaveNumber != "" {
		span.SetAttributes(attribute.String("wms.wave_number", event.WaveNumber))
	}
	if event.WorkflowID != "" {
		span.SetAttributes(attribute.String("wms.workflow_id", event.WorkflowID))
	}
}

// InstrumentedProducer wraps a Producer with metrics and tracing
type InstrumentedProducer struct {
	producer *Producer
	metrics  *metrics.Metrics
	logger   *logging.Logger
	tracer   trace.Tracer
}

// NewInstrumentedProducer creates a new instrumented producer
func NewInstrumentedProducer(producer *Producer, m *metrics.Metrics, logger *logging.Logger) *InstrumentedProducer {
	return &InstrumentedProducer{
		producer: producer,
		metrics:  m,
		logger:   logger,
		tracer:   otel.Tracer("kafka-producer"),
	}
}

// PublishEvent publishes a CloudEvent with metrics and tracing
func (p *InstrumentedProducer) PublishEvent(ctx context.Context, topic string, event *cloudevents.WMSCloudEvent) error {
	start := time.Now()

	// Start tracing span
	ctx, span := p.tracer.Start(ctx, "kafka.publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingDestinationNameKey.String(topic),
			semconv.MessagingOperationKey.String("publish"),
			attribute.String("messaging.kafka.event_type", event.Type),
			attribute.String("messaging.message_id", event.ID),
		),
	)
	defer span.End()

	// Add WMS CloudEvents extension attributes
	addWMSCloudEventAttributes(span, event)

	// Inject trace context into event headers (via correlation ID for now)
	carrier := tracing.MapCarrier{}
	tracing.InjectTraceContext(ctx, carrier)
	if _, ok := carrier["traceparent"]; ok {
		// Store trace context in event for propagation
		if event.CorrelationID == "" {
			event.CorrelationID = event.ID
		}
	}

	// Publish the event
	err := p.producer.PublishEvent(ctx, topic, event)
	duration := time.Since(start)

	// Record metrics
	success := err == nil
	if p.metrics != nil {
		p.metrics.RecordKafkaPublish(topic, event.Type, success, duration)
	}

	// Log the operation
	if p.logger != nil {
		p.logger.KafkaPublish(ctx, topic, event.Type, success, duration)
	}

	// Update span status
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int64("messaging.duration_ms", duration.Milliseconds()))
	}

	return err
}

// PublishEventAsync publishes a CloudEvent asynchronously with metrics
func (p *InstrumentedProducer) PublishEventAsync(ctx context.Context, topic string, event *cloudevents.WMSCloudEvent, callback func(error)) {
	start := time.Now()

	// Start tracing span (detached for async)
	_, span := p.tracer.Start(ctx, "kafka.publish.async",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingDestinationNameKey.String(topic),
			semconv.MessagingOperationKey.String("publish"),
			attribute.String("messaging.kafka.event_type", event.Type),
			attribute.String("messaging.message_id", event.ID),
			attribute.Bool("messaging.async", true),
		),
	)

	// Add WMS CloudEvents extension attributes
	addWMSCloudEventAttributes(span, event)

	wrappedCallback := func(err error) {
		defer span.End()
		duration := time.Since(start)

		success := err == nil
		if p.metrics != nil {
			p.metrics.RecordKafkaPublish(topic, event.Type, success, duration)
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}

		if callback != nil {
			callback(err)
		}
	}

	p.producer.PublishEventAsync(ctx, topic, event, wrappedCallback)
}

// PublishBatch publishes multiple events with metrics and tracing
func (p *InstrumentedProducer) PublishBatch(ctx context.Context, topic string, events []*cloudevents.WMSCloudEvent) error {
	start := time.Now()

	// Start tracing span
	ctx, span := p.tracer.Start(ctx, "kafka.publish.batch",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingDestinationNameKey.String(topic),
			semconv.MessagingOperationKey.String("publish"),
			attribute.Int("messaging.batch_size", len(events)),
		),
	)
	defer span.End()

	// Publish the batch
	err := p.producer.PublishBatch(ctx, topic, events)
	duration := time.Since(start)

	// Record metrics for each event
	success := err == nil
	if p.metrics != nil {
		for _, event := range events {
			p.metrics.RecordKafkaPublish(topic, event.Type, success, duration/time.Duration(len(events)))
		}
	}

	// Update span status
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int64("messaging.duration_ms", duration.Milliseconds()))
	}

	return err
}

// Close closes the underlying producer
func (p *InstrumentedProducer) Close() error {
	return p.producer.Close()
}

// InstrumentedConsumer wraps a Consumer with metrics and tracing
type InstrumentedConsumer struct {
	consumer *Consumer
	metrics  *metrics.Metrics
	logger   *logging.Logger
	tracer   trace.Tracer
}

// NewInstrumentedConsumer creates a new instrumented consumer
func NewInstrumentedConsumer(consumer *Consumer, m *metrics.Metrics, logger *logging.Logger) *InstrumentedConsumer {
	return &InstrumentedConsumer{
		consumer: consumer,
		metrics:  m,
		logger:   logger,
		tracer:   otel.Tracer("kafka-consumer"),
	}
}

// Subscribe subscribes to a topic with instrumented handler
func (c *InstrumentedConsumer) Subscribe(topic string, eventType string, handler EventHandler) {
	wrappedHandler := c.instrumentHandler(topic, eventType, handler)
	c.consumer.Subscribe(topic, eventType, wrappedHandler)
}

// SubscribeAll subscribes to all event types with instrumented handler
func (c *InstrumentedConsumer) SubscribeAll(topic string, handler EventHandler) {
	wrappedHandler := c.instrumentHandler(topic, "*", handler)
	c.consumer.SubscribeAll(topic, wrappedHandler)
}

// instrumentHandler wraps an event handler with metrics and tracing
func (c *InstrumentedConsumer) instrumentHandler(topic, eventType string, handler EventHandler) EventHandler {
	return func(ctx context.Context, event *cloudevents.WMSCloudEvent) error {
		start := time.Now()

		// Extract trace context from event if available
		if event.CorrelationID != "" {
			// Try to extract parent context (simplified - in production you'd parse trace headers)
			carrier := tracing.MapCarrier{
				"correlationId": event.CorrelationID,
			}
			ctx = tracing.ExtractTraceContext(ctx, carrier)
		}

		// Start tracing span
		ctx, span := c.tracer.Start(ctx, "kafka.consume",
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(
				semconv.MessagingSystemKey.String("kafka"),
				semconv.MessagingDestinationNameKey.String(topic),
				semconv.MessagingOperationKey.String("receive"),
				attribute.String("messaging.kafka.event_type", event.Type),
				attribute.String("messaging.message_id", event.ID),
				attribute.String("messaging.kafka.consumer_group", c.consumer.config.ConsumerGroup),
			),
		)
		defer span.End()

		// Add WMS CloudEvents extension attributes
		addWMSCloudEventAttributes(span, event)

		// Handle the event
		err := handler(ctx, event)
		duration := time.Since(start)

		// Record metrics
		success := err == nil
		if c.metrics != nil {
			c.metrics.RecordKafkaConsume(topic, event.Type, success)
		}

		// Log the operation
		if c.logger != nil {
			c.logger.KafkaConsume(ctx, topic, event.Type, 0, 0) // partition/offset not available here
		}

		// Update span status
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(attribute.Int64("messaging.processing_duration_ms", duration.Milliseconds()))
		}

		return err
	}
}

// Start starts the instrumented consumer
func (c *InstrumentedConsumer) Start(ctx context.Context) error {
	return c.consumer.Start(ctx)
}

// Close closes the underlying consumer
func (c *InstrumentedConsumer) Close() error {
	return c.consumer.Close()
}

// SetConsumerLag updates the consumer lag metric
func (c *InstrumentedConsumer) SetConsumerLag(topic string, partition int, lag int64) {
	if c.metrics != nil {
		c.metrics.SetKafkaConsumerLag(topic, partition, lag)
	}
}

// ProducerMetricsCollector collects producer metrics periodically
type ProducerMetricsCollector struct {
	producer *InstrumentedProducer
	metrics  *metrics.Metrics
}

// NewProducerMetricsCollector creates a new metrics collector
func NewProducerMetricsCollector(producer *InstrumentedProducer, m *metrics.Metrics) *ProducerMetricsCollector {
	return &ProducerMetricsCollector{
		producer: producer,
		metrics:  m,
	}
}

// ConsumerMetricsCollector collects consumer metrics periodically
type ConsumerMetricsCollector struct {
	consumer *InstrumentedConsumer
	metrics  *metrics.Metrics
}

// NewConsumerMetricsCollector creates a new metrics collector
func NewConsumerMetricsCollector(consumer *InstrumentedConsumer, m *metrics.Metrics) *ConsumerMetricsCollector {
	return &ConsumerMetricsCollector{
		consumer: consumer,
		metrics:  m,
	}
}

// KafkaTracePropagator helps propagate trace context through Kafka messages
type KafkaTracePropagator struct {
	propagator propagation.TextMapPropagator
}

// NewKafkaTracePropagator creates a new trace propagator
func NewKafkaTracePropagator() *KafkaTracePropagator {
	return &KafkaTracePropagator{
		propagator: otel.GetTextMapPropagator(),
	}
}

// InjectContext injects trace context into Kafka headers
func (p *KafkaTracePropagator) InjectContext(ctx context.Context, headers map[string]string) {
	carrier := propagation.MapCarrier(headers)
	p.propagator.Inject(ctx, carrier)
}

// ExtractContext extracts trace context from Kafka headers
func (p *KafkaTracePropagator) ExtractContext(ctx context.Context, headers map[string]string) context.Context {
	carrier := propagation.MapCarrier(headers)
	return p.propagator.Extract(ctx, carrier)
}
