package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/wms-platform/shared/pkg/logging"
)

var tracer = otel.Tracer("seller-portal/clients")

// DownstreamMetrics interface for recording downstream service metrics
type DownstreamMetrics interface {
	RecordRequest(service, operation, status string, duration time.Duration)
}

// ServiceClient provides HTTP client functionality for calling microservices
type ServiceClient struct {
	httpClient *http.Client
	baseURL    string
	logger     *logging.Logger
	metrics    DownstreamMetrics
	service    string
}

// NewServiceClient creates a new service client
func NewServiceClient(baseURL string, logger *logging.Logger, metrics DownstreamMetrics, service string) *ServiceClient {
	return &ServiceClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		logger:  logger,
		metrics: metrics,
		service: service,
	}
}

func (c *ServiceClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	start := time.Now()
	operation := method + " " + path

	ctx, span := tracer.Start(ctx, c.service+"."+method,
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.url", c.baseURL+path),
			attribute.String("service", c.service),
		),
	)
	defer span.End()

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			c.recordError(operation, start, err)
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		c.recordError(operation, start, err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Inject trace context into outgoing request headers
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		c.recordError(operation, start, err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		span.RecordError(err)
		c.recordError(operation, start, err)
		return err
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			c.recordError(operation, start, err)
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	c.recordSuccess(operation, start)
	return nil
}

func (c *ServiceClient) recordSuccess(operation string, start time.Time) {
	if c.metrics != nil {
		c.metrics.RecordRequest(c.service, operation, "success", time.Since(start))
	}
}

func (c *ServiceClient) recordError(operation string, start time.Time, err error) {
	if c.metrics != nil {
		c.metrics.RecordRequest(c.service, operation, "error", time.Since(start))
	}
	if c.logger != nil {
		c.logger.WithError(err).Error("Downstream service call failed",
			"service", c.service,
			"operation", operation,
		)
	}
}

// SellerClient calls the seller service
type SellerClient struct {
	client *ServiceClient
}

// NewSellerClient creates a new seller client (non-instrumented)
func NewSellerClient(baseURL string) *SellerClient {
	return &SellerClient{
		client: &ServiceClient{
			httpClient: &http.Client{Timeout: 30 * time.Second},
			baseURL:    baseURL,
			service:    "seller-service",
		},
	}
}

// NewInstrumentedSellerClient creates an instrumented seller client
func NewInstrumentedSellerClient(baseURL string, logger *logging.Logger, metrics DownstreamMetrics) *SellerClient {
	return &SellerClient{
		client: NewServiceClient(baseURL, logger, metrics, "seller-service"),
	}
}

// GetSeller retrieves seller details
func (c *SellerClient) GetSeller(ctx context.Context, sellerID string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", "/api/v1/sellers/"+sellerID, nil, &result)
	return result, err
}

// GetSellerAPIKeys retrieves API keys for a seller
func (c *SellerClient) GetSellerAPIKeys(ctx context.Context, sellerID string) ([]map[string]interface{}, error) {
	var result struct {
		Keys []map[string]interface{} `json:"apiKeys"`
	}
	err := c.client.doRequest(ctx, "GET", "/api/v1/sellers/"+sellerID+"/api-keys", nil, &result)
	return result.Keys, err
}

// GenerateAPIKey creates a new API key
func (c *SellerClient) GenerateAPIKey(ctx context.Context, sellerID string, req map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "POST", "/api/v1/sellers/"+sellerID+"/api-keys", req, &result)
	return result, err
}

// RevokeAPIKey revokes an API key
func (c *SellerClient) RevokeAPIKey(ctx context.Context, sellerID, keyID string) error {
	return c.client.doRequest(ctx, "DELETE", "/api/v1/sellers/"+sellerID+"/api-keys/"+keyID, nil, nil)
}

// OrderClient calls the order service
type OrderClient struct {
	client *ServiceClient
}

// NewOrderClient creates a new order client (non-instrumented)
func NewOrderClient(baseURL string) *OrderClient {
	return &OrderClient{
		client: &ServiceClient{
			httpClient: &http.Client{Timeout: 30 * time.Second},
			baseURL:    baseURL,
			service:    "order-service",
		},
	}
}

// NewInstrumentedOrderClient creates an instrumented order client
func NewInstrumentedOrderClient(baseURL string, logger *logging.Logger, metrics DownstreamMetrics) *OrderClient {
	return &OrderClient{
		client: NewServiceClient(baseURL, logger, metrics, "order-service"),
	}
}

// GetOrders retrieves orders for a seller
func (c *OrderClient) GetOrders(ctx context.Context, params map[string]string) (map[string]interface{}, error) {
	query := "?"
	for k, v := range params {
		query += k + "=" + v + "&"
	}
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", "/api/v1/orders"+query, nil, &result)
	return result, err
}

// GetOrder retrieves a single order
func (c *OrderClient) GetOrder(ctx context.Context, orderID string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", "/api/v1/orders/"+orderID, nil, &result)
	return result, err
}

// GetOrderStats retrieves order statistics
func (c *OrderClient) GetOrderStats(ctx context.Context, sellerID string, startDate, endDate string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/v1/orders/stats?sellerId=%s&startDate=%s&endDate=%s", sellerID, startDate, endDate)
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", path, nil, &result)
	return result, err
}

