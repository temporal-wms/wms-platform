package domain

import "errors"

// ErrInvalidStatus is returned when an invalid status value is provided
var ErrInvalidStatus = errors.New("invalid status value")

// ErrInvalidStatusTransition is returned when an invalid status transition is attempted
var ErrInvalidStatusTransition = errors.New("invalid status transition")

// Status represents an immutable order status value object
type Status struct {
	value string
}

// Valid status values
const (
	statusReceived     = "received"
	statusValidated    = "validated"
	statusWaveAssigned = "wave_assigned"
	statusPicking      = "picking"
	statusConsolidated = "consolidated"
	statusPacked       = "packed"
	statusShipped      = "shipped"
	statusDelivered    = "delivered"
	statusCancelled    = "cancelled"
)

// Predefined Status instances
var (
	StatusReceived     = Status{value: statusReceived}
	StatusValidated    = Status{value: statusValidated}
	StatusWaveAssigned = Status{value: statusWaveAssigned}
	StatusPicking      = Status{value: statusPicking}
	StatusConsolidated = Status{value: statusConsolidated}
	StatusPacked       = Status{value: statusPacked}
	StatusShipped      = Status{value: statusShipped}
	StatusDelivered    = Status{value: statusDelivered}
	StatusCancelled    = Status{value: statusCancelled}
)

// NewStatus creates a new Status value object with validation
func NewStatus(s string) (Status, error) {
	switch s {
	case statusReceived, statusValidated, statusWaveAssigned, statusPicking,
		statusConsolidated, statusPacked, statusShipped, statusDelivered, statusCancelled:
		return Status{value: s}, nil
	default:
		return Status{}, ErrInvalidStatus
	}
}

// MustNewStatus creates a Status or panics if invalid (use for constants only)
func MustNewStatus(s string) Status {
	status, err := NewStatus(s)
	if err != nil {
		panic(err)
	}
	return status
}

// String returns the string representation of the status
func (s Status) String() string {
	return s.value
}

// Equals checks if two statuses are equal
func (s Status) Equals(other Status) bool {
	return s.value == other.value
}

// IsReceived returns true if the status is received
func (s Status) IsReceived() bool {
	return s.value == statusReceived
}

// IsValidated returns true if the status is validated
func (s Status) IsValidated() bool {
	return s.value == statusValidated
}

// IsWaveAssigned returns true if the status is wave assigned
func (s Status) IsWaveAssigned() bool {
	return s.value == statusWaveAssigned
}

// IsPicking returns true if the status is picking
func (s Status) IsPicking() bool {
	return s.value == statusPicking
}

// IsConsolidated returns true if the status is consolidated
func (s Status) IsConsolidated() bool {
	return s.value == statusConsolidated
}

// IsPacked returns true if the status is packed
func (s Status) IsPacked() bool {
	return s.value == statusPacked
}

// IsShipped returns true if the status is shipped
func (s Status) IsShipped() bool {
	return s.value == statusShipped
}

// IsDelivered returns true if the status is delivered
func (s Status) IsDelivered() bool {
	return s.value == statusDelivered
}

// IsCancelled returns true if the status is cancelled
func (s Status) IsCancelled() bool {
	return s.value == statusCancelled
}

// IsFinal returns true if the status is a final state (shipped, delivered, or cancelled)
func (s Status) IsFinal() bool {
	return s.value == statusShipped || s.value == statusDelivered || s.value == statusCancelled
}

// CanTransitionTo checks if this status can transition to another status
func (s Status) CanTransitionTo(target Status) bool {
	// Define valid transitions based on order fulfillment state machine
	validTransitions := map[string][]string{
		statusReceived:     {statusValidated, statusCancelled},
		statusValidated:    {statusWaveAssigned, statusCancelled},
		statusWaveAssigned: {statusPicking, statusCancelled},
		statusPicking:      {statusConsolidated, statusPacked, statusCancelled},
		statusConsolidated: {statusPacked, statusCancelled},
		statusPacked:       {statusShipped, statusCancelled},
		statusShipped:      {statusDelivered},
		statusDelivered:    {}, // Terminal state
		statusCancelled:    {}, // Terminal state
	}

	allowedTargets, exists := validTransitions[s.value]
	if !exists {
		return false
	}

	for _, allowed := range allowedTargets {
		if target.value == allowed {
			return true
		}
	}

	return false
}

// MarshalText implements encoding.TextMarshaler for JSON/BSON serialization
func (s Status) MarshalText() ([]byte, error) {
	return []byte(s.value), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/BSON deserialization
func (s *Status) UnmarshalText(text []byte) error {
	status, err := NewStatus(string(text))
	if err != nil {
		return err
	}
	*s = status
	return nil
}
