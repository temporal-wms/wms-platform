package dto

import (
	"time"

	"github.com/wms-platform/services/unit-service/internal/domain"
)

// CreateUnitsResponse holds the response for unit creation
type CreateUnitsResponse struct {
	UnitIDs []string `json:"unitIds"`
	SKU     string   `json:"sku"`
	Count   int      `json:"count"`
}

// ReserveUnitsResponse holds the response for unit reservation
type ReserveUnitsResponse struct {
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

// UnitResponse holds the response for a single unit
type UnitResponse struct {
	UnitID            string              `json:"unitId"`
	SKU               string              `json:"sku"`
	OrderID           string              `json:"orderId,omitempty"`
	ShipmentID        string              `json:"shipmentId"`
	Status            string              `json:"status"`
	CurrentLocationID string              `json:"currentLocationId"`
	AssignedPathID    string              `json:"assignedPathId,omitempty"`
	ReservationID     string              `json:"reservationId,omitempty"`
	AllocationID      string              `json:"allocationId,omitempty"`
	ToteID            string              `json:"toteId,omitempty"`
	PackageID         string              `json:"packageId,omitempty"`
	ExceptionID       string              `json:"exceptionId,omitempty"`
	ExceptionReason   string              `json:"exceptionReason,omitempty"`
	Movements         []UnitMovementDTO   `json:"movements,omitempty"`
	ReceivedAt        time.Time           `json:"receivedAt"`
	ReservedAt        *time.Time          `json:"reservedAt,omitempty"`
	PickedAt          *time.Time          `json:"pickedAt,omitempty"`
	ConsolidatedAt    *time.Time          `json:"consolidatedAt,omitempty"`
	PackedAt          *time.Time          `json:"packedAt,omitempty"`
	ShippedAt         *time.Time          `json:"shippedAt,omitempty"`
}

// UnitMovementDTO represents a unit movement in the API
type UnitMovementDTO struct {
	MovementID     string    `json:"movementId"`
	FromLocationID string    `json:"fromLocationId"`
	ToLocationID   string    `json:"toLocationId"`
	FromStatus     string    `json:"fromStatus"`
	ToStatus       string    `json:"toStatus"`
	StationID      string    `json:"stationId,omitempty"`
	HandlerID      string    `json:"handlerId"`
	Timestamp      time.Time `json:"timestamp"`
	Notes          string    `json:"notes,omitempty"`
}

// UnitListResponse holds a list of units
type UnitListResponse struct {
	Units []UnitSummary `json:"units"`
	Total int           `json:"total"`
}

// UnitSummary holds summary info for a unit
type UnitSummary struct {
	UnitID   string `json:"unitId"`
	SKU      string `json:"sku"`
	OrderID  string `json:"orderId,omitempty"`
	Status   string `json:"status"`
	Location string `json:"location"`
}

// ExceptionResponse holds the response for an exception
type ExceptionResponse struct {
	ExceptionID   string     `json:"exceptionId"`
	UnitID        string     `json:"unitId"`
	OrderID       string     `json:"orderId"`
	SKU           string     `json:"sku"`
	ExceptionType string     `json:"exceptionType"`
	Stage         string     `json:"stage"`
	Description   string     `json:"description"`
	StationID     string     `json:"stationId,omitempty"`
	ReportedBy    string     `json:"reportedBy"`
	Resolution    string     `json:"resolution,omitempty"`
	ResolvedBy    string     `json:"resolvedBy,omitempty"`
	ResolvedAt    *time.Time `json:"resolvedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// ExceptionListResponse holds a list of exceptions
type ExceptionListResponse struct {
	Exceptions []ExceptionResponse `json:"exceptions"`
	Total      int                 `json:"total"`
}

// AuditTrailResponse holds the audit trail for a unit
type AuditTrailResponse struct {
	UnitID    string            `json:"unitId"`
	Movements []UnitMovementDTO `json:"movements"`
}

// ToUnitResponse converts domain Unit to API response
func ToUnitResponse(u *domain.Unit) UnitResponse {
	resp := UnitResponse{
		UnitID:            u.UnitID,
		SKU:               u.SKU,
		OrderID:           u.OrderID,
		ShipmentID:        u.ShipmentID,
		Status:            string(u.Status),
		CurrentLocationID: u.CurrentLocationID,
		AssignedPathID:    u.AssignedPathID,
		ReservationID:     u.ReservationID,
		AllocationID:      u.AllocationID,
		ToteID:            u.ToteID,
		PackageID:         u.PackageID,
		ExceptionID:       u.ExceptionID,
		ExceptionReason:   u.ExceptionReason,
		ReceivedAt:        u.ReceivedAt,
		ReservedAt:        u.ReservedAt,
		PickedAt:          u.PickedAt,
		ConsolidatedAt:    u.ConsolidatedAt,
		PackedAt:          u.PackedAt,
		ShippedAt:         u.ShippedAt,
	}

	resp.Movements = make([]UnitMovementDTO, len(u.Movements))
	for i, m := range u.Movements {
		resp.Movements[i] = UnitMovementDTO{
			MovementID:     m.MovementID,
			FromLocationID: m.FromLocationID,
			ToLocationID:   m.ToLocationID,
			FromStatus:     string(m.FromStatus),
			ToStatus:       string(m.ToStatus),
			StationID:      m.StationID,
			HandlerID:      m.HandlerID,
			Timestamp:      m.Timestamp,
			Notes:          m.Notes,
		}
	}

	return resp
}

// ToExceptionResponse converts domain UnitException to API response
func ToExceptionResponse(e *domain.UnitException) ExceptionResponse {
	return ExceptionResponse{
		ExceptionID:   e.ExceptionID,
		UnitID:        e.UnitID,
		OrderID:       e.OrderID,
		SKU:           e.SKU,
		ExceptionType: string(e.ExceptionType),
		Stage:         string(e.Stage),
		Description:   e.Description,
		StationID:     e.StationID,
		ReportedBy:    e.ReportedBy,
		Resolution:    e.Resolution,
		ResolvedBy:    e.ResolvedBy,
		ResolvedAt:    e.ResolvedAt,
		CreatedAt:     e.CreatedAt,
	}
}
