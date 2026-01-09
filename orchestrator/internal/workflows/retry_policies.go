package workflows

import (
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// RetryPolicyType defines different retry policy configurations
type RetryPolicyType int

const (
	// StandardRetry for normal operations (3 attempts, 1s-1m backoff)
	StandardRetry RetryPolicyType = iota
	// AggressiveRetry for critical operations (5 attempts, 500ms-30s backoff)
	AggressiveRetry
	// ConservativeRetry for expensive operations (2 attempts, 2s-2m backoff)
	ConservativeRetry
	// NoRetry for idempotent operations that should not retry
	NoRetry
)

// GetRetryPolicy returns a configured retry policy based on type
func GetRetryPolicy(policyType RetryPolicyType) *temporal.RetryPolicy {
	switch policyType {
	case AggressiveRetry:
		return &temporal.RetryPolicy{
			InitialInterval:    500 * time.Millisecond,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
			NonRetryableErrorTypes: []string{
				"ValidationError",
				"NotFoundError",
				"ConflictError",
			},
		}

	case ConservativeRetry:
		return &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    2 * time.Minute,
			MaximumAttempts:    2,
			NonRetryableErrorTypes: []string{
				"ValidationError",
				"NotFoundError",
			},
		}

	case NoRetry:
		return &temporal.RetryPolicy{
			MaximumAttempts: 1,
		}

	case StandardRetry:
		fallthrough
	default:
		return &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
			NonRetryableErrorTypes: []string{
				"ValidationError",
				"NotFoundError",
			},
		}
	}
}

// ActivityOptionsConfig defines configuration for activity options
type ActivityOptionsConfig struct {
	StartToCloseTimeout time.Duration
	RetryPolicy         RetryPolicyType
	HeartbeatTimeout    time.Duration
}

// GetActivityOptions returns configured activity options
func GetActivityOptions(config ActivityOptionsConfig) workflow.ActivityOptions {
	if config.StartToCloseTimeout == 0 {
		config.StartToCloseTimeout = 5 * time.Minute
	}

	opts := workflow.ActivityOptions{
		StartToCloseTimeout: config.StartToCloseTimeout,
		RetryPolicy:         GetRetryPolicy(config.RetryPolicy),
	}

	if config.HeartbeatTimeout > 0 {
		opts.HeartbeatTimeout = config.HeartbeatTimeout
	}

	return opts
}

// GetStandardActivityOptions returns standard activity options (most common use case)
func GetStandardActivityOptions() workflow.ActivityOptions {
	return GetActivityOptions(ActivityOptionsConfig{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy:         StandardRetry,
	})
}

// GetCriticalActivityOptions returns activity options for critical operations
func GetCriticalActivityOptions() workflow.ActivityOptions {
	return GetActivityOptions(ActivityOptionsConfig{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy:         AggressiveRetry,
		HeartbeatTimeout:    30 * time.Second,
	})
}

// GetLongRunningActivityOptions returns activity options for long-running operations
func GetLongRunningActivityOptions() workflow.ActivityOptions {
	return GetActivityOptions(ActivityOptionsConfig{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy:         ConservativeRetry,
		HeartbeatTimeout:    time.Minute,
	})
}

// ChildWorkflowOptionsConfig defines configuration for child workflow options
type ChildWorkflowOptionsConfig struct {
	WorkflowExecutionTimeout time.Duration
	RetryPolicy              RetryPolicyType
	ParentClosePolicy        enums.ParentClosePolicy
}

// GetChildWorkflowOptions returns configured child workflow options
func GetChildWorkflowOptions(config ChildWorkflowOptionsConfig) workflow.ChildWorkflowOptions {
	if config.WorkflowExecutionTimeout == 0 {
		config.WorkflowExecutionTimeout = 24 * time.Hour
	}

	if config.ParentClosePolicy == 0 {
		config.ParentClosePolicy = enums.PARENT_CLOSE_POLICY_TERMINATE
	}

	return workflow.ChildWorkflowOptions{
		WorkflowExecutionTimeout: config.WorkflowExecutionTimeout,
		RetryPolicy:              GetRetryPolicy(config.RetryPolicy),
		ParentClosePolicy:        config.ParentClosePolicy,
	}
}

