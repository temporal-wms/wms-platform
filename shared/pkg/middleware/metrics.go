package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wms-platform/shared/pkg/metrics"
)

// MetricsMiddleware creates middleware that records HTTP metrics
func MetricsMiddleware(m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip metrics endpoint to avoid recursion
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		// Track in-flight requests
		m.IncrementHTTPRequestsInFlight()
		defer m.DecrementHTTPRequestsInFlight()

		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics after request completes
		duration := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.FullPath() // Use route pattern, not actual path

		// If no route matched, use the raw path
		if path == "" {
			path = c.Request.URL.Path
		}

		// Record HTTP request metrics
		m.RecordHTTPRequest(method, path, status, duration)
	}
}

// MetricsEndpoint returns a handler for the /metrics endpoint
func MetricsEndpoint(m *metrics.Metrics) gin.HandlerFunc {
	handler := m.Handler()
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

// BusinessMetrics provides helpers for recording business-specific metrics
type BusinessMetrics struct {
	metrics *metrics.Metrics
}

// NewBusinessMetrics creates a new BusinessMetrics helper
func NewBusinessMetrics(m *metrics.Metrics) *BusinessMetrics {
	return &BusinessMetrics{metrics: m}
}

// RecordOrderCreated records an order creation event
func (b *BusinessMetrics) RecordOrderCreated(priority string) {
	b.metrics.RecordOrderCreated(priority)
}

// RecordOrderProcessed records an order processing event
func (b *BusinessMetrics) RecordOrderProcessed(status string) {
	b.metrics.RecordOrderProcessed(status)
}

// RecordItemsPicked records items picked
func (b *BusinessMetrics) RecordItemsPicked(zone string, count int) {
	b.metrics.RecordItemPicked(zone, count)
}

// RecordPackageShipped records a package shipment
func (b *BusinessMetrics) RecordPackageShipped(carrier string) {
	b.metrics.RecordPackageShipped(carrier)
}

// RecordCircuitBreakerState records circuit breaker state
func (b *BusinessMetrics) RecordCircuitBreakerState(name string, state int) {
	b.metrics.SetCircuitBreakerState(name, state)
}

// RecordCircuitBreakerTrip records a circuit breaker trip
func (b *BusinessMetrics) RecordCircuitBreakerTrip(name string) {
	b.metrics.RecordCircuitBreakerTrip(name)
}

// TemporalMetrics provides helpers for recording Temporal workflow metrics
type TemporalMetrics struct {
	metrics *metrics.Metrics
}

// NewTemporalMetrics creates a new TemporalMetrics helper
func NewTemporalMetrics(m *metrics.Metrics) *TemporalMetrics {
	return &TemporalMetrics{metrics: m}
}

// RecordWorkflowStarted records a workflow start
func (t *TemporalMetrics) RecordWorkflowStarted(workflowType string) {
	t.metrics.RecordWorkflowStarted(workflowType)
}

// RecordWorkflowCompleted records a workflow completion
func (t *TemporalMetrics) RecordWorkflowCompleted(workflowType string, success bool, duration time.Duration) {
	t.metrics.RecordWorkflowCompleted(workflowType, success, duration)
}

// RecordActivityStarted records an activity start
func (t *TemporalMetrics) RecordActivityStarted(activityType string) {
	t.metrics.RecordActivityStarted(activityType)
}

// RecordActivityCompleted records an activity completion
func (t *TemporalMetrics) RecordActivityCompleted(activityType string, success bool, duration time.Duration) {
	t.metrics.RecordActivityCompleted(activityType, success, duration)
}

// FailureMetrics provides helpers for recording failure-related metrics
type FailureMetrics struct {
	metrics *metrics.Metrics
}

// NewFailureMetrics creates a new FailureMetrics helper
func NewFailureMetrics(m *metrics.Metrics) *FailureMetrics {
	return &FailureMetrics{metrics: m}
}

// RecordOrderFailure records an order failure
func (f *FailureMetrics) RecordOrderFailure(failureType, priority string) {
	f.metrics.RecordOrderFailure(failureType, priority)
}

// RecordRetryAttempt records a retry attempt for an order
func (f *FailureMetrics) RecordRetryAttempt(failureType string, attemptNumber int) {
	f.metrics.RecordOrderRetryAttempt(failureType, attemptNumber)
}

// RecordRetrySuccess records a successful retry outcome
func (f *FailureMetrics) RecordRetrySuccess() {
	f.metrics.RecordOrderRetryOutcome("success")
}

// RecordRetryFailure records a failed retry outcome
func (f *FailureMetrics) RecordRetryFailure() {
	f.metrics.RecordOrderRetryOutcome("failure")
}

// RecordMovedToDLQ records an order moved to dead letter queue
func (f *FailureMetrics) RecordMovedToDLQ(failureType string) {
	f.metrics.RecordDLQEntry(failureType)
	f.metrics.RecordOrderRetryOutcome("dlq")
}

// RecordDLQResolution records a DLQ entry resolution with age
func (f *FailureMetrics) RecordDLQResolution(resolutionType string, ageHours float64) {
	f.metrics.RecordDLQResolution(resolutionType)
	f.metrics.RecordDLQAge(ageHours * 3600) // Convert hours to seconds
}

// UpdateDLQPendingStats updates the pending DLQ entries gauge by failure type
func (f *FailureMetrics) UpdateDLQPendingStats(statsByFailureType map[string]int) {
	for failureType, count := range statsByFailureType {
		f.metrics.SetDLQPending(failureType, count)
	}
}

// RecordWorkflowFailure records a workflow failure with stage information
func (f *FailureMetrics) RecordWorkflowFailure(workflowType, stage, failureType string) {
	f.metrics.RecordWorkflowFailure(workflowType, stage, failureType)
}

// RecordWorkflowRetry records a workflow retry
func (f *FailureMetrics) RecordWorkflowRetry(workflowType string) {
	f.metrics.RecordWorkflowRetry(workflowType)
}

// RecordReprocessingBatchResult records the results of a reprocessing batch
func (f *FailureMetrics) RecordReprocessingBatchResult(restarted, dlq, errors int) {
	if restarted > 0 {
		for i := 0; i < restarted; i++ {
			f.metrics.RecordReprocessingBatch("restarted")
		}
	}
	if dlq > 0 {
		for i := 0; i < dlq; i++ {
			f.metrics.RecordReprocessingBatch("dlq")
		}
	}
	if errors > 0 {
		for i := 0; i < errors; i++ {
			f.metrics.RecordReprocessingBatch("error")
		}
	}
}

// RecordRetryDuration records the duration between failure and retry
func (f *FailureMetrics) RecordRetryDuration(failureType string, duration time.Duration) {
	f.metrics.RecordRetryDuration(failureType, duration)
}

// RequestMetrics extracts metrics from a gin context for custom recording
type RequestMetrics struct {
	Method     string
	Path       string
	Status     int
	Duration   time.Duration
	ClientIP   string
	UserAgent  string
	RequestID  string
	StatusText string
}

// ExtractRequestMetrics extracts metrics from the current request
func ExtractRequestMetrics(c *gin.Context, duration time.Duration) *RequestMetrics {
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}

	requestID, _ := c.Get(ContextKeyRequestID)
	reqID, _ := requestID.(string)

	return &RequestMetrics{
		Method:     c.Request.Method,
		Path:       path,
		Status:     c.Writer.Status(),
		Duration:   duration,
		ClientIP:   c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		RequestID:  reqID,
		StatusText: statusText(c.Writer.Status()),
	}
}

