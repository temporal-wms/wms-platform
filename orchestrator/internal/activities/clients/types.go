package clients

import "time"

// Order represents an order from order-service
type Order struct {
	OrderID            string         `json:"orderId"`
	CustomerID         string         `json:"customerId"`
	Status             string         `json:"status"`
	Priority           string         `json:"priority"`
	Items              []OrderItem    `json:"items"`
	ShippingAddress    Address        `json:"shippingAddress"`
	PromisedDeliveryAt time.Time      `json:"promisedDeliveryAt"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	SKU      string  `json:"sku"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
	Weight   float64 `json:"weight"`
}

// Address represents a shipping address
type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

// OrderValidationResult represents the result of order validation
type OrderValidationResult struct {
	OrderID      string   `json:"orderId"`
	Valid        bool     `json:"valid"`
	Errors       []string `json:"errors,omitempty"`
	ValidatedAt  time.Time `json:"validatedAt"`
}

// InventoryItem represents an inventory item
type InventoryItem struct {
	SKU               string `json:"sku"`
	LocationID        string `json:"locationId"`
	AvailableQuantity int    `json:"availableQuantity"`
	ReservedQuantity  int    `json:"reservedQuantity"`
	Zone              string `json:"zone"`
	Aisle             string `json:"aisle"`
	Shelf             string `json:"shelf"`
	Bin               string `json:"bin"`
}

// ReserveInventoryRequest represents a request to reserve inventory
type ReserveInventoryRequest struct {
	OrderID string              `json:"orderId"`
	Items   []ReserveItemRequest `json:"items"`
}

// ReserveItemRequest represents a single item reservation request
type ReserveItemRequest struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// PickInventoryRequest represents a request to pick/decrement inventory
type PickInventoryRequest struct {
	OrderID    string `json:"orderId"`
	LocationID string `json:"locationId"`
	Quantity   int    `json:"quantity"`
	CreatedBy  string `json:"createdBy"`
}

// StageInventoryRequest represents a request to stage inventory (soft to hard allocation)
type StageInventoryRequest struct {
	ReservationID     string `json:"reservationId"`
	StagingLocationID string `json:"stagingLocationId"`
	StagedBy          string `json:"stagedBy"`
}

// PackInventoryRequest represents a request to mark inventory as packed
type PackInventoryRequest struct {
	AllocationID string `json:"allocationId"`
	PackedBy     string `json:"packedBy"`
}

// ShipInventoryRequest represents a request to ship inventory
type ShipInventoryRequest struct {
	AllocationID string `json:"allocationId"`
}

// ReturnToShelfRequest represents a request to return staged inventory to shelf
type ReturnToShelfRequest struct {
	AllocationID string `json:"allocationId"`
	ReturnedBy   string `json:"returnedBy"`
	Reason       string `json:"reason"`
}

// Route represents a pick route
type Route struct {
	RouteID           string      `json:"routeId"`
	OrderID           string      `json:"orderId"`
	WaveID            string      `json:"waveId"`
	Status            string      `json:"status"`
	Stops             []RouteStop `json:"stops"`
	Strategy          string      `json:"strategy"`
	EstimatedDistance float64     `json:"estimatedDistance"`
	CreatedAt         time.Time   `json:"createdAt"`
}

// RouteStop represents a stop in a pick route
type RouteStop struct {
	Sequence   int    `json:"sequence"`
	LocationID string `json:"locationId"`
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	Zone       string `json:"zone"`
	Aisle      string `json:"aisle"`
	Shelf      string `json:"shelf"`
	Bin        string `json:"bin"`
}

// CalculateRouteRequest represents a request to calculate a route
type CalculateRouteRequest struct {
	RouteID  string            `json:"routeId"`
	OrderID  string            `json:"orderId"`
	WaveID   string            `json:"waveId"`
	Items    []RouteItemRequest `json:"items"`
	Strategy string            `json:"strategy"`
}

