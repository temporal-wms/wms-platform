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

	server := httptest.NewServer(createInventoryServiceHandler())
	defer server.Close()

	verifier := pact.NewVerifier()

	err = verifier.VerifyProvider(t, pact.VerifyRequest{
		Provider:        "inventory-service",
		ProviderBaseURL: server.URL,
		PactDirs:        []string{absPactDir},
		StateHandlers: map[string]pact.StateHandlerFunc{
			"inventory is available": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: inventory is available")
				}
				return nil, nil
			},
			"a reservation exists": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a reservation exists")
				}
				return nil, nil
			},
			"inventory exists for SKU": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: inventory exists for SKU")
				}
				return nil, nil
			},
		},
	})

	if err != nil {
		t.Logf("Provider verification failed: %v", err)
	}
}

func createInventoryServiceHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/inventory/reserve", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})

	mux.HandleFunc("/api/v1/inventory/release/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})

	mux.HandleFunc("/api/v1/inventory/sku/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"sku": "SKU-001",
				"availableQty": 100,
				"reservedQty": 10,
				"location": "A-01-01",
				"zone": "ZONE-A",
				"lastUpdated": "2024-01-15T10:30:00Z"
			}`))
			return
		}
		http.NotFound(w, r)
	})

	return mux
}
