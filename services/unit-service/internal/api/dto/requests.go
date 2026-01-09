package dto

import "github.com/wms-platform/services/unit-service/internal/domain"

// CreateUnitsRequest holds the input for creating units
type CreateUnitsRequest struct {
	SKU        string `json:"sku" binding:"required"`
	ShipmentID string `json:"shipmentId" binding:"required"`
	LocationID string `json:"locationId" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,min=1"`
	CreatedBy  string `json:"createdBy" binding:"required"`
}

// ReserveUnitsRequest holds the input for reserving units
type ReserveUnitsRequest struct {
	OrderID   string            `json:"orderId" binding:"required"`
	PathID    string            `json:"pathId" binding:"required"`
	Items     []ReserveItemSpec `json:"items" binding:"required,dive"`
	HandlerID string            `json:"handlerId" binding:"required"`
}

// ReserveItemSpec specifies SKU and quantity to reserve
type ReserveItemSpec struct {
	SKU      string `json:"sku" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

// ConfirmPickRequest holds the input for confirming a unit pick
type ConfirmPickRequest struct {
	ToteID    string `json:"toteId" binding:"required"`
	PickerID  string `json:"pickerId" binding:"required"`
	StationID string `json:"stationId"`
}

// ConfirmConsolidationRequest holds the input for confirming consolidation
type ConfirmConsolidationRequest struct {
	DestinationBin string `json:"destinationBin" binding:"required"`
	WorkerID       string `json:"workerId" binding:"required"`
	StationID      string `json:"stationId"`
}

// ConfirmPackedRequest holds the input for confirming packing
type ConfirmPackedRequest struct {
	PackageID string `json:"packageId" binding:"required"`
	PackerID  string `json:"packerId" binding:"required"`
	StationID string `json:"stationId"`
}

// ConfirmShippedRequest holds the input for confirming shipping
type ConfirmShippedRequest struct {
	ShipmentID     string `json:"shipmentId" binding:"required"`
	TrackingNumber string `json:"trackingNumber" binding:"required"`
	HandlerID      string `json:"handlerId" binding:"required"`
}

// CreateExceptionRequest holds the input for creating an exception
type CreateExceptionRequest struct {
	ExceptionType string `json:"exceptionType" binding:"required"`
	Stage         string `json:"stage" binding:"required"`
	Description   string `json:"description" binding:"required"`
	StationID     string `json:"stationId"`
	ReportedBy    string `json:"reportedBy" binding:"required"`
}

// ResolveExceptionRequest holds the input for resolving an exception
type ResolveExceptionRequest struct {
	Resolution string `json:"resolution" binding:"required"`
	ResolvedBy string `json:"resolvedBy" binding:"required"`
}

// ToExceptionType converts string to domain type
func (r *CreateExceptionRequest) ToExceptionType() domain.ExceptionType {
	return domain.ExceptionType(r.ExceptionType)
}

// ToExceptionStage converts string to domain type
func (r *CreateExceptionRequest) ToExceptionStage() domain.ExceptionStage {
	return domain.ExceptionStage(r.Stage)
}
