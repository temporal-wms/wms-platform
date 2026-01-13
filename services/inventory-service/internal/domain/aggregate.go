package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors
var (
	ErrInsufficientStock           = errors.New("insufficient stock")
	ErrInvalidQuantity             = errors.New("invalid quantity")
	ErrReservationNotFound         = errors.New("reservation not found")
	ErrAllocationNotFound          = errors.New("hard allocation not found")
	ErrAlreadyHardAllocated        = errors.New("reservation already hard allocated")
	ErrCannotReleaseHardAllocation = errors.New("cannot release hard allocation without physical return")
	ErrInvalidAllocationStatus     = errors.New("invalid allocation status")
	ErrLocationNotFound            = errors.New("location not found")
	ErrNoShortageToRecord          = errors.New("actual quantity >= expected quantity, no shortage")
)

// VelocityClass represents the pick frequency classification (ABC analysis)
type VelocityClass string

const (
	VelocityA VelocityClass = "A" // High velocity (>50 picks/week)
	VelocityB VelocityClass = "B" // Medium velocity (10-50 picks/week)
	VelocityC VelocityClass = "C" // Low velocity (<10 picks/week)
)

// IsValid checks if the velocity class is valid
func (v VelocityClass) IsValid() bool {
	switch v {
	case VelocityA, VelocityB, VelocityC:
		return true
	default:
		return false
	}
}

// StorageStrategy represents the storage placement strategy
type StorageStrategy string

const (
	StorageChaotic  StorageStrategy = "chaotic"  // Random placement (Amazon-style)
	StorageDirected StorageStrategy = "directed" // System-assigned locations
	StorageVelocity StorageStrategy = "velocity" // Placement based on pick frequency
)

// IsValid checks if the storage strategy is valid
func (s StorageStrategy) IsValid() bool {
	switch s {
	case StorageChaotic, StorageDirected, StorageVelocity:
		return true
	default:
		return false
	}
}

// VelocityThresholds for ABC classification
const (
	VelocityAThreshold = 50 // >50 picks per week = A class
	VelocityBThreshold = 10 // 10-50 picks per week = B class
	// <10 picks per week = C class
)

// InventoryItem is the aggregate root for the Inventory bounded context
type InventoryItem struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	SKU         string             `bson:"sku"`
	ProductName string             `bson:"productName"`

	// Multi-tenant fields for 3PL/FBA-style operations
	TenantID    string `bson:"tenantId" json:"tenantId"`       // 3PL operator identifier
	FacilityID  string `bson:"facilityId" json:"facilityId"`   // Physical facility/warehouse complex
	WarehouseID string `bson:"warehouseId" json:"warehouseId"` // Specific warehouse within facility
	SellerID    string `bson:"sellerId,omitempty" json:"sellerId,omitempty"` // Merchant/seller who owns this inventory

	Locations []StockLocation `bson:"locations"`
	TotalQuantity         int                    `bson:"totalQuantity"`
	ReservedQuantity      int                    `bson:"reservedQuantity"`
	HardAllocatedQuantity int                    `bson:"hardAllocatedQuantity"`
	AvailableQuantity     int                    `bson:"availableQuantity"`
	ReorderPoint          int                    `bson:"reorderPoint"`
	ReorderQuantity       int                    `bson:"reorderQuantity"`
	Reservations          []Reservation          `bson:"reservations"`
	HardAllocations       []HardAllocation       `bson:"hardAllocations"`
	Transactions          []InventoryTransaction `bson:"transactions,omitempty"`
	LastCycleCount        *time.Time             `bson:"lastCycleCount,omitempty"`
	// Velocity and storage fields (Amazon-style optimization)
	VelocityClass   VelocityClass   `bson:"velocityClass" json:"velocityClass"`
	StorageStrategy StorageStrategy `bson:"storageStrategy" json:"storageStrategy"`
	PickFrequency   int             `bson:"pickFrequency" json:"pickFrequency"`     // picks per week
	LastStowedAt    *time.Time      `bson:"lastStowedAt,omitempty" json:"lastStowedAt,omitempty"`
	LastPickedAt    *time.Time      `bson:"lastPickedAt,omitempty" json:"lastPickedAt,omitempty"`
	CreatedAt       time.Time       `bson:"createdAt"`
	UpdatedAt       time.Time       `bson:"updatedAt"`
	DomainEvents    []DomainEvent   `bson:"-"`
}

