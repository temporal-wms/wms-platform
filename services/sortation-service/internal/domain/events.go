package domain

import "time"

// DomainEvent represents a domain event interface
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// SortationBatchCreatedEvent is emitted when a sortation batch is created
type SortationBatchCreatedEvent struct {
	BatchID          string    `json:"batchId"`
	SortationCenter  string    `json:"sortationCenter"`
	DestinationGroup string    `json:"destinationGroup"`
	CarrierID        string    `json:"carrierId"`
	CreatedAt        time.Time `json:"createdAt"`
}

func (e *SortationBatchCreatedEvent) EventType() string     { return "sortation.batch.created" }
func (e *SortationBatchCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// PackageReceivedForSortationEvent is emitted when a package is received for sortation
type PackageReceivedForSortationEvent struct {
	BatchID     string    `json:"batchId"`
	PackageID   string    `json:"packageId"`
	OrderID     string    `json:"orderId"`
	Destination string    `json:"destination"`
	ReceivedAt  time.Time `json:"receivedAt"`
}

func (e *PackageReceivedForSortationEvent) EventType() string     { return "sortation.package.received" }
func (e *PackageReceivedForSortationEvent) OccurredAt() time.Time { return e.ReceivedAt }

// PackageSortedEvent is emitted when a package is sorted to a chute
type PackageSortedEvent struct {
	BatchID   string    `json:"batchId"`
	PackageID string    `json:"packageId"`
	ChuteID   string    `json:"chuteId"`
	SortedBy  string    `json:"sortedBy"`
	SortedAt  time.Time `json:"sortedAt"`
}

func (e *PackageSortedEvent) EventType() string     { return "sortation.package.sorted" }
func (e *PackageSortedEvent) OccurredAt() time.Time { return e.SortedAt }

// BatchDispatchedEvent is emitted when a batch is dispatched
type BatchDispatchedEvent struct {
	BatchID      string    `json:"batchId"`
	TrailerID    string    `json:"trailerId"`
	DispatchDock string    `json:"dispatchDock"`
	PackageCount int       `json:"packageCount"`
	TotalWeight  float64   `json:"totalWeight"`
	DispatchedAt time.Time `json:"dispatchedAt"`
}

func (e *BatchDispatchedEvent) EventType() string     { return "sortation.batch.dispatched" }
func (e *BatchDispatchedEvent) OccurredAt() time.Time { return e.DispatchedAt }

// BatchReadyEvent is emitted when a batch is ready for dispatch
type BatchReadyEvent struct {
	BatchID          string    `json:"batchId"`
	DestinationGroup string    `json:"destinationGroup"`
	CarrierID        string    `json:"carrierId"`
	PackageCount     int       `json:"packageCount"`
	ReadyAt          time.Time `json:"readyAt"`
}

func (e *BatchReadyEvent) EventType() string     { return "sortation.batch.ready" }
func (e *BatchReadyEvent) OccurredAt() time.Time { return e.ReadyAt }
