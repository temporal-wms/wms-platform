package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/waving-service/internal/domain"
)

func TestOrderServiceClient_GetOrder(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		orderID     string
		wantErr     bool
		errContains string
	}{
		{
			name: "Successfully get order",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, "application/json", r.Header.Get("Accept"))
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{
						"orderId": "ORD-001",
						"customerId": "CUST-001",
						"priority": "same_day",
						"status": "validated",
						"totalItems": 5,
						"totalWeight": 10.5,
						"promisedDeliveryAt": "2024-12-31T23:59:59Z",
						"shipToCity": "New York",
						"shipToState": "NY"
					}`))
				}))
			},
			orderID: "ORD-001",
			wantErr: false,
		},
		{
			name: "Order not found",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			orderID:     "ORD-999",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "Service returns error status",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			orderID:     "ORD-001",
			wantErr:     true,
			errContains: "returned status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			client := NewOrderServiceClient(server.URL, nil)
			order, err := client.GetOrder(context.Background(), tt.orderID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, order)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, order)
				assert.Equal(t, tt.orderID, order.OrderID)
				assert.Equal(t, "CUST-001", order.CustomerID)
				assert.Equal(t, "same_day", order.Priority)
			}
		})
	}
}

func TestOrderServiceClient_GetOrdersReadyForWaving(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		filter      domain.OrderFilter
		wantErr     bool
		wantCount   int
	}{
		{
			name: "Successfully get orders",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, "100", r.URL.Query().Get("limit"))
					assert.Equal(t, "promisedDeliveryAt", r.URL.Query().Get("sortBy"))
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{
						"data": [
							{
								"orderId": "ORD-001",
								"customerId": "CUST-001",
								"priority": "same_day",
								"status": "validated",
								"totalItems": 5,
								"totalWeight": 10.5,
								"promisedDeliveryAt": "2024-12-31T23:59:59Z"
							},
							{
								"orderId": "ORD-002",
								"customerId": "CUST-002",
								"priority": "next_day",
								"status": "validated",
								"totalItems": 3,
								"totalWeight": 7.2,
								"promisedDeliveryAt": "2025-01-01T23:59:59Z"
							}
						],
						"page": 1,
						"pageSize": 100,
						"totalItems": 2,
						"totalPages": 1
					}`))
				}))
			},
			filter:    domain.OrderFilter{Limit: 100},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "Successfully get orders with priority filter",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{
						"data": [
							{
								"orderId": "ORD-001",
								"customerId": "CUST-001",
								"priority": "same_day",
								"status": "validated",
								"totalItems": 5,
								"totalWeight": 10.5,
								"promisedDeliveryAt": "2024-12-31T23:59:59Z"
							}
						],
						"page": 1,
						"pageSize": 100,
						"totalItems": 1,
						"totalPages": 1
					}`))
				}))
			},
			filter:    domain.OrderFilter{Limit: 100, Priority: []string{"same_day"}},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "Service returns error status",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			filter:    domain.OrderFilter{},
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			client := NewOrderServiceClient(server.URL, nil)
			orders, err := client.GetOrdersReadyForWaving(context.Background(), tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, orders)
			} else {
				require.NoError(t, err)
				require.NotNil(t, orders)
				assert.Equal(t, tt.wantCount, len(orders))
			}
		})
	}
}

func TestOrderServiceClient_NotifyWaveAssignment(t *testing.T) {
	tests := []struct {
		name           string
		orderID        string
		waveID         string
		scheduledStart time.Time
		wantErr        bool
		errContains    string
	}{
		{
			name:           "Successfully notify wave assignment",
			orderID:        "ORD-001",
			waveID:         "WAVE-001",
			scheduledStart: time.Now(),
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOrderServiceClient("http://localhost:8081", nil)

			err := client.NotifyWaveAssignment(context.Background(), tt.orderID, tt.waveID, tt.scheduledStart)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "temporal client not configured")
			}
		})
	}
}
