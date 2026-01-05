package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"github.com/wms-platform/orchestrator/internal/workflows"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

// MockOrderClient is a mock implementation of the order client
type MockOrderClient struct {
	mock.Mock
}

func (m *MockOrderClient) ValidateOrder(ctx context.Context, orderID string) (*clients.OrderValidationResult, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.OrderValidationResult), args.Error(1)
}

func (m *MockOrderClient) GetOrder(ctx context.Context, orderID string) (*clients.Order, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Order), args.Error(1)
}

func (m *MockOrderClient) CancelOrder(ctx context.Context, orderID string, reason string) error {
	args := m.Called(ctx, orderID, reason)
	return args.Error(0)
}

func (m *MockOrderClient) AssignToWave(ctx context.Context, orderID string, waveID string) error {
	args := m.Called(ctx, orderID, waveID)
	return args.Error(0)
}

func (m *MockOrderClient) StartPicking(ctx context.Context, orderID string) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

func (m *MockOrderClient) MarkConsolidated(ctx context.Context, orderID string) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

func (m *MockOrderClient) MarkPacked(ctx context.Context, orderID string) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

// TestValidateOrder_Success tests successful order validation
func TestValidateOrder_Success(t *testing.T) {
	testSuite := &testsuite.ActivityTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockClient := new(MockOrderClient)
	mockClient.On("ValidateOrder", mock.Anything, "ORD-001").Return(
		&clients.OrderValidationResult{
			Valid:  true,
			Errors: []string{},
		}, nil,
	)

	activities := &OrderActivities{
		clients: &clients.Clients{
			OrderClient: mockClient,
		},
	}
	env.RegisterActivity(activities.ValidateOrder)

	input := workflows.OrderFulfillmentInput{
		OrderID:    "ORD-001",
		CustomerID: "CUST-001",
		Items: []workflows.Item{
			{SKU: "SKU-001", Quantity: 1},
		},
	}

	val, err := env.ExecuteActivity(activities.ValidateOrder, input)
	require.NoError(t, err)

	var result bool
	require.NoError(t, val.Get(&result))
	require.True(t, result)

	mockClient.AssertExpectations(t)
}

// TestValidateOrder_InvalidOrder tests order validation failure with ApplicationError
func TestValidateOrder_InvalidOrder(t *testing.T) {
	testSuite := &testsuite.ActivityTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockClient := new(MockOrderClient)
	mockClient.On("ValidateOrder", mock.Anything, "ORD-002").Return(
		&clients.OrderValidationResult{
			Valid:  false,
			Errors: []string{"invalid SKU: SKU-999", "quantity must be positive"},
		}, nil,
	)

	activities := &OrderActivities{
		clients: &clients.Clients{
			OrderClient: mockClient,
		},
	}
	env.RegisterActivity(activities.ValidateOrder)

	input := workflows.OrderFulfillmentInput{
		OrderID: "ORD-002",
		Items: []workflows.Item{
			{SKU: "SKU-999", Quantity: -1},
		},
	}

	val, err := env.ExecuteActivity(activities.ValidateOrder, input)
	require.Error(t, err)

	// Verify it's an ApplicationError (non-retryable)
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "Expected ApplicationError for business validation failure")
	require.Equal(t, "OrderValidationFailed", appErr.Type())
	require.Contains(t, appErr.Message(), "invalid SKU")

	var result bool
	val.Get(&result)
	require.False(t, result)

	mockClient.AssertExpectations(t)
}

// TestValidateOrder_ServiceError tests system error (retryable)
func TestValidateOrder_ServiceError(t *testing.T) {
	testSuite := &testsuite.ActivityTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockClient := new(MockOrderClient)
	mockClient.On("ValidateOrder", mock.Anything, "ORD-003").Return(
		nil, errors.New("connection timeout"),
	)

	activities := &OrderActivities{
		clients: &clients.Clients{
			OrderClient: mockClient,
		},
	}
	env.RegisterActivity(activities.ValidateOrder)

	input := workflows.OrderFulfillmentInput{
		OrderID: "ORD-003",
	}

	val, err := env.ExecuteActivity(activities.ValidateOrder, input)
	require.Error(t, err)

	// Verify it's NOT an ApplicationError (retryable system error)
	var appErr *temporal.ApplicationError
	require.False(t, errors.As(err, &appErr), "System errors should NOT be ApplicationError")
	require.Contains(t, err.Error(), "connection timeout")

	var result bool
	val.Get(&result)
	require.False(t, result)

	mockClient.AssertExpectations(t)
}

