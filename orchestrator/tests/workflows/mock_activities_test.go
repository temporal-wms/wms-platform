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
