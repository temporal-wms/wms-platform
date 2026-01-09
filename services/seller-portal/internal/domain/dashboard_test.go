package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestPeriodConstruction tests period creation
func TestPeriodConstruction(t *testing.T) {
	now := time.Now()
	start := now.AddDate(0, 0, -7)

	period := Period{
		Start: start,
		End:   now,
		Type:  "week",
	}

	assert.Equal(t, "week", period.Type)
	assert.True(t, period.Start.Before(period.End))
}

// TestDashboardSummaryConstruction tests dashboard summary
func TestDashboardSummaryConstruction(t *testing.T) {
	summary := DashboardSummary{
		SellerID: "SLR-001",
		TenantID: "TNT-001",
		Period: Period{
			Start: time.Now().AddDate(0, 0, -1),
			End:   time.Now(),
			Type:  "today",
		},
		OrderMetrics: OrderMetrics{
			TotalOrders:   100,
			PendingOrders: 10,
		},
		InventoryMetrics: InventoryMetrics{
			TotalSKUs:      50,
			LowStockSKUs:   5,
			OutOfStockSKUs: 2,
		},
		GeneratedAt: time.Now(),
	}

	assert.Equal(t, "SLR-001", summary.SellerID)
	assert.Equal(t, int64(100), summary.OrderMetrics.TotalOrders)
	assert.Equal(t, int64(50), summary.InventoryMetrics.TotalSKUs)
}

// TestOrderMetricsCalculations tests order metrics
func TestOrderMetricsCalculations(t *testing.T) {
	metrics := OrderMetrics{
		TotalOrders:      100,
		PendingOrders:    15,
		InProgressOrders: 20,
		ShippedOrders:    45,
		DeliveredOrders:  15,
		CancelledOrders:  5,
		TotalRevenue:     5000.00,
		FulfillmentRate:  95.5,
		ShipOnTimeRate:   92.3,
		OrdersByChannel: map[string]int64{
			"shopify": 60,
			"amazon":  40,
		},
		OrdersByDay: []DailyCount{
			{Date: "2024-12-20", Count: 20},
			{Date: "2024-12-21", Count: 25},
			{Date: "2024-12-22", Count: 30},
		},
	}

	// Verify sum of status orders
	statusSum := metrics.PendingOrders + metrics.InProgressOrders +
		metrics.ShippedOrders + metrics.DeliveredOrders + metrics.CancelledOrders
	assert.Equal(t, int64(100), statusSum)

	// Verify channel totals
	var channelSum int64
	for _, count := range metrics.OrdersByChannel {
		channelSum += count
	}
	assert.Equal(t, int64(100), channelSum)

	// Calculate average order value
	if metrics.TotalOrders > 0 {
		metrics.AverageOrderValue = metrics.TotalRevenue / float64(metrics.TotalOrders)
	}
	assert.Equal(t, 50.00, metrics.AverageOrderValue)
}

// TestInventoryMetricsConstruction tests inventory metrics
func TestInventoryMetricsConstruction(t *testing.T) {
	metrics := InventoryMetrics{
		TotalSKUs:        150,
		TotalUnits:       10000,
		LowStockSKUs:     10,
		OutOfStockSKUs:   3,
		OverstockedSKUs:  5,
		AgeingInventory:  500,
		InventoryValue:   75000.00,
		StorageFees:      250.00,
		ByWarehouse: []WarehouseStock{
			{WarehouseID: "WH-001", WarehouseName: "East DC", TotalUnits: 6000, StorageUsed: 5000},
			{WarehouseID: "WH-002", WarehouseName: "West DC", TotalUnits: 4000, StorageUsed: 3500},
		},
		TopSellingProducts: []ProductMetric{
			{SKU: "PROD-001", Name: "Best Seller", UnitsSold: 500, Revenue: 5000},
			{SKU: "PROD-002", Name: "Popular Item", UnitsSold: 300, Revenue: 3000},
		},
	}

	// Verify warehouse totals
	var warehouseUnits int64
	for _, wh := range metrics.ByWarehouse {
		warehouseUnits += wh.TotalUnits
	}
	assert.Equal(t, metrics.TotalUnits, warehouseUnits)

	// Verify top selling products sorted
	assert.GreaterOrEqual(t, metrics.TopSellingProducts[0].UnitsSold, metrics.TopSellingProducts[1].UnitsSold)
}

