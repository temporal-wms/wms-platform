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

func newShopifyTestAdapter(server *httptest.Server) *ShopifyAdapter {
	return &ShopifyAdapter{httpClient: server.Client()}
}

func TestShopifyGetType(t *testing.T) {
	adapter := NewShopifyAdapter()
	require.Equal(t, domain.ChannelTypeShopify, adapter.GetType())
}

func TestShopifyValidateCredentials(t *testing.T) {
	adapter := NewShopifyAdapter()
	err := adapter.ValidateCredentials(context.Background(), domain.ChannelCredentials{})
	require.Error(t, err)

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/api/2024-01/shop.json" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter = newShopifyTestAdapter(server)
	creds := domain.ChannelCredentials{
		StoreDomain: server.Listener.Addr().String(),
		AccessToken: "token",
	}
	err = adapter.ValidateCredentials(context.Background(), creds)
	require.NoError(t, err)
}

func TestShopifyValidateCredentialsStatusError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/api/2024-01/shop.json" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	creds := domain.ChannelCredentials{
		StoreDomain: server.Listener.Addr().String(),
		AccessToken: "token",
	}
	err := adapter.ValidateCredentials(context.Background(), creds)
	require.Error(t, err)
}

func TestShopifyFetchOrdersAndOrder(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin/api/2024-01/orders.json":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"orders": []map[string]any{
					{
						"id":            10,
						"order_number":  "1001",
						"email":         "buyer@example.com",
						"phone":         "555-1111",
						"created_at":    time.Now().UTC(),
						"currency":      "USD",
						"subtotal_price": "12.50",
						"total_price":   "15.00",
						"total_tax":     "1.00",
						"total_discounts": "0.50",
						"total_shipping_price_set": "2.00",
						"financial_status": "paid",
						"fulfillment_status": "fulfilled",
						"note":           "note",
						"tags":           []string{"tag"},
						"customer": map[string]any{
							"id":         7,
							"email":      "buyer@example.com",
							"first_name": "Ada",
							"last_name":  "Lovelace",
							"phone":      "555-2222",
						},
						"shipping_address": map[string]any{
							"first_name": "Ada",
							"last_name":  "Lovelace",
							"address1":   "Main",
							"address2":   "Unit 1",
							"city":       "City",
							"province":   "ST",
							"zip":        "12345",
							"country":    "US",
							"phone":      "555-1111",
						},
						"line_items": []map[string]any{
							{
								"id":              1,
								"product_id":      2,
								"variant_id":      3,
								"title":           "Item",
								"sku":             "sku-1",
								"quantity":        2,
								"price":           "3.50",
								"total_discount":  "0.25",
								"requires_shipping": true,
								"grams":           100,
							},
						},
					},
				},
			})
		case "/admin/api/2024-01/orders/10.json":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"order": map[string]any{
					"id":           10,
					"order_number": "1001",
					"email":        "buyer@example.com",
					"phone":        "555-1111",
					"created_at":   time.Now().UTC(),
					"currency":     "USD",
					"subtotal_price": "12.50",
					"total_price":  "15.00",
					"total_tax":    "1.00",
					"total_discounts": "0.50",
					"total_shipping_price_set": "2.00",
					"financial_status": "paid",
					"fulfillment_status": "fulfilled",
					"note": "note",
					"tags": []string{"tag"},
					"customer": map[string]any{
						"id":         7,
						"email":      "buyer@example.com",
						"first_name": "Ada",
						"last_name":  "Lovelace",
						"phone":      "555-2222",
					},
					"shipping_address": map[string]any{
						"first_name": "Ada",
						"last_name":  "Lovelace",
						"address1":   "Main",
						"address2":   "Unit 1",
						"city":       "City",
						"province":   "ST",
						"zip":        "12345",
						"country":    "US",
						"phone":      "555-1111",
					},
					"line_items": []map[string]any{
						{
							"id":              1,
							"product_id":      2,
							"variant_id":      3,
							"title":           "Item",
							"sku":             "sku-1",
							"quantity":        2,
							"price":           "3.50",
							"total_discount":  "0.25",
							"requires_shipping": true,
							"grams":           100,
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel, err := domain.NewChannel("tenant-1", "seller-1", domain.ChannelTypeShopify, "Shop", "", domain.ChannelCredentials{
		StoreDomain: server.Listener.Addr().String(),
		AccessToken: "token",
	}, domain.SyncSettings{})
	require.NoError(t, err)

	orders, err := adapter.FetchOrders(context.Background(), channel, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, orders, 1)

	order, err := adapter.FetchOrder(context.Background(), channel, "10")
	require.NoError(t, err)
	require.NotNil(t, order)
}

func TestShopifyFetchOrdersError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/api/2024-01/orders.json" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		Status: domain.ChannelStatusActive,
		Credentials: domain.ChannelCredentials{
			StoreDomain: server.Listener.Addr().String(),
			AccessToken: "token",
		},
	}
	_, err := adapter.FetchOrders(context.Background(), channel, time.Now().Add(-time.Hour))
	require.Error(t, err)
}