// RouteItemRequest represents an item for route calculation
type RouteItemRequest struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// PickTask represents a pick task
type PickTask struct {
	TaskID           string       `json:"taskId"`
	OrderID          string       `json:"orderId"`
	WaveID           string       `json:"waveId"`
	RouteID          string       `json:"routeId"`
	WorkerID         string       `json:"workerId,omitempty"`
	Status           string       `json:"status"`
	Items            []PickItem   `json:"items"`
	PickedItemsCount int          `json:"pickedItemsCount,omitempty"`
	PickedItems      []PickedItem `json:"pickedItems,omitempty"`
	ToteID           string       `json:"toteId,omitempty"`
	CreatedAt        time.Time    `json:"createdAt"`
	StartedAt        *time.Time   `json:"startedAt,omitempty"`
	CompletedAt      *time.Time   `json:"completedAt,omitempty"`
}

// PickItem represents an item to be picked
type PickItem struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId"`
}

// PickedItem represents a picked item
type PickedItem struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId"`
	ToteID     string `json:"toteId"`
	PickedAt   time.Time `json:"pickedAt"`
}

// CreatePickTaskRequest represents a request to create a pick task
type CreatePickTaskRequest struct {
	TaskID   string     `json:"taskId"`
	OrderID  string     `json:"orderId"`
	WaveID   string     `json:"waveId"`
	RouteID  string     `json:"routeId"`
	Items    []PickItem `json:"items"`
}

// ConfirmPickRequest represents a request to confirm a pick
type ConfirmPickRequest struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId"`
	ToteID     string `json:"toteId"`
}

// ConsolidationUnit represents a consolidation unit
type ConsolidationUnit struct {
	ConsolidationID string              `json:"consolidationId"`
	OrderID         string              `json:"orderId"`
	WaveID          string              `json:"waveId"`
	Status          string              `json:"status"`
	Station         string              `json:"station,omitempty"`
	ExpectedItems   []ExpectedItem      `json:"expectedItems"`
	ConsolidatedItems []ConsolidatedItem `json:"consolidatedItems,omitempty"`
	DestinationBin  string              `json:"destinationBin,omitempty"`
	CreatedAt       time.Time           `json:"createdAt"`
	CompletedAt     *time.Time          `json:"completedAt,omitempty"`
}

// ExpectedItem represents an expected item for consolidation
type ExpectedItem struct {
	SKU          string `json:"sku"`
	Quantity     int    `json:"quantity"`
	SourceToteID string `json:"sourceToteId"`
}

// ConsolidatedItem represents a consolidated item
type ConsolidatedItem struct {
	SKU            string    `json:"sku"`
	Quantity       int       `json:"quantity"`
	SourceToteID   string    `json:"sourceToteId"`
	ConsolidatedAt time.Time `json:"consolidatedAt"`
}

// CreateConsolidationRequest represents a request to create consolidation
type CreateConsolidationRequest struct {
	ConsolidationID string         `json:"consolidationId"`
	OrderID         string         `json:"orderId"`
	WaveID          string         `json:"waveId"`
	Strategy        string         `json:"strategy,omitempty"`
	Items           []ExpectedItem `json:"items"`
}

// ConsolidateItemRequest represents a request to consolidate an item
type ConsolidateItemRequest struct {
	SKU          string `json:"sku"`
	Quantity     int    `json:"quantity"`
	SourceToteID string `json:"sourceToteId"`
	VerifiedBy   string `json:"verifiedBy"`
}

// PackTask represents a pack task
type PackTask struct {
	TaskID         string        `json:"taskId"`
	OrderID        string        `json:"orderId"`
	Status         string        `json:"status"`
	WorkerID       string        `json:"workerId,omitempty"`
	Items          []PackItem    `json:"items"`
	PackageID      string        `json:"packageId,omitempty"`
	PackageType    string        `json:"packageType,omitempty"`
	TrackingNumber string        `json:"trackingNumber,omitempty"`
	Carrier        string        `json:"carrier,omitempty"`
	Weight         float64       `json:"weight,omitempty"`
	Dimensions     *Dimensions   `json:"dimensions,omitempty"`
	CreatedAt      time.Time     `json:"createdAt"`
	CompletedAt    *time.Time    `json:"completedAt,omitempty"`
}

