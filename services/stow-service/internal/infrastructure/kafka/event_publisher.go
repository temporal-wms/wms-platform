package kafka

import (
	"context"
	"fmt"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/stow-service/internal/domain"
)

// EventPublisher implements domain event publishing using Kafka
type EventPublisher struct {
	producer     *kafka.InstrumentedProducer
	eventFactory *cloudevents.EventFactory
	topic        string
}

// NewEventPublisher creates a new Kafka-based event publisher
func NewEventPublisher(
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	topic string,
) *EventPublisher {
	return &EventPublisher{
		producer:     producer,
		eventFactory: eventFactory,
		topic:        topic,
	}
}

// Publish publishes a single domain event to Kafka
func (p *EventPublisher) Publish(ctx context.Context, event domain.DomainEvent) error {
	// Determine the source based on event type
	var source string
	switch e := event.(type) {
	case *domain.PutawayTaskCreatedEvent:
		source = "stow/task/" + e.TaskID
	case *domain.LocationAssignedEvent:
		source = "stow/task/" + e.TaskID
	case *domain.ItemStowedEvent:
		source = "stow/task/" + e.TaskID
	case *domain.PutawayTaskAssignedEvent:
		source = "stow/task/" + e.TaskID
	case *domain.PutawayTaskCompletedEvent:
		source = "stow/task/" + e.TaskID
	case *domain.PutawayTaskFailedEvent:
		source = "stow/task/" + e.TaskID
	default:
		source = "stow"
	}

	// Convert domain event to CloudEvent
	ce := p.eventFactory.CreateEvent(ctx, event.EventType(), source, event)

	// Publish to Kafka using the instrumented producer
	if err := p.producer.PublishEvent(ctx, p.topic, ce); err != nil {
		return fmt.Errorf("failed to publish event to kafka: %w", err)
	}

	return nil
}

// PublishAll publishes multiple domain events to Kafka
func (p *EventPublisher) PublishAll(ctx context.Context, events []domain.DomainEvent) error {
	for _, event := range events {
		if err := p.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// GetTopic returns the topic this publisher publishes to
func (p *EventPublisher) GetTopic() string {
	return p.topic
}
