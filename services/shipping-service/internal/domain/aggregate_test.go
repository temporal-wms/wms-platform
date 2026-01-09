package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures
func createTestCarrier() Carrier {
	return Carrier{
		Code:        "FEDEX",
		Name:        "FedEx Express",
		AccountID:   "ACCT-12345",
		ServiceType: "OVERNIGHT",
	}
}

func createTestPackageInfo() PackageInfo {
	return PackageInfo{
		PackageID:   "PKG-001",
		Weight:      2.5,
		Dimensions:  Dimensions{Length: 30, Width: 20, Height: 10},
		PackageType: "BOX",
	}
}

func createTestAddress(name string) Address {
	return Address{
		Name:       name,
		Company:    "Test Company",
		Street1:    "123 Main St",
		Street2:    "Suite 100",
		City:       "New York",
		State:      "NY",
		PostalCode: "10001",
		Country:    "US",
		Phone:      "+1-555-0100",
		Email:      "test@example.com",
	}
}

func createTestLabel() ShippingLabel {
	return ShippingLabel{
		TrackingNumber: "1Z999AA10123456784",
		LabelFormat:    "PDF",
		LabelData:      "base64encodeddata==",
		LabelURL:       "https://example.com/label.pdf",
		GeneratedAt:    time.Now(),
	}
}

func createTestManifest() Manifest {
	return Manifest{
		ManifestID:    "MAN-001",
		CarrierCode:   "FEDEX",
		ShipmentCount: 10,
		GeneratedAt:   time.Now(),
	}
}

// TestNewShipment tests shipment creation
func TestNewShipment(t *testing.T) {
	carrier := createTestCarrier()
	pkg := createTestPackageInfo()
	recipient := createTestAddress("John Doe")
	shipper := createTestAddress("Warehouse A")

	shipment := NewShipment("SHIP-001", "ORD-001", "PKG-001", "WAVE-001", carrier, pkg, recipient, shipper)

	require.NotNil(t, shipment)
	assert.Equal(t, "SHIP-001", shipment.ShipmentID)
	assert.Equal(t, "ORD-001", shipment.OrderID)
	assert.Equal(t, "PKG-001", shipment.PackageID)
	assert.Equal(t, "WAVE-001", shipment.WaveID)
	assert.Equal(t, ShipmentStatusPending, shipment.Status)
	assert.Equal(t, carrier.Code, shipment.Carrier.Code)
	assert.Equal(t, pkg.PackageID, shipment.Package.PackageID)
	assert.Equal(t, recipient.Name, shipment.Recipient.Name)
	assert.Equal(t, shipper.Name, shipment.Shipper.Name)
	assert.NotZero(t, shipment.CreatedAt)

	// Check domain event
	events := shipment.GetDomainEvents()
	assert.Len(t, events, 1)
	event, ok := events[0].(*ShipmentCreatedEvent)
	assert.True(t, ok)
	assert.Equal(t, "SHIP-001", event.ShipmentID)
}

// TestShipmentGenerateLabel tests label generation
func TestShipmentGenerateLabel(t *testing.T) {
	tests := []struct {
		name        string
		setupShipment func() *Shipment
		label       ShippingLabel
		expectError error
	}{
		{
			name: "Generate label for pending shipment",
			setupShipment: func() *Shipment {
				return NewShipment("SHIP-001", "ORD-001", "PKG-001", "WAVE-001",
					createTestCarrier(), createTestPackageInfo(),
					createTestAddress("John Doe"), createTestAddress("Warehouse A"))
			},
			label:       createTestLabel(),
			expectError: nil,
		},
		{
			name: "Cannot generate label for shipped shipment",
			setupShipment: func() *Shipment {
				shipment := NewShipment("SHIP-002", "ORD-002", "PKG-002", "WAVE-001",
					createTestCarrier(), createTestPackageInfo(),
					createTestAddress("Jane Smith"), createTestAddress("Warehouse A"))
				shipment.Status = ShipmentStatusShipped
				return shipment
			},
			label:       createTestLabel(),
			expectError: ErrShipmentAlreadyShipped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shipment := tt.setupShipment()
			err := shipment.GenerateLabel(tt.label)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, ShipmentStatusLabeled, shipment.Status)
				assert.NotNil(t, shipment.Label)
				assert.Equal(t, tt.label.TrackingNumber, shipment.Label.TrackingNumber)
				assert.NotNil(t, shipment.LabeledAt)

				// Check domain event
				events := shipment.GetDomainEvents()
				assert.GreaterOrEqual(t, len(events), 2) // Created + Labeled
				lastEvent, ok := events[len(events)-1].(*LabelGeneratedEvent)
				assert.True(t, ok)
				assert.Equal(t, tt.label.TrackingNumber, lastEvent.TrackingNumber)
			}
		})
	}
}

