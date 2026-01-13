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

func newWooTestAdapter(server *httptest.Server) *WooCommerceAdapter {
	return &WooCommerceAdapter{
		httpClient: server.Client(),
	}
}

func TestWooCommerceValidateCredentials(t *testing.T) {
	adapter := NewWooCommerceAdapter()
	require.Equal(t, domain.ChannelTypeWooCommerce, adapter.GetType())
	err := adapter.ValidateCredentials(context.Background(), domain.ChannelCredentials{})
	require.Error(t, err)

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wp-json/wc/v3/system_status" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter = newWooTestAdapter(server)
	creds := domain.ChannelCredentials{
		ShopURL:   server.URL,
		APIKey:    "key",
		APISecret: "secret",
	}
	err = adapter.ValidateCredentials(context.Background(), creds)
	require.NoError(t, err)
}

func TestWooCommerceValidateCredentialsInsecureURL(t *testing.T) {
	adapter := NewWooCommerceAdapter()
	err := adapter.ValidateCredentials(context.Background(), domain.ChannelCredentials{
		ShopURL:   "http://example.com",
		APIKey:    "key",
		APISecret: "secret",
	})
	require.Error(t, err)
}

func TestWooCommerceValidateCredentialsUnauthorized(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wp-json/wc/v3/system_status" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newWooTestAdapter(server)
	creds := domain.ChannelCredentials{
		ShopURL:   server.URL,
		APIKey:    "key",
		APISecret: "secret",
	}
	err := adapter.ValidateCredentials(context.Background(), creds)
	require.Error(t, err)
}

func TestWooCommerceValidateCredentialsStatusError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wp-json/wc/v3/system_status" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newWooTestAdapter(server)
	creds := domain.ChannelCredentials{
		ShopURL:   server.URL,
		APIKey:    "key",
		APISecret: "secret",
	}
	err := adapter.ValidateCredentials(context.Background(), creds)
	require.Error(t, err)
}

func TestWooCommerceFetchOrders(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wp-json/wc/v3/orders":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"id":           10,
					"number":       "1001",
					"status":       "processing",
					"date_created": time.Now().UTC(),
					"total":        "12.50",
					"currency":     "USD",
					"billing": map[string]any{
						"first_name": "Ada",
						"last_name":  "Lovelace",
						"email":      "buyer@example.com",
						"phone":      "555-1111",
					},
					"shipping": map[string]any{
						"first_name": "Ada",
						"last_name":  "Lovelace",
						"address_1":  "Main",
						"address_2":  "Unit 1",
						"city":       "City",
						"state":      "ST",
						"postcode":   "12345",
						"country":    "US",
						"phone":      "555-2222",
					},
					"line_items": []map[string]any{
						{
							"id":           1,
							"product_id":   2,
							"variation_id": 3,
							"sku":          "sku-1",
							"name":         "Item",
							"quantity":     2,
							"price":        3.5,
							"total":        "7.00",
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newWooTestAdapter(server)
	channel, err := domain.NewChannel("tenant-1", "seller-1", domain.ChannelTypeWooCommerce, "Shop", "", domain.ChannelCredentials{
		ShopURL:   server.URL,
		APIKey:    "key",
		APISecret: "secret",
	}, domain.SyncSettings{})
	require.NoError(t, err)

	orders, err := adapter.FetchOrders(context.Background(), channel, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, "10", orders[0].ExternalOrderID)
	require.Len(t, orders[0].LineItems, 1)
}

func TestWooCommerceFetchOrder(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wp-json/wc/v3/orders/10":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":           10,
				"number":       "1001",
				"status":       "processing",
				"date_created": time.Now().UTC(),
				"total":        "12.50",
				"currency":     "USD",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newWooTestAdapter(server)
	channel := &domain.Channel{
		TenantID:  "tenant-1",
		SellerID:  "seller-1",
		ChannelID: "ch-1",
		Type:      domain.ChannelTypeWooCommerce,
		Credentials: domain.ChannelCredentials{
			ShopURL:   server.URL,
			APIKey:    "key",
			APISecret: "secret",
		},
	}

	order, err := adapter.FetchOrder(context.Background(), channel, "10")
	require.NoError(t, err)
	require.NotNil(t, order)
	require.Equal(t, "10", order.ExternalOrderID)
}

func TestWooCommerceFetchOrderNotFound(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wp-json/wc/v3/orders/10" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newWooTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			ShopURL:   server.URL,
			APIKey:    "key",
			APISecret: "secret",
		},
	}

	order, err := adapter.FetchOrder(context.Background(), channel, "10")
	require.NoError(t, err)
	require.Nil(t, order)
}

