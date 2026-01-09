package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures
func createTestOrderItems() []OrderItem {
	return []OrderItem{
		{
			SKU:      "SKU-001",
			Name:     "Test Product",
			Quantity: 2,
			Weight:   1.5,
			Dimensions: Dims{
				Length: 10,
				Width:  5,
				Height: 3,
			},
			UnitPrice: 29.99,
		},
	}
}

func createTestAddress() Address {
	return Address{
		Street:        "123 Main St",
		City:          "San Francisco",
		State:         "CA",
		ZipCode:       "94105",
		Country:       "USA",
		Phone:         "+1-555-0123",
		RecipientName: "John Doe",
	}
}

// TestNewOrder tests order creation
func TestNewOrder(t *testing.T) {
	tests := []struct {
		name                string
		orderID             string
		customerID          string
		items               []OrderItem
		address             Address
		priority            Priority
		promisedDeliveryAt  time.Time
		expectError         error
	}{
		{
			name:               "Valid order creation",
			orderID:            "ORD-001",
			customerID:         "CUST-001",
			items:              createTestOrderItems(),
			address:            createTestAddress(),
			priority:           PrioritySameDay,
			promisedDeliveryAt: time.Now().Add(24 * time.Hour),
			expectError:        nil,
		},
		{
			name:               "Order with no items",
			orderID:            "ORD-002",
			customerID:         "CUST-001",
			items:              []OrderItem{},
			address:            createTestAddress(),
			priority:           PriorityStandard,
			promisedDeliveryAt: time.Now().Add(72 * time.Hour),
			expectError:        ErrNoItems,
		},
		{
			name:               "Order with invalid priority",
			orderID:            "ORD-003",
			customerID:         "CUST-001",
			items:              createTestOrderItems(),
			address:            createTestAddress(),
			priority:           Priority("invalid"),
			promisedDeliveryAt: time.Now().Add(24 * time.Hour),
			expectError:        ErrInvalidPriority,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := NewOrder(
				tt.orderID,
				tt.customerID,
				tt.items,
				tt.address,
				tt.priority,
				tt.promisedDeliveryAt,
			)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
				assert.Nil(t, order)
			} else {
				require.NoError(t, err)
				require.NotNil(t, order)
				assert.Equal(t, tt.orderID, order.OrderID)
				assert.Equal(t, tt.customerID, order.CustomerID)
				assert.Equal(t, StatusReceived, order.Status)
				assert.Len(t, order.Items, len(tt.items))
				assert.NotZero(t, order.CreatedAt)
				assert.NotZero(t, order.UpdatedAt)

				// Check domain event was created
				events := order.DomainEvents()
				assert.Len(t, events, 1)
				event, ok := events[0].(*OrderReceivedEvent)
				assert.True(t, ok)
				assert.Equal(t, tt.orderID, event.OrderID)
			}
		})
	}
}

// TestOrderValidate tests order validation
func TestOrderValidate(t *testing.T) {
	tests := []struct {
		name         string
		setupOrder   func() *Order
		expectError  error
		expectStatus Status
	}{
		{
			name: "Valid order validation",
			setupOrder: func() *Order {
				order, _ := NewOrder(
					"ORD-001",
					"CUST-001",
					createTestOrderItems(),
					createTestAddress(),
					PriorityStandard,
					time.Now().Add(72*time.Hour),
				)
				return order
			},
			expectError:  nil,
			expectStatus: StatusValidated,
		},
		{
			name: "Cannot validate cancelled order",
			setupOrder: func() *Order {
				order, _ := NewOrder(
					"ORD-002",
					"CUST-001",
					createTestOrderItems(),
					createTestAddress(),
					PriorityStandard,
					time.Now().Add(72*time.Hour),
				)
				order.Cancel("Test cancellation")
				return order
			},
			expectError:  ErrOrderCancelled,
			expectStatus: StatusCancelled,
		},
		{
			name: "Cannot validate already validated order",
			setupOrder: func() *Order {
				order, _ := NewOrder(
					"ORD-003",
					"CUST-001",
					createTestOrderItems(),
					createTestAddress(),
					PriorityStandard,
					time.Now().Add(72*time.Hour),
				)
				order.Validate()
				return order
			},
			expectError:  ErrInvalidStatus,
			expectStatus: StatusValidated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := tt.setupOrder()
			err := order.Validate()

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectStatus, order.Status)
			}
		})
	}
}

