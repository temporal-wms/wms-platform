package resilience

import "time"

// Circuit breaker default configuration values
const (
	DefaultMaxRequests           uint32        = 3
	DefaultInterval              time.Duration = 60 * time.Second
	DefaultTimeout               time.Duration = 30 * time.Second
	DefaultFailureThreshold      uint32        = 5
	DefaultSuccessThreshold      uint32        = 2
	DefaultFailureRatioThreshold float64       = 0.5
	DefaultMinRequestsToTrip     uint32        = 10
)

// Retry default configuration values
const (
	DefaultRetryMaxAttempts   int           = 3
	DefaultRetryInitialDelay  time.Duration = 100 * time.Millisecond
	DefaultRetryMaxDelay      time.Duration = 5 * time.Second
	DefaultRetryBackoffFactor float64       = 2.0
)
