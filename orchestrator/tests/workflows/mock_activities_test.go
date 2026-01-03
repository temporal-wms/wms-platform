package workflows_test

import "github.com/wms-platform/orchestrator/internal/workflows"

// Mock activity functions shared across workflow tests

// ValidateOrder is a mock activity for order validation
func ValidateOrder(input workflows.OrderFulfillmentInput) (bool, error) {
	return true, nil
}

// CalculateRoute is a mock activity for route calculation
func CalculateRoute(params map[string]interface{}) (workflows.RouteResult, error) {
	return workflows.RouteResult{}, nil
}

// CalculateMultiRoute is a mock activity for multi-route calculation
func CalculateMultiRoute(params map[string]interface{}) (workflows.MultiRouteResult, error) {
	return workflows.MultiRouteResult{
		OrderID:     "ORD-001",
		TotalRoutes: 1,
		Routes:      []workflows.RouteResult{{RouteID: "ROUTE-001"}},
	}, nil
}

// StartPicking is a mock activity for starting picking
func StartPicking(orderID string) error {
	return nil
}

// MarkConsolidated is a mock activity for marking order as consolidated
func MarkConsolidated(orderID string) error {
	return nil
}

// MarkPacked is a mock activity for marking order as packed
func MarkPacked(orderID string) error {
	return nil
}

// FindCapableStation is a mock activity for finding capable station
func FindCapableStation(input map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"stationId": "STATION-001"}, nil
}

// ExecuteSLAM is a mock activity for SLAM process
func ExecuteSLAM(input map[string]interface{}) (workflows.SLAMResult, error) {
	return workflows.SLAMResult{
		TaskID:         "SLAM-001",
		TrackingNumber: "TRACK-123",
		ManifestID:     "MANIFEST-001",
		Success:        true,
		CarrierID:      "CARRIER-001",
		Destination:    "12345",
	}, nil
}

// CancelOrder is a mock activity for order cancellation
func CancelOrder(orderID, reason string) error {
	return nil
}

// ReleaseInventoryReservation is a mock activity for releasing inventory
func ReleaseInventoryReservation(orderID string) error {
	return nil
}

// NotifyCustomerCancellation is a mock activity for customer notification
func NotifyCustomerCancellation(orderID, reason string) error {
	return nil
}

// DetermineProcessPath is a mock activity for process path determination
func DetermineProcessPath(input map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"pathId":                "PATH-001",
		"requirements":          []string{"single_item"},
		"consolidationRequired": false,
		"giftWrapRequired":      false,
		"specialHandling":       []string{},
	}, nil
}

// PersistProcessPath is a mock activity for persisting process path
func PersistProcessPath(input map[string]interface{}) (map[string]string, error) {
	return map[string]string{"pathId": "PATH-001"}, nil
}

// ReserveUnits is a mock activity for reserving units
func ReserveUnits(input map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"reservedUnits": []interface{}{
			map[string]interface{}{"unitId": "UNIT-001"},
			map[string]interface{}{"unitId": "UNIT-002"},
		},
		"failedItems": []interface{}{},
	}, nil
}

// AssignToWave is a mock activity for assigning order to wave
func AssignToWave(orderID, waveID string) error {
	return nil
}
