package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors
var (
	ErrConsolidationComplete = errors.New("consolidation is already complete")
	ErrItemNotExpected       = errors.New("item not expected in this consolidation")
	ErrAllItemsConsolidated  = errors.New("all items already consolidated")
)

// ConsolidationStatus represents the status of a consolidation unit
type ConsolidationStatus string

const (
	ConsolidationStatusPending    ConsolidationStatus = "pending"
	ConsolidationStatusInProgress ConsolidationStatus = "in_progress"
	ConsolidationStatusCompleted  ConsolidationStatus = "completed"
	ConsolidationStatusCancelled  ConsolidationStatus = "cancelled"
)

// ConsolidationStrategy represents the consolidation approach
type ConsolidationStrategy string

const (
	StrategyOrderBased   ConsolidationStrategy = "order"   // Group by single order
	StrategyCarrierBased ConsolidationStrategy = "carrier" // Group by shipping carrier
	StrategyRouteBased   ConsolidationStrategy = "route"   // Group by delivery route
	StrategyTimeBased    ConsolidationStrategy = "time"    // Group by shipping cutoff
)

// ConsolidationUnit is the aggregate root for the Consolidation bounded context
type ConsolidationUnit struct {
	ID                  primitive.ObjectID    `bson:"_id,omitempty"`
	ConsolidationID     string                `bson:"consolidationId"`
	OrderID             string                `bson:"orderId"`
	WaveID              string                `bson:"waveId"`
	Status              ConsolidationStatus   `bson:"status"`
	Strategy            ConsolidationStrategy `bson:"strategy"`
	ExpectedItems       []ExpectedItem        `bson:"expectedItems"`
	ConsolidatedItems   []ConsolidatedItem    `bson:"consolidatedItems"`
	SourceTotes         []string              `bson:"sourceTotes"`
	DestinationBin      string                `bson:"destinationBin"`
	Station             string                `bson:"station"`
	WorkerID            string                `bson:"workerId,omitempty"`
	TotalExpected       int                   `bson:"totalExpected"`
	TotalConsolidated   int                   `bson:"totalConsolidated"`
	ReadyForPacking     bool                  `bson:"readyForPacking"`
	CreatedAt           time.Time             `bson:"createdAt"`
	UpdatedAt           time.Time             `bson:"updatedAt"`
	StartedAt           *time.Time            `bson:"startedAt,omitempty"`
	CompletedAt         *time.Time            `bson:"completedAt,omitempty"`
	DomainEvents        []DomainEvent         `bson:"-"`
}

// ExpectedItem represents an item expected for consolidation
type ExpectedItem struct {
	SKU          string `bson:"sku"`
	ProductName  string `bson:"productName"`
	Quantity     int    `bson:"quantity"`
	SourceToteID string `bson:"sourceToteId"`
	Received     int    `bson:"received"`
	Status       string `bson:"status"` // pending, received, short
}

// ConsolidatedItem represents an item that has been consolidated
type ConsolidatedItem struct {
	SKU          string    `bson:"sku"`
	Quantity     int       `bson:"quantity"`
	SourceToteID string    `bson:"sourceToteId"`
	ScannedAt    time.Time `bson:"scannedAt"`
	VerifiedBy   string    `bson:"verifiedBy"`
}

// NewConsolidationUnit creates a new ConsolidationUnit aggregate
func NewConsolidationUnit(consolidationID, orderID, waveID string, strategy ConsolidationStrategy, items []ExpectedItem) (*ConsolidationUnit, error) {
	if len(items) == 0 {
		return nil, errors.New("consolidation must have at least one item")
	}

	now := time.Now()
	totalExpected := 0
	sourceTotes := make(map[string]bool)

	for i := range items {
		totalExpected += items[i].Quantity
		items[i].Status = "pending"
		items[i].Received = 0
		sourceTotes[items[i].SourceToteID] = true
	}

	toteList := make([]string, 0, len(sourceTotes))
	for tote := range sourceTotes {
		toteList = append(toteList, tote)
	}

	unit := &ConsolidationUnit{
		ConsolidationID:   consolidationID,
		OrderID:           orderID,
		WaveID:            waveID,
		Status:            ConsolidationStatusPending,
		Strategy:          strategy,
		ExpectedItems:     items,
		ConsolidatedItems: make([]ConsolidatedItem, 0),
		SourceTotes:       toteList,
		TotalExpected:     totalExpected,
		TotalConsolidated: 0,
		ReadyForPacking:   false,
		CreatedAt:         now,
		UpdatedAt:         now,
		DomainEvents:      make([]DomainEvent, 0),
	}

	unit.AddDomainEvent(&ConsolidationStartedEvent{
		ConsolidationID: consolidationID,
		OrderID:         orderID,
		ExpectedItems:   totalExpected,
		SourceTotes:     toteList,
		StartedAt:       now,
	})

	return unit, nil
}