// InventoryClient calls the inventory service
type InventoryClient struct {
	client *ServiceClient
}

// NewInventoryClient creates a new inventory client (non-instrumented)
func NewInventoryClient(baseURL string) *InventoryClient {
	return &InventoryClient{
		client: &ServiceClient{
			httpClient: &http.Client{Timeout: 30 * time.Second},
			baseURL:    baseURL,
			service:    "inventory-service",
		},
	}
}

// NewInstrumentedInventoryClient creates an instrumented inventory client
func NewInstrumentedInventoryClient(baseURL string, logger *logging.Logger, metrics DownstreamMetrics) *InventoryClient {
	return &InventoryClient{
		client: NewServiceClient(baseURL, logger, metrics, "inventory-service"),
	}
}

// GetInventory retrieves inventory for a seller
func (c *InventoryClient) GetInventory(ctx context.Context, params map[string]string) (map[string]interface{}, error) {
	query := "?"
	for k, v := range params {
		query += k + "=" + v + "&"
	}
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", "/api/v1/inventory"+query, nil, &result)
	return result, err
}

// GetInventoryStats retrieves inventory statistics
func (c *InventoryClient) GetInventoryStats(ctx context.Context, sellerID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/v1/inventory/stats?sellerId=%s", sellerID)
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", path, nil, &result)
	return result, err
}

// BillingClient calls the billing service
type BillingClient struct {
	client *ServiceClient
}

// NewBillingClient creates a new billing client (non-instrumented)
func NewBillingClient(baseURL string) *BillingClient {
	return &BillingClient{
		client: &ServiceClient{
			httpClient: &http.Client{Timeout: 30 * time.Second},
			baseURL:    baseURL,
			service:    "billing-service",
		},
	}
}

// NewInstrumentedBillingClient creates an instrumented billing client
func NewInstrumentedBillingClient(baseURL string, logger *logging.Logger, metrics DownstreamMetrics) *BillingClient {
	return &BillingClient{
		client: NewServiceClient(baseURL, logger, metrics, "billing-service"),
	}
}

// GetInvoices retrieves invoices for a seller
func (c *BillingClient) GetInvoices(ctx context.Context, params map[string]string) (map[string]interface{}, error) {
	query := "?"
	for k, v := range params {
		query += k + "=" + v + "&"
	}
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", "/api/v1/invoices"+query, nil, &result)
	return result, err
}

// GetInvoice retrieves a single invoice
func (c *BillingClient) GetInvoice(ctx context.Context, invoiceID string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", "/api/v1/invoices/"+invoiceID, nil, &result)
	return result, err
}

// GetBillingStats retrieves billing statistics
func (c *BillingClient) GetBillingStats(ctx context.Context, sellerID string, startDate, endDate string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/v1/billing/stats?sellerId=%s&startDate=%s&endDate=%s", sellerID, startDate, endDate)
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", path, nil, &result)
	return result, err
}

// GetCurrentBalance retrieves current balance
func (c *BillingClient) GetCurrentBalance(ctx context.Context, sellerID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/v1/sellers/%s/balance", sellerID)
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", path, nil, &result)
	return result, err
}

// ChannelClient calls the channel service
type ChannelClient struct {
	client *ServiceClient
}

// NewChannelClient creates a new channel client (non-instrumented)
func NewChannelClient(baseURL string) *ChannelClient {
	return &ChannelClient{
		client: &ServiceClient{
			httpClient: &http.Client{Timeout: 30 * time.Second},
			baseURL:    baseURL,
			service:    "channel-service",
		},
	}
}

// NewInstrumentedChannelClient creates an instrumented channel client
func NewInstrumentedChannelClient(baseURL string, logger *logging.Logger, metrics DownstreamMetrics) *ChannelClient {
	return &ChannelClient{
		client: NewServiceClient(baseURL, logger, metrics, "channel-service"),
	}
}

// GetChannels retrieves channels for a seller
func (c *ChannelClient) GetChannels(ctx context.Context, sellerID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/v1/sellers/%s/channels", sellerID)
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", path, nil, &result)
	return result, err
}

// GetChannel retrieves a single channel
func (c *ChannelClient) GetChannel(ctx context.Context, channelID string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "GET", "/api/v1/channels/"+channelID, nil, &result)
	return result, err
}

// ConnectChannel connects a new channel
func (c *ChannelClient) ConnectChannel(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "POST", "/api/v1/channels", req, &result)
	return result, err
}

// DisconnectChannel disconnects a channel
func (c *ChannelClient) DisconnectChannel(ctx context.Context, channelID string) error {
	return c.client.doRequest(ctx, "DELETE", "/api/v1/channels/"+channelID, nil, nil)
}

// SyncOrders triggers order sync for a channel
func (c *ChannelClient) SyncOrders(ctx context.Context, channelID string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "POST", "/api/v1/channels/"+channelID+"/sync/orders", nil, &result)
	return result, err
}

// SyncInventory triggers inventory sync for a channel
func (c *ChannelClient) SyncInventory(ctx context.Context, channelID string, items []map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.client.doRequest(ctx, "POST", "/api/v1/channels/"+channelID+"/sync/inventory", map[string]interface{}{"items": items}, &result)
	return result, err
}
