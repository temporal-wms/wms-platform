package domain

import "errors"

// ErrInvalidPriority is returned when an invalid priority value is provided
var ErrInvalidPriority = errors.New("invalid priority value")

// Priority represents an immutable order priority value object
type Priority struct {
	value string
}

// Valid priority values
const (
	prioritySameDay  = "same_day"
	priorityNextDay  = "next_day"
	priorityStandard = "standard"
)

// Predefined Priority instances
var (
	PrioritySameDay  = Priority{value: prioritySameDay}
	PriorityNextDay  = Priority{value: priorityNextDay}
	PriorityStandard = Priority{value: priorityStandard}
)

// NewPriority creates a new Priority value object with validation
func NewPriority(p string) (Priority, error) {
	switch p {
	case prioritySameDay, priorityNextDay, priorityStandard:
		return Priority{value: p}, nil
	default:
		return Priority{}, ErrInvalidPriority
	}
}

// MustNewPriority creates a Priority or panics if invalid (use for constants only)
func MustNewPriority(p string) Priority {
	priority, err := NewPriority(p)
	if err != nil {
		panic(err)
	}
	return priority
}

// String returns the string representation of the priority
func (p Priority) String() string {
	return p.value
}

// Equals checks if two priorities are equal
func (p Priority) Equals(other Priority) bool {
	return p.value == other.value
}

// IsSameDay returns true if the priority is same-day
func (p Priority) IsSameDay() bool {
	return p.value == prioritySameDay
}

// IsNextDay returns true if the priority is next-day
func (p Priority) IsNextDay() bool {
	return p.value == priorityNextDay
}

// IsStandard returns true if the priority is standard
func (p Priority) IsStandard() bool {
	return p.value == priorityStandard
}

// IsHigherThan returns true if this priority is higher than the other
func (p Priority) IsHigherThan(other Priority) bool {
	return p.rank() > other.rank()
}

// rank returns the priority rank (higher is more urgent)
func (p Priority) rank() int {
	switch p.value {
	case prioritySameDay:
		return 3
	case priorityNextDay:
		return 2
	case priorityStandard:
		return 1
	default:
		return 0
	}
}

// MarshalText implements encoding.TextMarshaler for JSON/BSON serialization
func (p Priority) MarshalText() ([]byte, error) {
	return []byte(p.value), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/BSON deserialization
func (p *Priority) UnmarshalText(text []byte) error {
	priority, err := NewPriority(string(text))
	if err != nil {
		return err
	}
	*p = priority
	return nil
}
