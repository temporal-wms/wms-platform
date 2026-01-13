package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/services/seller-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
)

type mockSellerRepository struct {
	sellers map[string]*domain.Seller
	byEmail map[string]*domain.Seller

	SaveFunc           func(ctx context.Context, seller *domain.Seller) error
	FindByIDFunc       func(ctx context.Context, sellerID string) (*domain.Seller, error)
	FindByTenantIDFunc func(ctx context.Context, tenantID string, pagination domain.Pagination) ([]*domain.Seller, error)
	FindByStatusFunc   func(ctx context.Context, status domain.SellerStatus, pagination domain.Pagination) ([]*domain.Seller, error)
	FindByEmailFunc    func(ctx context.Context, email string) (*domain.Seller, error)
}

func newMockSellerRepository() *mockSellerRepository {
	return &mockSellerRepository{
		sellers: make(map[string]*domain.Seller),
		byEmail: make(map[string]*domain.Seller),
	}
}

func (m *mockSellerRepository) AddSeller(seller *domain.Seller) {
	m.sellers[seller.SellerID] = seller
	m.byEmail[seller.ContactEmail] = seller
}

func (m *mockSellerRepository) Save(ctx context.Context, seller *domain.Seller) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, seller)
	}
	m.sellers[seller.SellerID] = seller
	m.byEmail[seller.ContactEmail] = seller
	return nil
}

func (m *mockSellerRepository) FindByID(ctx context.Context, sellerID string) (*domain.Seller, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, sellerID)
	}
	seller, ok := m.sellers[sellerID]
	if !ok {
		return nil, nil
	}
	return seller, nil
}

func (m *mockSellerRepository) FindByTenantID(ctx context.Context, tenantID string, pagination domain.Pagination) ([]*domain.Seller, error) {
	if m.FindByTenantIDFunc != nil {
		return m.FindByTenantIDFunc(ctx, tenantID, pagination)
	}
	var result []*domain.Seller
	for _, seller := range m.sellers {
		if seller.TenantID == tenantID {
			result = append(result, seller)
		}
	}
	return result, nil
}

func (m *mockSellerRepository) FindByStatus(ctx context.Context, status domain.SellerStatus, pagination domain.Pagination) ([]*domain.Seller, error) {
	if m.FindByStatusFunc != nil {
		return m.FindByStatusFunc(ctx, status, pagination)
	}
	var result []*domain.Seller
	for _, seller := range m.sellers {
		if seller.Status == status {
			result = append(result, seller)
		}
	}
	return result, nil
}

func (m *mockSellerRepository) FindByAPIKey(ctx context.Context, hashedKey string) (*domain.Seller, error) {
	return nil, nil
}

func (m *mockSellerRepository) FindByEmail(ctx context.Context, email string) (*domain.Seller, error) {
	if m.FindByEmailFunc != nil {
		return m.FindByEmailFunc(ctx, email)
	}
	seller, ok := m.byEmail[email]
	if !ok {
		return nil, nil
	}
	return seller, nil
}

func (m *mockSellerRepository) UpdateStatus(ctx context.Context, sellerID string, status domain.SellerStatus) error {
	return nil
}

func (m *mockSellerRepository) Delete(ctx context.Context, sellerID string) error {
	return nil
}

func (m *mockSellerRepository) Count(ctx context.Context, filter domain.SellerFilter) (int64, error) {
	return int64(len(m.sellers)), nil
}

func (m *mockSellerRepository) Search(ctx context.Context, query string, pagination domain.Pagination) ([]*domain.Seller, error) {
	return nil, nil
}

func TestNewSellerApplicationService(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))

	service := NewSellerApplicationService(mockRepo, logger)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.sellerRepo)
}

