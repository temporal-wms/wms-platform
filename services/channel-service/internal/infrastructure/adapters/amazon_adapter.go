package adapters

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wms-platform/services/channel-service/internal/domain"
)

// AmazonAdapter implements ChannelAdapter for Amazon Seller Central / SP-API
type AmazonAdapter struct {
	httpClient *http.Client
	baseURL    string
	authURL    string
}

// NewAmazonAdapter creates a new Amazon adapter
func NewAmazonAdapter() *AmazonAdapter {
	return &AmazonAdapter{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://sellingpartnerapi-na.amazon.com", // NA endpoint
		authURL:    "https://api.amazon.com/auth/o2/token",
	}
}

func (a *AmazonAdapter) GetType() domain.ChannelType {
	return domain.ChannelTypeAmazon
}

func (a *AmazonAdapter) ValidateCredentials(ctx context.Context, creds domain.ChannelCredentials) error {
	// Validate required Amazon SP-API credentials
	if creds.ClientID == "" {
		return fmt.Errorf("client_id (LWA Client ID) is required")
	}
	if creds.ClientSecret == "" {
		return fmt.Errorf("client_secret (LWA Client Secret) is required")
	}
	if creds.RefreshToken == "" {
		return fmt.Errorf("refresh_token is required")
	}

	// Additional Amazon-specific fields from AdditionalConfig
	if creds.AdditionalConfig == nil {
		return fmt.Errorf("additional_config with seller_id and marketplace_id is required")
	}

	if _, ok := creds.AdditionalConfig["seller_id"]; !ok {
		return fmt.Errorf("seller_id is required in additional_config")
	}
	if _, ok := creds.AdditionalConfig["marketplace_id"]; !ok {
		return fmt.Errorf("marketplace_id is required in additional_config")
	}

	// Try to get access token to validate credentials
	_, err := a.getAccessToken(ctx, creds)
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}

	return nil
}

