package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UnitStatus represents the lifecycle status of a physical unit
type UnitStatus string

const (
	UnitStatusReceived     UnitStatus = "received"
	UnitStatusReserved     UnitStatus = "reserved"
	UnitStatusStaged       UnitStatus = "staged"
	UnitStatusPicked       UnitStatus = "picked"
	UnitStatusConsolidated UnitStatus = "consolidated"
	UnitStatusPacked       UnitStatus = "packed"
	UnitStatusShipped      UnitStatus = "shipped"
	UnitStatusException    UnitStatus = "exception"
)

// IsValid checks if the status is a valid UnitStatus
func (s UnitStatus) IsValid() bool {
	switch s {
	case UnitStatusReceived, UnitStatusReserved, UnitStatusStaged,
		UnitStatusPicked, UnitStatusConsolidated, UnitStatusPacked,
		UnitStatusShipped, UnitStatusException:
		return true
	}
	return false
}

// UnitMovement tracks a unit's movement through the warehouse
type UnitMovement struct {
	MovementID     string     `bson:"movementId" json:"movementId"`
	FromLocationID string     `bson:"fromLocationId" json:"fromLocationId"`
	ToLocationID   string     `bson:"toLocationId" json:"toLocationId"`
	FromStatus     UnitStatus `bson:"fromStatus" json:"fromStatus"`
	ToStatus       UnitStatus `bson:"toStatus" json:"toStatus"`
	StationID      string     `bson:"stationId,omitempty" json:"stationId,omitempty"`
	HandlerID      string     `bson:"handlerId" json:"handlerId"`
	Timestamp      time.Time  `bson:"timestamp" json:"timestamp"`
	Notes          string     `bson:"notes,omitempty" json:"notes,omitempty"`
}

// Unit is the aggregate root for individual physical units
type Unit struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UnitID            string             `bson:"unitId" json:"unitId"`
	SKU               string             `bson:"sku" json:"sku"`

	// Multi-tenant fields for 3PL/FBA-style operations
	TenantID    string `bson:"tenantId" json:"tenantId"`       // 3PL operator identifier
	FacilityID  string `bson:"facilityId" json:"facilityId"`   // Physical facility/warehouse complex
	WarehouseID string `bson:"warehouseId" json:"warehouseId"` // Specific warehouse within facility
	SellerID    string `bson:"sellerId,omitempty" json:"sellerId,omitempty"` // Merchant/seller who owns this unit

	OrderID           string             `bson:"orderId,omitempty" json:"orderId,omitempty"`
	ShipmentID        string             `bson:"shipmentId" json:"shipmentId"`
	Status            UnitStatus         `bson:"status" json:"status"`
	CurrentLocationID string             `bson:"currentLocationId" json:"currentLocationId"`
	AssignedPathID    string             `bson:"assignedPathId,omitempty" json:"assignedPathId,omitempty"`

	// Allocation tracking
	ReservationID string `bson:"reservationId,omitempty" json:"reservationId,omitempty"`
	AllocationID  string `bson:"allocationId,omitempty" json:"allocationId,omitempty"`
	ToteID        string `bson:"toteId,omitempty" json:"toteId,omitempty"`
	PackageID     string `bson:"packageId,omitempty" json:"packageId,omitempty"`

	// Multi-route support fields
	RouteID      string `bson:"routeId,omitempty" json:"routeId,omitempty"`           // Associated picking route
	RouteIndex   int    `bson:"routeIndex" json:"routeIndex"`                         // Route index in multi-route order
	SourceToteID string `bson:"sourceToteId,omitempty" json:"sourceToteId,omitempty"` // Tote from picking

	// Audit trail
	Movements []UnitMovement `bson:"movements" json:"movements"`

	// Exception info
	ExceptionID     string `bson:"exceptionId,omitempty" json:"exceptionId,omitempty"`
	ExceptionReason string `bson:"exceptionReason,omitempty" json:"exceptionReason,omitempty"`

	// Timestamps
	ReceivedAt     time.Time  `bson:"receivedAt" json:"receivedAt"`
	ReservedAt     *time.Time `bson:"reservedAt,omitempty" json:"reservedAt,omitempty"`
	StagedAt       *time.Time `bson:"stagedAt,omitempty" json:"stagedAt,omitempty"`
	PickedAt       *time.Time `bson:"pickedAt,omitempty" json:"pickedAt,omitempty"`
	ConsolidatedAt *time.Time `bson:"consolidatedAt,omitempty" json:"consolidatedAt,omitempty"`
	PackedAt       *time.Time `bson:"packedAt,omitempty" json:"packedAt,omitempty"`
	ShippedAt      *time.Time `bson:"shippedAt,omitempty" json:"shippedAt,omitempty"`
	CreatedAt      time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time  `bson:"updatedAt" json:"updatedAt"`

	// Domain events (not persisted)
	domainEvents []DomainEvent `bson:"-" json:"-"`
}

