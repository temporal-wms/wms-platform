package adapters

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wms-platform/services/channel-service/internal/domain"
)

// WooCommerceAdapter implements ChannelAdapter for WooCommerce REST API
type WooCommerceAdapter struct {
	httpClient *http.Client
}

// NewWooCommerceAdapter creates a new WooCommerce adapter
func NewWooCommerceAdapter() *WooCommerceAdapter {
	return &WooCommerceAdapter{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *WooCommerceAdapter) GetType() domain.ChannelType {
	return domain.ChannelTypeWooCommerce
}

func (a *WooCommerceAdapter) ValidateCredentials(ctx context.Context, creds domain.ChannelCredentials) error {
	if creds.ShopURL == "" {
		return fmt.Errorf("shop_url is required")
	}
	if creds.APIKey == "" {
		return fmt.Errorf("api_key (consumer key) is required")
	}
	if creds.APISecret == "" {
		return fmt.Errorf("api_secret (consumer secret) is required")
	}

	// Validate URL format
	if !strings.HasPrefix(creds.ShopURL, "https://") {
		return fmt.Errorf("shop_url must use HTTPS")
	}

	// Try to access the API
	endpoint := a.buildURL(creds.ShopURL, "/wp-json/wc/v3/system_status")
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return err
	}
	a.setAuth(req, creds)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to WooCommerce: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API credentials")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to validate credentials: status %d", resp.StatusCode)
	}

	return nil
}

func (a *WooCommerceAdapter) buildURL(shopURL, path string) string {
	return strings.TrimSuffix(shopURL, "/") + path
}

