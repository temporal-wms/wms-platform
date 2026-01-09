package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors for Seller aggregate
var (
	ErrSellerNotActive       = errors.New("seller account is not active")
	ErrSellerNotFound        = errors.New("seller not found")
	ErrInvalidBillingCycle   = errors.New("invalid billing cycle")
	ErrContractExpired       = errors.New("seller contract has expired")
	ErrFacilityAlreadyAssigned = errors.New("facility already assigned to seller")
	ErrChannelAlreadyConnected = errors.New("channel already connected")
	ErrAPIKeyNotFound        = errors.New("API key not found")
	ErrAPIKeyRevoked         = errors.New("API key has been revoked")
	ErrAPIKeyExpired         = errors.New("API key has expired")
)

// SellerStatus represents seller account status
type SellerStatus string

const (
	SellerStatusPending   SellerStatus = "pending"
	SellerStatusActive    SellerStatus = "active"
	SellerStatusSuspended SellerStatus = "suspended"
	SellerStatusClosed    SellerStatus = "closed"
)

// IsValid checks if the status is valid
func (s SellerStatus) IsValid() bool {
	switch s {
	case SellerStatusPending, SellerStatusActive, SellerStatusSuspended, SellerStatusClosed:
		return true
	}
	return false
}

// BillingCycle represents the billing frequency
type BillingCycle string

const (
	BillingCycleDaily   BillingCycle = "daily"
	BillingCycleWeekly  BillingCycle = "weekly"
	BillingCycleMonthly BillingCycle = "monthly"
)

// IsValid checks if the billing cycle is valid
func (b BillingCycle) IsValid() bool {
	switch b {
	case BillingCycleDaily, BillingCycleWeekly, BillingCycleMonthly:
		return true
	}
	return false
}

