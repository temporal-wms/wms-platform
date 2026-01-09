package application

import (
	"context"

	"github.com/wms-platform/inventory-service/internal/infrastructure/projections"
	"github.com/wms-platform/shared/pkg/logging"
)

// InventoryQueryService handles read-only queries using the CQRS read model
// This is separated from InventoryService (write side)
type InventoryQueryService struct {
	projectionRepo projections.InventoryListProjectionRepository
	logger         *logging.Logger
}

// NewInventoryQueryService creates a new query service
func NewInventoryQueryService(
	projectionRepo projections.InventoryListProjectionRepository,
	logger *logging.Logger,
) *InventoryQueryService {
	return &InventoryQueryService{
		projectionRepo: projectionRepo,
		logger:         logger,
	}
}

// PagedInventoryResult represents a paginated list of inventory items
type PagedInventoryResult struct {
	Data    []InventoryListDTO `json:"data"`
	Total   int64              `json:"total"`
	Limit   int                `json:"limit"`
	Offset  int                `json:"offset"`
	HasMore bool               `json:"hasMore"`
}

// ListInventory queries inventory using the read model (fast, denormalized)
func (s *InventoryQueryService) ListInventory(ctx context.Context, query ListInventoryQuery) (*PagedInventoryResult, error) {
	// Convert query to projection filter
	filter := projections.InventoryListFilter{
		SKU:             query.SKU,
		ProductName:     query.ProductName,
		SearchTerm:      query.SearchTerm,
		IsLowStock:      query.IsLowStock,
		IsOutOfStock:    query.IsOutOfStock,
		MinQuantity:     query.MinQuantity,
		MaxQuantity:     query.MaxQuantity,
		HasReservations: query.HasReservations,
		LocationID:      query.LocationID,
		Zone:            query.Zone,
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
		page.SortBy = "updatedAt"
	}
	if page.SortOrder == "" {
		page.SortOrder = "desc"
	}

	// Query projections
	result, err := s.projectionRepo.FindWithFilter(ctx, filter, page)
	if err != nil {
		s.logger.Error("Failed to query inventory projections", "error", err)
		return nil, err
	}

	// Convert projections to DTOs
	dtos := make([]InventoryListDTO, len(result.Items))
	for i, proj := range result.Items {
		dtos[i] = s.projectionToDTO(&proj)
	}

	return &PagedInventoryResult{
		Data:    dtos,
		Total:   result.Total,
		Limit:   result.Limit,
		Offset:  result.Offset,
		HasMore: result.HasMore,
	}, nil
}

// GetInventorySummary retrieves a single inventory summary (from read model)
func (s *InventoryQueryService) GetInventorySummary(ctx context.Context, sku string) (*InventoryListDTO, error) {
	projection, err := s.projectionRepo.FindBySKU(ctx, sku)
	if err != nil {
		s.logger.Error("Failed to get inventory projection", "sku", sku, "error", err)
		return nil, err
	}

	if projection == nil {
		return nil, nil
	}

	dto := s.projectionToDTO(projection)
	return &dto, nil
}

// GetLowStockItems retrieves items with low stock
func (s *InventoryQueryService) GetLowStockItems(ctx context.Context, limit, offset int) (*PagedInventoryResult, error) {
	isLowStock := true
	query := ListInventoryQuery{
		IsLowStock: &isLowStock,
		Limit:      limit,
		Offset:     offset,
		SortBy:     "availableQuantity",
		SortOrder:  "asc",
	}

	return s.ListInventory(ctx, query)
}

// GetOutOfStockItems retrieves items that are out of stock
func (s *InventoryQueryService) GetOutOfStockItems(ctx context.Context, limit, offset int) (*PagedInventoryResult, error) {
	isOutOfStock := true
	query := ListInventoryQuery{
		IsOutOfStock: &isOutOfStock,
		Limit:        limit,
		Offset:       offset,
		SortBy:       "updatedAt",
		SortOrder:    "desc",
	}

	return s.ListInventory(ctx, query)
}

// GetInventoryByLocation retrieves all inventory at a specific location
func (s *InventoryQueryService) GetInventoryByLocation(ctx context.Context, locationID string, limit, offset int) (*PagedInventoryResult, error) {
	query := ListInventoryQuery{
		LocationID: &locationID,
		Limit:      limit,
		Offset:     offset,
		SortBy:     "availableQuantity",
		SortOrder:  "desc",
	}

	return s.ListInventory(ctx, query)
}

// CountInventoryByStatus counts inventory items matching criteria
func (s *InventoryQueryService) CountInventoryByStatus(ctx context.Context, isLowStock, isOutOfStock *bool) (int64, error) {
	filter := projections.InventoryListFilter{
		IsLowStock:   isLowStock,
		IsOutOfStock: isOutOfStock,
	}

	count, err := s.projectionRepo.Count(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to count inventory", "error", err)
		return 0, err
	}

	return count, nil
}

// Helper: Convert projection to DTO
func (s *InventoryQueryService) projectionToDTO(proj *projections.InventoryListProjection) InventoryListDTO {
	return InventoryListDTO{
		SKU:                proj.SKU,
		ProductName:        proj.ProductName,
		TotalQuantity:      proj.TotalQuantity,
		ReservedQuantity:   proj.ReservedQuantity,
		AvailableQuantity:  proj.AvailableQuantity,
		ReorderPoint:       proj.ReorderPoint,
		ReorderQuantity:    proj.ReorderQuantity,
		IsLowStock:         proj.IsLowStock,
		IsOutOfStock:       proj.IsOutOfStock,
		LocationCount:      proj.LocationCount,
		PrimaryLocation:    proj.PrimaryLocation,
		AvailableLocations: proj.AvailableLocations,
		ActiveReservations: proj.ActiveReservations,
		ReservedOrders:     proj.ReservedOrders,
	}
}
