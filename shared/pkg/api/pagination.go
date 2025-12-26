package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// PageRequest represents pagination request parameters
type PageRequest struct {
	Page     int64 `form:"page" binding:"min=1" json:"page"`
	PageSize int64 `form:"pageSize" binding:"min=1,max=100" json:"pageSize"`
}

// DefaultPageRequest returns a PageRequest with default values
func DefaultPageRequest() PageRequest {
	return PageRequest{
		Page:     1,
		PageSize: 20,
	}
}

// PageResponse represents a paginated response
type PageResponse[T any] struct {
	Data       []T   `json:"data"`
	Page       int64 `json:"page"`
	PageSize   int64 `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int64 `json:"totalPages"`
	HasNext    bool  `json:"hasNext"`
	HasPrev    bool  `json:"hasPrev"`
}

// NewPageResponse creates a new paginated response
func NewPageResponse[T any](data []T, page, pageSize, totalItems int64) PageResponse[T] {
	totalPages := (totalItems + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return PageResponse[T]{
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// ParsePagination parses pagination parameters from Gin context
func ParsePagination(c *gin.Context) PageRequest {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "20"), 10, 64)

	// Validate and adjust
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return PageRequest{
		Page:     page,
		PageSize: pageSize,
	}
}

// GetOffset calculates the offset for database queries
func (p PageRequest) GetOffset() int64 {
	return (p.Page - 1) * p.PageSize
}

// GetLimit returns the page size
func (p PageRequest) GetLimit() int64 {
	return p.PageSize
}

// SortOrder represents sort direction
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// SortRequest represents sorting parameters
type SortRequest struct {
	Field string    `form:"sortBy" json:"sortBy"`
	Order SortOrder `form:"order" json:"order"`
}

// DefaultSortRequest returns a SortRequest with default values
func DefaultSortRequest(defaultField string) SortRequest {
	return SortRequest{
		Field: defaultField,
		Order: SortDesc,
	}
}

// ParseSort parses sorting parameters from Gin context
func ParseSort(c *gin.Context, defaultField string) SortRequest {
	field := c.DefaultQuery("sortBy", defaultField)
	order := SortOrder(c.DefaultQuery("order", string(SortDesc)))

	if order != SortAsc && order != SortDesc {
		order = SortDesc
	}

	return SortRequest{
		Field: field,
		Order: order,
	}
}

// GetMongoSort returns MongoDB sort value (1 for asc, -1 for desc)
func (s SortRequest) GetMongoSort() int {
	if s.Order == SortAsc {
		return 1
	}
	return -1
}

// FilterRequest represents common filter parameters
type FilterRequest struct {
	Search    string            `form:"search" json:"search,omitempty"`
	Status    string            `form:"status" json:"status,omitempty"`
	DateFrom  string            `form:"dateFrom" json:"dateFrom,omitempty"`
	DateTo    string            `form:"dateTo" json:"dateTo,omitempty"`
	Filters   map[string]string `json:"filters,omitempty"`
}

// ParseFilter parses common filter parameters from Gin context
func ParseFilter(c *gin.Context) FilterRequest {
	return FilterRequest{
		Search:   c.Query("search"),
		Status:   c.Query("status"),
		DateFrom: c.Query("dateFrom"),
		DateTo:   c.Query("dateTo"),
	}
}

// ListRequest combines pagination, sorting, and filtering
type ListRequest struct {
	Pagination PageRequest
	Sort       SortRequest
	Filter     FilterRequest
}

// ParseListRequest parses all list parameters from Gin context
func ParseListRequest(c *gin.Context, defaultSortField string) ListRequest {
	return ListRequest{
		Pagination: ParsePagination(c),
		Sort:       ParseSort(c, defaultSortField),
		Filter:     ParseFilter(c),
	}
}
