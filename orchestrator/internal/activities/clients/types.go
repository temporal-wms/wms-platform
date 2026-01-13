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

// InventoryItemDetailed represents a detailed inventory item with reservations
type InventoryItemDetailed struct {
	SKU                   string               `json:"sku"`
	ProductName           string               `json:"productName"`
	Locations             []StockLocationDTO   `json:"locations"`
	TotalQuantity         int                  `json:"totalQuantity"`
	ReservedQuantity      int                  `json:"reservedQuantity"`
	HardAllocatedQuantity int                  `json:"hardAllocatedQuantity"`
	AvailableQuantity     int                  `json:"availableQuantity"`
	ReorderPoint          int                  `json:"reorderPoint"`
	ReorderQuantity       int                  `json:"reorderQuantity"`
	Reservations          []ReservationDTO     `json:"reservations,omitempty"`
	HardAllocations       []HardAllocationDTO  `json:"hardAllocations,omitempty"`
	LastCycleCount        *time.Time           `json:"lastCycleCount,omitempty"`
	CreatedAt             time.Time            `json:"createdAt"`
	UpdatedAt             time.Time            `json:"updatedAt"`
}

// StockLocationDTO represents stock at a specific location
type StockLocationDTO struct {
	LocationID    string `json:"locationId"`
	Zone          string `json:"zone"`
	Aisle         string `json:"aisle"`
	Rack          int    `json:"rack"`
	Level         int    `json:"level"`
	Quantity      int    `json:"quantity"`
	Reserved      int    `json:"reserved"`
	HardAllocated int    `json:"hardAllocated"`
	Available     int    `json:"available"`
}

