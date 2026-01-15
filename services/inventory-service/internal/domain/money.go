package domain

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Money represents a monetary value with currency
// Amount is stored in smallest currency unit (cents) to avoid floating point issues
type Money struct {
	amount   int64  // Stored in cents (or smallest currency unit)
	currency string // ISO 4217 currency code (USD, EUR, etc.)
}

// Errors
var (
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrInvalidCurrency       = errors.New("invalid currency code")
	ErrCurrencyMismatch      = errors.New("currency mismatch")
	ErrNegativeMoney         = errors.New("money amount cannot be negative")
	ErrDivisionByZero        = errors.New("division by zero")
	ErrInvalidMultiplier     = errors.New("multiplier must be positive")
)

// NewMoney creates a new Money value object
// amount is in smallest currency unit (cents)
func NewMoney(amount int64, currency string) (Money, error) {
	if amount < 0 {
		return Money{}, ErrNegativeMoney
	}

	if currency == "" {
		return Money{}, ErrInvalidCurrency
	}

	// Validate currency code format (should be 3 uppercase letters)
	if len(currency) != 3 {
		return Money{}, ErrInvalidCurrency
	}

	return Money{
		amount:   amount,
		currency: currency,
	}, nil
}

// ZeroMoney creates a zero money value
func ZeroMoney(currency string) Money {
	return Money{
		amount:   0,
		currency: currency,
	}
}

// Amount returns the amount in smallest currency unit (cents)
func (m Money) Amount() int64 {
	return m.amount
}

// Currency returns the ISO 4217 currency code
func (m Money) Currency() string {
	return m.currency
}

// IsZero returns true if the amount is zero
func (m Money) IsZero() bool {
	return m.amount == 0
}

// IsPositive returns true if the amount is greater than zero
func (m Money) IsPositive() bool {
	return m.amount > 0
}

// Add adds two money values (must have same currency)
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}

	return Money{
		amount:   m.amount + other.amount,
		currency: m.currency,
	}, nil
}

// Subtract subtracts other from this money (must have same currency)
func (m Money) Subtract(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}

	if m.amount < other.amount {
		return Money{}, ErrNegativeMoney
	}

	return Money{
		amount:   m.amount - other.amount,
		currency: m.currency,
	}, nil
}

// Multiply multiplies the amount by a quantity
func (m Money) Multiply(qty int) (Money, error) {
	if qty < 0 {
		return Money{}, ErrInvalidMultiplier
	}

	return Money{
		amount:   m.amount * int64(qty),
		currency: m.currency,
	}, nil
}

// Divide divides the amount by a divisor (for weighted average calculations)
func (m Money) Divide(divisor int) (Money, error) {
	if divisor == 0 {
		return Money{}, ErrDivisionByZero
	}

	if divisor < 0 {
		return Money{}, ErrInvalidMultiplier
	}

	return Money{
		amount:   m.amount / int64(divisor),
		currency: m.currency,
	}, nil
}

// Equals checks if two money values are equal (amount and currency)
func (m Money) Equals(other Money) bool {
	return m.amount == other.amount && m.currency == other.currency
}

// GreaterThan checks if this money is greater than other
func (m Money) GreaterThan(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, ErrCurrencyMismatch
	}

	return m.amount > other.amount, nil
}

// LessThan checks if this money is less than other
func (m Money) LessThan(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, ErrCurrencyMismatch
	}

	return m.amount < other.amount, nil
}

// String returns a string representation of the money
func (m Money) String() string {
	// Convert cents to dollars (or equivalent)
	dollars := float64(m.amount) / 100.0
	return fmt.Sprintf("%.2f %s", dollars, m.currency)
}

// ToCents is an alias for Amount() for clarity
func (m Money) ToCents() int64 {
	return m.amount
}

// MarshalBSONValue implements bson.ValueMarshaler
func (m Money) MarshalBSONValue() (bsontype.Type, []byte, error) {
	doc := primitive.D{
		{Key: "amount", Value: m.amount},
		{Key: "currency", Value: m.currency},
	}
	return bson.MarshalValue(doc)
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler
func (m *Money) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	var doc primitive.D
	if err := bson.UnmarshalValue(t, data, &doc); err != nil {
		return err
	}

	docMap := doc.Map()
	if amount, ok := docMap["amount"].(int64); ok {
		m.amount = amount
	}
	if currency, ok := docMap["currency"].(string); ok {
		m.currency = currency
	}

	return nil
}