func (a *AmazonAdapter) getAccessToken(ctx context.Context, creds domain.ChannelCredentials) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", creds.RefreshToken)
	data.Set("client_id", creds.ClientID)
	data.Set("client_secret", creds.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", a.authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

func (a *AmazonAdapter) FetchOrders(ctx context.Context, channel *domain.Channel, since time.Time) ([]*domain.ChannelOrder, error) {
	accessToken, err := a.getAccessToken(ctx, channel.Credentials)
	if err != nil {
		return nil, err
	}

	marketplaceID := channel.Credentials.AdditionalConfig["marketplace_id"].(string)

	// Build request to Orders API
	endpoint := fmt.Sprintf("%s/orders/v0/orders", a.baseURL)
	params := url.Values{}
	params.Set("MarketplaceIds", marketplaceID)
	params.Set("CreatedAfter", since.Format(time.RFC3339))
	params.Set("OrderStatuses", "Unshipped,PartiallyShipped")

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-amz-access-token", accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch orders: %s", string(body))
	}

	var ordersResp struct {
		Payload struct {
			Orders []struct {
				AmazonOrderID     string    `json:"AmazonOrderId"`
				OrderStatus       string    `json:"OrderStatus"`
				PurchaseDate      time.Time `json:"PurchaseDate"`
				OrderTotal        *struct {
					Amount       string `json:"Amount"`
					CurrencyCode string `json:"CurrencyCode"`
				} `json:"OrderTotal"`
				ShippingAddress *struct {
					Name          string `json:"Name"`
					AddressLine1  string `json:"AddressLine1"`
					AddressLine2  string `json:"AddressLine2"`
					City          string `json:"City"`
					StateOrRegion string `json:"StateOrRegion"`
					PostalCode    string `json:"PostalCode"`
					CountryCode   string `json:"CountryCode"`
					Phone         string `json:"Phone"`
				} `json:"ShippingAddress"`
				BuyerInfo *struct {
					BuyerEmail string `json:"BuyerEmail"`
					BuyerName  string `json:"BuyerName"`
				} `json:"BuyerInfo"`
			} `json:"Orders"`
		} `json:"payload"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ordersResp); err != nil {
		return nil, err
	}

	var orders []*domain.ChannelOrder
	now := time.Now()
	for _, order := range ordersResp.Payload.Orders {
		channelOrder := &domain.ChannelOrder{
			TenantID:            channel.TenantID,
			SellerID:            channel.SellerID,
			ChannelID:           channel.ChannelID,
			ExternalOrderID:     order.AmazonOrderID,
			ExternalOrderNumber: order.AmazonOrderID,
			FulfillmentStatus:   order.OrderStatus,
			ExternalCreatedAt:   order.PurchaseDate,
			Currency:            "USD",
			CreatedAt:           now,
			UpdatedAt:           now,
		}

		if order.OrderTotal != nil {
			var amount float64
			fmt.Sscanf(order.OrderTotal.Amount, "%f", &amount)
			channelOrder.Total = amount
			channelOrder.Currency = order.OrderTotal.CurrencyCode
		}

		if order.ShippingAddress != nil {
			// Parse name into first/last
			names := strings.SplitN(order.ShippingAddress.Name, " ", 2)
			firstName := ""
			lastName := ""
			if len(names) > 0 {
				firstName = names[0]
			}
			if len(names) > 1 {
				lastName = names[1]
			}
			channelOrder.ShippingAddr = domain.ChannelAddress{
				FirstName: firstName,
				LastName:  lastName,
				Address1:  order.ShippingAddress.AddressLine1,
				Address2:  order.ShippingAddress.AddressLine2,
				City:      order.ShippingAddress.City,
				Province:  order.ShippingAddress.StateOrRegion,
				Zip:       order.ShippingAddress.PostalCode,
				Country:   order.ShippingAddress.CountryCode,
				Phone:     order.ShippingAddress.Phone,
			}
		}

		if order.BuyerInfo != nil {
			channelOrder.Customer = domain.ChannelCustomer{
				Email: order.BuyerInfo.BuyerEmail,
			}
			// Parse buyer name
			names := strings.SplitN(order.BuyerInfo.BuyerName, " ", 2)
			if len(names) > 0 {
				channelOrder.Customer.FirstName = names[0]
			}
			if len(names) > 1 {
				channelOrder.Customer.LastName = names[1]
			}
		}

		// Fetch order items separately
		items, err := a.fetchOrderItems(ctx, channel, order.AmazonOrderID, accessToken)
		if err == nil {
			channelOrder.LineItems = items
		}

		orders = append(orders, channelOrder)
	}

	return orders, nil
}

func (a *AmazonAdapter) fetchOrderItems(ctx context.Context, channel *domain.Channel, orderID, accessToken string) ([]domain.ChannelLineItem, error) {
	endpoint := fmt.Sprintf("%s/orders/v0/orders/%s/orderItems", a.baseURL, orderID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-amz-access-token", accessToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch order items")
	}

	var itemsResp struct {
		Payload struct {
			OrderItems []struct {
				ASIN              string `json:"ASIN"`
				SellerSKU         string `json:"SellerSKU"`
				OrderItemId       string `json:"OrderItemId"`
				Title             string `json:"Title"`
				QuantityOrdered   int    `json:"QuantityOrdered"`
				ItemPrice         *struct {
					Amount string `json:"Amount"`
				} `json:"ItemPrice"`
			} `json:"OrderItems"`
		} `json:"payload"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&itemsResp); err != nil {
		return nil, err
	}

	var items []domain.ChannelLineItem
	for _, item := range itemsResp.Payload.OrderItems {
		lineItem := domain.ChannelLineItem{
			ExternalID: item.OrderItemId,
			SKU:        item.SellerSKU,
			Title:      item.Title,
			Quantity:   item.QuantityOrdered,
			ProductID:  item.ASIN,
		}

		if item.ItemPrice != nil {
			var price float64
			fmt.Sscanf(item.ItemPrice.Amount, "%f", &price)
			if item.QuantityOrdered > 0 {
				lineItem.Price = price / float64(item.QuantityOrdered)
			}
		}

		items = append(items, lineItem)
	}

	return items, nil
}

func (a *AmazonAdapter) FetchOrder(ctx context.Context, channel *domain.Channel, externalOrderID string) (*domain.ChannelOrder, error) {
	accessToken, err := a.getAccessToken(ctx, channel.Credentials)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/orders/v0/orders/%s", a.baseURL, externalOrderID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-amz-access-token", accessToken)

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

	// Parse response and build order (similar to FetchOrders)
	// Implementation would be similar to FetchOrders for a single order

	return nil, fmt.Errorf("not implemented")
}

