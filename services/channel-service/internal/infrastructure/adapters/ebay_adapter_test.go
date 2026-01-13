package adapters

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wms-platform/services/channel-service/internal/domain"
)

func newEbayTestAdapter(server *httptest.Server) *EbayAdapter {
	return &EbayAdapter{
		httpClient: server.Client(),
		baseURL:    server.URL,
		authURL:    server.URL + "/auth/token",
	}
}

func TestEbayValidateCredentials(t *testing.T) {
	adapter := NewEbayAdapter()
	require.Equal(t, domain.ChannelTypeEbay, adapter.GetType())
	err := adapter.ValidateCredentials(context.Background(), domain.ChannelCredentials{})
	require.Error(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter = newEbayTestAdapter(server)
	creds := domain.ChannelCredentials{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "refresh",
	}
	err = adapter.ValidateCredentials(context.Background(), creds)
	require.NoError(t, err)
}

func TestEbayValidateCredentialsTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/token" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newEbayTestAdapter(server)
	creds := domain.ChannelCredentials{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "refresh",
	}
	err := adapter.ValidateCredentials(context.Background(), creds)
	require.Error(t, err)
}

func TestEbayValidateCredentialsTokenDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/token" {
			_, _ = w.Write([]byte("not-json"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newEbayTestAdapter(server)
	creds := domain.ChannelCredentials{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "refresh",
	}
	err := adapter.ValidateCredentials(context.Background(), creds)
	require.Error(t, err)
}

func TestEbayFetchOrders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/sell/fulfillment/v1/order":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"orders": []map[string]any{
					{
						"orderId":                 "ORDER-1",
						"legacyOrderId":           "LEGACY-1",
						"creationDate":            time.Now().UTC(),
						"orderFulfillmentStatus":  "FULFILLED",
						"pricingSummary": map[string]any{
							"total": map[string]any{
								"value":    "12.50",
								"currency": "USD",
							},
						},
						"buyer": map[string]any{
							"username": "buyer",
						},
						"fulfillmentStartInstructions": []map[string]any{
							{
								"shippingStep": map[string]any{
									"shipTo": map[string]any{
										"fullName": "Ada Lovelace",
										"contactAddress": map[string]any{
											"addressLine1":  "Main",
											"addressLine2":  "Unit 1",
											"city":          "City",
											"stateOrProvince": "ST",
											"postalCode":    "12345",
											"countryCode":   "US",
										},
										"primaryPhone": map[string]any{
											"phoneNumber": "555-1111",
										},
										"email": "buyer@example.com",
									},
								},
							},
						},
						"lineItems": []map[string]any{
							{
								"lineItemId":  "line-1",
								"legacyItemId": "item-1",
								"sku":         "sku-1",
								"title":       "Item",
								"quantity":    2,
								"lineItemCost": map[string]any{
									"value": "10.00",
								},
							},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newEbayTestAdapter(server)
	channel, err := domain.NewChannel("tenant-1", "seller-1", domain.ChannelTypeEbay, "Ebay", "", domain.ChannelCredentials{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "refresh",
	}, domain.SyncSettings{})
	require.NoError(t, err)

	orders, err := adapter.FetchOrders(context.Background(), channel, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, "ORDER-1", orders[0].ExternalOrderID)
	require.Equal(t, "Ada", orders[0].ShippingAddr.FirstName)
	require.Len(t, orders[0].LineItems, 1)
}

func TestEbayFetchOrdersError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/sell/fulfillment/v1/order":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newEbayTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeEbay,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
		},
	}

	_, err := adapter.FetchOrders(context.Background(), channel, time.Now())
	require.Error(t, err)
}

func TestEbayFetchOrderNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/sell/fulfillment/v1/order/ORDER-404":
			w.WriteHeader(http.StatusNotFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newEbayTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeEbay,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
		},
	}

	order, err := adapter.FetchOrder(context.Background(), channel, "ORDER-404")
	require.NoError(t, err)
	require.Nil(t, order)
}

func TestEbayFetchOrderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/sell/fulfillment/v1/order/ORDER-500":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newEbayTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeEbay,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
		},
	}

	_, err := adapter.FetchOrder(context.Background(), channel, "ORDER-500")
	require.Error(t, err)
}

func TestEbayFetchOrderNotImplemented(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/sell/fulfillment/v1/order/ORDER-OK":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newEbayTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeEbay,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
		},
	}

	_, err := adapter.FetchOrder(context.Background(), channel, "ORDER-OK")
	require.Error(t, err)
}

func TestEbayPushTrackingAndInventory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/sell/fulfillment/v1/order/ORDER-1/shipping_fulfillment":
			w.WriteHeader(http.StatusCreated)
		case "/sell/inventory/v1/inventory_item/sku-1":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"sku": "sku-1",
					"availability": map[string]any{
						"shipToLocationAvailability": map[string]any{
							"quantity": 2,
						},
					},
				})
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newEbayTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeEbay,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
		},
	}

	err := adapter.PushTracking(context.Background(), channel, "ORDER-1", domain.TrackingInfo{
		TrackingNumber: "track-1",
		Carrier:        "UPS",
		LineItemIDs:    []string{"line-1"},
	})
	require.NoError(t, err)

	err = adapter.CreateFulfillment(context.Background(), channel, domain.FulfillmentRequest{
		OrderID:        "ORDER-1",
		TrackingNumber: "track-1",
		Carrier:        "UPS",
		LineItems:      []domain.FulfillmentLineItem{{LineItemID: "line-1"}},
	})
	require.NoError(t, err)

	err = adapter.SyncInventory(context.Background(), channel, []domain.InventoryUpdate{{SKU: "sku-1", Available: 2}})
	require.NoError(t, err)

	levels, err := adapter.GetInventoryLevels(context.Background(), channel, []string{"sku-1"})
	require.NoError(t, err)
	require.Len(t, levels, 1)
}

func TestEbayMapCarrierCodeDefault(t *testing.T) {
	adapter := NewEbayAdapter()
	require.Equal(t, "CarrierX", adapter.mapCarrierCode("CarrierX"))
}

func TestEbayPushTrackingError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/sell/fulfillment/v1/order/ORDER-1/shipping_fulfillment":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newEbayTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeEbay,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
		},
	}

	err := adapter.PushTracking(context.Background(), channel, "ORDER-1", domain.TrackingInfo{
		TrackingNumber: "track-1",
		Carrier:        "UPS",
	})
	require.Error(t, err)
}

func TestEbayRegisterWebhooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/commerce/notification/v1/subscription":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newEbayTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeEbay,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
		},
	}

	err := adapter.RegisterWebhooks(context.Background(), channel, "dest-1")
	require.NoError(t, err)
}

func TestEbayValidateWebhook(t *testing.T) {
	adapter := NewEbayAdapter()
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{ClientSecret: "secret"},
	}
	body := []byte("payload")

	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write(body)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	require.True(t, adapter.ValidateWebhook(context.Background(), channel, signature, body))
	require.False(t, adapter.ValidateWebhook(context.Background(), channel, "", body))
}
