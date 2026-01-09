package domain

import "time"

// DomainEvent is the base interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// SellerCreatedEvent is emitted when a new seller is created
type SellerCreatedEvent struct {
	SellerID    string    `json:"sellerId"`
	TenantID    string    `json:"tenantId"`
	CompanyName string    `json:"companyName"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (e *SellerCreatedEvent) EventType() string   { return "seller.created" }
func (e *SellerCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// SellerActivatedEvent is emitted when a seller is activated
type SellerActivatedEvent struct {
	SellerID    string    `json:"sellerId"`
	ActivatedAt time.Time `json:"activatedAt"`
}

func (e *SellerActivatedEvent) EventType() string   { return "seller.activated" }
func (e *SellerActivatedEvent) OccurredAt() time.Time { return e.ActivatedAt }

// SellerSuspendedEvent is emitted when a seller is suspended
type SellerSuspendedEvent struct {
	SellerID    string    `json:"sellerId"`
	Reason      string    `json:"reason"`
	SuspendedAt time.Time `json:"suspendedAt"`
}

func (e *SellerSuspendedEvent) EventType() string   { return "seller.suspended" }
func (e *SellerSuspendedEvent) OccurredAt() time.Time { return e.SuspendedAt }

// SellerClosedEvent is emitted when a seller account is closed
type SellerClosedEvent struct {
	SellerID string    `json:"sellerId"`
	Reason   string    `json:"reason"`
	ClosedAt time.Time `json:"closedAt"`
}

func (e *SellerClosedEvent) EventType() string   { return "seller.closed" }
func (e *SellerClosedEvent) OccurredAt() time.Time { return e.ClosedAt }

// FacilityAssignedEvent is emitted when a facility is assigned to a seller
type FacilityAssignedEvent struct {
	SellerID   string    `json:"sellerId"`
	FacilityID string    `json:"facilityId"`
	AssignedAt time.Time `json:"assignedAt"`
}

func (e *FacilityAssignedEvent) EventType() string   { return "seller.facility_assigned" }
func (e *FacilityAssignedEvent) OccurredAt() time.Time { return e.AssignedAt }

// ChannelConnectedEvent is emitted when a sales channel is connected
type ChannelConnectedEvent struct {
	SellerID    string    `json:"sellerId"`
	ChannelID   string    `json:"channelId"`
	ChannelType string    `json:"channelType"`
	ConnectedAt time.Time `json:"connectedAt"`
}

func (e *ChannelConnectedEvent) EventType() string   { return "seller.channel_connected" }
func (e *ChannelConnectedEvent) OccurredAt() time.Time { return e.ConnectedAt }

// FeeScheduleUpdatedEvent is emitted when a seller's fee schedule is updated
type FeeScheduleUpdatedEvent struct {
	SellerID  string    `json:"sellerId"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (e *FeeScheduleUpdatedEvent) EventType() string   { return "seller.fee_schedule_updated" }
func (e *FeeScheduleUpdatedEvent) OccurredAt() time.Time { return e.UpdatedAt }
