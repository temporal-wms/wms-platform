package application

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"seller-portal/internal/domain"
)

// TestGenerateAlertsLowStock tests low stock alert generation
func TestGenerateAlertsLowStock(t *testing.T) {
	service := &DashboardService{}
	summary := &domain.DashboardSummary{
		SellerID: "SLR-001",
		InventoryMetrics: domain.InventoryMetrics{
			LowStockSKUs:   5,
			OutOfStockSKUs: 0,
		},
		OrderMetrics: domain.OrderMetrics{
			PendingOrders:   5,
			FulfillmentRate: 95,
		},
		ChannelMetrics: []domain.ChannelMetrics{},
	}

	alerts := service.generateAlerts(nil, "SLR-001", summary)

	// Should have low stock alert
	var foundLowStock bool
	for _, alert := range alerts {
		if alert.Type == domain.AlertTypeLowStock {
			foundLowStock = true
			assert.Equal(t, "warning", alert.Severity)
			assert.Contains(t, alert.Message, "5 SKUs")
			assert.Equal(t, "/inventory?status=low_stock", alert.ActionURL)
		}
	}
	assert.True(t, foundLowStock, "Expected low stock alert")
}

// TestGenerateAlertsOutOfStock tests out of stock alert generation
func TestGenerateAlertsOutOfStock(t *testing.T) {
	service := &DashboardService{}
	summary := &domain.DashboardSummary{
		SellerID: "SLR-001",
		InventoryMetrics: domain.InventoryMetrics{
			LowStockSKUs:   0,
			OutOfStockSKUs: 3,
		},
		OrderMetrics: domain.OrderMetrics{
			PendingOrders:   5,
			FulfillmentRate: 95,
		},
		ChannelMetrics: []domain.ChannelMetrics{},
	}

	alerts := service.generateAlerts(nil, "SLR-001", summary)

	// Should have out of stock alert
	var foundOutOfStock bool
	for _, alert := range alerts {
		if alert.Type == domain.AlertTypeOutOfStock {
			foundOutOfStock = true
			assert.Equal(t, "critical", alert.Severity)
			assert.Contains(t, alert.Message, "3 SKUs")
		}
	}
	assert.True(t, foundOutOfStock, "Expected out of stock alert")
}

// TestGenerateAlertsHighPendingOrders tests pending orders alert
func TestGenerateAlertsHighPendingOrders(t *testing.T) {
	service := &DashboardService{}
	summary := &domain.DashboardSummary{
		SellerID: "SLR-001",
		InventoryMetrics: domain.InventoryMetrics{
			LowStockSKUs:   0,
			OutOfStockSKUs: 0,
		},
		OrderMetrics: domain.OrderMetrics{
			PendingOrders:   15, // More than 10
			FulfillmentRate: 95,
		},
		ChannelMetrics: []domain.ChannelMetrics{},
	}

	alerts := service.generateAlerts(nil, "SLR-001", summary)

	// Should have pending orders alert
	var foundPending bool
	for _, alert := range alerts {
		if alert.Type == domain.AlertTypeOrderIssue {
			foundPending = true
			assert.Equal(t, "warning", alert.Severity)
			assert.Contains(t, alert.Message, "15 pending orders")
		}
	}
	assert.True(t, foundPending, "Expected pending orders alert")
}

// TestGenerateAlertsNoPendingAlertWhenLow tests no alert for low pending
func TestGenerateAlertsNoPendingAlertWhenLow(t *testing.T) {
	service := &DashboardService{}
	summary := &domain.DashboardSummary{
		SellerID: "SLR-001",
		InventoryMetrics: domain.InventoryMetrics{
			LowStockSKUs:   0,
			OutOfStockSKUs: 0,
		},
		OrderMetrics: domain.OrderMetrics{
			PendingOrders:   5, // Less than 10
			FulfillmentRate: 95,
		},
		ChannelMetrics: []domain.ChannelMetrics{},
	}

	alerts := service.generateAlerts(nil, "SLR-001", summary)

	// Should NOT have pending orders alert
	for _, alert := range alerts {
		assert.NotEqual(t, domain.AlertTypeOrderIssue, alert.Type, "Should not have pending orders alert for 5 pending")
	}
}