// TestBillingMetricsConstruction tests billing metrics
func TestBillingMetricsConstruction(t *testing.T) {
	metrics := BillingMetrics{
		CurrentBalance:    500.00,
		PendingCharges:    150.00,
		LastInvoiceAmount: 1200.00,
		LastInvoiceDate:   time.Now().AddDate(0, 0, -15),
		NextInvoiceDate:   time.Now().AddDate(0, 0, 15),
		MTDCharges:        750.00,
		FeeBreakdown: FeeBreakdown{
			StorageFees:     200.00,
			PickFees:        300.00,
			PackFees:        150.00,
			ShippingFees:    500.00,
			ReceivingFees:   50.00,
			ReturnFees:      25.00,
			SpecialHandling: 10.00,
			OtherFees:       15.00,
		},
	}

	// Calculate total fees
	metrics.FeeBreakdown.Total = metrics.FeeBreakdown.StorageFees +
		metrics.FeeBreakdown.PickFees +
		metrics.FeeBreakdown.PackFees +
		metrics.FeeBreakdown.ShippingFees +
		metrics.FeeBreakdown.ReceivingFees +
		metrics.FeeBreakdown.ReturnFees +
		metrics.FeeBreakdown.SpecialHandling +
		metrics.FeeBreakdown.OtherFees

	assert.Equal(t, 1250.00, metrics.FeeBreakdown.Total)
}

// TestChannelMetricsConstruction tests channel metrics
func TestChannelMetricsConstruction(t *testing.T) {
	metrics := ChannelMetrics{
		ChannelID:     "CH-001",
		ChannelName:   "My Shopify Store",
		ChannelType:   "shopify",
		Status:        "active",
		TotalOrders:   500,
		PendingOrders: 25,
		LastSyncAt:    time.Now().Add(-15 * time.Minute),
		SyncStatus:    "completed",
		ErrorCount:    0,
	}

	assert.Equal(t, "shopify", metrics.ChannelType)
	assert.Equal(t, "active", metrics.Status)
	assert.Equal(t, int64(0), metrics.ErrorCount)
}

// TestAlertConstruction tests alert creation
func TestAlertConstruction(t *testing.T) {
	alert := Alert{
		ID:        "alert-001",
		Type:      AlertTypeLowStock,
		Severity:  "warning",
		Title:     "Low Stock Alert",
		Message:   "5 SKUs are running low on stock",
		ActionURL: "/inventory?status=low_stock",
		CreatedAt: time.Now(),
		Read:      false,
	}

	assert.Equal(t, AlertTypeLowStock, alert.Type)
	assert.Equal(t, "warning", alert.Severity)
	assert.False(t, alert.Read)
}

// TestAlertTypes tests alert type constants
func TestAlertTypes(t *testing.T) {
	alertTypes := []AlertType{
		AlertTypeLowStock,
		AlertTypeOutOfStock,
		AlertTypeOrderIssue,
		AlertTypeShippingDelay,
		AlertTypeBillingDue,
		AlertTypeChannelError,
		AlertTypePerformance,
		AlertTypeAnnouncement,
	}

	for _, at := range alertTypes {
		assert.NotEmpty(t, string(at))
	}

	// Verify specific values
	assert.Equal(t, AlertType("low_stock"), AlertTypeLowStock)
	assert.Equal(t, AlertType("out_of_stock"), AlertTypeOutOfStock)
	assert.Equal(t, AlertType("channel_error"), AlertTypeChannelError)
}

// TestOrderFilterConstruction tests order filter
func TestOrderFilterConstruction(t *testing.T) {
	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()

	filter := OrderFilter{
		SellerID:  "SLR-001",
		Status:    []string{"pending", "processing"},
		ChannelID: "CH-001",
		StartDate: &startDate,
		EndDate:   &endDate,
		Search:    "customer@email.com",
		Page:      1,
		PageSize:  50,
		SortBy:    "createdAt",
		SortOrder: "desc",
	}

	assert.Equal(t, "SLR-001", filter.SellerID)
	assert.Len(t, filter.Status, 2)
	assert.Equal(t, 50, filter.PageSize)
	assert.Equal(t, "desc", filter.SortOrder)
}

// TestInventoryFilterConstruction tests inventory filter
func TestInventoryFilterConstruction(t *testing.T) {
	filter := InventoryFilter{
		SellerID:    "SLR-001",
		WarehouseID: "WH-001",
		Status:      []string{"low_stock", "out_of_stock"},
		Search:      "PROD-",
		Page:        1,
		PageSize:    100,
		SortBy:      "available",
		SortOrder:   "asc",
	}

	assert.Equal(t, "WH-001", filter.WarehouseID)
	assert.Contains(t, filter.Status, "low_stock")
}

// TestInvoiceFilterConstruction tests invoice filter
func TestInvoiceFilterConstruction(t *testing.T) {
	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	filter := InvoiceFilter{
		SellerID:  "SLR-001",
		Status:    []string{"finalized", "paid"},
		StartDate: &startDate,
		EndDate:   &endDate,
		Page:      1,
		PageSize:  20,
	}

	assert.Equal(t, "SLR-001", filter.SellerID)
	assert.Len(t, filter.Status, 2)
}

