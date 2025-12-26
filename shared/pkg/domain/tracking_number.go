package domain

import (
	"errors"
	"regexp"
	"strings"
)

// ErrInvalidTrackingNumber is returned when an invalid tracking number is provided
var ErrInvalidTrackingNumber = errors.New("invalid tracking number")

// TrackingNumber represents an immutable tracking number value object
type TrackingNumber struct {
	value   string
	carrier string // Auto-detected carrier code
}

// Tracking number patterns for different carriers
var (
	// UPS: 18 characters starting with "1Z"
	upsPattern = regexp.MustCompile(`^1Z[A-Z0-9]{16}$`)

	// FedEx: 12 or 15 digits
	fedexPattern = regexp.MustCompile(`^\d{12}$|^\d{15}$`)

	// USPS: 20-22 digits
	uspsPattern = regexp.MustCompile(`^\d{20,22}$`)

	// DHL: 10 or 11 digits
	dhlPattern = regexp.MustCompile(`^\d{10,11}$`)
)

// NewTrackingNumber creates a new TrackingNumber value object with validation
func NewTrackingNumber(trackingNumber string) (TrackingNumber, error) {
	// Trim and uppercase
	trackingNumber = strings.ToUpper(strings.TrimSpace(trackingNumber))

	if trackingNumber == "" {
		return TrackingNumber{}, errors.New("tracking number cannot be empty")
	}

	// Detect carrier and validate format
	carrier := detectCarrier(trackingNumber)
	if carrier == "" {
		// If we can't detect carrier, still allow it but validate basic format
		if !isValidBasicFormat(trackingNumber) {
			return TrackingNumber{}, ErrInvalidTrackingNumber
		}
		carrier = "UNKNOWN"
	}

	return TrackingNumber{
		value:   trackingNumber,
		carrier: carrier,
	}, nil
}

// NewTrackingNumberForCarrier creates a TrackingNumber for a specific carrier with validation
func NewTrackingNumberForCarrier(trackingNumber, carrierCode string) (TrackingNumber, error) {
	trackingNumber = strings.ToUpper(strings.TrimSpace(trackingNumber))
	carrierCode = strings.ToUpper(strings.TrimSpace(carrierCode))

	if trackingNumber == "" {
		return TrackingNumber{}, errors.New("tracking number cannot be empty")
	}

	// Validate format based on carrier
	var valid bool
	switch carrierCode {
	case "UPS":
		valid = upsPattern.MatchString(trackingNumber)
	case "FEDEX":
		valid = fedexPattern.MatchString(trackingNumber)
	case "USPS":
		valid = uspsPattern.MatchString(trackingNumber)
	case "DHL":
		valid = dhlPattern.MatchString(trackingNumber)
	default:
		// Unknown carrier, use basic validation
		valid = isValidBasicFormat(trackingNumber)
	}

	if !valid {
		return TrackingNumber{}, ErrInvalidTrackingNumber
	}

	return TrackingNumber{
		value:   trackingNumber,
		carrier: carrierCode,
	}, nil
}

// MustNewTrackingNumber creates a TrackingNumber or panics if invalid (use for constants only)
func MustNewTrackingNumber(trackingNumber string) TrackingNumber {
	tn, err := NewTrackingNumber(trackingNumber)
	if err != nil {
		panic(err)
	}
	return tn
}

// Value returns the tracking number value
func (tn TrackingNumber) Value() string {
	return tn.value
}

// Carrier returns the detected carrier code
func (tn TrackingNumber) Carrier() string {
	return tn.carrier
}

// String returns the string representation of the tracking number
func (tn TrackingNumber) String() string {
	return tn.value
}

// Equals checks if two tracking numbers are equal
func (tn TrackingNumber) Equals(other TrackingNumber) bool {
	return tn.value == other.value
}

// IsUPS returns true if this is a UPS tracking number
func (tn TrackingNumber) IsUPS() bool {
	return tn.carrier == "UPS"
}

// IsFedEx returns true if this is a FedEx tracking number
func (tn TrackingNumber) IsFedEx() bool {
	return tn.carrier == "FEDEX"
}

// IsUSPS returns true if this is a USPS tracking number
func (tn TrackingNumber) IsUSPS() bool {
	return tn.carrier == "USPS"
}

// IsDHL returns true if this is a DHL tracking number
func (tn TrackingNumber) IsDHL() bool {
	return tn.carrier == "DHL"
}

// IsValid validates the tracking number format
func (tn TrackingNumber) IsValid() bool {
	switch tn.carrier {
	case "UPS":
		return upsPattern.MatchString(tn.value)
	case "FEDEX":
		return fedexPattern.MatchString(tn.value)
	case "USPS":
		return uspsPattern.MatchString(tn.value)
	case "DHL":
		return dhlPattern.MatchString(tn.value)
	default:
		return isValidBasicFormat(tn.value)
	}
}

// GetTrackingURL returns the tracking URL for this tracking number
func (tn TrackingNumber) GetTrackingURL() string {
	switch tn.carrier {
	case "UPS":
		return "https://www.ups.com/track?tracknum=" + tn.value
	case "FEDEX":
		return "https://www.fedex.com/fedextrack/?trknbr=" + tn.value
	case "USPS":
		return "https://tools.usps.com/go/TrackConfirmAction?tLabels=" + tn.value
	case "DHL":
		return "https://www.dhl.com/en/express/tracking.html?AWB=" + tn.value
	default:
		return ""
	}
}

// MarshalText implements encoding.TextMarshaler for JSON/BSON serialization
func (tn TrackingNumber) MarshalText() ([]byte, error) {
	return []byte(tn.value), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/BSON deserialization
func (tn *TrackingNumber) UnmarshalText(text []byte) error {
	trackingNumber, err := NewTrackingNumber(string(text))
	if err != nil {
		return err
	}
	*tn = trackingNumber
	return nil
}

// detectCarrier attempts to detect the carrier from the tracking number format
func detectCarrier(trackingNumber string) string {
	if upsPattern.MatchString(trackingNumber) {
		return "UPS"
	}
	if fedexPattern.MatchString(trackingNumber) {
		// FedEx has overlap with USPS and DHL, check length
		length := len(trackingNumber)
		if length == 12 || length == 15 {
			return "FEDEX"
		}
	}
	if uspsPattern.MatchString(trackingNumber) {
		return "USPS"
	}
	if dhlPattern.MatchString(trackingNumber) {
		return "DHL"
	}
	return ""
}

// isValidBasicFormat checks if the tracking number has a valid basic format
// (alphanumeric, 8-30 characters)
func isValidBasicFormat(trackingNumber string) bool {
	if len(trackingNumber) < 8 || len(trackingNumber) > 30 {
		return false
	}
	// Must be alphanumeric
	return regexp.MustCompile(`^[A-Z0-9]+$`).MatchString(trackingNumber)
}
