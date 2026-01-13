package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InventoryItemRefactored is the optimized aggregate root without unbounded arrays
// This version stores only the current state and computed totals
// Historical data (transactions, reservations, allocations) are stored in separate collections
type InventoryItemRefactored struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	SKU         string             `bson:"sku"`
	ProductName string             `bson:"productName"`

	// Multi-tenant fields for 3PL/FBA-style operations
	TenantID    string `bson:"tenantId" json:"tenantId"`
	FacilityID  string `bson:"facilityId" json:"facilityId"`
	WarehouseID string `bson:"warehouseId" json:"warehouseId"`
	SellerID    string `bson:"sellerId,omitempty" json:"sellerId,omitempty"`

	Locations             []StockLocation `bson:"locations"`
	TotalQuantity         int             `bson:"totalQuantity"`
	ReservedQuantity      int             `bson:"reservedQuantity"`      // Count only, no array
	HardAllocatedQuantity int             `bson:"hardAllocatedQuantity"` // Count only, no array
	AvailableQuantity     int             `bson:"availableQuantity"`
	ReorderPoint          int             `bson:"reorderPoint"`
	ReorderQuantity       int             `bson:"reorderQuantity"`

	// Velocity and storage fields (Amazon-style optimization)
	VelocityClass   VelocityClass   `bson:"velocityClass" json:"velocityClass"`
	StorageStrategy StorageStrategy `bson:"storageStrategy" json:"storageStrategy"`
	PickFrequency   int             `bson:"pickFrequency" json:"pickFrequency"`
	LastStowedAt    *time.Time      `bson:"lastStowedAt,omitempty" json:"lastStowedAt,omitempty"`
	LastPickedAt    *time.Time      `bson:"lastPickedAt,omitempty" json:"lastPickedAt,omitempty"`
	LastCycleCount  *time.Time      `bson:"lastCycleCount,omitempty"`

	CreatedAt    time.Time     `bson:"createdAt"`
	UpdatedAt    time.Time     `bson:"updatedAt"`
	DomainEvents []DomainEvent `bson:"-"`
}

// NewInventoryItemRefactored creates a new optimized InventoryItem aggregate
func NewInventoryItemRefactored(sku, productName string, reorderPoint, reorderQty int, tenant *InventoryTenantInfo) *InventoryItemRefactored {
	now := time.Now()
	item := &InventoryItemRefactored{
		SKU:                   sku,
		ProductName:           productName,
		Locations:             make([]StockLocation, 0),
		TotalQuantity:         0,
		ReservedQuantity:      0,
		HardAllocatedQuantity: 0,
		AvailableQuantity:     0,
		ReorderPoint:          reorderPoint,
		ReorderQuantity:       reorderQty,
		CreatedAt:             now,
		UpdatedAt:             now,
		DomainEvents:          make([]DomainEvent, 0),
	}

	if tenant != nil {
		item.TenantID = tenant.TenantID
		item.FacilityID = tenant.FacilityID
		item.WarehouseID = tenant.WarehouseID
		item.SellerID = tenant.SellerID
	} else {
		item.TenantID = "DEFAULT_TENANT"
		item.FacilityID = "DEFAULT_FACILITY"
		item.WarehouseID = "DEFAULT_WAREHOUSE"
	}

	return item
}

// ReceiveStock adds stock to a location (no transaction array)
func (i *InventoryItemRefactored) ReceiveStock(locationID, zone string, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Find or create location
	found := false
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			i.Locations[idx].Quantity += quantity
			i.Locations[idx].Available += quantity
			found = true
			break
		}
	}

	if !found {
		i.Locations = append(i.Locations, StockLocation{
			LocationID: locationID,
			Zone:       zone,
			Quantity:   quantity,
			Available:  quantity,
		})
	}

	i.TotalQuantity += quantity
	i.AvailableQuantity += quantity
	i.UpdatedAt = time.Now()

	// Emit event (transaction will be stored separately)
	i.AddDomainEvent(&InventoryReceivedEvent{
		SKU:        i.SKU,
		Quantity:   quantity,
		LocationID: locationID,
		ReceivedAt: time.Now(),
	})

	return nil
}

