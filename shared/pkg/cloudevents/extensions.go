package cloudevents

import (
	"github.com/wms-platform/shared/pkg/tenant"
)

// CloudEvents extension attribute names for WMS tenant context
const (
	// Tenant context extensions (used in CloudEvents and message headers)
	ExtTenantID    = "wmstenantid"
	ExtFacilityID  = "wmsfacilityid"
	ExtWarehouseID = "wmswarehouseid"
	ExtSellerID    = "wmssellerid"
	ExtChannelID   = "wmschannelid"

	// Business context extensions
	ExtCorrelationID = "wmscorrelationid"
	ExtWaveNumber    = "wmswavenumber"
	ExtWorkflowID    = "wmsworkflowid"
	ExtOrderID       = "wmsorderid"
)

// HTTP header names for WMS tenant context
const (
	HeaderTenantID    = "X-WMS-Tenant-ID"
	HeaderFacilityID  = "X-WMS-Facility-ID"
	HeaderWarehouseID = "X-WMS-Warehouse-ID"
	HeaderSellerID    = "X-WMS-Seller-ID"
	HeaderChannelID   = "X-WMS-Channel-ID"
)

// SetTenantContext sets tenant context extensions on a WMSCloudEvent
func (e *WMSCloudEvent) SetTenantContext(tc *tenant.Context) {
	if tc == nil {
		return
	}
	e.TenantID = tc.TenantID
	e.FacilityID = tc.FacilityID
	e.WarehouseID = tc.WarehouseID
	e.SellerID = tc.SellerID
	e.ChannelID = tc.ChannelID
}

// GetTenantContext extracts tenant context from a WMSCloudEvent
func (e *WMSCloudEvent) GetTenantContext() *tenant.Context {
	return &tenant.Context{
		TenantID:    e.TenantID,
		FacilityID:  e.FacilityID,
		WarehouseID: e.WarehouseID,
		SellerID:    e.SellerID,
		ChannelID:   e.ChannelID,
	}
}

// WithTenantContext is a builder method that sets tenant context and returns the event
func (e *WMSCloudEvent) WithTenantContext(tc *tenant.Context) *WMSCloudEvent {
	e.SetTenantContext(tc)
	return e
}

// WithTenant sets individual tenant fields and returns the event
func (e *WMSCloudEvent) WithTenant(tenantID, facilityID, warehouseID string) *WMSCloudEvent {
	e.TenantID = tenantID
	e.FacilityID = facilityID
	e.WarehouseID = warehouseID
	return e
}

// WithSeller sets seller and channel and returns the event
func (e *WMSCloudEvent) WithSeller(sellerID, channelID string) *WMSCloudEvent {
	e.SellerID = sellerID
	e.ChannelID = channelID
	return e
}

// HasTenantContext returns true if all required tenant fields are set
func (e *WMSCloudEvent) HasTenantContext() bool {
	return e.TenantID != "" && e.FacilityID != "" && e.WarehouseID != ""
}

// ValidateTenantContext validates that required tenant context is present
func (e *WMSCloudEvent) ValidateTenantContext() error {
	tc := e.GetTenantContext()
	return tc.Validate()
}