// PackItem represents an item to be packed
type PackItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// Dimensions represents package dimensions
type Dimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"`
}

// CreatePackTaskRequest represents a request to create a pack task
type CreatePackTaskRequest struct {
	TaskID  string     `json:"taskId"`
	OrderID string     `json:"orderId"`
	WaveID  string     `json:"waveId"`
	Items   []PackItem `json:"items"`
}

// SealPackageRequest represents a request to seal a package
type SealPackageRequest struct {
	PackageID   string     `json:"packageId"`
	PackageType string     `json:"packageType"`
	Weight      float64    `json:"weight"`
	Dimensions  Dimensions `json:"dimensions"`
}

// ApplyLabelRequest represents a request to apply a label
type ApplyLabelRequest struct {
	TrackingNumber string `json:"trackingNumber"`
	Carrier        string `json:"carrier"`
	LabelData      string `json:"labelData,omitempty"`
}

// Shipment represents a shipment
type Shipment struct {
	ShipmentID      string              `json:"shipmentId"`
	OrderID         string              `json:"orderId"`
	PackageID       string              `json:"packageId"`
	Carrier         ShipmentCarrier     `json:"carrier"`
	ServiceType     string              `json:"serviceType,omitempty"`
	TrackingNumber  string              `json:"trackingNumber,omitempty"`
	Status          string              `json:"status"`
	Package         ShipmentPackageInfo `json:"package,omitempty"`
	Recipient       ShipmentAddress     `json:"recipient,omitempty"`
	Shipper         ShipmentAddress     `json:"shipper,omitempty"`
	Label           *ShippingLabel      `json:"label,omitempty"`
	CreatedAt       time.Time           `json:"createdAt"`
	ShippedAt       *time.Time          `json:"shippedAt,omitempty"`
}

// ShipmentCarrier represents a shipping carrier
type ShipmentCarrier struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	AccountID   string `json:"accountId"`
	ServiceType string `json:"serviceType"`
}

// ShipmentPackageInfo represents package information for shipment
type ShipmentPackageInfo struct {
	PackageID   string     `json:"packageId"`
	Weight      float64    `json:"weight"`
	Dimensions  Dimensions `json:"dimensions"`
	PackageType string     `json:"packageType"`
}

