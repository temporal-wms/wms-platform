package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Resolution types for dead letter entries
const (
	ResolutionManualRetry = "manual_retry"
	ResolutionCancelled   = "cancelled"
	ResolutionEscalated   = "escalated"
)

// DeadLetterEntry represents an order that has been moved to the dead letter queue
// after exhausting all retry attempts
type DeadLetterEntry struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID            string             `bson:"orderId" json:"orderId"`
	CustomerID         string             `bson:"customerId" json:"customerId"`
	OriginalWorkflowID string             `bson:"originalWorkflowId" json:"originalWorkflowId"`
	FinalFailureStatus string             `bson:"finalFailureStatus" json:"finalFailureStatus"`
	FinalFailureReason string             `bson:"finalFailureReason" json:"finalFailureReason"`
	TotalRetryAttempts int                `bson:"totalRetryAttempts" json:"totalRetryAttempts"`
	RetryHistory       []RetryAttempt     `bson:"retryHistory" json:"retryHistory"`
	OrderSnapshot      OrderSnapshot      `bson:"orderSnapshot" json:"orderSnapshot"`
	MovedToQueueAt     time.Time          `bson:"movedToQueueAt" json:"movedToQueueAt"`
	Resolution         string             `bson:"resolution,omitempty" json:"resolution,omitempty"`
	ResolutionNotes    string             `bson:"resolutionNotes,omitempty" json:"resolutionNotes,omitempty"`
	ResolvedAt         *time.Time         `bson:"resolvedAt,omitempty" json:"resolvedAt,omitempty"`
	ResolvedBy         string             `bson:"resolvedBy,omitempty" json:"resolvedBy,omitempty"`
	CreatedAt          time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt          time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// RetryAttempt records details of each retry attempt
type RetryAttempt struct {
	AttemptNumber int       `bson:"attemptNumber" json:"attemptNumber"`
	WorkflowID    string    `bson:"workflowId" json:"workflowId"`
	RunID         string    `bson:"runId" json:"runId"`
	FailedAt      time.Time `bson:"failedAt" json:"failedAt"`
	FailureStatus string    `bson:"failureStatus" json:"failureStatus"`
	FailureReason string    `bson:"failureReason" json:"failureReason"`
}

// OrderSnapshot captures relevant order info at the time of DLQ entry
type OrderSnapshot struct {
	Priority           string    `bson:"priority" json:"priority"`
	PromisedDeliveryAt time.Time `bson:"promisedDeliveryAt" json:"promisedDeliveryAt"`
	ItemCount          int       `bson:"itemCount" json:"itemCount"`
	TotalWeight        float64   `bson:"totalWeight" json:"totalWeight"`
	ShippingCity       string    `bson:"shippingCity" json:"shippingCity"`
	ShippingState      string    `bson:"shippingState" json:"shippingState"`
}

// NewDeadLetterEntry creates a new dead letter entry from retry metadata and order info
func NewDeadLetterEntry(
	orderID string,
	customerID string,
	originalWorkflowID string,
	failureStatus string,
	failureReason string,
	totalRetries int,
	retryHistory []RetryAttempt,
	snapshot OrderSnapshot,
) *DeadLetterEntry {
	now := time.Now().UTC()
	return &DeadLetterEntry{
		ID:                 primitive.NewObjectID(),
		OrderID:            orderID,
		CustomerID:         customerID,
		OriginalWorkflowID: originalWorkflowID,
		FinalFailureStatus: failureStatus,
		FinalFailureReason: failureReason,
		TotalRetryAttempts: totalRetries,
		RetryHistory:       retryHistory,
		OrderSnapshot:      snapshot,
		MovedToQueueAt:     now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// Resolve marks the dead letter entry as resolved with the given resolution type
func (d *DeadLetterEntry) Resolve(resolution string, notes string, resolvedBy string) error {
	if d.Resolution != "" {
		return errors.New("dead letter entry already resolved")
	}

	if resolution != ResolutionManualRetry && resolution != ResolutionCancelled && resolution != ResolutionEscalated {
		return errors.New("invalid resolution type")
	}

	now := time.Now().UTC()
	d.Resolution = resolution
	d.ResolutionNotes = notes
	d.ResolvedBy = resolvedBy
	d.ResolvedAt = &now
	d.UpdatedAt = now

	return nil
}

// IsResolved returns true if the entry has been resolved
func (d *DeadLetterEntry) IsResolved() bool {
	return d.Resolution != ""
}

// AgeInHours returns how many hours the entry has been in the queue
func (d *DeadLetterEntry) AgeInHours() float64 {
	return time.Since(d.MovedToQueueAt).Hours()
}