// ReserveStock reserves stock for an order (updates counters only)
// Actual reservation record is created separately in inventory_reservations collection
func (i *InventoryItemRefactored) ReserveStock(locationID string, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Check availability at location
	found := false
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			if i.Locations[idx].Available < quantity {
				return ErrInsufficientStock
			}

			i.Locations[idx].Reserved += quantity
			i.Locations[idx].Available -= quantity
			found = true
			break
		}
	}

	if !found {
		return ErrLocationNotFound
	}

	i.ReservedQuantity += quantity
	i.AvailableQuantity -= quantity
	i.UpdatedAt = time.Now()

	return nil
}

// ReleaseReservation releases a reservation (updates counters only)
func (i *InventoryItemRefactored) ReleaseReservation(locationID string, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Return stock to available
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			i.Locations[idx].Reserved -= quantity
			i.Locations[idx].Available += quantity
			break
		}
	}

	i.ReservedQuantity -= quantity
	i.AvailableQuantity += quantity
	i.UpdatedAt = time.Now()

	return nil
}

// HardAllocateStock moves reserved stock to hard allocated (staged)
func (i *InventoryItemRefactored) HardAllocateStock(locationID string, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Move from reserved to hard allocated
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			if i.Locations[idx].Reserved < quantity {
				return errors.New("insufficient reserved quantity")
			}

			i.Locations[idx].Reserved -= quantity
			i.Locations[idx].HardAllocated += quantity
			break
		}
	}

	i.ReservedQuantity -= quantity
	i.HardAllocatedQuantity += quantity
	i.UpdatedAt = time.Now()

	return nil
}

// ReleaseHardAllocation releases a hard allocation back to available
func (i *InventoryItemRefactored) ReleaseHardAllocation(locationID string, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Move from hard allocated to available
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			if i.Locations[idx].HardAllocated < quantity {
				return errors.New("insufficient hard allocated quantity")
			}

			i.Locations[idx].HardAllocated -= quantity
			i.Locations[idx].Available += quantity
			break
		}
	}

	i.HardAllocatedQuantity -= quantity
	i.AvailableQuantity += quantity
	i.UpdatedAt = time.Now()

	return nil
}

// ShipStock ships hard allocated stock (removes from inventory)
func (i *InventoryItemRefactored) ShipStock(locationID string, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Reduce total quantity (inventory leaves warehouse)
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			if i.Locations[idx].HardAllocated < quantity {
				return errors.New("insufficient hard allocated quantity")
			}

			i.Locations[idx].Quantity -= quantity
			i.Locations[idx].HardAllocated -= quantity
			break
		}
	}

	i.TotalQuantity -= quantity
	i.HardAllocatedQuantity -= quantity
	i.UpdatedAt = time.Now()

	// Check for low stock
	if i.AvailableQuantity <= i.ReorderPoint {
		i.AddDomainEvent(&LowStockAlertEvent{
			SKU:             i.SKU,
			CurrentQuantity: i.AvailableQuantity,
			ReorderPoint:    i.ReorderPoint,
			AlertedAt:       time.Now(),
		})
	}

	return nil
}

// AdjustStock adjusts stock quantity (for cycle counts, corrections)
func (i *InventoryItemRefactored) AdjustStock(locationID string, newQuantity int) error {
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			oldQty := i.Locations[idx].Quantity
			diff := newQuantity - oldQty

			i.Locations[idx].Quantity = newQuantity
			i.Locations[idx].Available = newQuantity - i.Locations[idx].Reserved - i.Locations[idx].HardAllocated
			i.TotalQuantity += diff
			i.AvailableQuantity += diff

			i.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrLocationNotFound
}

