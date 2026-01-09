package idempotency

import (
	"context"
	"time"
)

// KeyRepository manages idempotency keys for REST APIs
// Implementations must ensure thread-safety and atomic operations
type KeyRepository interface {
	// AcquireLock attempts to acquire a lock for the given idempotency key
	// Returns:
	//   - The existing or newly created IdempotencyKey
	//   - A boolean indicating if this is a new key (true) or existing (false)
	//   - An error if the operation fails
	//
	// This operation must be atomic to prevent race conditions.
	// Implementation should use upsert with optimistic locking.
	AcquireLock(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, bool, error)

	// ReleaseLock releases the lock on an idempotency key
	// This is typically called when a request fails and needs to be retried
	ReleaseLock(ctx context.Context, keyID string) error

	// StoreResponse stores the final response for a completed request
	// This marks the request as completed and caches the response
	StoreResponse(ctx context.Context, keyID string, responseCode int, responseBody []byte, headers map[string]string) error

	// UpdateRecoveryPoint updates the recovery point for atomic phases
	// Used to track progress through multi-step operations
	UpdateRecoveryPoint(ctx context.Context, keyID string, phase string) error

	// Get retrieves an idempotency key by its key string and service ID
	Get(ctx context.Context, key, serviceID string) (*IdempotencyKey, error)

	// GetByID retrieves an idempotency key by its ID
	GetByID(ctx context.Context, keyID string) (*IdempotencyKey, error)

	// Clean removes expired idempotency keys
	// Typically called by a background cleanup job
	// Returns the number of keys deleted
	Clean(ctx context.Context, before time.Time) (int64, error)

	// EnsureIndexes ensures that all required indexes are created
	// Should be called on service startup
	EnsureIndexes(ctx context.Context) error
}

// MessageRepository manages processed messages for Kafka consumers
// Implementations must ensure exactly-once message processing
type MessageRepository interface {
	// MarkProcessed marks a message as processed
	// Returns an error if the message has already been processed
	// Implementation should ensure this is an atomic operation
	MarkProcessed(ctx context.Context, msg *ProcessedMessage) error

	// IsProcessed checks if a message has been processed
	// Returns true if the message has been processed, false otherwise
	IsProcessed(ctx context.Context, messageID, topic, consumerGroup string) (bool, error)

	// Clean removes expired processed messages
	// Returns the number of messages deleted
	Clean(ctx context.Context, before time.Time) (int64, error)

	// EnsureIndexes ensures that all required indexes are created
	// Should be called on service startup
	EnsureIndexes(ctx context.Context) error
}
