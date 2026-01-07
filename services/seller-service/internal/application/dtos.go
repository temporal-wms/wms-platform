package application

import (
	"time"

	"github.com/wms-platform/services/seller-service/internal/domain"
)

// SellerDTO represents a seller in API responses
type SellerDTO struct {
	SellerID           string                   `json:"sellerId"`
	TenantID           string                   `json:"tenantId"`
	CompanyName        string                   `json:"companyName"`
	ContactName        string                   `json:"contactName"`
	ContactEmail       string                   `json:"contactEmail"`
	ContactPhone       string                   `json:"contactPhone"`
	Status             string                   `json:"status"`
	ContractStartDate  time.Time                `json:"contractStartDate"`
	ContractEndDate    *time.Time               `json:"contractEndDate,omitempty"`
	BillingCycle       string                   `json:"billingCycle"`
	AssignedFacilities []FacilityAssignmentDTO  `json:"assignedFacilities"`
	FeeSchedule        *FeeScheduleDTO          `json:"feeSchedule,omitempty"`
	Integrations       []ChannelIntegrationDTO  `json:"integrations"`
	APIKeysCount       int                      `json:"apiKeysCount"`
	CreatedAt          time.Time                `json:"createdAt"`
	UpdatedAt          time.Time                `json:"updatedAt"`
}

// FacilityAssignmentDTO represents a facility assignment in API responses
type FacilityAssignmentDTO struct {
	FacilityID     string    `json:"facilityId"`
	FacilityName   string    `json:"facilityName"`
	WarehouseIDs   []string  `json:"warehouseIds"`
	AllocatedSpace float64   `json:"allocatedSpace"`
	AssignedAt     time.Time `json:"assignedAt"`
	IsDefault      bool      `json:"isDefault"`
}

// FeeScheduleDTO represents a fee schedule in API responses
type FeeScheduleDTO struct {
	StorageFeePerCubicFtPerDay float64             `json:"storageFeePerCubicFtPerDay"`
	PickFeePerUnit             float64             `json:"pickFeePerUnit"`
	PackFeePerOrder            float64             `json:"packFeePerOrder"`
	ReceivingFeePerUnit        float64             `json:"receivingFeePerUnit"`
	ShippingMarkupPercent      float64             `json:"shippingMarkupPercent"`
	ReturnProcessingFee        float64             `json:"returnProcessingFee"`
	GiftWrapFee                float64             `json:"giftWrapFee"`
	HazmatHandlingFee          float64             `json:"hazmatHandlingFee"`
	OversizedItemFee           float64             `json:"oversizedItemFee"`
	ColdChainFeePerUnit        float64             `json:"coldChainFeePerUnit"`
	FragileHandlingFee         float64             `json:"fragileHandlingFee"`
	VolumeDiscounts            []VolumeDiscountDTO `json:"volumeDiscounts"`
	EffectiveFrom              time.Time           `json:"effectiveFrom"`
	EffectiveTo                *time.Time          `json:"effectiveTo,omitempty"`
}

// VolumeDiscountDTO represents a volume discount tier in API responses
type VolumeDiscountDTO struct {
	MinUnits        int     `json:"minUnits"`
	MaxUnits        int     `json:"maxUnits"`
	DiscountPercent float64 `json:"discountPercent"`
}

// ChannelIntegrationDTO represents a channel integration in API responses
type ChannelIntegrationDTO struct {
	ChannelID    string                  `json:"channelId"`
	ChannelType  string                  `json:"channelType"`
	StoreName    string                  `json:"storeName"`
	StoreURL     string                  `json:"storeUrl,omitempty"`
	Status       string                  `json:"status"`
	SyncSettings ChannelSyncSettingsDTO  `json:"syncSettings"`
	ConnectedAt  time.Time               `json:"connectedAt"`
	LastSyncAt   *time.Time              `json:"lastSyncAt,omitempty"`
}

// ChannelSyncSettingsDTO represents sync settings in API responses
type ChannelSyncSettingsDTO struct {
	AutoImportOrders     bool `json:"autoImportOrders"`
	AutoSyncInventory    bool `json:"autoSyncInventory"`
	AutoPushTracking     bool `json:"autoPushTracking"`
	InventorySyncMinutes int  `json:"inventorySyncMinutes"`
}

