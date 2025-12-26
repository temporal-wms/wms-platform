package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all WMS metrics
type Metrics struct {
	serviceName string
	registry    *prometheus.Registry

	// HTTP metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// Kafka metrics
	KafkaEventsPublished *prometheus.CounterVec
	KafkaEventsConsumed  *prometheus.CounterVec
	KafkaPublishDuration *prometheus.HistogramVec
	KafkaConsumeLag      *prometheus.GaugeVec

	// MongoDB metrics
	MongoDBOperations       *prometheus.CounterVec
	MongoDBOperationDuration *prometheus.HistogramVec
	MongoDBConnectionsOpen  prometheus.Gauge

	// Temporal metrics
	WorkflowsStarted   *prometheus.CounterVec
	WorkflowsCompleted *prometheus.CounterVec
	WorkflowDuration   *prometheus.HistogramVec
	ActivitiesStarted  *prometheus.CounterVec
	ActivitiesCompleted *prometheus.CounterVec
	ActivityDuration   *prometheus.HistogramVec

	// Business metrics
	OrdersCreated    *prometheus.CounterVec
	OrdersProcessed  *prometheus.CounterVec
	ItemsPicked      *prometheus.CounterVec
	PackagesShipped  *prometheus.CounterVec

	// Circuit breaker metrics
	CircuitBreakerState   *prometheus.GaugeVec
	CircuitBreakerTrips   *prometheus.CounterVec
}

// Config holds metrics configuration
type Config struct {
	ServiceName string
	Namespace   string
	Subsystem   string
}

// DefaultConfig returns default metrics configuration
func DefaultConfig(serviceName string) *Config {
	return &Config{
		ServiceName: serviceName,
		Namespace:   "wms",
		Subsystem:   serviceName,
	}
}

// New creates a new Metrics instance
func New(config *Config) *Metrics {
	registry := prometheus.NewRegistry()

	// Register standard Go metrics
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	m := &Metrics{
		serviceName: config.ServiceName,
		registry:    registry,
	}

	// HTTP metrics
	m.HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"service", "method", "path", "status"},
	)

	m.HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method", "path"},
	)

	m.HTTPRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   config.Namespace,
			Name:        "http_requests_in_flight",
			Help:        "Number of HTTP requests currently being processed",
			ConstLabels: prometheus.Labels{"service": config.ServiceName},
		},
	)

	// Kafka metrics
	m.KafkaEventsPublished = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "kafka_events_published_total",
			Help:      "Total number of Kafka events published",
		},
		[]string{"service", "topic", "event_type", "status"},
	)

	m.KafkaEventsConsumed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "kafka_events_consumed_total",
			Help:      "Total number of Kafka events consumed",
		},
		[]string{"service", "topic", "event_type", "status"},
	)

	m.KafkaPublishDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Name:      "kafka_publish_duration_seconds",
			Help:      "Kafka publish duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"service", "topic"},
	)

	m.KafkaConsumeLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Name:      "kafka_consumer_lag",
			Help:      "Kafka consumer lag (messages behind)",
		},
		[]string{"service", "topic", "partition"},
	)

	// MongoDB metrics
	m.MongoDBOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "mongodb_operations_total",
			Help:      "Total number of MongoDB operations",
		},
		[]string{"service", "collection", "operation", "status"},
	)

	m.MongoDBOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Name:      "mongodb_operation_duration_seconds",
			Help:      "MongoDB operation duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		},
		[]string{"service", "collection", "operation"},
	)

	m.MongoDBConnectionsOpen = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   config.Namespace,
			Name:        "mongodb_connections_open",
			Help:        "Number of open MongoDB connections",
			ConstLabels: prometheus.Labels{"service": config.ServiceName},
		},
	)

	// Temporal workflow metrics
	m.WorkflowsStarted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "temporal_workflows_started_total",
			Help:      "Total number of Temporal workflows started",
		},
		[]string{"service", "workflow_type"},
	)

	m.WorkflowsCompleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "temporal_workflows_completed_total",
			Help:      "Total number of Temporal workflows completed",
		},
		[]string{"service", "workflow_type", "status"},
	)

	m.WorkflowDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Name:      "temporal_workflow_duration_seconds",
			Help:      "Temporal workflow duration in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"service", "workflow_type"},
	)

	// Temporal activity metrics
	m.ActivitiesStarted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "temporal_activities_started_total",
			Help:      "Total number of Temporal activities started",
		},
		[]string{"service", "activity_type"},
	)

	m.ActivitiesCompleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "temporal_activities_completed_total",
			Help:      "Total number of Temporal activities completed",
		},
		[]string{"service", "activity_type", "status"},
	)

	m.ActivityDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Name:      "temporal_activity_duration_seconds",
			Help:      "Temporal activity duration in seconds",
			Buckets:   []float64{.1, .5, 1, 5, 10, 30, 60, 300},
		},
		[]string{"service", "activity_type"},
	)

	// Business metrics
	m.OrdersCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "orders_created_total",
			Help:      "Total number of orders created",
		},
		[]string{"service", "priority"},
	)

	m.OrdersProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "orders_processed_total",
			Help:      "Total number of orders processed",
		},
		[]string{"service", "status"},
	)

	m.ItemsPicked = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "items_picked_total",
			Help:      "Total number of items picked",
		},
		[]string{"service", "zone"},
	)

	m.PackagesShipped = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "packages_shipped_total",
			Help:      "Total number of packages shipped",
		},
		[]string{"service", "carrier"},
	)

	// Circuit breaker metrics
	m.CircuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Name:      "circuit_breaker_state",
			Help:      "Circuit breaker state (0=closed, 1=half-open, 2=open)",
		},
		[]string{"service", "name"},
	)

	m.CircuitBreakerTrips = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "circuit_breaker_trips_total",
			Help:      "Total number of circuit breaker trips",
		},
		[]string{"service", "name"},
	)

	// Register all metrics
	registry.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestsInFlight,
		m.KafkaEventsPublished,
		m.KafkaEventsConsumed,
		m.KafkaPublishDuration,
		m.KafkaConsumeLag,
		m.MongoDBOperations,
		m.MongoDBOperationDuration,
		m.MongoDBConnectionsOpen,
		m.WorkflowsStarted,
		m.WorkflowsCompleted,
		m.WorkflowDuration,
		m.ActivitiesStarted,
		m.ActivitiesCompleted,
		m.ActivityDuration,
		m.OrdersCreated,
		m.OrdersProcessed,
		m.ItemsPicked,
		m.PackagesShipped,
		m.CircuitBreakerState,
		m.CircuitBreakerTrips,
	)

	return m
}

