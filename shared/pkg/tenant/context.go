package tenant

import (
	"context"
	"errors"
)

// Context keys for tenant information
type contextKey string

const (
	tenantIDKey    contextKey = "tenantId"
	facilityIDKey  contextKey = "facilityId"
	warehouseIDKey contextKey = "warehouseId"
	sellerIDKey    contextKey = "sellerId"
	channelIDKey   contextKey = "channelId"
)

// Errors for tenant context operations
var (
	ErrMissingTenantContext = errors.New("tenant context is required")
	ErrUnauthorizedAccess   = errors.New("unauthorized access to tenant resource")
	ErrMissingTenantID      = errors.New("tenantId is required")
	ErrMissingFacilityID    = errors.New("facilityId is required")
	ErrMissingWarehouseID   = errors.New("warehouseId is required")
	ErrMissingSellerID      = errors.New("sellerId is required for this operation")
)

// Context holds all tenant-related identifiers for multi-tenant operations.
// This struct is used to scope all database queries and operations to a specific tenant.
type Context struct {
	// TenantID is the 3PL operator identifier (the company running the warehouse)
	TenantID string `json:"tenantId"`

	// FacilityID is the physical facility/warehouse complex identifier
	FacilityID string `json:"facilityId"`

	// WarehouseID is a specific warehouse within a facility
	WarehouseID string `json:"warehouseId"`

	// SellerID is the merchant/seller using 3PL services
	SellerID string `json:"sellerId"`

	// ChannelID is the sales channel (Shopify, Amazon, eBay, etc.)
	ChannelID string `json:"channelId"`
}

// FromContext extracts TenantContext from context.Context.
// Returns an error if required tenant fields are missing.
func FromContext(ctx context.Context) (*Context, error) {
	tc := &Context{}

	if v := ctx.Value(tenantIDKey); v != nil {
		if id, ok := v.(string); ok {
			tc.TenantID = id
		}
	}
	if v := ctx.Value(facilityIDKey); v != nil {
		if id, ok := v.(string); ok {
			tc.FacilityID = id
		}
	}
	if v := ctx.Value(warehouseIDKey); v != nil {
		if id, ok := v.(string); ok {
			tc.WarehouseID = id
		}
	}
	if v := ctx.Value(sellerIDKey); v != nil {
		if id, ok := v.(string); ok {
			tc.SellerID = id
		}
	}
	if v := ctx.Value(channelIDKey); v != nil {
		if id, ok := v.(string); ok {
			tc.ChannelID = id
		}
	}

	// At minimum, we need either TenantID or FacilityID for scoping
	if tc.TenantID == "" && tc.FacilityID == "" {
		return nil, ErrMissingTenantContext
	}

	return tc, nil
}

// FromContextOptional extracts TenantContext from context.Context.
// Unlike FromContext, this returns an empty context if none exists (for backward compatibility).
func FromContextOptional(ctx context.Context) *Context {
	tc, _ := FromContext(ctx)
	if tc == nil {
		return &Context{}
	}
	return tc
}

// ToContext adds TenantContext values to context.Context.
func ToContext(ctx context.Context, tc *Context) context.Context {
	if tc == nil {
		return ctx
	}

	if tc.TenantID != "" {
		ctx = context.WithValue(ctx, tenantIDKey, tc.TenantID)
	}
	if tc.FacilityID != "" {
		ctx = context.WithValue(ctx, facilityIDKey, tc.FacilityID)
	}
	if tc.WarehouseID != "" {
		ctx = context.WithValue(ctx, warehouseIDKey, tc.WarehouseID)
	}
	if tc.SellerID != "" {
		ctx = context.WithValue(ctx, sellerIDKey, tc.SellerID)
	}
	if tc.ChannelID != "" {
		ctx = context.WithValue(ctx, channelIDKey, tc.ChannelID)
	}

	return ctx
}

// WithTenantID returns a new context with the tenant ID set
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// WithFacilityID returns a new context with the facility ID set
func WithFacilityID(ctx context.Context, facilityID string) context.Context {
	return context.WithValue(ctx, facilityIDKey, facilityID)
}