func (a *WooCommerceAdapter) setAuth(req *http.Request, creds domain.ChannelCredentials) {
	// WooCommerce uses Basic Auth with consumer key/secret
	auth := base64.StdEncoding.EncodeToString([]byte(creds.APIKey + ":" + creds.APISecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
}

func (a *WooCommerceAdapter) FetchOrders(ctx context.Context, channel *domain.Channel, since time.Time) ([]*domain.ChannelOrder, error) {
	endpoint := a.buildURL(channel.Credentials.ShopURL, "/wp-json/wc/v3/orders")
	endpoint += fmt.Sprintf("?after=%s&status=processing,on-hold&per_page=50", since.Format(time.RFC3339))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	a.setAuth(req, channel.Credentials)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch orders: %s", string(body))
	}

	var wooOrders []struct {
		ID              int       `json:"id"`
		Number          string    `json:"number"`
		Status          string    `json:"status"`
		DateCreated     time.Time `json:"date_created"`
		Total           string    `json:"total"`
		Currency        string    `json:"currency"`
		Billing         struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Email     string `json:"email"`
			Phone     string `json:"phone"`
		} `json:"billing"`
		Shipping struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Address1  string `json:"address_1"`
			Address2  string `json:"address_2"`
			City      string `json:"city"`
			State     string `json:"state"`
			Postcode  string `json:"postcode"`
			Country   string `json:"country"`
			Phone     string `json:"phone"`
		} `json:"shipping"`
		LineItems []struct {
			ID          int     `json:"id"`
			ProductID   int     `json:"product_id"`
			VariationID int     `json:"variation_id"`
			SKU         string  `json:"sku"`
			Name        string  `json:"name"`
			Quantity    int     `json:"quantity"`
			Price       float64 `json:"price"`
			Total       string  `json:"total"`
		} `json:"line_items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&wooOrders); err != nil {
		return nil, err
	}

	var orders []*domain.ChannelOrder
	now := time.Now()
	for _, order := range wooOrders {
		var total float64
		fmt.Sscanf(order.Total, "%f", &total)

		channelOrder := &domain.ChannelOrder{
			TenantID:            channel.TenantID,
			SellerID:            channel.SellerID,
			ChannelID:           channel.ChannelID,
			ExternalOrderID:     fmt.Sprintf("%d", order.ID),
			ExternalOrderNumber: order.Number,
			FulfillmentStatus:   order.Status,
			ExternalCreatedAt:   order.DateCreated,
			Total:               total,
			Currency:            order.Currency,
			CreatedAt:           now,
			UpdatedAt:           now,
			Customer: domain.ChannelCustomer{
				ExternalID: order.Billing.Email,
				Email:      order.Billing.Email,
				FirstName:  order.Billing.FirstName,
				LastName:   order.Billing.LastName,
				Phone:      order.Billing.Phone,
			},
			ShippingAddr: domain.ChannelAddress{
				FirstName: order.Shipping.FirstName,
				LastName:  order.Shipping.LastName,
				Address1:  order.Shipping.Address1,
				Address2:  order.Shipping.Address2,
				City:      order.Shipping.City,
				Province:  order.Shipping.State,
				Zip:       order.Shipping.Postcode,
				Country:   order.Shipping.Country,
				Phone:     order.Shipping.Phone,
			},
		}

		for _, item := range order.LineItems {
			variantID := ""
			if item.VariationID > 0 {
				variantID = fmt.Sprintf("%d", item.VariationID)
			}

			channelOrder.LineItems = append(channelOrder.LineItems, domain.ChannelLineItem{
				ExternalID: fmt.Sprintf("%d", item.ID),
				SKU:        item.SKU,
				Title:      item.Name,
				Quantity:   item.Quantity,
				Price:      item.Price,
				ProductID:  fmt.Sprintf("%d", item.ProductID),
				VariantID:  variantID,
			})
		}

		orders = append(orders, channelOrder)
	}

	return orders, nil
}

func (a *WooCommerceAdapter) FetchOrder(ctx context.Context, channel *domain.Channel, externalOrderID string) (*domain.ChannelOrder, error) {
	endpoint := a.buildURL(channel.Credentials.ShopURL, "/wp-json/wc/v3/orders/"+externalOrderID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	a.setAuth(req, channel.Credentials)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch order: %s", string(body))
	}

	var wooOrder struct {
		ID              int       `json:"id"`
		Number          string    `json:"number"`
		Status          string    `json:"status"`
		DateCreated     time.Time `json:"date_created"`
		Total           string    `json:"total"`
		Currency        string    `json:"currency"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&wooOrder); err != nil {
		return nil, err
	}

	var total float64
	fmt.Sscanf(wooOrder.Total, "%f", &total)

	now := time.Now()
	return &domain.ChannelOrder{
		TenantID:            channel.TenantID,
		SellerID:            channel.SellerID,
		ChannelID:           channel.ChannelID,
		ExternalOrderID:     fmt.Sprintf("%d", wooOrder.ID),
		ExternalOrderNumber: wooOrder.Number,
		FulfillmentStatus:   wooOrder.Status,
		ExternalCreatedAt:   wooOrder.DateCreated,
		Total:               total,
		Currency:            wooOrder.Currency,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

func (a *WooCommerceAdapter) PushTracking(ctx context.Context, channel *domain.Channel, externalOrderID string, tracking domain.TrackingInfo) error {
	// Update order with tracking info using order notes or meta data
	// WooCommerce doesn't have built-in tracking, usually uses plugins
	// We'll update order status and add a note

	// First, update order status to completed
	endpoint := a.buildURL(channel.Credentials.ShopURL, "/wp-json/wc/v3/orders/"+externalOrderID)

	updateReq := map[string]interface{}{
		"status": "completed",
	}

	body, _ := json.Marshal(updateReq)
	req, err := http.NewRequestWithContext(ctx, "PUT", endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	a.setAuth(req, channel.Credentials)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	// Add order note with tracking info
	noteEndpoint := a.buildURL(channel.Credentials.ShopURL, "/wp-json/wc/v3/orders/"+externalOrderID+"/notes")

	noteContent := fmt.Sprintf("Order shipped via %s. Tracking number: %s", tracking.Carrier, tracking.TrackingNumber)
	if tracking.TrackingURL != "" {
		noteContent += fmt.Sprintf("\nTrack your order: %s", tracking.TrackingURL)
	}

	noteReq := map[string]interface{}{
		"note":               noteContent,
		"customer_note":      tracking.NotifyCustomer,
	}

	noteBody, _ := json.Marshal(noteReq)
	noteHttpReq, err := http.NewRequestWithContext(ctx, "POST", noteEndpoint, bytes.NewReader(noteBody))
	if err != nil {
		return err
	}
	a.setAuth(noteHttpReq, channel.Credentials)

	noteResp, err := a.httpClient.Do(noteHttpReq)
	if err != nil {
		return err
	}
	defer noteResp.Body.Close()

	if noteResp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(noteResp.Body)
		return fmt.Errorf("failed to add tracking note: %s", string(respBody))
	}

	return nil
}

func (a *WooCommerceAdapter) SyncInventory(ctx context.Context, channel *domain.Channel, items []domain.InventoryUpdate) error {
	for _, item := range items {
		// Find product by SKU
		searchEndpoint := a.buildURL(channel.Credentials.ShopURL, "/wp-json/wc/v3/products")
		searchEndpoint += fmt.Sprintf("?sku=%s", item.SKU)

		searchReq, err := http.NewRequestWithContext(ctx, "GET", searchEndpoint, nil)
		if err != nil {
			continue
		}
		a.setAuth(searchReq, channel.Credentials)

		searchResp, err := a.httpClient.Do(searchReq)
		if err != nil {
			continue
		}

		var products []struct {
			ID int `json:"id"`
		}
		json.NewDecoder(searchResp.Body).Decode(&products)
		searchResp.Body.Close()

		if len(products) == 0 {
			continue
		}

		// Update product stock
		productID := products[0].ID
		updateEndpoint := a.buildURL(channel.Credentials.ShopURL, fmt.Sprintf("/wp-json/wc/v3/products/%d", productID))

		updateReq := map[string]interface{}{
			"stock_quantity":  item.Available,
			"manage_stock":    true,
			"stock_status":    "instock",
		}
		if item.Available <= 0 {
			updateReq["stock_status"] = "outofstock"
		}

		body, _ := json.Marshal(updateReq)
		req, err := http.NewRequestWithContext(ctx, "PUT", updateEndpoint, bytes.NewReader(body))
		if err != nil {
			continue
		}
		a.setAuth(req, channel.Credentials)

		resp, err := a.httpClient.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
	}

	return nil
}

func (a *WooCommerceAdapter) GetInventoryLevels(ctx context.Context, channel *domain.Channel, skus []string) ([]domain.InventoryLevel, error) {
	var levels []domain.InventoryLevel

	for _, sku := range skus {
		endpoint := a.buildURL(channel.Credentials.ShopURL, "/wp-json/wc/v3/products")
		endpoint += fmt.Sprintf("?sku=%s", sku)

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			continue
		}
		a.setAuth(req, channel.Credentials)

		resp, err := a.httpClient.Do(req)
		if err != nil {
			continue
		}

		var products []struct {
			ID            int    `json:"id"`
			SKU           string `json:"sku"`
			StockQuantity int    `json:"stock_quantity"`
			StockStatus   string `json:"stock_status"`
		}
		json.NewDecoder(resp.Body).Decode(&products)
		resp.Body.Close()

		for _, product := range products {
			levels = append(levels, domain.InventoryLevel{
				SKU:       product.SKU,
				ProductID: fmt.Sprintf("%d", product.ID),
				Available: product.StockQuantity,
				OnHand:    product.StockQuantity,
			})
		}
	}

	return levels, nil
}

