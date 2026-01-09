package domain

import "context"

// PackTaskRepository defines the interface for pack task persistence
type PackTaskRepository interface {
	Save(ctx context.Context, task *PackTask) error
	FindByID(ctx context.Context, taskID string) (*PackTask, error)
	FindByOrderID(ctx context.Context, orderID string) (*PackTask, error)
	FindByWaveID(ctx context.Context, waveID string) ([]*PackTask, error)
	FindByPackerID(ctx context.Context, packerID string) ([]*PackTask, error)
	FindByStatus(ctx context.Context, status PackTaskStatus) ([]*PackTask, error)
	FindByStation(ctx context.Context, station string) ([]*PackTask, error)
	FindPending(ctx context.Context, limit int) ([]*PackTask, error)
	FindByTrackingNumber(ctx context.Context, trackingNumber string) (*PackTask, error)
	Delete(ctx context.Context, taskID string) error
}

// LabelGenerator defines the interface for generating shipping labels
type LabelGenerator interface {
	GenerateLabel(ctx context.Context, request LabelRequest) (*ShippingLabel, error)
}

// LabelRequest represents a request to generate a shipping label
type LabelRequest struct {
	OrderID     string     `json:"orderId"`
	PackageID   string     `json:"packageId"`
	Carrier     string     `json:"carrier"`
	ServiceType string     `json:"serviceType"`
	Weight      float64    `json:"weight"`
	Dimensions  Dimensions `json:"dimensions"`
	Recipient   Address    `json:"recipient"`
	Shipper     Address    `json:"shipper"`
}

// Address represents a shipping address
type Address struct {
	Name       string `json:"name"`
	Company    string `json:"company,omitempty"`
	Street1    string `json:"street1"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
	Phone      string `json:"phone,omitempty"`
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	Publish(ctx context.Context, event DomainEvent) error
	PublishAll(ctx context.Context, events []DomainEvent) error
}
