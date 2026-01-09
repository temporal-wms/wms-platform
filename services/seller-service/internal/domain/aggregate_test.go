package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSeller tests seller creation
func TestNewSeller(t *testing.T) {
	tests := []struct {
		name         string
		tenantID     string
		companyName  string
		contactName  string
		contactEmail string
		billingCycle BillingCycle
		expectError  bool
	}{
		{
			name:         "Valid seller creation",
			tenantID:     "TNT-001",
			companyName:  "Acme Corp",
			contactName:  "John Smith",
			contactEmail: "john@acme.com",
			billingCycle: BillingCycleMonthly,
			expectError:  false,
		},
		{
			name:         "Invalid billing cycle",
			tenantID:     "TNT-001",
			companyName:  "Acme Corp",
			contactName:  "John Smith",
			contactEmail: "john@acme.com",
			billingCycle: BillingCycle("invalid"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seller, err := NewSeller(tt.tenantID, tt.companyName, tt.contactName, tt.contactEmail, tt.billingCycle)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, seller)
			} else {
				require.NoError(t, err)
				require.NotNil(t, seller)
				assert.NotEmpty(t, seller.SellerID)
				assert.Equal(t, tt.tenantID, seller.TenantID)
				assert.Equal(t, tt.companyName, seller.CompanyName)
				assert.Equal(t, tt.contactName, seller.ContactName)
				assert.Equal(t, tt.contactEmail, seller.ContactEmail)
				assert.Equal(t, SellerStatusPending, seller.Status)
				assert.Equal(t, tt.billingCycle, seller.BillingCycle)
				assert.NotNil(t, seller.FeeSchedule)
				assert.NotZero(t, seller.CreatedAt)

				// Should have domain event
				events := seller.DomainEvents()
				assert.Len(t, events, 1)
			}
		})
	}
}

// TestSellerActivate tests seller activation
func TestSellerActivate(t *testing.T) {
	tests := []struct {
		name        string
		setupSeller func() *Seller
		expectError bool
	}{
		{
			name: "Activate pending seller",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				return seller
			},
			expectError: false,
		},
		{
			name: "Activate suspended seller",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				seller.Status = SellerStatusSuspended
				return seller
			},
			expectError: false,
		},
		{
			name: "Cannot activate active seller",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				seller.Status = SellerStatusActive
				return seller
			},
			expectError: true,
		},
		{
			name: "Cannot activate closed seller",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				seller.Status = SellerStatusClosed
				return seller
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seller := tt.setupSeller()
			seller.ClearDomainEvents()
			err := seller.Activate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, SellerStatusActive, seller.Status)
				assert.Len(t, seller.DomainEvents(), 1)
			}
		})
	}
}

// TestSellerSuspend tests seller suspension
func TestSellerSuspend(t *testing.T) {
	tests := []struct {
		name        string
		setupSeller func() *Seller
		expectError bool
	}{
		{
			name: "Suspend active seller",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				seller.Status = SellerStatusActive
				return seller
			},
			expectError: false,
		},
		{
			name: "Cannot suspend pending seller",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				return seller
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seller := tt.setupSeller()
			seller.ClearDomainEvents()
			err := seller.Suspend("Payment overdue")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, SellerStatusSuspended, seller.Status)
				assert.Len(t, seller.DomainEvents(), 1)
			}
		})
	}
}

// TestSellerClose tests seller account closure
func TestSellerClose(t *testing.T) {
	seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
	seller.ClearDomainEvents()

	err := seller.Close("Contract ended")
	assert.NoError(t, err)
	assert.Equal(t, SellerStatusClosed, seller.Status)
	assert.Len(t, seller.DomainEvents(), 1)

	// Closing again should be idempotent
	err = seller.Close("Again")
	assert.NoError(t, err)
}

// TestSellerAssignFacility tests facility assignment
func TestSellerAssignFacility(t *testing.T) {
	tests := []struct {
		name        string
		setupSeller func() *Seller
		facilityID  string
		isDefault   bool
		expectError error
	}{
		{
			name: "Assign facility to active seller",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				seller.Status = SellerStatusActive
				return seller
			},
			facilityID:  "FAC-001",
			isDefault:   true,
			expectError: nil,
		},
		{
			name: "Assign facility to pending seller",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				return seller
			},
			facilityID:  "FAC-001",
			isDefault:   true,
			expectError: nil,
		},
		{
			name: "Cannot assign to suspended seller",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				seller.Status = SellerStatusSuspended
				return seller
			},
			facilityID:  "FAC-001",
			isDefault:   true,
			expectError: ErrSellerNotActive,
		},
		{
			name: "Cannot assign duplicate facility",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				seller.Status = SellerStatusActive
				seller.AssignFacility("FAC-001", "East DC", []string{"WH-001"}, 1000, true)
				return seller
			},
			facilityID:  "FAC-001",
			isDefault:   false,
			expectError: ErrFacilityAlreadyAssigned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seller := tt.setupSeller()
			seller.ClearDomainEvents()
			err := seller.AssignFacility(tt.facilityID, "Test Facility", []string{"WH-001"}, 1000, tt.isDefault)

			if tt.expectError != nil {
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, seller.DomainEvents(), 1)

				// Verify facility is assigned
				found := false
				for _, f := range seller.AssignedFacilities {
					if f.FacilityID == tt.facilityID {
						found = true
						assert.Equal(t, tt.isDefault, f.IsDefault)
					}
				}
				assert.True(t, found)
			}
		})
	}
}

