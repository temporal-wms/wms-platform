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
	FacilityServiceURL      string
	UnitServiceURL          string
	ProcessPathServiceURL   string
	BillingServiceURL       string
	ChannelServiceURL       string
	SellerServiceURL        string
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

// PostJSON performs an HTTP POST request with JSON body to a named service and returns the response
func (c *ServiceClients) PostJSON(ctx context.Context, serviceName string, path string, body interface{}) (interface{}, error) {
	baseURL := c.getServiceURL(serviceName)
	if baseURL == "" {
		return nil, fmt.Errorf("unknown service: %s", serviceName)
	}
	url := baseURL + path
	var result map[string]interface{}
	if err := c.doRequest(ctx, http.MethodPost, url, body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// getServiceURL returns the base URL for a named service
func (c *ServiceClients) getServiceURL(serviceName string) string {
	switch serviceName {
	case "billing":
		return c.config.BillingServiceURL
	case "channel":
		return c.config.ChannelServiceURL
	case "order":
		return c.config.OrderServiceURL
	case "inventory":
		return c.config.InventoryServiceURL
	case "seller":
		return c.config.SellerServiceURL
	default:
		return ""
	}
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

// StartPicking marks an order as picking in progress
func (c *ServiceClients) StartPicking(ctx context.Context, orderID string) error {
	url := fmt.Sprintf("%s/api/v1/orders/%s/start-picking", c.config.OrderServiceURL, orderID)
	return c.doRequest(ctx, http.MethodPut, url, nil, nil)
}

// MarkConsolidated marks an order as consolidated
func (c *ServiceClients) MarkConsolidated(ctx context.Context, orderID string) error {
	url := fmt.Sprintf("%s/api/v1/orders/%s/mark-consolidated", c.config.OrderServiceURL, orderID)
	return c.doRequest(ctx, http.MethodPut, url, nil, nil)
}

// MarkPacked marks an order as packed
func (c *ServiceClients) MarkPacked(ctx context.Context, orderID string) error {
	url := fmt.Sprintf("%s/api/v1/orders/%s/mark-packed", c.config.OrderServiceURL, orderID)
	return c.doRequest(ctx, http.MethodPut, url, nil, nil)
}

// AssignToWave assigns an order to a wave (updates order status to wave_assigned)
func (c *ServiceClients) AssignToWave(ctx context.Context, orderID, waveID string) error {
	url := fmt.Sprintf("%s/api/v1/orders/%s/assign-wave", c.config.OrderServiceURL, orderID)
	body := map[string]string{"waveId": waveID}
	return c.doRequest(ctx, http.MethodPut, url, body, nil)
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

// PickInventory decrements inventory for picked items
func (c *ServiceClients) PickInventory(ctx context.Context, sku string, req *PickInventoryRequest) error {
	url := fmt.Sprintf("%s/api/v1/inventory/%s/pick", c.config.InventoryServiceURL, sku)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// StageInventory converts soft reservation to hard allocation (physical staging)
func (c *ServiceClients) StageInventory(ctx context.Context, sku string, req *StageInventoryRequest) error {
	url := fmt.Sprintf("%s/api/v1/inventory/%s/stage", c.config.InventoryServiceURL, sku)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// PackInventory marks a hard allocation as packed
func (c *ServiceClients) PackInventory(ctx context.Context, sku string, req *PackInventoryRequest) error {
	url := fmt.Sprintf("%s/api/v1/inventory/%s/pack", c.config.InventoryServiceURL, sku)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// ShipInventory ships a packed allocation (removes inventory from system)
func (c *ServiceClients) ShipInventory(ctx context.Context, sku string, req *ShipInventoryRequest) error {
	url := fmt.Sprintf("%s/api/v1/inventory/%s/ship", c.config.InventoryServiceURL, sku)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// ReturnInventoryToShelf returns hard allocated inventory back to shelf
func (c *ServiceClients) ReturnInventoryToShelf(ctx context.Context, sku string, req *ReturnToShelfRequest) error {
	url := fmt.Sprintf("%s/api/v1/inventory/%s/return-to-shelf", c.config.InventoryServiceURL, sku)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// RecordStockShortage records a confirmed stock shortage discovered during picking
func (c *ServiceClients) RecordStockShortage(ctx context.Context, sku string, req *RecordShortageRequest) error {
	url := fmt.Sprintf("%s/api/v1/inventory/%s/shortage", c.config.InventoryServiceURL, sku)
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
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

// CalculateMultiRoute calculates multiple routes for an order (zone and capacity splitting)
func (c *ServiceClients) CalculateMultiRoute(ctx context.Context, req *CalculateRouteRequest) (*MultiRouteResult, error) {
	url := fmt.Sprintf("%s/api/v1/routes/calculate-multi", c.config.RoutingServiceURL)
	var result MultiRouteResult
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
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

// StartPackTask marks a pack task as started (sets startedAt timestamp)
func (c *ServiceClients) StartPackTask(ctx context.Context, taskID string) (*PackTask, error) {
	url := fmt.Sprintf("%s/api/v1/tasks/%s/start", c.config.PackingServiceURL, taskID)
	var result PackTask
	if err := c.doRequest(ctx, http.MethodPost, url, nil, &result); err != nil {
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

// Facility Service methods (for process path routing and station management)

// FindCapableStations finds stations with all required capabilities
func (c *ServiceClients) FindCapableStations(ctx context.Context, req *FindCapableStationsRequest) ([]Station, error) {
	url := fmt.Sprintf("%s/api/v1/stations/find-capable", c.config.FacilityServiceURL)
	var result []Station
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetStation retrieves a station by ID
func (c *ServiceClients) GetStation(ctx context.Context, stationID string) (*Station, error) {
	url := fmt.Sprintf("%s/api/v1/stations/%s", c.config.FacilityServiceURL, stationID)
	var result Station
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetStationsByZone retrieves stations in a zone
func (c *ServiceClients) GetStationsByZone(ctx context.Context, zone string) ([]Station, error) {
	url := fmt.Sprintf("%s/api/v1/stations/zone/%s", c.config.FacilityServiceURL, zone)
	var result []Station
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetStationsByType retrieves stations by type
func (c *ServiceClients) GetStationsByType(ctx context.Context, stationType string) ([]Station, error) {
	url := fmt.Sprintf("%s/api/v1/stations/type/%s", c.config.FacilityServiceURL, stationType)
	var result []Station
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Unit Service methods

// CreateUnitsRequest represents a request to create units at receiving
type CreateUnitsRequest struct {
	SKU        string `json:"sku"`
	ShipmentID string `json:"shipmentId"`
	LocationID string `json:"locationId"`
	Quantity   int    `json:"quantity"`
	CreatedBy  string `json:"createdBy"`
}

// CreateUnitsResponse represents the result of creating units
type CreateUnitsResponse struct {
	UnitIDs []string `json:"unitIds"`
	SKU     string   `json:"sku"`
	Count   int      `json:"count"`
}

// CreateUnits generates UUIDs for units at receiving
func (c *ServiceClients) CreateUnits(ctx context.Context, sku, shipmentID, locationID string, quantity int, createdBy string) (*CreateUnitsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/units", c.config.UnitServiceURL)
	req := CreateUnitsRequest{
		SKU:        sku,
		ShipmentID: shipmentID,
		LocationID: locationID,
		Quantity:   quantity,
		CreatedBy:  createdBy,
	}
	var result CreateUnitsResponse
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReserveUnitsRequest represents a request to reserve units for an order
type ReserveUnitsRequest struct {
	OrderID   string            `json:"orderId"`
	PathID    string            `json:"pathId"`
	Items     []ReserveUnitItem `json:"items"`
	HandlerID string            `json:"handlerId"`
}

// ReserveUnitItem specifies SKU and quantity to reserve
type ReserveUnitItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// ReservedUnit holds info about a reserved unit
type ReservedUnit struct {
	UnitID     string `json:"unitId"`
	SKU        string `json:"sku"`
	LocationID string `json:"locationId"`
}

// FailedReserveItem holds info about a failed reservation
type FailedReserveItem struct {
	SKU       string `json:"sku"`
	Requested int    `json:"requested"`
	Available int    `json:"available"`
	Reason    string `json:"reason"`
}

// ReserveUnitsResponse represents the result of reserving units
type ReserveUnitsResponse struct {
	ReservedUnits []ReservedUnit      `json:"reservedUnits"`
	FailedItems   []FailedReserveItem `json:"failedItems,omitempty"`
}

// ReserveUnits reserves specific units for an order with a path
func (c *ServiceClients) ReserveUnits(ctx context.Context, orderID, pathID string, items interface{}, handlerID string) (*ReserveUnitsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/units/reserve", c.config.UnitServiceURL)
	req := map[string]interface{}{
		"orderId":   orderID,
		"pathId":    pathID,
		"items":     items,
		"handlerId": handlerID,
	}
	var result ReserveUnitsResponse
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UnitForOrder holds information about a unit for an order
type UnitForOrder struct {
	UnitID     string `json:"unitId"`
	SKU        string `json:"sku"`
	Status     string `json:"status"`
	LocationID string `json:"locationId"`
}

// GetUnitsForOrder retrieves all units reserved for an order
func (c *ServiceClients) GetUnitsForOrder(ctx context.Context, orderID string) ([]UnitForOrder, error) {
	url := fmt.Sprintf("%s/api/v1/units/order/%s", c.config.UnitServiceURL, orderID)
	var result []UnitForOrder
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ConfirmUnitPick confirms that a specific unit has been picked
func (c *ServiceClients) ConfirmUnitPick(ctx context.Context, unitID, toteID, pickerID, stationID string) error {
	url := fmt.Sprintf("%s/api/v1/units/%s/pick", c.config.UnitServiceURL, unitID)
	req := map[string]string{
		"toteId":    toteID,
		"pickerId":  pickerID,
		"stationId": stationID,
	}
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// ConfirmUnitConsolidation confirms that a specific unit has been consolidated
func (c *ServiceClients) ConfirmUnitConsolidation(ctx context.Context, unitID, destinationBin, workerID, stationID string) error {
	url := fmt.Sprintf("%s/api/v1/units/%s/consolidate", c.config.UnitServiceURL, unitID)
	req := map[string]string{
		"destinationBin": destinationBin,
		"workerId":       workerID,
		"stationId":      stationID,
	}
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// ConfirmUnitPacked confirms that a specific unit has been packed
func (c *ServiceClients) ConfirmUnitPacked(ctx context.Context, unitID, packageID, packerID, stationID string) error {
	url := fmt.Sprintf("%s/api/v1/units/%s/pack", c.config.UnitServiceURL, unitID)
	req := map[string]string{
		"packageId": packageID,
		"packerId":  packerID,
		"stationId": stationID,
	}
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// ConfirmUnitShipped confirms that a specific unit has been shipped
func (c *ServiceClients) ConfirmUnitShipped(ctx context.Context, unitID, shipmentID, trackingNumber, handlerID string) error {
	url := fmt.Sprintf("%s/api/v1/units/%s/ship", c.config.UnitServiceURL, unitID)
	req := map[string]string{
		"shipmentId":     shipmentID,
		"trackingNumber": trackingNumber,
		"handlerId":      handlerID,
	}
	return c.doRequest(ctx, http.MethodPost, url, req, nil)
}

// UnitExceptionResult holds the result of creating an exception
type UnitExceptionResult struct {
	ExceptionID string `json:"exceptionId"`
	UnitID      string `json:"unitId"`
}

// CreateUnitException creates an exception for a failed unit
func (c *ServiceClients) CreateUnitException(ctx context.Context, unitID, exceptionType, stage, description, stationID, reportedBy string) (*UnitExceptionResult, error) {
	url := fmt.Sprintf("%s/api/v1/units/%s/exception", c.config.UnitServiceURL, unitID)
	req := map[string]string{
		"exceptionType": exceptionType,
		"stage":         stage,
		"description":   description,
		"stationId":     stationID,
		"reportedBy":    reportedBy,
	}
	var result UnitExceptionResult
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UnitMovement holds information about a unit movement
type UnitMovement struct {
	MovementID     string `json:"movementId"`
	FromLocationID string `json:"fromLocationId"`
	ToLocationID   string `json:"toLocationId"`
	FromStatus     string `json:"fromStatus"`
	ToStatus       string `json:"toStatus"`
	StationID      string `json:"stationId"`
	HandlerID      string `json:"handlerId"`
	Timestamp      string `json:"timestamp"`
	Notes          string `json:"notes"`
}

// GetUnitAuditTrail retrieves the full movement history for a unit
func (c *ServiceClients) GetUnitAuditTrail(ctx context.Context, unitID string) ([]UnitMovement, error) {
	url := fmt.Sprintf("%s/api/v1/units/%s/audit", c.config.UnitServiceURL, unitID)
	var result []UnitMovement
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PersistProcessPathResult holds the result of persisting a process path
type PersistProcessPathResult struct {
	PathID  string `json:"pathId"`
	OrderID string `json:"orderId"`
}

// PersistProcessPath saves the process path to ensure all units follow the same path
func (c *ServiceClients) PersistProcessPath(ctx context.Context, orderID string, items []ProcessPathItem, giftWrap bool, totalValue float64) (*PersistProcessPathResult, error) {
	url := fmt.Sprintf("%s/api/v1/process-paths/determine", c.config.ProcessPathServiceURL)
	req := map[string]interface{}{
		"orderId":    orderID,
		"items":      items,
		"giftWrap":   giftWrap,
		"totalValue": totalValue,
	}
	var result PersistProcessPathResult
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ProcessPathInfo holds information about a persisted process path
type ProcessPathInfo struct {
	PathID                string   `json:"pathId"`
	OrderID               string   `json:"orderId"`
	Requirements          []string `json:"requirements"`
	ConsolidationRequired bool     `json:"consolidationRequired"`
	GiftWrapRequired      bool     `json:"giftWrapRequired"`
	SpecialHandling       []string `json:"specialHandling"`
	TargetStationID       string   `json:"targetStationId,omitempty"`
}

// GetProcessPath retrieves a persisted process path by ID
func (c *ServiceClients) GetProcessPath(ctx context.Context, pathID string) (*ProcessPathInfo, error) {
	url := fmt.Sprintf("%s/api/v1/process-paths/%s", c.config.ProcessPathServiceURL, pathID)
	var result ProcessPathInfo
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Process Path Service methods

// DetermineProcessPathRequest represents a request to determine process path
type DetermineProcessPathRequest struct {
	OrderID          string            `json:"orderId"`
	Items            []ProcessPathItem `json:"items"`
	GiftWrap         bool              `json:"giftWrap"`
	GiftWrapDetails  *GiftWrapDetails  `json:"giftWrapDetails,omitempty"`
	HazmatDetails    *HazmatDetails    `json:"hazmatDetails,omitempty"`
	ColdChainDetails *ColdChainDetails `json:"coldChainDetails,omitempty"`
	TotalValue       float64           `json:"totalValue"`
}

// ProcessPathItem represents an item for process path determination
type ProcessPathItem struct {
	SKU               string  `json:"sku"`
	Quantity          int     `json:"quantity"`
	Weight            float64 `json:"weight"`
	IsFragile         bool    `json:"isFragile"`
	IsHazmat          bool    `json:"isHazmat"`
	RequiresColdChain bool    `json:"requiresColdChain"`
}

// DetermineProcessPathViaService calls the process-path-service to determine the process path
func (c *ServiceClients) DetermineProcessPathViaService(ctx context.Context, req *DetermineProcessPathRequest) (*ProcessPath, error) {
	url := fmt.Sprintf("%s/api/v1/process-paths/determine", c.config.ProcessPathServiceURL)
	var result ProcessPath
	if err := c.doRequest(ctx, http.MethodPost, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetProcessPathFromService retrieves a process path from process-path-service by pathId
func (c *ServiceClients) GetProcessPathFromService(ctx context.Context, pathID string) (*ProcessPath, error) {
	url := fmt.Sprintf("%s/api/v1/process-paths/%s", c.config.ProcessPathServiceURL, pathID)
	var result ProcessPath
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetProcessPathByOrderID retrieves a process path by order ID from process-path-service
func (c *ServiceClients) GetProcessPathByOrderID(ctx context.Context, orderID string) (*ProcessPath, error) {
	url := fmt.Sprintf("%s/api/v1/process-paths/order/%s", c.config.ProcessPathServiceURL, orderID)
	var result ProcessPath
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AssignStationToProcessPath assigns a station to a process path in process-path-service
func (c *ServiceClients) AssignStationToProcessPath(ctx context.Context, pathID, stationID string) (*ProcessPath, error) {
	url := fmt.Sprintf("%s/api/v1/process-paths/%s/station", c.config.ProcessPathServiceURL, pathID)
	req := map[string]string{"stationId": stationID}
	var result ProcessPath
	if err := c.doRequest(ctx, http.MethodPut, url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