// TestOrderCancel tests order cancellation
func TestOrderCancel(t *testing.T) {
	order, err := NewOrder(
		"ORD-001",
		"CUST-001",
		createTestOrderItems(),
		createTestAddress(),
		PriorityStandard,
		time.Now().Add(72*time.Hour),
	)
	require.NoError(t, err)

	reason := "Customer requested cancellation"
	err = order.Cancel(reason)

	assert.NoError(t, err)
	assert.Equal(t, StatusCancelled, order.Status)

	// Check domain event
	events := order.DomainEvents()
	found := false
	for _, event := range events {
		if cancelledEvent, ok := event.(*OrderCancelledEvent); ok {
			assert.Equal(t, order.OrderID, cancelledEvent.OrderID)
			assert.Equal(t, reason, cancelledEvent.Reason)
			found = true
			break
		}
	}
	assert.True(t, found, "OrderCancelledEvent should be present")
}

// TestOrderAssignWave tests wave assignment
func TestOrderAssignWave(t *testing.T) {
	tests := []struct {
		name        string
		setupOrder  func() *Order
		waveID      string
		expectError error
	}{
		{
			name: "Valid wave assignment",
			setupOrder: func() *Order {
				order, _ := NewOrder(
					"ORD-001",
					"CUST-001",
					createTestOrderItems(),
					createTestAddress(),
					PriorityStandard,
					time.Now().Add(72*time.Hour),
				)
				order.Validate()
				return order
			},
			waveID:      "WAVE-001",
			expectError: nil,
		},
		{
			name: "Cannot assign wave to received order",
			setupOrder: func() *Order {
				order, _ := NewOrder(
					"ORD-002",
					"CUST-001",
					createTestOrderItems(),
					createTestAddress(),
					PriorityStandard,
					time.Now().Add(72*time.Hour),
				)
				return order
			},
			waveID:      "WAVE-001",
			expectError: ErrInvalidStatus,
		},
		{
			name: "Cannot reassign wave",
			setupOrder: func() *Order {
				order, _ := NewOrder(
					"ORD-003",
					"CUST-001",
					createTestOrderItems(),
					createTestAddress(),
					PriorityStandard,
					time.Now().Add(72*time.Hour),
				)
				order.Validate()
				order.AssignToWave("WAVE-001")
				return order
			},
			waveID:      "WAVE-002",
			expectError: ErrInvalidStatus, // Order is now in WaveAssigned status, not Validated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := tt.setupOrder()
			err := order.AssignToWave(tt.waveID)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.waveID, order.WaveID)
				assert.Equal(t, StatusWaveAssigned, order.Status)
			}
		})
	}
}

// TestPriorityIsValid tests priority validation
func TestPriorityIsValid(t *testing.T) {
	tests := []struct {
		priority Priority
		expected bool
	}{
		{PrioritySameDay, true},
		{PriorityNextDay, true},
		{PriorityStandard, true},
		{Priority("invalid"), false},
		{Priority(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.priority.IsValid())
		})
	}
}

// TestOrderDomainEvents tests domain event handling
func TestOrderDomainEvents(t *testing.T) {
	order, err := NewOrder(
		"ORD-001",
		"CUST-001",
		createTestOrderItems(),
		createTestAddress(),
		PrioritySameDay,
		time.Now().Add(24*time.Hour),
	)
	require.NoError(t, err)

	// Check initial event
	events := order.DomainEvents()
	assert.Len(t, events, 1)

	// Perform actions that generate events
	order.Validate()
	events = order.DomainEvents()
	assert.Len(t, events, 2)

	order.AssignToWave("WAVE-001")
	events = order.DomainEvents()
	assert.Len(t, events, 3)

	// Clear events
	order.ClearDomainEvents()
	events = order.DomainEvents()
	assert.Len(t, events, 0)
}

// BenchmarkNewOrder benchmarks order creation
func BenchmarkNewOrder(b *testing.B) {
	items := createTestOrderItems()
	address := createTestAddress()
	promisedTime := time.Now().Add(24 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewOrder(
			"ORD-001",
			"CUST-001",
			items,
			address,
			PrioritySameDay,
			promisedTime,
		)
	}
}