// TestShipmentAddToManifest tests adding shipment to manifest
func TestShipmentAddToManifest(t *testing.T) {
	tests := []struct {
		name        string
		setupShipment func() *Shipment
		manifest    Manifest
		expectError error
	}{
		{
			name: "Add labeled shipment to manifest",
			setupShipment: func() *Shipment {
				shipment := NewShipment("SHIP-001", "ORD-001", "PKG-001", "WAVE-001",
					createTestCarrier(), createTestPackageInfo(),
					createTestAddress("John Doe"), createTestAddress("Warehouse A"))
				shipment.GenerateLabel(createTestLabel())
				return shipment
			},
			manifest:    createTestManifest(),
			expectError: nil,
		},
		{
			name: "Cannot add shipment without label",
			setupShipment: func() *Shipment {
				return NewShipment("SHIP-002", "ORD-002", "PKG-002", "WAVE-001",
					createTestCarrier(), createTestPackageInfo(),
					createTestAddress("Jane Smith"), createTestAddress("Warehouse A"))
			},
			manifest:    createTestManifest(),
			expectError: ErrNoLabel,
		},
		{
			name: "Cannot add already manifested shipment",
			setupShipment: func() *Shipment {
				shipment := NewShipment("SHIP-003", "ORD-003", "PKG-003", "WAVE-001",
					createTestCarrier(), createTestPackageInfo(),
					createTestAddress("Bob Johnson"), createTestAddress("Warehouse A"))
				shipment.GenerateLabel(createTestLabel())
				shipment.AddToManifest(createTestManifest())
				return shipment
			},
			manifest:    createTestManifest(),
			expectError: ErrShipmentAlreadyManifested,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shipment := tt.setupShipment()
			err := shipment.AddToManifest(tt.manifest)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, ShipmentStatusManifested, shipment.Status)
				assert.NotNil(t, shipment.Manifest)
				assert.Equal(t, tt.manifest.ManifestID, shipment.Manifest.ManifestID)
				assert.NotNil(t, shipment.ManifestedAt)
			}
		})
	}
}

// TestShipmentConfirmShipment tests confirming shipment
func TestShipmentConfirmShipment(t *testing.T) {
	estimatedDelivery := time.Now().Add(48 * time.Hour)

	tests := []struct {
		name              string
		setupShipment     func() *Shipment
		estimatedDelivery *time.Time
		expectError       error
	}{
		{
			name: "Confirm manifested shipment",
			setupShipment: func() *Shipment {
				shipment := NewShipment("SHIP-001", "ORD-001", "PKG-001", "WAVE-001",
					createTestCarrier(), createTestPackageInfo(),
					createTestAddress("John Doe"), createTestAddress("Warehouse A"))
				shipment.GenerateLabel(createTestLabel())
				shipment.AddToManifest(createTestManifest())
				return shipment
			},
			estimatedDelivery: &estimatedDelivery,
			expectError:       nil,
		},
		{
			name: "Cannot confirm already shipped shipment",
			setupShipment: func() *Shipment {
				shipment := NewShipment("SHIP-002", "ORD-002", "PKG-002", "WAVE-001",
					createTestCarrier(), createTestPackageInfo(),
					createTestAddress("Jane Smith"), createTestAddress("Warehouse A"))
				shipment.Status = ShipmentStatusShipped
				return shipment
			},
			estimatedDelivery: &estimatedDelivery,
			expectError:       ErrShipmentAlreadyShipped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shipment := tt.setupShipment()
			err := shipment.ConfirmShipment(tt.estimatedDelivery)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, ShipmentStatusShipped, shipment.Status)
				assert.NotNil(t, shipment.ShippedAt)
				assert.Equal(t, tt.estimatedDelivery, shipment.EstimatedDelivery)

				// Check domain event
				events := shipment.GetDomainEvents()
				assert.GreaterOrEqual(t, len(events), 4) // Created + Labeled + Manifested + Shipped
				lastEvent, ok := events[len(events)-1].(*ShipConfirmedEvent)
				assert.True(t, ok)
				assert.Equal(t, shipment.ShipmentID, lastEvent.ShipmentID)
			}
		})
	}
}

// TestShipmentConfirmDelivery tests confirming delivery
func TestShipmentConfirmDelivery(t *testing.T) {
	shipment := NewShipment("SHIP-001", "ORD-001", "PKG-001", "WAVE-001",
		createTestCarrier(), createTestPackageInfo(),
		createTestAddress("John Doe"), createTestAddress("Warehouse A"))
	shipment.GenerateLabel(createTestLabel())
	shipment.AddToManifest(createTestManifest())
	estimatedDelivery := time.Now().Add(48 * time.Hour)
	shipment.ConfirmShipment(&estimatedDelivery)

	deliveredAt := time.Now()
	err := shipment.ConfirmDelivery(deliveredAt)

	assert.NoError(t, err)
	assert.Equal(t, ShipmentStatusDelivered, shipment.Status)
	assert.NotNil(t, shipment.ActualDelivery)
	assert.Equal(t, deliveredAt, *shipment.ActualDelivery)
}