// UnitTenantInfo holds multi-tenant identification for unit creation
type UnitTenantInfo struct {
	TenantID    string
	FacilityID  string
	WarehouseID string
	SellerID    string
}

// NewUnit creates a new Unit at receiving (backward compatible, uses default tenant)
func NewUnit(sku, shipmentID, locationID, createdBy string) *Unit {
	return NewUnitWithTenant(sku, shipmentID, locationID, createdBy, nil)
}

// NewUnitWithTenant creates a new Unit at receiving with tenant context
func NewUnitWithTenant(sku, shipmentID, locationID, createdBy string, tenant *UnitTenantInfo) *Unit {
	now := time.Now()
	unitID := uuid.New().String()

	unit := &Unit{
		UnitID:            unitID,
		SKU:               sku,
		ShipmentID:        shipmentID,
		Status:            UnitStatusReceived,
		CurrentLocationID: locationID,
		Movements:         make([]UnitMovement, 0),
		ReceivedAt:        now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Set tenant information
	if tenant != nil {
		unit.TenantID = tenant.TenantID
		unit.FacilityID = tenant.FacilityID
		unit.WarehouseID = tenant.WarehouseID
		unit.SellerID = tenant.SellerID
	} else {
		// Default tenant for backward compatibility
		unit.TenantID = "DEFAULT_TENANT"
		unit.FacilityID = "DEFAULT_FACILITY"
		unit.WarehouseID = "DEFAULT_WAREHOUSE"
	}

	// Record initial movement
	unit.Movements = append(unit.Movements, UnitMovement{
		MovementID:   uuid.New().String(),
		ToLocationID: locationID,
		FromStatus:   "",
		ToStatus:     UnitStatusReceived,
		HandlerID:    createdBy,
		Timestamp:    now,
		Notes:        "Unit created at receiving",
	})

	unit.addEvent(NewUnitCreatedEvent(unit, createdBy))

	return unit
}

// Reserve reserves the unit for an order
func (u *Unit) Reserve(orderID, pathID, reservationID, handlerID string) error {
	if u.Status != UnitStatusReceived {
		return fmt.Errorf("cannot reserve unit in status %s", u.Status)
	}

	now := time.Now()
	oldStatus := u.Status

	u.OrderID = orderID
	u.AssignedPathID = pathID
	u.ReservationID = reservationID
	u.Status = UnitStatusReserved
	u.ReservedAt = &now
	u.UpdatedAt = now

	u.recordMovement(u.CurrentLocationID, u.CurrentLocationID, oldStatus, u.Status, "", handlerID, "Reserved for order")
	u.addEvent(NewUnitReservedEvent(u))

	return nil
}

// Stage stages the unit (hard allocation)
func (u *Unit) Stage(allocationID, stagingLocationID, handlerID string) error {
	if u.Status != UnitStatusReserved {
		return fmt.Errorf("cannot stage unit in status %s", u.Status)
	}

	now := time.Now()
	oldStatus := u.Status
	oldLocation := u.CurrentLocationID

	u.AllocationID = allocationID
	u.CurrentLocationID = stagingLocationID
	u.Status = UnitStatusStaged
	u.StagedAt = &now
	u.UpdatedAt = now

	u.recordMovement(oldLocation, stagingLocationID, oldStatus, u.Status, "", handlerID, "Staged for picking")
	u.addEvent(NewUnitStagedEvent(u))

	return nil
}

// Pick marks the unit as picked
func (u *Unit) Pick(toteID, pickerID, stationID string) error {
	if u.Status != UnitStatusStaged && u.Status != UnitStatusReserved {
		return fmt.Errorf("cannot pick unit in status %s", u.Status)
	}

	now := time.Now()
	oldStatus := u.Status

	u.ToteID = toteID
	u.Status = UnitStatusPicked
	u.PickedAt = &now
	u.UpdatedAt = now

	u.recordMovement(u.CurrentLocationID, toteID, oldStatus, u.Status, stationID, pickerID, "Picked into tote")
	u.addEvent(NewUnitPickedEvent(u, pickerID, stationID))

	return nil
}

// Consolidate marks the unit as consolidated
func (u *Unit) Consolidate(destinationBin, workerID, stationID string) error {
	if u.Status != UnitStatusPicked {
		return fmt.Errorf("cannot consolidate unit in status %s", u.Status)
	}

	now := time.Now()
	oldStatus := u.Status
	oldLocation := u.CurrentLocationID

	u.CurrentLocationID = destinationBin
	u.Status = UnitStatusConsolidated
	u.ConsolidatedAt = &now
	u.UpdatedAt = now

	u.recordMovement(oldLocation, destinationBin, oldStatus, u.Status, stationID, workerID, "Consolidated")
	u.addEvent(NewUnitConsolidatedEvent(u, workerID, stationID))

	return nil
}

// Pack marks the unit as packed
func (u *Unit) Pack(packageID, packerID, stationID string) error {
	if u.Status != UnitStatusPicked && u.Status != UnitStatusConsolidated {
		return fmt.Errorf("cannot pack unit in status %s", u.Status)
	}

	now := time.Now()
	oldStatus := u.Status

	u.PackageID = packageID
	u.Status = UnitStatusPacked
	u.PackedAt = &now
	u.UpdatedAt = now

	u.recordMovement(u.CurrentLocationID, packageID, oldStatus, u.Status, stationID, packerID, "Packed into package")
	u.addEvent(NewUnitPackedEvent(u, packerID, stationID))

	return nil
}

// Ship marks the unit as shipped
func (u *Unit) Ship(shipmentID, trackingNumber, handlerID string) error {
	if u.Status != UnitStatusPacked {
		return fmt.Errorf("cannot ship unit in status %s", u.Status)
	}

	now := time.Now()
	oldStatus := u.Status

	u.Status = UnitStatusShipped
	u.ShippedAt = &now
	u.UpdatedAt = now

	u.recordMovement(u.CurrentLocationID, "shipped", oldStatus, u.Status, "", handlerID, fmt.Sprintf("Shipped with tracking %s", trackingNumber))
	u.addEvent(NewUnitShippedEvent(u, shipmentID, trackingNumber))

	return nil
}

// Release releases the unit reservation and returns it to available status
func (u *Unit) Release(handlerID, reason string) error {
	if u.Status != UnitStatusReserved {
		return fmt.Errorf("cannot release unit in status %s", u.Status)
	}

	now := time.Now()
	oldStatus := u.Status

	// Clear reservation data
	orderID := u.OrderID
	u.OrderID = ""
	u.AssignedPathID = ""
	u.ReservationID = ""
	u.Status = UnitStatusReceived
	u.ReservedAt = nil
	u.UpdatedAt = now

	u.recordMovement(u.CurrentLocationID, u.CurrentLocationID, oldStatus, u.Status, "", handlerID, reason)
	u.addEvent(NewUnitReleasedEvent(u, orderID, reason))

	return nil
}

// MarkException marks the unit as having an exception
func (u *Unit) MarkException(exceptionID, reason, handlerID, stationID string) error {
	now := time.Now()
	oldStatus := u.Status

	u.ExceptionID = exceptionID
	u.ExceptionReason = reason
	u.Status = UnitStatusException
	u.UpdatedAt = now

	u.recordMovement(u.CurrentLocationID, u.CurrentLocationID, oldStatus, u.Status, stationID, handlerID, reason)
	u.addEvent(NewUnitExceptionEvent(u, exceptionID, reason, handlerID))

	return nil
}

// recordMovement adds a movement to the audit trail
func (u *Unit) recordMovement(fromLocation, toLocation string, fromStatus, toStatus UnitStatus, stationID, handlerID, notes string) {
	u.Movements = append(u.Movements, UnitMovement{
		MovementID:     uuid.New().String(),
		FromLocationID: fromLocation,
		ToLocationID:   toLocation,
		FromStatus:     fromStatus,
		ToStatus:       toStatus,
		StationID:      stationID,
		HandlerID:      handlerID,
		Timestamp:      time.Now(),
		Notes:          notes,
	})
}

// Events returns all domain events and clears them
func (u *Unit) Events() []DomainEvent {
	events := u.domainEvents
	u.domainEvents = nil
	return events
}

func (u *Unit) addEvent(event DomainEvent) {
	u.domainEvents = append(u.domainEvents, event)
}

// GetAuditTrail returns the full movement history
func (u *Unit) GetAuditTrail() []UnitMovement {
	return u.Movements
}

// AssignToRoute associates the unit with a picking route
func (u *Unit) AssignToRoute(routeID string, routeIndex int) {
	u.RouteID = routeID
	u.RouteIndex = routeIndex
	u.UpdatedAt = time.Now()
}

// SetSourceTote records which tote the unit was picked into
func (u *Unit) SetSourceTote(toteID string) {
	u.SourceToteID = toteID
	u.UpdatedAt = time.Now()
}

// HasRouteAssignment returns true if the unit is assigned to a route
func (u *Unit) HasRouteAssignment() bool {
	return u.RouteID != ""
}

// DomainEvent interface for unit events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}
