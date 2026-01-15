package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Problem solve errors
var (
	ErrProblemTicketNotFound      = errors.New("problem ticket not found")
	ErrInvalidProblemType         = errors.New("invalid problem type")
	ErrInvalidResolution          = errors.New("invalid resolution")
	ErrProblemAlreadyResolved     = errors.New("problem already resolved")
	ErrCannotResolveWithoutReason = errors.New("resolution requires reason or notes")
)

// ProblemType represents the type of problem encountered
type ProblemType string

const (
	ProblemTypeDamaged        ProblemType = "damaged"
	ProblemTypeShortage       ProblemType = "shortage"
	ProblemTypeOverage        ProblemType = "overage"
	ProblemTypeUnexpectedItem ProblemType = "unexpected_item"
	ProblemTypeWrongItem      ProblemType = "wrong_item"
	ProblemTypeNeedsPrep      ProblemType = "needs_prep"
	ProblemTypeMislabeled     ProblemType = "mislabeled"
	ProblemTypeQualityIssue   ProblemType = "quality_issue"
)

// IsValid checks if the problem type is valid
func (p ProblemType) IsValid() bool {
	switch p {
	case ProblemTypeDamaged, ProblemTypeShortage, ProblemTypeOverage,
		ProblemTypeUnexpectedItem, ProblemTypeWrongItem, ProblemTypeNeedsPrep,
		ProblemTypeMislabeled, ProblemTypeQualityIssue:
		return true
	default:
		return false
	}
}

// ProblemResolution represents the resolution status of a problem
type ProblemResolution string

const (
	ResolutionPending    ProblemResolution = "pending"
	ResolutionAccepted   ProblemResolution = "accepted"     // Accept as-is
	ResolutionRejected   ProblemResolution = "rejected"     // Return to vendor
	ResolutionAdjusted   ProblemResolution = "adjusted"     // Quantity adjusted
	ResolutionDisposed   ProblemResolution = "disposed"     // Item disposed
	ResolutionRepackaged ProblemResolution = "repackaged"   // Prep completed
	ResolutionInvestigate ProblemResolution = "investigate" // Needs further investigation
)

