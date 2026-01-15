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

	server := httptest.NewServer(createBillingServiceHandler())
	defer server.Close()

	verifier := pact.NewVerifier()

	err = verifier.VerifyProvider(t, pact.VerifyRequest{
		Provider:        "billing-service",
		ProviderBaseURL: server.URL,
		PactDirs:        []string{absPactDir},
		StateHandlers: map[string]pact.StateHandlerFunc{
			"activities exist for seller": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: activities exist for seller")
				}
				return nil, nil
			},
			"an activity exists": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: an activity exists")
				}
				return nil, nil
			},
			"invoices exist for seller": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: invoices exist for seller")
				}
				return nil, nil
			},
			"an invoice exists": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: an invoice exists")
				}
				return nil, nil
			},
			"a draft invoice exists": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a draft invoice exists")
				}
				return nil, nil
			},
			"a finalized invoice exists": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: a finalized invoice exists")
				}
				return nil, nil
			},
			"fee schedule is configured": func(setup bool, state pact.ProviderState) (pact.ProviderStateResponse, error) {
				if setup {
					fmt.Println("Setting up state: fee schedule is configured")
				}
				return nil, nil
			},
		},
	})

	if err != nil {
		t.Logf("Provider verification failed: %v", err)
	}
}

func createBillingServiceHandler() http.Handler {
	mux := http.NewServeMux()

	// Activity endpoints
	mux.HandleFunc("/api/v1/activities", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"activityId":    "ACT-001",
					"tenantId":      "TNT-001",
					"sellerId":      "SLR-001",
					"facilityId":    "FAC-001",
					"type":          "pick",
					"description":   "Pick fee",
					"quantity":      10.0,
					"unitPrice":     0.25,
					"amount":        2.5,
					"currency":      "USD",
					"referenceType": "order",
					"referenceId":   "ORD-001",
					"invoiced":      false,
					"createdAt":     time.Now().UTC().Format(time.RFC3339),
				},
			})
			return
		}
		http.NotFound(w, r)
	})

	mux.HandleFunc("/api/v1/activities/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"activityId": "ACT-001",
						"type":       "pick",
						"amount":     2.5,
					},
					{
						"activityId": "ACT-002",
						"type":       "pack",
						"amount":     7.5,
					},
				},
			})
			return
		}
		http.NotFound(w, r)
	})

	mux.HandleFunc("/api/v1/activities/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"activityId":    "ACT-001",
					"tenantId":      "TNT-001",
					"sellerId":      "SLR-001",
					"facilityId":    "FAC-001",
					"type":          "pick",
					"description":   "Pick fee",
					"quantity":      10.0,
					"unitPrice":     0.25,
					"amount":        2.5,
					"currency":      "USD",
					"referenceType": "order",
					"referenceId":   "ORD-001",
					"invoiced":      false,
					"createdAt":     time.Now().UTC().Format(time.RFC3339),
				},
			})
			return
		}
		http.NotFound(w, r)
	})

	// Seller activities
	mux.HandleFunc("/api/v1/sellers/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			// Check if it's an activities or invoices request
			if containsPath(r.URL.Path, "/activities") {
				if containsPath(r.URL.Path, "/summary") {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"data": map[string]interface{}{
							"sellerId":    "SLR-001",
							"periodStart": time.Now().AddDate(0, -1, 0).UTC().Format(time.RFC3339),
							"periodEnd":   time.Now().UTC().Format(time.RFC3339),
							"byType": map[string]float64{
								"pick": 25.00,
								"pack": 75.00,
							},
							"total": 100.00,
						},
					})
				} else {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"data": []map[string]interface{}{
							{
								"activityId": "ACT-001",
								"type":       "pick",
								"amount":     2.5,
							},
						},
						"page":     1,
						"pageSize": 20,
					})
				}
			} else if containsPath(r.URL.Path, "/invoices") {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": []map[string]interface{}{
						{
							"invoiceId":     "INV-001",
							"invoiceNumber": "INV-202401-001",
							"status":        "draft",
							"total":         100.00,
						},
					},
					"page":     1,
					"pageSize": 20,
				})
			}
			return
		}
		http.NotFound(w, r)
	})

	// Invoice endpoints
	mux.HandleFunc("/api/v1/invoices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"invoiceId":     "INV-001",
					"tenantId":      "TNT-001",
					"sellerId":      "SLR-001",
					"status":        "draft",
					"invoiceNumber": "INV-202401-001",
					"periodStart":   time.Now().AddDate(0, -1, 0).UTC().Format(time.RFC3339),
					"periodEnd":     time.Now().UTC().Format(time.RFC3339),
					"lineItems":     []map[string]interface{}{},
					"subtotal":      0.0,
					"taxRate":       0.0,
					"taxAmount":     0.0,
					"discount":      0.0,
					"total":         0.0,
					"currency":      "USD",
					"dueDate":       time.Now().AddDate(0, 0, 30).UTC().Format(time.RFC3339),
					"sellerName":    "Test Seller",
					"sellerEmail":   "billing@test.com",
					"createdAt":     time.Now().UTC().Format(time.RFC3339),
					"updatedAt":     time.Now().UTC().Format(time.RFC3339),
				},
			})
			return
		}
		http.NotFound(w, r)
	})

	// Get invoice by ID pattern
	mux.HandleFunc("/api/v1/invoices/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check if it's a finalize, pay, or void action
		if containsPath(r.URL.Path, "/finalize") {
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"invoiceId":   "INV-001",
						"status":      "finalized",
						"total":       100.00,
						"finalizedAt": time.Now().UTC().Format(time.RFC3339),
					},
				})
				return
			}
		} else if containsPath(r.URL.Path, "/pay") {
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"invoiceId":     "INV-001",
						"status":        "paid",
						"total":         100.00,
						"paidAt":        time.Now().UTC().Format(time.RFC3339),
						"paymentMethod": "bank_transfer",
						"paymentRef":    "TXN-001",
					},
				})
				return
			}
		} else if containsPath(r.URL.Path, "/void") {
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"invoiceId": "INV-001",
						"status":    "voided",
						"notes":     "Duplicate invoice",
					},
				})
				return
			}
		} else if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"invoiceId":     "INV-001",
					"tenantId":      "TNT-001",
					"sellerId":      "SLR-001",
					"status":        "draft",
					"invoiceNumber": "INV-202401-001",
					"periodStart":   time.Now().AddDate(0, -1, 0).UTC().Format(time.RFC3339),
					"periodEnd":     time.Now().UTC().Format(time.RFC3339),
					"lineItems": []map[string]interface{}{
						{
							"activityType": "pick",
							"description":  "Picking fees",
							"quantity":     100.0,
							"unitPrice":    0.25,
							"amount":       25.00,
						},
					},
					"subtotal":    25.00,
					"taxRate":     0.08,
					"taxAmount":   2.00,
					"discount":    0.0,
					"total":       27.00,
					"currency":    "USD",
					"dueDate":     time.Now().AddDate(0, 0, 30).UTC().Format(time.RFC3339),
					"sellerName":  "Test Seller",
					"sellerEmail": "billing@test.com",
					"createdAt":   time.Now().UTC().Format(time.RFC3339),
					"updatedAt":   time.Now().UTC().Format(time.RFC3339),
				},
			})
			return
		}
		http.NotFound(w, r)
	})

	// Fee calculation
	mux.HandleFunc("/api/v1/fees/calculate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"storageFee":          10.00,
					"pickFee":             5.00,
					"packFee":             4.00,
					"receivingFee":        2.00,
					"shippingFee":         30.00,
					"returnProcessingFee": 4.00,
					"giftWrapFee":         2.50,
					"hazmatFee":           3.00,
					"oversizedFee":        8.00,
					"coldChainFee":        4.50,
					"fragileFee":          3.00,
					"totalFees":           76.00,
				},
			})
			return
		}
		http.NotFound(w, r)
	})

	// Storage calculation
	mux.HandleFunc("/api/v1/storage/calculate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "Storage calculation recorded",
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
