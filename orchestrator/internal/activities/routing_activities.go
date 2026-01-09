package activities

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"github.com/wms-platform/orchestrator/internal/workflows"
	"go.temporal.io/sdk/activity"
)

// CalculateRouteInput represents input for route calculation
type CalculateRouteInput struct {
	OrderID string           `json:"orderId"`
	WaveID  string           `json:"waveId"`
	Items   []workflows.Item `json:"items"`
}

// CalculateMultiRoute calculates multiple routes for an order (zone and capacity splitting)
func (a *RoutingActivities) CalculateMultiRoute(ctx context.Context, input map[string]interface{}) (*workflows.MultiRouteResult, error) {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	waveID, _ := input["waveId"].(string)
	itemsRaw, _ := input["items"].([]interface{})

	logger.Info("Calculating multi-route", "orderId", orderID, "waveId", waveID, "itemCount", len(itemsRaw))

	// Convert items to route request format
	items := make([]clients.RouteItemRequest, 0)
	for _, itemRaw := range itemsRaw {
		if item, ok := itemRaw.(map[string]interface{}); ok {
			sku, _ := item["sku"].(string)
			quantity, _ := item["quantity"].(float64)
			items = append(items, clients.RouteItemRequest{
				SKU:      sku,
				Quantity: int(quantity),
			})
		}
	}

	// Generate route ID prefix
	routeIDPrefix := "RT-" + uuid.New().String()[:8]

	// Call routing-service to calculate multi-route
	result, err := a.clients.CalculateMultiRoute(ctx, &clients.CalculateRouteRequest{
		RouteID:  routeIDPrefix,
		OrderID:  orderID,
		WaveID:   waveID,
		Items:    items,
		Strategy: "s_shape",
	})
	if err != nil {
		logger.Error("Failed to calculate multi-route", "orderId", orderID, "error", err)
		return nil, fmt.Errorf("multi-route calculation failed: %w", err)
	}

	// Convert to workflow result format
	routes := make([]workflows.RouteResult, len(result.Routes))
	for i, route := range result.Routes {
		stops := make([]workflows.RouteStop, len(route.Stops))
		for j, stop := range route.Stops {
			stops[j] = workflows.RouteStop{
				LocationID: stop.LocationID,
				SKU:        stop.SKU,
				Quantity:   stop.Quantity,
			}
		}
		routes[i] = workflows.RouteResult{
			RouteID:           route.RouteID,
			Stops:             stops,
			EstimatedDistance: route.EstimatedDistance,
			Strategy:          route.Strategy,
		}
	}

	multiResult := &workflows.MultiRouteResult{
		OrderID:       result.OrderID,
		Routes:        routes,
		TotalRoutes:   result.TotalRoutes,
		SplitReason:   result.SplitReason,
		ZoneBreakdown: result.ZoneBreakdown,
		TotalItems:    result.TotalItems,
	}

	logger.Info("Multi-route calculated successfully",
		"orderId", orderID,
		"totalRoutes", multiResult.TotalRoutes,
		"splitReason", multiResult.SplitReason,
	)

	return multiResult, nil
}

// CalculateRoute calculates the optimal pick route for an order
func (a *RoutingActivities) CalculateRoute(ctx context.Context, input map[string]interface{}) (*workflows.RouteResult, error) {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	waveID, _ := input["waveId"].(string)
	itemsRaw, _ := input["items"].([]interface{})

	logger.Info("Calculating route", "orderId", orderID, "waveId", waveID)

	// Convert items to route request format
	items := make([]clients.RouteItemRequest, 0)
	for _, itemRaw := range itemsRaw {
		if item, ok := itemRaw.(map[string]interface{}); ok {
			sku, _ := item["sku"].(string)
			quantity, _ := item["quantity"].(float64)
			items = append(items, clients.RouteItemRequest{
				SKU:      sku,
				Quantity: int(quantity),
			})
		}
	}

	// Generate route ID
	routeID := "RT-" + uuid.New().String()[:8]

	// Call routing-service to calculate route
	// Valid strategies: return, s_shape, largest_gap, combined, nearest
	route, err := a.clients.CalculateRoute(ctx, &clients.CalculateRouteRequest{
		RouteID:  routeID,
		OrderID:  orderID,
		WaveID:   waveID,
		Items:    items,
		Strategy: "s_shape",
	})
	if err != nil {
		logger.Error("Failed to calculate route", "orderId", orderID, "error", err)
		return nil, fmt.Errorf("route calculation failed: %w", err)
	}

	// Convert to workflow result format
	stops := make([]workflows.RouteStop, len(route.Stops))
	for i, stop := range route.Stops {
		stops[i] = workflows.RouteStop{
			LocationID: stop.LocationID,
			SKU:        stop.SKU,
			Quantity:   stop.Quantity,
		}
	}

	result := &workflows.RouteResult{
		RouteID:           route.RouteID,
		Stops:             stops,
		EstimatedDistance: route.EstimatedDistance,
		Strategy:          route.Strategy,
	}

	logger.Info("Route calculated successfully",
		"orderId", orderID,
		"routeId", result.RouteID,
		"stops", len(result.Stops),
	)

	return result, nil
}
