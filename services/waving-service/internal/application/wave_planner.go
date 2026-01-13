package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/wms-platform/waving-service/internal/domain"
)

// WavePlanner implements the wave planning algorithms
type WavePlanner struct {
	waveRepo     domain.WaveRepository
	orderService domain.OrderService
}

// NewWavePlanner creates a new WavePlanner
func NewWavePlanner(waveRepo domain.WaveRepository, orderService domain.OrderService) *WavePlanner {
	return &WavePlanner{
		waveRepo:     waveRepo,
		orderService: orderService,
	}
}

// PlanWave creates an optimized wave from available orders
func (p *WavePlanner) PlanWave(ctx context.Context, config domain.WavePlanningConfig) (*domain.Wave, error) {
	// Get orders ready for waving based on filter
	filter := domain.OrderFilter{
		Priority:                config.PriorityFilter,
		Zone:                    []string{config.Zone},
		Carrier:                 config.CarrierFilter,
		MaxItems:                config.MaxItems,
		CutoffBefore:            config.CutoffTime,
		Limit:                   config.MaxOrders * 2, // Get more than needed for optimization
		ProcessPathRequirements: config.RequiredProcessPaths,
		SpecialHandling:         config.SpecialHandlingFilter,
		ExcludeProcessPaths:     config.ExcludedProcessPaths,
	}

	orders, err := p.orderService.GetOrdersReadyForWaving(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders for waving: %w", err)
	}

	if len(orders) == 0 {
		return nil, fmt.Errorf("no orders available for waving")
	}

	// If grouping by process path, filter orders for compatibility
	if config.GroupByProcessPath {
		orders = filterOrdersByProcessPathCompatibility(orders, config.WaveType, config.RequiredProcessPaths)
		if len(orders) == 0 {
			return nil, fmt.Errorf("no compatible orders available after process path filtering")
		}
	}

	// Generate wave ID
	waveID := generateWaveID(config.WaveType)

	// Create wave configuration
	waveConfig := domain.WaveConfiguration{
		MaxOrders:           config.MaxOrders,
		MaxItems:            config.MaxItems,
		MaxWeight:           config.MaxWeight,
		CarrierFilter:       config.CarrierFilter,
		PriorityFilter:      config.PriorityFilter,
		ZoneFilter:          []string{config.Zone},
		CutoffTime:          config.CutoffTime,
		AutoRelease:         true,
		OptimizeForCarrier:  len(config.CarrierFilter) > 0,
		OptimizeForZone:     config.Zone != "",
		OptimizeForPriority: len(config.PriorityFilter) > 0,
	}

	// Create new wave
	wave, err := domain.NewWave(waveID, config.WaveType, config.FulfillmentMode, waveConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create wave: %w", err)
	}

	wave.SetZone(config.Zone)

	// Sort orders by priority and cutoff time
	sortedOrders := sortOrdersForWave(orders, config)

	// Add orders to wave respecting constraints
	totalItems := 0
	totalWeight := 0.0

	for _, order := range sortedOrders {
		// Check if adding this order would exceed limits
		if config.MaxOrders > 0 && wave.GetOrderCount() >= config.MaxOrders {
			break
		}
		if config.MaxItems > 0 && totalItems+order.ItemCount > config.MaxItems {
			continue // Skip this order, try next
		}
		if config.MaxWeight > 0 && totalWeight+order.TotalWeight > config.MaxWeight {
			continue // Skip this order, try next
		}

		if err := wave.AddOrder(order); err != nil {
			continue // Skip orders that can't be added
		}

		totalItems += order.ItemCount
		totalWeight += order.TotalWeight
	}

	if wave.GetOrderCount() == 0 {
		return nil, fmt.Errorf("could not add any orders to wave")
	}

	// Populate process path capabilities and requirements from orders
	populateWaveProcessPathCapabilities(wave)

	// Calculate labor requirements (now considers certifications)
	laborAllocation := calculateLaborRequirements(wave)
	wave.AllocateLabor(laborAllocation)

	// Calculate priority based on orders
	wavePriority := calculateWavePriority(wave)
	wave.SetPriority(wavePriority)

	return wave, nil
}

// OptimizeWave optimizes an existing wave
func (p *WavePlanner) OptimizeWave(ctx context.Context, wave *domain.Wave) (*domain.Wave, error) {
	if wave.Status != domain.WaveStatusPlanning && wave.Status != domain.WaveStatusScheduled {
		return nil, fmt.Errorf("can only optimize waves in planning or scheduled status")
	}

	// Sort orders within the wave for optimal picking
	optimizedOrders := optimizeOrderSequence(wave.Orders)

	// Replace orders with optimized sequence
	wave.Orders = optimizedOrders

	// Recalculate labor requirements
	laborAllocation := calculateLaborRequirements(wave)
	wave.AllocateLabor(laborAllocation)

	// Add optimization event
	wave.AddDomainEvent(&domain.WaveOptimizedEvent{
		WaveID:            wave.WaveID,
		OptimizationType:  "sequence",
		OrdersReorganized: len(wave.Orders),
		EstimatedSavings:  5.0, // Placeholder - would calculate actual savings
		OptimizedAt:       time.Now(),
	})

	return wave, nil
}