// TestSellerOrderConstruction tests seller order
func TestSellerOrderConstruction(t *testing.T) {
	now := time.Now()
	shippedAt := now.Add(-24 * time.Hour)

	order := SellerOrder{
		OrderID:         "ORD-001",
		ExternalOrderID: "12345",
		ChannelID:       "CH-001",
		ChannelName:     "Shopify Store",
		Status:          "shipped",
		CustomerName:    "John Doe",
		CustomerEmail:   "john@example.com",
		TotalAmount:     99.99,
		ItemCount:       3,
		TrackingNumber:  "1Z999AA10123456784",
		Carrier:         "UPS",
		ShippingAddress: Address{
			Name:       "John Doe",
			Address1:   "123 Main St",
			City:       "New York",
			Province:   "NY",
			PostalCode: "10001",
			Country:    "US",
		},
		CreatedAt: now.Add(-48 * time.Hour),
		ShippedAt: &shippedAt,
		LineItems: []LineItem{
			{SKU: "PROD-001", Name: "Widget", Quantity: 2, UnitPrice: 29.99, TotalPrice: 59.98},
			{SKU: "PROD-002", Name: "Gadget", Quantity: 1, UnitPrice: 39.99, TotalPrice: 39.99},
		},
	}

	assert.Equal(t, "shipped", order.Status)
	assert.Equal(t, "UPS", order.Carrier)
	assert.Len(t, order.LineItems, 2)

	// Verify line items total
	var itemsTotal float64
	for _, item := range order.LineItems {
		itemsTotal += item.TotalPrice
	}
	// Note: TotalAmount may include tax/shipping, so just verify items total
	assert.InDelta(t, 99.97, itemsTotal, 0.01)
}

// TestSellerInventoryConstruction tests seller inventory
func TestSellerInventoryConstruction(t *testing.T) {
	inventory := SellerInventory{
		SKU:          "PROD-001",
		Name:         "Premium Widget",
		Available:    95,
		Reserved:     25,
		InTransit:    50,
		TotalOnHand:  120,
		ReorderPoint: 50,
		DaysOfSupply: 14,
		Status:       "available",
		Locations: []InventoryLocation{
			{WarehouseID: "WH-001", WarehouseName: "East DC", LocationID: "A-1-1", Quantity: 70},
			{WarehouseID: "WH-002", WarehouseName: "West DC", LocationID: "B-2-1", Quantity: 50},
		},
	}

	// Verify total on hand
	var locationTotal int64
	for _, loc := range inventory.Locations {
		locationTotal += loc.Quantity
	}
	assert.Equal(t, inventory.TotalOnHand, locationTotal)

	// Verify available + reserved = total on hand
	assert.Equal(t, inventory.TotalOnHand, inventory.Available+inventory.Reserved)
}

// TestSellerInvoiceConstruction tests seller invoice
func TestSellerInvoiceConstruction(t *testing.T) {
	now := time.Now()
	paidAt := now.Add(-24 * time.Hour)

	invoice := SellerInvoice{
		InvoiceID:     "INV-001",
		InvoiceNumber: "INV-202412-001",
		Status:        "paid",
		PeriodStart:   now.AddDate(0, -1, 0),
		PeriodEnd:     now.AddDate(0, 0, -1),
		Subtotal:      1000.00,
		Tax:           80.00,
		Total:         1080.00,
		DueDate:       now.AddDate(0, 0, 15),
		PaidAt:        &paidAt,
		LineItems: []InvoiceLineItem{
			{Description: "Storage Fees", Quantity: 30, UnitPrice: 10.00, Amount: 300.00, FeeType: "storage"},
			{Description: "Pick Fees", Quantity: 500, UnitPrice: 0.25, Amount: 125.00, FeeType: "pick"},
			{Description: "Pack Fees", Quantity: 100, UnitPrice: 1.50, Amount: 150.00, FeeType: "pack"},
			{Description: "Shipping", Quantity: 100, UnitPrice: 4.25, Amount: 425.00, FeeType: "shipping"},
		},
	}

	// Verify line items sum to subtotal
	var lineItemsTotal float64
	for _, item := range invoice.LineItems {
		lineItemsTotal += item.Amount
	}
	assert.Equal(t, invoice.Subtotal, lineItemsTotal)

	// Verify total = subtotal + tax
	assert.Equal(t, invoice.Subtotal+invoice.Tax, invoice.Total)
}

