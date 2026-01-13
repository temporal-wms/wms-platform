package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrAllocationNotStaged  = errors.New("allocation is not in staged status")
	ErrAllocationNotPacked  = errors.New("allocation is not in packed status")
	ErrAllocationAlreadyShipped = errors.New("allocation has already been shipped")
)

// AllocationStatus represents the status of a hard allocation
type AllocationStatus string

const (
	AllocationStatusStaged   AllocationStatus = "staged"
	AllocationStatusPacked   AllocationStatus = "packed"
	AllocationStatusShipped  AllocationStatus = "shipped"
	AllocationStatusReturned AllocationStatus = "returned"
)

// InventoryAllocationAggregate represents physically staged/locked inventory as a separate aggregate
// This prevents unbounded growth of hardAllocations array in InventoryItem
type InventoryAllocationAggregate struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	AllocationID  string             `bson:"allocationId"`
	SKU           string             `bson:"sku"` // Reference to inventory item

	// Multi-tenant fields
	TenantID    string `bson:"tenantId"`
	FacilityID  string `bson:"facilityId"`
	WarehouseID string `bson:"warehouseId"`
	SellerID    string `bson:"sellerId,omitempty"`

	ReservationID     string           `bson:"reservationId"` // Reference to reservation
	OrderID           string           `bson:"orderId"`
	Quantity          int              `bson:"quantity"`
	SourceLocationID  string           `bson:"sourceLocationId"`
	StagingLocationID string           `bson:"stagingLocationId"`
	Status            AllocationStatus `bson:"status"`
	UnitIDs           []string         `bson:"unitIds,omitempty"` // Specific units allocated for unit-level tracking

	StagedBy  string     `bson:"stagedBy"`
	PackedBy  string     `bson:"packedBy,omitempty"`
	ShippedBy string     `bson:"shippedBy,omitempty"`
	CreatedAt time.Time  `bson:"createdAt"`
	PackedAt  *time.Time `bson:"packedAt,omitempty"`
	ShippedAt *time.Time `bson:"shippedAt,omitempty"`
	UpdatedAt time.Time  `bson:"updatedAt"`

	DomainEvents []DomainEvent `bson:"-"`
}

// AllocationTenantInfo holds multi-tenant identification for allocations
type AllocationTenantInfo struct {
	TenantID    string
	FacilityID  string
	WarehouseID string
	SellerID    string
}

// NewInventoryAllocation creates a new inventory allocation aggregate
func NewInventoryAllocation(
	allocationID string,
	sku string,
	reservationID string,
	orderID string,
	quantity int,
	sourceLocationID string,
	stagingLocationID string,
	unitIDs []string,
	stagedBy string,
	tenant *AllocationTenantInfo,
) *InventoryAllocationAggregate {
	now := time.Now()
	allocation := &InventoryAllocationAggregate{
		AllocationID:      allocationID,
		SKU:               sku,
		ReservationID:     reservationID,
		OrderID:           orderID,
		Quantity:          quantity,
		SourceLocationID:  sourceLocationID,
		StagingLocationID: stagingLocationID,
		Status:            AllocationStatusStaged,
		UnitIDs:           unitIDs,
		StagedBy:          stagedBy,
		CreatedAt:         now,
		UpdatedAt:         now,
		DomainEvents:      make([]DomainEvent, 0),
	}

	if tenant != nil {
		allocation.TenantID = tenant.TenantID
		allocation.FacilityID = tenant.FacilityID
		allocation.WarehouseID = tenant.WarehouseID
		allocation.SellerID = tenant.SellerID
	}

	// Emit staged event
	allocation.AddDomainEvent(&InventoryStagedEvent{
		SKU:               sku,
		AllocationID:      allocationID,
		OrderID:           orderID,
		Quantity:          quantity,
		SourceLocationID:  sourceLocationID,
		StagingLocationID: stagingLocationID,
		StagedBy:          stagedBy,
		StagedAt:          now,
	})

	return allocation
}

// MarkPacked marks the allocation as packed (ready for shipping)
func (a *InventoryAllocationAggregate) MarkPacked(packedBy string) error {
	if a.Status != AllocationStatusStaged {
		return ErrAllocationNotStaged
	}

	now := time.Now()
	a.Status = AllocationStatusPacked
	a.PackedBy = packedBy
	a.PackedAt = &now
	a.UpdatedAt = now

	// Emit packed event
	a.AddDomainEvent(&InventoryPackedEvent{
		SKU:          a.SKU,
		AllocationID: a.AllocationID,
		OrderID:      a.OrderID,
		PackedBy:     packedBy,
		PackedAt:     now,
	})

	return nil
}

// MarkShipped marks the allocation as shipped
func (a *InventoryAllocationAggregate) MarkShipped(shippedBy string) error {
	if a.Status != AllocationStatusPacked {
		return ErrAllocationNotPacked
	}

	now := time.Now()
	a.Status = AllocationStatusShipped
	a.ShippedBy = shippedBy
	a.ShippedAt = &now
	a.UpdatedAt = now

	// Emit shipped event
	a.AddDomainEvent(&InventoryShippedEvent{
		SKU:          a.SKU,
		AllocationID: a.AllocationID,
		OrderID:      a.OrderID,
		Quantity:     a.Quantity,
		ShippedAt:    now,
	})

	return nil
}

// ReturnToShelf returns the allocation to the shelf
func (a *InventoryAllocationAggregate) ReturnToShelf(returnedBy string, reason string) error {
	if a.Status == AllocationStatusShipped {
		return ErrAllocationAlreadyShipped
	}

	a.Status = AllocationStatusReturned
	a.UpdatedAt = time.Now()

	// Emit return event
	a.AddDomainEvent(&InventoryReturnedToShelfEvent{
		SKU:              a.SKU,
		AllocationID:     a.AllocationID,
		OrderID:          a.OrderID,
		Quantity:         a.Quantity,
		SourceLocationID: a.SourceLocationID,
		ReturnedBy:       returnedBy,
		Reason:           reason,
		ReturnedAt:       time.Now(),
	})

	return nil
}

// IsActive returns true if the allocation is in staged or packed status
func (a *InventoryAllocationAggregate) IsActive() bool {
	return a.Status == AllocationStatusStaged || a.Status == AllocationStatusPacked
}

// IsShipped returns true if the allocation has been shipped
func (a *InventoryAllocationAggregate) IsShipped() bool {
	return a.Status == AllocationStatusShipped
}

// IsReturned returns true if the allocation has been returned
func (a *InventoryAllocationAggregate) IsReturned() bool {
	return a.Status == AllocationStatusReturned
}

// AddDomainEvent adds a domain event
func (a *InventoryAllocationAggregate) AddDomainEvent(event DomainEvent) {
	a.DomainEvents = append(a.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (a *InventoryAllocationAggregate) ClearDomainEvents() {
	a.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (a *InventoryAllocationAggregate) GetDomainEvents() []DomainEvent {
	return a.DomainEvents
}
