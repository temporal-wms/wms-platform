package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/routing-service/internal/domain"
)

// RouteCalculator implements route calculation logic
type RouteCalculator struct {
	routeRepo        domain.RouteRepository
	warehouseLayout  domain.WarehouseLayout
	inventoryLocator domain.InventoryLocator
}

// NewRouteCalculator creates a new RouteCalculator
func NewRouteCalculator(
	routeRepo domain.RouteRepository,
	warehouseLayout domain.WarehouseLayout,
	inventoryLocator domain.InventoryLocator,
) *RouteCalculator {
	return &RouteCalculator{
		routeRepo:        routeRepo,
		warehouseLayout:  warehouseLayout,
		inventoryLocator: inventoryLocator,
	}
}

// CalculateRoutes calculates one or more optimized routes for given items
// Splits by zone and capacity (max 30 items per route)
func (c *RouteCalculator) CalculateRoutes(ctx context.Context, request domain.RouteRequest) (*domain.MultiRouteResult, error) {
	if len(request.Items) == 0 {
		return nil, fmt.Errorf("no items provided for route calculation")
	}

	result := &domain.MultiRouteResult{
		OrderID:       request.OrderID,
		Routes:        make([]*domain.PickRoute, 0),
		ZoneBreakdown: make(map[string]int),
		TotalItems:    0,
		CreatedAt:     time.Now(),
	}

	// Count total items
	for _, item := range request.Items {
		result.TotalItems += item.Quantity
	}

	// Step 1: Group items by zone (based on aisle prefix)
	zoneGroups := c.groupItemsByZone(request.Items)

	// Determine split reason
	splitByZone := len(zoneGroups) > 1
	splitByCapacity := false

	// Check if any zone group exceeds capacity
	for _, group := range zoneGroups {
		if len(group.Items) > domain.MaxItemsPerRoute {
			splitByCapacity = true
			break
		}
	}

	// Set split reason
	if splitByZone && splitByCapacity {
		result.SplitReason = domain.SplitReasonBoth
	} else if splitByZone {
		result.SplitReason = domain.SplitReasonZone
	} else if splitByCapacity {
		result.SplitReason = domain.SplitReasonCapacity
	} else {
		result.SplitReason = domain.SplitReasonNone
	}

	// Step 2: Split each zone group by capacity and create routes
	var allRouteItems [][]domain.RouteItem

	for _, group := range zoneGroups {
		result.ZoneBreakdown[group.Zone] = len(group.Items)

		// Split by capacity if needed
		chunks := c.splitByCapacity(group.Items, domain.MaxItemsPerRoute)
		allRouteItems = append(allRouteItems, chunks...)
	}

	// Step 3: Create individual routes
	totalRoutes := len(allRouteItems)
	result.TotalRoutes = totalRoutes

	for i, items := range allRouteItems {
		routeID := generateMultiRouteID(request.OrderID, i)

		// Determine strategy
		strategy := request.Strategy
		if strategy == "" {
			suggestedStrategy, err := c.SuggestStrategy(ctx, items)
			if err != nil {
				strategy = domain.StrategySShape
			} else {
				strategy = suggestedStrategy
			}
		}

		// Create multi-route aware pick route
		route, err := domain.NewMultiRoutePickRoute(routeID, request.OrderID, request.WaveID, strategy, items, i, totalRoutes)
		if err != nil {
			return nil, fmt.Errorf("failed to create route %d: %w", i, err)
		}

		// Set zone from first item in route
		if len(items) > 0 {
			route.Zone = domain.GetZoneForAisle(items[0].Location.Aisle)
		}

		// Assign unique tote ID for this route
		route.SourceToteID = fmt.Sprintf("TOTE-%s-%d", request.OrderID[len(request.OrderID)-8:], i)

		// Get start/end locations
		startLoc := request.StartLocation
		if startLoc.LocationID == "" && c.warehouseLayout != nil {
			startLoc = c.warehouseLayout.GetPickStartLocation(ctx, route.Zone)
		}

		endLoc := request.EndLocation
		if endLoc.LocationID == "" && c.warehouseLayout != nil {
			endLoc = c.warehouseLayout.GetConsolidationLocation(ctx, route.Zone)
		}

		// Optimize route
		if err := route.OptimizeRoute(startLoc, endLoc); err != nil {
			return nil, fmt.Errorf("failed to optimize route %d: %w", i, err)
		}

		result.Routes = append(result.Routes, route)
	}

	return result, nil
}

