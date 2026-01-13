package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wms-platform/services/seller-service/internal/domain"
)

func TestNewTestSeller(t *testing.T) {
	seller := NewTestSeller()

	assert.NotNil(t, seller)
	assert.NotEmpty(t, seller.SellerID)
	assert.Equal(t, "TNT-001", seller.TenantID)
	assert.Equal(t, "Test Corp", seller.CompanyName)
	assert.Equal(t, "John Doe", seller.ContactName)
	assert.Equal(t, "john@test.com", seller.ContactEmail)
	assert.Equal(t, domain.SellerStatusActive, seller.Status)
}

func TestNewTestSellerDTO(t *testing.T) {
	dto := NewTestSellerDTO()

	assert.NotNil(t, dto)
	assert.Equal(t, "SLR-001", dto.SellerID)
	assert.Equal(t, "TNT-001", dto.TenantID)
	assert.Equal(t, "Test Corp", dto.CompanyName)
}

func TestNewTestCreateSellerCommand(t *testing.T) {
	cmd := NewTestCreateSellerCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "TNT-001", cmd.TenantID)
	assert.Equal(t, "Test Corp", cmd.CompanyName)
	assert.Equal(t, "monthly", cmd.BillingCycle)
}

func TestNewTestUpdateFeeScheduleCommand(t *testing.T) {
	cmd := NewTestUpdateFeeScheduleCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, 0.06, cmd.StorageFeePerCubicFtPerDay)
	assert.Equal(t, 0.30, cmd.PickFeePerUnit)
	assert.Len(t, cmd.VolumeDiscounts, 1)
}

func TestNewTestConnectChannelCommand(t *testing.T) {
	cmd := NewTestConnectChannelCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "shopify", cmd.ChannelType)
	assert.Equal(t, "My Store", cmd.StoreName)
	assert.NotNil(t, cmd.Credentials)
	assert.True(t, cmd.SyncSettings.AutoImportOrders)
}

func TestNewTestGenerateAPIKeyCommand(t *testing.T) {
	cmd := NewTestGenerateAPIKeyCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "Test Key", cmd.Name)
	assert.Len(t, cmd.Scopes, 2)
}