// TestSellerRemoveFacility tests facility removal
func TestSellerRemoveFacility(t *testing.T) {
	seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
	seller.Status = SellerStatusActive
	seller.AssignFacility("FAC-001", "East DC", []string{"WH-001"}, 1000, true)

	// Remove existing facility
	err := seller.RemoveFacility("FAC-001")
	assert.NoError(t, err)
	assert.Empty(t, seller.AssignedFacilities)

	// Remove non-existent facility
	err = seller.RemoveFacility("FAC-999")
	assert.Error(t, err)
}

// TestSellerGetDefaultFacility tests getting default facility
func TestSellerGetDefaultFacility(t *testing.T) {
	seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
	seller.Status = SellerStatusActive

	// No facilities
	assert.Nil(t, seller.GetDefaultFacility())

	// Add non-default facility
	seller.AssignFacility("FAC-001", "East DC", []string{"WH-001"}, 1000, false)
	def := seller.GetDefaultFacility()
	assert.NotNil(t, def)
	assert.Equal(t, "FAC-001", def.FacilityID)

	// Add default facility
	seller.AssignFacility("FAC-002", "West DC", []string{"WH-002"}, 2000, true)
	def = seller.GetDefaultFacility()
	assert.NotNil(t, def)
	assert.Equal(t, "FAC-002", def.FacilityID)
}

// TestSellerAddChannelIntegration tests channel integration
func TestSellerAddChannelIntegration(t *testing.T) {
	tests := []struct {
		name        string
		setupSeller func() *Seller
		channelType string
		storeName   string
		expectError error
	}{
		{
			name: "Add integration to active seller",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				seller.Status = SellerStatusActive
				return seller
			},
			channelType: "shopify",
			storeName:   "My Store",
			expectError: nil,
		},
		{
			name: "Cannot add to inactive seller",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				return seller
			},
			channelType: "shopify",
			storeName:   "My Store",
			expectError: ErrSellerNotActive,
		},
		{
			name: "Cannot add duplicate channel",
			setupSeller: func() *Seller {
				seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
				seller.Status = SellerStatusActive
				seller.AddChannelIntegration("shopify", "My Store", "", nil, ChannelSyncSettings{})
				return seller
			},
			channelType: "shopify",
			storeName:   "My Store",
			expectError: ErrChannelAlreadyConnected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seller := tt.setupSeller()
			seller.ClearDomainEvents()
			err := seller.AddChannelIntegration(tt.channelType, tt.storeName, "", nil, ChannelSyncSettings{})

			if tt.expectError != nil {
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, seller.DomainEvents(), 1)
			}
		})
	}
}

// TestSellerDisconnectChannel tests channel disconnection
func TestSellerDisconnectChannel(t *testing.T) {
	seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
	seller.Status = SellerStatusActive
	seller.AddChannelIntegration("shopify", "My Store", "", nil, ChannelSyncSettings{})

	channelID := seller.Integrations[0].ChannelID

	err := seller.DisconnectChannel(channelID)
	assert.NoError(t, err)
	assert.Equal(t, "disconnected", seller.Integrations[0].Status)

	// Disconnect non-existent
	err = seller.DisconnectChannel("CH-999")
	assert.Error(t, err)
}

// TestSellerGenerateAPIKey tests API key generation
func TestSellerGenerateAPIKey(t *testing.T) {
	seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)

	// Cannot generate for inactive seller
	_, _, err := seller.GenerateAPIKey("Test Key", []string{"orders:read"}, nil)
	assert.Equal(t, ErrSellerNotActive, err)

	// Activate and generate
	seller.Status = SellerStatusActive
	key, rawKey, err := seller.GenerateAPIKey("Test Key", []string{"orders:read"}, nil)
	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.NotEmpty(t, rawKey)
	assert.Equal(t, "Test Key", key.Name)
	assert.Equal(t, []string{"orders:read"}, key.Scopes)
	assert.Len(t, seller.APIKeys, 1)
}

// TestSellerRevokeAPIKey tests API key revocation
func TestSellerRevokeAPIKey(t *testing.T) {
	seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
	seller.Status = SellerStatusActive
	key, _, _ := seller.GenerateAPIKey("Test Key", []string{"orders:read"}, nil)

	// Revoke existing key
	err := seller.RevokeAPIKey(key.KeyID)
	assert.NoError(t, err)
	assert.NotNil(t, seller.APIKeys[0].RevokedAt)

	// Revoke again (idempotent)
	err = seller.RevokeAPIKey(key.KeyID)
	assert.NoError(t, err)

	// Revoke non-existent
	err = seller.RevokeAPIKey("key-999")
	assert.Equal(t, ErrAPIKeyNotFound, err)
}