// RecordShortage records a confirmed stock shortage
func (i *InventoryItemRefactored) RecordShortage(locationID string, shortageQty int) error {
	if shortageQty <= 0 {
		return ErrNoShortageToRecord
	}

	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			i.Locations[idx].Quantity -= shortageQty
			i.Locations[idx].Available -= shortageQty
			if i.Locations[idx].Available < 0 {
				overReserved := -i.Locations[idx].Available
				i.Locations[idx].Reserved -= overReserved
				i.Locations[idx].Available = 0
				i.ReservedQuantity -= overReserved
			}

			i.TotalQuantity -= shortageQty
			i.AvailableQuantity -= shortageQty
			if i.AvailableQuantity < 0 {
				i.AvailableQuantity = 0
			}

			i.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrLocationNotFound
}

// UpdatePickFrequency updates the pick frequency and recalculates velocity class
func (i *InventoryItemRefactored) UpdatePickFrequency(picksPerWeek int) {
	oldClass := i.VelocityClass
	i.PickFrequency = picksPerWeek
	i.VelocityClass = i.CalculateVelocityClass()
	now := time.Now()
	i.LastPickedAt = &now
	i.UpdatedAt = now

	if oldClass != i.VelocityClass {
		i.AddDomainEvent(&VelocityClassChangedEvent{
			SKU:           i.SKU,
			OldClass:      string(oldClass),
			NewClass:      string(i.VelocityClass),
			PickFrequency: i.PickFrequency,
			ChangedAt:     now,
		})
	}
}

// CalculateVelocityClass determines the velocity class based on pick frequency
func (i *InventoryItemRefactored) CalculateVelocityClass() VelocityClass {
	if i.PickFrequency > VelocityAThreshold {
		return VelocityA
	} else if i.PickFrequency >= VelocityBThreshold {
		return VelocityB
	}
	return VelocityC
}

// RecordCycleCount records a cycle count
func (i *InventoryItemRefactored) RecordCycleCount() {
	now := time.Now()
	i.LastCycleCount = &now
	i.UpdatedAt = now
}

// SetStorageStrategy sets the storage strategy
func (i *InventoryItemRefactored) SetStorageStrategy(strategy StorageStrategy) {
	if strategy.IsValid() {
		i.StorageStrategy = strategy
		i.UpdatedAt = time.Now()
	}
}

// RecordStow records when an item was stowed
func (i *InventoryItemRefactored) RecordStow() {
	now := time.Now()
	i.LastStowedAt = &now
	i.UpdatedAt = now
}

// GetLocationStock returns stock at a specific location
func (i *InventoryItemRefactored) GetLocationStock(locationID string) *StockLocation {
	for _, loc := range i.Locations {
		if loc.LocationID == locationID {
			return &loc
		}
	}
	return nil
}

// GetAvailableLocations returns locations with available stock
func (i *InventoryItemRefactored) GetAvailableLocations() []StockLocation {
	available := make([]StockLocation, 0)
	for _, loc := range i.Locations {
		if loc.Available > 0 {
			available = append(available, loc)
		}
	}
	return available
}

// Domain event methods
func (i *InventoryItemRefactored) AddDomainEvent(event DomainEvent) {
	i.DomainEvents = append(i.DomainEvents, event)
}

func (i *InventoryItemRefactored) ClearDomainEvents() {
	i.DomainEvents = make([]DomainEvent, 0)
}

func (i *InventoryItemRefactored) GetDomainEvents() []DomainEvent {
	return i.DomainEvents
}

// Validation methods
func (i *InventoryItemRefactored) IsHighVelocity() bool {
	return i.VelocityClass == VelocityA
}

func (i *InventoryItemRefactored) IsMediumVelocity() bool {
	return i.VelocityClass == VelocityB
}

func (i *InventoryItemRefactored) IsLowVelocity() bool {
	return i.VelocityClass == VelocityC
}

func (i *InventoryItemRefactored) IsLowStock() bool {
	return i.AvailableQuantity <= i.ReorderPoint
}
