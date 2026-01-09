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

// Planning workflow configuration
const (
	// PlanningWorkflowTimeout is the maximum duration for the planning workflow
	PlanningWorkflowTimeout time.Duration = 5 * time.Hour

	// PlanningActivityTimeout is the timeout for planning-specific activities
	PlanningActivityTimeout time.Duration = 2 * time.Minute
)

// SLAM (Scan, Label, Apply, Manifest) configuration
const (
	// WeightToleranceThreshold is the maximum allowed weight variance percentage
	WeightToleranceThreshold float64 = 5.0

	// SLAMActivityTimeout is the timeout for SLAM activities
	SLAMActivityTimeout time.Duration = 2 * time.Minute
)

// ReprocessableStatuses are the failure statuses that can be retried
// Only transient/timeout failures are eligible for automatic retry
var ReprocessableStatuses = []string{
	"wave_timeout",
	"pick_timeout",
}

// WES (Warehouse Execution System) configuration
const (
	// WESExecutionWorkflowTimeout is the maximum duration for WES execution
	WESExecutionWorkflowTimeout time.Duration = 4 * time.Hour

	// WESTaskQueue is the Temporal task queue for WES workers
	WESTaskQueue = "wes-execution-queue"
)
