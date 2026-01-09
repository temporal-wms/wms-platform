package activities_test

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/wms-platform/orchestrator/internal/activities/clients"
)

// MockServiceClients is a mock implementation of service clients for testing
type MockServiceClients struct {
	mock.Mock
}

// ValidateOrder mocks the ValidateOrder method
func (m *MockServiceClients) ValidateOrder(ctx context.Context, orderID string) (*clients.OrderValidationResult, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.OrderValidationResult), args.Error(1)
}

// GetOrder mocks the GetOrder method
func (m *MockServiceClients) GetOrder(ctx context.Context, orderID string) (*clients.Order, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Order), args.Error(1)
}

// CancelOrder mocks the CancelOrder method
func (m *MockServiceClients) CancelOrder(ctx context.Context, orderID, reason string) error {
	args := m.Called(ctx, orderID, reason)
	return args.Error(0)
}

// ReleaseInventoryReservation mocks the ReleaseInventoryReservation method
func (m *MockServiceClients) ReleaseInventoryReservation(ctx context.Context, orderID string) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

// CalculateRoute mocks the CalculateRoute method
func (m *MockServiceClients) CalculateRoute(ctx context.Context, req *clients.CalculateRouteRequest) (*clients.Route, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Route), args.Error(1)
}
