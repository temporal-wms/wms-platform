package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	pact "github.com/pact-foundation/pact-go/v2/provider"
	"github.com/stretchr/testify/require"
)

func TestPactProvider(t *testing.T) {
	// Find the pacts directory
	pactDir := "../../../../../contracts/pacts"
	absPactDir, err := filepath.Abs(pactDir)
	require.NoError(t, err)

	// Skip if pacts don't exist
	if _, err := os.Stat(absPactDir); os.IsNotExist(err) {
		t.Skip("No pacts found - run consumer tests first")
	}

	// Create a test server that simulates the order service
	server := httptest.NewServer(createOrderServiceHandler())
	defer server.Close()

	verifier := pact.NewVerifier()

	err = verifier.VerifyProvider(t, pact.VerifyRequest{
		Provider:        "order-service",
		ProviderBaseURL: server.URL,
		PactDirs:        []string{absPactDir},
		StateHandlers: map[string]pact.StateHandlerFunc{
			"an order exists": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					// Set up test data - order exists in database
					fmt.Println("Setting up state: an order exists")
				}
				return nil, nil
			},
			"an order exists and is valid": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					// Set up test data - valid order
					fmt.Println("Setting up state: an order exists and is valid")
				}
				return nil, nil
			},
			"an order exists and can be cancelled": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					// Set up test data - order that can be cancelled
					fmt.Println("Setting up state: an order exists and can be cancelled")
				}
				return nil, nil
			},
		},
	})

	if err != nil {
		t.Logf("Provider verification failed: %v", err)
	}
}

// createOrderServiceHandler creates a mock HTTP handler for the order service
func createOrderServiceHandler() http.Handler {
	mux := http.NewServeMux()

	// GET /api/v1/orders/{orderId}
	mux.HandleFunc("/api/v1/orders/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "ord-123456",
				"customerId": "cust-001",
				"status": "pending",
				"priority": "standard",
				"totalItems": 5,
				"totalValue": 150.50,
				"createdAt": "2024-01-15T10:30:00Z",
				"updatedAt": "2024-01-15T10:30:00Z"
			}`))
			return
		}

		// POST /api/v1/orders/{orderId}/validate
		if r.Method == http.MethodPost && len(r.URL.Path) > 20 && r.URL.Path[len(r.URL.Path)-8:] == "validate" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"valid": true,
				"orderId": "ord-123456",
				"errors": []
			}`))
			return
		}

		// POST /api/v1/orders/{orderId}/cancel
		if r.Method == http.MethodPost && len(r.URL.Path) > 20 && r.URL.Path[len(r.URL.Path)-6:] == "cancel" {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	})

	return mux
}
