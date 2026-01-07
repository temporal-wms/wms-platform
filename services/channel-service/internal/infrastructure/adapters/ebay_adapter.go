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
	"net/url"
	"strings"
	"time"

	"github.com/wms-platform/services/channel-service/internal/domain"
)

// EbayAdapter implements ChannelAdapter for eBay
type EbayAdapter struct {
	httpClient *http.Client
	baseURL    string
	authURL    string
}

// NewEbayAdapter creates a new eBay adapter
func NewEbayAdapter() *EbayAdapter {
	return &EbayAdapter{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.ebay.com",
		authURL:    "https://api.ebay.com/identity/v1/oauth2/token",
	}
}

func (a *EbayAdapter) GetType() domain.ChannelType {
	return domain.ChannelTypeEbay
}

func (a *EbayAdapter) ValidateCredentials(ctx context.Context, creds domain.ChannelCredentials) error {
	if creds.ClientID == "" {
		return fmt.Errorf("client_id (App ID) is required")
	}
	if creds.ClientSecret == "" {
		return fmt.Errorf("client_secret (Cert ID) is required")
	}
	if creds.RefreshToken == "" {
		return fmt.Errorf("refresh_token is required")
	}

	// Try to get access token
	_, err := a.getAccessToken(ctx, creds)
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}

	return nil
}

