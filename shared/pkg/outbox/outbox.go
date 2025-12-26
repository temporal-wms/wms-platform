package outbox

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/wms-platform/shared/pkg/cloudevents"
)

// OutboxEvent represents an event stored in the outbox for reliable delivery
type OutboxEvent struct {
	ID            string          `bson:"_id" json:"id"`
	AggregateID   string          `bson:"aggregateId" json:"aggregateId"`
	AggregateType string          `bson:"aggregateType" json:"aggregateType"`
	EventType     string          `bson:"eventType" json:"eventType"`
	Topic         string          `bson:"topic" json:"topic"`
	Payload       json.RawMessage `bson:"payload" json:"payload"`
	CreatedAt     time.Time       `bson:"createdAt" json:"createdAt"`
	PublishedAt   *time.Time      `bson:"publishedAt,omitempty" json:"publishedAt,omitempty"`
	RetryCount    int             `bson:"retryCount" json:"retryCount"`
	LastError     string          `bson:"lastError,omitempty" json:"lastError,omitempty"`
	MaxRetries    int             `bson:"maxRetries" json:"maxRetries"`
}

// DomainEvent interface for domain events
type DomainEvent interface {
	EventType() string
}

// NewOutboxEvent creates a new outbox event from a domain event
func NewOutboxEvent(aggregateID, aggregateType, topic string, event DomainEvent) (*OutboxEvent, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	return &OutboxEvent{
		ID:            uuid.New().String(),
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     event.EventType(),
		Topic:         topic,
		Payload:       payload,
		CreatedAt:     time.Now(),
		RetryCount:    0,
		MaxRetries:    10, // Default max retries
	}, nil
}

// NewOutboxEventFromCloudEvent creates an outbox event from a CloudEvent
func NewOutboxEventFromCloudEvent(aggregateID, aggregateType, topic string, cloudEvent *cloudevents.WMSCloudEvent) (*OutboxEvent, error) {
	payload, err := json.Marshal(cloudEvent)
	if err != nil {
		return nil, err
	}

	return &OutboxEvent{
		ID:            uuid.New().String(),
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     cloudEvent.Type,
		Topic:         topic,
		Payload:       payload,
		CreatedAt:     time.Now(),
		RetryCount:    0,
		MaxRetries:    10,
	}, nil
}

// IsPublished checks if the event has been published
func (e *OutboxEvent) IsPublished() bool {
	return e.PublishedAt != nil
}

// ShouldRetry checks if the event should be retried
func (e *OutboxEvent) ShouldRetry() bool {
	return !e.IsPublished() && e.RetryCount < e.MaxRetries
}

// ToCloudEvent converts the outbox event payload to a CloudEvent
func (e *OutboxEvent) ToCloudEvent() (*cloudevents.WMSCloudEvent, error) {
	var cloudEvent cloudevents.WMSCloudEvent
	if err := json.Unmarshal(e.Payload, &cloudEvent); err != nil {
		return nil, err
	}
	return &cloudEvent, nil
}
