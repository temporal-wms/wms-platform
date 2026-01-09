package eventcapture

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/wms-platform/wms-runtime/validation-service/internal/validation"
)

// KafkaSubscriber subscribes to all WMS Kafka topics and captures events
type KafkaSubscriber struct {
	brokers        string
	consumer       *kafka.Consumer
	eventStore     *EventStore
	eventValidator *validation.EventValidator
	topics         []string
}

// NewKafkaSubscriber creates a new Kafka subscriber
func NewKafkaSubscriber(brokers string, eventStore *EventStore, eventValidator *validation.EventValidator) *KafkaSubscriber {
	return &KafkaSubscriber{
		brokers:        brokers,
		eventStore:     eventStore,
		eventValidator: eventValidator,
		topics: []string{
			"wms.orders.events",
			"wms.waves.events",
			"wms.picking.events",
			"wms.inventory.events",
			"wms.labor.events",
			"wms.consolidation.events",
			"wms.packing.events",
			"wms.shipping.events",
			"wms.routing.events",
			"wms.walling.events",
			"wms.wes.events",
			"wms.facility.events",
			"wms.receiving.events",
			"wms.stow.events",
			"wms.sortation.events",
			"wms.unit.events",
		},
	}
}

// Start begins consuming events from Kafka
func (s *KafkaSubscriber) Start(ctx context.Context) error {
	// Create Kafka consumer
	config := &kafka.ConfigMap{
		"bootstrap.servers":  s.brokers,
		"group.id":           "validation-service",
		"auto.offset.reset":  "latest",
		"enable.auto.commit": true,
	}

	var err error
	s.consumer, err = kafka.NewConsumer(config)
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	// Subscribe to all WMS topics
	if err := s.consumer.SubscribeTopics(s.topics, nil); err != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	log.Printf("Subscribed to %d Kafka topics", len(s.topics))

	// Start consuming
	for {
		select {
		case <-ctx.Done():
			log.Println("Kafka subscriber context cancelled, stopping...")
			return nil
		default:
			msg, err := s.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					continue
				}
				log.Printf("Consumer error: %v", err)
				continue
			}

			// Process the message
			s.processMessage(msg)
		}
	}
}

// processMessage processes a Kafka message and stores it
func (s *KafkaSubscriber) processMessage(msg *kafka.Message) {
	// Parse CloudEvents format
	var rawEvent map[string]interface{}
	if err := json.Unmarshal(msg.Value, &rawEvent); err != nil {
		log.Printf("Failed to parse event: %v", err)
		return
	}

	// Extract CloudEvents fields
	eventType, _ := rawEvent["type"].(string)
	eventSource, _ := rawEvent["source"].(string)
	cloudEventsID, _ := rawEvent["id"].(string)

	// Extract orderId from data field
	data, _ := rawEvent["data"].(map[string]interface{})
	orderID := extractOrderID(data, eventType)

	// Skip if no orderId found
	if orderID == "" {
		return
	}

	// Only process if this order is being tracked
	if !s.eventStore.IsTracking(orderID) {
		return
	}

	// Create captured event
	capturedEvent := &CapturedEvent{
		ID:            cloudEventsID,
		Type:          eventType,
		Source:        eventSource,
		OrderID:       orderID,
		Topic:         *msg.TopicPartition.Topic,
		Partition:     msg.TopicPartition.Partition,
		Offset:        int64(msg.TopicPartition.Offset),
		Timestamp:     msg.Timestamp,
		CapturedAt:    time.Now(),
		Data:          data,
		CloudEventsID: cloudEventsID,
	}

	// Validate event (optional, non-blocking)
	go s.validateEvent(capturedEvent, rawEvent)

	// Store event
	s.eventStore.AddEvent(capturedEvent)

	log.Printf("Captured event: type=%s, orderId=%s, topic=%s", eventType, orderID, *msg.TopicPartition.Topic)
}

// validateEvent validates an event against AsyncAPI schema
func (s *KafkaSubscriber) validateEvent(capturedEvent *CapturedEvent, rawEvent map[string]interface{}) {
	if err := s.eventValidator.Validate(capturedEvent.Type, rawEvent); err != nil {
		log.Printf("Event validation failed: type=%s, orderId=%s, error=%v",
			capturedEvent.Type, capturedEvent.OrderID, err)
	}
}

// extractOrderID extracts orderId from event data
func extractOrderID(data map[string]interface{}, eventType string) string {
	// Try direct orderId field
	if orderID, ok := data["orderId"].(string); ok && orderID != "" {
		return orderID
	}

	// Try order_id field
	if orderID, ok := data["order_id"].(string); ok && orderID != "" {
		return orderID
	}

	// Try nested order.orderId
	if order, ok := data["order"].(map[string]interface{}); ok {
		if orderID, ok := order["orderId"].(string); ok && orderID != "" {
			return orderID
		}
		if orderID, ok := order["order_id"].(string); ok && orderID != "" {
			return orderID
		}
	}

	// Try aggregateId (for DDD events)
	if aggregateID, ok := data["aggregateId"].(string); ok && aggregateID != "" {
		// Check if this looks like an order ID (starts with ORD-)
		if strings.HasPrefix(aggregateID, "ORD-") {
			return aggregateID
		}
	}

	return ""
}

// Close closes the Kafka consumer
func (s *KafkaSubscriber) Close() error {
	if s.consumer != nil {
		return s.consumer.Close()
	}
	return nil
}