// APIKeyDTO represents an API key in API responses (without sensitive data)
type APIKeyDTO struct {
	KeyID      string     `json:"keyId"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	IsRevoked  bool       `json:"isRevoked"`
}

// APIKeyCreatedDTO represents a newly created API key (includes raw key - only shown once)
type APIKeyCreatedDTO struct {
	KeyID     string     `json:"keyId"`
	Name      string     `json:"name"`
	RawKey    string     `json:"apiKey"` // Only shown once at creation
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// SellerListResponse represents a paginated list of sellers
type SellerListResponse struct {
	Data       []SellerDTO `json:"data"`
	Total      int64       `json:"total"`
	Page       int64       `json:"page"`
	PageSize   int64       `json:"pageSize"`
	TotalPages int64       `json:"totalPages"`
}

// ToSellerDTO converts a domain Seller to a SellerDTO
func ToSellerDTO(seller *domain.Seller) *SellerDTO {
	facilities := make([]FacilityAssignmentDTO, len(seller.AssignedFacilities))
	for i, f := range seller.AssignedFacilities {
		facilities[i] = FacilityAssignmentDTO{
			FacilityID:     f.FacilityID,
			FacilityName:   f.FacilityName,
			WarehouseIDs:   f.WarehouseIDs,
			AllocatedSpace: f.AllocatedSpace,
			AssignedAt:     f.AssignedAt,
			IsDefault:      f.IsDefault,
		}
	}

	integrations := make([]ChannelIntegrationDTO, len(seller.Integrations))
	for i, ch := range seller.Integrations {
		integrations[i] = ChannelIntegrationDTO{
			ChannelID:   ch.ChannelID,
			ChannelType: ch.ChannelType,
			StoreName:   ch.StoreName,
			StoreURL:    ch.StoreURL,
			Status:      ch.Status,
			SyncSettings: ChannelSyncSettingsDTO{
				AutoImportOrders:     ch.SyncSettings.AutoImportOrders,
				AutoSyncInventory:    ch.SyncSettings.AutoSyncInventory,
				AutoPushTracking:     ch.SyncSettings.AutoPushTracking,
				InventorySyncMinutes: ch.SyncSettings.InventorySyncMinutes,
			},
			ConnectedAt: ch.ConnectedAt,
			LastSyncAt:  ch.LastSyncAt,
		}
	}

	var feeScheduleDTO *FeeScheduleDTO
	if seller.FeeSchedule != nil {
		volumeDiscounts := make([]VolumeDiscountDTO, len(seller.FeeSchedule.VolumeDiscounts))
		for i, vd := range seller.FeeSchedule.VolumeDiscounts {
			volumeDiscounts[i] = VolumeDiscountDTO{
				MinUnits:        vd.MinUnits,
				MaxUnits:        vd.MaxUnits,
				DiscountPercent: vd.DiscountPercent,
			}
		}

		feeScheduleDTO = &FeeScheduleDTO{
			StorageFeePerCubicFtPerDay: seller.FeeSchedule.StorageFeePerCubicFtPerDay,
			PickFeePerUnit:             seller.FeeSchedule.PickFeePerUnit,
			PackFeePerOrder:            seller.FeeSchedule.PackFeePerOrder,
			ReceivingFeePerUnit:        seller.FeeSchedule.ReceivingFeePerUnit,
			ShippingMarkupPercent:      seller.FeeSchedule.ShippingMarkupPercent,
			ReturnProcessingFee:        seller.FeeSchedule.ReturnProcessingFee,
			GiftWrapFee:                seller.FeeSchedule.GiftWrapFee,
			HazmatHandlingFee:          seller.FeeSchedule.HazmatHandlingFee,
			OversizedItemFee:           seller.FeeSchedule.OversizedItemFee,
			ColdChainFeePerUnit:        seller.FeeSchedule.ColdChainFeePerUnit,
			FragileHandlingFee:         seller.FeeSchedule.FragileHandlingFee,
			VolumeDiscounts:            volumeDiscounts,
			EffectiveFrom:              seller.FeeSchedule.EffectiveFrom,
			EffectiveTo:                seller.FeeSchedule.EffectiveTo,
		}
	}

	// Count active API keys
	activeKeysCount := 0
	for _, k := range seller.APIKeys {
		if !k.IsRevoked() {
			activeKeysCount++
		}
	}

	return &SellerDTO{
		SellerID:           seller.SellerID,
		TenantID:           seller.TenantID,
		CompanyName:        seller.CompanyName,
		ContactName:        seller.ContactName,
		ContactEmail:       seller.ContactEmail,
		ContactPhone:       seller.ContactPhone,
		Status:             string(seller.Status),
		ContractStartDate:  seller.ContractStartDate,
		ContractEndDate:    seller.ContractEndDate,
		BillingCycle:       string(seller.BillingCycle),
		AssignedFacilities: facilities,
		FeeSchedule:        feeScheduleDTO,
		Integrations:       integrations,
		APIKeysCount:       activeKeysCount,
		CreatedAt:          seller.CreatedAt,
		UpdatedAt:          seller.UpdatedAt,
	}
}

// ToAPIKeyDTOs converts domain API keys to DTOs
func ToAPIKeyDTOs(keys []domain.APIKey) []APIKeyDTO {
	dtos := make([]APIKeyDTO, len(keys))
	for i, k := range keys {
		dtos[i] = APIKeyDTO{
			KeyID:      k.KeyID,
			Name:       k.Name,
			Prefix:     k.Prefix,
			Scopes:     k.Scopes,
			ExpiresAt:  k.ExpiresAt,
			LastUsedAt: k.LastUsedAt,
			CreatedAt:  k.CreatedAt,
			IsRevoked:  k.IsRevoked(),
		}
	}
	return dtos
}
