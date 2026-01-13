package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrReservationExpired     = errors.New("reservation has expired")
	ErrReservationNotActive   = errors.New("reservation is not active")
	ErrReservationAlreadyUsed = errors.New("reservation has already been used")
)

// ReservationStatus represents the status of a reservation
type ReservationStatus string

const (
	ReservationStatusActive    ReservationStatus = "active"
	ReservationStatusStaged    ReservationStatus = "staged"
	ReservationStatusFulfilled ReservationStatus = "fulfilled"
	ReservationStatusCancelled ReservationStatus = "cancelled"
	ReservationStatusExpired   ReservationStatus = "expired"
)

// InventoryReservationAggregate represents a stock reservation as a separate aggregate
// This prevents unbounded growth of reservations array in InventoryItem
type InventoryReservationAggregate struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	ReservationID string             `bson:"reservationId"`
	SKU           string             `bson:"sku"` // Reference to inventory item

	// Multi-tenant fields
	TenantID    string `bson:"tenantId"`
	FacilityID  string `bson:"facilityId"`
	WarehouseID string `bson:"warehouseId"`
	SellerID    string `bson:"sellerId,omitempty"`

	OrderID    string            `bson:"orderId"`
	Quantity   int               `bson:"quantity"`
	LocationID string            `bson:"locationId"`
	Status     ReservationStatus `bson:"status"`
	UnitIDs    []string          `bson:"unitIds,omitempty"` // Specific units reserved for unit-level tracking

	CreatedAt time.Time `bson:"createdAt"`
	ExpiresAt time.Time `bson:"expiresAt"`
	UpdatedAt time.Time `bson:"updatedAt"`

	// Optional: Track who/what created/updated the reservation
	CreatedBy string `bson:"createdBy,omitempty"`
	UpdatedBy string `bson:"updatedBy,omitempty"`

	DomainEvents []DomainEvent `bson:"-"`
}

// ReservationTenantInfo holds multi-tenant identification for reservations
type ReservationTenantInfo struct {
	TenantID    string
	FacilityID  string
	WarehouseID string
	SellerID    string
}

// NewInventoryReservation creates a new inventory reservation aggregate
func NewInventoryReservation(
	reservationID string,
	sku string,
	orderID string,
	locationID string,
	quantity int,
	unitIDs []string,
	createdBy string,
	tenant *ReservationTenantInfo,
) *InventoryReservationAggregate {
	now := time.Now()
	reservation := &InventoryReservationAggregate{
		ReservationID: reservationID,
		SKU:           sku,
		OrderID:       orderID,
		Quantity:      quantity,
		LocationID:    locationID,
		Status:        ReservationStatusActive,
		UnitIDs:       unitIDs,
		CreatedAt:     now,
		ExpiresAt:     now.Add(24 * time.Hour), // Default 24hr expiration
		UpdatedAt:     now,
		CreatedBy:     createdBy,
		DomainEvents:  make([]DomainEvent, 0),
	}

	if tenant != nil {
		reservation.TenantID = tenant.TenantID
		reservation.FacilityID = tenant.FacilityID
		reservation.WarehouseID = tenant.WarehouseID
		reservation.SellerID = tenant.SellerID
	}

	return reservation
}

// MarkStaged marks the reservation as staged (hard allocated)
func (r *InventoryReservationAggregate) MarkStaged(updatedBy string) error {
	if r.Status != ReservationStatusActive {
		return ErrReservationNotActive
	}

	if time.Now().After(r.ExpiresAt) {
		return ErrReservationExpired
	}

	r.Status = ReservationStatusStaged
	r.UpdatedAt = time.Now()
	r.UpdatedBy = updatedBy

	return nil
}

// MarkFulfilled marks the reservation as fulfilled (shipped)
func (r *InventoryReservationAggregate) MarkFulfilled(updatedBy string) error {
	if r.Status != ReservationStatusStaged && r.Status != ReservationStatusActive {
		return ErrReservationAlreadyUsed
	}

	r.Status = ReservationStatusFulfilled
	r.UpdatedAt = time.Now()
	r.UpdatedBy = updatedBy

	return nil
}

// Cancel cancels the reservation
func (r *InventoryReservationAggregate) Cancel(updatedBy string, reason string) error {
	if r.Status == ReservationStatusFulfilled {
		return errors.New("cannot cancel fulfilled reservation")
	}

	r.Status = ReservationStatusCancelled
	r.UpdatedAt = time.Now()
	r.UpdatedBy = updatedBy

	return nil
}

// MarkExpired marks the reservation as expired
func (r *InventoryReservationAggregate) MarkExpired() {
	if r.Status == ReservationStatusActive && time.Now().After(r.ExpiresAt) {
		r.Status = ReservationStatusExpired
		r.UpdatedAt = time.Now()
	}
}

// IsActive returns true if the reservation is active and not expired
func (r *InventoryReservationAggregate) IsActive() bool {
	return r.Status == ReservationStatusActive && time.Now().Before(r.ExpiresAt)
}

// IsExpired returns true if the reservation has expired
func (r *InventoryReservationAggregate) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// ExtendExpiration extends the reservation expiration time
func (r *InventoryReservationAggregate) ExtendExpiration(duration time.Duration) error {
	if r.Status != ReservationStatusActive {
		return ErrReservationNotActive
	}

	r.ExpiresAt = r.ExpiresAt.Add(duration)
	r.UpdatedAt = time.Now()

	return nil
}

// AddDomainEvent adds a domain event
func (r *InventoryReservationAggregate) AddDomainEvent(event DomainEvent) {
	r.DomainEvents = append(r.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (r *InventoryReservationAggregate) ClearDomainEvents() {
	r.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (r *InventoryReservationAggregate) GetDomainEvents() []DomainEvent {
	return r.DomainEvents
}
