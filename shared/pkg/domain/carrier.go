package domain

import (
	"encoding/json"
	"errors"
	"strings"
)

// ErrInvalidCarrier is returned when an invalid carrier value is provided
var ErrInvalidCarrier = errors.New("invalid carrier value")

// Carrier represents an immutable shipping carrier value object
type Carrier struct {
	code        string
	name        string
	accountID   string
	serviceType string
}

// Valid carrier codes
const (
	carrierCodeUPS   = "UPS"
	carrierCodeFedEx = "FEDEX"
	carrierCodeUSPS  = "USPS"
	carrierCodeDHL   = "DHL"
)

// Carrier names
const (
	carrierNameUPS   = "United Parcel Service"
	carrierNameFedEx = "Federal Express"
	carrierNameUSPS  = "United States Postal Service"
	carrierNameDHL   = "DHL Express"
)

// Predefined Carrier instances (without account/service - use NewCarrier for those)
var (
	CarrierUPS   = Carrier{code: carrierCodeUPS, name: carrierNameUPS}
	CarrierFedEx = Carrier{code: carrierCodeFedEx, name: carrierNameFedEx}
	CarrierUSPS  = Carrier{code: carrierCodeUSPS, name: carrierNameUSPS}
	CarrierDHL   = Carrier{code: carrierCodeDHL, name: carrierNameDHL}
)

// NewCarrier creates a new Carrier value object with validation
func NewCarrier(code, accountID, serviceType string) (Carrier, error) {
	code = strings.ToUpper(strings.TrimSpace(code))

	var name string
	switch code {
	case carrierCodeUPS:
		name = carrierNameUPS
	case carrierCodeFedEx:
		name = carrierNameFedEx
	case carrierCodeUSPS:
		name = carrierNameUSPS
	case carrierCodeDHL:
		name = carrierNameDHL
	default:
		return Carrier{}, ErrInvalidCarrier
	}

	if accountID == "" {
		return Carrier{}, errors.New("carrier account ID cannot be empty")
	}

	if serviceType == "" {
		return Carrier{}, errors.New("carrier service type cannot be empty")
	}

	return Carrier{
		code:        code,
		name:        name,
		accountID:   accountID,
		serviceType: serviceType,
	}, nil
}

// NewCarrierWithDefaults creates a Carrier with just the code (for predefined carriers)
func NewCarrierWithDefaults(code string) (Carrier, error) {
	code = strings.ToUpper(strings.TrimSpace(code))

	var name string
	switch code {
	case carrierCodeUPS:
		name = carrierNameUPS
	case carrierCodeFedEx:
		name = carrierNameFedEx
	case carrierCodeUSPS:
		name = carrierNameUSPS
	case carrierCodeDHL:
		name = carrierNameDHL
	default:
		return Carrier{}, ErrInvalidCarrier
	}

	return Carrier{
		code: code,
		name: name,
	}, nil
}

// MustNewCarrier creates a Carrier or panics if invalid (use for constants only)
func MustNewCarrier(code, accountID, serviceType string) Carrier {
	carrier, err := NewCarrier(code, accountID, serviceType)
	if err != nil {
		panic(err)
	}
	return carrier
}

// Code returns the carrier code (UPS, FEDEX, USPS, DHL)
func (c Carrier) Code() string {
	return c.code
}

// Name returns the full carrier name
func (c Carrier) Name() string {
	return c.name
}

// AccountID returns the carrier account ID
func (c Carrier) AccountID() string {
	return c.accountID
}

// ServiceType returns the service type (e.g., "Ground", "Express", "2Day")
func (c Carrier) ServiceType() string {
	return c.serviceType
}

// String returns the string representation (code)
func (c Carrier) String() string {
	return c.code
}

// Equals checks if two carriers are equal
func (c Carrier) Equals(other Carrier) bool {
	return c.code == other.code &&
		c.accountID == other.accountID &&
		c.serviceType == other.serviceType
}

// IsUPS returns true if the carrier is UPS
func (c Carrier) IsUPS() bool {
	return c.code == carrierCodeUPS
}

// IsFedEx returns true if the carrier is FedEx
func (c Carrier) IsFedEx() bool {
	return c.code == carrierCodeFedEx
}

// IsUSPS returns true if the carrier is USPS
func (c Carrier) IsUSPS() bool {
	return c.code == carrierCodeUSPS
}

// IsDHL returns true if the carrier is DHL
func (c Carrier) IsDHL() bool {
	return c.code == carrierCodeDHL
}

// SupportsInternationalShipping returns true if the carrier supports international shipping
func (c Carrier) SupportsInternationalShipping() bool {
	// All major carriers support international, but USPS has more restrictions
	return c.code == carrierCodeUPS || c.code == carrierCodeFedEx || c.code == carrierCodeDHL
}

// RequiresCustomsDocumentation returns true if this carrier/service requires customs docs
func (c Carrier) RequiresCustomsDocumentation() bool {
	// International services require customs documentation
	serviceType := strings.ToLower(c.serviceType)
	return strings.Contains(serviceType, "international") ||
		strings.Contains(serviceType, "worldwide") ||
		strings.Contains(serviceType, "express")
}

// MarshalJSON implements json.Marshaler for JSON serialization
func (c Carrier) MarshalJSON() ([]byte, error) {
	// Serialize as a structured object for BSON/JSON compatibility
	return json.Marshal(struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		AccountID   string `json:"accountId"`
		ServiceType string `json:"serviceType"`
	}{
		Code:        c.code,
		Name:        c.name,
		AccountID:   c.accountID,
		ServiceType: c.serviceType,
	})
}

// UnmarshalJSON implements json.Unmarshaler for JSON deserialization
func (c *Carrier) UnmarshalJSON(data []byte) error {
	var raw struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		AccountID   string `json:"accountId"`
		ServiceType string `json:"serviceType"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// If accountID and serviceType are empty, use defaults
	if raw.AccountID == "" || raw.ServiceType == "" {
		carrier, err := NewCarrierWithDefaults(raw.Code)
		if err != nil {
			return err
		}
		*c = carrier
		return nil
	}

	carrier, err := NewCarrier(raw.Code, raw.AccountID, raw.ServiceType)
	if err != nil {
		return err
	}

	*c = carrier
	return nil
}
