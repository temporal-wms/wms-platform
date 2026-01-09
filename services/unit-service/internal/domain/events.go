package domain

import "time"

// UnitCreatedEvent - when a unit is generated at receiving
type UnitCreatedEvent struct {
	UnitID     string    `json:"unitId"`
	SKU        string    `json:"sku"`
	ShipmentID string    `json:"shipmentId"`
	LocationID string    `json:"locationId"`
	CreatedBy  string    `json:"createdBy"`
	CreatedAt  time.Time `json:"createdAt"`
}

func NewUnitCreatedEvent(u *Unit, createdBy string) *UnitCreatedEvent {
	return &UnitCreatedEvent{
		UnitID:     u.UnitID,
		SKU:        u.SKU,
		ShipmentID: u.ShipmentID,
		LocationID: u.CurrentLocationID,
		CreatedBy:  createdBy,
		CreatedAt:  u.CreatedAt,
	}
}

func (e *UnitCreatedEvent) EventType() string  { return "UnitCreatedEvent" }
func (e *UnitCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// UnitReservedEvent - when a unit is reserved for an order
type UnitReservedEvent struct {
	UnitID        string    `json:"unitId"`
	SKU           string    `json:"sku"`
	OrderID       string    `json:"orderId"`
	PathID        string    `json:"pathId"`
	ReservationID string    `json:"reservationId"`
	ReservedAt    time.Time `json:"reservedAt"`
}

func NewUnitReservedEvent(u *Unit) *UnitReservedEvent {
	return &UnitReservedEvent{
		UnitID:        u.UnitID,
		SKU:           u.SKU,
		OrderID:       u.OrderID,
		PathID:        u.AssignedPathID,
		ReservationID: u.ReservationID,
		ReservedAt:    *u.ReservedAt,
	}
}

func (e *UnitReservedEvent) EventType() string  { return "UnitReservedEvent" }
func (e *UnitReservedEvent) OccurredAt() time.Time { return e.ReservedAt }

// UnitStagedEvent - when a unit is staged (hard allocated)
type UnitStagedEvent struct {
	UnitID            string    `json:"unitId"`
	SKU               string    `json:"sku"`
	OrderID           string    `json:"orderId"`
	AllocationID      string    `json:"allocationId"`
	StagingLocationID string    `json:"stagingLocationId"`
	StagedAt          time.Time `json:"stagedAt"`
}

func NewUnitStagedEvent(u *Unit) *UnitStagedEvent {
	return &UnitStagedEvent{
		UnitID:            u.UnitID,
		SKU:               u.SKU,
		OrderID:           u.OrderID,
		AllocationID:      u.AllocationID,
		StagingLocationID: u.CurrentLocationID,
		StagedAt:          *u.StagedAt,
	}
}

func (e *UnitStagedEvent) EventType() string  { return "UnitStagedEvent" }
func (e *UnitStagedEvent) OccurredAt() time.Time { return e.StagedAt }

// UnitPickedEvent - when a unit is physically picked
type UnitPickedEvent struct {
	UnitID    string    `json:"unitId"`
	SKU       string    `json:"sku"`
	OrderID   string    `json:"orderId"`
	PickerID  string    `json:"pickerId"`
	ToteID    string    `json:"toteId"`
	StationID string    `json:"stationId"`
	PickedAt  time.Time `json:"pickedAt"`
}

func NewUnitPickedEvent(u *Unit, pickerID, stationID string) *UnitPickedEvent {
	return &UnitPickedEvent{
		UnitID:    u.UnitID,
		SKU:       u.SKU,
		OrderID:   u.OrderID,
		PickerID:  pickerID,
		ToteID:    u.ToteID,
		StationID: stationID,
		PickedAt:  *u.PickedAt,
	}
}

func (e *UnitPickedEvent) EventType() string  { return "UnitPickedEvent" }
func (e *UnitPickedEvent) OccurredAt() time.Time { return e.PickedAt }

// UnitConsolidatedEvent - when a unit is consolidated
type UnitConsolidatedEvent struct {
	UnitID         string    `json:"unitId"`
	SKU            string    `json:"sku"`
	OrderID        string    `json:"orderId"`
	DestinationBin string    `json:"destinationBin"`
	WorkerID       string    `json:"workerId"`
	StationID      string    `json:"stationId"`
	ConsolidatedAt time.Time `json:"consolidatedAt"`
}

func NewUnitConsolidatedEvent(u *Unit, workerID, stationID string) *UnitConsolidatedEvent {
	return &UnitConsolidatedEvent{
		UnitID:         u.UnitID,
		SKU:            u.SKU,
		OrderID:        u.OrderID,
		DestinationBin: u.CurrentLocationID,
		WorkerID:       workerID,
		StationID:      stationID,
		ConsolidatedAt: *u.ConsolidatedAt,
	}
}

func (e *UnitConsolidatedEvent) EventType() string  { return "UnitConsolidatedEvent" }
func (e *UnitConsolidatedEvent) OccurredAt() time.Time { return e.ConsolidatedAt }

// UnitPackedEvent - when a unit is packed
type UnitPackedEvent struct {
	UnitID    string    `json:"unitId"`
	SKU       string    `json:"sku"`
	OrderID   string    `json:"orderId"`
	PackageID string    `json:"packageId"`
	PackerID  string    `json:"packerId"`
	StationID string    `json:"stationId"`
	PackedAt  time.Time `json:"packedAt"`
}

func NewUnitPackedEvent(u *Unit, packerID, stationID string) *UnitPackedEvent {
	return &UnitPackedEvent{
		UnitID:    u.UnitID,
		SKU:       u.SKU,
		OrderID:   u.OrderID,
		PackageID: u.PackageID,
		PackerID:  packerID,
		StationID: stationID,
		PackedAt:  *u.PackedAt,
	}
}

func (e *UnitPackedEvent) EventType() string  { return "UnitPackedEvent" }
func (e *UnitPackedEvent) OccurredAt() time.Time { return e.PackedAt }

// UnitShippedEvent - when a unit is shipped
type UnitShippedEvent struct {
	UnitID         string    `json:"unitId"`
	SKU            string    `json:"sku"`
	OrderID        string    `json:"orderId"`
	ShipmentID     string    `json:"shipmentId"`
	TrackingNumber string    `json:"trackingNumber"`
	ShippedAt      time.Time `json:"shippedAt"`
}

func NewUnitShippedEvent(u *Unit, shipmentID, trackingNumber string) *UnitShippedEvent {
	return &UnitShippedEvent{
		UnitID:         u.UnitID,
		SKU:            u.SKU,
		OrderID:        u.OrderID,
		ShipmentID:     shipmentID,
		TrackingNumber: trackingNumber,
		ShippedAt:      *u.ShippedAt,
	}
}

func (e *UnitShippedEvent) EventType() string  { return "UnitShippedEvent" }
func (e *UnitShippedEvent) OccurredAt() time.Time { return e.ShippedAt }

// UnitExceptionEvent - when a unit has an exception
type UnitExceptionEvent struct {
	UnitID        string    `json:"unitId"`
	SKU           string    `json:"sku"`
	OrderID       string    `json:"orderId"`
	ExceptionID   string    `json:"exceptionId"`
	ExceptionType string    `json:"exceptionType"`
	Reason        string    `json:"reason"`
	ReportedBy    string    `json:"reportedBy"`
	OccurredAt_   time.Time `json:"occurredAt"`
}

func NewUnitExceptionEvent(u *Unit, exceptionID, reason, reportedBy string) *UnitExceptionEvent {
	return &UnitExceptionEvent{
		UnitID:        u.UnitID,
		SKU:           u.SKU,
		OrderID:       u.OrderID,
		ExceptionID:   exceptionID,
		ExceptionType: "unit_exception",
		Reason:        reason,
		ReportedBy:    reportedBy,
		OccurredAt_:   time.Now(),
	}
}

func (e *UnitExceptionEvent) EventType() string  { return "UnitExceptionEvent" }
func (e *UnitExceptionEvent) OccurredAt() time.Time { return e.OccurredAt_ }
