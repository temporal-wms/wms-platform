package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InventoryTransactionAggregate represents an inventory change as a separate aggregate
// This allows for unbounded transaction history without bloating the InventoryItem document
type InventoryTransactionAggregate struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	TransactionID string             `bson:"transactionId"`
	SKU           string             `bson:"sku"` // Reference to inventory item

	// Multi-tenant fields
	TenantID    string `bson:"tenantId"`
	FacilityID  string `bson:"facilityId"`
	WarehouseID string `bson:"warehouseId"`
	SellerID    string `bson:"sellerId,omitempty"`

	Type        string `bson:"type"` // receive, pick, adjust, transfer, ship, shortage, return_to_shelf
	Quantity    int    `bson:"quantity"`
	LocationID  string `bson:"locationId"`
	ReferenceID string `bson:"referenceId"` // Order ID, PO ID, Allocation ID, etc.
	Reason      string `bson:"reason,omitempty"`
	CreatedAt   time.Time `bson:"createdAt"`
	CreatedBy   string    `bson:"createdBy"`

	DomainEvents []DomainEvent `bson:"-"`
}

// TransactionTenantInfo holds multi-tenant identification for transactions
type TransactionTenantInfo struct {
	TenantID    string
	FacilityID  string
	WarehouseID string
	SellerID    string
}

// NewInventoryTransaction creates a new inventory transaction aggregate
func NewInventoryTransaction(
	transactionID string,
	sku string,
	transactionType string,
	quantity int,
	locationID string,
	referenceID string,
	reason string,
	createdBy string,
	tenant *TransactionTenantInfo,
) *InventoryTransactionAggregate {
	txn := &InventoryTransactionAggregate{
		TransactionID: transactionID,
		SKU:           sku,
		Type:          transactionType,
		Quantity:      quantity,
		LocationID:    locationID,
		ReferenceID:   referenceID,
		Reason:        reason,
		CreatedAt:     time.Now(),
		CreatedBy:     createdBy,
		DomainEvents:  make([]DomainEvent, 0),
	}

	if tenant != nil {
		txn.TenantID = tenant.TenantID
		txn.FacilityID = tenant.FacilityID
		txn.WarehouseID = tenant.WarehouseID
		txn.SellerID = tenant.SellerID
	}

	return txn
}

// AddDomainEvent adds a domain event
func (t *InventoryTransactionAggregate) AddDomainEvent(event DomainEvent) {
	t.DomainEvents = append(t.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (t *InventoryTransactionAggregate) ClearDomainEvents() {
	t.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (t *InventoryTransactionAggregate) GetDomainEvents() []DomainEvent {
	return t.DomainEvents
}