// TestShipmentCancel tests shipment cancellation
func TestShipmentCancel(t *testing.T) {
	tests := []struct {
		name        string
		setupShipment func() *Shipment
		reason      string
		expectError bool
	}{
		{
			name: "Cancel pending shipment",
			setupShipment: func() *Shipment {
				return NewShipment("SHIP-001", "ORD-001", "PKG-001", "WAVE-001",
					createTestCarrier(), createTestPackageInfo(),
					createTestAddress("John Doe"), createTestAddress("Warehouse A"))
			},
			reason:      "Order cancelled by customer",
			expectError: false,
		},
		{
			name: "Cancel labeled shipment",
			setupShipment: func() *Shipment {
				shipment := NewShipment("SHIP-002", "ORD-002", "PKG-002", "WAVE-001",
					createTestCarrier(), createTestPackageInfo(),
					createTestAddress("Jane Smith"), createTestAddress("Warehouse A"))
				shipment.GenerateLabel(createTestLabel())
				return shipment
			},
			reason:      "Address incorrect",
			expectError: false,
		},
		{
			name: "Cannot cancel shipped shipment",
			setupShipment: func() *Shipment {
				shipment := NewShipment("SHIP-003", "ORD-003", "PKG-003", "WAVE-001",
					createTestCarrier(), createTestPackageInfo(),
					createTestAddress("Bob Johnson"), createTestAddress("Warehouse A"))
				shipment.Status = ShipmentStatusShipped
				return shipment
			},
			reason:      "Too late",
			expectError: true,
		},
		{
			name: "Cannot cancel delivered shipment",
			setupShipment: func() *Shipment {
				shipment := NewShipment("SHIP-004", "ORD-004", "PKG-004", "WAVE-001",
					createTestCarrier(), createTestPackageInfo(),
					createTestAddress("Alice Brown"), createTestAddress("Warehouse A"))
				shipment.Status = ShipmentStatusDelivered
				return shipment
			},
			reason:      "Too late",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shipment := tt.setupShipment()
			err := shipment.Cancel(tt.reason)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot cancel")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, ShipmentStatusCancelled, shipment.Status)
			}
		})
	}
}

// TestShipmentWorkflow tests complete shipment workflow
func TestShipmentWorkflow(t *testing.T) {
	// Create shipment
	shipment := NewShipment("SHIP-001", "ORD-001", "PKG-001", "WAVE-001",
		createTestCarrier(), createTestPackageInfo(),
		createTestAddress("John Doe"), createTestAddress("Warehouse A"))
	assert.Equal(t, ShipmentStatusPending, shipment.Status)

	// Generate label
	err := shipment.GenerateLabel(createTestLabel())
	assert.NoError(t, err)
	assert.Equal(t, ShipmentStatusLabeled, shipment.Status)

	// Add to manifest
	err = shipment.AddToManifest(createTestManifest())
	assert.NoError(t, err)
	assert.Equal(t, ShipmentStatusManifested, shipment.Status)

	// Confirm shipment
	estimatedDelivery := time.Now().Add(48 * time.Hour)
	err = shipment.ConfirmShipment(&estimatedDelivery)
	assert.NoError(t, err)
	assert.Equal(t, ShipmentStatusShipped, shipment.Status)

	// Confirm delivery
	deliveredAt := time.Now().Add(36 * time.Hour)
	err = shipment.ConfirmDelivery(deliveredAt)
	assert.NoError(t, err)
	assert.Equal(t, ShipmentStatusDelivered, shipment.Status)

	// Verify all events generated
	events := shipment.GetDomainEvents()
	assert.Len(t, events, 4) // Created, Labeled, Manifested, Shipped
}

// TestShipmentDomainEvents tests domain event handling
func TestShipmentDomainEvents(t *testing.T) {
	shipment := NewShipment("SHIP-001", "ORD-001", "PKG-001", "WAVE-001",
		createTestCarrier(), createTestPackageInfo(),
		createTestAddress("John Doe"), createTestAddress("Warehouse A"))

	// Check initial event
	events := shipment.GetDomainEvents()
	assert.Len(t, events, 1)
	_, ok := events[0].(*ShipmentCreatedEvent)
	assert.True(t, ok)

	// Generate label
	shipment.GenerateLabel(createTestLabel())
	events = shipment.GetDomainEvents()
	assert.Len(t, events, 2)

	// Clear events
	shipment.ClearDomainEvents()
	events = shipment.GetDomainEvents()
	assert.Len(t, events, 0)
}

// BenchmarkNewShipment benchmarks shipment creation
func BenchmarkNewShipment(b *testing.B) {
	carrier := createTestCarrier()
	pkg := createTestPackageInfo()
	recipient := createTestAddress("John Doe")
	shipper := createTestAddress("Warehouse A")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewShipment("SHIP-001", "ORD-001", "PKG-001", "WAVE-001", carrier, pkg, recipient, shipper)
	}
}

// BenchmarkGenerateLabel benchmarks label generation
func BenchmarkGenerateLabel(b *testing.B) {
	shipment := NewShipment("SHIP-001", "ORD-001", "PKG-001", "WAVE-001",
		createTestCarrier(), createTestPackageInfo(),
		createTestAddress("John Doe"), createTestAddress("Warehouse A"))
	label := createTestLabel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset status for benchmark
		shipment.Status = ShipmentStatusPending
		shipment.GenerateLabel(label)
	}
}
