package application

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wms-platform/services/seller-service/internal/domain"
)

func TestUpdateFeeScheduleCommand_ToDomainFeeSchedule(t *testing.T) {
	tests := []struct {
		name    string
		cmd     UpdateFeeScheduleCommand
		wantNil bool
	}{
		{
			name: "Full fee schedule",
			cmd: UpdateFeeScheduleCommand{
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
				VolumeDiscounts: []VolumeDiscount{
					{
						MinUnits:        100,
						MaxUnits:        500,
						DiscountPercent: 5.0,
					},
				},
			},
			wantNil: false,
		},
		{
			name:    "Empty fee schedule",
			cmd:     UpdateFeeScheduleCommand{},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cmd.ToDomainFeeSchedule()

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.cmd.StorageFeePerCubicFtPerDay, result.StorageFeePerCubicFtPerDay)
				assert.Equal(t, tt.cmd.PickFeePerUnit, result.PickFeePerUnit)
				assert.Equal(t, tt.cmd.PackFeePerOrder, result.PackFeePerOrder)
				assert.Equal(t, tt.cmd.ShippingMarkupPercent, result.ShippingMarkupPercent)
				assert.Len(t, result.VolumeDiscounts, len(tt.cmd.VolumeDiscounts))
			}
		})
	}
}