func TestSellerApplicationService_CreateSeller(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	tests := []struct {
		name    string
		cmd     CreateSellerCommand
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			cmd: CreateSellerCommand{
				TenantID:     "TNT-001",
				CompanyName:  "Test Corp",
				ContactName:  "John Doe",
				ContactEmail: "john@test.com",
				BillingCycle: "monthly",
			},
			wantErr: false,
		},
		{
			name: "Duplicate email",
			cmd: CreateSellerCommand{
				TenantID:     "TNT-001",
				CompanyName:  "Test Corp",
				ContactName:  "John Doe",
				ContactEmail: "john@test.com",
				BillingCycle: "monthly",
			},
			setup: func() {
				seller := newTestSeller()
				seller.ContactEmail = "john@test.com"
				mockRepo.AddSeller(seller)
			},
			wantErr: true,
		},
		{
			name: "Invalid billing cycle",
			cmd: CreateSellerCommand{
				TenantID:     "TNT-001",
				CompanyName:  "Test Corp",
				ContactName:  "John Doe",
				ContactEmail: "john@test.com",
				BillingCycle: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			ctx := context.Background()
			result, err := service.CreateSeller(ctx, tt.cmd)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.cmd.CompanyName, result.CompanyName)
				assert.Equal(t, tt.cmd.ContactEmail, result.ContactEmail)
				assert.Equal(t, "pending", result.Status)
			}
		})
	}
}

func TestSellerApplicationService_GetSeller(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	tests := []struct {
		name     string
		sellerID string
		setup    func()
		wantErr  bool
	}{
		{
			name:     "Success",
			sellerID: "SLR-001",
			setup: func() {
				seller := newTestSeller()
				seller.SellerID = "SLR-001"
				mockRepo.AddSeller(seller)
			},
			wantErr: false,
		},
		{
			name:     "Not found",
			sellerID: "SLR-999",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			ctx := context.Background()
			result, err := service.GetSeller(ctx, GetSellerQuery{SellerID: tt.sellerID})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.sellerID, result.SellerID)
			}
		})
	}
}

func TestSellerApplicationService_ListSellers(t *testing.T) {

	tests := []struct {
		name        string
		query       ListSellersQuery
		wantDataLen int
		wantErr     bool
	}{
		{
			name:        "Success with default pagination",
			query:       ListSellersQuery{Page: 1, PageSize: 20},
			wantDataLen: 0,
			wantErr:     false,
		},
		{
			name:        "Success with sellers",
			query:       ListSellersQuery{Page: 1, PageSize: 20},
			wantDataLen: 2,
			wantErr:     false,
		},
		{
			name:        "Filter by tenant",
			query:       ListSellersQuery{Page: 1, PageSize: 20, TenantID: strPtr("TNT-001")},
			wantDataLen: 1,
			wantErr:     false,
		},
		{
			name:        "Filter by status",
			query:       ListSellersQuery{Page: 1, PageSize: 20, Status: strPtr("active")},
			wantDataLen: 1,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := newMockSellerRepository()
			logger := logging.New(logging.DefaultConfig("test"))
			service := NewSellerApplicationService(mockRepo, logger)

			if tt.name == "Success with sellers" {
				seller1 := newTestSeller()
				seller1.SellerID = "SLR-001"
				seller1.TenantID = "TNT-001"
				mockRepo.AddSeller(seller1)

				seller2 := newTestSeller()
				seller2.SellerID = "SLR-002"
				seller2.TenantID = "TNT-001"
				mockRepo.AddSeller(seller2)
			} else if tt.name == "Filter by tenant" {
				seller := newTestSeller()
				seller.TenantID = "TNT-001"
				seller.SellerID = "SLR-001"
				mockRepo.AddSeller(seller)
			} else if tt.name == "Filter by status" {
				seller := newTestSeller()
				seller.Status = domain.SellerStatusActive
				seller.SellerID = "SLR-001"
				mockRepo.AddSeller(seller)
			}

			ctx := context.Background()
			result, err := service.ListSellers(ctx, tt.query)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.LessOrEqual(t, len(result.Data), tt.wantDataLen)
			}
		})
	}
}

