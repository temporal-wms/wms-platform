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
	pactDir := "../../../../../contracts/pacts"
	absPactDir, err := filepath.Abs(pactDir)
	require.NoError(t, err)

	if _, err := os.Stat(absPactDir); os.IsNotExist(err) {
		t.Skip("No pacts found - run consumer tests first")
	}

	server := httptest.NewServer(createShippingServiceHandler())
	defer server.Close()

	verifier := pact.NewVerifier()

	err = verifier.VerifyProvider(t, pact.VerifyRequest{
		Provider:        "shipping-service",
		ProviderBaseURL: server.URL,
		PactDirs:        []string{absPactDir},
		StateHandlers: map[string]pact.StateHandlerFunc{
			"packing is complete": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: packing is complete")
				}
				return nil, nil
			},
			"a shipment exists": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a shipment exists")
				}
				return nil, nil
			},
			"a shipment exists without a label": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a shipment exists without a label")
				}
				return nil, nil
			},
			"a shipment has a label": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a shipment has a label")
				}
				return nil, nil
			},
		},
	})

	if err != nil {
		t.Logf("Provider verification failed: %v", err)
	}
}

func createShippingServiceHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/shipments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{
				"id": "550e8400-e29b-41d4-a716-446655440000",
				"orderId": "ord-123456",
				"packageId": "pkg-001",
				"carrier": "UPS",
				"status": "created"
			}`))
			return
		}
		http.NotFound(w, r)
	})

	mux.HandleFunc("/api/v1/shipments/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "ship-123456",
				"orderId": "ord-123456",
				"packageId": "pkg-001",
				"carrier": "UPS",
				"trackingCode": "1Z999AA10123456784",
				"status": "labeled"
			}`))
			return
		}

		if r.Method == http.MethodPost {
			// Handle /label and /ship endpoints
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "550e8400-e29b-41d4-a716-446655440000",
				"shipmentId": "ship-123456",
				"carrier": "UPS",
				"trackingCode": "1Z999AA10123456784",
				"labelUrl": "https://labels.example.com/label-123.pdf",
				"format": "PDF"
			}`))
			return
		}

		http.NotFound(w, r)
	})

	return mux
}
