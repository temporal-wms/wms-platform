package domain

import "errors"

// ErrInvalidWaveType is returned when an invalid wave type value is provided
var ErrInvalidWaveType = errors.New("invalid wave type value")

// WaveType represents an immutable wave type value object
type WaveType struct {
	value string
}

// Valid wave type values
const (
	waveTypeDigital   = "digital"
	waveTypeWholesale = "wholesale"
	waveTypePriority  = "priority"
	waveTypeMixed     = "mixed"
)

// Predefined WaveType instances
var (
	WaveTypeDigital   = WaveType{value: waveTypeDigital}
	WaveTypeWholesale = WaveType{value: waveTypeWholesale}
	WaveTypePriority  = WaveType{value: waveTypePriority}
	WaveTypeMixed     = WaveType{value: waveTypeMixed}
)

// NewWaveType creates a new WaveType value object with validation
func NewWaveType(wt string) (WaveType, error) {
	switch wt {
	case waveTypeDigital, waveTypeWholesale, waveTypePriority, waveTypeMixed:
		return WaveType{value: wt}, nil
	default:
		return WaveType{}, ErrInvalidWaveType
	}
}

// MustNewWaveType creates a WaveType or panics if invalid (use for constants only)
func MustNewWaveType(wt string) WaveType {
	waveType, err := NewWaveType(wt)
	if err != nil {
		panic(err)
	}
	return waveType
}

// String returns the string representation of the wave type
func (wt WaveType) String() string {
	return wt.value
}

// Equals checks if two wave types are equal
func (wt WaveType) Equals(other WaveType) bool {
	return wt.value == other.value
}

// IsDigital returns true if the wave type is digital
func (wt WaveType) IsDigital() bool {
	return wt.value == waveTypeDigital
}

// IsWholesale returns true if the wave type is wholesale
func (wt WaveType) IsWholesale() bool {
	return wt.value == waveTypeWholesale
}

// IsPriority returns true if the wave type is priority
func (wt WaveType) IsPriority() bool {
	return wt.value == waveTypePriority
}

// IsMixed returns true if the wave type is mixed
func (wt WaveType) IsMixed() bool {
	return wt.value == waveTypeMixed
}

// RequiresSorting returns true if this wave type typically requires sorting
func (wt WaveType) RequiresSorting() bool {
	// Mixed and wholesale waves typically require more sorting
	return wt.value == waveTypeMixed || wt.value == waveTypeWholesale
}

// MarshalText implements encoding.TextMarshaler for JSON/BSON serialization
func (wt WaveType) MarshalText() ([]byte, error) {
	return []byte(wt.value), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/BSON deserialization
func (wt *WaveType) UnmarshalText(text []byte) error {
	waveType, err := NewWaveType(string(text))
	if err != nil {
		return err
	}
	*wt = waveType
	return nil
}