// TestGenerateAlertsChannelError tests channel error alert
func TestGenerateAlertsChannelError(t *testing.T) {
	service := &DashboardService{}
	summary := &domain.DashboardSummary{
		SellerID: "SLR-001",
		InventoryMetrics: domain.InventoryMetrics{
			LowStockSKUs:   0,
			OutOfStockSKUs: 0,
		},
		OrderMetrics: domain.OrderMetrics{
			PendingOrders:   5,
			FulfillmentRate: 95,
		},
		ChannelMetrics: []domain.ChannelMetrics{
			{
				ChannelID:   "CH-001",
				ChannelName: "Shopify Store",
				ErrorCount:  3,
			},
		},
	}

	alerts := service.generateAlerts(nil, "SLR-001", summary)

	// Should have channel error alert
	var foundChannelError bool
	for _, alert := range alerts {
		if alert.Type == domain.AlertTypeChannelError {
			foundChannelError = true
			assert.Contains(t, alert.Title, "Shopify Store")
			assert.Contains(t, alert.Message, "3 errors")
		}
	}
	assert.True(t, foundChannelError, "Expected channel error alert")
}

// TestGenerateAlertsLowFulfillmentRate tests performance alert
func TestGenerateAlertsLowFulfillmentRate(t *testing.T) {
	service := &DashboardService{}
	summary := &domain.DashboardSummary{
		SellerID: "SLR-001",
		InventoryMetrics: domain.InventoryMetrics{
			LowStockSKUs:   0,
			OutOfStockSKUs: 0,
		},
		OrderMetrics: domain.OrderMetrics{
			PendingOrders:   5,
			FulfillmentRate: 85, // Below 90%
		},
		ChannelMetrics: []domain.ChannelMetrics{},
	}

	alerts := service.generateAlerts(nil, "SLR-001", summary)

	// Should have performance alert
	var foundPerformance bool
	for _, alert := range alerts {
		if alert.Type == domain.AlertTypePerformance {
			foundPerformance = true
			assert.Equal(t, "info", alert.Severity)
			assert.Contains(t, alert.Message, "85.0%")
		}
	}
	assert.True(t, foundPerformance, "Expected performance alert")
}

// TestGenerateAlertsNoPerformanceAlertWhenGood tests no alert when performance is good
func TestGenerateAlertsNoPerformanceAlertWhenGood(t *testing.T) {
	service := &DashboardService{}
	summary := &domain.DashboardSummary{
		SellerID: "SLR-001",
		InventoryMetrics: domain.InventoryMetrics{
			LowStockSKUs:   0,
			OutOfStockSKUs: 0,
		},
		OrderMetrics: domain.OrderMetrics{
			PendingOrders:   5,
			FulfillmentRate: 95, // Above 90%
		},
		ChannelMetrics: []domain.ChannelMetrics{},
	}

	alerts := service.generateAlerts(nil, "SLR-001", summary)

	// Should NOT have performance alert
	for _, alert := range alerts {
		assert.NotEqual(t, domain.AlertTypePerformance, alert.Type, "Should not have performance alert when rate >= 90%")
	}
}

// TestGenerateAlertsMultiple tests multiple alerts generation
func TestGenerateAlertsMultiple(t *testing.T) {
	service := &DashboardService{}
	summary := &domain.DashboardSummary{
		SellerID: "SLR-001",
		InventoryMetrics: domain.InventoryMetrics{
			LowStockSKUs:   5,
			OutOfStockSKUs: 2,
		},
		OrderMetrics: domain.OrderMetrics{
			PendingOrders:   15,
			FulfillmentRate: 85,
		},
		ChannelMetrics: []domain.ChannelMetrics{
			{ChannelID: "CH-001", ChannelName: "Shopify", ErrorCount: 2},
		},
	}

	alerts := service.generateAlerts(nil, "SLR-001", summary)

	// Should have multiple alerts
	alertTypes := make(map[domain.AlertType]bool)
	for _, alert := range alerts {
		alertTypes[alert.Type] = true
	}

	assert.True(t, alertTypes[domain.AlertTypeLowStock], "Expected low stock alert")
	assert.True(t, alertTypes[domain.AlertTypeOutOfStock], "Expected out of stock alert")
	assert.True(t, alertTypes[domain.AlertTypeOrderIssue], "Expected order issue alert")
	assert.True(t, alertTypes[domain.AlertTypePerformance], "Expected performance alert")
	assert.True(t, alertTypes[domain.AlertTypeChannelError], "Expected channel error alert")
}

