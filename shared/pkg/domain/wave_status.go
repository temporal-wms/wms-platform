package domain

import "errors"

// ErrInvalidWaveStatus is returned when an invalid wave status value is provided
var ErrInvalidWaveStatus = errors.New("invalid wave status value")

// ErrInvalidWaveStatusTransition is returned when an invalid wave status transition is attempted
var ErrInvalidWaveStatusTransition = errors.New("invalid wave status transition")

// WaveStatus represents an immutable wave status value object
type WaveStatus struct {
	value string
}

// Valid wave status values
const (
	waveStatusPlanning   = "planning"
	waveStatusScheduled  = "scheduled"
	waveStatusReleased   = "released"
	waveStatusInProgress = "in_progress"
	waveStatusCompleted  = "completed"
	waveStatusCancelled  = "cancelled"
)

// Predefined WaveStatus instances
var (
	WaveStatusPlanning   = WaveStatus{value: waveStatusPlanning}
	WaveStatusScheduled  = WaveStatus{value: waveStatusScheduled}
	WaveStatusReleased   = WaveStatus{value: waveStatusReleased}
	WaveStatusInProgress = WaveStatus{value: waveStatusInProgress}
	WaveStatusCompleted  = WaveStatus{value: waveStatusCompleted}
	WaveStatusCancelled  = WaveStatus{value: waveStatusCancelled}
)

// NewWaveStatus creates a new WaveStatus value object with validation
func NewWaveStatus(ws string) (WaveStatus, error) {
	switch ws {
	case waveStatusPlanning, waveStatusScheduled, waveStatusReleased,
		waveStatusInProgress, waveStatusCompleted, waveStatusCancelled:
		return WaveStatus{value: ws}, nil
	default:
		return WaveStatus{}, ErrInvalidWaveStatus
	}
}

// MustNewWaveStatus creates a WaveStatus or panics if invalid (use for constants only)
func MustNewWaveStatus(ws string) WaveStatus {
	waveStatus, err := NewWaveStatus(ws)
	if err != nil {
		panic(err)
	}
	return waveStatus
}

// String returns the string representation of the wave status
func (ws WaveStatus) String() string {
	return ws.value
}

// Equals checks if two wave statuses are equal
func (ws WaveStatus) Equals(other WaveStatus) bool {
	return ws.value == other.value
}

// IsPlanning returns true if the wave status is planning
func (ws WaveStatus) IsPlanning() bool {
	return ws.value == waveStatusPlanning
}

// IsScheduled returns true if the wave status is scheduled
func (ws WaveStatus) IsScheduled() bool {
	return ws.value == waveStatusScheduled
}

// IsReleased returns true if the wave status is released
func (ws WaveStatus) IsReleased() bool {
	return ws.value == waveStatusReleased
}

// IsInProgress returns true if the wave status is in progress
func (ws WaveStatus) IsInProgress() bool {
	return ws.value == waveStatusInProgress
}

// IsCompleted returns true if the wave status is completed
func (ws WaveStatus) IsCompleted() bool {
	return ws.value == waveStatusCompleted
}

// IsCancelled returns true if the wave status is cancelled
func (ws WaveStatus) IsCancelled() bool {
	return ws.value == waveStatusCancelled
}

// IsFinal returns true if the wave status is a final state (completed or cancelled)
func (ws WaveStatus) IsFinal() bool {
	return ws.value == waveStatusCompleted || ws.value == waveStatusCancelled
}

// IsActive returns true if the wave is in an active state (not completed or cancelled)
func (ws WaveStatus) IsActive() bool {
	return !ws.IsFinal()
}

// CanAddOrders returns true if orders can be added to the wave in this status
func (ws WaveStatus) CanAddOrders() bool {
	// Orders can only be added during planning and scheduled states
	return ws.value == waveStatusPlanning || ws.value == waveStatusScheduled
}

// CanTransitionTo checks if this wave status can transition to another status
func (ws WaveStatus) CanTransitionTo(target WaveStatus) bool {
	// Define valid transitions based on wave lifecycle state machine
	validTransitions := map[string][]string{
		waveStatusPlanning:   {waveStatusScheduled, waveStatusCancelled},
		waveStatusScheduled:  {waveStatusReleased, waveStatusCancelled},
		waveStatusReleased:   {waveStatusInProgress, waveStatusCancelled},
		waveStatusInProgress: {waveStatusCompleted, waveStatusCancelled},
		waveStatusCompleted:  {}, // Terminal state
		waveStatusCancelled:  {}, // Terminal state
	}

	allowedTargets, exists := validTransitions[ws.value]
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
func (ws WaveStatus) MarshalText() ([]byte, error) {
	return []byte(ws.value), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/BSON deserialization
func (ws *WaveStatus) UnmarshalText(text []byte) error {
	waveStatus, err := NewWaveStatus(string(text))
	if err != nil {
		return err
	}
	*ws = waveStatus
	return nil
}
