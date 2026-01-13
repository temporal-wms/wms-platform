package adapters

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wms-platform/services/channel-service/internal/domain"
)

func newAmazonTestAdapter(server *httptest.Server) *AmazonAdapter {
	return &AmazonAdapter{
		httpClient: server.Client(),
		baseURL:    server.URL,
		authURL:    server.URL + "/auth/token",
	}
}

func TestAmazonValidateCredentials(t *testing.T) {
	adapter := NewAmazonAdapter()
	require.Equal(t, domain.ChannelTypeAmazon, adapter.GetType())

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

	adapter = newAmazonTestAdapter(server)
	creds := domain.ChannelCredentials{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "refresh",
		AdditionalConfig: map[string]interface{}{
			"seller_id":      "seller",
			"marketplace_id": "market",
		},
	}
	err = adapter.ValidateCredentials(context.Background(), creds)
	require.NoError(t, err)
}

func TestAmazonValidateCredentialsMissingConfig(t *testing.T) {
	adapter := NewAmazonAdapter()
	err := adapter.ValidateCredentials(context.Background(), domain.ChannelCredentials{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "refresh",
	})
	require.Error(t, err)
}

func TestAmazonValidateCredentialsTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/token" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	creds := domain.ChannelCredentials{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "refresh",
		AdditionalConfig: map[string]interface{}{
			"seller_id":      "seller",
			"marketplace_id": "market",
		},
	}
	err := adapter.ValidateCredentials(context.Background(), creds)
	require.Error(t, err)
}

func TestAmazonValidateCredentialsTokenDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/token" {
			_, _ = w.Write([]byte("not-json"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	creds := domain.ChannelCredentials{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "refresh",
		AdditionalConfig: map[string]interface{}{
			"seller_id":      "seller",
			"marketplace_id": "market",
		},
	}
	err := adapter.ValidateCredentials(context.Background(), creds)
	require.Error(t, err)
}

func TestAmazonFetchOrders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/orders/v0/orders":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"Orders": []map[string]any{
						{
							"AmazonOrderId": "ORDER-1",
							"OrderStatus":   "Unshipped",
							"PurchaseDate":  time.Now().UTC(),
							"OrderTotal": map[string]any{
								"Amount":       "12.50",
								"CurrencyCode": "USD",
							},
							"ShippingAddress": map[string]any{
								"Name":          "Ada Lovelace",
								"AddressLine1":  "Main",
								"AddressLine2":  "Unit 1",
								"City":          "City",
								"StateOrRegion": "ST",
								"PostalCode":    "12345",
								"CountryCode":   "US",
								"Phone":         "555-1111",
							},
							"BuyerInfo": map[string]any{
								"BuyerEmail": "buyer@example.com",
								"BuyerName":  "Ada Lovelace",
							},
						},
					},
				},
			})
		case "/orders/v0/orders/ORDER-1/orderItems":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"OrderItems": []map[string]any{
						{
							"ASIN":            "ASIN-1",
							"SellerSKU":       "sku-1",
							"OrderItemId":     "item-1",
							"Title":           "Item",
							"QuantityOrdered": 2,
							"ItemPrice": map[string]any{
								"Amount": "10.00",
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

	adapter := newAmazonTestAdapter(server)
	channel, err := domain.NewChannel("tenant-1", "seller-1", domain.ChannelTypeAmazon, "Amazon", "", domain.ChannelCredentials{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "refresh",
		AdditionalConfig: map[string]interface{}{
			"seller_id":      "seller",
			"marketplace_id": "market",
		},
	}, domain.SyncSettings{})
	require.NoError(t, err)

	orders, err := adapter.FetchOrders(context.Background(), channel, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, "ORDER-1", orders[0].ExternalOrderID)
	require.Equal(t, "Ada", orders[0].ShippingAddr.FirstName)
	require.Len(t, orders[0].LineItems, 1)
}

func TestAmazonFetchOrdersItemError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/orders/v0/orders":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"Orders": []map[string]any{
						{
							"AmazonOrderId": "ORDER-1",
							"OrderStatus":   "Unshipped",
							"PurchaseDate":  time.Now().UTC(),
						},
					},
				},
			})
		case "/orders/v0/orders/ORDER-1/orderItems":
			w.WriteHeader(http.StatusBadRequest)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeAmazon,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
			AdditionalConfig: map[string]interface{}{
				"seller_id":      "seller",
				"marketplace_id": "market",
			},
		},
	}

	orders, err := adapter.FetchOrders(context.Background(), channel, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Len(t, orders[0].LineItems, 0)
}

func TestAmazonFetchOrdersError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/orders/v0/orders":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeAmazon,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
			AdditionalConfig: map[string]interface{}{
				"seller_id":      "seller",
				"marketplace_id": "market",
			},
		},
	}

	_, err := adapter.FetchOrders(context.Background(), channel, time.Now().Add(-time.Hour))
	require.Error(t, err)
}

func TestAmazonFetchOrderNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/orders/v0/orders/ORDER-2":
			w.WriteHeader(http.StatusNotFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeAmazon,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
			AdditionalConfig: map[string]interface{}{
				"seller_id":      "seller",
				"marketplace_id": "market",
			},
		},
	}

	order, err := adapter.FetchOrder(context.Background(), channel, "ORDER-2")
	require.NoError(t, err)
	require.Nil(t, order)
}

func TestAmazonFetchOrderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/orders/v0/orders/ORDER-500":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeAmazon,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
			AdditionalConfig: map[string]interface{}{
				"seller_id":      "seller",
				"marketplace_id": "market",
			},
		},
	}

	_, err := adapter.FetchOrder(context.Background(), channel, "ORDER-500")
	require.Error(t, err)
}

func TestAmazonFetchOrderNotImplemented(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/orders/v0/orders/ORDER-OK":
			_ = json.NewEncoder(w).Encode(map[string]any{"payload": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeAmazon,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
			AdditionalConfig: map[string]interface{}{
				"seller_id":      "seller",
				"marketplace_id": "market",
			},
		},
	}

	_, err := adapter.FetchOrder(context.Background(), channel, "ORDER-OK")
	require.Error(t, err)
}

func TestAmazonPushTrackingAndInventory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/feeds/2021-06-30/feeds":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeAmazon,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
			AdditionalConfig: map[string]interface{}{
				"seller_id":      "seller",
				"marketplace_id": "market",
			},
		},
	}

	err := adapter.PushTracking(context.Background(), channel, "ORDER-1", domain.TrackingInfo{
		TrackingNumber: "track-1",
		Carrier:        "ups",
	})
	require.NoError(t, err)

	err = adapter.SyncInventory(context.Background(), channel, []domain.InventoryUpdate{{SKU: "sku-1", Available: 2}})
	require.NoError(t, err)

	err = adapter.CreateFulfillment(context.Background(), channel, domain.FulfillmentRequest{
		OrderID:        "ORDER-1",
		TrackingNumber: "track-1",
		Carrier:        "ups",
	})
	require.NoError(t, err)

	require.NoError(t, adapter.RegisterWebhooks(context.Background(), channel, "https://example.com/webhook"))
}

func TestAmazonPushTrackingError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/feeds/2021-06-30/feeds":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeAmazon,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
			AdditionalConfig: map[string]interface{}{
				"seller_id":      "seller",
				"marketplace_id": "market",
			},
		},
	}

	err := adapter.PushTracking(context.Background(), channel, "ORDER-1", domain.TrackingInfo{
		TrackingNumber: "track-1",
		Carrier:        "ups",
	})
	require.Error(t, err)
}

func TestAmazonSyncInventoryError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/feeds/2021-06-30/feeds":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeAmazon,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
			AdditionalConfig: map[string]interface{}{
				"seller_id":      "seller",
				"marketplace_id": "market",
			},
		},
	}

	err := adapter.SyncInventory(context.Background(), channel, []domain.InventoryUpdate{{SKU: "sku-1", Available: 2}})
	require.Error(t, err)
}

func TestAmazonGetInventoryLevels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/fba/inventory/v1/summaries":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"inventorySummaries": []map[string]any{
						{
							"sellerSku":         "sku-1",
							"asin":              "ASIN-1",
							"totalQuantity":     5,
							"availableQuantity": 3,
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeAmazon,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
			AdditionalConfig: map[string]interface{}{
				"seller_id":      "seller",
				"marketplace_id": "market",
			},
		},
	}

	levels, err := adapter.GetInventoryLevels(context.Background(), channel, []string{"sku-1"})
	require.NoError(t, err)
	require.Len(t, levels, 1)
	require.Equal(t, "sku-1", levels[0].SKU)
}

func TestAmazonGetInventoryLevelsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   3600,
			})
		case "/fba/inventory/v1/summaries":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newAmazonTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeAmazon,
		Credentials: domain.ChannelCredentials{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
			AdditionalConfig: map[string]interface{}{
				"seller_id":      "seller",
				"marketplace_id": "market",
			},
		},
	}

	_, err := adapter.GetInventoryLevels(context.Background(), channel, []string{"sku-1"})
	require.Error(t, err)
}

func TestAmazonValidateWebhook(t *testing.T) {
	adapter := NewAmazonAdapter()
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{ClientSecret: "secret"},
	}
	body := []byte("payload")

	mac := hmacSHA256Hex("secret", body)
	require.True(t, adapter.ValidateWebhook(context.Background(), channel, mac, body))
	require.False(t, adapter.ValidateWebhook(context.Background(), channel, "", body))
}

func hmacSHA256Hex(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
