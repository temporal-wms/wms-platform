package resilience

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sony/gobreaker"
)

// Common errors
var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// CircuitBreakerConfig holds configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name                   string
	MaxRequests            uint32        // Maximum number of requests allowed in half-open state
	Interval               time.Duration // Time interval to clear failure count (0 = never clear)
	Timeout                time.Duration // How long to wait before transitioning from open to half-open
	FailureThreshold       uint32        // Number of failures to trip the circuit
	SuccessThreshold       uint32        // Number of successes needed in half-open to close circuit
	FailureRatioThreshold  float64       // Failure ratio to trip (0.5 = 50%)
	MinRequestsToTrip      uint32        // Minimum requests before evaluating ratio
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig(name string) *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		Name:                  name,
		MaxRequests:           DefaultMaxRequests,
		Interval:              DefaultInterval,
		Timeout:               DefaultTimeout,
		FailureThreshold:      DefaultFailureThreshold,
		SuccessThreshold:      DefaultSuccessThreshold,
		FailureRatioThreshold: DefaultFailureRatioThreshold,
		MinRequestsToTrip:     DefaultMinRequestsToTrip,
	}
}

// CircuitBreaker wraps gobreaker with logging and metrics
type CircuitBreaker struct {
	cb     *gobreaker.CircuitBreaker
	name   string
	logger *slog.Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig, logger *slog.Logger) *CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        config.Name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip if failure threshold exceeded
			if counts.ConsecutiveFailures >= config.FailureThreshold {
				return true
			}

			// Trip if failure ratio exceeded (with minimum requests)
			if counts.Requests >= config.MinRequestsToTrip {
				failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
				return failureRatio >= config.FailureRatioThreshold
			}

			return false
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Warn("Circuit breaker state changed",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	}

	return &CircuitBreaker{
		cb:     gobreaker.NewCircuitBreaker(settings),
		name:   config.Name,
		logger: logger,
	}
}

// Execute runs a function through the circuit breaker
func (c *CircuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return fn()
	})

	if err == gobreaker.ErrOpenState {
		c.logger.Warn("Circuit breaker is open", "name", c.name)
		return nil, fmt.Errorf("service unavailable: circuit breaker open for %s", c.name)
	}

	if err == gobreaker.ErrTooManyRequests {
		c.logger.Warn("Circuit breaker: too many requests", "name", c.name)
		return nil, fmt.Errorf("service unavailable: too many requests for %s", c.name)
	}

	return result, err
}

// State returns the current state of the circuit breaker
func (c *CircuitBreaker) State() gobreaker.State {
	return c.cb.State()
}

// Name returns the circuit breaker name
func (c *CircuitBreaker) Name() string {
	return c.name
}

// Counts returns the current counts
func (c *CircuitBreaker) Counts() gobreaker.Counts {
	return c.cb.Counts()
}

// CircuitBreakerRegistry manages multiple circuit breakers
type CircuitBreakerRegistry struct {
	breakers map[string]*CircuitBreaker
	logger   *slog.Logger
}

// NewCircuitBreakerRegistry creates a new registry
func NewCircuitBreakerRegistry(logger *slog.Logger) *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logger,
	}
}

// Get returns a circuit breaker by name, creating it if it doesn't exist
func (r *CircuitBreakerRegistry) Get(name string) *CircuitBreaker {
	if cb, exists := r.breakers[name]; exists {
		return cb
	}

	config := DefaultCircuitBreakerConfig(name)
	cb := NewCircuitBreaker(config, r.logger)
	r.breakers[name] = cb
	return cb
}

// GetWithConfig returns a circuit breaker with custom config
func (r *CircuitBreakerRegistry) GetWithConfig(config *CircuitBreakerConfig) *CircuitBreaker {
	if cb, exists := r.breakers[config.Name]; exists {
		return cb
	}

	cb := NewCircuitBreaker(config, r.logger)
	r.breakers[config.Name] = cb
	return cb
}

// Status returns the status of all circuit breakers
func (r *CircuitBreakerRegistry) Status() map[string]CircuitBreakerStatus {
	status := make(map[string]CircuitBreakerStatus)
	for name, cb := range r.breakers {
		counts := cb.Counts()
		status[name] = CircuitBreakerStatus{
			Name:                  name,
			State:                 cb.State().String(),
			Requests:              counts.Requests,
			TotalSuccesses:        counts.TotalSuccesses,
			TotalFailures:         counts.TotalFailures,
			ConsecutiveSuccesses:  counts.ConsecutiveSuccesses,
			ConsecutiveFailures:   counts.ConsecutiveFailures,
		}
	}
	return status
}

// CircuitBreakerStatus holds status information for a circuit breaker
type CircuitBreakerStatus struct {
	Name                 string `json:"name"`
	State                string `json:"state"`
	Requests             uint32 `json:"requests"`
	TotalSuccesses       uint32 `json:"totalSuccesses"`
	TotalFailures        uint32 `json:"totalFailures"`
	ConsecutiveSuccesses uint32 `json:"consecutiveSuccesses"`
	ConsecutiveFailures  uint32 `json:"consecutiveFailures"`
}

// Retry configuration
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors func(error) bool
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   DefaultRetryMaxAttempts,
		InitialDelay:  DefaultRetryInitialDelay,
		MaxDelay:      DefaultRetryMaxDelay,
		BackoffFactor: DefaultRetryBackoffFactor,
		RetryableErrors: func(err error) bool {
			// By default, don't retry
			return false
		},
	}
}

// Retry executes a function with retry logic
func Retry(ctx context.Context, config *RetryConfig, fn func() error) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if config.RetryableErrors != nil && !config.RetryableErrors(err) {
			return err
		}

		// Don't sleep after last attempt
		if attempt < config.MaxAttempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			// Increase delay with exponential backoff
			delay = time.Duration(float64(delay) * config.BackoffFactor)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", config.MaxAttempts, lastErr)
}

// RetryWithResult executes a function with retry logic and returns a result
func RetryWithResult[T any](ctx context.Context, config *RetryConfig, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if config.RetryableErrors != nil && !config.RetryableErrors(err) {
			return zero, err
		}

		// Don't sleep after last attempt
		if attempt < config.MaxAttempts-1 {
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(delay):
			}

			// Increase delay with exponential backoff
			delay = time.Duration(float64(delay) * config.BackoffFactor)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}

	return zero, fmt.Errorf("max retries (%d) exceeded: %w", config.MaxAttempts, lastErr)
}
