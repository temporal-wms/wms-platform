package application

import "github.com/wms-platform/routing-service/internal/domain"

// ToPickRouteDTO converts a domain PickRoute to PickRouteDTO
func ToPickRouteDTO(route *domain.PickRoute) *PickRouteDTO {
	if route == nil {
		return nil
	}

	stops := make([]RouteStopDTO, 0, len(route.Stops))
	for _, stop := range route.Stops {
		stops = append(stops, ToRouteStopDTO(stop))
	}

	return &PickRouteDTO{
		RouteID:           route.RouteID,
		OrderID:           route.OrderID,
		WaveID:            route.WaveID,
		PickerID:          route.PickerID,
		Status:            string(route.Status),
		Strategy:          string(route.Strategy),
		Stops:             stops,
		EstimatedDistance: route.EstimatedDistance,
		ActualDistance:    route.ActualDistance,
		EstimatedTime:     int64(route.EstimatedTime.Seconds()),
		ActualTime:        int64(route.ActualTime.Seconds()),
		StartLocation:     ToLocationDTO(route.StartLocation),
		EndLocation:       ToLocationDTO(route.EndLocation),
		Zone:              route.Zone,
		TotalItems:        route.TotalItems,
		PickedItems:       route.PickedItems,
		CreatedAt:         route.CreatedAt,
		UpdatedAt:         route.UpdatedAt,
		StartedAt:         route.StartedAt,
		CompletedAt:       route.CompletedAt,
	}
}

// ToRouteStopDTO converts a domain RouteStop to RouteStopDTO
func ToRouteStopDTO(stop domain.RouteStop) RouteStopDTO {
	return RouteStopDTO{
		StopNumber: stop.StopNumber,
		Location:   ToLocationDTO(stop.Location),
		SKU:        stop.SKU,
		Quantity:   stop.Quantity,
		PickedQty:  stop.PickedQty,
		Status:     stop.Status,
		ToteID:     stop.ToteID,
		PickedAt:   stop.PickedAt,
		Notes:      stop.Notes,
	}
}

// ToLocationDTO converts a domain Location to LocationDTO
func ToLocationDTO(loc domain.Location) LocationDTO {
	return LocationDTO{
		LocationID: loc.LocationID,
		Aisle:      loc.Aisle,
		Rack:       loc.Rack,
		Level:      loc.Level,
		Position:   loc.Position,
		Zone:       loc.Zone,
		X:          loc.X,
		Y:          loc.Y,
	}
}

// ToPickRouteDTOs converts a slice of domain PickRoutes to PickRouteDTOs
func ToPickRouteDTOs(routes []*domain.PickRoute) []PickRouteDTO {
	dtos := make([]PickRouteDTO, 0, len(routes))
	for _, route := range routes {
		if dto := ToPickRouteDTO(route); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}

// ToMultiRouteResultDTO converts a domain MultiRouteResult to MultiRouteResultDTO
func ToMultiRouteResultDTO(result *domain.MultiRouteResult) *MultiRouteResultDTO {
	if result == nil {
		return nil
	}

	routes := make([]PickRouteDTO, 0, len(result.Routes))
	for _, route := range result.Routes {
		if dto := ToPickRouteDTO(route); dto != nil {
			routes = append(routes, *dto)
		}
	}

	return &MultiRouteResultDTO{
		OrderID:       result.OrderID,
		Routes:        routes,
		TotalRoutes:   result.TotalRoutes,
		SplitReason:   string(result.SplitReason),
		ZoneBreakdown: result.ZoneBreakdown,
		TotalItems:    result.TotalItems,
		CreatedAt:     result.CreatedAt,
	}
}