// TestAPIKeyInfoConstruction tests API key info
func TestAPIKeyInfoConstruction(t *testing.T) {
	now := time.Now()
	lastUsed := now.Add(-1 * time.Hour)
	expires := now.AddDate(0, 6, 0)

	keyInfo := APIKeyInfo{
		KeyID:       "key-001",
		Name:        "Production API Key",
		Prefix:      "wms_live",
		Permissions: []string{"orders:read", "orders:write", "inventory:read"},
		LastUsedAt:  &lastUsed,
		ExpiresAt:   &expires,
		CreatedAt:   now.AddDate(0, -1, 0),
		Status:      "active",
	}

	assert.Equal(t, "active", keyInfo.Status)
	assert.Len(t, keyInfo.Permissions, 3)
	assert.Contains(t, keyInfo.Permissions, "orders:read")
}

// TestAddressConstruction tests address
func TestAddressConstruction(t *testing.T) {
	address := Address{
		Name:       "John Doe",
		Address1:   "123 Main Street",
		Address2:   "Apt 4B",
		City:       "New York",
		Province:   "NY",
		PostalCode: "10001",
		Country:    "US",
	}

	assert.NotEmpty(t, address.Name)
	assert.NotEmpty(t, address.Address1)
	assert.Equal(t, "US", address.Country)
}

// TestLineItemCalculation tests line item total calculation
func TestLineItemCalculation(t *testing.T) {
	item := LineItem{
		SKU:       "PROD-001",
		Name:      "Test Product",
		Quantity:  5,
		UnitPrice: 19.99,
	}

	// Calculate total
	item.TotalPrice = float64(item.Quantity) * item.UnitPrice
	assert.InDelta(t, 99.95, item.TotalPrice, 0.01)
}

// TestWarehouseStockConstruction tests warehouse stock
func TestWarehouseStockConstruction(t *testing.T) {
	stock := WarehouseStock{
		WarehouseID:   "WH-001",
		WarehouseName: "East Coast Distribution Center",
		TotalUnits:    5000,
		StorageUsed:   4500.5,
	}

	assert.Equal(t, "WH-001", stock.WarehouseID)
	assert.Equal(t, int64(5000), stock.TotalUnits)
	assert.Greater(t, stock.StorageUsed, float64(0))
}

// TestProductMetricConstruction tests product metric
func TestProductMetricConstruction(t *testing.T) {
	metric := ProductMetric{
		SKU:          "PROD-001",
		Name:         "Best Seller Widget",
		Quantity:     500,
		Revenue:      9999.00,
		UnitsSold:    200,
		DaysOfSupply: 30,
	}

	// Calculate implied unit price
	if metric.UnitsSold > 0 {
		unitPrice := metric.Revenue / float64(metric.UnitsSold)
		assert.InDelta(t, 49.995, unitPrice, 0.01)
	}
}

// TestDailyCountConstruction tests daily count
func TestDailyCountConstruction(t *testing.T) {
	counts := []DailyCount{
		{Date: "2024-12-20", Count: 45},
		{Date: "2024-12-21", Count: 52},
		{Date: "2024-12-22", Count: 38},
		{Date: "2024-12-23", Count: 61},
		{Date: "2024-12-24", Count: 55},
	}

	// Calculate average
	var total int64
	for _, c := range counts {
		total += c.Count
	}
	avg := float64(total) / float64(len(counts))
	assert.InDelta(t, 50.2, avg, 0.1)
}

// TestDailyChargeConstruction tests daily charge
func TestDailyChargeConstruction(t *testing.T) {
	charges := []DailyCharge{
		{Date: "2024-12-20", Amount: 45.50},
		{Date: "2024-12-21", Amount: 52.25},
		{Date: "2024-12-22", Amount: 38.75},
	}

	// Calculate total
	var total float64
	for _, c := range charges {
		total += c.Amount
	}
	assert.InDelta(t, 136.50, total, 0.01)
}

// BenchmarkDashboardSummaryConstruction benchmarks summary creation
func BenchmarkDashboardSummaryConstruction(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DashboardSummary{
			SellerID: "SLR-001",
			TenantID: "TNT-001",
			Period: Period{
				Start: time.Now().AddDate(0, 0, -1),
				End:   time.Now(),
				Type:  "today",
			},
			OrderMetrics: OrderMetrics{
				TotalOrders:     100,
				OrdersByChannel: make(map[string]int64),
				OrdersByDay:     make([]DailyCount, 0),
			},
			InventoryMetrics: InventoryMetrics{
				TotalSKUs:          50,
				ByWarehouse:        make([]WarehouseStock, 0),
				TopSellingProducts: make([]ProductMetric, 0),
				SlowMovingProducts: make([]ProductMetric, 0),
			},
			BillingMetrics: BillingMetrics{
				ChargesByDay: make([]DailyCharge, 0),
			},
			ChannelMetrics: make([]ChannelMetrics, 0),
			Alerts:         make([]Alert, 0),
			GeneratedAt:    time.Now(),
		}
	}
}
