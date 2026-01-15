package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/wms-platform/shared/pkg/cloudevents"
)

// Producer handles publishing messages to Kafka topics
type Producer struct {
	writers map[string]*kafka.Writer
	config  *Config
}

// NewProducer creates a new Kafka producer
func NewProducer(config *Config) *Producer {
	return &Producer{
		writers: make(map[string]*kafka.Writer),
		config:  config,
	}
}

// getWriter returns a writer for the specified topic, creating one if necessary
func (p *Producer) getWriter(topic string) *kafka.Writer {
	if writer, exists := p.writers[topic]; exists {
		return writer
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(p.config.Brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    p.config.BatchSize,
		BatchTimeout: p.config.BatchTimeout,
		RequiredAcks: kafka.RequiredAcks(p.config.RequiredAcks),
		Async:        false,
	}

	p.writers[topic] = writer
	return writer
}

// PublishEvent publishes a CloudEvent to the specified topic
func (p *Producer) PublishEvent(ctx context.Context, topic string, event *cloudevents.WMSCloudEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	writer := p.getWriter(topic)

	msg := kafka.Message{
		Key:   []byte(event.Subject),
		Value: data,
		Headers: []kafka.Header{
			{Key: "ce-specversion", Value: []byte(event.SpecVersion)},
			{Key: "ce-type", Value: []byte(event.Type)},
			{Key: "ce-source", Value: []byte(event.Source)},
			{Key: "ce-id", Value: []byte(event.ID)},
			{Key: "ce-time", Value: []byte(event.Time.Format(time.RFC3339))},
			{Key: "content-type", Value: []byte(event.DataContentType)},
		},
		Time: event.Time,
	}

	if event.CorrelationID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "ce-wmscorrelationid",
			Value: []byte(event.CorrelationID),
		})
	}

	if event.WaveNumber != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "ce-wmswavenumber",
			Value: []byte(event.WaveNumber),
		})
	}

	if event.WorkflowID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "ce-wmsworkflowid",
			Value: []byte(event.WorkflowID),
		})
	}

	if event.FacilityID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "ce-wmsfacilityid",
			Value: []byte(event.FacilityID),
		})
	}

	if event.WarehouseID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "ce-wmswarehouseid",
			Value: []byte(event.WarehouseID),
		})
	}

	if event.OrderID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "ce-wmsorderid",
			Value: []byte(event.OrderID),
		})
	}

	// Add W3C Trace Context headers
	if event.TraceParent != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "ce-traceparent",
			Value: []byte(event.TraceParent),
		})
	}

	if event.TraceState != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "ce-tracestate",
			Value: []byte(event.TraceState),
		})
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish event to topic %s: %w", topic, err)
	}

	return nil
}

// PublishEventAsync publishes a CloudEvent asynchronously
func (p *Producer) PublishEventAsync(ctx context.Context, topic string, event *cloudevents.WMSCloudEvent, callback func(error)) {
	go func() {
		err := p.PublishEvent(ctx, topic, event)
		if callback != nil {
			callback(err)
		}
	}()
}

// PublishBatch publishes multiple events to a topic
func (p *Producer) PublishBatch(ctx context.Context, topic string, events []*cloudevents.WMSCloudEvent) error {
	messages := make([]kafka.Message, 0, len(events))

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", event.ID, err)
		}

		msg := kafka.Message{
			Key:   []byte(event.Subject),
			Value: data,
			Headers: []kafka.Header{
				{Key: "ce-specversion", Value: []byte(event.SpecVersion)},
				{Key: "ce-type", Value: []byte(event.Type)},
				{Key: "ce-source", Value: []byte(event.Source)},
				{Key: "ce-id", Value: []byte(event.ID)},
				{Key: "ce-time", Value: []byte(event.Time.Format(time.RFC3339))},
				{Key: "content-type", Value: []byte(event.DataContentType)},
			},
			Time: event.Time,
		}

		// Add WMS extension headers
		if event.CorrelationID != "" {
			msg.Headers = append(msg.Headers, kafka.Header{Key: "ce-wmscorrelationid", Value: []byte(event.CorrelationID)})
		}
		if event.WaveNumber != "" {
			msg.Headers = append(msg.Headers, kafka.Header{Key: "ce-wmswavenumber", Value: []byte(event.WaveNumber)})
		}
		if event.WorkflowID != "" {
			msg.Headers = append(msg.Headers, kafka.Header{Key: "ce-wmsworkflowid", Value: []byte(event.WorkflowID)})
		}
		if event.FacilityID != "" {
			msg.Headers = append(msg.Headers, kafka.Header{Key: "ce-wmsfacilityid", Value: []byte(event.FacilityID)})
		}
		if event.WarehouseID != "" {
			msg.Headers = append(msg.Headers, kafka.Header{Key: "ce-wmswarehouseid", Value: []byte(event.WarehouseID)})
		}
		if event.OrderID != "" {
			msg.Headers = append(msg.Headers, kafka.Header{Key: "ce-wmsorderid", Value: []byte(event.OrderID)})
		}

		// Add W3C Trace Context headers
		if event.TraceParent != "" {
			msg.Headers = append(msg.Headers, kafka.Header{Key: "ce-traceparent", Value: []byte(event.TraceParent)})
		}
		if event.TraceState != "" {
			msg.Headers = append(msg.Headers, kafka.Header{Key: "ce-tracestate", Value: []byte(event.TraceState)})
		}

		messages = append(messages, msg)
	}

	writer := p.getWriter(topic)
	if err := writer.WriteMessages(ctx, messages...); err != nil {
		return fmt.Errorf("failed to publish batch to topic %s: %w", topic, err)
	}

	return nil
}

// Close closes all writers
func (p *Producer) Close() error {
	var lastErr error
	for topic, writer := range p.writers {
		if err := writer.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close writer for topic %s: %w", topic, err)
		}
	}
	return lastErr
}
