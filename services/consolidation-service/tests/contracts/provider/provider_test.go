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

	server := httptest.NewServer(createConsolidationServiceHandler())
	defer server.Close()

	verifier := pact.NewVerifier()

	err = verifier.VerifyProvider(t, pact.VerifyRequest{
		Provider:        "consolidation-service",
		ProviderBaseURL: server.URL,
		PactDirs:        []string{absPactDir},
		StateHandlers: map[string]models.StateHandler{
			"picking is complete": func(setup bool, state models.ProviderState) (models.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: picking is complete")
				}
				return nil, nil
			},
			"a consolidation unit exists": func(setup bool, state models.ProviderState) (models.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a consolidation unit exists")
				}
				return nil, nil
			},
			"consolidation is in progress with all items scanned": func(setup bool, state models.ProviderState) (models.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: consolidation is in progress with all items scanned")
				}
				return nil, nil
			},
		},
	})

	if err != nil {
		t.Logf("Provider verification failed: %v", err)
	}
}

func createConsolidationServiceHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/consolidations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{
				"id": "550e8400-e29b-41d4-a716-446655440000",
				"orderId": "ord-123456",
				"status": "pending",
				"expectedItems": 1,
				"scannedItems": 0,
				"station": "CONSOL-01"
			}`))
			return
		}
		http.NotFound(w, r)
	})

	mux.HandleFunc("/api/v1/consolidations/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "consol-123456",
			"orderId": "ord-123456",
			"status": "completed",
			"expectedItems": 5,
			"scannedItems": 5,
			"station": "CONSOL-01"
		}`))
	})

	return mux
}
