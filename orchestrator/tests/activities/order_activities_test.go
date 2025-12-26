package activities_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/orchestrator/internal/activities"
	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"github.com/wms-platform/orchestrator/internal/workflows"
	"go.temporal.io/sdk/testsuite"
)

func TestValidateOrder_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/validate")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clients.OrderValidationResult{Valid: true})
	}))
	defer server.Close()

	// Create activities with mock server
	config := &activities.ServiceClientsConfig{
		OrderServiceURL: server.URL,
	}
	serviceClients := activities.NewServiceClients(config)
	orderActivities := activities.NewOrderActivities(serviceClients, slog.Default())

	// Register activity
	env.RegisterActivity(orderActivities.ValidateOrder)

	// Execute
	input := workflows.OrderFulfillmentInput{
		OrderID:    "ORD-001",
		CustomerID: "CUST-001",
	}
	result, err := env.ExecuteActivity(orderActivities.ValidateOrder, input)

	// Assert
	require.NoError(t, err)
	var valid bool
	require.NoError(t, result.Get(&valid))
	assert.True(t, valid)
}

func TestValidateOrder_Error(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Setup mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	// Create activities with mock server
	config := &activities.ServiceClientsConfig{
		OrderServiceURL: server.URL,
	}
	serviceClients := activities.NewServiceClients(config)
	orderActivities := activities.NewOrderActivities(serviceClients, slog.Default())

	// Register activity
	env.RegisterActivity(orderActivities.ValidateOrder)

	// Execute
	input := workflows.OrderFulfillmentInput{
		OrderID:    "ORD-001",
		CustomerID: "CUST-001",
	}
	_, err := env.ExecuteActivity(orderActivities.ValidateOrder, input)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order validation failed")
}

func TestCancelOrder_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/cancel")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create activities with mock server
	config := &activities.ServiceClientsConfig{
		OrderServiceURL: server.URL,
	}
	serviceClients := activities.NewServiceClients(config)
	orderActivities := activities.NewOrderActivities(serviceClients, slog.Default())

	// Register activity
	env.RegisterActivity(orderActivities.CancelOrder)

	// Execute
	_, err := env.ExecuteActivity(orderActivities.CancelOrder, "ORD-001", "customer request")

	// Assert
	require.NoError(t, err)
}

func TestCancelOrder_Error(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Setup mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("order not found"))
	}))
	defer server.Close()

	// Create activities with mock server
	config := &activities.ServiceClientsConfig{
		OrderServiceURL: server.URL,
	}
	serviceClients := activities.NewServiceClients(config)
	orderActivities := activities.NewOrderActivities(serviceClients, slog.Default())

	// Register activity
	env.RegisterActivity(orderActivities.CancelOrder)

	// Execute
	_, err := env.ExecuteActivity(orderActivities.CancelOrder, "ORD-001", "customer request")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to cancel order")
}

func TestNotifyCustomerCancellation_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clients.Order{
			OrderID:    "ORD-001",
			CustomerID: "CUST-001",
		})
	}))
	defer server.Close()

	// Create activities with mock server
	config := &activities.ServiceClientsConfig{
		OrderServiceURL: server.URL,
	}
	serviceClients := activities.NewServiceClients(config)
	orderActivities := activities.NewOrderActivities(serviceClients, slog.Default())

	// Register activity
	env.RegisterActivity(orderActivities.NotifyCustomerCancellation)

	// Execute
	_, err := env.ExecuteActivity(orderActivities.NotifyCustomerCancellation, "ORD-001", "customer request")

	// Assert - this is best-effort, always returns nil
	require.NoError(t, err)
}

func TestNotifyCustomerCancellation_BestEffort(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Setup mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create activities with mock server
	config := &activities.ServiceClientsConfig{
		OrderServiceURL: server.URL,
	}
	serviceClients := activities.NewServiceClients(config)
	orderActivities := activities.NewOrderActivities(serviceClients, slog.Default())

	// Register activity
	env.RegisterActivity(orderActivities.NotifyCustomerCancellation)

	// Execute
	_, err := env.ExecuteActivity(orderActivities.NotifyCustomerCancellation, "ORD-001", "customer request")

	// Assert - best-effort behavior means it returns nil even on failure
	require.NoError(t, err)
}