// TestGenerateAlertsNoAlerts tests no alerts generation
func TestGenerateAlertsNoAlerts(t *testing.T) {
	service := &DashboardService{}
	summary := &domain.DashboardSummary{
		SellerID: "SLR-001",
		InventoryMetrics: domain.InventoryMetrics{
			LowStockSKUs:   0,
			OutOfStockSKUs: 0,
		},
		OrderMetrics: domain.OrderMetrics{
			PendingOrders:   5,
			FulfillmentRate: 0, // Zero means no data
		},
		ChannelMetrics: []domain.ChannelMetrics{
			{ChannelID: "CH-001", ChannelName: "Shopify", ErrorCount: 0},
		},
	}

	alerts := service.generateAlerts(nil, "SLR-001", summary)

	// Should have no alerts
	assert.Empty(t, alerts, "Expected no alerts")
}

// TestGenerateAlertsAlertIDFormat tests alert ID format
func TestGenerateAlertsAlertIDFormat(t *testing.T) {
	service := &DashboardService{}
	summary := &domain.DashboardSummary{
		SellerID: "SLR-001",
		InventoryMetrics: domain.InventoryMetrics{
			LowStockSKUs: 5,
		},
		OrderMetrics:   domain.OrderMetrics{},
		ChannelMetrics: []domain.ChannelMetrics{},
	}

	alerts := service.generateAlerts(nil, "SLR-001", summary)

	for _, alert := range alerts {
		assert.NotEmpty(t, alert.ID)
		assert.Contains(t, alert.ID, "SLR-001")
		assert.NotZero(t, alert.CreatedAt)
	}
}

// TestNewDashboardService tests service creation
func TestNewDashboardService(t *testing.T) {
	service := NewDashboardService(nil, nil, nil, nil, nil)

	assert.NotNil(t, service)
	assert.Nil(t, service.sellerClient)
	assert.Nil(t, service.orderClient)
	assert.Nil(t, service.inventoryClient)
	assert.Nil(t, service.billingClient)
	assert.Nil(t, service.channelClient)
}

// TestGetDashboardSummaryCreatesValidSummary tests summary creation
func TestGetDashboardSummaryCreatesValidSummary(t *testing.T) {
	// This test verifies the summary structure without actual clients
	service := NewDashboardService(nil, nil, nil, nil, nil)

	// Note: This will return partial data since clients are nil
	now := time.Now()
	period := domain.Period{
		Start: now.AddDate(0, 0, -1),
		End:   now,
		Type:  "today",
	}

	summary, err := service.GetDashboardSummary(nil, "SLR-001", period)

	// Even with nil clients, should not error - returns partial data
	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, "SLR-001", summary.SellerID)
	assert.Equal(t, "today", summary.Period.Type)
	assert.NotZero(t, summary.GeneratedAt)
}

// BenchmarkGenerateAlerts benchmarks alert generation
func BenchmarkGenerateAlerts(b *testing.B) {
	service := &DashboardService{}
	summary := &domain.DashboardSummary{
		SellerID: "SLR-001",
		InventoryMetrics: domain.InventoryMetrics{
			LowStockSKUs:   5,
			OutOfStockSKUs: 2,
		},
		OrderMetrics: domain.OrderMetrics{
			PendingOrders:   15,
			FulfillmentRate: 85,
		},
		ChannelMetrics: []domain.ChannelMetrics{
			{ChannelID: "CH-001", ChannelName: "Shopify", ErrorCount: 2},
			{ChannelID: "CH-002", ChannelName: "Amazon", ErrorCount: 1},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.generateAlerts(nil, "SLR-001", summary)
	}
}
