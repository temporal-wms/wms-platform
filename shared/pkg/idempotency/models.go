package idempotency

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// IdempotencyKey represents a stored idempotency key for REST APIs
// It stores the request fingerprint and response to enable safe retries
type IdempotencyKey struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty"`
	Key                string             `bson:"key"`                // The idempotency key from header
	UserID             string             `bson:"userId,omitempty"`   // Optional: scope to user
	ServiceID          string             `bson:"serviceId"`          // Service name (e.g., "order-service")
	RequestPath        string             `bson:"requestPath"`        // API endpoint path
	RequestMethod      string             `bson:"requestMethod"`      // HTTP method (POST, PUT, PATCH, DELETE)
	RequestFingerprint string             `bson:"requestFingerprint"` // SHA256 hash of request body

	// Locking mechanism to prevent concurrent processing
	LockedAt *time.Time `bson:"lockedAt,omitempty"`

	// Atomic phases support for complex multi-step operations
	RecoveryPoint string `bson:"recoveryPoint,omitempty"`

	// Response storage for caching
	ResponseCode    int               `bson:"responseCode,omitempty"`
	ResponseBody    []byte            `bson:"responseBody,omitempty"`
	ResponseHeaders map[string]string `bson:"responseHeaders,omitempty"`

	// Metadata
	CreatedAt   time.Time  `bson:"createdAt"`
	CompletedAt *time.Time `bson:"completedAt,omitempty"`
	ExpiresAt   time.Time  `bson:"expiresAt"` // For TTL index (24 hours default)
}

// IsCompleted returns true if the request has been completed
func (ik *IdempotencyKey) IsCompleted() bool {
	return ik.CompletedAt != nil
}

// IsLocked returns true if the request is currently being processed
func (ik *IdempotencyKey) IsLocked() bool {
	return ik.LockedAt != nil && ik.CompletedAt == nil
}

// ProcessedMessage represents a deduplicated Kafka message
// Used to track which CloudEvents have been processed to enable exactly-once semantics
type ProcessedMessage struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	MessageID     string             `bson:"messageId"`     // CloudEvent.ID
	Topic         string             `bson:"topic"`         // Kafka topic
	EventType     string             `bson:"eventType"`     // CloudEvent.Type
	ConsumerGroup string             `bson:"consumerGroup"` // Kafka consumer group
	ServiceID     string             `bson:"serviceId"`     // Service name

	// Processing metadata
	ProcessedAt time.Time `bson:"processedAt"`
	ExpiresAt   time.Time `bson:"expiresAt"` // For TTL index (24 hours default)

	// Optional: correlation data for debugging
	CorrelationID string `bson:"correlationId,omitempty"`
	WorkflowID    string `bson:"workflowId,omitempty"`
}

// PhaseState represents state for atomic phases
// Used to track progress through multi-step operations that can be recovered
type PhaseState struct {
	Phase       string                 `bson:"phase"`
	Data        map[string]interface{} `bson:"data"`
	CompletedAt time.Time              `bson:"completedAt"`
}
