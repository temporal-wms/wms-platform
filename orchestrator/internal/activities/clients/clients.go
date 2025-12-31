package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ServiceClients holds HTTP clients for all WMS services
type ServiceClients struct {
	config     *Config
	httpClient *http.Client
}

// Config holds service URLs
type Config struct {
	OrderServiceURL         string
	InventoryServiceURL     string
	RoutingServiceURL       string
	PickingServiceURL       string
	ConsolidationServiceURL string
	PackingServiceURL       string
	ShippingServiceURL      string
	LaborServiceURL         string
	WavingServiceURL        string
}

// NewServiceClients creates a new ServiceClients instance
func NewServiceClients(config *Config) *ServiceClients {
	return &ServiceClients{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request and decodes the response
func (c *ServiceClients) doRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// OrderService methods

// ValidateOrder calls order-service to validate an order
func (c *ServiceClients) ValidateOrder(ctx context.Context, orderID string) (*OrderValidationResult, error) {
	url := fmt.Sprintf("%s/api/v1/orders/%s/validate", c.config.OrderServiceURL, orderID)
	// Order-service returns an OrderDTO, not OrderValidationResult
	var order Order
	if err := c.doRequest(ctx, http.MethodPut, url, nil, &order); err != nil {
		return nil, err
	}
	// Check if order was validated successfully by checking status
	isValid := order.Status == "validated" || order.Status == "wave_assigned" ||
		order.Status == "picking" || order.Status == "consolidated" ||
		order.Status == "packed" || order.Status == "shipped"
	return &OrderValidationResult{
		OrderID:     order.OrderID,
		Valid:       isValid,
		ValidatedAt: order.UpdatedAt,
	}, nil
}

// GetOrder retrieves an order from order-service
func (c *ServiceClients) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	url := fmt.Sprintf("%s/api/v1/orders/%s", c.config.OrderServiceURL, orderID)
	var result Order
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CancelOrder cancels an order
func (c *ServiceClients) CancelOrder(ctx context.Context, orderID, reason string) error {
	url := fmt.Sprintf("%s/api/v1/orders/%s/cancel", c.config.OrderServiceURL, orderID)
	body := map[string]string{"reason": reason}
	return c.doRequest(ctx, http.MethodPost, url, body, nil)
}

// InventoryService methods

// ReserveInventory reserves inventory for an order
func (c *ServiceClients) ReserveInventory(ctx context.Context, req *ReserveInventoryRequest) error {
	url := fmt.Sprintf("%s/api/v1/inventory/reserve", c.config.InventoryServiceURL)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// ReleaseInventoryReservation releases reserved inventory
func (c *ServiceClients) ReleaseInventoryReservation(ctx context.Context, orderID string) error {
	url := fmt.Sprintf("%s/api/v1/inventory/release/%s", c.config.InventoryServiceURL, orderID)
	return c.doRequest(ctx, http.MethodPost, url, nil, nil)
}

// GetInventoryBySKU retrieves inventory for a SKU
func (c *ServiceClients) GetInventoryBySKU(ctx context.Context, sku string) (*InventoryItem, error) {
	url := fmt.Sprintf("%s/api/v1/inventory/sku/%s", c.config.InventoryServiceURL, sku)
	var result InventoryItem
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RoutingService methods

// CalculateRoute calculates the optimal pick route
func (c *ServiceClients) CalculateRoute(ctx context.Context, req *CalculateRouteRequest) (*Route, error) {
	url := fmt.Sprintf("%s/api/v1/routes", c.config.RoutingServiceURL)
	var result Route
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRoute retrieves a route by ID
func (c *ServiceClients) GetRoute(ctx context.Context, routeID string) (*Route, error) {
	url := fmt.Sprintf("%s/api/v1/routes/%s", c.config.RoutingServiceURL, routeID)
	var result Route
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PickingService methods

// CreatePickTask creates a new pick task
func (c *ServiceClients) CreatePickTask(ctx context.Context, req *CreatePickTaskRequest) (*PickTask, error) {
	url := fmt.Sprintf("%s/api/v1/tasks", c.config.PickingServiceURL)
	var result PickTask
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPickTask retrieves a pick task by ID
func (c *ServiceClients) GetPickTask(ctx context.Context, taskID string) (*PickTask, error) {
	url := fmt.Sprintf("%s/api/v1/tasks/%s", c.config.PickingServiceURL, taskID)
	var result PickTask
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AssignPickTask assigns a worker to a pick task
func (c *ServiceClients) AssignPickTask(ctx context.Context, taskID, pickerID, toteID string) error {
	url := fmt.Sprintf("%s/api/v1/tasks/%s/assign", c.config.PickingServiceURL, taskID)
	body := map[string]string{
		"pickerId": pickerID,
		"toteId":   toteID,
	}
	return c.doRequest(ctx, http.MethodPost, url, body, nil)
}

// ConfirmPick confirms item picking
func (c *ServiceClients) ConfirmPick(ctx context.Context, taskID string, req *ConfirmPickRequest) error {
	url := fmt.Sprintf("%s/api/v1/tasks/%s/pick", c.config.PickingServiceURL, taskID)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// CompletePickTask marks a pick task as complete
func (c *ServiceClients) CompletePickTask(ctx context.Context, taskID string) (*PickTask, error) {
	url := fmt.Sprintf("%s/api/v1/tasks/%s/complete", c.config.PickingServiceURL, taskID)
	var result PickTask
	if err := c.doRequest(ctx, http.MethodPost, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ConsolidationService methods

// CreateConsolidation creates a new consolidation unit
func (c *ServiceClients) CreateConsolidation(ctx context.Context, req *CreateConsolidationRequest) (*ConsolidationUnit, error) {
	url := fmt.Sprintf("%s/api/v1/consolidations", c.config.ConsolidationServiceURL)
	var result ConsolidationUnit
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetConsolidation retrieves a consolidation unit
func (c *ServiceClients) GetConsolidation(ctx context.Context, consolidationID string) (*ConsolidationUnit, error) {
	url := fmt.Sprintf("%s/api/v1/consolidations/%s", c.config.ConsolidationServiceURL, consolidationID)
	var result ConsolidationUnit
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ConsolidateItem consolidates an item
func (c *ServiceClients) ConsolidateItem(ctx context.Context, consolidationID string, req *ConsolidateItemRequest) error {
	url := fmt.Sprintf("%s/api/v1/consolidations/%s/consolidate", c.config.ConsolidationServiceURL, consolidationID)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// CompleteConsolidation marks consolidation as complete
func (c *ServiceClients) CompleteConsolidation(ctx context.Context, consolidationID string) (*ConsolidationUnit, error) {
	url := fmt.Sprintf("%s/api/v1/consolidations/%s/complete", c.config.ConsolidationServiceURL, consolidationID)
	var result ConsolidationUnit
	if err := c.doRequest(ctx, http.MethodPost, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PackingService methods

// CreatePackTask creates a new pack task
func (c *ServiceClients) CreatePackTask(ctx context.Context, req *CreatePackTaskRequest) (*PackTask, error) {
	url := fmt.Sprintf("%s/api/v1/tasks", c.config.PackingServiceURL)
	var result PackTask
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPackTask retrieves a pack task
func (c *ServiceClients) GetPackTask(ctx context.Context, taskID string) (*PackTask, error) {
	url := fmt.Sprintf("%s/api/v1/tasks/%s", c.config.PackingServiceURL, taskID)
	var result PackTask
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SealPackage seals a package
func (c *ServiceClients) SealPackage(ctx context.Context, taskID string, req *SealPackageRequest) error {
	url := fmt.Sprintf("%s/api/v1/tasks/%s/seal", c.config.PackingServiceURL, taskID)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// ApplyLabel applies a label to a package
func (c *ServiceClients) ApplyLabel(ctx context.Context, taskID string, req *ApplyLabelRequest) error {
	url := fmt.Sprintf("%s/api/v1/tasks/%s/label", c.config.PackingServiceURL, taskID)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// CompletePackTask marks packing as complete
func (c *ServiceClients) CompletePackTask(ctx context.Context, taskID string) (*PackTask, error) {
	url := fmt.Sprintf("%s/api/v1/tasks/%s/complete", c.config.PackingServiceURL, taskID)
	var result PackTask
	if err := c.doRequest(ctx, http.MethodPost, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ShippingService methods

// CreateShipment creates a new shipment
func (c *ServiceClients) CreateShipment(ctx context.Context, req *CreateShipmentRequest) (*Shipment, error) {
	url := fmt.Sprintf("%s/api/v1/shipments", c.config.ShippingServiceURL)
	var result Shipment
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetShipment retrieves a shipment
func (c *ServiceClients) GetShipment(ctx context.Context, shipmentID string) (*Shipment, error) {
	url := fmt.Sprintf("%s/api/v1/shipments/%s", c.config.ShippingServiceURL, shipmentID)
	var result Shipment
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GenerateLabelRequest represents a request to generate a label
type GenerateLabelRequest struct {
	LabelFormat string `json:"labelFormat"`
}

// GenerateLabel generates a shipping label
func (c *ServiceClients) GenerateLabel(ctx context.Context, shipmentID string) (*ShippingLabel, error) {
	url := fmt.Sprintf("%s/api/v1/shipments/%s/label", c.config.ShippingServiceURL, shipmentID)
	var result ShippingLabel
	req := GenerateLabelRequest{
		LabelFormat: "PDF",
	}
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// MarkShipped marks a shipment as shipped
func (c *ServiceClients) MarkShipped(ctx context.Context, shipmentID string) error {
	url := fmt.Sprintf("%s/api/v1/shipments/%s/ship", c.config.ShippingServiceURL, shipmentID)
	return c.doRequest(ctx, http.MethodPost, url, nil, nil)
}

// LaborService methods

// AssignWorker assigns a worker to a task
func (c *ServiceClients) AssignWorker(ctx context.Context, req *AssignWorkerRequest) (*LaborTask, error) {
	url := fmt.Sprintf("%s/api/v1/tasks", c.config.LaborServiceURL)
	var result LaborTask
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAvailableWorkers gets available workers for a task type
func (c *ServiceClients) GetAvailableWorkers(ctx context.Context, taskType, zone string) ([]Worker, error) {
	url := fmt.Sprintf("%s/api/v1/workers/available?taskType=%s&zone=%s", c.config.LaborServiceURL, taskType, zone)
	var result []Worker
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// WavingService methods

// AssignOrderToWave assigns an order to a wave
func (c *ServiceClients) AssignOrderToWave(ctx context.Context, waveID, orderID string) error {
	url := fmt.Sprintf("%s/api/v1/waves/%s/orders", c.config.WavingServiceURL, waveID)
	body := map[string]string{"orderId": orderID}
	return c.doRequest(ctx, http.MethodPost, url, body, nil)
}

// GetWave retrieves a wave by ID
func (c *ServiceClients) GetWave(ctx context.Context, waveID string) (*Wave, error) {
	url := fmt.Sprintf("%s/api/v1/waves/%s", c.config.WavingServiceURL, waveID)
	var result Wave
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Reprocessing Service methods

// EligibleOrder represents an order eligible for reprocessing
type EligibleOrder struct {
	OrderID       string    `json:"orderId"`
	WorkflowID    string    `json:"workflowId"`
	RunID         string    `json:"runId"`
	FailureStatus string    `json:"failureStatus"`
	FailureReason string    `json:"failureReason"`
	FailedAt      time.Time `json:"failedAt"`
	RetryCount    int       `json:"retryCount"`
	CustomerID    string    `json:"customerId"`
	Priority      string    `json:"priority"`
}

// EligibleOrdersResponse represents the response from the eligible orders endpoint
type EligibleOrdersResponse struct {
	Data  []EligibleOrder `json:"data"`
	Total int64           `json:"total"`
}

// GetEligibleOrders retrieves orders eligible for reprocessing from order-service
func (c *ServiceClients) GetEligibleOrders(ctx context.Context, failureStatuses []string, maxRetries int, limit int) (*EligibleOrdersResponse, error) {
	url := fmt.Sprintf("%s/api/v1/reprocessing/eligible?limit=%d&maxRetries=%d",
		c.config.OrderServiceURL, limit, maxRetries)

	for _, status := range failureStatuses {
		url += fmt.Sprintf("&status=%s", status)
	}

	var result EligibleOrdersResponse
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IncrementRetryCountRequest is the request for incrementing retry count
type IncrementRetryCountRequest struct {
	FailureStatus string `json:"failureStatus"`
	FailureReason string `json:"failureReason"`
	WorkflowID    string `json:"workflowId"`
	RunID         string `json:"runId"`
}

// IncrementRetryCount increments the retry count for an order in order-service
func (c *ServiceClients) IncrementRetryCount(ctx context.Context, orderID string, req *IncrementRetryCountRequest) error {
	url := fmt.Sprintf("%s/api/v1/reprocessing/orders/%s/retry-count", c.config.OrderServiceURL, orderID)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// ResetOrderForRetry resets an order for retry processing
func (c *ServiceClients) ResetOrderForRetry(ctx context.Context, orderID string) error {
	url := fmt.Sprintf("%s/api/v1/reprocessing/orders/%s/reset", c.config.OrderServiceURL, orderID)
	return c.doRequest(ctx, http.MethodPost, url, nil, nil)
}

// MoveToDeadLetterRequest is the request for moving an order to DLQ
type MoveToDeadLetterRequest struct {
	FailureStatus string `json:"failureStatus"`
	FailureReason string `json:"failureReason"`
	RetryCount    int    `json:"retryCount"`
	WorkflowID    string `json:"workflowId"`
	RunID         string `json:"runId"`
}

// MoveToDeadLetter moves an order to the dead letter queue
func (c *ServiceClients) MoveToDeadLetter(ctx context.Context, orderID string, req *MoveToDeadLetterRequest) error {
	url := fmt.Sprintf("%s/api/v1/reprocessing/orders/%s/dlq", c.config.OrderServiceURL, orderID)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// Station Service methods (for process path routing)

// FindCapableStations finds stations with all required capabilities
func (c *ServiceClients) FindCapableStations(ctx context.Context, req *FindCapableStationsRequest) ([]Station, error) {
	url := fmt.Sprintf("%s/api/v1/stations/find-capable", c.config.LaborServiceURL)
	var result []Station
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetStation retrieves a station by ID
func (c *ServiceClients) GetStation(ctx context.Context, stationID string) (*Station, error) {
	url := fmt.Sprintf("%s/api/v1/stations/%s", c.config.LaborServiceURL, stationID)
	var result Station
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetStationsByZone retrieves stations in a zone
func (c *ServiceClients) GetStationsByZone(ctx context.Context, zone string) ([]Station, error) {
	url := fmt.Sprintf("%s/api/v1/stations/zone/%s", c.config.LaborServiceURL, zone)
	var result []Station
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetStationsByType retrieves stations by type
func (c *ServiceClients) GetStationsByType(ctx context.Context, stationType string) ([]Station, error) {
	url := fmt.Sprintf("%s/api/v1/stations/type/%s", c.config.LaborServiceURL, stationType)
	var result []Station
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}
