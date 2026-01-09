package idempotency

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds idempotency-related Prometheus metrics
type Metrics struct {
	// IdempotencyHits tracks how many times a cached response was returned
	// Labels: service, endpoint, method
	IdempotencyHits *prometheus.CounterVec

	// IdempotencyMisses tracks how many times a new request was processed
	// Labels: service, endpoint, method
	IdempotencyMisses *prometheus.CounterVec

	// IdempotencyParameterMismatches tracks parameter mismatch errors
	// Labels: service, endpoint, method
	IdempotencyParameterMismatches *prometheus.CounterVec

	// IdempotencyConcurrentCollisions tracks concurrent request conflicts
	// Labels: service, endpoint, method
	IdempotencyConcurrentCollisions *prometheus.CounterVec

	// IdempotencyLockAcquisitionDuration tracks time to acquire locks
	// Labels: service, endpoint, method
	IdempotencyLockAcquisitionDuration *prometheus.HistogramVec

	// IdempotencyStorageErrors tracks storage failures
	// Labels: service, operation
	IdempotencyStorageErrors *prometheus.CounterVec

	// MessageDeduplicationHits tracks duplicate messages skipped
	// Labels: service, topic, event_type
	MessageDeduplicationHits *prometheus.CounterVec

	// MessageDeduplicationMisses tracks new messages processed
	// Labels: service, topic, event_type
	MessageDeduplicationMisses *prometheus.CounterVec

	// MessageDeduplicationErrors tracks errors during deduplication
	// Labels: service, topic, event_type
	MessageDeduplicationErrors *prometheus.CounterVec
}

// NewMetrics creates a new Metrics instance with all counters and histograms registered
func NewMetrics(registry prometheus.Registerer) *Metrics {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	factory := promauto.With(registry)

	return &Metrics{
		IdempotencyHits: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "idempotency_hits_total",
				Help: "Total number of idempotency cache hits (cached response returned)",
			},
			[]string{"service", "endpoint", "method"},
		),

		IdempotencyMisses: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "idempotency_misses_total",
				Help: "Total number of idempotency cache misses (new request processed)",
			},
			[]string{"service", "endpoint", "method"},
		),

		IdempotencyParameterMismatches: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "idempotency_parameter_mismatches_total",
				Help: "Total number of parameter mismatch errors (same key, different body)",
			},
			[]string{"service", "endpoint", "method"},
		),

		IdempotencyConcurrentCollisions: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "idempotency_concurrent_collisions_total",
				Help: "Total number of concurrent request collisions (409 Conflict)",
			},
			[]string{"service", "endpoint", "method"},
		),

		IdempotencyLockAcquisitionDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "idempotency_lock_acquisition_duration_seconds",
				Help:    "Time taken to acquire idempotency lock",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "endpoint", "method"},
		),

		IdempotencyStorageErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "idempotency_storage_errors_total",
				Help: "Total number of idempotency storage errors",
			},
			[]string{"service", "operation"},
		),

		MessageDeduplicationHits: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_deduplication_hits_total",
				Help: "Total number of duplicate messages skipped",
			},
			[]string{"service", "topic", "event_type"},
		),

		MessageDeduplicationMisses: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_deduplication_misses_total",
				Help: "Total number of new messages processed",
			},
			[]string{"service", "topic", "event_type"},
		),

		MessageDeduplicationErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_deduplication_errors_total",
				Help: "Total number of errors during message deduplication",
			},
			[]string{"service", "topic", "event_type"},
		),
	}
}

// RecordHit records an idempotency cache hit
func (m *Metrics) RecordHit(service, endpoint, method string) {
	if m.IdempotencyHits != nil {
		m.IdempotencyHits.WithLabelValues(service, endpoint, method).Inc()
	}
}

// RecordMiss records an idempotency cache miss
func (m *Metrics) RecordMiss(service, endpoint, method string) {
	if m.IdempotencyMisses != nil {
		m.IdempotencyMisses.WithLabelValues(service, endpoint, method).Inc()
	}
}

// RecordParameterMismatch records a parameter mismatch error
func (m *Metrics) RecordParameterMismatch(service, endpoint, method string) {
	if m.IdempotencyParameterMismatches != nil {
		m.IdempotencyParameterMismatches.WithLabelValues(service, endpoint, method).Inc()
	}
}

// RecordConcurrentCollision records a concurrent request collision
func (m *Metrics) RecordConcurrentCollision(service, endpoint, method string) {
	if m.IdempotencyConcurrentCollisions != nil {
		m.IdempotencyConcurrentCollisions.WithLabelValues(service, endpoint, method).Inc()
	}
}

// RecordLockAcquisitionDuration records the time taken to acquire a lock
func (m *Metrics) RecordLockAcquisitionDuration(service, endpoint, method string, duration float64) {
	if m.IdempotencyLockAcquisitionDuration != nil {
		m.IdempotencyLockAcquisitionDuration.WithLabelValues(service, endpoint, method).Observe(duration)
	}
}

// RecordStorageError records a storage error
func (m *Metrics) RecordStorageError(service, operation string) {
	if m.IdempotencyStorageErrors != nil {
		m.IdempotencyStorageErrors.WithLabelValues(service, operation).Inc()
	}
}

// RecordMessageDeduplicationHit records a duplicate message being skipped
func (m *Metrics) RecordMessageDeduplicationHit(service, topic, eventType string) {
	if m.MessageDeduplicationHits != nil {
		m.MessageDeduplicationHits.WithLabelValues(service, topic, eventType).Inc()
	}
}

// RecordMessageDeduplicationMiss records a new message being processed
func (m *Metrics) RecordMessageDeduplicationMiss(service, topic, eventType string) {
	if m.MessageDeduplicationMisses != nil {
		m.MessageDeduplicationMisses.WithLabelValues(service, topic, eventType).Inc()
	}
}

// RecordMessageDeduplicationError records an error during message deduplication
func (m *Metrics) RecordMessageDeduplicationError(service, topic, eventType string) {
	if m.MessageDeduplicationErrors != nil {
		m.MessageDeduplicationErrors.WithLabelValues(service, topic, eventType).Inc()
	}
}