// Handler returns an HTTP handler for metrics endpoint
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// Registry returns the prometheus registry
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	statusStr := strconv.Itoa(status)
	m.HTTPRequestsTotal.WithLabelValues(m.serviceName, method, path, statusStr).Inc()
	m.HTTPRequestDuration.WithLabelValues(m.serviceName, method, path).Observe(duration.Seconds())
}

// RecordKafkaPublish records a Kafka publish event
func (m *Metrics) RecordKafkaPublish(topic, eventType string, success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "error"
	}
	m.KafkaEventsPublished.WithLabelValues(m.serviceName, topic, eventType, status).Inc()
	m.KafkaPublishDuration.WithLabelValues(m.serviceName, topic).Observe(duration.Seconds())
}

// RecordKafkaConsume records a Kafka consume event
func (m *Metrics) RecordKafkaConsume(topic, eventType string, success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	m.KafkaEventsConsumed.WithLabelValues(m.serviceName, topic, eventType, status).Inc()
}

// SetKafkaConsumerLag sets the Kafka consumer lag
func (m *Metrics) SetKafkaConsumerLag(topic string, partition int, lag int64) {
	m.KafkaConsumeLag.WithLabelValues(m.serviceName, topic, strconv.Itoa(partition)).Set(float64(lag))
}

// RecordMongoDBOperation records a MongoDB operation
func (m *Metrics) RecordMongoDBOperation(collection, operation string, success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "error"
	}
	m.MongoDBOperations.WithLabelValues(m.serviceName, collection, operation, status).Inc()
	m.MongoDBOperationDuration.WithLabelValues(m.serviceName, collection, operation).Observe(duration.Seconds())
}

// SetMongoDBConnections sets the number of open MongoDB connections
func (m *Metrics) SetMongoDBConnections(count int) {
	m.MongoDBConnectionsOpen.Set(float64(count))
}

// RecordWorkflowStarted records a workflow start
func (m *Metrics) RecordWorkflowStarted(workflowType string) {
	m.WorkflowsStarted.WithLabelValues(m.serviceName, workflowType).Inc()
}

// RecordWorkflowCompleted records a workflow completion
func (m *Metrics) RecordWorkflowCompleted(workflowType string, success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "error"
	}
	m.WorkflowsCompleted.WithLabelValues(m.serviceName, workflowType, status).Inc()
	m.WorkflowDuration.WithLabelValues(m.serviceName, workflowType).Observe(duration.Seconds())
}

// RecordActivityStarted records an activity start
func (m *Metrics) RecordActivityStarted(activityType string) {
	m.ActivitiesStarted.WithLabelValues(m.serviceName, activityType).Inc()
}

// RecordActivityCompleted records an activity completion
func (m *Metrics) RecordActivityCompleted(activityType string, success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "error"
	}
	m.ActivitiesCompleted.WithLabelValues(m.serviceName, activityType, status).Inc()
	m.ActivityDuration.WithLabelValues(m.serviceName, activityType).Observe(duration.Seconds())
}

// RecordOrderCreated records an order creation
func (m *Metrics) RecordOrderCreated(priority string) {
	m.OrdersCreated.WithLabelValues(m.serviceName, priority).Inc()
}

// RecordOrderProcessed records an order processing
func (m *Metrics) RecordOrderProcessed(status string) {
	m.OrdersProcessed.WithLabelValues(m.serviceName, status).Inc()
}

// RecordItemPicked records an item pick
func (m *Metrics) RecordItemPicked(zone string, count int) {
	m.ItemsPicked.WithLabelValues(m.serviceName, zone).Add(float64(count))
}

// RecordPackageShipped records a package shipment
func (m *Metrics) RecordPackageShipped(carrier string) {
	m.PackagesShipped.WithLabelValues(m.serviceName, carrier).Inc()
}

// SetCircuitBreakerState sets the circuit breaker state
func (m *Metrics) SetCircuitBreakerState(name string, state int) {
	m.CircuitBreakerState.WithLabelValues(m.serviceName, name).Set(float64(state))
}

// RecordCircuitBreakerTrip records a circuit breaker trip
func (m *Metrics) RecordCircuitBreakerTrip(name string) {
	m.CircuitBreakerTrips.WithLabelValues(m.serviceName, name).Inc()
}

// IncrementHTTPRequestsInFlight increments in-flight requests
func (m *Metrics) IncrementHTTPRequestsInFlight() {
	m.HTTPRequestsInFlight.Inc()
}

// DecrementHTTPRequestsInFlight decrements in-flight requests
func (m *Metrics) DecrementHTTPRequestsInFlight() {
	m.HTTPRequestsInFlight.Dec()
}