// groupItemsByZone groups items by their warehouse zone based on aisle prefix
func (c *RouteCalculator) groupItemsByZone(items []domain.RouteItem) []domain.ZoneGroup {
	zoneMap := make(map[string][]domain.RouteItem)

	for _, item := range items {
		zone := domain.GetZoneForAisle(item.Location.Aisle)
		zoneMap[zone] = append(zoneMap[zone], item)
	}

	// Convert map to slice and sort by zone name for deterministic ordering
	groups := make([]domain.ZoneGroup, 0, len(zoneMap))
	for zone, zoneItems := range zoneMap {
		groups = append(groups, domain.ZoneGroup{
			Zone:  zone,
			Items: zoneItems,
		})
	}

	// Sort by zone name
	for i := 0; i < len(groups)-1; i++ {
		for j := i + 1; j < len(groups); j++ {
			if groups[j].Zone < groups[i].Zone {
				groups[i], groups[j] = groups[j], groups[i]
			}
		}
	}

	return groups
}

// splitByCapacity splits items into chunks of maxSize
func (c *RouteCalculator) splitByCapacity(items []domain.RouteItem, maxSize int) [][]domain.RouteItem {
	if len(items) <= maxSize {
		return [][]domain.RouteItem{items}
	}

	var chunks [][]domain.RouteItem
	for i := 0; i < len(items); i += maxSize {
		end := i + maxSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}

	return chunks
}

// generateMultiRouteID generates a unique route ID for multi-route orders
func generateMultiRouteID(orderID string, routeIndex int) string {
	return fmt.Sprintf("RT-%s-%d-%d", orderID, routeIndex, time.Now().UnixNano()%100000)
}

// CalculateRoute calculates an optimized route for given items
func (c *RouteCalculator) CalculateRoute(ctx context.Context, request domain.RouteRequest) (*domain.PickRoute, error) {
	// Generate route ID
	routeID := generateRouteID(request.OrderID)

	// Determine strategy if not specified
	strategy := request.Strategy
	if strategy == "" {
		suggestedStrategy, err := c.SuggestStrategy(ctx, request.Items)
		if err != nil {
			strategy = domain.StrategySShape // Default fallback
		} else {
			strategy = suggestedStrategy
		}
	}

	// Create route
	route, err := domain.NewPickRoute(routeID, request.OrderID, request.WaveID, strategy, request.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to create route: %w", err)
	}

	// Set start and end locations
	startLoc := request.StartLocation
	if startLoc.LocationID == "" && c.warehouseLayout != nil {
		startLoc = c.warehouseLayout.GetPickStartLocation(ctx, request.Zone)
	}

	endLoc := request.EndLocation
	if endLoc.LocationID == "" && c.warehouseLayout != nil {
		endLoc = c.warehouseLayout.GetConsolidationLocation(ctx, request.Zone)
	}

	// Optimize route
	if err := route.OptimizeRoute(startLoc, endLoc); err != nil {
		return nil, fmt.Errorf("failed to optimize route: %w", err)
	}

	return route, nil
}

// RecalculateRoute recalculates an existing route
func (c *RouteCalculator) RecalculateRoute(ctx context.Context, route *domain.PickRoute) (*domain.PickRoute, error) {
	// Get remaining pending stops
	var remainingItems []domain.RouteItem
	for _, stop := range route.Stops {
		if stop.Status == "pending" {
			remainingItems = append(remainingItems, domain.RouteItem{
				SKU:      stop.SKU,
				Quantity: stop.Quantity,
				Location: stop.Location,
			})
		}
	}

	if len(remainingItems) == 0 {
		return route, nil // No recalculation needed
	}

	// Create new route with remaining items
	newRouteID := route.RouteID + "-R"
	newRoute, err := domain.NewPickRoute(newRouteID, route.OrderID, route.WaveID, route.Strategy, remainingItems)
	if err != nil {
		return nil, err
	}

	// Get current picker location (last completed stop or start)
	currentLoc := route.StartLocation
	for _, stop := range route.Stops {
		if stop.Status == "completed" {
			currentLoc = stop.Location
		}
	}

	// Optimize from current position
	if err := newRoute.OptimizeRoute(currentLoc, route.EndLocation); err != nil {
		return nil, err
	}

	return newRoute, nil
}

// SuggestStrategy suggests the best routing strategy for given items
func (c *RouteCalculator) SuggestStrategy(ctx context.Context, items []domain.RouteItem) (domain.RoutingStrategy, error) {
	if len(items) == 0 {
		return domain.StrategySShape, nil
	}

	// Analyze item distribution
	aisles := make(map[string]int)
	for _, item := range items {
		aisles[item.Location.Aisle]++
	}

	numAisles := len(aisles)
	itemsPerAisle := float64(len(items)) / float64(numAisles)

	// Decision logic based on analysis
	switch {
	case len(items) <= 3:
		// Few items - use nearest neighbor
		return domain.StrategyNearest, nil

	case numAisles == 1:
		// Single aisle - use return strategy
		return domain.StrategyReturn, nil

	case itemsPerAisle >= 5:
		// High density - use S-shape
		return domain.StrategySShape, nil

	case itemsPerAisle >= 2:
		// Medium density - use combined
		return domain.StrategyCombined, nil

	default:
		// Low density - use largest gap
		return domain.StrategyLargestGap, nil
	}
}

