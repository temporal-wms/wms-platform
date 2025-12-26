package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors
var (
	ErrInsufficientStock  = errors.New("insufficient stock")
	ErrInvalidQuantity    = errors.New("invalid quantity")
	ErrReservationNotFound = errors.New("reservation not found")
)

// InventoryItem is the aggregate root for the Inventory bounded context
type InventoryItem struct {
	ID               primitive.ObjectID     `bson:"_id,omitempty"`
	SKU              string                 `bson:"sku"`
	ProductName      string                 `bson:"productName"`
	Locations        []StockLocation        `bson:"locations"`
	TotalQuantity    int                    `bson:"totalQuantity"`
	ReservedQuantity int                    `bson:"reservedQuantity"`
	AvailableQuantity int                   `bson:"availableQuantity"`
	ReorderPoint     int                    `bson:"reorderPoint"`
	ReorderQuantity  int                    `bson:"reorderQuantity"`
	Reservations     []Reservation          `bson:"reservations"`
	Transactions     []InventoryTransaction `bson:"transactions,omitempty"`
	LastCycleCount   *time.Time             `bson:"lastCycleCount,omitempty"`
	CreatedAt        time.Time              `bson:"createdAt"`
	UpdatedAt        time.Time              `bson:"updatedAt"`
	DomainEvents     []DomainEvent          `bson:"-"`
}

// StockLocation represents inventory at a specific location
type StockLocation struct {
	LocationID string `bson:"locationId"`
	Zone       string `bson:"zone"`
	Aisle      string `bson:"aisle"`
	Rack       int    `bson:"rack"`
	Level      int    `bson:"level"`
	Quantity   int    `bson:"quantity"`
	Reserved   int    `bson:"reserved"`
	Available  int    `bson:"available"`
}

// Reservation represents a stock reservation for an order
type Reservation struct {
	ReservationID string    `bson:"reservationId"`
	OrderID       string    `bson:"orderId"`
	Quantity      int       `bson:"quantity"`
	LocationID    string    `bson:"locationId"`
	Status        string    `bson:"status"` // active, fulfilled, cancelled
	CreatedAt     time.Time `bson:"createdAt"`
	ExpiresAt     time.Time `bson:"expiresAt"`
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

// NewInventoryItem creates a new InventoryItem aggregate
func NewInventoryItem(sku, productName string, reorderPoint, reorderQty int) *InventoryItem {
	now := time.Now()
	return &InventoryItem{
		SKU:              sku,
		ProductName:      productName,
		Locations:        make([]StockLocation, 0),
		TotalQuantity:    0,
		ReservedQuantity: 0,
		AvailableQuantity: 0,
		ReorderPoint:     reorderPoint,
		ReorderQuantity:  reorderQty,
		Reservations:     make([]Reservation, 0),
		Transactions:     make([]InventoryTransaction, 0),
		CreatedAt:        now,
		UpdatedAt:        now,
		DomainEvents:     make([]DomainEvent, 0),
	}
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
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Check availability at location
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			if i.Locations[idx].Available < quantity {
				return ErrInsufficientStock
			}

			i.Locations[idx].Reserved += quantity
			i.Locations[idx].Available -= quantity
			break
		}
	}

	i.ReservedQuantity += quantity
	i.AvailableQuantity -= quantity

	reservation := Reservation{
		ReservationID: generateReservationID(),
		OrderID:       orderID,
		Quantity:      quantity,
		LocationID:    locationID,
		Status:        "active",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}
	i.Reservations = append(i.Reservations, reservation)
	i.UpdatedAt = time.Now()

	return nil
}

// Pick picks stock (fulfills reservation)
func (i *InventoryItem) Pick(orderID, locationID string, quantity int, createdBy string) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Find and update reservation
	for idx := range i.Reservations {
		if i.Reservations[idx].OrderID == orderID && i.Reservations[idx].LocationID == locationID {
			i.Reservations[idx].Status = "fulfilled"
			break
		}
	}

	// Update location quantities
	for idx := range i.Locations {
		if i.Locations[idx].LocationID == locationID {
			i.Locations[idx].Quantity -= quantity
			i.Locations[idx].Reserved -= quantity
			break
		}
	}

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