// Seller is the aggregate root for seller/merchant management
type Seller struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SellerID     string             `bson:"sellerId" json:"sellerId"`
	TenantID     string             `bson:"tenantId" json:"tenantId"` // 3PL operator
	CompanyName  string             `bson:"companyName" json:"companyName"`
	ContactName  string             `bson:"contactName" json:"contactName"`
	ContactEmail string             `bson:"contactEmail" json:"contactEmail"`
	ContactPhone string             `bson:"contactPhone" json:"contactPhone"`
	Status       SellerStatus       `bson:"status" json:"status"`

	// Contract details
	ContractStartDate time.Time    `bson:"contractStartDate" json:"contractStartDate"`
	ContractEndDate   *time.Time   `bson:"contractEndDate,omitempty" json:"contractEndDate,omitempty"`
	BillingCycle      BillingCycle `bson:"billingCycle" json:"billingCycle"`

	// Warehouse assignments
	AssignedFacilities []FacilityAssignment `bson:"assignedFacilities" json:"assignedFacilities"`

	// Fee schedule
	FeeSchedule *FeeSchedule `bson:"feeSchedule" json:"feeSchedule"`

	// Channel integrations (Shopify, Amazon, etc.)
	Integrations []ChannelIntegration `bson:"integrations" json:"integrations"`

	// API access
	APIKeys []APIKey `bson:"apiKeys" json:"-"` // Don't expose in JSON

	// Metadata
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`

	// Domain events (not persisted)
	domainEvents []DomainEvent `bson:"-" json:"-"`
}

// FacilityAssignment represents a seller's access to a facility
type FacilityAssignment struct {
	FacilityID     string    `bson:"facilityId" json:"facilityId"`
	FacilityName   string    `bson:"facilityName" json:"facilityName"`
	WarehouseIDs   []string  `bson:"warehouseIds" json:"warehouseIds"`
	AllocatedSpace float64   `bson:"allocatedSpace" json:"allocatedSpace"` // in sq ft
	AssignedAt     time.Time `bson:"assignedAt" json:"assignedAt"`
	IsDefault      bool      `bson:"isDefault" json:"isDefault"`
}

// FeeSchedule defines the pricing for a seller
type FeeSchedule struct {
	// Storage fees
	StorageFeePerCubicFtPerDay float64 `bson:"storageFeePerCubicFtPerDay" json:"storageFeePerCubicFtPerDay"`

	// Fulfillment fees
	PickFeePerUnit    float64 `bson:"pickFeePerUnit" json:"pickFeePerUnit"`
	PackFeePerOrder   float64 `bson:"packFeePerOrder" json:"packFeePerOrder"`
	ReceivingFeePerUnit float64 `bson:"receivingFeePerUnit" json:"receivingFeePerUnit"`

	// Shipping
	ShippingMarkupPercent float64 `bson:"shippingMarkupPercent" json:"shippingMarkupPercent"`

	// Returns
	ReturnProcessingFee float64 `bson:"returnProcessingFee" json:"returnProcessingFee"`

	// Special handling fees
	GiftWrapFee         float64 `bson:"giftWrapFee" json:"giftWrapFee"`
	HazmatHandlingFee   float64 `bson:"hazmatHandlingFee" json:"hazmatHandlingFee"`
	OversizedItemFee    float64 `bson:"oversizedItemFee" json:"oversizedItemFee"`
	ColdChainFeePerUnit float64 `bson:"coldChainFeePerUnit" json:"coldChainFeePerUnit"`
	FragileHandlingFee  float64 `bson:"fragileHandlingFee" json:"fragileHandlingFee"`

	// Volume discounts
	VolumeDiscounts []VolumeDiscount `bson:"volumeDiscounts" json:"volumeDiscounts"`

	// Effective dates
	EffectiveFrom time.Time  `bson:"effectiveFrom" json:"effectiveFrom"`
	EffectiveTo   *time.Time `bson:"effectiveTo,omitempty" json:"effectiveTo,omitempty"`
}

// VolumeDiscount represents tiered pricing based on volume
type VolumeDiscount struct {
	MinUnits        int     `bson:"minUnits" json:"minUnits"`
	MaxUnits        int     `bson:"maxUnits" json:"maxUnits"`
	DiscountPercent float64 `bson:"discountPercent" json:"discountPercent"`
}

// ChannelIntegration represents an external sales channel connection
type ChannelIntegration struct {
	ChannelID    string            `bson:"channelId" json:"channelId"`
	ChannelType  string            `bson:"channelType" json:"channelType"` // shopify, amazon, ebay, woocommerce
	StoreName    string            `bson:"storeName" json:"storeName"`
	StoreURL     string            `bson:"storeUrl,omitempty" json:"storeUrl,omitempty"`
	Status       string            `bson:"status" json:"status"` // active, paused, disconnected
	Credentials  map[string]string `bson:"credentials" json:"-"` // encrypted, never expose
	SyncSettings ChannelSyncSettings `bson:"syncSettings" json:"syncSettings"`
	ConnectedAt  time.Time         `bson:"connectedAt" json:"connectedAt"`
	LastSyncAt   *time.Time        `bson:"lastSyncAt,omitempty" json:"lastSyncAt,omitempty"`
}

// ChannelSyncSettings defines how to sync with external channel
type ChannelSyncSettings struct {
	AutoImportOrders     bool `bson:"autoImportOrders" json:"autoImportOrders"`
	AutoSyncInventory    bool `bson:"autoSyncInventory" json:"autoSyncInventory"`
	AutoPushTracking     bool `bson:"autoPushTracking" json:"autoPushTracking"`
	InventorySyncMinutes int  `bson:"inventorySyncMinutes" json:"inventorySyncMinutes"`
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	KeyID      string     `bson:"keyId" json:"keyId"`
	Name       string     `bson:"name" json:"name"`
	Prefix     string     `bson:"prefix" json:"prefix"`     // First 8 chars for identification
	HashedKey  string     `bson:"hashedKey" json:"-"`       // Never exposed
	Scopes     []string   `bson:"scopes" json:"scopes"`     // orders:read, inventory:write, etc.
	ExpiresAt  *time.Time `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	LastUsedAt *time.Time `bson:"lastUsedAt,omitempty" json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time  `bson:"createdAt" json:"createdAt"`
	RevokedAt  *time.Time `bson:"revokedAt,omitempty" json:"revokedAt,omitempty"`
}

// IsRevoked checks if the API key has been revoked
func (k *APIKey) IsRevoked() bool {
	return k.RevokedAt != nil
}

// IsExpired checks if the API key has expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// NewSeller creates a new Seller aggregate
func NewSeller(tenantID, companyName, contactName, contactEmail string, billingCycle BillingCycle) (*Seller, error) {
	if !billingCycle.IsValid() {
		return nil, ErrInvalidBillingCycle
	}

	now := time.Now().UTC()
	sellerID := fmt.Sprintf("SLR-%s", uuid.New().String()[:8])

	seller := &Seller{
		ID:                 primitive.NewObjectID(),
		SellerID:           sellerID,
		TenantID:           tenantID,
		CompanyName:        companyName,
		ContactName:        contactName,
		ContactEmail:       contactEmail,
		Status:             SellerStatusPending,
		ContractStartDate:  now,
		BillingCycle:       billingCycle,
		AssignedFacilities: make([]FacilityAssignment, 0),
		FeeSchedule:        DefaultFeeSchedule(),
		Integrations:       make([]ChannelIntegration, 0),
		APIKeys:            make([]APIKey, 0),
		CreatedAt:          now,
		UpdatedAt:          now,
		domainEvents:       make([]DomainEvent, 0),
	}

	seller.addDomainEvent(&SellerCreatedEvent{
		SellerID:    sellerID,
		TenantID:    tenantID,
		CompanyName: companyName,
		CreatedAt:   now,
	})

	return seller, nil
}

// DefaultFeeSchedule returns a default fee schedule
func DefaultFeeSchedule() *FeeSchedule {
	return &FeeSchedule{
		StorageFeePerCubicFtPerDay: 0.05,
		PickFeePerUnit:             0.25,
		PackFeePerOrder:            1.50,
		ReceivingFeePerUnit:        0.15,
		ShippingMarkupPercent:      5.0,
		ReturnProcessingFee:        3.00,
		GiftWrapFee:                2.50,
		HazmatHandlingFee:          5.00,
		OversizedItemFee:           10.00,
		ColdChainFeePerUnit:        1.00,
		FragileHandlingFee:         1.50,
		VolumeDiscounts:            make([]VolumeDiscount, 0),
		EffectiveFrom:              time.Now().UTC(),
	}
}

// Activate activates the seller account
func (s *Seller) Activate() error {
	if s.Status != SellerStatusPending && s.Status != SellerStatusSuspended {
		return errors.New("can only activate pending or suspended sellers")
	}

	s.Status = SellerStatusActive
	s.UpdatedAt = time.Now().UTC()

	s.addDomainEvent(&SellerActivatedEvent{
		SellerID:    s.SellerID,
		ActivatedAt: s.UpdatedAt,
	})

	return nil
}

// Suspend suspends the seller account
func (s *Seller) Suspend(reason string) error {
	if s.Status != SellerStatusActive {
		return errors.New("can only suspend active sellers")
	}

	s.Status = SellerStatusSuspended
	s.UpdatedAt = time.Now().UTC()

	s.addDomainEvent(&SellerSuspendedEvent{
		SellerID:    s.SellerID,
		Reason:      reason,
		SuspendedAt: s.UpdatedAt,
	})

	return nil
}

// Close closes the seller account
func (s *Seller) Close(reason string) error {
	if s.Status == SellerStatusClosed {
		return nil // Already closed
	}

	s.Status = SellerStatusClosed
	s.UpdatedAt = time.Now().UTC()

	s.addDomainEvent(&SellerClosedEvent{
		SellerID: s.SellerID,
		Reason:   reason,
		ClosedAt: s.UpdatedAt,
	})

	return nil
}

// AssignFacility assigns a facility to the seller
func (s *Seller) AssignFacility(facilityID, facilityName string, warehouseIDs []string, allocatedSpace float64, isDefault bool) error {
	if s.Status != SellerStatusActive && s.Status != SellerStatusPending {
		return ErrSellerNotActive
	}

	// Check if already assigned
	for _, f := range s.AssignedFacilities {
		if f.FacilityID == facilityID {
			return ErrFacilityAlreadyAssigned
		}
	}

	// If this is the default, unset other defaults
	if isDefault {
		for i := range s.AssignedFacilities {
			s.AssignedFacilities[i].IsDefault = false
		}
	}

	s.AssignedFacilities = append(s.AssignedFacilities, FacilityAssignment{
		FacilityID:     facilityID,
		FacilityName:   facilityName,
		WarehouseIDs:   warehouseIDs,
		AllocatedSpace: allocatedSpace,
		AssignedAt:     time.Now().UTC(),
		IsDefault:      isDefault,
	})
	s.UpdatedAt = time.Now().UTC()

	s.addDomainEvent(&FacilityAssignedEvent{
		SellerID:   s.SellerID,
		FacilityID: facilityID,
		AssignedAt: s.UpdatedAt,
	})

	return nil
}

// RemoveFacility removes a facility assignment
func (s *Seller) RemoveFacility(facilityID string) error {
	for i, f := range s.AssignedFacilities {
		if f.FacilityID == facilityID {
			s.AssignedFacilities = append(s.AssignedFacilities[:i], s.AssignedFacilities[i+1:]...)
			s.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return errors.New("facility not assigned to seller")
}

// GetDefaultFacility returns the default facility assignment
func (s *Seller) GetDefaultFacility() *FacilityAssignment {
	for _, f := range s.AssignedFacilities {
		if f.IsDefault {
			return &f
		}
	}
	if len(s.AssignedFacilities) > 0 {
		return &s.AssignedFacilities[0]
	}
	return nil
}

// UpdateFeeSchedule updates the seller's fee schedule
func (s *Seller) UpdateFeeSchedule(feeSchedule *FeeSchedule) {
	s.FeeSchedule = feeSchedule
	s.UpdatedAt = time.Now().UTC()
}

// AddChannelIntegration adds a sales channel integration
func (s *Seller) AddChannelIntegration(channelType, storeName, storeURL string, credentials map[string]string, syncSettings ChannelSyncSettings) error {
	if s.Status != SellerStatusActive {
		return ErrSellerNotActive
	}

	// Check if channel already connected
	for _, ch := range s.Integrations {
		if ch.ChannelType == channelType && ch.StoreName == storeName {
			return ErrChannelAlreadyConnected
		}
	}

	channelID := fmt.Sprintf("CH-%s", uuid.New().String()[:8])

	s.Integrations = append(s.Integrations, ChannelIntegration{
		ChannelID:    channelID,
		ChannelType:  channelType,
		StoreName:    storeName,
		StoreURL:     storeURL,
		Status:       "active",
		Credentials:  credentials,
		SyncSettings: syncSettings,
		ConnectedAt:  time.Now().UTC(),
	})
	s.UpdatedAt = time.Now().UTC()

	s.addDomainEvent(&ChannelConnectedEvent{
		SellerID:    s.SellerID,
		ChannelID:   channelID,
		ChannelType: channelType,
		ConnectedAt: s.UpdatedAt,
	})

	return nil
}

// DisconnectChannel disconnects a sales channel
func (s *Seller) DisconnectChannel(channelID string) error {
	for i := range s.Integrations {
		if s.Integrations[i].ChannelID == channelID {
			s.Integrations[i].Status = "disconnected"
			s.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return errors.New("channel not found")
}

// UpdateChannelLastSync updates the last sync timestamp for a channel
func (s *Seller) UpdateChannelLastSync(channelID string) {
	for i := range s.Integrations {
		if s.Integrations[i].ChannelID == channelID {
			now := time.Now().UTC()
			s.Integrations[i].LastSyncAt = &now
			s.UpdatedAt = now
			return
		}
	}
}

// GenerateAPIKey generates a new API key for the seller
func (s *Seller) GenerateAPIKey(name string, scopes []string, expiresAt *time.Time) (*APIKey, string, error) {
	if s.Status != SellerStatusActive {
		return nil, "", ErrSellerNotActive
	}

	// Generate a secure random key
	rawKey := generateSecureKey(32)
	hashedKey := hashKey(rawKey)

	apiKey := APIKey{
		KeyID:     uuid.New().String(),
		Name:      name,
		Prefix:    rawKey[:8],
		HashedKey: hashedKey,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}

	s.APIKeys = append(s.APIKeys, apiKey)
	s.UpdatedAt = time.Now().UTC()

	// Return raw key only once - it won't be stored
	return &apiKey, rawKey, nil
}

// RevokeAPIKey revokes an API key
func (s *Seller) RevokeAPIKey(keyID string) error {
	for i := range s.APIKeys {
		if s.APIKeys[i].KeyID == keyID {
			if s.APIKeys[i].RevokedAt != nil {
				return nil // Already revoked
			}
			now := time.Now().UTC()
			s.APIKeys[i].RevokedAt = &now
			s.UpdatedAt = now
			return nil
		}
	}
	return ErrAPIKeyNotFound
}

// ValidateAPIKey validates an API key and returns the key info if valid
func (s *Seller) ValidateAPIKey(rawKey string) (*APIKey, error) {
	hashedKey := hashKey(rawKey)

	for i := range s.APIKeys {
		if s.APIKeys[i].HashedKey == hashedKey {
			if s.APIKeys[i].IsRevoked() {
				return nil, ErrAPIKeyRevoked
			}
			if s.APIKeys[i].IsExpired() {
				return nil, ErrAPIKeyExpired
			}

			// Update last used
			now := time.Now().UTC()
			s.APIKeys[i].LastUsedAt = &now
			s.UpdatedAt = now

			return &s.APIKeys[i], nil
		}
	}
	return nil, ErrAPIKeyNotFound
}

// GetActiveAPIKeys returns non-revoked API keys (without hashed keys)
func (s *Seller) GetActiveAPIKeys() []APIKey {
	active := make([]APIKey, 0)
	for _, k := range s.APIKeys {
		if !k.IsRevoked() {
			// Create a copy without the hashed key
			keyCopy := k
			keyCopy.HashedKey = ""
			active = append(active, keyCopy)
		}
	}
	return active
}

// Domain event helpers
func (s *Seller) addDomainEvent(event DomainEvent) {
	s.domainEvents = append(s.domainEvents, event)
}

// DomainEvents returns all pending domain events
func (s *Seller) DomainEvents() []DomainEvent {
	return s.domainEvents
}

// ClearDomainEvents clears all pending domain events
func (s *Seller) ClearDomainEvents() {
	s.domainEvents = make([]DomainEvent, 0)
}

// Helper functions
func generateSecureKey(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