// SuggestOrders suggests orders to add to a wave
func (p *WavePlanner) SuggestOrders(ctx context.Context, wave *domain.Wave, limit int) ([]domain.WaveOrder, error) {
	// Build filter based on wave configuration
	filter := domain.OrderFilter{
		Priority:     wave.Configuration.PriorityFilter,
		Zone:         wave.Configuration.ZoneFilter,
		Carrier:      wave.Configuration.CarrierFilter,
		CutoffBefore: wave.Configuration.CutoffTime,
		Limit:        limit,
	}

	orders, err := p.orderService.GetOrdersReadyForWaving(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get order suggestions: %w", err)
	}

	// Filter out orders already in the wave
	existingOrderIDs := make(map[string]bool)
	for _, o := range wave.Orders {
		existingOrderIDs[o.OrderID] = true
	}

	var suggestions []domain.WaveOrder
	for _, order := range orders {
		if !existingOrderIDs[order.OrderID] {
			suggestions = append(suggestions, order)
		}
		if len(suggestions) >= limit {
			break
		}
	}

	return suggestions, nil
}

// generateWaveID generates a unique wave ID
func generateWaveID(waveType domain.WaveType) string {
	now := time.Now()
	prefix := "WV"
	switch waveType {
	case domain.WaveTypeDigital:
		prefix = "WV-DIG"
	case domain.WaveTypeWholesale:
		prefix = "WV-WHL"
	case domain.WaveTypePriority:
		prefix = "WV-PRI"
	case domain.WaveTypeMixed:
		prefix = "WV-MIX"
	}
	return fmt.Sprintf("%s-%s-%d", prefix, now.Format("20060102"), now.UnixNano()%100000)
}

// sortOrdersForWave sorts orders for optimal wave assignment
func sortOrdersForWave(orders []domain.WaveOrder, config domain.WavePlanningConfig) []domain.WaveOrder {
	sorted := make([]domain.WaveOrder, len(orders))
	copy(sorted, orders)

	sort.Slice(sorted, func(i, j int) bool {
		// Priority first (same_day > next_day > standard)
		priorityOrder := map[string]int{
			"same_day": 1,
			"next_day": 2,
			"standard": 3,
		}
		pi := priorityOrder[sorted[i].Priority]
		pj := priorityOrder[sorted[j].Priority]
		if pi != pj {
			return pi < pj
		}

		// Then by carrier cutoff time
		if !sorted[i].CarrierCutoff.Equal(sorted[j].CarrierCutoff) {
			return sorted[i].CarrierCutoff.Before(sorted[j].CarrierCutoff)
		}

		// Then by promised delivery
		if !sorted[i].PromisedDeliveryAt.Equal(sorted[j].PromisedDeliveryAt) {
			return sorted[i].PromisedDeliveryAt.Before(sorted[j].PromisedDeliveryAt)
		}

		// Finally by zone (group same zones together)
		return sorted[i].Zone < sorted[j].Zone
	})

	return sorted
}

// optimizeOrderSequence optimizes the sequence of orders for efficient picking
func optimizeOrderSequence(orders []domain.WaveOrder) []domain.WaveOrder {
	if len(orders) <= 1 {
		return orders
	}

	optimized := make([]domain.WaveOrder, len(orders))
	copy(optimized, orders)

	// Group by zone for zone-based picking
	sort.Slice(optimized, func(i, j int) bool {
		// Group by zone
		if optimized[i].Zone != optimized[j].Zone {
			return optimized[i].Zone < optimized[j].Zone
		}
		// Within zone, sort by item count (pick smaller orders first for faster completion)
		return optimized[i].ItemCount < optimized[j].ItemCount
	})

	return optimized
}

// calculateLaborRequirements estimates labor needed for a wave
func calculateLaborRequirements(wave *domain.Wave) domain.LaborAllocation {
	totalItems := wave.GetTotalItems()
	orderCount := wave.GetOrderCount()

	// Heuristics for labor estimation
	// Assume: 1 picker handles ~100 items/hour, 1 packer handles ~50 packages/hour
	itemsPerPicker := 100.0
	ordersPerPacker := 50.0

	pickersNeeded := int(float64(totalItems)/itemsPerPicker) + 1
	packersNeeded := int(float64(orderCount)/ordersPerPacker) + 1

	// Minimum 1 picker and 1 packer
	if pickersNeeded < 1 {
		pickersNeeded = 1
	}
	if packersNeeded < 1 {
		packersNeeded = 1
	}

	return domain.LaborAllocation{
		PickersRequired: pickersNeeded,
		PackersRequired: packersNeeded,
	}
}

