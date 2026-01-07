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
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/wms-platform/services/channel-service/internal/domain"
)

var shopifyTracer = otel.Tracer("channel-service/adapters/shopify")

// ShopifyAdapter implements the ChannelAdapter interface for Shopify
type ShopifyAdapter struct {
	httpClient *http.Client
}

// NewShopifyAdapter creates a new Shopify adapter
func NewShopifyAdapter() *ShopifyAdapter {
	return &ShopifyAdapter{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetType returns the channel type
func (a *ShopifyAdapter) GetType() domain.ChannelType {
	return domain.ChannelTypeShopify
}

// ValidateCredentials validates Shopify credentials
func (a *ShopifyAdapter) ValidateCredentials(ctx context.Context, creds domain.ChannelCredentials) error {
	if creds.StoreDomain == "" {
		return fmt.Errorf("store domain is required")
	}
	if creds.AccessToken == "" {
		return fmt.Errorf("access token is required")
	}

	// Test the credentials by fetching shop info
	url := fmt.Sprintf("https://%s/admin/api/2024-01/shop.json", creds.StoreDomain)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Shopify-Access-Token", creds.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Shopify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid credentials: status %d", resp.StatusCode)
	}

	return nil
}

// FetchOrders fetches orders from Shopify
func (a *ShopifyAdapter) FetchOrders(ctx context.Context, channel *domain.Channel, since time.Time) ([]*domain.ChannelOrder, error) {
	ctx, span := shopifyTracer.Start(ctx, "shopify.FetchOrders",
		trace.WithAttributes(
			attribute.String("channel.id", channel.ChannelID),
			attribute.String("channel.type", "shopify"),
			attribute.String("since", since.Format(time.RFC3339)),
		),
	)
	defer span.End()

	if !channel.IsActive() {
		span.SetAttributes(attribute.String("error", "channel_not_active"))
		return nil, domain.ErrChannelNotActive
	}

	url := fmt.Sprintf(
		"https://%s/admin/api/2024-01/orders.json?status=any&created_at_min=%s&limit=250",
		channel.Credentials.StoreDomain,
		since.Format(time.RFC3339),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	req.Header.Set("X-Shopify-Access-Token", channel.Credentials.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to fetch orders: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("failed to fetch orders: status %d, body: %s", resp.StatusCode, string(body))
		span.RecordError(err)
		return nil, err
	}

	var response shopifyOrdersResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	orders := make([]*domain.ChannelOrder, 0, len(response.Orders))
	for _, so := range response.Orders {
		order := a.mapShopifyOrder(channel, &so)
		orders = append(orders, order)
	}

	span.SetAttributes(attribute.Int("orders.fetched", len(orders)))
	return orders, nil
}

// FetchOrder fetches a single order from Shopify
func (a *ShopifyAdapter) FetchOrder(ctx context.Context, channel *domain.Channel, externalOrderID string) (*domain.ChannelOrder, error) {
	url := fmt.Sprintf(
		"https://%s/admin/api/2024-01/orders/%s.json",
		channel.Credentials.StoreDomain,
		externalOrderID,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Shopify-Access-Token", channel.Credentials.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch order: status %d", resp.StatusCode)
	}

	var response struct {
		Order shopifyOrder `json:"order"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return a.mapShopifyOrder(channel, &response.Order), nil
}

// PushTracking pushes tracking info to Shopify
func (a *ShopifyAdapter) PushTracking(ctx context.Context, channel *domain.Channel, externalOrderID string, tracking domain.TrackingInfo) error {
	// First, get the order to find the fulfillment order ID
	url := fmt.Sprintf(
		"https://%s/admin/api/2024-01/orders/%s/fulfillment_orders.json",
		channel.Credentials.StoreDomain,
		externalOrderID,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Shopify-Access-Token", channel.Credentials.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get fulfillment orders: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get fulfillment orders: status %d", resp.StatusCode)
	}

	var foResponse struct {
		FulfillmentOrders []struct {
			ID        int64  `json:"id"`
			Status    string `json:"status"`
			LineItems []struct {
				ID       int64 `json:"id"`
				Quantity int   `json:"quantity"`
			} `json:"line_items"`
		} `json:"fulfillment_orders"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&foResponse); err != nil {
		return fmt.Errorf("failed to decode fulfillment orders: %w", err)
	}

	// Find the first open fulfillment order
	var fulfillmentOrderID int64
	var lineItems []map[string]interface{}
	for _, fo := range foResponse.FulfillmentOrders {
		if fo.Status == "open" || fo.Status == "in_progress" {
			fulfillmentOrderID = fo.ID
			for _, li := range fo.LineItems {
				lineItems = append(lineItems, map[string]interface{}{
					"id":       li.ID,
					"quantity": li.Quantity,
				})
			}
			break
		}
	}

	if fulfillmentOrderID == 0 {
		return fmt.Errorf("no open fulfillment order found")
	}

	// Create fulfillment
	fulfillmentURL := fmt.Sprintf(
		"https://%s/admin/api/2024-01/fulfillments.json",
		channel.Credentials.StoreDomain,
	)

	fulfillmentData := map[string]interface{}{
		"fulfillment": map[string]interface{}{
			"line_items_by_fulfillment_order": []map[string]interface{}{
				{
					"fulfillment_order_id": fulfillmentOrderID,
					"fulfillment_order_line_items": lineItems,
				},
			},
			"tracking_info": map[string]interface{}{
				"number":  tracking.TrackingNumber,
				"company": tracking.Carrier,
				"url":     tracking.TrackingURL,
			},
			"notify_customer": tracking.NotifyCustomer,
		},
	}

	body, err := json.Marshal(fulfillmentData)
	if err != nil {
		return err
	}

	req, err = http.NewRequestWithContext(ctx, "POST", fulfillmentURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("X-Shopify-Access-Token", channel.Credentials.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err = a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create fulfillment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create fulfillment: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SyncInventory syncs inventory levels to Shopify
func (a *ShopifyAdapter) SyncInventory(ctx context.Context, channel *domain.Channel, items []domain.InventoryUpdate) error {
	for _, item := range items {
		if item.VariantID == "" || item.LocationID == "" {
			continue
		}

		// First get the inventory item ID
		inventoryItemID, err := a.getInventoryItemID(ctx, channel, item.VariantID)
		if err != nil {
			return fmt.Errorf("failed to get inventory item ID for variant %s: %w", item.VariantID, err)
		}

		// Set inventory level
		url := fmt.Sprintf(
			"https://%s/admin/api/2024-01/inventory_levels/set.json",
			channel.Credentials.StoreDomain,
		)

		data := map[string]interface{}{
			"location_id":       item.LocationID,
			"inventory_item_id": inventoryItemID,
			"available":         item.Available,
		}

		body, err := json.Marshal(data)
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return err
		}

		req.Header.Set("X-Shopify-Access-Token", channel.Credentials.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to set inventory: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to set inventory for %s: status %d", item.SKU, resp.StatusCode)
		}
	}

	return nil
}

// GetInventoryLevels gets inventory levels from Shopify
func (a *ShopifyAdapter) GetInventoryLevels(ctx context.Context, channel *domain.Channel, skus []string) ([]domain.InventoryLevel, error) {
	// This would require mapping SKUs to variant IDs first
	// For now, return empty - full implementation would query products by SKU
	return []domain.InventoryLevel{}, nil
}

// CreateFulfillment creates a fulfillment in Shopify
func (a *ShopifyAdapter) CreateFulfillment(ctx context.Context, channel *domain.Channel, fulfillment domain.FulfillmentRequest) error {
	return a.PushTracking(ctx, channel, fulfillment.OrderID, domain.TrackingInfo{
		TrackingNumber: fulfillment.TrackingNumber,
		Carrier:        fulfillment.Carrier,
		TrackingURL:    fulfillment.TrackingURL,
		NotifyCustomer: fulfillment.NotifyCustomer,
	})
}

// RegisterWebhooks registers webhooks with Shopify
func (a *ShopifyAdapter) RegisterWebhooks(ctx context.Context, channel *domain.Channel, webhookURL string) error {
	topics := []string{
		"orders/create",
		"orders/updated",
		"orders/cancelled",
		"inventory_levels/update",
	}

	for _, topic := range topics {
		url := fmt.Sprintf(
			"https://%s/admin/api/2024-01/webhooks.json",
			channel.Credentials.StoreDomain,
		)

		data := map[string]interface{}{
			"webhook": map[string]interface{}{
				"topic":   topic,
				"address": fmt.Sprintf("%s/%s/%s", webhookURL, channel.ChannelID, topic),
				"format":  "json",
			},
		}

		body, err := json.Marshal(data)
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return err
		}

		req.Header.Set("X-Shopify-Access-Token", channel.Credentials.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to register webhook %s: %w", topic, err)
		}
		resp.Body.Close()

		// 201 Created or 422 (already exists) are both acceptable
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusUnprocessableEntity {
			return fmt.Errorf("failed to register webhook %s: status %d", topic, resp.StatusCode)
		}
	}

	return nil
}

// ValidateWebhook validates an incoming Shopify webhook
func (a *ShopifyAdapter) ValidateWebhook(ctx context.Context, channel *domain.Channel, signature string, body []byte) bool {
	if channel.Credentials.WebhookSecret == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(channel.Credentials.WebhookSecret))
	mac.Write(body)
	expectedMAC := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}

