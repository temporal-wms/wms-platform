package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Sortation errors
var (
	ErrBatchNotFound           = errors.New("sortation batch not found")
	ErrPackageNotFound         = errors.New("package not found in batch")
	ErrInvalidBatchStatus      = errors.New("invalid batch status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrPackageAlreadySorted    = errors.New("package already sorted")
	ErrChuteNotAvailable       = errors.New("chute not available")
	ErrBatchAlreadyDispatched  = errors.New("batch already dispatched")
)

// SortationStatus represents the status of a sortation batch
type SortationStatus string

const (
	SortationStatusReceiving   SortationStatus = "receiving"
	SortationStatusSorting     SortationStatus = "sorting"
	SortationStatusReady       SortationStatus = "ready"
	SortationStatusDispatching SortationStatus = "dispatching"
	SortationStatusDispatched  SortationStatus = "dispatched"
	SortationStatusCancelled   SortationStatus = "cancelled"
)

// IsValid checks if the status is valid
func (s SortationStatus) IsValid() bool {
	switch s {
	case SortationStatusReceiving, SortationStatusSorting, SortationStatusReady,
		SortationStatusDispatching, SortationStatusDispatched, SortationStatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the status can transition to another status
func (s SortationStatus) CanTransitionTo(target SortationStatus) bool {
	validTransitions := map[SortationStatus][]SortationStatus{
		SortationStatusReceiving:   {SortationStatusSorting, SortationStatusCancelled},
		SortationStatusSorting:     {SortationStatusReady, SortationStatusCancelled},
		SortationStatusReady:       {SortationStatusDispatching, SortationStatusCancelled},
		SortationStatusDispatching: {SortationStatusDispatched},
		SortationStatusDispatched:  {},
		SortationStatusCancelled:   {},
	}

	allowedTargets, exists := validTransitions[s]
	if !exists {
		return false
	}

	for _, allowed := range allowedTargets {
		if target == allowed {
			return true
		}
	}
	return false
}

// SortedPackage represents a package in the sortation batch
type SortedPackage struct {
	PackageID     string     `bson:"packageId" json:"packageId"`
	OrderID       string     `bson:"orderId" json:"orderId"`
	TrackingNumber string    `bson:"trackingNumber" json:"trackingNumber"`
	Destination   string     `bson:"destination" json:"destination"` // zip code or region
	CarrierID     string     `bson:"carrierId" json:"carrierId"`
	Weight        float64    `bson:"weight" json:"weight"`
	AssignedChute string     `bson:"assignedChute,omitempty" json:"assignedChute,omitempty"`
	SortedAt      *time.Time `bson:"sortedAt,omitempty" json:"sortedAt,omitempty"`
	SortedBy      string     `bson:"sortedBy,omitempty" json:"sortedBy,omitempty"`
	IsSorted      bool       `bson:"isSorted" json:"isSorted"`
}

// Chute represents a sortation chute
type Chute struct {
	ChuteID       string `bson:"chuteId" json:"chuteId"`
	ChuteNumber   int    `bson:"chuteNumber" json:"chuteNumber"`
	Destination   string `bson:"destination" json:"destination"` // zip/region this chute handles
	CarrierID     string `bson:"carrierId" json:"carrierId"`
	Capacity      int    `bson:"capacity" json:"capacity"`
	CurrentCount  int    `bson:"currentCount" json:"currentCount"`
	Status        string `bson:"status" json:"status"` // active, full, maintenance
}

// AvailableCapacity returns remaining capacity
func (c *Chute) AvailableCapacity() int {
	return c.Capacity - c.CurrentCount
}

// IsAvailable checks if the chute can accept more packages
func (c *Chute) IsAvailable() bool {
	return c.Status == "active" && c.CurrentCount < c.Capacity
}

// SortationBatch is the aggregate root for the Sortation bounded context
type SortationBatch struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BatchID          string             `bson:"batchId" json:"batchId"`
	TenantID    string `bson:"tenantId" json:"tenantId"`
	FacilityID  string `bson:"facilityId" json:"facilityId"`
	WarehouseID string `bson:"warehouseId" json:"warehouseId"`
	SortationCenter  string             `bson:"sortationCenter" json:"sortationCenter"`
	DestinationGroup string             `bson:"destinationGroup" json:"destinationGroup"` // zip prefix or region
	CarrierID        string             `bson:"carrierId" json:"carrierId"`
	Packages         []SortedPackage    `bson:"packages" json:"packages"`
	AssignedChute    string             `bson:"assignedChute,omitempty" json:"assignedChute,omitempty"`
	Status           SortationStatus    `bson:"status" json:"status"`
	TotalPackages    int                `bson:"totalPackages" json:"totalPackages"`
	SortedCount      int                `bson:"sortedCount" json:"sortedCount"`
	TotalWeight      float64            `bson:"totalWeight" json:"totalWeight"`
	TrailerID        string             `bson:"trailerId,omitempty" json:"trailerId,omitempty"`
	DispatchDock     string             `bson:"dispatchDock,omitempty" json:"dispatchDock,omitempty"`
	ScheduledDispatch *time.Time        `bson:"scheduledDispatch,omitempty" json:"scheduledDispatch,omitempty"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`
	DispatchedAt     *time.Time         `bson:"dispatchedAt,omitempty" json:"dispatchedAt,omitempty"`
	DomainEvents     []DomainEvent      `bson:"-" json:"-"`
}

// NewSortationBatch creates a new SortationBatch aggregate
func NewSortationBatch(batchID, sortationCenter, destinationGroup, carrierID string) *SortationBatch {
	now := time.Now().UTC()
	batch := &SortationBatch{
		ID:               primitive.NewObjectID(),
		BatchID:          batchID,
		SortationCenter:  sortationCenter,
		DestinationGroup: destinationGroup,
		CarrierID:        carrierID,
		Packages:         make([]SortedPackage, 0),
		Status:           SortationStatusReceiving,
		TotalPackages:    0,
		SortedCount:      0,
		TotalWeight:      0,
		CreatedAt:        now,
		UpdatedAt:        now,
		DomainEvents:     make([]DomainEvent, 0),
	}

	batch.addDomainEvent(&SortationBatchCreatedEvent{
		BatchID:          batchID,
		SortationCenter:  sortationCenter,
		DestinationGroup: destinationGroup,
		CarrierID:        carrierID,
		CreatedAt:        now,
	})

	return batch
}

// AddPackage adds a package to the batch
func (b *SortationBatch) AddPackage(pkg SortedPackage) error {
	if b.Status != SortationStatusReceiving && b.Status != SortationStatusSorting {
		return ErrInvalidStatusTransition
	}

	pkg.IsSorted = false
	b.Packages = append(b.Packages, pkg)
	b.TotalPackages++
	b.TotalWeight += pkg.Weight
	b.UpdatedAt = time.Now().UTC()

	b.addDomainEvent(&PackageReceivedForSortationEvent{
		BatchID:    b.BatchID,
		PackageID:  pkg.PackageID,
		OrderID:    pkg.OrderID,
		Destination: pkg.Destination,
		ReceivedAt: b.UpdatedAt,
	})

	return nil
}

// StartSorting transitions the batch to sorting status
func (b *SortationBatch) StartSorting() error {
	if !b.Status.CanTransitionTo(SortationStatusSorting) {
		return ErrInvalidStatusTransition
	}

	b.Status = SortationStatusSorting
	b.UpdatedAt = time.Now().UTC()

	return nil
}

// SortPackage marks a package as sorted to a chute
func (b *SortationBatch) SortPackage(packageID, chuteID, workerID string) error {
	if b.Status != SortationStatusSorting && b.Status != SortationStatusReceiving {
		return ErrInvalidStatusTransition
	}

	// Transition to sorting if receiving
	if b.Status == SortationStatusReceiving {
		b.Status = SortationStatusSorting
	}

	// Find the package
	for i := range b.Packages {
		if b.Packages[i].PackageID == packageID {
			if b.Packages[i].IsSorted {
				return ErrPackageAlreadySorted
			}

			now := time.Now().UTC()
			b.Packages[i].AssignedChute = chuteID
			b.Packages[i].SortedAt = &now
			b.Packages[i].SortedBy = workerID
			b.Packages[i].IsSorted = true
			b.SortedCount++
			b.UpdatedAt = now

			b.addDomainEvent(&PackageSortedEvent{
				BatchID:   b.BatchID,
				PackageID: packageID,
				ChuteID:   chuteID,
				SortedBy:  workerID,
				SortedAt:  now,
			})

			return nil
		}
	}

	return ErrPackageNotFound
}

// MarkReady marks the batch as ready for dispatch
func (b *SortationBatch) MarkReady() error {
	if !b.Status.CanTransitionTo(SortationStatusReady) {
		return ErrInvalidStatusTransition
	}

	b.Status = SortationStatusReady
	b.UpdatedAt = time.Now().UTC()

	return nil
}

// AssignToTrailer assigns the batch to a trailer
func (b *SortationBatch) AssignToTrailer(trailerID, dispatchDock string) error {
	if b.Status != SortationStatusReady {
		return ErrInvalidStatusTransition
	}

	b.TrailerID = trailerID
	b.DispatchDock = dispatchDock
	b.Status = SortationStatusDispatching
	b.UpdatedAt = time.Now().UTC()

	return nil
}

// Dispatch dispatches the batch
func (b *SortationBatch) Dispatch() error {
	if !b.Status.CanTransitionTo(SortationStatusDispatched) {
		return ErrInvalidStatusTransition
	}

	now := time.Now().UTC()
	b.Status = SortationStatusDispatched
	b.DispatchedAt = &now
	b.UpdatedAt = now

	b.addDomainEvent(&BatchDispatchedEvent{
		BatchID:     b.BatchID,
		TrailerID:   b.TrailerID,
		DispatchDock: b.DispatchDock,
		PackageCount: b.TotalPackages,
		TotalWeight:  b.TotalWeight,
		DispatchedAt: now,
	})

	return nil
}

// Cancel cancels the batch
func (b *SortationBatch) Cancel(reason string) error {
	if b.Status == SortationStatusDispatched {
		return ErrBatchAlreadyDispatched
	}

	b.Status = SortationStatusCancelled
	b.UpdatedAt = time.Now().UTC()

	return nil
}

// GetUnsortedPackages returns packages that haven't been sorted
func (b *SortationBatch) GetUnsortedPackages() []SortedPackage {
	unsorted := make([]SortedPackage, 0)
	for _, pkg := range b.Packages {
		if !pkg.IsSorted {
			unsorted = append(unsorted, pkg)
		}
	}
	return unsorted
}

// IsFullySorted checks if all packages are sorted
func (b *SortationBatch) IsFullySorted() bool {
	return b.SortedCount >= b.TotalPackages
}

// GetSortingProgress returns the sorting progress percentage
func (b *SortationBatch) GetSortingProgress() float64 {
	if b.TotalPackages == 0 {
		return 0
	}
	return float64(b.SortedCount) / float64(b.TotalPackages) * 100
}

// addDomainEvent adds a domain event
func (b *SortationBatch) addDomainEvent(event DomainEvent) {
	b.DomainEvents = append(b.DomainEvents, event)
}

// GetDomainEvents returns all domain events
func (b *SortationBatch) GetDomainEvents() []DomainEvent {
	return b.DomainEvents
}

// ClearDomainEvents clears all domain events
func (b *SortationBatch) ClearDomainEvents() {
	b.DomainEvents = make([]DomainEvent, 0)
}
