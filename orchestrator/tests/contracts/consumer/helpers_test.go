package consumer_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	pact "github.com/pact-foundation/pact-go/v2"
	"github.com/pact-foundation/pact-go/v2/matchers"
	"github.com/stretchr/testify/require"
)

const (
	consumerName = "orchestrator"
	pactDir      = "../../../../contracts/pacts"
)

// setupPact creates a new Pact mock provider for consumer tests.
func setupPact(t *testing.T, provider string) (*pact.MockServer, error) {
	mockServer, err := pact.NewHTTPMock(pact.MockHTTPProviderConfig{
		Consumer: consumerName,
		Provider: provider,
		PactDir:  pactDir,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mock server for %s: %w", provider, err)
	}

	t.Cleanup(func() {
		if err := mockServer.WritePact(); err != nil {
			t.Errorf("Failed to write pact: %v", err)
		}
	})

	return mockServer, nil
}

// ensurePactDir ensures the pact directory exists.
func ensurePactDir(t *testing.T) {
	absPath, err := filepath.Abs(pactDir)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		err = os.MkdirAll(absPath, 0755)
		require.NoError(t, err)
	}
}

// Common matchers for reuse

// UUIDMatcher returns a matcher for UUID strings.
func UUIDMatcher() matchers.Matcher {
	return matchers.Regex("550e8400-e29b-41d4-a716-446655440000", `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
}

// TimestampMatcher returns a matcher for ISO8601 timestamps.
func TimestampMatcher() matchers.Matcher {
	return matchers.Regex("2024-01-15T10:30:00Z", `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)
}

// StatusMatcher returns a matcher for status strings.
func StatusMatcher(example string) matchers.Matcher {
	return matchers.String(example)
}

// IntMatcher returns a matcher for integer values.
func IntMatcher(example int) matchers.Matcher {
	return matchers.Integer(example)
}

// FloatMatcher returns a matcher for float values.
func FloatMatcher(example float64) matchers.Matcher {
	return matchers.Decimal(example)
}

// StringMatcher returns a matcher for string values.
func StringMatcher(example string) matchers.Matcher {
	return matchers.String(example)
}

// BoolMatcher returns a matcher for boolean values.
func BoolMatcher(example bool) matchers.Matcher {
	return matchers.Boolean(example)
}

// ArrayMatcher returns a matcher for arrays with at least one element like the example.
func ArrayMatcher(example interface{}) matchers.Matcher {
	return matchers.EachLike(example, 1)
}

// toJSON converts an object to JSON string.
func toJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// Test data structures for common request/response types

// OrderResponse represents a standard order response.
type OrderResponse struct {
	ID         string  `json:"id"`
	CustomerID string  `json:"customerId"`
	Status     string  `json:"status"`
	Priority   string  `json:"priority"`
	TotalItems int     `json:"totalItems"`
	TotalValue float64 `json:"totalValue"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
}

// ValidationResponse represents an order validation response.
type ValidationResponse struct {
	Valid   bool     `json:"valid"`
	OrderID string   `json:"orderId"`
	Errors  []string `json:"errors,omitempty"`
}

// InventoryItemResponse represents an inventory item response.
type InventoryItemResponse struct {
	SKU              string `json:"sku"`
	AvailableQty     int    `json:"availableQty"`
	ReservedQty      int    `json:"reservedQty"`
	Location         string `json:"location"`
	Zone             string `json:"zone"`
	LastUpdated      string `json:"lastUpdated"`
}

// RouteResponse represents a route calculation response.
type RouteResponse struct {
	ID            string   `json:"id"`
	Stops         []string `json:"stops"`
	TotalDistance float64  `json:"totalDistance"`
	EstimatedTime int      `json:"estimatedTime"`
	Status        string   `json:"status"`
}

// PickTaskResponse represents a pick task response.
type PickTaskResponse struct {
	ID        string `json:"id"`
	OrderID   string `json:"orderId"`
	WaveID    string `json:"waveId"`
	WorkerID  string `json:"workerId,omitempty"`
	Status    string `json:"status"`
	Priority  int    `json:"priority"`
	ItemCount int    `json:"itemCount"`
	CreatedAt string `json:"createdAt"`
}

// ConsolidationResponse represents a consolidation unit response.
type ConsolidationResponse struct {
	ID            string `json:"id"`
	OrderID       string `json:"orderId"`
	Status        string `json:"status"`
	ExpectedItems int    `json:"expectedItems"`
	ScannedItems  int    `json:"scannedItems"`
	Station       string `json:"station"`
}

// PackTaskResponse represents a pack task response.
type PackTaskResponse struct {
	ID          string  `json:"id"`
	OrderID     string  `json:"orderId"`
	Status      string  `json:"status"`
	PackageType string  `json:"packageType,omitempty"`
	Weight      float64 `json:"weight,omitempty"`
	HasLabel    bool    `json:"hasLabel"`
}

// ShipmentResponse represents a shipment response.
type ShipmentResponse struct {
	ID           string `json:"id"`
	OrderID      string `json:"orderId"`
	PackageID    string `json:"packageId"`
	Carrier      string `json:"carrier"`
	TrackingCode string `json:"trackingCode,omitempty"`
	Status       string `json:"status"`
}

// ShippingLabelResponse represents a shipping label response.
type ShippingLabelResponse struct {
	ID           string `json:"id"`
	ShipmentID   string `json:"shipmentId"`
	Carrier      string `json:"carrier"`
	TrackingCode string `json:"trackingCode"`
	LabelURL     string `json:"labelUrl"`
	Format       string `json:"format"`
}

// WorkerResponse represents a worker response.
type WorkerResponse struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Zone     string   `json:"zone"`
	Skills   []string `json:"skills"`
	TaskType string   `json:"taskType"`
}

// LaborTaskResponse represents a labor task assignment response.
type LaborTaskResponse struct {
	ID       string `json:"id"`
	WorkerID string `json:"workerId"`
	TaskType string `json:"taskType"`
	TaskRef  string `json:"taskRef"`
	Status   string `json:"status"`
}

// WaveResponse represents a wave response.
type WaveResponse struct {
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	OrderIDs   []string `json:"orderIds"`
	OrderCount int      `json:"orderCount"`
	Priority   int      `json:"priority"`
	CreatedAt  string   `json:"createdAt"`
}