func TestConnectChannelCommand_ToDomainSyncSettings(t *testing.T) {
	tests := []struct {
		name string
		cmd  ConnectChannelCommand
		want domain.ChannelSyncSettings
	}{
		{
			name: "Full sync settings",
			cmd: ConnectChannelCommand{
				SyncSettings: ChannelSyncSettings{
					AutoImportOrders:     true,
					AutoSyncInventory:    true,
					AutoPushTracking:     true,
					InventorySyncMinutes: 30,
				},
			},
			want: domain.ChannelSyncSettings{
				AutoImportOrders:     true,
				AutoSyncInventory:    true,
				AutoPushTracking:     true,
				InventorySyncMinutes: 30,
			},
		},
		{
			name: "Empty sync settings",
			cmd: ConnectChannelCommand{
				SyncSettings: ChannelSyncSettings{},
			},
			want: domain.ChannelSyncSettings{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cmd.ToDomainSyncSettings()
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestToSellerDTO(t *testing.T) {
	now := time.Now().UTC()

	seller := &domain.Seller{
		SellerID:     "SLR-001",
		TenantID:     "TNT-001",
		CompanyName:  "Test Corp",
		ContactName:  "John Doe",
		ContactEmail: "john@test.com",
		ContactPhone: "555-1234",
		Status:       domain.SellerStatusActive,
		BillingCycle: domain.BillingCycleMonthly,
		AssignedFacilities: []domain.FacilityAssignment{
			{
				FacilityID:     "FAC-001",
				FacilityName:   "Test DC",
				WarehouseIDs:   []string{"WH-001"},
				AllocatedSpace: 1000,
				AssignedAt:     now,
				IsDefault:      true,
			},
		},
		FeeSchedule: &domain.FeeSchedule{
			StorageFeePerCubicFtPerDay: 0.05,
			PickFeePerUnit:             0.25,
			VolumeDiscounts: []domain.VolumeDiscount{
				{
					MinUnits:        100,
					MaxUnits:        500,
					DiscountPercent: 5.0,
				},
			},
			EffectiveFrom: now,
		},
		Integrations: []domain.ChannelIntegration{
			{
				ChannelID:   "CH-001",
				ChannelType: "shopify",
				StoreName:   "My Store",
				StoreURL:    "https://mystore.com",
				Status:      "active",
				SyncSettings: domain.ChannelSyncSettings{
					AutoImportOrders: true,
				},
				ConnectedAt: now,
			},
		},
		APIKeys: []domain.APIKey{
			{
				KeyID:     "key-001",
				Name:      "Test Key",
				Prefix:    "abcd1234",
				Scopes:    []string{"orders:read"},
				CreatedAt: now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := ToSellerDTO(seller)

	assert.NotNil(t, result)
	assert.Equal(t, seller.SellerID, result.SellerID)
	assert.Equal(t, seller.TenantID, result.TenantID)
	assert.Equal(t, seller.CompanyName, result.CompanyName)
	assert.Equal(t, seller.ContactName, result.ContactName)
	assert.Equal(t, seller.ContactEmail, result.ContactEmail)
	assert.Equal(t, seller.ContactPhone, result.ContactPhone)
	assert.Equal(t, "active", result.Status)
	assert.Equal(t, "monthly", result.BillingCycle)
	assert.Len(t, result.AssignedFacilities, 1)
	assert.Equal(t, "FAC-001", result.AssignedFacilities[0].FacilityID)
	assert.NotNil(t, result.FeeSchedule)
	assert.Equal(t, 0.05, result.FeeSchedule.StorageFeePerCubicFtPerDay)
	assert.Len(t, result.FeeSchedule.VolumeDiscounts, 1)
	assert.Len(t, result.Integrations, 1)
	assert.Equal(t, "shopify", result.Integrations[0].ChannelType)
	assert.Equal(t, 1, result.APIKeysCount)
}

func TestToSellerDTO_WithNilFeeSchedule(t *testing.T) {
	seller := &domain.Seller{
		SellerID:     "SLR-001",
		TenantID:     "TNT-001",
		CompanyName:  "Test Corp",
		ContactName:  "John Doe",
		ContactEmail: "john@test.com",
		Status:       domain.SellerStatusActive,
		BillingCycle: domain.BillingCycleMonthly,
		FeeSchedule:  nil,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	result := ToSellerDTO(seller)

	assert.NotNil(t, result)
	assert.Nil(t, result.FeeSchedule)
}

func TestToSellerDTO_WithNilContractEndDate(t *testing.T) {
	seller := &domain.Seller{
		SellerID:        "SLR-001",
		TenantID:        "TNT-001",
		CompanyName:     "Test Corp",
		ContactName:     "John Doe",
		ContactEmail:    "john@test.com",
		Status:          domain.SellerStatusActive,
		BillingCycle:    domain.BillingCycleMonthly,
		ContractEndDate: nil,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	result := ToSellerDTO(seller)

	assert.NotNil(t, result)
	assert.Nil(t, result.ContractEndDate)
}

func TestToSellerDTO_WithMultipleAPIKeys(t *testing.T) {
	now := time.Now().UTC()

	seller := &domain.Seller{
		SellerID:     "SLR-001",
		TenantID:     "TNT-001",
		CompanyName:  "Test Corp",
		ContactName:  "John Doe",
		ContactEmail: "john@test.com",
		Status:       domain.SellerStatusActive,
		BillingCycle: domain.BillingCycleMonthly,
		APIKeys: []domain.APIKey{
			{
				KeyID:     "key-001",
				Name:      "Active Key",
				CreatedAt: now,
			},
			{
				KeyID:     "key-002",
				Name:      "Revoked Key",
				RevokedAt: &now,
				CreatedAt: now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := ToSellerDTO(seller)

	assert.NotNil(t, result)
	assert.Equal(t, 1, result.APIKeysCount)
}

func TestToAPIKeyDTOs(t *testing.T) {
	now := time.Now().UTC()
	revokedTime := now.Add(-1 * time.Hour)

	keys := []domain.APIKey{
		{
			KeyID:      "key-001",
			Name:       "Active Key",
			Prefix:     "abcd1234",
			Scopes:     []string{"orders:read", "inventory:write"},
			ExpiresAt:  &now,
			LastUsedAt: &now,
			CreatedAt:  now,
		},
		{
			KeyID:     "key-002",
			Name:      "Revoked Key",
			Prefix:    "efgh5678",
			Scopes:    []string{"orders:read"},
			RevokedAt: &revokedTime,
			CreatedAt: now,
		},
	}

	result := ToAPIKeyDTOs(keys)

	assert.NotNil(t, result)
	assert.Len(t, result, 2)

	assert.Equal(t, "key-001", result[0].KeyID)
	assert.Equal(t, "Active Key", result[0].Name)
	assert.Equal(t, "abcd1234", result[0].Prefix)
	assert.Equal(t, []string{"orders:read", "inventory:write"}, result[0].Scopes)
	assert.Equal(t, &now, result[0].ExpiresAt)
	assert.Equal(t, &now, result[0].LastUsedAt)
	assert.Equal(t, now, result[0].CreatedAt)
	assert.False(t, result[0].IsRevoked)

	assert.Equal(t, "key-002", result[1].KeyID)
	assert.Equal(t, "Revoked Key", result[1].Name)
	assert.Equal(t, "efgh5678", result[1].Prefix)
	assert.Equal(t, []string{"orders:read"}, result[1].Scopes)
	assert.Equal(t, now, result[1].CreatedAt)
	assert.True(t, result[1].IsRevoked)
}

func TestToAPIKeyDTOs_Empty(t *testing.T) {
	result := ToAPIKeyDTOs([]domain.APIKey{})
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestPagination_DefaultPagination(t *testing.T) {
	pag := domain.DefaultPagination()

	assert.Equal(t, int64(1), pag.Page)
	assert.Equal(t, int64(20), pag.PageSize)
}

func TestPagination_Skip(t *testing.T) {
	tests := []struct {
		name     string
		page     int64
		pageSize int64
		want     int64
	}{
		{
			name:     "Page 1, size 20",
			page:     1,
			pageSize: 20,
			want:     0,
		},
		{
			name:     "Page 2, size 20",
			page:     2,
			pageSize: 20,
			want:     20,
		},
		{
			name:     "Page 3, size 10",
			page:     3,
			pageSize: 10,
			want:     20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pag := domain.Pagination{
				Page:     tt.page,
				PageSize: tt.pageSize,
			}
			assert.Equal(t, tt.want, pag.Skip())
		})
	}
}

func TestPagination_Limit(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int64
		want     int64
	}{
		{
			name:     "Size 10",
			pageSize: 10,
			want:     10,
		},
		{
			name:     "Size 20",
			pageSize: 20,
			want:     20,
		},
		{
			name:     "Size 50",
			pageSize: 50,
			want:     50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pag := domain.Pagination{
				PageSize: tt.pageSize,
			}
			assert.Equal(t, tt.want, pag.Limit())
		})
	}
}
