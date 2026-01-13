package application

import (
	"time"

	"github.com/wms-platform/services/seller-service/internal/domain"
)

// CreateSellerCommand represents the command to create a new seller
type CreateSellerCommand struct {
	TenantID     string `json:"tenantId" binding:"required"`
	CompanyName  string `json:"companyName" binding:"required"`
	ContactName  string `json:"contactName" binding:"required"`
	ContactEmail string `json:"contactEmail" binding:"required,email"`
	ContactPhone string `json:"contactPhone"`
	BillingCycle string `json:"billingCycle" binding:"required,oneof=daily weekly monthly"`
}

// ActivateSellerCommand represents the command to activate a seller
type ActivateSellerCommand struct {
	SellerID string `json:"sellerId" binding:"required"`
}

// SuspendSellerCommand represents the command to suspend a seller
type SuspendSellerCommand struct {
	SellerID string `json:"sellerId" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

// CloseSellerCommand represents the command to close a seller account
type CloseSellerCommand struct {
	SellerID string `json:"sellerId" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

// AssignFacilityCommand represents the command to assign a facility to a seller
type AssignFacilityCommand struct {
	SellerID       string   `json:"sellerId" binding:"required"`
	FacilityID     string   `json:"facilityId" binding:"required"`
	FacilityName   string   `json:"facilityName" binding:"required"`
	WarehouseIDs   []string `json:"warehouseIds"`
	AllocatedSpace float64  `json:"allocatedSpace"`
	IsDefault      bool     `json:"isDefault"`
}

// RemoveFacilityCommand represents the command to remove a facility from a seller
type RemoveFacilityCommand struct {
	SellerID   string `json:"sellerId" binding:"required"`
	FacilityID string `json:"facilityId" binding:"required"`
}

// UpdateFeeScheduleCommand represents the command to update a seller's fee schedule
type UpdateFeeScheduleCommand struct {
	SellerID                   string           `json:"sellerId"` // Set from URL path by handler
	StorageFeePerCubicFtPerDay float64          `json:"storageFeePerCubicFtPerDay"`
	PickFeePerUnit             float64          `json:"pickFeePerUnit"`
	PackFeePerOrder            float64          `json:"packFeePerOrder"`
	ReceivingFeePerUnit        float64          `json:"receivingFeePerUnit"`
	ShippingMarkupPercent      float64          `json:"shippingMarkupPercent"`
	ReturnProcessingFee        float64          `json:"returnProcessingFee"`
	GiftWrapFee                float64          `json:"giftWrapFee"`
	HazmatHandlingFee          float64          `json:"hazmatHandlingFee"`
	OversizedItemFee           float64          `json:"oversizedItemFee"`
	ColdChainFeePerUnit        float64          `json:"coldChainFeePerUnit"`
	FragileHandlingFee         float64          `json:"fragileHandlingFee"`
	VolumeDiscounts            []VolumeDiscount `json:"volumeDiscounts"`
}

// VolumeDiscount represents a volume-based discount tier
type VolumeDiscount struct {
	MinUnits        int     `json:"minUnits"`
	MaxUnits        int     `json:"maxUnits"`
	DiscountPercent float64 `json:"discountPercent"`
}

// ConnectChannelCommand represents the command to connect a sales channel
type ConnectChannelCommand struct {
	SellerID         string              `json:"sellerId"` // Set from URL path by handler
	ChannelType      string              `json:"channelType" binding:"required,oneof=shopify amazon ebay woocommerce"`
	StoreName        string              `json:"storeName" binding:"required"`
	StoreURL         string              `json:"storeUrl"`
	Credentials      map[string]string   `json:"credentials" binding:"required"`
	SyncSettings     ChannelSyncSettings `json:"syncSettings"`
}

// ChannelSyncSettings defines sync behavior for a channel
type ChannelSyncSettings struct {
	AutoImportOrders     bool `json:"autoImportOrders"`
	AutoSyncInventory    bool `json:"autoSyncInventory"`
	AutoPushTracking     bool `json:"autoPushTracking"`
	InventorySyncMinutes int  `json:"inventorySyncMinutes"`
}

// DisconnectChannelCommand represents the command to disconnect a sales channel
type DisconnectChannelCommand struct {
	SellerID  string `json:"sellerId" binding:"required"`
	ChannelID string `json:"channelId" binding:"required"`
}

// GenerateAPIKeyCommand represents the command to generate an API key
type GenerateAPIKeyCommand struct {
	SellerID  string     `json:"sellerId"` // Set from URL path by handler
	Name      string     `json:"name" binding:"required"`
	Scopes    []string   `json:"scopes" binding:"required,min=1"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

// RevokeAPIKeyCommand represents the command to revoke an API key
type RevokeAPIKeyCommand struct {
	SellerID string `json:"sellerId" binding:"required"`
	KeyID    string `json:"keyId" binding:"required"`
}

// Query types

// GetSellerQuery represents a query to get a seller by ID
type GetSellerQuery struct {
	SellerID string
}

// ListSellersQuery represents a query to list sellers with filters
type ListSellersQuery struct {
	TenantID   *string
	Status     *string
	FacilityID *string
	HasChannel *string
	Page       int64
	PageSize   int64
}

// SearchSellersQuery represents a query to search sellers
type SearchSellersQuery struct {
	Query    string
	Page     int64
	PageSize int64
}

// ToDomainFeeSchedule converts command to domain FeeSchedule
func (c *UpdateFeeScheduleCommand) ToDomainFeeSchedule() *domain.FeeSchedule {
	volumeDiscounts := make([]domain.VolumeDiscount, len(c.VolumeDiscounts))
	for i, vd := range c.VolumeDiscounts {
		volumeDiscounts[i] = domain.VolumeDiscount{
			MinUnits:        vd.MinUnits,
			MaxUnits:        vd.MaxUnits,
			DiscountPercent: vd.DiscountPercent,
		}
	}

	return &domain.FeeSchedule{
		StorageFeePerCubicFtPerDay: c.StorageFeePerCubicFtPerDay,
		PickFeePerUnit:             c.PickFeePerUnit,
		PackFeePerOrder:            c.PackFeePerOrder,
		ReceivingFeePerUnit:        c.ReceivingFeePerUnit,
		ShippingMarkupPercent:      c.ShippingMarkupPercent,
		ReturnProcessingFee:        c.ReturnProcessingFee,
		GiftWrapFee:                c.GiftWrapFee,
		HazmatHandlingFee:          c.HazmatHandlingFee,
		OversizedItemFee:           c.OversizedItemFee,
		ColdChainFeePerUnit:        c.ColdChainFeePerUnit,
		FragileHandlingFee:         c.FragileHandlingFee,
		VolumeDiscounts:            volumeDiscounts,
		EffectiveFrom:              time.Now().UTC(),
	}
}

// ToDomainSyncSettings converts command sync settings to domain
func (c *ConnectChannelCommand) ToDomainSyncSettings() domain.ChannelSyncSettings {
	return domain.ChannelSyncSettings{
		AutoImportOrders:     c.SyncSettings.AutoImportOrders,
		AutoSyncInventory:    c.SyncSettings.AutoSyncInventory,
		AutoPushTracking:     c.SyncSettings.AutoPushTracking,
		InventorySyncMinutes: c.SyncSettings.InventorySyncMinutes,
	}
}