func TestSellerApplicationService_ActivateSeller(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	tests := []struct {
		name       string
		sellerID   string
		setup      func()
		wantErr    bool
		wantStatus string
	}{
		{
			name:     "Success activate pending seller",
			sellerID: "SLR-001",
			setup: func() {
				seller := newTestSeller()
				seller.SellerID = "SLR-001"
				seller.Status = domain.SellerStatusPending
				mockRepo.AddSeller(seller)
			},
			wantErr:    false,
			wantStatus: "active",
		},
		{
			name:     "Success activate suspended seller",
			sellerID: "SLR-002",
			setup: func() {
				seller := newTestSeller()
				seller.SellerID = "SLR-002"
				seller.Status = domain.SellerStatusSuspended
				mockRepo.AddSeller(seller)
			},
			wantErr:    false,
			wantStatus: "active",
		},
		{
			name:     "Cannot activate active seller",
			sellerID: "SLR-003",
			setup: func() {
				seller := newTestSeller()
				seller.SellerID = "SLR-003"
				seller.Status = domain.SellerStatusActive
				mockRepo.AddSeller(seller)
			},
			wantErr:    true,
			wantStatus: "active",
		},
		{
			name:       "Seller not found",
			sellerID:   "SLR-999",
			wantErr:    true,
			wantStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			ctx := context.Background()
			result, err := service.ActivateSeller(ctx, ActivateSellerCommand{SellerID: tt.sellerID})

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantStatus == "" {
					assert.Nil(t, result)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.wantStatus, result.Status)
			}
		})
	}
}

func TestSellerApplicationService_SuspendSeller(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	seller := newTestSeller()
	seller.SellerID = "SLR-001"
	seller.Status = domain.SellerStatusActive
	mockRepo.AddSeller(seller)

	ctx := context.Background()
	result, err := service.SuspendSeller(ctx, SuspendSellerCommand{
		SellerID: "SLR-001",
		Reason:   "Test suspension",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "suspended", result.Status)
}

func TestSellerApplicationService_CloseSeller(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	seller := newTestSeller()
	seller.SellerID = "SLR-001"
	seller.Status = domain.SellerStatusActive
	mockRepo.AddSeller(seller)

	ctx := context.Background()
	result, err := service.CloseSeller(ctx, CloseSellerCommand{
		SellerID: "SLR-001",
		Reason:   "Contract ended",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "closed", result.Status)
}

func TestSellerApplicationService_AssignFacility(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	seller := newTestSeller()
	seller.SellerID = "SLR-001"
	seller.Status = domain.SellerStatusActive
	mockRepo.AddSeller(seller)

	tests := []struct {
		name     string
		cmd      AssignFacilityCommand
		wantErr  bool
		wantFacs int
	}{
		{
			name: "Success",
			cmd: AssignFacilityCommand{
				SellerID:       "SLR-001",
				FacilityID:     "FAC-001",
				FacilityName:   "Test DC",
				WarehouseIDs:   []string{"WH-001"},
				AllocatedSpace: 1000,
				IsDefault:      true,
			},
			wantErr:  false,
			wantFacs: 1,
		},
		{
			name: "Seller not found",
			cmd: AssignFacilityCommand{
				SellerID:     "SLR-999",
				FacilityID:   "FAC-001",
				FacilityName: "Test DC",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := service.AssignFacility(ctx, tt.cmd)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result.AssignedFacilities, tt.wantFacs)
			}
		})
	}
}

