package application

import "time"

// PackTaskDTO represents a packing task in responses
type PackTaskDTO struct {
	TaskID          string              `json:"taskId"`
	OrderID         string              `json:"orderId"`
	ConsolidationID string              `json:"consolidationId,omitempty"`
	WaveID          string              `json:"waveId"`
	Status          string              `json:"status"`
	Items           []PackItemDTO       `json:"items"`
	Package         PackageDTO          `json:"package"`
	ShippingLabel   *ShippingLabelDTO   `json:"shippingLabel,omitempty"`
	PackerID        string              `json:"packerId,omitempty"`
	Station         string              `json:"station"`
	Priority        int                 `json:"priority"`
	CreatedAt       time.Time           `json:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt"`
	StartedAt       *time.Time          `json:"startedAt,omitempty"`
	PackedAt        *time.Time          `json:"packedAt,omitempty"`
	LabeledAt       *time.Time          `json:"labeledAt,omitempty"`
	CompletedAt     *time.Time          `json:"completedAt,omitempty"`
}

// PackItemDTO represents an item to be packed
type PackItemDTO struct {
	SKU         string  `json:"sku"`
	ProductName string  `json:"productName"`
	Quantity    int     `json:"quantity"`
	Weight      float64 `json:"weight"`
	Fragile     bool    `json:"fragile"`
	Verified    bool    `json:"verified"`
}

// PackageDTO represents the packaging used
type PackageDTO struct {
	PackageID     string        `json:"packageId"`
	Type          string        `json:"type"`
	SuggestedType string        `json:"suggestedType"`
	Dimensions    DimensionsDTO `json:"dimensions"`
	Weight        float64       `json:"weight"`
	TotalWeight   float64       `json:"totalWeight"`
	Materials     []string      `json:"materials"`
	Sealed        bool          `json:"sealed"`
	SealedAt      *time.Time    `json:"sealedAt,omitempty"`
}

// DimensionsDTO represents package dimensions
type DimensionsDTO struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// ShippingLabelDTO represents a shipping label
type ShippingLabelDTO struct {
	TrackingNumber string     `json:"trackingNumber"`
	Carrier        string     `json:"carrier"`
	ServiceType    string     `json:"serviceType"`
	LabelURL       string     `json:"labelUrl,omitempty"`
	LabelData      string     `json:"labelData,omitempty"`
	GeneratedAt    time.Time  `json:"generatedAt"`
	AppliedAt      *time.Time `json:"appliedAt,omitempty"`
}