// WithWarehouseID returns a new context with the warehouse ID set
func WithWarehouseID(ctx context.Context, warehouseID string) context.Context {
	return context.WithValue(ctx, warehouseIDKey, warehouseID)
}

// WithSellerID returns a new context with the seller ID set
func WithSellerID(ctx context.Context, sellerID string) context.Context {
	return context.WithValue(ctx, sellerIDKey, sellerID)
}

// WithChannelID returns a new context with the channel ID set
func WithChannelID(ctx context.Context, channelID string) context.Context {
	return context.WithValue(ctx, channelIDKey, channelID)
}

// GetTenantID extracts tenant ID from context
func GetTenantID(ctx context.Context) string {
	if v := ctx.Value(tenantIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// GetFacilityID extracts facility ID from context
func GetFacilityID(ctx context.Context) string {
	if v := ctx.Value(facilityIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// GetWarehouseID extracts warehouse ID from context
func GetWarehouseID(ctx context.Context) string {
	if v := ctx.Value(warehouseIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// GetSellerID extracts seller ID from context
func GetSellerID(ctx context.Context) string {
	if v := ctx.Value(sellerIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// GetChannelID extracts channel ID from context
func GetChannelID(ctx context.Context) string {
	if v := ctx.Value(channelIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// IsEmpty returns true if the context has no tenant identifiers set
func (tc *Context) IsEmpty() bool {
	return tc.TenantID == "" && tc.FacilityID == "" && tc.WarehouseID == "" && tc.SellerID == ""
}

// HasSeller returns true if a seller ID is set
func (tc *Context) HasSeller() bool {
	return tc.SellerID != ""
}

// HasFacility returns true if a facility ID is set
func (tc *Context) HasFacility() bool {
	return tc.FacilityID != ""
}

// HasWarehouse returns true if a warehouse ID is set
func (tc *Context) HasWarehouse() bool {
	return tc.WarehouseID != ""
}

// HasTenant returns true if a tenant ID is set
func (tc *Context) HasTenant() bool {
	return tc.TenantID != ""
}

// Validate checks that all required tenant context fields are present.
// Required fields are: TenantID, FacilityID, WarehouseID
// Returns an error if any required field is missing.
func (tc *Context) Validate() error {
	if tc.TenantID == "" {
		return ErrMissingTenantID
	}
	if tc.FacilityID == "" {
		return ErrMissingFacilityID
	}
	if tc.WarehouseID == "" {
		return ErrMissingWarehouseID
	}
	return nil
}

// ValidateWithSeller validates required fields including seller ID.
// Use this for operations that require seller context.
func (tc *Context) ValidateWithSeller() error {
	if err := tc.Validate(); err != nil {
		return err
	}
	if tc.SellerID == "" {
		return ErrMissingSellerID
	}
	return nil
}

// ValidateOwnership verifies that a resource belongs to this tenant context.
// Used to prevent cross-tenant data access.
func (tc *Context) ValidateOwnership(resourceTenantID, resourceFacilityID, resourceSellerID string) error {
	// Validate tenant ID if present in context
	if tc.TenantID != "" && resourceTenantID != "" && tc.TenantID != resourceTenantID {
		return ErrUnauthorizedAccess
	}

	// Validate facility ID if present in context
	if tc.FacilityID != "" && resourceFacilityID != "" && tc.FacilityID != resourceFacilityID {
		return ErrUnauthorizedAccess
	}

	// Validate seller ID if present in context
	if tc.SellerID != "" && resourceSellerID != "" && tc.SellerID != resourceSellerID {
		return ErrUnauthorizedAccess
	}

	return nil
}

// DefaultContext returns a default tenant context for backward compatibility.
// Used during migration period for existing data without tenant fields.
const (
	DefaultTenantID    = "DEFAULT_TENANT"
	DefaultFacilityID  = "DEFAULT_FACILITY"
	DefaultWarehouseID = "DEFAULT_WAREHOUSE"
)

// Default returns a default tenant context for backward compatibility
func Default() *Context {
	return &Context{
		TenantID:    DefaultTenantID,
		FacilityID:  DefaultFacilityID,
		WarehouseID: DefaultWarehouseID,
	}
}