// ReservationDTO represents a stock reservation
type ReservationDTO struct {
	ReservationID string    `json:"reservationId"`
	OrderID       string    `json:"orderId"`
	Quantity      int       `json:"quantity"`
	LocationID    string    `json:"locationId"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

// HardAllocationDTO represents a hard allocation (physically staged inventory)
type HardAllocationDTO struct {
	AllocationID      string    `json:"allocationId"`
	ReservationID     string    `json:"reservationId"`
	OrderID           string    `json:"orderId"`
	Quantity          int       `json:"quantity"`
	StagingLocationID string    `json:"stagingLocationId"`
	Status            string    `json:"status"`
	StagedBy          string    `json:"stagedBy"`
	CreatedAt         time.Time `json:"createdAt"`
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

// RecordShortageRequest represents a request to record a stock shortage
type RecordShortageRequest struct {
	LocationID  string `json:"locationId"`
	OrderID     string `json:"orderId"`
	ExpectedQty int    `json:"expectedQty"`
	ActualQty   int    `json:"actualQty"`
	Reason      string `json:"reason"`
	ReportedBy  string `json:"reportedBy"`
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

// MultiRouteResult contains the result of multi-route calculation
type MultiRouteResult struct {
	OrderID       string         `json:"orderId"`
	Routes        []Route        `json:"routes"`
	TotalRoutes   int            `json:"totalRoutes"`
	SplitReason   string         `json:"splitReason"`
	ZoneBreakdown map[string]int `json:"zoneBreakdown"`
	TotalItems    int            `json:"totalItems"`
	CreatedAt     time.Time      `json:"createdAt"`
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

// Station Capacity Management Types

// ReserveStationCapacityRequest represents a request to reserve station capacity
type ReserveStationCapacityRequest struct {
	StationID     string `json:"stationId"`
	OrderID       string `json:"orderId"`
	RequiredSlots int    `json:"requiredSlots"`
	ReservationID string `json:"reservationId"`
}

// ReserveStationCapacityResponse represents the response from reserving station capacity
type ReserveStationCapacityResponse struct {
	ReservationID     string `json:"reservationId"`
	StationID         string `json:"stationId"`
	ReservedSlots     int    `json:"reservedSlots"`
	RemainingCapacity int    `json:"remainingCapacity"`
}

// ReleaseStationCapacityRequest represents a request to release station capacity
type ReleaseStationCapacityRequest struct {
	StationID     string `json:"stationId"`
	OrderID       string `json:"orderId"`
	ReservationID string `json:"reservationId"`
	Reason        string `json:"reason,omitempty"`
}

// Labor Service Types

// Worker represents a warehouse worker
type Worker struct {
	WorkerID        string   `json:"workerId"`
	Name            string   `json:"name"`
	Skills          []string `json:"skills"`
	Certifications  []string `json:"certifications"`
	CurrentTask     string   `json:"currentTask,omitempty"`
	Zone            string   `json:"zone"`
	Status          string   `json:"status"` // available, on_task, on_break, off_duty
	CurrentShift    string   `json:"currentShift,omitempty"`
	AssignedStation string   `json:"assignedStation,omitempty"`
}

// FindCertifiedWorkersRequest represents a request to find certified workers
type FindCertifiedWorkersRequest struct {
	RequiredSkills []string `json:"requiredSkills"`
	Zone           string   `json:"zone,omitempty"`
	ShiftTime      string   `json:"shiftTime,omitempty"`
	MinCount       int      `json:"minCount"`
}

// AssignWorkerRequest represents a request to assign a worker
type AssignWorkerRequest struct {
	OrderID        string   `json:"orderId"`
	StationID      string   `json:"stationId"`
	RequiredSkills []string `json:"requiredSkills"`
	Zone           string   `json:"zone,omitempty"`
	Priority       string   `json:"priority,omitempty"`
}

// GetAvailableWorkersRequest represents a request to get available workers
type GetAvailableWorkersRequest struct {
	Zone           string   `json:"zone,omitempty"`
	RequiredSkills []string `json:"requiredSkills,omitempty"`
	ShiftTime      string   `json:"shiftTime,omitempty"`
}

// Equipment Availability and Management Types

// Equipment represents warehouse equipment
type Equipment struct {
	EquipmentID   string `json:"equipmentId"`
	EquipmentType string `json:"equipmentType"` // cold_storage, forklift, hazmat_kit, etc.
	Name          string `json:"name"`
	Zone          string `json:"zone"`
	Status        string `json:"status"` // available, in_use, maintenance, reserved
	StationID     string `json:"stationId,omitempty"`
	AssignedTo    string `json:"assignedTo,omitempty"` // Order or task ID
}

// CheckEquipmentAvailabilityRequest represents a request to check equipment availability
type CheckEquipmentAvailabilityRequest struct {
	EquipmentTypes []string `json:"equipmentTypes"`
	Zone           string   `json:"zone,omitempty"`
	RequiredCount  int      `json:"requiredCount"`
}

// CheckEquipmentAvailabilityResponse represents the response from checking equipment availability
type CheckEquipmentAvailabilityResponse struct {
	AvailableEquipment map[string]int `json:"availableEquipment"` // Type -> count
}

// ReserveEquipmentRequest represents a request to reserve equipment
type ReserveEquipmentRequest struct {
	EquipmentType string `json:"equipmentType"`
	OrderID       string `json:"orderId"`
	Quantity      int    `json:"quantity"`
	Zone          string `json:"zone,omitempty"`
	ReservationID string `json:"reservationId"`
}

// ReserveEquipmentResponse represents the response from reserving equipment
type ReserveEquipmentResponse struct {
	ReservationID        string   `json:"reservationId"`
	EquipmentType        string   `json:"equipmentType"`
	ReservedEquipmentIDs []string `json:"reservedEquipmentIds"`
}

// ReleaseEquipmentRequest represents a request to release equipment
type ReleaseEquipmentRequest struct {
	ReservationID string `json:"reservationId"`
	EquipmentType string `json:"equipmentType"`
	OrderID       string `json:"orderId"`
	Reason        string `json:"reason,omitempty"`
}

// GetEquipmentByTypeRequest represents a request to get equipment by type
type GetEquipmentByTypeRequest struct {
	EquipmentType string `json:"equipmentType"`
	Zone          string `json:"zone,omitempty"`
	Status        string `json:"status,omitempty"`
}

// Routing Optimizer Types

// OptimizeRoutingRequest represents a request for optimized routing
type OptimizeRoutingRequest struct {
	OrderID            string    `json:"orderId"`
	Priority           string    `json:"priority"`
	Requirements       []string  `json:"requirements"`
	SpecialHandling    []string  `json:"specialHandling"`
	ItemCount          int       `json:"itemCount"`
	TotalWeight        float64   `json:"totalWeight"`
	PromisedDeliveryAt time.Time `json:"promisedDeliveryAt"`
	RequiredSkills     []string  `json:"requiredSkills"`
	RequiredEquipment  []string  `json:"requiredEquipment"`
	Zone               string    `json:"zone,omitempty"`
	StationType        string    `json:"stationType"`
}

// OptimizeRoutingResponse represents the optimized routing decision
type OptimizeRoutingResponse struct {
	SelectedStationID string                     `json:"selectedStationId"`
	Score             float64                    `json:"score"`
	Reasoning         map[string]float64         `json:"reasoning"`
	AlternateStations []AlternateStationResponse `json:"alternateStations"`
	Confidence        float64                    `json:"confidence"`
	DecisionTime      time.Time                  `json:"decisionTime"`
}

// AlternateStationResponse represents an alternate station option
type AlternateStationResponse struct {
	StationID string  `json:"stationId"`
	Score     float64 `json:"score"`
	Rank      int     `json:"rank"`
}

// GetRoutingMetricsRequest represents a request for routing metrics
type GetRoutingMetricsRequest struct {
	FacilityID string `json:"facilityId,omitempty"`
	Zone       string `json:"zone,omitempty"`
	TimeWindow string `json:"timeWindow,omitempty"`
}

// GetRoutingMetricsResponse represents routing metrics
type GetRoutingMetricsResponse struct {
	TotalRoutingDecisions   int                `json:"totalRoutingDecisions"`
	AverageDecisionTimeMs   int64              `json:"averageDecisionTimeMs"`
	AverageConfidence       float64            `json:"averageConfidence"`
	StationUtilization      map[string]float64 `json:"stationUtilization"`
	CapacityConstrainedRate float64            `json:"capacityConstrainedRate"`
	RouteChanges            int                `json:"routeChanges"`
	RebalancingRecommended  bool               `json:"rebalancingRecommended"`
	LastUpdated             time.Time          `json:"lastUpdated"`
}

// RerouteOrderRequest represents a request to reroute an order
type RerouteOrderRequest struct {
	OrderID      string   `json:"orderId"`
	CurrentPath  string   `json:"currentPath"`
	Reason       string   `json:"reason"`
	Requirements []string `json:"requirements"`
	Priority     string   `json:"priority"`
	ForceReroute bool     `json:"forceReroute"`
}

// RerouteOrderResponse represents the rerouting decision
type RerouteOrderResponse struct {
	NewStationID string    `json:"newStationId"`
	Score        float64   `json:"score"`
	Confidence   float64   `json:"confidence"`
	RerouteTime  time.Time `json:"rerouteTime"`
}

// EscalateProcessPathRequest represents a request to escalate a process path
type EscalateProcessPathRequest struct {
	PathID      string `json:"pathId"`
	ToTier      string `json:"toTier"`
	Trigger     string `json:"trigger"`
	Reason      string `json:"reason"`
	EscalatedBy string `json:"escalatedBy,omitempty"`
}

// EscalateProcessPathResponse represents the escalation result
type EscalateProcessPathResponse struct {
	PathID           string    `json:"pathId"`
	NewTier          string    `json:"newTier"`
	EscalatedAt      time.Time `json:"escalatedAt"`
	FallbackStations []string  `json:"fallbackStations,omitempty"`
}

// DowngradeProcessPathRequest represents a request to downgrade a process path
type DowngradeProcessPathRequest struct {
	PathID       string `json:"pathId"`
	ToTier       string `json:"toTier"`
	Reason       string `json:"reason"`
	DowngradedBy string `json:"downgradedBy,omitempty"`
}

// DowngradeProcessPathResponse represents the downgrade result
type DowngradeProcessPathResponse struct {
	PathID       string    `json:"pathId"`
	NewTier      string    `json:"newTier"`
	DowngradedAt time.Time `json:"downgradedAt"`
}