func TestSellerApplicationService_RemoveFacility(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	seller := newTestSeller()
	seller.SellerID = "SLR-001"
	seller.Status = domain.SellerStatusActive
	err := seller.AssignFacility("FAC-001", "Test DC", []string{"WH-001"}, 1000, true)
	if err != nil {
		t.Fatal(err)
	}
	mockRepo.AddSeller(seller)

	ctx := context.Background()
	result, err := service.RemoveFacility(ctx, RemoveFacilityCommand{
		SellerID:   "SLR-001",
		FacilityID: "FAC-001",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.AssignedFacilities)
}

func TestSellerApplicationService_UpdateFeeSchedule(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	seller := newTestSeller()
	seller.SellerID = "SLR-001"
	mockRepo.AddSeller(seller)

	cmd := UpdateFeeScheduleCommand{
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
	}
	cmd.SellerID = "SLR-001"

	ctx := context.Background()
	result, err := service.UpdateFeeSchedule(ctx, cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.FeeSchedule)
	assert.Equal(t, cmd.StorageFeePerCubicFtPerDay, result.FeeSchedule.StorageFeePerCubicFtPerDay)
}

func TestSellerApplicationService_ConnectChannel(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	seller := newTestSeller()
	seller.SellerID = "SLR-001"
	seller.Status = domain.SellerStatusActive
	mockRepo.AddSeller(seller)

	cmd := ConnectChannelCommand{
		ChannelType:  "shopify",
		StoreName:    "My Store",
		StoreURL:     "https://mystore.com",
		Credentials:  map[string]string{"apiKey": "test-key"},
		SyncSettings: ChannelSyncSettings{AutoImportOrders: true},
	}
	cmd.SellerID = "SLR-001"

	ctx := context.Background()
	result, err := service.ConnectChannel(ctx, cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Integrations, 1)
	assert.Equal(t, cmd.ChannelType, result.Integrations[0].ChannelType)
}

func TestSellerApplicationService_DisconnectChannel(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	seller := newTestSeller()
	seller.SellerID = "SLR-001"
	seller.Status = domain.SellerStatusActive
	err := seller.AddChannelIntegration("shopify", "My Store", "", nil, domain.ChannelSyncSettings{})
	if err != nil {
		t.Fatal(err)
	}
	mockRepo.AddSeller(seller)

	channelID := seller.Integrations[0].ChannelID

	ctx := context.Background()
	result, err := service.DisconnectChannel(ctx, DisconnectChannelCommand{
		SellerID:  "SLR-001",
		ChannelID: channelID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "disconnected", result.Integrations[0].Status)
}

func TestSellerApplicationService_GenerateAPIKey(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	seller := newTestSeller()
	seller.SellerID = "SLR-001"
	seller.Status = domain.SellerStatusActive
	mockRepo.AddSeller(seller)

	cmd := GenerateAPIKeyCommand{
		Name:   "Test Key",
		Scopes: []string{"orders:read"},
	}
	cmd.SellerID = "SLR-001"

	ctx := context.Background()
	result, err := service.GenerateAPIKey(ctx, cmd)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.KeyID)
	assert.NotEmpty(t, result.RawKey)
	assert.Equal(t, cmd.Name, result.Name)
	assert.Equal(t, cmd.Scopes, result.Scopes)
}

func TestSellerApplicationService_RevokeAPIKey(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	seller := newTestSeller()
	seller.SellerID = "SLR-001"
	seller.Status = domain.SellerStatusActive
	key, _, _ := seller.GenerateAPIKey("Test Key", []string{"orders:read"}, nil)
	mockRepo.AddSeller(seller)

	ctx := context.Background()
	err := service.RevokeAPIKey(ctx, RevokeAPIKeyCommand{
		SellerID: "SLR-001",
		KeyID:    key.KeyID,
	})

	require.NoError(t, err)
}

func TestSellerApplicationService_GetAPIKeys(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	seller := newTestSeller()
	seller.SellerID = "SLR-001"
	seller.Status = domain.SellerStatusActive
	key, _, _ := seller.GenerateAPIKey("Test Key", []string{"orders:read"}, nil)
	mockRepo.AddSeller(seller)

	ctx := context.Background()
	result, err := service.GetAPIKeys(ctx, "SLR-001")

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, key.KeyID, result[0].KeyID)
}

func TestSellerApplicationService_SearchSellers(t *testing.T) {
	mockRepo := newMockSellerRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewSellerApplicationService(mockRepo, logger)

	seller := newTestSeller()
	seller.SellerID = "SLR-001"
	mockRepo.AddSeller(seller)

	tests := []struct {
		name    string
		query   SearchSellersQuery
		wantLen int
	}{
		{
			name:    "Search with results",
			query:   SearchSellersQuery{Query: "Test", Page: 1, PageSize: 20},
			wantLen: 0,
		},
		{
			name:    "Search no results",
			query:   SearchSellersQuery{Query: "NonExistent", Page: 1, PageSize: 20},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := service.SearchSellers(ctx, tt.query)

			require.NoError(t, err)
			assert.Len(t, result.Data, tt.wantLen)
		})
	}
}

func newTestSeller() *domain.Seller {
	seller, err := domain.NewSeller("TNT-001", "Test Corp", "John Doe", "john@test.com", domain.BillingCycleMonthly)
	if err != nil {
		panic(err)
	}
	seller.Status = domain.SellerStatusActive
	return seller
}

func strPtr(s string) *string {
	return &s
}
