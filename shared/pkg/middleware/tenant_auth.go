package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wms-platform/shared/pkg/tenant"
)

// TenantAuthConfig holds configuration for tenant authorization middleware
type TenantAuthConfig struct {
	// Required when true, requests without tenant context will be rejected
	Required bool

	// Validator is an optional interface to validate tenant access
	Validator TenantValidator

	// DefaultTenantID is used when no tenant header is provided and Required is false
	DefaultTenantID string

	// DefaultFacilityID is used when no facility header is provided and Required is false
	DefaultFacilityID string

	// DefaultWarehouseID is used when no warehouse header is provided and Required is false
	DefaultWarehouseID string
}

// TenantValidator interface for validating tenant access
type TenantValidator interface {
	// ValidateTenantAccess checks if the user (from auth context) has access to the tenant
	ValidateTenantAccess(userID, tenantID, facilityID string) error

	// GetUserTenants returns the list of tenants a user has access to
	GetUserTenants(userID string) ([]string, error)
}

// DefaultTenantAuthConfig returns a default configuration for backward compatibility
func DefaultTenantAuthConfig() *TenantAuthConfig {
	return &TenantAuthConfig{
		Required:           false,
		DefaultTenantID:    tenant.DefaultTenantID,
		DefaultFacilityID:  tenant.DefaultFacilityID,
		DefaultWarehouseID: tenant.DefaultWarehouseID,
	}
}

// TenantAuth middleware extracts tenant context from headers and adds it to the request context.
// It can optionally validate that the requesting user has access to the tenant.
func TenantAuth(config *TenantAuthConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultTenantAuthConfig()
	}

	return func(c *gin.Context) {
		// Extract tenant context from headers
		tenantID := c.GetHeader(HeaderWMSTenantID)
		facilityID := c.GetHeader(HeaderWMSFacilityID)
		warehouseID := c.GetHeader(HeaderWMSWarehouseID)
		sellerID := c.GetHeader(HeaderWMSSellerID)
		channelID := c.GetHeader(HeaderWMSChannelID)

		// Apply defaults if not provided and config allows
		if tenantID == "" && !config.Required {
			tenantID = config.DefaultTenantID
		}
		if facilityID == "" && !config.Required {
			facilityID = config.DefaultFacilityID
		}
		if warehouseID == "" && !config.Required {
			warehouseID = config.DefaultWarehouseID
		}

		// Check if tenant context is required but missing
		if config.Required && tenantID == "" && facilityID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "MISSING_TENANT_CONTEXT",
				"message": "Tenant or facility context is required",
			})
			return
		}

		// Validate tenant access if validator is configured
		if config.Validator != nil && tenantID != "" {
			// Get user ID from auth context (set by authentication middleware)
			userID := c.GetString("userId")
			if userID == "" {
				// Try alternative key names
				if val, exists := c.Get("user_id"); exists {
					if uid, ok := val.(string); ok {
						userID = uid
					}
				}
			}

			if userID != "" {
				if err := config.Validator.ValidateTenantAccess(userID, tenantID, facilityID); err != nil {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"code":    "UNAUTHORIZED_TENANT_ACCESS",
						"message": "Access to this tenant/facility is not authorized",
					})
					return
				}
			}
		}

		// Create tenant context
		tc := &tenant.Context{
			TenantID:    tenantID,
			FacilityID:  facilityID,
			WarehouseID: warehouseID,
			SellerID:    sellerID,
			ChannelID:   channelID,
		}

		// Add tenant context to Go context
		ctx := tenant.ToContext(c.Request.Context(), tc)
		c.Request = c.Request.WithContext(ctx)

		// Also store in Gin context for easy access in handlers
		c.Set("tenantContext", tc)
		c.Set(ContextKeyWMSTenantID, tenantID)
		c.Set(ContextKeyWMSSellerID, sellerID)
		c.Set(ContextKeyWMSChannelID, channelID)

		c.Next()
	}
}

// GetTenantContext retrieves the tenant context from Gin context
func GetTenantContext(c *gin.Context) *tenant.Context {
	if val, exists := c.Get("tenantContext"); exists {
		if tc, ok := val.(*tenant.Context); ok {
			return tc
		}
	}

	// Fallback: try to build from individual context keys
	return &tenant.Context{
		TenantID:    GetWMSTenantID(c),
		FacilityID:  GetWMSFacilityID(c),
		WarehouseID: GetWMSWarehouseID(c),
		SellerID:    GetWMSSellerID(c),
		ChannelID:   GetWMSChannelID(c),
	}
}

// RequireTenant is a middleware that ensures tenant context is present.
// Use this for endpoints that must have tenant context.
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		tc := GetTenantContext(c)
		if tc == nil || tc.IsEmpty() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "MISSING_TENANT_CONTEXT",
				"message": "Tenant context is required for this endpoint",
			})
			return
		}
		c.Next()
	}
}

// RequireSeller is a middleware that ensures seller context is present.
// Use this for endpoints that are seller-specific (3PL/FBA-style).
func RequireSeller() gin.HandlerFunc {
	return func(c *gin.Context) {
		tc := GetTenantContext(c)
		if tc == nil || !tc.HasSeller() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "MISSING_SELLER_CONTEXT",
				"message": "Seller context is required for this endpoint",
			})
			return
		}
		c.Next()
	}
}

// RequireFacility is a middleware that ensures facility context is present.
// Use this for endpoints that are facility-specific.
func RequireFacility() gin.HandlerFunc {
	return func(c *gin.Context) {
		tc := GetTenantContext(c)
		if tc == nil || !tc.HasFacility() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "MISSING_FACILITY_CONTEXT",
				"message": "Facility context is required for this endpoint",
			})
			return
		}
		c.Next()
	}
}

// SellerOnly is a middleware that restricts access to seller-owned resources.
// It verifies that the requesting seller matches the resource owner.
func SellerOnly(getResourceSellerID func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tc := GetTenantContext(c)
		if tc == nil || !tc.HasSeller() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "MISSING_SELLER_CONTEXT",
				"message": "Seller context is required",
			})
			return
		}

		resourceSellerID := getResourceSellerID(c)
		if resourceSellerID != "" && resourceSellerID != tc.SellerID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "UNAUTHORIZED_SELLER_ACCESS",
				"message": "Access to this resource is not authorized",
			})
			return
		}

		c.Next()
	}
}

// TenantFromPath extracts tenant context from URL path parameters.
// Useful for APIs like /tenants/:tenantId/orders
func TenantFromPath(tenantParam, facilityParam, sellerParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tc := GetTenantContext(c)
		if tc == nil {
			tc = &tenant.Context{}
		}

		// Override with path parameters if provided
		if tenantParam != "" {
			if tenantID := c.Param(tenantParam); tenantID != "" {
				tc.TenantID = tenantID
			}
		}
		if facilityParam != "" {
			if facilityID := c.Param(facilityParam); facilityID != "" {
				tc.FacilityID = facilityID
			}
		}
		if sellerParam != "" {
			if sellerID := c.Param(sellerParam); sellerID != "" {
				tc.SellerID = sellerID
			}
		}

		// Update context
		ctx := tenant.ToContext(c.Request.Context(), tc)
		c.Request = c.Request.WithContext(ctx)
		c.Set("tenantContext", tc)

		c.Next()
	}
}
