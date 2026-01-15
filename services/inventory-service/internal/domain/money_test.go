package domain

import (
	"testing"
)

func TestNewMoney(t *testing.T) {
	tests := []struct {
		name        string
		amount      int64
		currency    string
		expectError bool
	}{
		{
			name:        "valid money",
			amount:      1000,
			currency:    "USD",
			expectError: false,
		},
		{
			name:        "zero amount is valid",
			amount:      0,
			currency:    "USD",
			expectError: false,
		},
		{
			name:        "negative amount",
			amount:      -100,
			currency:    "USD",
			expectError: true,
		},
		{
			name:        "empty currency",
			amount:      1000,
			currency:    "",
			expectError: true,
		},
		{
			name:        "invalid currency code length",
			amount:      1000,
			currency:    "US",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := NewMoney(tt.amount, tt.currency)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if money.Amount() != tt.amount {
					t.Errorf("expected amount %d, got %d", tt.amount, money.Amount())
				}
				if money.Currency() != tt.currency {
					t.Errorf("expected currency %s, got %s", tt.currency, money.Currency())
				}
			}
		})
	}
}

func TestMoney_Add(t *testing.T) {
	tests := []struct {
		name        string
		money1      Money
		money2      Money
		expected    int64
		expectError bool
	}{
		{
			name:        "add same currency",
			money1:      mustNewMoney(1000, "USD"),
			money2:      mustNewMoney(500, "USD"),
			expected:    1500,
			expectError: false,
		},
		{
			name:        "add zero",
			money1:      mustNewMoney(1000, "USD"),
			money2:      ZeroMoney("USD"),
			expected:    1000,
			expectError: false,
		},
		{
			name:        "currency mismatch",
			money1:      mustNewMoney(1000, "USD"),
			money2:      mustNewMoney(500, "EUR"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.money1.Add(tt.money2)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.Amount() != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, result.Amount())
				}
			}
		})
	}
}

func TestMoney_Subtract(t *testing.T) {
	tests := []struct {
		name        string
		money1      Money
		money2      Money
		expected    int64
		expectError bool
	}{
		{
			name:        "subtract same currency",
			money1:      mustNewMoney(1000, "USD"),
			money2:      mustNewMoney(500, "USD"),
			expected:    500,
			expectError: false,
		},
		{
			name:        "subtract to zero",
			money1:      mustNewMoney(1000, "USD"),
			money2:      mustNewMoney(1000, "USD"),
			expected:    0,
			expectError: false,
		},
		{
			name:        "subtract more than available",
			money1:      mustNewMoney(500, "USD"),
			money2:      mustNewMoney(1000, "USD"),
			expectError: true,
		},
		{
			name:        "currency mismatch",
			money1:      mustNewMoney(1000, "USD"),
			money2:      mustNewMoney(500, "EUR"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.money1.Subtract(tt.money2)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.Amount() != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, result.Amount())
				}
			}
		})
	}
}

func TestMoney_Multiply(t *testing.T) {
	tests := []struct {
		name        string
		money       Money
		qty         int
		expected    int64
		expectError bool
	}{
		{
			name:        "multiply by positive",
			money:       mustNewMoney(1500, "USD"),
			qty:         10,
			expected:    15000,
			expectError: false,
		},
		{
			name:        "multiply by zero",
			money:       mustNewMoney(1500, "USD"),
			qty:         0,
			expected:    0,
			expectError: false,
		},
		{
			name:        "multiply by negative",
			money:       mustNewMoney(1500, "USD"),
			qty:         -5,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.money.Multiply(tt.qty)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.Amount() != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, result.Amount())
				}
			}
		})
	}
}

func TestMoney_Divide(t *testing.T) {
	tests := []struct {
		name        string
		money       Money
		divisor     int
		expected    int64
		expectError bool
	}{
		{
			name:        "divide evenly",
			money:       mustNewMoney(1500, "USD"),
			divisor:     10,
			expected:    150,
			expectError: false,
		},
		{
			name:        "divide with remainder",
			money:       mustNewMoney(1000, "USD"),
			divisor:     3,
			expected:    333, // Integer division
			expectError: false,
		},
		{
			name:        "divide by zero",
			money:       mustNewMoney(1500, "USD"),
			divisor:     0,
			expectError: true,
		},
		{
			name:        "divide by negative",
			money:       mustNewMoney(1500, "USD"),
			divisor:     -5,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.money.Divide(tt.divisor)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.Amount() != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, result.Amount())
				}
			}
		})
	}
}

func TestMoney_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		money    Money
		expected bool
	}{
		{
			name:     "zero money",
			money:    ZeroMoney("USD"),
			expected: true,
		},
		{
			name:     "positive money",
			money:    mustNewMoney(100, "USD"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.money.IsZero() != tt.expected {
				t.Errorf("expected IsZero() = %v, got %v", tt.expected, tt.money.IsZero())
			}
		})
	}
}

func TestMoney_Equals(t *testing.T) {
	tests := []struct {
		name     string
		money1   Money
		money2   Money
		expected bool
	}{
		{
			name:     "equal money",
			money1:   mustNewMoney(1000, "USD"),
			money2:   mustNewMoney(1000, "USD"),
			expected: true,
		},
		{
			name:     "different amounts",
			money1:   mustNewMoney(1000, "USD"),
			money2:   mustNewMoney(500, "USD"),
			expected: false,
		},
		{
			name:     "different currencies",
			money1:   mustNewMoney(1000, "USD"),
			money2:   mustNewMoney(1000, "EUR"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.money1.Equals(tt.money2) != tt.expected {
				t.Errorf("expected Equals() = %v, got %v", tt.expected, tt.money1.Equals(tt.money2))
			}
		})
	}
}

func TestMoney_String(t *testing.T) {
	tests := []struct {
		name     string
		money    Money
		expected string
	}{
		{
			name:     "dollars and cents",
			money:    mustNewMoney(1550, "USD"),
			expected: "15.50 USD",
		},
		{
			name:     "zero",
			money:    ZeroMoney("USD"),
			expected: "0.00 USD",
		},
		{
			name:     "large amount",
			money:    mustNewMoney(1000000, "USD"),
			expected: "10000.00 USD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.money.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.money.String())
			}
		})
	}
}

// Helper function for tests
func mustNewMoney(amount int64, currency string) Money {
	money, err := NewMoney(amount, currency)
	if err != nil {
		panic(err)
	}
	return money
}
