package tenant

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

// RepositoryHelper provides tenant-aware query building for MongoDB repositories.
// Embed this in your repository structs to add tenant filtering capabilities.
type RepositoryHelper struct {
	// EnforceTenant when true, returns an error if tenant context is missing
	EnforceTenant bool
}

// NewRepositoryHelper creates a new RepositoryHelper
func NewRepositoryHelper(enforceTenant bool) *RepositoryHelper {
	return &RepositoryHelper{
		EnforceTenant: enforceTenant,
	}
}

// WithTenantFilter adds tenant filtering to a MongoDB query filter.
// It extracts tenant context from the context and adds appropriate filter conditions.
func (h *RepositoryHelper) WithTenantFilter(ctx context.Context, filter bson.M) (bson.M, error) {
	tc, err := FromContext(ctx)
	if err != nil {
		if h.EnforceTenant {
			return nil, err
		}
		// Return original filter if tenant context is not required
		return filter, nil
	}

	// Create a new filter to avoid modifying the original
	tenantFilter := bson.M{}
	for k, v := range filter {
		tenantFilter[k] = v
	}

	// Add tenant filters based on what's present in context
	if tc.TenantID != "" {
		tenantFilter["tenantId"] = tc.TenantID
	}
	if tc.FacilityID != "" {
		tenantFilter["facilityId"] = tc.FacilityID
	}
	if tc.WarehouseID != "" {
		tenantFilter["warehouseId"] = tc.WarehouseID
	}
	if tc.SellerID != "" {
		tenantFilter["sellerId"] = tc.SellerID
	}

	return tenantFilter, nil
}

// WithTenantFilterOptional adds tenant filtering without requiring tenant context.
// Uses default tenant values if context is missing.
func (h *RepositoryHelper) WithTenantFilterOptional(ctx context.Context, filter bson.M) bson.M {
	tc := FromContextOptional(ctx)

	// Create a new filter to avoid modifying the original
	tenantFilter := bson.M{}
	for k, v := range filter {
		tenantFilter[k] = v
	}

	// Add tenant filters based on what's present in context
	if tc.TenantID != "" {
		tenantFilter["tenantId"] = tc.TenantID
	}
	if tc.FacilityID != "" {
		tenantFilter["facilityId"] = tc.FacilityID
	}
	if tc.WarehouseID != "" {
		tenantFilter["warehouseId"] = tc.WarehouseID
	}
	if tc.SellerID != "" {
		tenantFilter["sellerId"] = tc.SellerID
	}

	return tenantFilter
}

// ValidateOwnership verifies that a resource belongs to the tenant in context.
// Use this after fetching a resource to ensure the caller has access.
func (h *RepositoryHelper) ValidateOwnership(ctx context.Context, resourceTenantID, resourceFacilityID, resourceSellerID string) error {
	tc, err := FromContext(ctx)
	if err != nil {
		if h.EnforceTenant {
			return err
		}
		return nil
	}

	return tc.ValidateOwnership(resourceTenantID, resourceFacilityID, resourceSellerID)
}

// ExtractTenantFields extracts tenant fields from context for setting on new entities.
// Returns default values if context is missing.
func (h *RepositoryHelper) ExtractTenantFields(ctx context.Context) (tenantID, facilityID, warehouseID, sellerID string) {
	tc := FromContextOptional(ctx)

	if tc.TenantID != "" {
		tenantID = tc.TenantID
	} else {
		tenantID = DefaultTenantID
	}

	if tc.FacilityID != "" {
		facilityID = tc.FacilityID
	} else {
		facilityID = DefaultFacilityID
	}

	if tc.WarehouseID != "" {
		warehouseID = tc.WarehouseID
	} else {
		warehouseID = DefaultWarehouseID
	}

	sellerID = tc.SellerID // Can be empty for non-3PL operations

	return
}

// TenantIndexes returns standard MongoDB index definitions for tenant fields.
// Add these to your collection indexes for efficient tenant-scoped queries.
func TenantIndexes() []bson.D {
	return []bson.D{
		// Tenant + Facility compound index
		{{Key: "tenantId", Value: 1}, {Key: "facilityId", Value: 1}},
		// Tenant + Seller compound index (for 3PL queries)
		{{Key: "tenantId", Value: 1}, {Key: "sellerId", Value: 1}},
		// Full tenant context index
		{{Key: "tenantId", Value: 1}, {Key: "facilityId", Value: 1}, {Key: "warehouseId", Value: 1}},
	}
}