// StockLocation represents inventory at a specific location
type StockLocation struct {
	LocationID    string `bson:"locationId"`
	Zone          string `bson:"zone"`
	Aisle         string `bson:"aisle"`
	Rack          int    `bson:"rack"`
	Level         int    `bson:"level"`
	Quantity      int    `bson:"quantity"`
	Reserved      int    `bson:"reserved"`
	HardAllocated int    `bson:"hardAllocated"`
	Available     int    `bson:"available"`
}

// Reservation represents a stock reservation for an order
type Reservation struct {
	ReservationID string    `bson:"reservationId"`
	OrderID       string    `bson:"orderId"`
	Quantity      int       `bson:"quantity"`
	LocationID    string    `bson:"locationId"`
	Status        string    `bson:"status"` // active, staged, fulfilled, cancelled
	UnitIDs       []string  `bson:"unitIds,omitempty"` // Specific units reserved for unit-level tracking
	CreatedAt     time.Time `bson:"createdAt"`
	ExpiresAt     time.Time `bson:"expiresAt"`
}

// HardAllocation represents physically staged/locked inventory
// Created when a picker physically moves items to a staging area
type HardAllocation struct {
	AllocationID      string     `bson:"allocationId"`
	ReservationID     string     `bson:"reservationId"`
	OrderID           string     `bson:"orderId"`
	Quantity          int        `bson:"quantity"`
	SourceLocationID  string     `bson:"sourceLocationId"`
	StagingLocationID string     `bson:"stagingLocationId"`
	Status            string     `bson:"status"` // staged, packed, shipped, returned
	UnitIDs           []string   `bson:"unitIds,omitempty"` // Specific units allocated for unit-level tracking
	StagedBy          string     `bson:"stagedBy"`
	PackedBy          string     `bson:"packedBy,omitempty"`
	CreatedAt         time.Time  `bson:"createdAt"`
	PackedAt          *time.Time `bson:"packedAt,omitempty"`
	ShippedAt         *time.Time `bson:"shippedAt,omitempty"`
}

// InventoryTransaction represents an inventory change
type InventoryTransaction struct {
	TransactionID string    `bson:"transactionId"`
	Type          string    `bson:"type"` // receive, pick, adjust, transfer
	Quantity      int       `bson:"quantity"`
	LocationID    string    `bson:"locationId"`
	ReferenceID   string    `bson:"referenceId"` // Order ID, PO ID, etc.
	Reason        string    `bson:"reason,omitempty"`
	CreatedAt     time.Time `bson:"createdAt"`
	CreatedBy     string    `bson:"createdBy"`
}

// InventoryTenantInfo holds multi-tenant identification for inventory
type InventoryTenantInfo struct {
	TenantID    string
	FacilityID  string
	WarehouseID string
	SellerID    string
}

// NewInventoryItem creates a new InventoryItem aggregate (backward compatible, uses default tenant)
func NewInventoryItem(sku, productName string, reorderPoint, reorderQty int) *InventoryItem {
	return NewInventoryItemWithTenant(sku, productName, reorderPoint, reorderQty, nil)
}