// BatchCalculateRoutes calculates routes for multiple orders in a wave
func (c *RouteCalculator) BatchCalculateRoutes(ctx context.Context, requests []domain.RouteRequest) ([]*domain.PickRoute, error) {
	routes := make([]*domain.PickRoute, 0, len(requests))

	for _, request := range requests {
		route, err := c.CalculateRoute(ctx, request)
		if err != nil {
			// Log error but continue with other routes
			fmt.Printf("Failed to calculate route for order %s: %v\n", request.OrderID, err)
			continue
		}
		routes = append(routes, route)
	}

	return routes, nil
}

// OptimizeWaveRoutes optimizes routes across a wave for zone efficiency
func (c *RouteCalculator) OptimizeWaveRoutes(ctx context.Context, routes []*domain.PickRoute) ([]*domain.PickRoute, error) {
	if len(routes) <= 1 {
		return routes, nil
	}

	// Group routes by zone
	zoneRoutes := make(map[string][]*domain.PickRoute)
	for _, route := range routes {
		zoneRoutes[route.Zone] = append(zoneRoutes[route.Zone], route)
	}

	// Optimize within each zone
	optimized := make([]*domain.PickRoute, 0, len(routes))
	for _, zoneGroup := range zoneRoutes {
		// Sort by estimated time (shortest first)
		sortRoutesByTime(zoneGroup)
		optimized = append(optimized, zoneGroup...)
	}

	return optimized, nil
}

// AnalyzeRouteEfficiency analyzes route efficiency metrics
func (c *RouteCalculator) AnalyzeRouteEfficiency(ctx context.Context, route *domain.PickRoute) (*RouteAnalysis, error) {
	analysis := &RouteAnalysis{
		RouteID:           route.RouteID,
		Strategy:          route.Strategy,
		TotalStops:        len(route.Stops),
		TotalItems:        route.TotalItems,
		EstimatedDistance: route.EstimatedDistance,
		EstimatedTime:     route.EstimatedTime,
	}

	// Calculate aisle distribution
	aisles := make(map[string]int)
	for _, stop := range route.Stops {
		aisles[stop.Location.Aisle]++
	}
	analysis.AisleCount = len(aisles)
	analysis.AisleDistribution = aisles

	// Calculate efficiency score (lower distance per item is better)
	if route.TotalItems > 0 {
		analysis.DistancePerItem = route.EstimatedDistance / float64(route.TotalItems)
	}

	// Estimate efficiency compared to naive approach
	naiveDistance := estimateNaiveDistance(route.Stops)
	if naiveDistance > 0 {
		analysis.EfficiencyGain = (naiveDistance - route.EstimatedDistance) / naiveDistance * 100
	}

	return analysis, nil
}

// RouteAnalysis contains route efficiency metrics
type RouteAnalysis struct {
	RouteID           string                 `json:"routeId"`
	Strategy          domain.RoutingStrategy `json:"strategy"`
	TotalStops        int                    `json:"totalStops"`
	TotalItems        int                    `json:"totalItems"`
	AisleCount        int                    `json:"aisleCount"`
	AisleDistribution map[string]int         `json:"aisleDistribution"`
	EstimatedDistance float64                `json:"estimatedDistance"`
	EstimatedTime     time.Duration          `json:"estimatedTime"`
	DistancePerItem   float64                `json:"distancePerItem"`
	EfficiencyGain    float64                `json:"efficiencyGain"` // percentage
}

// Helper functions

func generateRouteID(orderID string) string {
	return fmt.Sprintf("RT-%s-%d", orderID, time.Now().UnixNano()%100000)
}

func sortRoutesByTime(routes []*domain.PickRoute) {
	for i := 0; i < len(routes)-1; i++ {
		for j := i + 1; j < len(routes); j++ {
			if routes[j].EstimatedTime < routes[i].EstimatedTime {
				routes[i], routes[j] = routes[j], routes[i]
			}
		}
	}
}

func estimateNaiveDistance(stops []domain.RouteStop) float64 {
	if len(stops) <= 1 {
		return 0
	}

	// Naive approach: visit stops in original order
	totalDistance := 0.0
	for i := 0; i < len(stops)-1; i++ {
		dx := stops[i+1].Location.X - stops[i].Location.X
		dy := stops[i+1].Location.Y - stops[i].Location.Y
		totalDistance += (dx*dx + dy*dy)
	}

	return totalDistance
}
