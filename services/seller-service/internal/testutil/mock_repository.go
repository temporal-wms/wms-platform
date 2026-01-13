package testutil

import (
	"context"
	"errors"

	"github.com/wms-platform/services/seller-service/internal/domain"
)

// MockSellerRepository is a mock implementation of domain.SellerRepository
type MockSellerRepository struct {
	sellers  map[string]*domain.Seller
	byEmail  map[string]*domain.Seller
	byAPIKey map[string]*domain.Seller

	SaveFunc           func(ctx context.Context, seller *domain.Seller) error
	FindByIDFunc       func(ctx context.Context, sellerID string) (*domain.Seller, error)
	FindByTenantIDFunc func(ctx context.Context, tenantID string, pagination domain.Pagination) ([]*domain.Seller, error)
	FindByStatusFunc   func(ctx context.Context, status domain.SellerStatus, pagination domain.Pagination) ([]*domain.Seller, error)
	FindByAPIKeyFunc   func(ctx context.Context, hashedKey string) (*domain.Seller, error)
	FindByEmailFunc    func(ctx context.Context, email string) (*domain.Seller, error)
	UpdateStatusFunc   func(ctx context.Context, sellerID string, status domain.SellerStatus) error
	DeleteFunc         func(ctx context.Context, sellerID string) error
	CountFunc          func(ctx context.Context, filter domain.SellerFilter) (int64, error)
	SearchFunc         func(ctx context.Context, query string, pagination domain.Pagination) ([]*domain.Seller, error)
}

// NewMockSellerRepository creates a new mock repository
func NewMockSellerRepository() *MockSellerRepository {
	return &MockSellerRepository{
		sellers:  make(map[string]*domain.Seller),
		byEmail:  make(map[string]*domain.Seller),
		byAPIKey: make(map[string]*domain.Seller),
	}
}

// AddSeller adds a seller to the mock repository
func (m *MockSellerRepository) AddSeller(seller *domain.Seller) {
	m.sellers[seller.SellerID] = seller
	m.byEmail[seller.ContactEmail] = seller
}

// Save implements domain.SellerRepository
func (m *MockSellerRepository) Save(ctx context.Context, seller *domain.Seller) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, seller)
	}
	m.sellers[seller.SellerID] = seller
	m.byEmail[seller.ContactEmail] = seller
	return nil
}

// FindByID implements domain.SellerRepository
func (m *MockSellerRepository) FindByID(ctx context.Context, sellerID string) (*domain.Seller, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, sellerID)
	}
	seller, ok := m.sellers[sellerID]
	if !ok {
		return nil, nil
	}
	return seller, nil
}

// FindByTenantID implements domain.SellerRepository
func (m *MockSellerRepository) FindByTenantID(ctx context.Context, tenantID string, pagination domain.Pagination) ([]*domain.Seller, error) {
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

// FindByStatus implements domain.SellerRepository
func (m *MockSellerRepository) FindByStatus(ctx context.Context, status domain.SellerStatus, pagination domain.Pagination) ([]*domain.Seller, error) {
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

// FindByAPIKey implements domain.SellerRepository
func (m *MockSellerRepository) FindByAPIKey(ctx context.Context, hashedKey string) (*domain.Seller, error) {
	if m.FindByAPIKeyFunc != nil {
		return m.FindByAPIKeyFunc(ctx, hashedKey)
	}
	return m.byAPIKey[hashedKey], nil
}

// FindByEmail implements domain.SellerRepository
func (m *MockSellerRepository) FindByEmail(ctx context.Context, email string) (*domain.Seller, error) {
	if m.FindByEmailFunc != nil {
		return m.FindByEmailFunc(ctx, email)
	}
	seller, ok := m.byEmail[email]
	if !ok {
		return nil, nil
	}
	return seller, nil
}

// UpdateStatus implements domain.SellerRepository
func (m *MockSellerRepository) UpdateStatus(ctx context.Context, sellerID string, status domain.SellerStatus) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, sellerID, status)
	}
	seller, ok := m.sellers[sellerID]
	if !ok {
		return errors.New("seller not found")
	}
	seller.Status = status
	return nil
}

// Delete implements domain.SellerRepository
func (m *MockSellerRepository) Delete(ctx context.Context, sellerID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, sellerID)
	}
	seller, ok := m.sellers[sellerID]
	if !ok {
		return errors.New("seller not found")
	}
	delete(m.sellers, sellerID)
	delete(m.byEmail, seller.ContactEmail)
	return nil
}

// Count implements domain.SellerRepository
func (m *MockSellerRepository) Count(ctx context.Context, filter domain.SellerFilter) (int64, error) {
	if m.CountFunc != nil {
		return m.CountFunc(ctx, filter)
	}
	count := int64(len(m.sellers))
	return count, nil
}

// Search implements domain.SellerRepository
func (m *MockSellerRepository) Search(ctx context.Context, query string, pagination domain.Pagination) ([]*domain.Seller, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, pagination)
	}
	var result []*domain.Seller
	for _, seller := range m.sellers {
		result = append(result, seller)
	}
	return result, nil
}

// GetOutboxRepository is not needed for tests
func (m *MockSellerRepository) GetOutboxRepository() interface{} {
	return nil
}
