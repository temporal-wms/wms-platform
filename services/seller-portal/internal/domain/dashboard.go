package domain

import "time"

// DashboardSummary represents the seller dashboard overview
type DashboardSummary struct {
	SellerID         string           `json:"sellerId"`
	TenantID         string           `json:"tenantId"`
	Period           Period           `json:"period"`
	OrderMetrics     OrderMetrics     `json:"orderMetrics"`
	InventoryMetrics InventoryMetrics `json:"inventoryMetrics"`
	BillingMetrics   BillingMetrics   `json:"billingMetrics"`
	ChannelMetrics   []ChannelMetrics `json:"channelMetrics"`
	Alerts           []Alert          `json:"alerts"`
	GeneratedAt      time.Time        `json:"generatedAt"`
}

// Period represents a time period for metrics
type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Type  string    `json:"type"` // today, week, month, custom
}

// OrderMetrics represents order-related metrics
type OrderMetrics struct {
	TotalOrders       int64            `json:"totalOrders"`
	PendingOrders     int64            `json:"pendingOrders"`
	InProgressOrders  int64            `json:"inProgressOrders"`
	ShippedOrders     int64            `json:"shippedOrders"`
	DeliveredOrders   int64            `json:"deliveredOrders"`
	CancelledOrders   int64            `json:"cancelledOrders"`
	ReturnedOrders    int64            `json:"returnedOrders"`
	TotalRevenue      float64          `json:"totalRevenue"`
	AverageOrderValue float64          `json:"averageOrderValue"`
	OrdersByChannel   map[string]int64 `json:"ordersByChannel"`
	OrdersByDay       []DailyCount     `json:"ordersByDay"`
	FulfillmentRate   float64          `json:"fulfillmentRate"` // Percentage of on-time fulfillment
	ShipOnTimeRate    float64          `json:"shipOnTimeRate"`  // Percentage shipped by promised date
}

// DailyCount represents a count for a specific date
type DailyCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// InventoryMetrics represents inventory-related metrics
type InventoryMetrics struct {
	TotalSKUs         int64              `json:"totalSkus"`
	TotalUnits        int64              `json:"totalUnits"`
	LowStockSKUs      int64              `json:"lowStockSkus"`
	OutOfStockSKUs    int64              `json:"outOfStockSkus"`
	OverstockedSKUs   int64              `json:"overstockedSkus"`
	AgeingInventory   int64              `json:"ageingInventory"` // Units > 90 days old
	InventoryValue    float64            `json:"inventoryValue"`
	StorageFees       float64            `json:"storageFees"`
	ByWarehouse       []WarehouseStock   `json:"byWarehouse"`
	TopSellingProducts []ProductMetric   `json:"topSellingProducts"`
	SlowMovingProducts []ProductMetric   `json:"slowMovingProducts"`
}

// WarehouseStock represents inventory in a warehouse
type WarehouseStock struct {
	WarehouseID   string  `json:"warehouseId"`
	WarehouseName string  `json:"warehouseName"`
	TotalUnits    int64   `json:"totalUnits"`
	StorageUsed   float64 `json:"storageUsed"` // Cubic feet
}

// ProductMetric represents metrics for a product
type ProductMetric struct {
	SKU           string  `json:"sku"`
	Name          string  `json:"name"`
	Quantity      int64   `json:"quantity"`
	Revenue       float64 `json:"revenue"`
	UnitsSold     int64   `json:"unitsSold"`
	DaysOfSupply  int     `json:"daysOfSupply"`
}

// BillingMetrics represents billing-related metrics
type BillingMetrics struct {
	CurrentBalance    float64          `json:"currentBalance"`
	PendingCharges    float64          `json:"pendingCharges"`
	LastInvoiceAmount float64          `json:"lastInvoiceAmount"`
	LastInvoiceDate   time.Time        `json:"lastInvoiceDate,omitempty"`
	NextInvoiceDate   time.Time        `json:"nextInvoiceDate,omitempty"`
	MTDCharges        float64          `json:"mtdCharges"` // Month-to-date
	FeeBreakdown      FeeBreakdown     `json:"feeBreakdown"`
	ChargesByDay      []DailyCharge    `json:"chargesByDay"`
}

// FeeBreakdown breaks down fees by type
type FeeBreakdown struct {
	StorageFees       float64 `json:"storageFees"`
	PickFees          float64 `json:"pickFees"`
	PackFees          float64 `json:"packFees"`
	ShippingFees      float64 `json:"shippingFees"`
	ReceivingFees     float64 `json:"receivingFees"`
	ReturnFees        float64 `json:"returnFees"`
	SpecialHandling   float64 `json:"specialHandling"`
	OtherFees         float64 `json:"otherFees"`
	Total             float64 `json:"total"`
}

// DailyCharge represents charges for a specific date
type DailyCharge struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

// ChannelMetrics represents metrics for a sales channel
type ChannelMetrics struct {
	ChannelID       string    `json:"channelId"`
	ChannelName     string    `json:"channelName"`
	ChannelType     string    `json:"channelType"`
	Status          string    `json:"status"`
	TotalOrders     int64     `json:"totalOrders"`
	PendingOrders   int64     `json:"pendingOrders"`
	LastSyncAt      time.Time `json:"lastSyncAt,omitempty"`
	SyncStatus      string    `json:"syncStatus"`
	ErrorCount      int64     `json:"errorCount"`
}