func (a *WooCommerceAdapter) CreateFulfillment(ctx context.Context, channel *domain.Channel, fulfillment domain.FulfillmentRequest) error {
	return a.PushTracking(ctx, channel, fulfillment.OrderID, domain.TrackingInfo{
		TrackingNumber: fulfillment.TrackingNumber,
		Carrier:        fulfillment.Carrier,
		TrackingURL:    fulfillment.TrackingURL,
		NotifyCustomer: fulfillment.NotifyCustomer,
	})
}

func (a *WooCommerceAdapter) RegisterWebhooks(ctx context.Context, channel *domain.Channel, webhookURL string) error {
	topics := []struct {
		Name  string
		Topic string
	}{
		{"Order Created", "order.created"},
		{"Order Updated", "order.updated"},
		{"Product Updated", "product.updated"},
	}

	for _, t := range topics {
		endpoint := a.buildURL(channel.Credentials.ShopURL, "/wp-json/wc/v3/webhooks")

		webhookReq := map[string]interface{}{
			"name":        t.Name,
			"topic":       t.Topic,
			"delivery_url": webhookURL,
			"status":      "active",
		}

		body, _ := json.Marshal(webhookReq)
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		if err != nil {
			continue
		}
		a.setAuth(req, channel.Credentials)

		resp, err := a.httpClient.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
	}

	return nil
}

func (a *WooCommerceAdapter) ValidateWebhook(ctx context.Context, channel *domain.Channel, signature string, body []byte) bool {
	if signature == "" {
		return false
	}

	// WooCommerce uses X-WC-Webhook-Signature header
	// HMAC-SHA256 of body with webhook secret
	secret := channel.Credentials.WebhookSecret
	if secret == "" {
		secret = channel.Credentials.APISecret
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}