// NewInventoryItemWithTenant creates a new InventoryItem aggregate with tenant context
func NewInventoryItemWithTenant(sku, productName string, reorderPoint, reorderQty int, tenant *InventoryTenantInfo) *InventoryItem {
	now := time.Now()
	item := &InventoryItem{
		SKU:                   sku,
		ProductName:           productName,
		Locations:             make([]StockLocation, 0),
		TotalQuantity:         0,
		ReservedQuantity:      0,
		HardAllocatedQuantity: 0,
		AvailableQuantity:     0,
		ReorderPoint:          reorderPoint,
		ReorderQuantity:       reorderQty,
		Reservations:          make([]Reservation, 0),
		HardAllocations:       make([]HardAllocation, 0),
		Transactions:          make([]InventoryTransaction, 0),
		CreatedAt:             now,
		UpdatedAt:             now,
		DomainEvents:          make([]DomainEvent, 0),
	}

	// Set tenant information
	if tenant != nil {
		item.TenantID = tenant.TenantID
		item.FacilityID = tenant.FacilityID
		item.WarehouseID = tenant.WarehouseID
		item.SellerID = tenant.SellerID
	} else {
		// Default tenant for backward compatibility
		item.TenantID = "DEFAULT_TENANT"
		item.FacilityID = "DEFAULT_FACILITY"
		item.WarehouseID = "DEFAULT_WAREHOUSE"
	}

	return item
}

// ReceiveStock adds stock to a location
func (i *InventoryItem) ReceiveStock(locationID, zone string, quantity int, referenceID, createdBy string) error {
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

	// Record transaction
	i.Transactions = append(i.Transactions, InventoryTransaction{
		TransactionID: generateTransactionID(),
		Type:          "receive",
		Quantity:      quantity,
		LocationID:    locationID,
		ReferenceID:   referenceID,
		CreatedAt:     time.Now(),
		CreatedBy:     createdBy,
	})

	i.AddDomainEvent(&InventoryReceivedEvent{
		SKU:        i.SKU,
		Quantity:   quantity,
		LocationID: locationID,
		ReceivedAt: time.Now(),
	})

	return nil
}

// Reserve reserves stock for an order
func (i *InventoryItem) Reserve(orderID, locationID string, quantity int) error {
	return i.ReserveWithUnits(orderID, locationID, quantity, nil)
}

// ReserveWithUnits reserves stock for an order with specific unit IDs
func (i *InventoryItem) ReserveWithUnits(orderID, locationID string, quantity int, unitIDs []string) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Check availability at location
	foundLocation := false
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			foundLocation = true
			if i.Locations[idx].Available < quantity {
				return ErrInsufficientStock
			}

			i.Locations[idx].Reserved += quantity
			i.Locations[idx].Available -= quantity
			break
		}
	}
	if !foundLocation {
		return ErrLocationNotFound
	}

	i.ReservedQuantity += quantity
	i.AvailableQuantity -= quantity

	reservation := Reservation{
		ReservationID: generateReservationID(),
		OrderID:       orderID,
		Quantity:      quantity,
		LocationID:    locationID,
		Status:        "active",
		UnitIDs:       unitIDs,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}
	i.Reservations = append(i.Reservations, reservation)
	i.UpdatedAt = time.Now()

	i.AddDomainEvent(&InventoryReservedEvent{
		SKU:         i.SKU,
		OrderID:     orderID,
		LocationID:  locationID,
		Quantity:    quantity,
		ReservedAt:  time.Now(),
	})

	return nil
}

// Pick picks stock (fulfills reservation)
func (i *InventoryItem) Pick(orderID, locationID string, quantity int, createdBy string) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Find and update reservation
	reservationIdx := -1
	for idx := range i.Reservations {
		if i.Reservations[idx].OrderID == orderID && i.Reservations[idx].LocationID == locationID {
			reservationIdx = idx
			break
		}
	}
	if reservationIdx == -1 {
		return ErrReservationNotFound
	}
	if i.Reservations[reservationIdx].Quantity < quantity {
		return ErrInsufficientStock
	}

	// Update location quantities
	locationIdx := -1
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			locationIdx = idx
			break
		}
	}
	if locationIdx == -1 {
		return ErrLocationNotFound
	}

	if i.Reservations[reservationIdx].Quantity == quantity {
		i.Reservations[reservationIdx].Status = "fulfilled"
	} else {
		i.Reservations[reservationIdx].Quantity -= quantity
	}

	i.Locations[locationIdx].Quantity -= quantity
	i.Locations[locationIdx].Reserved -= quantity

	i.TotalQuantity -= quantity
	i.ReservedQuantity -= quantity
	i.UpdatedAt = time.Now()

	// Record transaction
	i.Transactions = append(i.Transactions, InventoryTransaction{
		TransactionID: generateTransactionID(),
		Type:          "pick",
		Quantity:      -quantity,
		LocationID:    locationID,
		ReferenceID:   orderID,
		CreatedAt:     time.Now(),
		CreatedBy:     createdBy,
	})

	// Check if we need to trigger reorder
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

