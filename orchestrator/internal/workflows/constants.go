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

// Reprocessing configuration
const (
	// MaxReprocessingRetries is the maximum number of retry attempts before moving to DLQ
	MaxReprocessingRetries = 5

	// ReprocessingBatchInterval is how often the reprocessing batch job runs
	ReprocessingBatchInterval time.Duration = 5 * time.Minute

	// ReprocessingBatchSize is the maximum number of orders to process per batch
	ReprocessingBatchSize = 100

	// ReprocessingScheduleID is the ID for the Temporal schedule
	ReprocessingScheduleID = "order-reprocessing-schedule"
)

// ReprocessableStatuses are the failure statuses that can be retried
// Only transient/timeout failures are eligible for automatic retry
var ReprocessableStatuses = []string{
	"wave_timeout",
	"pick_timeout",
}
