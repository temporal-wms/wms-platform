package testutil

import (
	"time"

	"github.com/wms-platform/services/seller-service/internal/application"
	"github.com/wms-platform/services/seller-service/internal/domain"
)

// NewTestSeller creates a test seller with default values
func NewTestSeller() *domain.Seller {
	seller, err := domain.NewSeller("TNT-001", "Test Corp", "John Doe", "john@test.com", domain.BillingCycleMonthly)
	if err != nil {
		panic(err)
	}
	seller.Status = domain.SellerStatusActive
	return seller
}

// NewTestSellerDTO creates a test SellerDTO
func NewTestSellerDTO() *application.SellerDTO {
	return &application.SellerDTO{
		SellerID:           "SLR-001",
		TenantID:           "TNT-001",
		CompanyName:        "Test Corp",
		ContactName:        "John Doe",
		ContactEmail:       "john@test.com",
		Status:             "active",
		BillingCycle:       "monthly",
		AssignedFacilities: []application.FacilityAssignmentDTO{},
		Integrations:       []application.ChannelIntegrationDTO{},
		APIKeysCount:       0,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

// NewTestCreateSellerCommand creates a test CreateSellerCommand
func NewTestCreateSellerCommand() application.CreateSellerCommand {
	return application.CreateSellerCommand{
		TenantID:     "TNT-001",
		CompanyName:  "Test Corp",
		ContactName:  "John Doe",
		ContactEmail: "john@test.com",
		ContactPhone: "555-1234",
		BillingCycle: "monthly",
	}
}

// NewTestUpdateFeeScheduleCommand creates a test UpdateFeeScheduleCommand
func NewTestUpdateFeeScheduleCommand() application.UpdateFeeScheduleCommand {
	return application.UpdateFeeScheduleCommand{
		StorageFeePerCubicFtPerDay: 0.06,
		PickFeePerUnit:             0.30,
		PackFeePerOrder:            1.75,
		ReceivingFeePerUnit:        0.20,
		ShippingMarkupPercent:      6.0,
		ReturnProcessingFee:        3.50,
		GiftWrapFee:                3.00,
		HazmatHandlingFee:          6.00,
		OversizedItemFee:           12.00,
		ColdChainFeePerUnit:        1.50,
		FragileHandlingFee:         2.00,
		VolumeDiscounts: []application.VolumeDiscount{
			{
				MinUnits:        100,
				MaxUnits:        500,
				DiscountPercent: 5.0,
			},
		},
	}
}

// NewTestConnectChannelCommand creates a test ConnectChannelCommand
func NewTestConnectChannelCommand() application.ConnectChannelCommand {
	return application.ConnectChannelCommand{
		ChannelType: "shopify",
		StoreName:   "My Store",
		StoreURL:    "https://mystore.com",
		Credentials: map[string]string{
			"apiKey":   "test-key",
			"password": "test-password",
		},
		SyncSettings: application.ChannelSyncSettings{
			AutoImportOrders:     true,
			AutoSyncInventory:    true,
			AutoPushTracking:     true,
			InventorySyncMinutes: 30,
		},
	}
}

// NewTestGenerateAPIKeyCommand creates a test GenerateAPIKeyCommand
func NewTestGenerateAPIKeyCommand() application.GenerateAPIKeyCommand {
	return application.GenerateAPIKeyCommand{
		Name:   "Test Key",
		Scopes: []string{"orders:read", "inventory:read"},
	}
}
