package outbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
)

// Publisher publishes events from the outbox to Kafka
type Publisher struct {
	repo          Repository
	producer      *kafka.InstrumentedProducer
	logger        *logging.Logger
	metrics       *metrics.Metrics
	interval      time.Duration
	batchSize     int
	mu            sync.Mutex
	running       bool
	stopCh        chan struct{}
	stoppedCh     chan struct{}
	publishedCnt  int
	failedCnt     int
}

// PublisherConfig holds configuration for the outbox publisher
type PublisherConfig struct {
	PollInterval time.Duration
	BatchSize    int
}

// DefaultPublisherConfig returns default configuration
func DefaultPublisherConfig() *PublisherConfig {
	return &PublisherConfig{
		PollInterval: 1 * time.Second,
		BatchSize:    100,
	}
}

// NewPublisher creates a new outbox publisher
func NewPublisher(
	repo Repository,
	producer *kafka.InstrumentedProducer,
	logger *logging.Logger,
	metrics *metrics.Metrics,
	config *PublisherConfig,
) *Publisher {
	if config == nil {
		config = DefaultPublisherConfig()
	}

	return &Publisher{
		repo:      repo,
		producer:  producer,
		logger:    logger,
		metrics:   metrics,
		interval:  config.PollInterval,
		batchSize: config.BatchSize,
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
}

// Start starts the outbox publisher
func (p *Publisher) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("publisher already running")
	}
	p.running = true
	p.mu.Unlock()

	p.logger.Info("Starting outbox publisher", "interval", p.interval, "batchSize", p.batchSize)

	go p.run(ctx)
	return nil
}

// Stop stops the outbox publisher
func (p *Publisher) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return fmt.Errorf("publisher not running")
	}
	p.mu.Unlock()

	p.logger.Info("Stopping outbox publisher")
	close(p.stopCh)
	<-p.stoppedCh

	p.mu.Lock()
	p.running = false
	p.mu.Unlock()

	p.logger.Info("Outbox publisher stopped", "published", p.publishedCnt, "failed", p.failedCnt)
	return nil
}

// run is the main publisher loop
func (p *Publisher) run(ctx context.Context) {
	defer close(p.stoppedCh)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.processEvents(ctx)
		case <-p.stopCh:
			p.logger.Info("Publisher received stop signal")
			return
		case <-ctx.Done():
			p.logger.Info("Publisher context cancelled")
			return
		}
	}
}

// processEvents processes unpublished events
func (p *Publisher) processEvents(ctx context.Context) {
	events, err := p.repo.FindUnpublished(ctx, p.batchSize)
	if err != nil {
		p.logger.WithError(err).Error("Failed to find unpublished events")
		return
	}

	// Record pending events count
	if p.metrics != nil {
		p.metrics.SetOutboxPending(len(events))
	}

	if len(events) == 0 {
		return
	}

	p.logger.Info("Processing outbox events", "count", len(events))

	for _, event := range events {
		duration, err := p.publishEvent(ctx, event)
		if err != nil {
			p.logger.WithError(err).Error("Failed to publish event",
				"eventId", event.ID,
				"eventType", event.EventType,
				"aggregateId", event.AggregateID,
			)
			p.failedCnt++

			// Record failed publish metric
			if p.metrics != nil {
				p.metrics.RecordOutboxPublish(event.EventType, false, duration)
			}

			// Increment retry count
			if err := p.repo.IncrementRetry(ctx, event.ID, err.Error()); err != nil {
				p.logger.WithError(err).Error("Failed to increment retry count", "eventId", event.ID)
			}

			// Record retry metric
			if p.metrics != nil {
				p.metrics.RecordOutboxRetry(event.EventType)
			}
		} else {
			p.publishedCnt++

			// Record successful publish metric
			if p.metrics != nil {
				p.metrics.RecordOutboxPublish(event.EventType, true, duration)
			}

			// Mark as published
			if err := p.repo.MarkPublished(ctx, event.ID); err != nil {
				p.logger.WithError(err).Error("Failed to mark event as published", "eventId", event.ID)
			}
		}
	}
}

// publishEvent publishes a single event to Kafka and returns the duration
func (p *Publisher) publishEvent(ctx context.Context, event *OutboxEvent) (time.Duration, error) {
	start := time.Now()

	// Convert to CloudEvent
	cloudEvent, err := event.ToCloudEvent()
	if err != nil {
		return time.Since(start), fmt.Errorf("failed to convert to CloudEvent: %w", err)
	}

	// Publish to Kafka
	if err := p.producer.PublishEvent(ctx, event.Topic, cloudEvent); err != nil {
		return time.Since(start), fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	duration := time.Since(start)

	p.logger.Info("Published event from outbox",
		"eventId", event.ID,
		"eventType", event.EventType,
		"topic", event.Topic,
		"aggregateId", event.AggregateID,
		"duration", duration,
	)

	return duration, nil
}

// IsRunning returns whether the publisher is running
func (p *Publisher) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// Stats returns publisher statistics
func (p *Publisher) Stats() map[string]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]int{
		"published": p.publishedCnt,
		"failed":    p.failedCnt,
	}
}
