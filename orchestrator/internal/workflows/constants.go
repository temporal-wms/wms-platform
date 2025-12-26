package workflows

import "time"

// Activity and workflow timeout defaults
const (
	DefaultActivityTimeout      time.Duration = 5 * time.Minute
	DefaultChildWorkflowTimeout time.Duration = 24 * time.Hour
)

// Retry policy defaults
const (
	DefaultRetryInitialInterval    time.Duration = time.Second
	DefaultRetryMaxInterval        time.Duration = time.Minute
	DefaultRetryBackoffCoefficient float64       = 2.0
	DefaultMaxRetryAttempts        int32         = 3
)

// Wave assignment timeouts by priority
const (
	WaveTimeoutSameDay time.Duration = 30 * time.Minute
	WaveTimeoutNextDay time.Duration = 2 * time.Hour
	WaveTimeoutDefault time.Duration = 4 * time.Hour
)