func TestWooCommerceFetchOrderError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wp-json/wc/v3/orders/10" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newWooTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			ShopURL:   server.URL,
			APIKey:    "key",
			APISecret: "secret",
		},
	}

	_, err := adapter.FetchOrder(context.Background(), channel, "10")
	require.Error(t, err)
}

func TestWooCommercePushTracking(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wp-json/wc/v3/orders/10":
			w.WriteHeader(http.StatusOK)
		case "/wp-json/wc/v3/orders/10/notes":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newWooTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			ShopURL:   server.URL,
			APIKey:    "key",
			APISecret: "secret",
		},
	}

	err := adapter.PushTracking(context.Background(), channel, "10", domain.TrackingInfo{
		TrackingNumber: "track-1",
		Carrier:        "UPS",
		TrackingURL:    "https://track",
		NotifyCustomer: true,
	})
	require.NoError(t, err)

	err = adapter.CreateFulfillment(context.Background(), channel, domain.FulfillmentRequest{
		OrderID:        "10",
		TrackingNumber: "track-1",
		Carrier:        "UPS",
		TrackingURL:    "https://track",
		NotifyCustomer: true,
	})
	require.NoError(t, err)
}

func TestWooCommercePushTrackingError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wp-json/wc/v3/orders/10":
			w.WriteHeader(http.StatusOK)
		case "/wp-json/wc/v3/orders/10/notes":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newWooTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			ShopURL:   server.URL,
			APIKey:    "key",
			APISecret: "secret",
		},
	}

	err := adapter.PushTracking(context.Background(), channel, "10", domain.TrackingInfo{
		TrackingNumber: "track-1",
		Carrier:        "UPS",
	})
	require.Error(t, err)
}

func TestWooCommerceSyncInventoryAndGetLevels(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wp-json/wc/v3/products":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{
						"id":             5,
						"sku":            "sku-1",
						"stock_quantity": 2,
						"stock_status":   "instock",
					},
				})
				return
			}
			http.NotFound(w, r)
		case "/wp-json/wc/v3/products/5":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newWooTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			ShopURL:   server.URL,
			APIKey:    "key",
			APISecret: "secret",
		},
	}

	err := adapter.SyncInventory(context.Background(), channel, []domain.InventoryUpdate{
		{SKU: "sku-1", Available: 2},
	})
	require.NoError(t, err)

	levels, err := adapter.GetInventoryLevels(context.Background(), channel, []string{"sku-1"})
	require.NoError(t, err)
	require.Len(t, levels, 1)
	require.Equal(t, "sku-1", levels[0].SKU)
}

func TestWooCommerceSyncInventoryNoProducts(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wp-json/wc/v3/products" {
			_ = json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newWooTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			ShopURL:   server.URL,
			APIKey:    "key",
			APISecret: "secret",
		},
	}

	err := adapter.SyncInventory(context.Background(), channel, []domain.InventoryUpdate{
		{SKU: "sku-1", Available: 2},
	})
	require.NoError(t, err)
}

func TestWooCommerceRegisterWebhooks(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wp-json/wc/v3/webhooks" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newWooTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			ShopURL:   server.URL,
			APIKey:    "key",
			APISecret: "secret",
		},
	}

	err := adapter.RegisterWebhooks(context.Background(), channel, "https://example.com/webhook")
	require.NoError(t, err)
}

func TestWooCommerceValidateWebhook(t *testing.T) {
	adapter := NewWooCommerceAdapter()
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			APISecret: "secret",
		},
	}
	body := []byte("payload")
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write(body)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	require.True(t, adapter.ValidateWebhook(context.Background(), channel, signature, body))
	require.False(t, adapter.ValidateWebhook(context.Background(), channel, "", body))
}
