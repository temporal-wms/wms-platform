package activities_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/orchestrator/internal/activities"
	"go.temporal.io/sdk/testsuite"
)

func TestReleaseInventoryReservation_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/release/")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create activities with mock server
	config := &activities.ServiceClientsConfig{
		InventoryServiceURL: server.URL,
	}
	serviceClients := activities.NewServiceClients(config)
	inventoryActivities := activities.NewInventoryActivities(serviceClients, slog.Default())

	// Register activity
	env.RegisterActivity(inventoryActivities.ReleaseInventoryReservation)

	// Execute
	_, err := env.ExecuteActivity(inventoryActivities.ReleaseInventoryReservation, "ORD-001")

	// Assert
	require.NoError(t, err)
}

func TestReleaseInventoryReservation_Error(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Setup mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("inventory service unavailable"))
	}))
	defer server.Close()

	// Create activities with mock server
	config := &activities.ServiceClientsConfig{
		InventoryServiceURL: server.URL,
	}
	serviceClients := activities.NewServiceClients(config)
	inventoryActivities := activities.NewInventoryActivities(serviceClients, slog.Default())

	// Register activity
	env.RegisterActivity(inventoryActivities.ReleaseInventoryReservation)

	// Execute
	_, err := env.ExecuteActivity(inventoryActivities.ReleaseInventoryReservation, "ORD-001")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to release inventory reservation")
}
