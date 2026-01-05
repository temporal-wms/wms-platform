package idempotency

import (
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultMaxKeyLength is the maximum length for an idempotency key (Stripe standard)
	DefaultMaxKeyLength = 255

	// DefaultLockTimeout is the default timeout for acquiring a lock
	DefaultLockTimeout = 5 * time.Minute

	// DefaultRetentionPeriod is the default retention period for idempotency keys
	DefaultRetentionPeriod = 24 * time.Hour

	// DefaultMaxResponseSize is the maximum response size to cache (1MB)
	DefaultMaxResponseSize = 1 * 1024 * 1024
)

// Config holds configuration for the idempotency middleware
type Config struct {
	// ServiceName is the name of the service (e.g., "order-service")
	ServiceName string

	// Repository is the storage backend for idempotency keys
	Repository KeyRepository

	// RequireKey indicates whether idempotency key is required for all mutating operations
	// If false, operations without a key will proceed normally (backward compatibility)
	RequireKey bool

	// OnlyMutating indicates whether to only apply idempotency to mutating methods
	// If true, only POST, PUT, PATCH, DELETE requests are checked
	// GET requests are idempotent by definition and are skipped
	OnlyMutating bool

	// UserIDExtractor is an optional function to extract user ID from context
	// Used to scope idempotency keys per user
	UserIDExtractor func(*gin.Context) string

	// MaxKeyLength is the maximum allowed length for an idempotency key
	MaxKeyLength int

	// LockTimeout is the duration after which a lock is considered stale
	LockTimeout time.Duration

	// RetentionPeriod is how long idempotency keys are retained
	RetentionPeriod time.Duration

	// MaxResponseSize is the maximum response size to cache
	// Responses larger than this will not be cached
	MaxResponseSize int

	// Metrics is the metrics reporter for observability
	Metrics *Metrics
}

// DefaultConfig returns a default configuration for the given service
func DefaultConfig(serviceName string, repository KeyRepository) *Config {
	return &Config{
		ServiceName:     serviceName,
		Repository:      repository,
		RequireKey:      false, // Start with optional for backward compatibility
		OnlyMutating:    true,  // Only check mutating operations
		MaxKeyLength:    DefaultMaxKeyLength,
		LockTimeout:     DefaultLockTimeout,
		RetentionPeriod: DefaultRetentionPeriod,
		MaxResponseSize: DefaultMaxResponseSize,
	}
}

// ConsumerConfig holds configuration for Kafka consumer message deduplication
type ConsumerConfig struct {
	// ServiceName is the name of the service consuming messages
	ServiceName string

	// Topic is the Kafka topic being consumed
	Topic string

	// ConsumerGroup is the Kafka consumer group
	ConsumerGroup string

	// Repository is the storage backend for processed messages
	Repository MessageRepository

	// RetentionPeriod is how long processed message IDs are retained
	RetentionPeriod time.Duration
}

// DefaultConsumerConfig returns a default consumer configuration
func DefaultConsumerConfig(serviceName, topic, consumerGroup string, repository MessageRepository) *ConsumerConfig {
	return &ConsumerConfig{
		ServiceName:     serviceName,
		Topic:           topic,
		ConsumerGroup:   consumerGroup,
		Repository:      repository,
		RetentionPeriod: DefaultRetentionPeriod,
	}
}