// ReleaseReservation releases a reservation
func (i *InventoryItem) ReleaseReservation(orderID string) error {
	for idx := range i.Reservations {
		if i.Reservations[idx].OrderID == orderID && i.Reservations[idx].Status == "active" {
			reservation := &i.Reservations[idx]
			reservation.Status = "cancelled"

			// Return stock to available
			for locIdx := range i.Locations {
				if i.Locations[locIdx].LocationID == reservation.LocationID {
					i.Locations[locIdx].Reserved -= reservation.Quantity
					i.Locations[locIdx].Available += reservation.Quantity
					break
				}
			}

			i.ReservedQuantity -= reservation.Quantity
			i.AvailableQuantity += reservation.Quantity
			i.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrReservationNotFound
}

// Adjust adjusts stock quantity (for cycle counts, corrections)
func (i *InventoryItem) Adjust(locationID string, newQuantity int, reason, createdBy string) error {
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			oldQty := i.Locations[idx].Quantity
			diff := newQuantity - oldQty

			i.Locations[idx].Quantity = newQuantity
			i.Locations[idx].Available = newQuantity - i.Locations[idx].Reserved
			i.TotalQuantity += diff
			i.AvailableQuantity += diff

			// Record transaction
			i.Transactions = append(i.Transactions, InventoryTransaction{
				TransactionID: generateTransactionID(),
				Type:          "adjust",
				Quantity:      diff,
				LocationID:    locationID,
				Reason:        reason,
				CreatedAt:     time.Now(),
				CreatedBy:     createdBy,
			})

			i.AddDomainEvent(&InventoryAdjustedEvent{
				SKU:         i.SKU,
				LocationID:  locationID,
				OldQuantity: oldQty,
				NewQuantity: newQuantity,
				Reason:      reason,
				AdjustedAt:  time.Now(),
			})

			i.UpdatedAt = time.Now()
			return nil
		}
	}
	return errors.New("location not found")
}

// RecordCycleCount records a cycle count
func (i *InventoryItem) RecordCycleCount() {
	now := time.Now()
	i.LastCycleCount = &now
	i.UpdatedAt = now
}

// GetLocationStock returns stock at a specific location
func (i *InventoryItem) GetLocationStock(locationID string) *StockLocation {
	for _, loc := range i.Locations {
		if loc.LocationID == locationID {
			return &loc
		}
	}
	return nil
}

// GetAvailableLocations returns locations with available stock
func (i *InventoryItem) GetAvailableLocations() []StockLocation {
	available := make([]StockLocation, 0)
	for _, loc := range i.Locations {
		if loc.Available > 0 {
			available = append(available, loc)
		}
	}
	return available
}

// UpdatePickFrequency updates the pick frequency and recalculates velocity class
func (i *InventoryItem) UpdatePickFrequency(picksPerWeek int) {
	i.PickFrequency = picksPerWeek
	i.VelocityClass = i.CalculateVelocityClass()
	now := time.Now()
	i.LastPickedAt = &now
	i.UpdatedAt = now
}

// CalculateVelocityClass determines the velocity class based on pick frequency
func (i *InventoryItem) CalculateVelocityClass() VelocityClass {
	if i.PickFrequency > VelocityAThreshold {
		return VelocityA
	} else if i.PickFrequency >= VelocityBThreshold {
		return VelocityB
	}
	return VelocityC
}

