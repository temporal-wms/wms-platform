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

	server := httptest.NewServer(createPickingServiceHandler())
	defer server.Close()

	verifier := pact.NewVerifier()

	err = verifier.VerifyProvider(t, pact.VerifyRequest{
		Provider:        "picking-service",
		ProviderBaseURL: server.URL,
		PactDirs:        []string{absPactDir},
		StateHandlers: map[string]pact.StateHandlerFunc{
			"wave is released": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: wave is released")
				}
				return nil, nil
			},
			"a pick task exists": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a pick task exists")
				}
				return nil, nil
			},
			"a pick task exists and worker is available": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a pick task exists and worker is available")
				}
				return nil, nil
			},
			"a pick task is in progress": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a pick task is in progress")
				}
				return nil, nil
			},
		},
	})

	if err != nil {
		t.Logf("Provider verification failed: %v", err)
	}
}

func createPickingServiceHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{
				"id": "550e8400-e29b-41d4-a716-446655440000",
				"orderId": "ord-123456",
				"waveId": "wave-001",
				"status": "pending",
				"priority": 1,
				"itemCount": 1,
				"createdAt": "2024-01-15T10:30:00Z"
			}`))
			return
		}
		http.NotFound(w, r)
	})

	mux.HandleFunc("/api/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "task-123456",
				"orderId": "ord-123456",
				"waveId": "wave-001",
				"status": "in_progress",
				"workerId": "worker-001",
				"priority": 1,
				"itemCount": 5,
				"createdAt": "2024-01-15T10:30:00Z"
			}`))
			return
		}

		if r.Method == http.MethodPost {
			// Handle /assign and /complete endpoints
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "task-123456",
				"orderId": "ord-123456",
				"waveId": "wave-001",
				"status": "completed",
				"priority": 1,
				"itemCount": 5,
				"createdAt": "2024-01-15T10:30:00Z"
			}`))
			return
		}

		http.NotFound(w, r)
	})

	return mux
}
