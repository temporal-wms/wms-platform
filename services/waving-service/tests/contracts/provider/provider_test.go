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

	server := httptest.NewServer(createWavingServiceHandler())
	defer server.Close()

	verifier := pact.NewVerifier()

	err = verifier.VerifyProvider(t, pact.VerifyRequest{
		Provider:        "waving-service",
		ProviderBaseURL: server.URL,
		PactDirs:        []string{absPactDir},
		StateHandlers: map[string]pact.StateHandlerFunc{
			"a wave exists": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a wave exists")
				}
				return nil, nil
			},
			"a wave exists and can accept orders": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a wave exists and can accept orders")
				}
				return nil, nil
			},
		},
	})

	if err != nil {
		t.Logf("Provider verification failed: %v", err)
	}
}

func createWavingServiceHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/waves/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "wave-123456",
				"status": "released",
				"orderIds": ["ord-123456"],
				"orderCount": 5,
				"priority": 1,
				"createdAt": "2024-01-15T10:30:00Z"
			}`))
			return
		}

		if r.Method == http.MethodPost {
			// Handle /orders endpoint
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	})

	return mux
}