func (a *EbayAdapter) getAccessToken(ctx context.Context, creds domain.ChannelCredentials) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", creds.RefreshToken)
	data.Set("scope", "https://api.ebay.com/oauth/api_scope https://api.ebay.com/oauth/api_scope/sell.fulfillment https://api.ebay.com/oauth/api_scope/sell.inventory")

	req, err := http.NewRequestWithContext(ctx, "POST", a.authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	// Basic auth with client credentials
	auth := base64.StdEncoding.EncodeToString([]byte(creds.ClientID + ":" + creds.ClientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
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

func (a *EbayAdapter) FetchOrders(ctx context.Context, channel *domain.Channel, since time.Time) ([]*domain.ChannelOrder, error) {
	accessToken, err := a.getAccessToken(ctx, channel.Credentials)
	if err != nil {
		return nil, err
	}

	// Use Fulfillment API to get orders
	endpoint := fmt.Sprintf("%s/sell/fulfillment/v1/order", a.baseURL)
	params := url.Values{}
	params.Set("filter", fmt.Sprintf("creationdate:[%s..]", since.Format("2006-01-02T15:04:05.000Z")))
	params.Set("limit", "50")

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
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
		Orders []struct {
			OrderID            string    `json:"orderId"`
			LegacyOrderID      string    `json:"legacyOrderId"`
			CreationDate       time.Time `json:"creationDate"`
			OrderFulfillmentStatus string `json:"orderFulfillmentStatus"`
			PricingSummary     struct {
				Total struct {
					Value    string `json:"value"`
					Currency string `json:"currency"`
				} `json:"total"`
			} `json:"pricingSummary"`
			Buyer struct {
				Username string `json:"username"`
			} `json:"buyer"`
			FulfillmentStartInstructions []struct {
				ShippingStep struct {
					ShipTo struct {
						FullName    string `json:"fullName"`
						ContactAddress struct {
							AddressLine1 string `json:"addressLine1"`
							AddressLine2 string `json:"addressLine2"`
							City         string `json:"city"`
							StateOrProvince string `json:"stateOrProvince"`
							PostalCode   string `json:"postalCode"`
							CountryCode  string `json:"countryCode"`
						} `json:"contactAddress"`
						PrimaryPhone struct {
							PhoneNumber string `json:"phoneNumber"`
						} `json:"primaryPhone"`
						Email string `json:"email"`
					} `json:"shipTo"`
				} `json:"shippingStep"`
			} `json:"fulfillmentStartInstructions"`
			LineItems []struct {
				LineItemID string `json:"lineItemId"`
				LegacyItemID string `json:"legacyItemId"`
				SKU        string `json:"sku"`
				Title      string `json:"title"`
				Quantity   int    `json:"quantity"`
				LineItemCost struct {
					Value string `json:"value"`
				} `json:"lineItemCost"`
			} `json:"lineItems"`
		} `json:"orders"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ordersResp); err != nil {
		return nil, err
	}

	var orders []*domain.ChannelOrder
	now := time.Now()
	for _, order := range ordersResp.Orders {
		channelOrder := &domain.ChannelOrder{
			TenantID:            channel.TenantID,
			SellerID:            channel.SellerID,
			ChannelID:           channel.ChannelID,
			ExternalOrderID:     order.OrderID,
			ExternalOrderNumber: order.LegacyOrderID,
			FulfillmentStatus:   order.OrderFulfillmentStatus,
			ExternalCreatedAt:   order.CreationDate,
			Currency:            order.PricingSummary.Total.Currency,
			CreatedAt:           now,
			UpdatedAt:           now,
		}

		var total float64
		fmt.Sscanf(order.PricingSummary.Total.Value, "%f", &total)
		channelOrder.Total = total

		// Get shipping address from first fulfillment instruction
		if len(order.FulfillmentStartInstructions) > 0 {
			shipTo := order.FulfillmentStartInstructions[0].ShippingStep.ShipTo
			// Parse name into first/last
			names := strings.SplitN(shipTo.FullName, " ", 2)
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
				Address1:  shipTo.ContactAddress.AddressLine1,
				Address2:  shipTo.ContactAddress.AddressLine2,
				City:      shipTo.ContactAddress.City,
				Province:  shipTo.ContactAddress.StateOrProvince,
				Zip:       shipTo.ContactAddress.PostalCode,
				Country:   shipTo.ContactAddress.CountryCode,
				Phone:     shipTo.PrimaryPhone.PhoneNumber,
			}
			channelOrder.Customer = domain.ChannelCustomer{
				Email:     shipTo.Email,
				FirstName: firstName,
				LastName:  lastName,
			}
		}

		// Parse line items
		for _, item := range order.LineItems {
			var price float64
			fmt.Sscanf(item.LineItemCost.Value, "%f", &price)
			unitPrice := price
			if item.Quantity > 0 {
				unitPrice = price / float64(item.Quantity)
			}

			channelOrder.LineItems = append(channelOrder.LineItems, domain.ChannelLineItem{
				ExternalID: item.LineItemID,
				SKU:        item.SKU,
				Title:      item.Title,
				Quantity:   item.Quantity,
				Price:      unitPrice,
				ProductID:  item.LegacyItemID,
			})
		}

		orders = append(orders, channelOrder)
	}

	return orders, nil
}

func (a *EbayAdapter) FetchOrder(ctx context.Context, channel *domain.Channel, externalOrderID string) (*domain.ChannelOrder, error) {
	accessToken, err := a.getAccessToken(ctx, channel.Credentials)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/sell/fulfillment/v1/order/%s", a.baseURL, externalOrderID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

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

	// Parse and return order (similar to FetchOrders)
	return nil, fmt.Errorf("not implemented")
}

func (a *EbayAdapter) PushTracking(ctx context.Context, channel *domain.Channel, externalOrderID string, tracking domain.TrackingInfo) error {
	accessToken, err := a.getAccessToken(ctx, channel.Credentials)
	if err != nil {
		return err
	}

	// Create shipping fulfillment
	endpoint := fmt.Sprintf("%s/sell/fulfillment/v1/order/%s/shipping_fulfillment", a.baseURL, externalOrderID)

	fulfillmentReq := map[string]interface{}{
		"trackingNumber":  tracking.TrackingNumber,
		"shippingCarrierCode": a.mapCarrierCode(tracking.Carrier),
		"lineItems": tracking.LineItemIDs,
	}

	body, _ := json.Marshal(fulfillmentReq)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to push tracking: %s", string(respBody))
	}

	return nil
}

func (a *EbayAdapter) mapCarrierCode(carrier string) string {
	carrierMap := map[string]string{
		"ups":   "UPS",
		"usps":  "USPS",
		"fedex": "FEDEX",
		"dhl":   "DHL",
	}
	if code, ok := carrierMap[strings.ToLower(carrier)]; ok {
		return code
	}
	return carrier
}

func (a *EbayAdapter) SyncInventory(ctx context.Context, channel *domain.Channel, items []domain.InventoryUpdate) error {
	accessToken, err := a.getAccessToken(ctx, channel.Credentials)
	if err != nil {
		return err
	}

	// Use Inventory API to update quantities
	for _, item := range items {
		endpoint := fmt.Sprintf("%s/sell/inventory/v1/inventory_item/%s", a.baseURL, item.SKU)

		// First get the inventory item
		getReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			continue
		}
		getReq.Header.Set("Authorization", "Bearer "+accessToken)

		getResp, err := a.httpClient.Do(getReq)
		if err != nil {
			continue
		}

		if getResp.StatusCode == http.StatusOK {
			var invItem map[string]interface{}
			json.NewDecoder(getResp.Body).Decode(&invItem)
			getResp.Body.Close()

			// Update availability
			if availability, ok := invItem["availability"].(map[string]interface{}); ok {
				if shipToLocAvail, ok := availability["shipToLocationAvailability"].(map[string]interface{}); ok {
					shipToLocAvail["quantity"] = item.Available
				}
			}

			// PUT the updated item
			body, _ := json.Marshal(invItem)
			putReq, err := http.NewRequestWithContext(ctx, "PUT", endpoint, bytes.NewReader(body))
			if err != nil {
				continue
			}
			putReq.Header.Set("Authorization", "Bearer "+accessToken)
			putReq.Header.Set("Content-Type", "application/json")

			putResp, err := a.httpClient.Do(putReq)
			if err != nil {
				continue
			}
			putResp.Body.Close()
		} else {
			getResp.Body.Close()
		}
	}

	return nil
}

func (a *EbayAdapter) GetInventoryLevels(ctx context.Context, channel *domain.Channel, skus []string) ([]domain.InventoryLevel, error) {
	accessToken, err := a.getAccessToken(ctx, channel.Credentials)
	if err != nil {
		return nil, err
	}

	var levels []domain.InventoryLevel

	for _, sku := range skus {
		endpoint := fmt.Sprintf("%s/sell/inventory/v1/inventory_item/%s", a.baseURL, sku)

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := a.httpClient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var invItem struct {
				SKU          string `json:"sku"`
				Availability struct {
					ShipToLocationAvailability struct {
						Quantity int `json:"quantity"`
					} `json:"shipToLocationAvailability"`
				} `json:"availability"`
			}
			json.NewDecoder(resp.Body).Decode(&invItem)

			levels = append(levels, domain.InventoryLevel{
				SKU:       invItem.SKU,
				Available: invItem.Availability.ShipToLocationAvailability.Quantity,
				OnHand:    invItem.Availability.ShipToLocationAvailability.Quantity,
			})
		}
		resp.Body.Close()
	}

	return levels, nil
}

func (a *EbayAdapter) CreateFulfillment(ctx context.Context, channel *domain.Channel, fulfillment domain.FulfillmentRequest) error {
	lineItemIDs := make([]string, len(fulfillment.LineItems))
	for i, item := range fulfillment.LineItems {
		lineItemIDs[i] = item.LineItemID
	}

	return a.PushTracking(ctx, channel, fulfillment.OrderID, domain.TrackingInfo{
		TrackingNumber: fulfillment.TrackingNumber,
		Carrier:        fulfillment.Carrier,
		TrackingURL:    fulfillment.TrackingURL,
		LineItemIDs:    lineItemIDs,
	})
}

func (a *EbayAdapter) RegisterWebhooks(ctx context.Context, channel *domain.Channel, webhookURL string) error {
	accessToken, err := a.getAccessToken(ctx, channel.Credentials)
	if err != nil {
		return err
	}

	// Create notification subscription
	endpoint := fmt.Sprintf("%s/commerce/notification/v1/subscription", a.baseURL)

	topics := []string{
		"MARKETPLACE_ACCOUNT_DELETION",
	}

	for _, topic := range topics {
		subReq := map[string]interface{}{
			"topicId": topic,
			"destinationId": webhookURL,
			"status": "ENABLED",
		}

		body, _ := json.Marshal(subReq)
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.httpClient.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
	}

	return nil
}

func (a *EbayAdapter) ValidateWebhook(ctx context.Context, channel *domain.Channel, signature string, body []byte) bool {
	if signature == "" {
		return false
	}

	// eBay uses X-EBAY-SIGNATURE header
	mac := hmac.New(sha256.New, []byte(channel.Credentials.ClientSecret))
	mac.Write(body)
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}
