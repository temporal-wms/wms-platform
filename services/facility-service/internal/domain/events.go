package domain

import "time"

// DomainEvent interface for domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}
