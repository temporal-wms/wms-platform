package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wms-platform/services/channel-service/internal/domain"
)

func TestParseFloat(t *testing.T) {
	require.Equal(t, 12.5, parseFloat("12.5"))
	require.Equal(t, 0.0, parseFloat("not-a-number"))
}

func TestMapShopifyOrder(t *testing.T) {
	channel, err := domain.NewChannel("tenant-1", "seller-1", domain.ChannelTypeShopify, "Shop", "", domain.ChannelCredentials{}, domain.SyncSettings{})
	require.NoError(t, err)

	order := &shopifyOrder{
		ID:                 10,
		OrderNumber:        "1001",
		Email:              "buyer@example.com",
		Phone:              "555-1111",
		CreatedAt:          time.Now().UTC(),
		Currency:           "USD",
		SubtotalPrice:      "12.50",
		TotalPrice:         "15.00",
		TotalTax:           "1.00",
		TotalDiscounts:     "0.50",
		TotalShippingPrice: "2.00",
		FinancialStatus:    "paid",
		FulfillmentStatus:  "fulfilled",
		Note:               "note",
		Tags:               []string{"tag-a"},
		Customer: shopifyCustomer{
			ID:        7,
			Email:     "buyer@example.com",
			FirstName: "Ada",
			LastName:  "Lovelace",
			Phone:     "555-2222",
		},
		ShippingAddress: shopifyAddress{
			FirstName: "Ada",
			LastName:  "Lovelace",
			Address1:  "Main",
			City:      "City",
			Province:  "ST",
			Zip:       "12345",
			Country:   "US",
		},
		BillingAddress: &shopifyAddress{
			FirstName: "Ada",
			LastName:  "Lovelace",
			Address1:  "Main",
			City:      "City",
			Province:  "ST",
			Zip:       "12345",
			Country:   "US",
		},
		LineItems: []shopifyLineItem{
			{
				ID:               1,
				ProductID:        2,
				VariantID:        3,
				Title:            "Item",
				SKU:              "sku-1",
				Quantity:         2,
				Price:            "3.50",
				TotalDiscount:    "0.25",
				RequiresShipping: true,
				Grams:            100,
			},
		},
	}

	adapter := NewShopifyAdapter()
	result := adapter.mapShopifyOrder(channel, order)

	require.Equal(t, "10", result.ExternalOrderID)
	require.Equal(t, "1001", result.ExternalOrderNumber)
	require.Equal(t, "7", result.Customer.ExternalID)
	require.Equal(t, 1, len(result.LineItems))
	require.Equal(t, 3.5, result.LineItems[0].Price)
	require.Equal(t, 2, result.LineItems[0].Quantity)
	require.Equal(t, 0.25, result.LineItems[0].TotalDiscount)
	require.Equal(t, 15.0, result.Total)
	require.NotNil(t, result.BillingAddr)
	require.NotZero(t, result.CreatedAt)
	require.NotZero(t, result.UpdatedAt)
}

func TestGetInventoryItemID(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/admin/api/2024-01/variants/456.json", r.URL.Path)
		require.Equal(t, "token", r.Header.Get("X-Shopify-Access-Token"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		payload := map[string]any{
			"variant": map[string]any{
				"inventory_item_id": 123,
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(payload))
	}))
	defer server.Close()

	channel, err := domain.NewChannel("tenant-1", "seller-1", domain.ChannelTypeShopify, "Shop", "", domain.ChannelCredentials{
		StoreDomain: server.Listener.Addr().String(),
		AccessToken: "token",
	}, domain.SyncSettings{})
	require.NoError(t, err)

	adapter := &ShopifyAdapter{httpClient: server.Client()}
	id, err := adapter.getInventoryItemID(context.Background(), channel, "456")
	require.NoError(t, err)
	require.Equal(t, int64(123), id)
}