// IsValid checks if the resolution is valid
func (r ProblemResolution) IsValid() bool {
	switch r {
	case ResolutionPending, ResolutionAccepted, ResolutionRejected,
		ResolutionAdjusted, ResolutionDisposed, ResolutionRepackaged,
		ResolutionInvestigate:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the resolution can transition to another resolution
func (r ProblemResolution) CanTransitionTo(target ProblemResolution) bool {
	// Only pending can transition to other states
	if r == ResolutionPending {
		return target != ResolutionPending
	}
	// Investigate can transition to any non-pending state
	if r == ResolutionInvestigate {
		return target != ResolutionPending && target != ResolutionInvestigate
	}
	// All other states are terminal
	return false
}

// ProblemTicket is the aggregate root for problem solve
type ProblemTicket struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TicketID         string             `bson:"ticketId" json:"ticketId"`
	TenantID         string             `bson:"tenantId" json:"tenantId"`
	FacilityID       string             `bson:"facilityId" json:"facilityId"`
	WarehouseID      string             `bson:"warehouseId" json:"warehouseId"`
	ShipmentID       string             `bson:"shipmentId" json:"shipmentId"`
	SKU              string             `bson:"sku,omitempty" json:"sku,omitempty"`
	ProductName      string             `bson:"productName,omitempty" json:"productName,omitempty"`
	ProblemType      ProblemType        `bson:"problemType" json:"problemType"`
	Description      string             `bson:"description" json:"description"`
	Quantity         int                `bson:"quantity" json:"quantity"`
	AffectedUnitIDs  []string           `bson:"affectedUnitIds,omitempty" json:"affectedUnitIds,omitempty"`
	Resolution       ProblemResolution  `bson:"resolution" json:"resolution"`
	ResolutionNotes  string             `bson:"resolutionNotes,omitempty" json:"resolutionNotes,omitempty"`
	CreatedBy        string             `bson:"createdBy" json:"createdBy"`
	AssignedTo       string             `bson:"assignedTo,omitempty" json:"assignedTo,omitempty"`
	ResolvedBy       string             `bson:"resolvedBy,omitempty" json:"resolvedBy,omitempty"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	ResolvedAt       *time.Time         `bson:"resolvedAt,omitempty" json:"resolvedAt,omitempty"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`
	Priority         string             `bson:"priority" json:"priority"` // high, medium, low
	ImageURLs        []string           `bson:"imageUrls,omitempty" json:"imageUrls,omitempty"`
	DomainEvents     []DomainEvent      `bson:"-" json:"-"`
}

// NewProblemTicket creates a new problem ticket
func NewProblemTicket(
	ticketID string,
	shipmentID string,
	sku string,
	productName string,
	problemType ProblemType,
	description string,
	quantity int,
	createdBy string,
	priority string,
) (*ProblemTicket, error) {
	if !problemType.IsValid() {
		return nil, ErrInvalidProblemType
	}

	if description == "" {
		return nil, errors.New("description is required")
	}

	if priority == "" {
		priority = "medium"
	}

	now := time.Now().UTC()
	ticket := &ProblemTicket{
		ID:           primitive.NewObjectID(),
		TicketID:     ticketID,
		ShipmentID:   shipmentID,
		SKU:          sku,
		ProductName:  productName,
		ProblemType:  problemType,
		Description:  description,
		Quantity:     quantity,
		Resolution:   ResolutionPending,
		CreatedBy:    createdBy,
		Priority:     priority,
		CreatedAt:    now,
		UpdatedAt:    now,
		DomainEvents: make([]DomainEvent, 0),
	}

	ticket.addDomainEvent(&ProblemCreatedEvent{
		TicketID:    ticketID,
		ShipmentID:  shipmentID,
		SKU:         sku,
		ProblemType: string(problemType),
		Description: description,
		Quantity:    quantity,
		Priority:    priority,
		CreatedBy:   createdBy,
		OccurredAt_: now,
	})

	return ticket, nil
}

// AssignTo assigns the problem ticket to a supervisor/resolver
func (t *ProblemTicket) AssignTo(userID string) error {
	if t.Resolution != ResolutionPending && t.Resolution != ResolutionInvestigate {
		return ErrProblemAlreadyResolved
	}

	t.AssignedTo = userID
	t.UpdatedAt = time.Now().UTC()

	return nil
}

// Resolve resolves the problem ticket with a resolution decision
func (t *ProblemTicket) Resolve(resolution ProblemResolution, notes string, resolvedBy string) error {
	if !resolution.IsValid() {
		return ErrInvalidResolution
	}

	if !t.Resolution.CanTransitionTo(resolution) {
		return errors.New("invalid resolution transition")
	}

	if notes == "" && (resolution == ResolutionRejected || resolution == ResolutionDisposed || resolution == ResolutionAdjusted) {
		return ErrCannotResolveWithoutReason
	}

	now := time.Now().UTC()
	t.Resolution = resolution
	t.ResolutionNotes = notes
	t.ResolvedBy = resolvedBy
	t.ResolvedAt = &now
	t.UpdatedAt = now

	t.addDomainEvent(&ProblemResolvedEvent{
		TicketID:        t.TicketID,
		ShipmentID:      t.ShipmentID,
		SKU:             t.SKU,
		ProblemType:     string(t.ProblemType),
		Resolution:      string(resolution),
		ResolutionNotes: notes,
		ResolvedBy:      resolvedBy,
		OccurredAt_:     now,
	})

	// Emit specific events based on resolution
	switch resolution {
	case ResolutionDisposed:
		t.addDomainEvent(&ItemDisposedEvent{
			TicketID:    t.TicketID,
			ShipmentID:  t.ShipmentID,
			SKU:         t.SKU,
			Quantity:    t.Quantity,
			Reason:      notes,
			DisposedBy:  resolvedBy,
			OccurredAt_: now,
		})
	case ResolutionRejected:
		t.addDomainEvent(&ReturnCreatedEvent{
			TicketID:    t.TicketID,
			ShipmentID:  t.ShipmentID,
			SKU:         t.SKU,
			Quantity:    t.Quantity,
			Reason:      notes,
			CreatedBy:   resolvedBy,
			OccurredAt_: now,
		})
	}

	return nil
}

// UpdateResolution updates the resolution (e.g., from investigate to another state)
func (t *ProblemTicket) UpdateResolution(resolution ProblemResolution, notes string, updatedBy string) error {
	if !resolution.IsValid() {
		return ErrInvalidResolution
	}

	if !t.Resolution.CanTransitionTo(resolution) {
		return errors.New("invalid resolution transition")
	}

	t.Resolution = resolution
	if notes != "" {
		if t.ResolutionNotes == "" {
			t.ResolutionNotes = notes
		} else {
			t.ResolutionNotes += "\n" + notes
		}
	}
	t.UpdatedAt = time.Now().UTC()

	return nil
}

// AddImage adds an image URL to the problem ticket
func (t *ProblemTicket) AddImage(imageURL string) {
	if t.ImageURLs == nil {
		t.ImageURLs = make([]string, 0)
	}
	t.ImageURLs = append(t.ImageURLs, imageURL)
	t.UpdatedAt = time.Now().UTC()
}

// IsPending checks if the ticket is still pending
func (t *ProblemTicket) IsPending() bool {
	return t.Resolution == ResolutionPending || t.Resolution == ResolutionInvestigate
}

// IsResolved checks if the ticket has been resolved
func (t *ProblemTicket) IsResolved() bool {
	return t.ResolvedAt != nil
}

// addDomainEvent adds a domain event
func (t *ProblemTicket) addDomainEvent(event DomainEvent) {
	t.DomainEvents = append(t.DomainEvents, event)
}

// GetDomainEvents returns all domain events
func (t *ProblemTicket) GetDomainEvents() []DomainEvent {
	return t.DomainEvents
}

// ClearDomainEvents clears all domain events
func (t *ProblemTicket) ClearDomainEvents() {
	t.DomainEvents = make([]DomainEvent, 0)
}

// ProblemTicketRepository defines the repository interface for problem tickets
type ProblemTicketRepository interface {
	Save(ticket *ProblemTicket) error
	FindByID(ticketID string) (*ProblemTicket, error)
	FindByShipmentID(shipmentID string) ([]*ProblemTicket, error)
	FindPending(limit int) ([]*ProblemTicket, error)
	FindByResolution(resolution ProblemResolution, limit int) ([]*ProblemTicket, error)
}