// IncrementPickFrequency increments the pick counter and updates velocity
func (i *InventoryItem) IncrementPickFrequency() {
	i.PickFrequency++
	oldClass := i.VelocityClass
	i.VelocityClass = i.CalculateVelocityClass()
	now := time.Now()
	i.LastPickedAt = &now
	i.UpdatedAt = now

	// Emit event if velocity class changed
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

// SetStorageStrategy sets the storage strategy for this item
func (i *InventoryItem) SetStorageStrategy(strategy StorageStrategy) {
	if strategy.IsValid() {
		i.StorageStrategy = strategy
		i.UpdatedAt = time.Now()
	}
}

// RecordStow records when an item was stowed
func (i *InventoryItem) RecordStow() {
	now := time.Now()
	i.LastStowedAt = &now
	i.UpdatedAt = now
}

// IsHighVelocity returns true if item is class A (high velocity)
func (i *InventoryItem) IsHighVelocity() bool {
	return i.VelocityClass == VelocityA
}

// IsMediumVelocity returns true if item is class B (medium velocity)
func (i *InventoryItem) IsMediumVelocity() bool {
	return i.VelocityClass == VelocityB
}

// IsLowVelocity returns true if item is class C (low velocity)
func (i *InventoryItem) IsLowVelocity() bool {
	return i.VelocityClass == VelocityC
}

// AddDomainEvent adds a domain event
func (i *InventoryItem) AddDomainEvent(event DomainEvent) {
	i.DomainEvents = append(i.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (i *InventoryItem) ClearDomainEvents() {
	i.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (i *InventoryItem) GetDomainEvents() []DomainEvent {
	return i.DomainEvents
}

func generateTransactionID() string {
	return "TXN-" + time.Now().Format("20060102150405")
}

func generateReservationID() string {
	return "RES-" + time.Now().Format("20060102150405")
}

func generateAllocationID() string {
	return "ALLOC-" + time.Now().Format("20060102150405.000")
}

// Stage converts a soft reservation to a hard allocation (physical staging)
// This is called when picker physically moves items to staging area
func (i *InventoryItem) Stage(reservationID, stagingLocationID, stagedBy string) error {
	// Find the active reservation
	var reservation *Reservation
	var reservationIdx int
	for idx := range i.Reservations {
		if i.Reservations[idx].ReservationID == reservationID && i.Reservations[idx].Status == "active" {
			reservation = &i.Reservations[idx]
			reservationIdx = idx
			break
		}
	}

	if reservation == nil {
		return ErrReservationNotFound
	}

	// Check if already hard allocated
	for _, alloc := range i.HardAllocations {
		if alloc.ReservationID == reservationID && alloc.Status != "returned" {
			return ErrAlreadyHardAllocated
		}
	}

	// Update reservation status to "staged"
	i.Reservations[reservationIdx].Status = "staged"

	// Move quantity from Reserved to HardAllocated at location level
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == reservation.LocationID {
			i.Locations[idx].Reserved -= reservation.Quantity
			i.Locations[idx].HardAllocated += reservation.Quantity
			break
		}
	}

	// Update aggregate counters
	i.ReservedQuantity -= reservation.Quantity
	i.HardAllocatedQuantity += reservation.Quantity

	// Create hard allocation (copy unit IDs from reservation for unit-level tracking)
	allocation := HardAllocation{
		AllocationID:      generateAllocationID(),
		ReservationID:     reservationID,
		OrderID:           reservation.OrderID,
		Quantity:          reservation.Quantity,
		SourceLocationID:  reservation.LocationID,
		StagingLocationID: stagingLocationID,
		Status:            "staged",
		UnitIDs:           reservation.UnitIDs,
		StagedBy:          stagedBy,
		CreatedAt:         time.Now(),
	}
	i.HardAllocations = append(i.HardAllocations, allocation)
	i.UpdatedAt = time.Now()

	// Emit domain event
	i.AddDomainEvent(&InventoryStagedEvent{
		SKU:               i.SKU,
		AllocationID:      allocation.AllocationID,
		OrderID:           reservation.OrderID,
		Quantity:          reservation.Quantity,
		SourceLocationID:  reservation.LocationID,
		StagingLocationID: stagingLocationID,
		StagedBy:          stagedBy,
		StagedAt:          time.Now(),
	})

	return nil
}

// Pack marks a hard allocation as packed (ready for shipping)
func (i *InventoryItem) Pack(allocationID, packedBy string) error {
	for idx := range i.HardAllocations {
		if i.HardAllocations[idx].AllocationID == allocationID {
			if i.HardAllocations[idx].Status != "staged" {
				return ErrInvalidAllocationStatus
			}
			now := time.Now()
			i.HardAllocations[idx].Status = "packed"
			i.HardAllocations[idx].PackedBy = packedBy
			i.HardAllocations[idx].PackedAt = &now
			i.UpdatedAt = time.Now()

			i.AddDomainEvent(&InventoryPackedEvent{
				SKU:          i.SKU,
				AllocationID: allocationID,
				OrderID:      i.HardAllocations[idx].OrderID,
				PackedBy:     packedBy,
				PackedAt:     now,
			})
			return nil
		}
	}
	return ErrAllocationNotFound
}

// Ship marks a hard allocation as shipped and removes inventory
func (i *InventoryItem) Ship(allocationID string) error {
	for idx := range i.HardAllocations {
		if i.HardAllocations[idx].AllocationID == allocationID {
			if i.HardAllocations[idx].Status != "packed" {
				return ErrInvalidAllocationStatus
			}
			allocation := &i.HardAllocations[idx]
			now := time.Now()
			allocation.Status = "shipped"
			allocation.ShippedAt = &now

			// Reduce total quantity (inventory leaves warehouse)
			i.TotalQuantity -= allocation.Quantity
			i.HardAllocatedQuantity -= allocation.Quantity

			// Update source location
			for locIdx := range i.Locations {
				if i.Locations[locIdx].LocationID == allocation.SourceLocationID {
					i.Locations[locIdx].Quantity -= allocation.Quantity
					i.Locations[locIdx].HardAllocated -= allocation.Quantity
					break
				}
			}

			// Mark associated reservation as fulfilled
			for resIdx := range i.Reservations {
				if i.Reservations[resIdx].ReservationID == allocation.ReservationID {
					i.Reservations[resIdx].Status = "fulfilled"
					break
				}
			}

			i.UpdatedAt = time.Now()

			// Record transaction
			i.Transactions = append(i.Transactions, InventoryTransaction{
				TransactionID: generateTransactionID(),
				Type:          "ship",
				Quantity:      -allocation.Quantity,
				LocationID:    allocation.SourceLocationID,
				ReferenceID:   allocation.OrderID,
				CreatedAt:     time.Now(),
				CreatedBy:     "system",
			})

			i.AddDomainEvent(&InventoryShippedEvent{
				SKU:          i.SKU,
				AllocationID: allocationID,
				OrderID:      allocation.OrderID,
				Quantity:     allocation.Quantity,
				ShippedAt:    now,
			})

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
	}
	return ErrAllocationNotFound
}

// ReturnToShelf returns a hard allocated item back to shelf
// This is required to release hard allocations (physical return needed)
func (i *InventoryItem) ReturnToShelf(allocationID, returnedBy, reason string) error {
	for idx := range i.HardAllocations {
		if i.HardAllocations[idx].AllocationID == allocationID {
			allocation := &i.HardAllocations[idx]

			if allocation.Status == "shipped" {
				return errors.New("cannot return shipped inventory")
			}

			// Move quantity back from HardAllocated to Available
			for locIdx := range i.Locations {
				if i.Locations[locIdx].LocationID == allocation.SourceLocationID {
					i.Locations[locIdx].HardAllocated -= allocation.Quantity
					i.Locations[locIdx].Available += allocation.Quantity
					break
				}
			}

			// Update aggregate counters
			i.HardAllocatedQuantity -= allocation.Quantity
			i.AvailableQuantity += allocation.Quantity

			// Mark allocation as returned
			allocation.Status = "returned"

			// Cancel the associated reservation
			for resIdx := range i.Reservations {
				if i.Reservations[resIdx].ReservationID == allocation.ReservationID {
					i.Reservations[resIdx].Status = "cancelled"
					break
				}
			}

			i.UpdatedAt = time.Now()

			// Record transaction
			i.Transactions = append(i.Transactions, InventoryTransaction{
				TransactionID: generateTransactionID(),
				Type:          "return_to_shelf",
				Quantity:      allocation.Quantity,
				LocationID:    allocation.SourceLocationID,
				ReferenceID:   allocation.OrderID,
				Reason:        reason,
				CreatedAt:     time.Now(),
				CreatedBy:     returnedBy,
			})

			i.AddDomainEvent(&InventoryReturnedToShelfEvent{
				SKU:              i.SKU,
				AllocationID:     allocationID,
				OrderID:          allocation.OrderID,
				Quantity:         allocation.Quantity,
				SourceLocationID: allocation.SourceLocationID,
				ReturnedBy:       returnedBy,
				Reason:           reason,
				ReturnedAt:       time.Now(),
			})

			return nil
		}
	}
	return ErrAllocationNotFound
}

// GetHardAllocation returns a hard allocation by ID
func (i *InventoryItem) GetHardAllocation(allocationID string) *HardAllocation {
	for _, alloc := range i.HardAllocations {
		if alloc.AllocationID == allocationID {
			return &alloc
		}
	}
	return nil
}

// GetActiveHardAllocations returns all non-shipped/returned allocations
func (i *InventoryItem) GetActiveHardAllocations() []HardAllocation {
	active := make([]HardAllocation, 0)
	for _, alloc := range i.HardAllocations {
		if alloc.Status == "staged" || alloc.Status == "packed" {
			active = append(active, alloc)
		}
	}
	return active
}

// RecordShortage records a confirmed stock shortage discovered during picking
// This adjusts inventory to match reality and emits events for audit/compensation
func (i *InventoryItem) RecordShortage(locationID, orderID string, expectedQty, actualQty int, reason, reportedBy string) error {
	// Find the location
	var loc *StockLocation
	var locIdx int
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			loc = &i.Locations[idx]
			locIdx = idx
			break
		}
	}

	if loc == nil {
		return ErrLocationNotFound
	}

	shortageQty := expectedQty - actualQty
	if shortageQty <= 0 {
		return ErrNoShortageToRecord
	}

	// Record what the system thought was there
	systemQuantity := loc.Quantity

	// Adjust the location quantity to match reality
	// We reduce total by shortage amount (the missing inventory)
	i.Locations[locIdx].Quantity -= shortageQty
	i.Locations[locIdx].Available -= shortageQty
	if i.Locations[locIdx].Available < 0 {
		// If we had reserved more than actually exists, adjust reserved too
		overReserved := -i.Locations[locIdx].Available
		i.Locations[locIdx].Reserved -= overReserved
		i.Locations[locIdx].Available = 0
		i.ReservedQuantity -= overReserved
	}

	// Update aggregate totals
	i.TotalQuantity -= shortageQty
	i.AvailableQuantity -= shortageQty
	if i.AvailableQuantity < 0 {
		i.AvailableQuantity = 0
	}

	i.UpdatedAt = time.Now()

	// Record transaction for audit trail
	i.Transactions = append(i.Transactions, InventoryTransaction{
		TransactionID: generateTransactionID(),
		Type:          "shortage",
		Quantity:      -shortageQty,
		LocationID:    locationID,
		ReferenceID:   orderID,
		Reason:        reason,
		CreatedAt:     time.Now(),
		CreatedBy:     reportedBy,
	})

	// Emit stock shortage event
	i.AddDomainEvent(&StockShortageEvent{
		SKU:              i.SKU,
		LocationID:       locationID,
		OrderID:          orderID,
		ExpectedQuantity: expectedQty,
		ActualQuantity:   actualQty,
		ShortageQuantity: shortageQty,
		ReportedBy:       reportedBy,
		Reason:           reason,
		OccurredAt_:      time.Now(),
	})

	// Emit discrepancy event for audit/reporting
	i.AddDomainEvent(&InventoryDiscrepancyEvent{
		SKU:             i.SKU,
		LocationID:      locationID,
		SystemQuantity:  systemQuantity,
		ActualQuantity:  actualQty,
		DiscrepancyType: "shortage",
		Source:          "picking",
		ReferenceID:     orderID,
		DetectedAt:      time.Now(),
	})

	// Check if we need to trigger low stock alert
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