func statusText(status int) string {
	switch {
	case status >= 500:
		return "server_error"
	case status >= 400:
		return "client_error"
	case status >= 300:
		return "redirect"
	case status >= 200:
		return "success"
	default:
		return "informational"
	}
}

// MetricsConfig holds configuration for metrics middleware
type MetricsConfig struct {
	// ServiceName is the name of the service
	ServiceName string

	// Namespace is the Prometheus namespace
	Namespace string

	// EnableGoMetrics enables Go runtime metrics
	EnableGoMetrics bool

	// EnableProcessMetrics enables process metrics
	EnableProcessMetrics bool

	// HistogramBuckets defines custom histogram buckets for request duration
	HistogramBuckets []float64

	// ExcludePaths lists paths to exclude from metrics
	ExcludePaths []string
}

// DefaultMetricsConfig returns a default metrics configuration
func DefaultMetricsConfig(serviceName string) *MetricsConfig {
	return &MetricsConfig{
		ServiceName:          serviceName,
		Namespace:            "wms",
		EnableGoMetrics:      true,
		EnableProcessMetrics: true,
		HistogramBuckets:     []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		ExcludePaths:         []string{"/metrics", "/health", "/ready"},
	}
}

// MetricsMiddlewareWithConfig creates metrics middleware with custom configuration
func MetricsMiddlewareWithConfig(m *metrics.Metrics, config *MetricsConfig) gin.HandlerFunc {
	excludeMap := make(map[string]bool)
	for _, path := range config.ExcludePaths {
		excludeMap[path] = true
	}

	return func(c *gin.Context) {
		// Skip excluded paths
		if excludeMap[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Track in-flight requests
		m.IncrementHTTPRequestsInFlight()
		defer m.DecrementHTTPRequestsInFlight()

		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics after request completes
		duration := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.FullPath()

		if path == "" {
			path = c.Request.URL.Path
		}

		m.RecordHTTPRequest(method, path, status, duration)
	}
}

// ResponseSizeMiddleware tracks response sizes
func ResponseSizeMiddleware(serviceName string, registry interface{}) gin.HandlerFunc {
	// This could be extended to track response sizes if needed
	return func(c *gin.Context) {
		c.Next()
		// Response size tracking would go here
	}
}

// LatencyPercentileTracker provides percentile tracking helpers
type LatencyPercentileTracker struct {
	serviceName string
}

// NewLatencyPercentileTracker creates a new latency tracker
func NewLatencyPercentileTracker(serviceName string) *LatencyPercentileTracker {
	return &LatencyPercentileTracker{serviceName: serviceName}
}

// FormatDuration formats a duration for logging
func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return strconv.FormatFloat(float64(d.Nanoseconds())/1000, 'f', 2, 64) + "µs"
	}
	if d < time.Second {
		return strconv.FormatFloat(float64(d.Nanoseconds())/1000000, 'f', 2, 64) + "ms"
	}
	return strconv.FormatFloat(d.Seconds(), 'f', 2, 64) + "s"
}