func TestShopifyFetchOrdersDecodeError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/api/2024-01/orders.json" {
			_, _ = w.Write([]byte("not-json"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		Status: domain.ChannelStatusActive,
		Credentials: domain.ChannelCredentials{
			StoreDomain: server.Listener.Addr().String(),
			AccessToken: "token",
		},
	}
	_, err := adapter.FetchOrders(context.Background(), channel, time.Now().Add(-time.Hour))
	require.Error(t, err)
}

func TestShopifyFetchOrderError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/api/2024-01/orders/10.json" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			StoreDomain: server.Listener.Addr().String(),
			AccessToken: "token",
		},
	}
	_, err := adapter.FetchOrder(context.Background(), channel, "10")
	require.Error(t, err)
}

func TestShopifyFetchOrderDecodeError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/api/2024-01/orders/10.json" {
			_, _ = w.Write([]byte("not-json"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			StoreDomain: server.Listener.Addr().String(),
			AccessToken: "token",
		},
	}
	_, err := adapter.FetchOrder(context.Background(), channel, "10")
	require.Error(t, err)
}

func TestShopifyFetchOrdersInactiveChannel(t *testing.T) {
	adapter := NewShopifyAdapter()
	channel := &domain.Channel{
		Status: domain.ChannelStatusPaused,
	}
	_, err := adapter.FetchOrders(context.Background(), channel, time.Now())
	require.ErrorIs(t, err, domain.ErrChannelNotActive)
}

func TestShopifyPushTrackingAndFulfillment(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin/api/2024-01/orders/10/fulfillment_orders.json":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"fulfillment_orders": []map[string]any{
					{
						"id":     99,
						"status": "open",
						"line_items": []map[string]any{
							{"id": 1, "quantity": 1},
						},
					},
				},
			})
		case "/admin/api/2024-01/fulfillments.json":
			w.WriteHeader(http.StatusCreated)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			StoreDomain: server.Listener.Addr().String(),
			AccessToken: "token",
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
	})
	require.NoError(t, err)
}

func TestShopifyPushTrackingCreateError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin/api/2024-01/orders/10/fulfillment_orders.json":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"fulfillment_orders": []map[string]any{
					{"id": 99, "status": "open", "line_items": []map[string]any{{"id": 1, "quantity": 1}}},
				},
			})
		case "/admin/api/2024-01/fulfillments.json":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			StoreDomain: server.Listener.Addr().String(),
			AccessToken: "token",
		},
	}

	err := adapter.PushTracking(context.Background(), channel, "10", domain.TrackingInfo{
		TrackingNumber: "track-1",
		Carrier:        "UPS",
	})
	require.Error(t, err)
}