// calculateWavePriority calculates wave priority based on contained orders
func calculateWavePriority(wave *domain.Wave) int {
	if len(wave.Orders) == 0 {
		return 5 // Default medium priority
	}

	hasSameDay := false
	hasNextDay := false

	for _, order := range wave.Orders {
		switch order.Priority {
		case "same_day":
			hasSameDay = true
		case "next_day":
			hasNextDay = true
		}
	}

	if hasSameDay {
		return 1 // Highest priority
	}
	if hasNextDay {
		return 2
	}
	return 3 // Standard priority
}

// Process Path Helper Functions

// filterOrdersByProcessPathCompatibility filters orders based on wave type and process path requirements
func filterOrdersByProcessPathCompatibility(orders []domain.WaveOrder, waveType domain.WaveType, requiredPaths []string) []domain.WaveOrder {
	var compatible []domain.WaveOrder

	for _, order := range orders {
		if isOrderCompatibleWithWaveType(order, waveType, requiredPaths) {
			compatible = append(compatible, order)
		}
	}

	return compatible
}

// isOrderCompatibleWithWaveType checks if an order is compatible with a specific wave type
func isOrderCompatibleWithWaveType(order domain.WaveOrder, waveType domain.WaveType, requiredPaths []string) bool {
	// If specific requirements are specified, check if order matches
	if len(requiredPaths) > 0 {
		for _, required := range requiredPaths {
			if !hasProcessPathRequirement(order.ProcessPathRequirements, required) {
				return false
			}
		}
		return true
	}

	// Check compatibility based on wave type
	switch waveType {
	case domain.WaveTypeHazmat:
		return hasProcessPathRequirement(order.ProcessPathRequirements, "hazmat")
	case domain.WaveTypeColdChain:
		return hasProcessPathRequirement(order.ProcessPathRequirements, "cold_chain")
	case domain.WaveTypeHighValue:
		return hasProcessPathRequirement(order.ProcessPathRequirements, "high_value")
	case domain.WaveTypeFragile:
		return hasProcessPathRequirement(order.ProcessPathRequirements, "fragile")
	case domain.WaveTypeStandard:
		// Standard waves should not have special handling requirements
		return !hasAnySpecialHandlingRequirements(order.ProcessPathRequirements)
	case domain.WaveTypeSpecialized, domain.WaveTypeMixed:
		// These wave types accept any orders
		return true
	default:
		// Digital, wholesale, priority waves accept all (filtered by business type elsewhere)
		return true
	}
}

// hasProcessPathRequirement checks if a requirement exists in the list
func hasProcessPathRequirement(requirements []string, requirement string) bool {
	for _, r := range requirements {
		if r == requirement {
			return true
		}
	}
	return false
}

// hasAnySpecialHandlingRequirements checks if order has any special handling requirements
func hasAnySpecialHandlingRequirements(requirements []string) bool {
	specialHandlingTypes := []string{"hazmat", "cold_chain", "high_value", "fragile", "oversized", "gift_wrap"}
	for _, req := range requirements {
		for _, special := range specialHandlingTypes {
			if req == special {
				return true
			}
		}
	}
	return false
}

// populateWaveProcessPathCapabilities populates wave capabilities and requirements from orders
func populateWaveProcessPathCapabilities(wave *domain.Wave) {
	requiresCertification := false
	capabilitiesMap := make(map[string]bool)
	handlingTypesMap := make(map[string]bool)
	stationTypesMap := make(map[string]bool)

	for _, order := range wave.Orders {
		// Track certification requirements
		if order.RequiresCertification {
			requiresCertification = true
		}

		// Collect all unique process path requirements
		for _, req := range order.ProcessPathRequirements {
			// Map requirements to capabilities
			switch req {
			case "hazmat":
				capabilitiesMap["hazmat_handling"] = true
				handlingTypesMap["hazmat_compliance"] = true
				stationTypesMap["hazmat_certified"] = true
			case "cold_chain":
				capabilitiesMap["temperature_control"] = true
				handlingTypesMap["cold_chain_packaging"] = true
				stationTypesMap["cold_storage"] = true
			case "high_value":
				capabilitiesMap["high_value_verification"] = true
				handlingTypesMap["high_value_verification"] = true
				stationTypesMap["secure_station"] = true
			case "fragile":
				capabilitiesMap["fragile_handling"] = true
				handlingTypesMap["fragile_packing"] = true
				stationTypesMap["packing_station"] = true
			case "oversized":
				capabilitiesMap["heavy_lifting"] = true
				handlingTypesMap["oversized_handling"] = true
				stationTypesMap["oversized_station"] = true
			case "gift_wrap":
				capabilitiesMap["gift_wrapping"] = true
				handlingTypesMap["gift_wrap"] = true
				stationTypesMap["gift_wrap_station"] = true
			}
		}

		// Collect special handling types
		for _, handling := range order.SpecialHandling {
			handlingTypesMap[handling] = true
		}
	}

	// Convert maps to slices
	for capability := range capabilitiesMap {
		wave.AddRequiredCapability(capability)
	}
	for handling := range handlingTypesMap {
		wave.AddSpecialHandlingType(handling)
	}
	for stationType := range stationTypesMap {
		wave.AddStationRequirement(stationType)
	}

	wave.SetRequiresCertifiedLabor(requiresCertification)
}
