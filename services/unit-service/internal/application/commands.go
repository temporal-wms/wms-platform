package application

import "github.com/wms-platform/services/unit-service/internal/domain"

// CreateUnitsCommand holds the input for creating units at receiving
type CreateUnitsCommand struct {
	SKU        string `json:"sku"`
	ShipmentID string `json:"shipmentId"`
	LocationID string `json:"locationId"`
	Quantity   int    `json:"quantity"`
	CreatedBy  string `json:"createdBy"`
	// Multi-tenant context
	TenantID    string `json:"tenantId,omitempty"`
	FacilityID  string `json:"facilityId,omitempty"`
	WarehouseID string `json:"warehouseId,omitempty"`
	SellerID    string `json:"sellerId,omitempty"`
}

// ReserveUnitsCommand holds the input for reserving units for an order
type ReserveUnitsCommand struct {
	OrderID   string            `json:"orderId"`
	PathID    string            `json:"pathId"`
	Items     []ReserveItemSpec `json:"items"`
	HandlerID string            `json:"handlerId"`
	// Multi-tenant context
	TenantID    string `json:"tenantId,omitempty"`
	FacilityID  string `json:"facilityId,omitempty"`
	WarehouseID string `json:"warehouseId,omitempty"`
	SellerID    string `json:"sellerId,omitempty"`
}

// ReserveItemSpec specifies SKU and quantity to reserve
type ReserveItemSpec struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// ConfirmPickCommand holds the input for confirming a unit pick
type ConfirmPickCommand struct {
	UnitID    string `json:"unitId"`
	ToteID    string `json:"toteId"`
	PickerID  string `json:"pickerId"`
	StationID string `json:"stationId"`
}

// ConfirmConsolidationCommand holds the input for confirming unit consolidation
type ConfirmConsolidationCommand struct {
	UnitID         string `json:"unitId"`
	DestinationBin string `json:"destinationBin"`
	WorkerID       string `json:"workerId"`
	StationID      string `json:"stationId"`
}

// ConfirmPackedCommand holds the input for confirming a unit is packed
type ConfirmPackedCommand struct {
	UnitID    string `json:"unitId"`
	PackageID string `json:"packageId"`
	PackerID  string `json:"packerId"`
	StationID string `json:"stationId"`
}

// ConfirmShippedCommand holds the input for confirming a unit is shipped
type ConfirmShippedCommand struct {
	UnitID         string `json:"unitId"`
	ShipmentID     string `json:"shipmentId"`
	TrackingNumber string `json:"trackingNumber"`
	HandlerID      string `json:"handlerId"`
}

// CreateExceptionCommand holds the input for creating a unit exception
type CreateExceptionCommand struct {
	UnitID        string                `json:"unitId"`
	ExceptionType domain.ExceptionType  `json:"exceptionType"`
	Stage         domain.ExceptionStage `json:"stage"`
	Description   string                `json:"description"`
	StationID     string                `json:"stationId"`
	ReportedBy    string                `json:"reportedBy"`
}

// ResolveExceptionCommand holds the input for resolving an exception
type ResolveExceptionCommand struct {
	ExceptionID string `json:"exceptionId"`
	Resolution  string `json:"resolution"`
	ResolvedBy  string `json:"resolvedBy"`
}

// ReleaseUnitsCommand holds the input for releasing unit reservations
type ReleaseUnitsCommand struct {
	OrderID   string `json:"orderId"`
	HandlerID string `json:"handlerId"`
	Reason    string `json:"reason"`
}

// CreateUnitsResult holds the result of creating units
type CreateUnitsResult struct {
	UnitIDs []string `json:"unitIds"`
	SKU     string   `json:"sku"`
	Count   int      `json:"count"`
}

// ReserveUnitsResult holds the result of reserving units
type ReserveUnitsResult struct {
	ReservedUnits []ReservedUnitInfo `json:"reservedUnits"`
	FailedItems   []FailedReserve    `json:"failedItems,omitempty"`
}

// ReservedUnitInfo holds info about a reserved unit
type ReservedUnitInfo struct {
	UnitID     string `json:"unitId"`
	SKU        string `json:"sku"`
	LocationID string `json:"locationId"`
}

// FailedReserve holds info about a failed reservation
type FailedReserve struct {
	SKU       string `json:"sku"`
	Requested int    `json:"requested"`
	Available int    `json:"available"`
	Reason    string `json:"reason"`
}