func TestShopifyPushTrackingNoOpenOrder(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/api/2024-01/orders/10/fulfillment_orders.json" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"fulfillment_orders": []map[string]any{
					{"id": 99, "status": "closed"},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			StoreDomain: server.Listener.Addr().String(),
			AccessToken: "token",
		},
	}

	err := adapter.PushTracking(context.Background(), channel, "10", domain.TrackingInfo{
		TrackingNumber: "track-1",
		Carrier:        "UPS",
	})
	require.Error(t, err)
}

func TestShopifySyncInventory(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin/api/2024-01/variants/123.json":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"variant": map[string]any{
					"inventory_item_id": 999,
				},
			})
		case "/admin/api/2024-01/inventory_levels/set.json":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			StoreDomain: server.Listener.Addr().String(),
			AccessToken: "token",
		},
	}

	err := adapter.SyncInventory(context.Background(), channel, []domain.InventoryUpdate{
		{SKU: "sku-1", VariantID: "123", LocationID: "1", Available: 5},
	})
	require.NoError(t, err)

	levels, err := adapter.GetInventoryLevels(context.Background(), channel, []string{"sku-1"})
	require.NoError(t, err)
	require.Empty(t, levels)
}

func TestShopifySyncInventoryMissingFields(t *testing.T) {
	adapter := NewShopifyAdapter()
	channel := &domain.Channel{}

	err := adapter.SyncInventory(context.Background(), channel, []domain.InventoryUpdate{
		{SKU: "sku-1"},
	})
	require.NoError(t, err)
}

func TestShopifySyncInventorySetError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin/api/2024-01/variants/123.json":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"variant": map[string]any{
					"inventory_item_id": 999,
				},
			})
		case "/admin/api/2024-01/inventory_levels/set.json":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			StoreDomain: server.Listener.Addr().String(),
			AccessToken: "token",
		},
	}

	err := adapter.SyncInventory(context.Background(), channel, []domain.InventoryUpdate{
		{SKU: "sku-1", VariantID: "123", LocationID: "1", Available: 5},
	})
	require.Error(t, err)
}

func TestShopifyGetInventoryItemIDDecodeError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/api/2024-01/variants/123.json" {
			_, _ = w.Write([]byte("not-json"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{
			StoreDomain: server.Listener.Addr().String(),
			AccessToken: "token",
		},
	}

	_, err := adapter.getInventoryItemID(context.Background(), channel, "123")
	require.Error(t, err)
}

func TestShopifyRegisterWebhooksAndValidate(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/api/2024-01/webhooks.json" {
			w.WriteHeader(http.StatusCreated)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		ChannelID: "ch-1",
		Credentials: domain.ChannelCredentials{
			StoreDomain:   server.Listener.Addr().String(),
			AccessToken:   "token",
			WebhookSecret: "secret",
		},
	}

	err := adapter.RegisterWebhooks(context.Background(), channel, "https://example.com/webhooks")
	require.NoError(t, err)

	body := []byte("payload")
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write(body)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	require.True(t, adapter.ValidateWebhook(context.Background(), channel, signature, body))
}

func TestShopifyValidateWebhookMissingSecret(t *testing.T) {
	adapter := NewShopifyAdapter()
	channel := &domain.Channel{
		Credentials: domain.ChannelCredentials{},
	}
	require.False(t, adapter.ValidateWebhook(context.Background(), channel, "sig", []byte("payload")))
}

func TestShopifyRegisterWebhooksStatusError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/api/2024-01/webhooks.json" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newShopifyTestAdapter(server)
	channel := &domain.Channel{
		ChannelID: "ch-1",
		Credentials: domain.ChannelCredentials{
			StoreDomain: server.Listener.Addr().String(),
			AccessToken: "token",
		},
	}

	err := adapter.RegisterWebhooks(context.Background(), channel, "https://example.com/webhooks")
	require.Error(t, err)
}
