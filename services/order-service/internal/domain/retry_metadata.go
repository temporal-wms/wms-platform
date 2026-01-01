package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DefaultMaxRetries is the default maximum number of retry attempts
const DefaultMaxRetries = 5

// RetryMetadata tracks retry attempts for failed order workflows
type RetryMetadata struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID        string             `bson:"orderId" json:"orderId"`
	RetryCount     int                `bson:"retryCount" json:"retryCount"`
	MaxRetries     int                `bson:"maxRetries" json:"maxRetries"`
	LastFailureAt  time.Time          `bson:"lastFailureAt" json:"lastFailureAt"`
	FailureStatus  string             `bson:"failureStatus" json:"failureStatus"` // wave_timeout, pick_timeout
	FailureReason  string             `bson:"failureReason" json:"failureReason"`
	LastWorkflowID string             `bson:"lastWorkflowId" json:"lastWorkflowId"`
	LastRunID      string             `bson:"lastRunId" json:"lastRunId"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// NewRetryMetadata creates a new RetryMetadata for a failed order
func NewRetryMetadata(orderID string, failureStatus string, failureReason string, workflowID string, runID string) *RetryMetadata {
	now := time.Now().UTC()
	return &RetryMetadata{
		ID:             primitive.NewObjectID(),
		OrderID:        orderID,
		RetryCount:     0,
		MaxRetries:     DefaultMaxRetries,
		LastFailureAt:  now,
		FailureStatus:  failureStatus,
		FailureReason:  failureReason,
		LastWorkflowID: workflowID,
		LastRunID:      runID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IncrementRetry increments the retry count and updates failure info
func (r *RetryMetadata) IncrementRetry(failureStatus string, failureReason string, workflowID string, runID string) {
	r.RetryCount++
	r.LastFailureAt = time.Now().UTC()
	r.FailureStatus = failureStatus
	r.FailureReason = failureReason
	r.LastWorkflowID = workflowID
	r.LastRunID = runID
	r.UpdatedAt = time.Now().UTC()
}

// CanRetry returns true if more retry attempts are allowed
func (r *RetryMetadata) CanRetry() bool {
	return r.RetryCount < r.MaxRetries
}

// ShouldMoveToDeadLetter returns true if max retries have been exhausted
func (r *RetryMetadata) ShouldMoveToDeadLetter() bool {
	return r.RetryCount >= r.MaxRetries
}

// RemainingRetries returns the number of retry attempts remaining
func (r *RetryMetadata) RemainingRetries() int {
	remaining := r.MaxRetries - r.RetryCount
	if remaining < 0 {
		return 0
	}
	return remaining
}