// TestCancelOrder_Success tests successful order cancellation
func TestCancelOrder_Success(t *testing.T) {
	testSuite := &testsuite.ActivityTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockClient := new(MockOrderClient)
	mockClient.On("CancelOrder", mock.Anything, "ORD-001", "Customer requested").Return(nil)

	activities := &OrderActivities{
		clients: &clients.Clients{
			OrderClient: mockClient,
		},
	}
	env.RegisterActivity(activities.CancelOrder)

	val, err := env.ExecuteActivity(activities.CancelOrder, "ORD-001", "Customer requested")
	require.NoError(t, err)
	require.NoError(t, val.Get(nil))

	mockClient.AssertExpectations(t)
}

// TestAssignToWave_Success tests successful wave assignment
func TestAssignToWave_Success(t *testing.T) {
	testSuite := &testsuite.ActivityTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockClient := new(MockOrderClient)
	mockClient.On("AssignToWave", mock.Anything, "ORD-001", "WAVE-001").Return(nil)

	activities := &OrderActivities{
		clients: &clients.Clients{
			OrderClient: mockClient,
		},
	}
	env.RegisterActivity(activities.AssignToWave)

	val, err := env.ExecuteActivity(activities.AssignToWave, "ORD-001", "WAVE-001")
	require.NoError(t, err)
	require.NoError(t, val.Get(nil))

	mockClient.AssertExpectations(t)
}

// TestStartPicking_Success tests marking order as picking
func TestStartPicking_Success(t *testing.T) {
	testSuite := &testsuite.ActivityTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockClient := new(MockOrderClient)
	mockClient.On("StartPicking", mock.Anything, "ORD-001").Return(nil)

	activities := &OrderActivities{
		clients: &clients.Clients{
			OrderClient: mockClient,
		},
	}
	env.RegisterActivity(activities.StartPicking)

	val, err := env.ExecuteActivity(activities.StartPicking, "ORD-001")
	require.NoError(t, err)
	require.NoError(t, val.Get(nil))

	mockClient.AssertExpectations(t)
}

// TestNotifyCustomerCancellation_Success tests customer notification
func TestNotifyCustomerCancellation_Success(t *testing.T) {
	testSuite := &testsuite.ActivityTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockClient := new(MockOrderClient)
	mockClient.On("GetOrder", mock.Anything, "ORD-001").Return(
		&clients.Order{
			OrderID:    "ORD-001",
			CustomerID: "CUST-001",
		}, nil,
	)

	activities := &OrderActivities{
		clients: &clients.Clients{
			OrderClient: mockClient,
		},
	}
	env.RegisterActivity(activities.NotifyCustomerCancellation)

	val, err := env.ExecuteActivity(activities.NotifyCustomerCancellation, "ORD-001", "Out of stock")
	require.NoError(t, err)
	require.NoError(t, val.Get(nil))

	mockClient.AssertExpectations(t)
}

// TestNotifyCustomerCancellation_GetOrderFails tests graceful handling when getting order fails
func TestNotifyCustomerCancellation_GetOrderFails(t *testing.T) {
	testSuite := &testsuite.ActivityTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockClient := new(MockOrderClient)
	mockClient.On("GetOrder", mock.Anything, "ORD-001").Return(
		nil, errors.New("order not found"),
	)

	activities := &OrderActivities{
		clients: &clients.Clients{
			OrderClient: mockClient,
		},
	}
	env.RegisterActivity(activities.NotifyCustomerCancellation)

	// Should not fail - notification is best-effort
	val, err := env.ExecuteActivity(activities.NotifyCustomerCancellation, "ORD-001", "Out of stock")
	require.NoError(t, err)
	require.NoError(t, val.Get(nil))

	mockClient.AssertExpectations(t)
}
