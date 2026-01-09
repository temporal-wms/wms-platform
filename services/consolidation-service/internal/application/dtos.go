package application

import "time"

// ConsolidationDTO represents a consolidation unit in responses
type ConsolidationDTO struct {
	ConsolidationID   string                `json:"consolidationId"`
	OrderID           string                `json:"orderId"`
	WaveID            string                `json:"waveId"`
	Status            string                `json:"status"`
	Strategy          string                `json:"strategy"`
	ExpectedItems     []ExpectedItemDTO     `json:"expectedItems"`
	ConsolidatedItems []ConsolidatedItemDTO `json:"consolidatedItems"`
	SourceTotes       []string              `json:"sourceTotes"`
	DestinationBin    string                `json:"destinationBin"`
	Station           string                `json:"station"`
	WorkerID          string                `json:"workerId,omitempty"`
	TotalExpected     int                   `json:"totalExpected"`
	TotalConsolidated int                   `json:"totalConsolidated"`
	ReadyForPacking   bool                  `json:"readyForPacking"`
	CreatedAt         time.Time             `json:"createdAt"`
	UpdatedAt         time.Time             `json:"updatedAt"`
	StartedAt         *time.Time            `json:"startedAt,omitempty"`
	CompletedAt       *time.Time            `json:"completedAt,omitempty"`
}

// ExpectedItemDTO represents an item expected for consolidation
type ExpectedItemDTO struct {
	SKU          string `json:"sku"`
	ProductName  string `json:"productName"`
	Quantity     int    `json:"quantity"`
	SourceToteID string `json:"sourceToteId"`
	Received     int    `json:"received"`
	Status       string `json:"status"`
}

// ConsolidatedItemDTO represents an item that has been consolidated
type ConsolidatedItemDTO struct {
	SKU          string    `json:"sku"`
	Quantity     int       `json:"quantity"`
	SourceToteID string    `json:"sourceToteId"`
	ScannedAt    time.Time `json:"scannedAt"`
	VerifiedBy   string    `json:"verifiedBy"`
}