// Alert represents an alert for the seller
type Alert struct {
	ID        string    `json:"id"`
	Type      AlertType `json:"type"`
	Severity  string    `json:"severity"` // info, warning, critical
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	ActionURL string    `json:"actionUrl,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	Read      bool      `json:"read"`
}

// AlertType represents types of alerts
type AlertType string

const (
	AlertTypeLowStock       AlertType = "low_stock"
	AlertTypeOutOfStock     AlertType = "out_of_stock"
	AlertTypeOrderIssue     AlertType = "order_issue"
	AlertTypeShippingDelay  AlertType = "shipping_delay"
	AlertTypeBillingDue     AlertType = "billing_due"
	AlertTypeChannelError   AlertType = "channel_error"
	AlertTypePerformance    AlertType = "performance"
	AlertTypeAnnouncement   AlertType = "announcement"
)

// OrderFilter represents filters for order queries
type OrderFilter struct {
	SellerID   string     `json:"sellerId"`
	Status     []string   `json:"status,omitempty"`
	ChannelID  string     `json:"channelId,omitempty"`
	StartDate  *time.Time `json:"startDate,omitempty"`
	EndDate    *time.Time `json:"endDate,omitempty"`
	Search     string     `json:"search,omitempty"` // Search by order ID, customer name
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	SortBy     string     `json:"sortBy"`
	SortOrder  string     `json:"sortOrder"` // asc, desc
}

// InventoryFilter represents filters for inventory queries
type InventoryFilter struct {
	SellerID     string   `json:"sellerId"`
	WarehouseID  string   `json:"warehouseId,omitempty"`
	Status       []string `json:"status,omitempty"` // available, low_stock, out_of_stock
	Search       string   `json:"search,omitempty"` // Search by SKU, name
	Page         int      `json:"page"`
	PageSize     int      `json:"pageSize"`
	SortBy       string   `json:"sortBy"`
	SortOrder    string   `json:"sortOrder"`
}

// InvoiceFilter represents filters for invoice queries
type InvoiceFilter struct {
	SellerID  string     `json:"sellerId"`
	Status    []string   `json:"status,omitempty"` // draft, finalized, paid
	StartDate *time.Time `json:"startDate,omitempty"`
	EndDate   *time.Time `json:"endDate,omitempty"`
	Page      int        `json:"page"`
	PageSize  int        `json:"pageSize"`
}

// SellerOrder represents an order in the seller portal context
type SellerOrder struct {
	OrderID         string      `json:"orderId"`
	ExternalOrderID string      `json:"externalOrderId,omitempty"`
	ChannelID       string      `json:"channelId,omitempty"`
	ChannelName     string      `json:"channelName,omitempty"`
	Status          string      `json:"status"`
	CustomerName    string      `json:"customerName"`
	CustomerEmail   string      `json:"customerEmail"`
	TotalAmount     float64     `json:"totalAmount"`
	ItemCount       int         `json:"itemCount"`
	TrackingNumber  string      `json:"trackingNumber,omitempty"`
	Carrier         string      `json:"carrier,omitempty"`
	ShippingAddress Address     `json:"shippingAddress"`
	CreatedAt       time.Time   `json:"createdAt"`
	ShippedAt       *time.Time  `json:"shippedAt,omitempty"`
	DeliveredAt     *time.Time  `json:"deliveredAt,omitempty"`
	LineItems       []LineItem  `json:"lineItems"`
}

// Address represents a shipping address
type Address struct {
	Name       string `json:"name"`
	Address1   string `json:"address1"`
	Address2   string `json:"address2,omitempty"`
	City       string `json:"city"`
	Province   string `json:"province"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

// LineItem represents an order line item
type LineItem struct {
	SKU        string  `json:"sku"`
	Name       string  `json:"name"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unitPrice"`
	TotalPrice float64 `json:"totalPrice"`
}

// SellerInventory represents inventory in the seller portal context
type SellerInventory struct {
	SKU              string    `json:"sku"`
	Name             string    `json:"name"`
	Available        int64     `json:"available"`
	Reserved         int64     `json:"reserved"`
	InTransit        int64     `json:"inTransit"`
	TotalOnHand      int64     `json:"totalOnHand"`
	ReorderPoint     int64     `json:"reorderPoint"`
	DaysOfSupply     int       `json:"daysOfSupply"`
	Status           string    `json:"status"` // available, low_stock, out_of_stock
	LastRestocked    time.Time `json:"lastRestocked,omitempty"`
	Locations        []InventoryLocation `json:"locations"`
}

// InventoryLocation represents inventory at a location
type InventoryLocation struct {
	WarehouseID   string `json:"warehouseId"`
	WarehouseName string `json:"warehouseName"`
	LocationID    string `json:"locationId"`
	Quantity      int64  `json:"quantity"`
}

// SellerInvoice represents an invoice in the seller portal context
type SellerInvoice struct {
	InvoiceID     string        `json:"invoiceId"`
	InvoiceNumber string        `json:"invoiceNumber"`
	Status        string        `json:"status"`
	PeriodStart   time.Time     `json:"periodStart"`
	PeriodEnd     time.Time     `json:"periodEnd"`
	Subtotal      float64       `json:"subtotal"`
	Tax           float64       `json:"tax"`
	Total         float64       `json:"total"`
	DueDate       time.Time     `json:"dueDate"`
	PaidAt        *time.Time    `json:"paidAt,omitempty"`
	LineItems     []InvoiceLineItem `json:"lineItems"`
	DownloadURL   string        `json:"downloadUrl,omitempty"`
}

// InvoiceLineItem represents a line item on an invoice
type InvoiceLineItem struct {
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	Amount      float64 `json:"amount"`
	FeeType     string  `json:"feeType"`
}

// APIKeyInfo represents API key information for display
type APIKeyInfo struct {
	KeyID       string    `json:"keyId"`
	Name        string    `json:"name"`
	Prefix      string    `json:"prefix"` // First 8 chars for identification
	Permissions []string  `json:"permissions"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	Status      string     `json:"status"` // active, revoked
}
