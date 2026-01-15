package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/logging"
)

// EventHandler is a function that handles a CloudEvent
type EventHandler func(ctx context.Context, event *cloudevents.WMSCloudEvent) error

// Consumer handles consuming messages from Kafka topics
type Consumer struct {
	config   *Config
	readers  map[string]*kafka.Reader
	handlers map[string]map[string]EventHandler // topic -> eventType -> handler
	logger   *slog.Logger
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config *Config, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Consumer{
		config:   config,
		readers:  make(map[string]*kafka.Reader),
		handlers: make(map[string]map[string]EventHandler),
		logger:   logger,
	}
}

// Subscribe subscribes to a topic with a handler for a specific event type
func (c *Consumer) Subscribe(topic string, eventType string, handler EventHandler) {
	if _, exists := c.handlers[topic]; !exists {
		c.handlers[topic] = make(map[string]EventHandler)
	}
	c.handlers[topic][eventType] = handler
}

// SubscribeAll subscribes to all event types on a topic with a single handler
func (c *Consumer) SubscribeAll(topic string, handler EventHandler) {
	c.Subscribe(topic, "*", handler)
}

// getReader returns a reader for the specified topic, creating one if necessary
func (c *Consumer) getReader(topic string) *kafka.Reader {
	if reader, exists := c.readers[topic]; exists {
		return reader
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.config.Brokers,
		GroupID:        c.config.ConsumerGroup,
		Topic:          topic,
		MinBytes:       c.config.MinBytes,
		MaxBytes:       c.config.MaxBytes,
		MaxWait:        c.config.MaxWait,
		CommitInterval: c.config.CommitTimeout,
	})

	c.readers[topic] = reader
	return reader
}

// Start starts consuming messages from all subscribed topics
func (c *Consumer) Start(ctx context.Context) error {
	for topic := range c.handlers {
		go c.consumeTopic(ctx, topic)
	}

	<-ctx.Done()
	return ctx.Err()
}

// consumeTopic consumes messages from a single topic
func (c *Consumer) consumeTopic(ctx context.Context, topic string) {
	reader := c.getReader(topic)

	c.logger.Info("Starting consumer for topic", "topic", topic, "group", c.config.ConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping consumer for topic", "topic", topic)
			return
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Error("Error fetching message", "topic", topic, "error", err)
				continue
			}

			event, err := c.parseMessage(msg)
			if err != nil {
				c.logger.Error("Error parsing message", "topic", topic, "error", err)
				// Commit the message anyway to avoid blocking
				if commitErr := reader.CommitMessages(ctx, msg); commitErr != nil {
					c.logger.Error("Error committing message", "topic", topic, "error", commitErr)
				}
				continue
			}

			if err := c.handleEvent(ctx, topic, event); err != nil {
				c.logger.Error("Error handling event",
					"topic", topic,
					"eventType", event.Type,
					"eventId", event.ID,
					"error", err,
				)
				// Don't commit on handler error - this allows retry
				continue
			}

			if err := reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("Error committing message", "topic", topic, "error", err)
			}
		}
	}
}

// parseMessage parses a Kafka message into a CloudEvent
func (c *Consumer) parseMessage(msg kafka.Message) (*cloudevents.WMSCloudEvent, error) {
	var event cloudevents.WMSCloudEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Extract CloudEvents headers (WMS extensions + W3C trace context)
	for _, header := range msg.Headers {
		switch header.Key {
		case "ce-wmscorrelationid":
			event.CorrelationID = string(header.Value)
		case "ce-wmswavenumber":
			event.WaveNumber = string(header.Value)
		case "ce-wmsworkflowid":
			event.WorkflowID = string(header.Value)
		case "ce-wmsfacilityid":
			event.FacilityID = string(header.Value)
		case "ce-wmswarehouseid":
			event.WarehouseID = string(header.Value)
		case "ce-wmsorderid":
			event.OrderID = string(header.Value)
		// W3C Distributed Tracing extensions
		case "ce-traceparent":
			event.TraceParent = string(header.Value)
		case "ce-tracestate":
			event.TraceState = string(header.Value)
		}
	}

	return &event, nil
}

// handleEvent routes an event to the appropriate handler
func (c *Consumer) handleEvent(ctx context.Context, topic string, event *cloudevents.WMSCloudEvent) error {
	handlers, exists := c.handlers[topic]
	if !exists {
		return fmt.Errorf("no handlers registered for topic %s", topic)
	}

	// Enrich context with CloudEvents WMS extensions for logging
	ctx = logging.ContextWithCloudEventExtensions(
		ctx,
		event.CorrelationID,
		event.WaveNumber,
		event.WorkflowID,
		event.FacilityID,
		event.WarehouseID,
		event.OrderID,
	)

	// Try specific handler first
	if handler, exists := handlers[event.Type]; exists {
		return handler(ctx, event)
	}

	// Fall back to wildcard handler
	if handler, exists := handlers["*"]; exists {
		return handler(ctx, event)
	}

	c.logger.Warn("No handler found for event type", "topic", topic, "eventType", event.Type)
	return nil
}

// Close closes all readers
func (c *Consumer) Close() error {
	var lastErr error
	for topic, reader := range c.readers {
		if err := reader.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close reader for topic %s: %w", topic, err)
		}
	}
	return lastErr
}