// GetStandardChildWorkflowOptions returns standard child workflow options
func GetStandardChildWorkflowOptions() workflow.ChildWorkflowOptions {
	return GetChildWorkflowOptions(ChildWorkflowOptionsConfig{
		WorkflowExecutionTimeout: 24 * time.Hour,
		RetryPolicy:              StandardRetry,
		ParentClosePolicy:        enums.PARENT_CLOSE_POLICY_TERMINATE,
	})
}

// TimeoutConfig defines timeout configurations based on order priority
type TimeoutConfig struct {
	WaveAssignment time.Duration
	Picking        time.Duration
	Consolidation  time.Duration
	Packing        time.Duration
	Shipping       time.Duration
}

// GetTimeoutsByPriority returns timeout configurations based on order priority
func GetTimeoutsByPriority(priority string) TimeoutConfig {
	switch priority {
	case "same_day":
		return TimeoutConfig{
			WaveAssignment: 30 * time.Minute,
			Picking:        15 * time.Minute,
			Consolidation:  10 * time.Minute,
			Packing:        15 * time.Minute,
			Shipping:       15 * time.Minute,
		}
	case "next_day":
		return TimeoutConfig{
			WaveAssignment: 2 * time.Hour,
			Picking:        30 * time.Minute,
			Consolidation:  20 * time.Minute,
			Packing:        30 * time.Minute,
			Shipping:       30 * time.Minute,
		}
	case "standard":
		fallthrough
	default:
		return TimeoutConfig{
			WaveAssignment: 4 * time.Hour,
			Picking:        1 * time.Hour,
			Consolidation:  30 * time.Minute,
			Packing:        1 * time.Hour,
			Shipping:       1 * time.Hour,
		}
	}
}

// ErrorClassification helps categorize errors for retry decisions
type ErrorClassification struct {
	IsTransient bool
	IsRetryable bool
	Category    string
}

// ClassifyError categorizes an error to determine retry behavior
func ClassifyError(err error) ErrorClassification {
	if err == nil {
		return ErrorClassification{
			IsTransient: false,
			IsRetryable: false,
			Category:    "none",
		}
	}

	errMsg := err.Error()

	// Validation errors - never retry
	if contains(errMsg, "validation") || contains(errMsg, "invalid") {
		return ErrorClassification{
			IsTransient: false,
			IsRetryable: false,
			Category:    "validation",
		}
	}

	// Not found errors - never retry
	if contains(errMsg, "not found") {
		return ErrorClassification{
			IsTransient: false,
			IsRetryable: false,
			Category:    "not_found",
		}
	}

	// Conflict errors - never retry
	if contains(errMsg, "conflict") || contains(errMsg, "already exists") {
		return ErrorClassification{
			IsTransient: false,
			IsRetryable: false,
			Category:    "conflict",
		}
	}

	// Timeout errors - retry with backoff
	if contains(errMsg, "timeout") || contains(errMsg, "deadline exceeded") {
		return ErrorClassification{
			IsTransient: true,
			IsRetryable: true,
			Category:    "timeout",
		}
	}

	// Connection errors - retry aggressively
	if contains(errMsg, "connection") || contains(errMsg, "unavailable") {
		return ErrorClassification{
			IsTransient: true,
			IsRetryable: true,
			Category:    "connection",
		}
	}

	// Circuit breaker errors - retry conservatively
	if contains(errMsg, "circuit breaker") {
		return ErrorClassification{
			IsTransient: true,
			IsRetryable: true,
			Category:    "circuit_breaker",
		}
	}

	// Default: assume transient and retryable
	return ErrorClassification{
		IsTransient: true,
		IsRetryable: true,
		Category:    "unknown",
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
