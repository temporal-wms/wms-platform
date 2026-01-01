package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/wms-platform/shared/pkg/temporal"
	"github.com/wms-platform/waving-service/internal/domain"
)

// OrderDTO represents order data fetched from order-service
type OrderDTO struct {
	OrderID            string    `json:"orderId"`
	CustomerID         string    `json:"customerId"`
	Priority           string    `json:"priority"`
	Status             string    `json:"status"`
	TotalItems         int       `json:"totalItems"`
	TotalWeight        float64   `json:"totalWeight"`
	PromisedDeliveryAt time.Time `json:"promisedDeliveryAt"`
	ShipToCity         string    `json:"shipToCity"`
	ShipToState        string    `json:"shipToState"`
}

// PagedOrdersResponse represents paginated orders response from order-service
type PagedOrdersResponse struct {
	Data       []OrderDTO `json:"data"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	TotalItems int        `json:"totalItems"`
	TotalPages int        `json:"totalPages"`
}

// OrderServiceClient handles communication with order-service
// Implements domain.OrderService interface
type OrderServiceClient struct {
	baseURL        string
	httpClient     *http.Client
	temporalClient *temporal.Client
}

// NewOrderServiceClient creates a new OrderServiceClient
func NewOrderServiceClient(baseURL string, temporalClient *temporal.Client) *OrderServiceClient {
	return &OrderServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		temporalClient: temporalClient,
	}
}

// GetOrder fetches a single order by ID from order-service
func (c *OrderServiceClient) GetOrder(ctx context.Context, orderID string) (*OrderDTO, error) {
	url := fmt.Sprintf("%s/api/v1/orders/%s", c.baseURL, orderID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("order service returned status %d", resp.StatusCode)
	}

	var order OrderDTO
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}

	return &order, nil
}

// GetOrdersReadyForWaving fetches validated orders ready for waving
// Implements domain.OrderService interface
func (c *OrderServiceClient) GetOrdersReadyForWaving(ctx context.Context, filter domain.OrderFilter) ([]domain.WaveOrder, error) {
	url := fmt.Sprintf("%s/api/v1/orders/status/validated", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	if filter.Limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", filter.Limit))
	} else {
		q.Add("limit", "100")
	}
	q.Add("sortBy", "promisedDeliveryAt")
	q.Add("sortOrder", "asc")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch validated orders: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("order service returned status %d", resp.StatusCode)
	}

	var pagedResponse PagedOrdersResponse
	if err := json.NewDecoder(resp.Body).Decode(&pagedResponse); err != nil {
		return nil, fmt.Errorf("failed to decode orders response: %w", err)
	}

	// Convert OrderDTO to domain.WaveOrder
	waveOrders := make([]domain.WaveOrder, 0, len(pagedResponse.Data))
	for _, order := range pagedResponse.Data {
		// Apply priority filter if specified
		if len(filter.Priority) > 0 {
			matched := false
			for _, p := range filter.Priority {
				if order.Priority == p {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		waveOrder := domain.WaveOrder{
			OrderID:            order.OrderID,
			CustomerID:         order.CustomerID,
			Priority:           order.Priority,
			ItemCount:          order.TotalItems,
			TotalWeight:        order.TotalWeight,
			PromisedDeliveryAt: order.PromisedDeliveryAt,
			CarrierCutoff:      order.PromisedDeliveryAt.Add(-4 * time.Hour), // Default 4 hours before delivery
			Zone:               "", // Could be derived from ship-to location
			Status:             "pending",
			AddedAt:            time.Now(),
		}
		waveOrders = append(waveOrders, waveOrder)
	}

	return waveOrders, nil
}

// NotifyWaveAssignment signals the order's Temporal workflow about wave assignment
// Implements domain.OrderService interface
func (c *OrderServiceClient) NotifyWaveAssignment(ctx context.Context, orderID, waveID string, scheduledStart time.Time) error {
	if c.temporalClient == nil {
		return fmt.Errorf("temporal client not configured")
	}

	workflowID := fmt.Sprintf("order-fulfillment-%s", orderID)

	signal := struct {
		WaveID         string    `json:"waveId"`
		ScheduledStart time.Time `json:"scheduledStart"`
	}{
		WaveID:         waveID,
		ScheduledStart: scheduledStart,
	}

	err := c.temporalClient.SignalWorkflow(ctx, workflowID, "", "waveAssigned", signal)
	if err != nil {
		return fmt.Errorf("failed to signal workflow %s: %w", workflowID, err)
	}

	return nil
}