// Helper methods

func (a *ShopifyAdapter) getInventoryItemID(ctx context.Context, channel *domain.Channel, variantID string) (int64, error) {
	url := fmt.Sprintf(
		"https://%s/admin/api/2024-01/variants/%s.json",
		channel.Credentials.StoreDomain,
		variantID,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("X-Shopify-Access-Token", channel.Credentials.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var response struct {
		Variant struct {
			InventoryItemID int64 `json:"inventory_item_id"`
		} `json:"variant"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	return response.Variant.InventoryItemID, nil
}

func (a *ShopifyAdapter) mapShopifyOrder(channel *domain.Channel, so *shopifyOrder) *domain.ChannelOrder {
	lineItems := make([]domain.ChannelLineItem, len(so.LineItems))
	for i, li := range so.LineItems {
		lineItems[i] = domain.ChannelLineItem{
			ExternalID:       fmt.Sprintf("%d", li.ID),
			SKU:              li.SKU,
			ProductID:        fmt.Sprintf("%d", li.ProductID),
			VariantID:        fmt.Sprintf("%d", li.VariantID),
			Title:            li.Title,
			Quantity:         li.Quantity,
			Price:            parseFloat(li.Price),
			TotalDiscount:    parseFloat(li.TotalDiscount),
			RequiresShipping: li.RequiresShipping,
			Grams:            li.Grams,
		}
	}

	var billingAddr *domain.ChannelAddress
	if so.BillingAddress != nil {
		billingAddr = &domain.ChannelAddress{
			FirstName: so.BillingAddress.FirstName,
			LastName:  so.BillingAddress.LastName,
			Company:   so.BillingAddress.Company,
			Address1:  so.BillingAddress.Address1,
			Address2:  so.BillingAddress.Address2,
			City:      so.BillingAddress.City,
			Province:  so.BillingAddress.Province,
			Zip:       so.BillingAddress.Zip,
			Country:   so.BillingAddress.Country,
			Phone:     so.BillingAddress.Phone,
		}
	}

	now := time.Now().UTC()
	return &domain.ChannelOrder{
		TenantID:            channel.TenantID,
		SellerID:            channel.SellerID,
		ChannelID:           channel.ChannelID,
		ExternalOrderID:     fmt.Sprintf("%d", so.ID),
		ExternalOrderNumber: so.OrderNumber,
		ExternalCreatedAt:   so.CreatedAt,
		Customer: domain.ChannelCustomer{
			ExternalID: fmt.Sprintf("%d", so.Customer.ID),
			Email:      so.Email,
			FirstName:  so.Customer.FirstName,
			LastName:   so.Customer.LastName,
			Phone:      so.Phone,
		},
		ShippingAddr: domain.ChannelAddress{
			FirstName: so.ShippingAddress.FirstName,
			LastName:  so.ShippingAddress.LastName,
			Company:   so.ShippingAddress.Company,
			Address1:  so.ShippingAddress.Address1,
			Address2:  so.ShippingAddress.Address2,
			City:      so.ShippingAddress.City,
			Province:  so.ShippingAddress.Province,
			Zip:       so.ShippingAddress.Zip,
			Country:   so.ShippingAddress.Country,
			Phone:     so.ShippingAddress.Phone,
		},
		BillingAddr:       billingAddr,
		LineItems:         lineItems,
		Currency:          so.Currency,
		Subtotal:          parseFloat(so.SubtotalPrice),
		ShippingCost:      parseFloat(so.TotalShippingPrice),
		Tax:               parseFloat(so.TotalTax),
		Discount:          parseFloat(so.TotalDiscounts),
		Total:             parseFloat(so.TotalPrice),
		FinancialStatus:   so.FinancialStatus,
		FulfillmentStatus: so.FulfillmentStatus,
		Tags:              so.Tags,
		Notes:             so.Note,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// Shopify API response types

type shopifyOrdersResponse struct {
	Orders []shopifyOrder `json:"orders"`
}

type shopifyOrder struct {
	ID                 int64            `json:"id"`
	OrderNumber        string           `json:"order_number"`
	Email              string           `json:"email"`
	Phone              string           `json:"phone"`
	CreatedAt          time.Time        `json:"created_at"`
	Currency           string           `json:"currency"`
	SubtotalPrice      string           `json:"subtotal_price"`
	TotalPrice         string           `json:"total_price"`
	TotalTax           string           `json:"total_tax"`
	TotalDiscounts     string           `json:"total_discounts"`
	TotalShippingPrice string           `json:"total_shipping_price_set"`
	FinancialStatus    string           `json:"financial_status"`
	FulfillmentStatus  string           `json:"fulfillment_status"`
	Note               string           `json:"note"`
	Tags               []string         `json:"tags"`
	Customer           shopifyCustomer  `json:"customer"`
	ShippingAddress    shopifyAddress   `json:"shipping_address"`
	BillingAddress     *shopifyAddress  `json:"billing_address"`
	LineItems          []shopifyLineItem `json:"line_items"`
}

type shopifyCustomer struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
}

type shopifyAddress struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Company   string `json:"company"`
	Address1  string `json:"address1"`
	Address2  string `json:"address2"`
	City      string `json:"city"`
	Province  string `json:"province"`
	Zip       string `json:"zip"`
	Country   string `json:"country"`
	Phone     string `json:"phone"`
}

type shopifyLineItem struct {
	ID               int64  `json:"id"`
	ProductID        int64  `json:"product_id"`
	VariantID        int64  `json:"variant_id"`
	Title            string `json:"title"`
	SKU              string `json:"sku"`
	Quantity         int    `json:"quantity"`
	Price            string `json:"price"`
	TotalDiscount    string `json:"total_discount"`
	RequiresShipping bool   `json:"requires_shipping"`
	Grams            int    `json:"grams"`
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