func (a *AmazonAdapter) PushTracking(ctx context.Context, channel *domain.Channel, externalOrderID string, tracking domain.TrackingInfo) error {
	accessToken, err := a.getAccessToken(ctx, channel.Credentials)
	if err != nil {
		return err
	}

	// Use Feeds API to submit tracking
	// This is a simplified version - actual implementation would use Feed submission
	endpoint := fmt.Sprintf("%s/feeds/2021-06-30/feeds", a.baseURL)

	feedContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<AmazonEnvelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="amzn-envelope.xsd">
  <Header>
    <DocumentVersion>1.01</DocumentVersion>
    <MerchantIdentifier>%s</MerchantIdentifier>
  </Header>
  <MessageType>OrderFulfillment</MessageType>
  <Message>
    <MessageID>1</MessageID>
    <OrderFulfillment>
      <AmazonOrderID>%s</AmazonOrderID>
      <FulfillmentDate>%s</FulfillmentDate>
      <FulfillmentData>
        <CarrierName>%s</CarrierName>
        <ShippingMethod>Standard</ShippingMethod>
        <ShipperTrackingNumber>%s</ShipperTrackingNumber>
      </FulfillmentData>
    </OrderFulfillment>
  </Message>
</AmazonEnvelope>`,
		channel.Credentials.AdditionalConfig["seller_id"],
		externalOrderID,
		time.Now().Format(time.RFC3339),
		tracking.Carrier,
		tracking.TrackingNumber,
	)

	feedReq := map[string]interface{}{
		"feedType":      "POST_ORDER_FULFILLMENT_DATA",
		"marketplaceIds": []string{channel.Credentials.AdditionalConfig["marketplace_id"].(string)},
	}

	feedJSON, _ := json.Marshal(feedReq)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(feedJSON)))
	if err != nil {
		return err
	}
	req.Header.Set("x-amz-access-token", accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to submit tracking feed: %s", string(body))
	}

	// Would need to upload feed content to returned URL
	_ = feedContent // Used in actual implementation

	return nil
}

func (a *AmazonAdapter) SyncInventory(ctx context.Context, channel *domain.Channel, items []domain.InventoryUpdate) error {
	accessToken, err := a.getAccessToken(ctx, channel.Credentials)
	if err != nil {
		return err
	}

	// Use Feeds API for inventory updates
	endpoint := fmt.Sprintf("%s/feeds/2021-06-30/feeds", a.baseURL)

	feedReq := map[string]interface{}{
		"feedType":       "POST_INVENTORY_AVAILABILITY_DATA",
		"marketplaceIds": []string{channel.Credentials.AdditionalConfig["marketplace_id"].(string)},
	}

	feedJSON, _ := json.Marshal(feedReq)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(feedJSON)))
	if err != nil {
		return err
	}
	req.Header.Set("x-amz-access-token", accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to submit inventory feed: %s", string(body))
	}

	return nil
}

func (a *AmazonAdapter) GetInventoryLevels(ctx context.Context, channel *domain.Channel, skus []string) ([]domain.InventoryLevel, error) {
	accessToken, err := a.getAccessToken(ctx, channel.Credentials)
	if err != nil {
		return nil, err
	}

	// Use FBA Inventory API
	endpoint := fmt.Sprintf("%s/fba/inventory/v1/summaries", a.baseURL)
	params := url.Values{}
	params.Set("granularityType", "Marketplace")
	params.Set("granularityId", channel.Credentials.AdditionalConfig["marketplace_id"].(string))
	params.Set("sellerSkus", strings.Join(skus, ","))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-amz-access-token", accessToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get inventory: %s", string(body))
	}

	var invResp struct {
		Payload struct {
			InventorySummaries []struct {
				SellerSku       string `json:"sellerSku"`
				ASIN            string `json:"asin"`
				TotalQuantity   int    `json:"totalQuantity"`
				AvailableQty    int    `json:"availableQuantity"`
			} `json:"inventorySummaries"`
		} `json:"payload"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&invResp); err != nil {
		return nil, err
	}

	var levels []domain.InventoryLevel
	for _, inv := range invResp.Payload.InventorySummaries {
		levels = append(levels, domain.InventoryLevel{
			SKU:       inv.SellerSku,
			ProductID: inv.ASIN,
			Available: inv.AvailableQty,
			OnHand:    inv.TotalQuantity,
		})
	}

	return levels, nil
}

func (a *AmazonAdapter) CreateFulfillment(ctx context.Context, channel *domain.Channel, fulfillment domain.FulfillmentRequest) error {
	return a.PushTracking(ctx, channel, fulfillment.OrderID, domain.TrackingInfo{
		TrackingNumber: fulfillment.TrackingNumber,
		Carrier:        fulfillment.Carrier,
		TrackingURL:    fulfillment.TrackingURL,
	})
}

func (a *AmazonAdapter) RegisterWebhooks(ctx context.Context, channel *domain.Channel, webhookURL string) error {
	// Amazon uses EventBridge for notifications, not traditional webhooks
	// This would require setting up EventBridge destinations
	return nil
}

func (a *AmazonAdapter) ValidateWebhook(ctx context.Context, channel *domain.Channel, signature string, body []byte) bool {
	// Validate Amazon SNS/EventBridge signature
	if signature == "" {
		return false
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(channel.Credentials.ClientSecret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}
