package kafka

import (
	"context"
	"log/slog"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/resilience"
)

// CircuitBreakerProducer wraps InstrumentedProducer with circuit breaker protection
type CircuitBreakerProducer struct {
	producer       *InstrumentedProducer
	circuitBreaker *resilience.CircuitBreaker
	logger         *logging.Logger
}

// NewCircuitBreakerProducer creates a new circuit breaker protected Kafka producer
func NewCircuitBreakerProducer(producer *InstrumentedProducer, logger *logging.Logger) *CircuitBreakerProducer {
	// Create circuit breaker config for Kafka producer
	config := &resilience.CircuitBreakerConfig{
		Name:                  "kafka-producer",
		MaxRequests:           5,
		Interval:              60, // 1 minute
		Timeout:               30, // 30 seconds
		FailureThreshold:      5,
		SuccessThreshold:      2,
		FailureRatioThreshold: 0.5,
		MinRequestsToTrip:     10,
	}

	var slogLogger *slog.Logger
	if logger != nil && logger.Logger != nil {
		slogLogger = logger.Logger
	} else {
		slogLogger = slog.Default()
	}

	cb := resilience.NewCircuitBreaker(config, slogLogger)

	return &CircuitBreakerProducer{
		producer:       producer,
		circuitBreaker: cb,
		logger:         logger,
	}
}

// PublishEvent publishes a CloudEvent with circuit breaker protection
func (p *CircuitBreakerProducer) PublishEvent(ctx context.Context, topic string, event *cloudevents.WMSCloudEvent) error {
	_, err := p.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, p.producer.PublishEvent(ctx, topic, event)
	})
	return err
}

// PublishEventAsync publishes a CloudEvent asynchronously with circuit breaker protection
func (p *CircuitBreakerProducer) PublishEventAsync(ctx context.Context, topic string, event *cloudevents.WMSCloudEvent, callback func(error)) {
	// For async operations, we check circuit breaker state first
	if p.circuitBreaker.State() == 2 { // Open state
		if callback != nil {
			callback(resilience.ErrCircuitOpen)
		}
		return
	}

	// Wrap the callback to record success/failure with circuit breaker
	wrappedCallback := func(err error) {
		// Record the result with circuit breaker
		if err != nil {
			p.circuitBreaker.Execute(ctx, func() (interface{}, error) {
				return nil, err
			})
		}

		if callback != nil {
			callback(err)
		}
	}

	p.producer.PublishEventAsync(ctx, topic, event, wrappedCallback)
}

// PublishBatch publishes multiple events with circuit breaker protection
func (p *CircuitBreakerProducer) PublishBatch(ctx context.Context, topic string, events []*cloudevents.WMSCloudEvent) error {
	_, err := p.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, p.producer.PublishBatch(ctx, topic, events)
	})
	return err
}

// Close closes the underlying producer
func (p *CircuitBreakerProducer) Close() error {
	return p.producer.Close()
}

// Underlying returns the underlying InstrumentedProducer
func (p *CircuitBreakerProducer) Underlying() *InstrumentedProducer {
	return p.producer
}

// CircuitBreakerConsumer wraps InstrumentedConsumer with circuit breaker protection
type CircuitBreakerConsumer struct {
	consumer       *InstrumentedConsumer
	circuitBreaker *resilience.CircuitBreaker
	logger         *logging.Logger
}

// NewCircuitBreakerConsumer creates a new circuit breaker protected Kafka consumer
func NewCircuitBreakerConsumer(consumer *InstrumentedConsumer, logger *logging.Logger) *CircuitBreakerConsumer {
	// Create circuit breaker config for Kafka consumer
	config := &resilience.CircuitBreakerConfig{
		Name:                  "kafka-consumer",
		MaxRequests:           5,
		Interval:              60, // 1 minute
		Timeout:               30, // 30 seconds
		FailureThreshold:      10, // Higher threshold for consumers
		SuccessThreshold:      3,
		FailureRatioThreshold: 0.7, // Higher ratio for consumers
		MinRequestsToTrip:     20,
	}

	var slogLogger *slog.Logger
	if logger != nil && logger.Logger != nil {
		slogLogger = logger.Logger
	} else {
		slogLogger = slog.Default()
	}

	cb := resilience.NewCircuitBreaker(config, slogLogger)

	return &CircuitBreakerConsumer{
		consumer:       consumer,
		circuitBreaker: cb,
		logger:         logger,
	}
}

// Subscribe subscribes to a topic with circuit breaker protected handler
func (c *CircuitBreakerConsumer) Subscribe(topic string, eventType string, handler EventHandler) {
	// Wrap handler with circuit breaker
	wrappedHandler := func(ctx context.Context, event *cloudevents.WMSCloudEvent) error {
		_, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
			return nil, handler(ctx, event)
		})
		return err
	}

	c.consumer.Subscribe(topic, eventType, wrappedHandler)
}

// SubscribeAll subscribes to all event types with circuit breaker protected handler
func (c *CircuitBreakerConsumer) SubscribeAll(topic string, handler EventHandler) {
	// Wrap handler with circuit breaker
	wrappedHandler := func(ctx context.Context, event *cloudevents.WMSCloudEvent) error {
		_, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
			return nil, handler(ctx, event)
		})
		return err
	}

	c.consumer.SubscribeAll(topic, wrappedHandler)
}

// Start starts the circuit breaker protected consumer
func (c *CircuitBreakerConsumer) Start(ctx context.Context) error {
	_, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, c.consumer.Start(ctx)
	})
	return err
}

// Close closes the underlying consumer
func (c *CircuitBreakerConsumer) Close() error {
	return c.consumer.Close()
}

// SetConsumerLag updates the consumer lag metric
func (c *CircuitBreakerConsumer) SetConsumerLag(topic string, partition int, lag int64) {
	c.consumer.SetConsumerLag(topic, partition, lag)
}

// Underlying returns the underlying InstrumentedConsumer
func (c *CircuitBreakerConsumer) Underlying() *InstrumentedConsumer {
	return c.consumer
}

// NewProductionProducer creates a fully configured Kafka producer with instrumentation and circuit breaker
func NewProductionProducer(config *Config, m *metrics.Metrics, logger *logging.Logger) *CircuitBreakerProducer {
	// Create base producer
	baseProducer := NewProducer(config)

	// Wrap with instrumentation
	instrumentedProducer := NewInstrumentedProducer(baseProducer, m, logger)

	// Wrap with circuit breaker
	cbProducer := NewCircuitBreakerProducer(instrumentedProducer, logger)

	return cbProducer
}

// NewProductionConsumer creates a fully configured Kafka consumer with instrumentation and circuit breaker
func NewProductionConsumer(config *Config, m *metrics.Metrics, logger *logging.Logger) *CircuitBreakerConsumer {
	// Create base consumer
	baseConsumer := NewConsumer(config, logger.Logger)

	// Wrap with instrumentation
	instrumentedConsumer := NewInstrumentedConsumer(baseConsumer, m, logger)

	// Wrap with circuit breaker
	cbConsumer := NewCircuitBreakerConsumer(instrumentedConsumer, logger)

	return cbConsumer
}