// AssignStation assigns a consolidation station and worker
func (c *ConsolidationUnit) AssignStation(station, workerID, destinationBin string) error {
	if c.Status == ConsolidationStatusCompleted {
		return ErrConsolidationComplete
	}

	c.Station = station
	c.WorkerID = workerID
	c.DestinationBin = destinationBin
	c.UpdatedAt = time.Now()

	return nil
}

// Start marks the consolidation as in progress
func (c *ConsolidationUnit) Start() error {
	if c.Status != ConsolidationStatusPending {
		return errors.New("consolidation already started")
	}

	now := time.Now()
	c.Status = ConsolidationStatusInProgress
	c.StartedAt = &now
	c.UpdatedAt = now

	return nil
}

// ConsolidateItem records a consolidated item
func (c *ConsolidationUnit) ConsolidateItem(sku string, quantity int, sourceToteID, verifiedBy string) error {
	if c.Status == ConsolidationStatusCompleted {
		return ErrConsolidationComplete
	}

	if c.Status == ConsolidationStatusPending {
		if err := c.Start(); err != nil {
			return err
		}
	}

	// Find the expected item
	found := false
	for i := range c.ExpectedItems {
		if c.ExpectedItems[i].SKU == sku && c.ExpectedItems[i].SourceToteID == sourceToteID {
			found = true
			c.ExpectedItems[i].Received += quantity

			if c.ExpectedItems[i].Received >= c.ExpectedItems[i].Quantity {
				c.ExpectedItems[i].Status = "received"
			} else if c.ExpectedItems[i].Received > 0 {
				c.ExpectedItems[i].Status = "partial"
			}
			break
		}
	}

	if !found {
		return ErrItemNotExpected
	}

	// Record consolidated item
	now := time.Now()
	consolidated := ConsolidatedItem{
		SKU:          sku,
		Quantity:     quantity,
		SourceToteID: sourceToteID,
		ScannedAt:    now,
		VerifiedBy:   verifiedBy,
	}

	c.ConsolidatedItems = append(c.ConsolidatedItems, consolidated)
	c.TotalConsolidated += quantity
	c.UpdatedAt = now

	c.AddDomainEvent(&ItemConsolidatedEvent{
		ConsolidationID: c.ConsolidationID,
		SKU:             sku,
		Quantity:        quantity,
		SourceToteID:    sourceToteID,
		DestinationBin:  c.DestinationBin,
		ConsolidatedAt:  now,
	})

	// Check if all items are consolidated
	allConsolidated := true
	for _, item := range c.ExpectedItems {
		if item.Status != "received" {
			allConsolidated = false
			break
		}
	}

	if allConsolidated {
		return c.Complete()
	}

	return nil
}

// Complete marks the consolidation as completed
func (c *ConsolidationUnit) Complete() error {
	if c.Status == ConsolidationStatusCompleted {
		return ErrConsolidationComplete
	}

	now := time.Now()
	c.Status = ConsolidationStatusCompleted
	c.ReadyForPacking = true
	c.CompletedAt = &now
	c.UpdatedAt = now

	c.AddDomainEvent(&ConsolidationCompletedEvent{
		ConsolidationID:   c.ConsolidationID,
		OrderID:           c.OrderID,
		DestinationBin:    c.DestinationBin,
		TotalConsolidated: c.TotalConsolidated,
		ReadyForPacking:   true,
		CompletedAt:       now,
	})

	return nil
}

// MarkShort marks items that couldn't be fully consolidated
func (c *ConsolidationUnit) MarkShort(sku, sourceToteID string, shortQty int, reason string) error {
	for i := range c.ExpectedItems {
		if c.ExpectedItems[i].SKU == sku && c.ExpectedItems[i].SourceToteID == sourceToteID {
			c.ExpectedItems[i].Status = "short"
			c.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrItemNotExpected
}

// Cancel cancels the consolidation
func (c *ConsolidationUnit) Cancel(reason string) error {
	if c.Status == ConsolidationStatusCompleted {
		return ErrConsolidationComplete
	}

	c.Status = ConsolidationStatusCancelled
	c.UpdatedAt = time.Now()

	return nil
}

// GetProgress returns the completion percentage
func (c *ConsolidationUnit) GetProgress() float64 {
	if c.TotalExpected == 0 {
		return 0
	}
	return float64(c.TotalConsolidated) / float64(c.TotalExpected) * 100
}

// GetPendingItems returns items not yet consolidated
func (c *ConsolidationUnit) GetPendingItems() []ExpectedItem {
	pending := make([]ExpectedItem, 0)
	for _, item := range c.ExpectedItems {
		if item.Status == "pending" || item.Status == "partial" {
			pending = append(pending, item)
		}
	}
	return pending
}

// AddDomainEvent adds a domain event
func (c *ConsolidationUnit) AddDomainEvent(event DomainEvent) {
	c.DomainEvents = append(c.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (c *ConsolidationUnit) ClearDomainEvents() {
	c.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (c *ConsolidationUnit) GetDomainEvents() []DomainEvent {
	return c.DomainEvents
}
