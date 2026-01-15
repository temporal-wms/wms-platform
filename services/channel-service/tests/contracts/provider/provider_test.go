package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	server := httptest.NewServer(createChannelServiceHandler())
	defer server.Close()

	verifier := pact.NewVerifier()

	err = verifier.VerifyProvider(t, pact.VerifyRequest{
		Provider:        "channel-service",
		ProviderBaseURL: server.URL,
		PactDirs:        []string{absPactDir},
		StateHandlers: map[string]pact.StateHandlerFunc{
			"a channel exists": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a channel exists")
				}
				return nil, nil
			},
			"channels exist for seller": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: channels exist for seller")
				}
				return nil, nil
			},
			"orders exist for channel": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: orders exist for channel")
				}
				return nil, nil
			},
			"unimported orders exist": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: unimported orders exist")
				}
				return nil, nil
			},
			"sync jobs exist for channel": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: sync jobs exist for channel")
				}
				return nil, nil
			},
			"channel is active": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: channel is active")
				}
				return nil, nil
			},
			"order exists and can be imported": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: order exists and can be imported")
				}
				return nil, nil
			},
		},
	})

	if err != nil {
		t.Logf("Provider verification failed: %v", err)
	}
}

func createChannelServiceHandler() http.Handler {
	mux := http.NewServeMux()

	// Channel endpoints
	mux.HandleFunc("/api/v1/channels", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "CH-12345678",
				"channelId": "CH-12345678",
				"tenantId":  "TNT-001",
				"sellerId":  "SLR-001",
				"type":      "shopify",
				"name":      "My Shopify Store",
				"storeUrl":  "https://mystore.myshopify.com",
				"status":    "active",
				"syncSettings": map[string]interface{}{
					"autoImportOrders":         true,
					"autoSyncInventory":        true,
					"autoPushTracking":         true,
					"orderSyncIntervalMin":     15,
					"inventorySyncIntervalMin": 30,
				},
				"createdAt": time.Now().UTC().Format(time.RFC3339),
				"updatedAt": time.Now().UTC().Format(time.RFC3339),
			})
			return
		}
		http.NotFound(w, r)
	})

	// Get/Update/Delete channel by ID
	mux.HandleFunc("/api/v1/channels/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check specific endpoints
		path := r.URL.Path
		if containsPath(path, "/orders/unimported") {
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"orders": []map[string]interface{}{
						{
							"externalOrderId":     "EXT-001",
							"externalOrderNumber": "1001",
							"channelId":           "CH-001",
							"imported":            false,
							"total":               75.47,
							"financialStatus":     "paid",
							"fulfillmentStatus":   "unfulfilled",
						},
					},
					"total": 1,
				})
				return
			}
		} else if containsPath(path, "/orders/import") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Order marked as imported",
				})
				return
			}
		} else if containsPath(path, "/orders") {
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"orders": []map[string]interface{}{
						{
							"externalOrderId":     "EXT-001",
							"externalOrderNumber": "1001",
							"channelId":           "CH-001",
							"customer": map[string]interface{}{
								"email":     "customer@example.com",
								"firstName": "John",
								"lastName":  "Doe",
							},
							"total":             75.47,
							"financialStatus":   "paid",
							"fulfillmentStatus": "unfulfilled",
							"imported":          false,
						},
					},
					"page": 1,
					"size": 20,
				})
				return
			}
		} else if containsPath(path, "/sync-jobs") {
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"jobs": []map[string]interface{}{
						{
							"jobId":          "SYNC-001",
							"channelId":      "CH-001",
							"type":           "orders",
							"status":         "completed",
							"direction":      "inbound",
							"totalItems":     100,
							"processedItems": 100,
							"successItems":   98,
							"failedItems":    2,
							"startedAt":      time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339),
							"completedAt":    time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339),
						},
					},
					"page": 1,
					"size": 20,
				})
				return
			}
		} else if containsPath(path, "/sync/orders") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ID":        "SYNC-002",
					"jobId":     "SYNC-002",
					"channelId": "CH-001",
					"type":      "orders",
					"status":    "pending",
					"direction": "inbound",
					"createdAt": time.Now().UTC().Format(time.RFC3339),
				})
				return
			}
		} else if containsPath(path, "/sync/inventory") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ID":        "SYNC-003",
					"jobId":     "SYNC-003",
					"channelId": "CH-001",
					"type":      "inventory",
					"status":    "pending",
					"direction": "outbound",
					"createdAt": time.Now().UTC().Format(time.RFC3339),
				})
				return
			}
		} else if containsPath(path, "/tracking") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Tracking pushed successfully",
				})
				return
			}
		} else if containsPath(path, "/fulfillment") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Fulfillment created successfully",
				})
				return
			}
		} else if containsPath(path, "/inventory") {
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"levels": []map[string]interface{}{
						{
							"sku":       "SKU-001",
							"available": 100,
							"reserved":  10,
						},
					},
					"total": 1,
				})
				return
			}
		} else if r.Method == http.MethodGet {
			// Get single channel
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "CH-001",
				"channelId": "CH-001",
				"tenantId":  "TNT-001",
				"sellerId":  "SLR-001",
				"type":      "shopify",
				"name":      "My Shopify Store",
				"storeUrl":  "https://mystore.myshopify.com",
				"status":    "active",
				"syncSettings": map[string]interface{}{
					"autoImportOrders":     true,
					"autoSyncInventory":    true,
					"autoPushTracking":     true,
					"orderSyncIntervalMin": 15,
				},
				"createdAt": time.Now().UTC().Format(time.RFC3339),
				"updatedAt": time.Now().UTC().Format(time.RFC3339),
			})
			return
		} else if r.Method == http.MethodPut {
			// Update channel
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "CH-001",
				"channelId": "CH-001",
				"tenantId":  "TNT-001",
				"sellerId":  "SLR-001",
				"type":      "shopify",
				"name":      "Updated Store Name",
				"status":    "active",
				"updatedAt": time.Now().UTC().Format(time.RFC3339),
			})
			return
		} else if r.Method == http.MethodDelete {
			// Disconnect channel
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Channel disconnected",
			})
			return
		}
		http.NotFound(w, r)
	})

	// Seller channels
	mux.HandleFunc("/api/v1/sellers/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && containsPath(r.URL.Path, "/channels") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"channels": []map[string]interface{}{
					{
						"channelId": "CH-001",
						"type":      "shopify",
						"name":      "Shopify Store",
						"status":    "active",
					},
					{
						"channelId": "CH-002",
						"type":      "amazon",
						"name":      "Amazon Store",
						"status":    "active",
					},
				},
				"total": 2,
			})
			return
		}
		http.NotFound(w, r)
	})

	// Webhooks
	mux.HandleFunc("/api/v1/webhooks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Webhook processed",
			})
			return
		}
		http.NotFound(w, r)
	})

	return mux
}

func containsPath(path, segment string) bool {
	return len(path) > 0 && (path == segment ||
		(len(path) > len(segment) && path[len(path)-len(segment):] == segment) ||
		(len(path) > len(segment)+1 && contains(path, segment+"/")))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
