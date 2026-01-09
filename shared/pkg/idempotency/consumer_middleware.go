package idempotency

import (
	"context"
	"log/slog"
	"time"

	"github.com/wms-platform/shared/pkg/cloudevents"
)

// EventHandler is a function that handles a CloudEvent
// This mirrors the kafka.EventHandler type
type EventHandler func(ctx context.Context, event *cloudevents.WMSCloudEvent) error

// DeduplicatingHandler wraps an event handler with deduplication logic
// It ensures exactly-once message processing by checking if a message has already been processed
func DeduplicatingHandler(config *ConsumerConfig, handler EventHandler) EventHandler {
	return func(ctx context.Context, event *cloudevents.WMSCloudEvent) error {
		// Check if message has already been processed
		processed, err := config.Repository.IsProcessed(
			ctx,
			event.ID,
			config.Topic,
			config.ConsumerGroup,
		)

		if err != nil {
			slog.Error("Failed to check if message is processed",
				"error", err,
				"messageId", event.ID,
				"topic", config.Topic,
				"eventType", event.Type,
				"service", config.ServiceName,
			)

			// Record metric
			// Note: metrics would need to be passed in config if needed here
			return err
		}

		if processed {
			// Message already processed, skip
			slog.Info("Duplicate message skipped",
				"messageId", event.ID,
				"topic", config.Topic,
				"eventType", event.Type,
				"service", config.ServiceName,
			)

			// This is a successful case - message was already processed
			return nil
		}

		slog.Debug("Processing new message",
			"messageId", event.ID,
			"topic", config.Topic,
			"eventType", event.Type,
			"service", config.ServiceName,
		)

		// Process the event
		if err := handler(ctx, event); err != nil {
			slog.Error("Failed to process message",
				"error", err,
				"messageId", event.ID,
				"topic", config.Topic,
				"eventType", event.Type,
				"service", config.ServiceName,
			)

			// Don't mark as processed on error - allow retry
			return err
		}

		// Mark message as processed
		msg := &ProcessedMessage{
			MessageID:     event.ID,
			Topic:         config.Topic,
			EventType:     event.Type,
			ConsumerGroup: config.ConsumerGroup,
			ServiceID:     config.ServiceName,
			ProcessedAt:   time.Now().UTC(),
			ExpiresAt:     time.Now().UTC().Add(config.RetentionPeriod),
			CorrelationID: event.CorrelationID,
			WorkflowID:    event.WorkflowID,
		}

		if err := config.Repository.MarkProcessed(ctx, msg); err != nil {
			// Check if it's a duplicate key error (race condition)
			if err == ErrMessageAlreadyProcessed {
				slog.Warn("Message was processed concurrently",
					"messageId", event.ID,
					"topic", config.Topic,
					"eventType", event.Type,
					"service", config.ServiceName,
				)
				// This is OK - the message was processed successfully
				return nil
			}

			slog.Error("Failed to mark message as processed",
				"error", err,
				"messageId", event.ID,
				"topic", config.Topic,
				"eventType", event.Type,
				"service", config.ServiceName,
			)

			// This is a problem - we processed the message but couldn't record it
			// The message might be reprocessed on next retry
			return err
		}

		slog.Debug("Message processed and marked",
			"messageId", event.ID,
			"topic", config.Topic,
			"eventType", event.Type,
			"service", config.ServiceName,
		)

		return nil
	}
}

// DeduplicatingHandlerWithMetrics wraps an event handler with deduplication and metrics
func DeduplicatingHandlerWithMetrics(config *ConsumerConfig, metrics *Metrics, handler EventHandler) EventHandler {
	return func(ctx context.Context, event *cloudevents.WMSCloudEvent) error {
		// Check if message has already been processed
		processed, err := config.Repository.IsProcessed(
			ctx,
			event.ID,
			config.Topic,
			config.ConsumerGroup,
		)

		if err != nil {
			slog.Error("Failed to check if message is processed",
				"error", err,
				"messageId", event.ID,
				"topic", config.Topic,
				"eventType", event.Type,
				"service", config.ServiceName,
			)

			// Record metric
			if metrics != nil {
				metrics.RecordMessageDeduplicationError(config.ServiceName, config.Topic, event.Type)
			}

			return err
		}

		if processed {
			// Message already processed, skip
			slog.Info("Duplicate message skipped",
				"messageId", event.ID,
				"topic", config.Topic,
				"eventType", event.Type,
				"service", config.ServiceName,
			)

			// Record metric
			if metrics != nil {
				metrics.RecordMessageDeduplicationHit(config.ServiceName, config.Topic, event.Type)
			}

			return nil
		}

		// Record cache miss
		if metrics != nil {
			metrics.RecordMessageDeduplicationMiss(config.ServiceName, config.Topic, event.Type)
		}

		slog.Debug("Processing new message",
			"messageId", event.ID,
			"topic", config.Topic,
			"eventType", event.Type,
			"service", config.ServiceName,
		)

		// Process the event
		if err := handler(ctx, event); err != nil {
			slog.Error("Failed to process message",
				"error", err,
				"messageId", event.ID,
				"topic", config.Topic,
				"eventType", event.Type,
				"service", config.ServiceName,
			)

			// Don't mark as processed on error - allow retry
			return err
		}

		// Mark message as processed
		msg := &ProcessedMessage{
			MessageID:     event.ID,
			Topic:         config.Topic,
			EventType:     event.Type,
			ConsumerGroup: config.ConsumerGroup,
			ServiceID:     config.ServiceName,
			ProcessedAt:   time.Now().UTC(),
			ExpiresAt:     time.Now().UTC().Add(config.RetentionPeriod),
			CorrelationID: event.CorrelationID,
			WorkflowID:    event.WorkflowID,
		}

		if err := config.Repository.MarkProcessed(ctx, msg); err != nil {
			// Check if it's a duplicate key error (race condition)
			if err == ErrMessageAlreadyProcessed {
				slog.Warn("Message was processed concurrently",
					"messageId", event.ID,
					"topic", config.Topic,
					"eventType", event.Type,
					"service", config.ServiceName,
				)
				// This is OK - the message was processed successfully
				return nil
			}

			slog.Error("Failed to mark message as processed",
				"error", err,
				"messageId", event.ID,
				"topic", config.Topic,
				"eventType", event.Type,
				"service", config.ServiceName,
			)

			// Record metric
			if metrics != nil {
				metrics.RecordMessageDeduplicationError(config.ServiceName, config.Topic, event.Type)
			}

			return err
		}

		slog.Debug("Message processed and marked",
			"messageId", event.ID,
			"topic", config.Topic,
			"eventType", event.Type,
			"service", config.ServiceName,
		)

		return nil
	}
}