// ShipmentAddress represents a shipping address
type ShipmentAddress struct {
	Name       string `json:"name"`
	Company    string `json:"company,omitempty"`
	Street1    string `json:"street1"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
	Phone      string `json:"phone,omitempty"`
	Email      string `json:"email,omitempty"`
}

// CreateShipmentRequest represents a request to create a shipment
type CreateShipmentRequest struct {
	ShipmentID string              `json:"shipmentId"`
	OrderID    string              `json:"orderId"`
	PackageID  string              `json:"packageId"`
	WaveID     string              `json:"waveId,omitempty"`
	Carrier    ShipmentCarrier     `json:"carrier"`
	Package    ShipmentPackageInfo `json:"package"`
	Recipient  ShipmentAddress     `json:"recipient"`
	Shipper    ShipmentAddress     `json:"shipper"`
}

// ShippingLabel represents a shipping label
type ShippingLabel struct {
	TrackingNumber string          `json:"trackingNumber"`
	LabelURL       string          `json:"labelUrl,omitempty"`
	LabelData      string          `json:"labelData,omitempty"`
	LabelFormat    string          `json:"labelFormat,omitempty"`
	Carrier        ShipmentCarrier `json:"carrier"`
	CreatedAt      time.Time       `json:"createdAt"`
}

// LaborTask represents a labor task
type LaborTask struct {
	TaskID     string     `json:"taskId"`
	TaskType   string     `json:"taskType"`
	WorkerID   string     `json:"workerId"`
	Status     string     `json:"status"`
	Zone       string     `json:"zone"`
	Priority   int        `json:"priority"`
	AssignedAt time.Time  `json:"assignedAt"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// Worker represents a worker
type Worker struct {
	WorkerID       string   `json:"workerId"`
	Name           string   `json:"name"`
	Status         string   `json:"status"`
	CurrentZone    string   `json:"currentZone"`
	Certifications []string `json:"certifications"`
}

// AssignWorkerRequest represents a request to assign a worker
type AssignWorkerRequest struct {
	TaskID   string `json:"taskId"`
	TaskType string `json:"taskType"`
	WorkerID string `json:"workerId,omitempty"`
	Zone     string `json:"zone"`
	Priority int    `json:"priority"`
}

// Wave represents a wave
type Wave struct {
	WaveID         string    `json:"waveId"`
	Status         string    `json:"status"`
	Type           string    `json:"type"`
	OrderIDs       []string  `json:"orderIds"`
	ScheduledStart time.Time `json:"scheduledStart"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Station represents a packing/consolidation station with capabilities
type Station struct {
	StationID          string            `json:"stationId"`
	Name               string            `json:"name"`
	Zone               string            `json:"zone"`
	StationType        string            `json:"stationType"`
	Status             string            `json:"status"`
	Capabilities       []string          `json:"capabilities"`
	MaxConcurrentTasks int               `json:"maxConcurrentTasks"`
	CurrentTasks       int               `json:"currentTasks"`
	AvailableCapacity  int               `json:"availableCapacity"`
	AssignedWorkerID   string            `json:"assignedWorkerId,omitempty"`
	Equipment          []StationEquipment `json:"equipment"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

// StationEquipment represents equipment at a station
type StationEquipment struct {
	EquipmentID   string `json:"equipmentId"`
	EquipmentType string `json:"equipmentType"`
	Status        string `json:"status"`
}

// FindCapableStationsRequest represents a request to find capable stations
type FindCapableStationsRequest struct {
	Requirements []string `json:"requirements"`
	StationType  string   `json:"stationType"`
	Zone         string   `json:"zone"`
}

// ProcessRequirement represents a fulfillment requirement
type ProcessRequirement string

const (
	RequirementSingleItem ProcessRequirement = "single_item"
	RequirementMultiItem  ProcessRequirement = "multi_item"
	RequirementGiftWrap   ProcessRequirement = "gift_wrap"
	RequirementHazmat     ProcessRequirement = "hazmat"
	RequirementOversized  ProcessRequirement = "oversized"
	RequirementFragile    ProcessRequirement = "fragile"
	RequirementColdChain  ProcessRequirement = "cold_chain"
	RequirementHighValue  ProcessRequirement = "high_value"
)

// ProcessPath represents the determined process path for an order
type ProcessPath struct {
	PathID                string               `json:"pathId"`
	OrderID               string               `json:"orderId"`
	Requirements          []ProcessRequirement `json:"requirements"`
	ConsolidationRequired bool                 `json:"consolidationRequired"`
	GiftWrapRequired      bool                 `json:"giftWrapRequired"`
	SpecialHandling       []string             `json:"specialHandling"`
	TargetStation         string               `json:"targetStation,omitempty"`
}

// GiftWrapDetails contains details for gift wrap processing
type GiftWrapDetails struct {
	WrapType    string `json:"wrapType"`
	GiftMessage string `json:"giftMessage"`
	HidePrice   bool   `json:"hidePrice"`
}

// HazmatDetails contains details for hazardous material handling
type HazmatDetails struct {
	Class              string `json:"class"`
	UNNumber           string `json:"unNumber"`
	PackingGroup       string `json:"packingGroup"`
	ProperShippingName string `json:"properShippingName"`
	LimitedQuantity    bool   `json:"limitedQuantity"`
}

// ColdChainDetails contains details for temperature-controlled shipping
type ColdChainDetails struct {
	MinTempCelsius  float64 `json:"minTempCelsius"`
	MaxTempCelsius  float64 `json:"maxTempCelsius"`
	RequiresDryIce  bool    `json:"requiresDryIce"`
	RequiresGelPack bool    `json:"requiresGelPack"`
}
