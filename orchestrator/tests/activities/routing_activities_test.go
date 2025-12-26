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

func TestCalculateRoute_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/routes", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clients.Route{
			RouteID:           "RT-12345678",
			EstimatedDistance: 150.5,
			Strategy:          "shortest_path",
			Stops: []clients.RouteStop{
				{LocationID: "LOC-A1", SKU: "SKU-001", Quantity: 2},
				{LocationID: "LOC-B2", SKU: "SKU-002", Quantity: 1},
			},
		})
	}))
	defer server.Close()

	// Create activities with mock server
	config := &activities.ServiceClientsConfig{
		RoutingServiceURL: server.URL,
	}
	serviceClients := activities.NewServiceClients(config)
	routingActivities := activities.NewRoutingActivities(serviceClients, slog.Default())

	// Register activity
	env.RegisterActivity(routingActivities.CalculateRoute)

	// Execute
	input := map[string]interface{}{
		"orderId": "ORD-001",
		"waveId":  "WAVE-001",
		"items": []interface{}{
			map[string]interface{}{"sku": "SKU-001", "quantity": float64(2)},
			map[string]interface{}{"sku": "SKU-002", "quantity": float64(1)},
		},
	}
	result, err := env.ExecuteActivity(routingActivities.CalculateRoute, input)

	// Assert
	require.NoError(t, err)
	var routeResult workflows.RouteResult
	require.NoError(t, result.Get(&routeResult))
	assert.Equal(t, "RT-12345678", routeResult.RouteID)
	assert.Equal(t, "shortest_path", routeResult.Strategy)
	assert.Len(t, routeResult.Stops, 2)
}

func TestCalculateRoute_Error(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Setup mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("routing service unavailable"))
	}))
	defer server.Close()

	// Create activities with mock server
	config := &activities.ServiceClientsConfig{
		RoutingServiceURL: server.URL,
	}
	serviceClients := activities.NewServiceClients(config)
	routingActivities := activities.NewRoutingActivities(serviceClients, slog.Default())

	// Register activity
	env.RegisterActivity(routingActivities.CalculateRoute)

	// Execute
	input := map[string]interface{}{
		"orderId": "ORD-001",
		"waveId":  "WAVE-001",
		"items": []interface{}{
			map[string]interface{}{"sku": "SKU-001", "quantity": float64(2)},
		},
	}
	_, err := env.ExecuteActivity(routingActivities.CalculateRoute, input)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "route calculation failed")
}
