package application

import (
	"context"

	"github.com/wms-platform/services/order-service/internal/infrastructure/projections"
	"github.com/wms-platform/shared/pkg/logging"
)

// OrderQueryService handles read-only queries using the CQRS read model
// This is separated from OrderApplicationService (write side)
type OrderQueryService struct {
	projectionRepo projections.OrderListProjectionRepository
	logger         *logging.Logger
}

// NewOrderQueryService creates a new query service
func NewOrderQueryService(
	projectionRepo projections.OrderListProjectionRepository,
	logger *logging.Logger,
) *OrderQueryService {
	return &OrderQueryService{
		projectionRepo: projectionRepo,
		logger:         logger,
	}
}

// ListOrders queries orders using the read model (fast, denormalized)
func (s *OrderQueryService) ListOrders(ctx context.Context, query ListOrdersQuery) (*PagedOrdersResult, error) {
	// Convert query to projection filter
	filter := projections.OrderListFilter{
		Status:         query.Status,
		Priority:       query.Priority,
		WaveID:         query.WaveID,
		CustomerID:     query.CustomerID,
		AssignedPicker: query.AssignedPicker,
		ShipToState:    query.ShipToState,
		ShipToCountry:  query.ShipToCountry,
		IsLate:         query.IsLate,
		IsPriority:     query.IsPriority,
		SearchTerm:     query.SearchTerm,
	}

	// Pagination
	page := projections.Pagination{
		Limit:     query.Limit,
		Offset:    query.Offset,
		SortBy:    query.SortBy,
		SortOrder: query.SortOrder,
	}

	// Set defaults
	if page.Limit == 0 {
		page.Limit = 50
	}
	if page.SortBy == "" {
		page.SortBy = "receivedAt"
	}
	if page.SortOrder == "" {
		page.SortOrder = "desc"
	}

	// Query projections
	result, err := s.projectionRepo.FindWithFilter(ctx, filter, page)
	if err != nil {
		s.logger.Error("Failed to query order projections", "error", err)
		return nil, err
	}

	// Convert projections to DTOs
	dtos := make([]OrderListDTO, len(result.Items))
	for i, proj := range result.Items {
		dtos[i] = s.projectionToDTO(&proj)
	}

	// Calculate pagination metadata
	totalPages := (result.Total + int64(result.Limit) - 1) / int64(result.Limit)
	if totalPages < 1 {
		totalPages = 1
	}
	currentPage := (int64(result.Offset) / int64(result.Limit)) + 1

	return &PagedOrdersResult{
		Data:       dtos,
		Page:       currentPage,
		PageSize:   int64(result.Limit),
		TotalItems: result.Total,
		TotalPages: totalPages,
	}, nil
}

// GetOrderSummary retrieves a single order summary (from read model)
func (s *OrderQueryService) GetOrderSummary(ctx context.Context, orderID string) (*OrderListDTO, error) {
	projection, err := s.projectionRepo.FindByID(ctx, orderID)
	if err != nil {
		s.logger.Error("Failed to get order projection", "orderId", orderID, "error", err)
		return nil, err
	}

	if projection == nil {
		return nil, nil
	}

	dto := s.projectionToDTO(projection)
	return &dto, nil
}

// GetLateOrders retrieves orders that are past their promised delivery date
func (s *OrderQueryService) GetLateOrders(ctx context.Context, limit, offset int) (*PagedOrdersResult, error) {
	isLate := true
	query := ListOrdersQuery{
		IsLate:    &isLate,
		Limit:     limit,
		Offset:    offset,
		SortBy:    "promisedDeliveryAt",
		SortOrder: "asc",
	}

	return s.ListOrders(ctx, query)
}

// GetPriorityOrders retrieves priority orders (same_day, next_day)
func (s *OrderQueryService) GetPriorityOrders(ctx context.Context, limit, offset int) (*PagedOrdersResult, error) {
	isPriority := true
	query := ListOrdersQuery{
		IsPriority: &isPriority,
		Limit:      limit,
		Offset:     offset,
		SortBy:     "receivedAt",
		SortOrder:  "asc",
	}

	return s.ListOrders(ctx, query)
}

// GetOrdersByWave retrieves all orders assigned to a specific wave
func (s *OrderQueryService) GetOrdersByWave(ctx context.Context, waveID string, limit, offset int) (*PagedOrdersResult, error) {
	query := ListOrdersQuery{
		WaveID:    &waveID,
		Limit:     limit,
		Offset:    offset,
		SortBy:    "receivedAt",
		SortOrder: "asc",
	}

	return s.ListOrders(ctx, query)
}

// GetOrdersByPicker retrieves all orders assigned to a specific picker
func (s *OrderQueryService) GetOrdersByPicker(ctx context.Context, pickerID string, limit, offset int) (*PagedOrdersResult, error) {
	query := ListOrdersQuery{
		AssignedPicker: &pickerID,
		Limit:          limit,
		Offset:         offset,
		SortBy:         "pickingStartedAt",
		SortOrder:      "desc",
	}

	return s.ListOrders(ctx, query)
}

// CountOrdersByStatus counts orders in a specific status
func (s *OrderQueryService) CountOrdersByStatus(ctx context.Context, status string) (int64, error) {
	filter := projections.OrderListFilter{
		Status: &status,
	}

	count, err := s.projectionRepo.Count(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to count orders by status", "status", status, "error", err)
		return 0, err
	}

	return count, nil
}

// Helper: Convert projection to DTO
func (s *OrderQueryService) projectionToDTO(proj *projections.OrderListProjection) OrderListDTO {
	return OrderListDTO{
		OrderID:           proj.OrderID,
		CustomerID:        proj.CustomerID,
		CustomerName:      proj.CustomerName,
		Status:            proj.Status,
		Priority:          proj.Priority,
		TotalItems:        proj.TotalItems,
		TotalWeight:       proj.TotalWeight,
		TotalValue:        proj.TotalValue,
		WaveID:            proj.WaveID,
		WaveStatus:        proj.WaveStatus,
		WaveType:          proj.WaveType,
		AssignedPicker:    proj.AssignedPicker,
		TrackingNumber:    proj.TrackingNumber,
		Carrier:           proj.Carrier,
		ShipToCity:        proj.ShipToCity,
		ShipToState:       proj.ShipToState,
		ShipToZipCode:     proj.ShipToZipCode,
		DaysUntilPromised: proj.DaysUntilPromised,
		IsLate:            proj.IsLate,
		IsPriority:        proj.IsPriority,
		ReceivedAt:        proj.ReceivedAt.Format("2006-01-02T15:04:05Z"),
		PromisedDeliveryAt: proj.PromisedDeliveryAt.Format("2006-01-02T15:04:05Z"),
		CreatedAt:         proj.CreatedAt,
		UpdatedAt:         proj.UpdatedAt,
	}
}