// TestSellerValidateAPIKey tests API key validation
func TestSellerValidateAPIKey(t *testing.T) {
	seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
	seller.Status = SellerStatusActive
	key, rawKey, _ := seller.GenerateAPIKey("Test Key", []string{"orders:read"}, nil)

	// Valid key
	validatedKey, err := seller.ValidateAPIKey(rawKey)
	assert.NoError(t, err)
	assert.NotNil(t, validatedKey)
	assert.Equal(t, key.KeyID, validatedKey.KeyID)

	// Invalid key
	_, err = seller.ValidateAPIKey("invalid-key")
	assert.Equal(t, ErrAPIKeyNotFound, err)

	// Revoked key
	seller.RevokeAPIKey(key.KeyID)
	_, err = seller.ValidateAPIKey(rawKey)
	assert.Equal(t, ErrAPIKeyRevoked, err)
}

// TestSellerValidateExpiredAPIKey tests expired API key
func TestSellerValidateExpiredAPIKey(t *testing.T) {
	seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
	seller.Status = SellerStatusActive

	// Generate expired key
	expiry := time.Now().Add(-1 * time.Hour)
	_, rawKey, _ := seller.GenerateAPIKey("Expired Key", []string{"orders:read"}, &expiry)

	_, err := seller.ValidateAPIKey(rawKey)
	assert.Equal(t, ErrAPIKeyExpired, err)
}

// TestSellerGetActiveAPIKeys tests getting active keys
func TestSellerGetActiveAPIKeys(t *testing.T) {
	seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
	seller.Status = SellerStatusActive

	key1, _, _ := seller.GenerateAPIKey("Key 1", []string{"orders:read"}, nil)
	seller.GenerateAPIKey("Key 2", []string{"inventory:read"}, nil)

	// Revoke one
	seller.RevokeAPIKey(key1.KeyID)

	activeKeys := seller.GetActiveAPIKeys()
	assert.Len(t, activeKeys, 1)
	assert.Equal(t, "Key 2", activeKeys[0].Name)
	assert.Empty(t, activeKeys[0].HashedKey) // Should not expose hashed key
}

// TestBillingCycleIsValid tests billing cycle validation
func TestBillingCycleIsValid(t *testing.T) {
	assert.True(t, BillingCycleDaily.IsValid())
	assert.True(t, BillingCycleWeekly.IsValid())
	assert.True(t, BillingCycleMonthly.IsValid())
	assert.False(t, BillingCycle("invalid").IsValid())
}

// TestSellerStatusIsValid tests seller status validation
func TestSellerStatusIsValid(t *testing.T) {
	assert.True(t, SellerStatusPending.IsValid())
	assert.True(t, SellerStatusActive.IsValid())
	assert.True(t, SellerStatusSuspended.IsValid())
	assert.True(t, SellerStatusClosed.IsValid())
	assert.False(t, SellerStatus("invalid").IsValid())
}

// TestDefaultFeeSchedule tests default fee schedule
func TestDefaultFeeSchedule(t *testing.T) {
	schedule := DefaultFeeSchedule()

	assert.NotNil(t, schedule)
	assert.Equal(t, 0.05, schedule.StorageFeePerCubicFtPerDay)
	assert.Equal(t, 0.25, schedule.PickFeePerUnit)
	assert.Equal(t, 1.50, schedule.PackFeePerOrder)
	assert.Equal(t, 0.15, schedule.ReceivingFeePerUnit)
	assert.NotZero(t, schedule.EffectiveFrom)
}

// TestAPIKeyIsRevoked tests revoked check
func TestAPIKeyIsRevoked(t *testing.T) {
	key := APIKey{KeyID: "test"}
	assert.False(t, key.IsRevoked())

	now := time.Now()
	key.RevokedAt = &now
	assert.True(t, key.IsRevoked())
}

// TestAPIKeyIsExpired tests expiry check
func TestAPIKeyIsExpired(t *testing.T) {
	key := APIKey{KeyID: "test"}
	assert.False(t, key.IsExpired()) // No expiry set

	future := time.Now().Add(24 * time.Hour)
	key.ExpiresAt = &future
	assert.False(t, key.IsExpired())

	past := time.Now().Add(-24 * time.Hour)
	key.ExpiresAt = &past
	assert.True(t, key.IsExpired())
}

// TestSellerDomainEvents tests domain event handling
func TestSellerDomainEvents(t *testing.T) {
	seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)

	// Should have creation event
	events := seller.DomainEvents()
	assert.Len(t, events, 1)

	// Clear events
	seller.ClearDomainEvents()
	events = seller.DomainEvents()
	assert.Len(t, events, 0)
}

// BenchmarkNewSeller benchmarks seller creation
func BenchmarkNewSeller(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
	}
}

// BenchmarkGenerateAPIKey benchmarks API key generation
func BenchmarkGenerateAPIKey(b *testing.B) {
	seller, _ := NewSeller("TNT-001", "Acme Corp", "John", "john@acme.com", BillingCycleMonthly)
	seller.Status = SellerStatusActive

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		seller.GenerateAPIKey("Test", []string{"orders:read"}, nil)
	}
}
