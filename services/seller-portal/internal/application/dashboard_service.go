package application

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/wms-platform/services/seller-portal/internal/domain"
	"github.com/wms-platform/services/seller-portal/internal/infrastructure/clients"
)

// DashboardService provides dashboard functionality
type DashboardService struct {
	sellerClient    *clients.SellerClient
	orderClient     *clients.OrderClient
	inventoryClient *clients.InventoryClient
	billingClient   *clients.BillingClient
	channelClient   *clients.ChannelClient
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(
	sellerClient *clients.SellerClient,
	orderClient *clients.OrderClient,
	inventoryClient *clients.InventoryClient,
	billingClient *clients.BillingClient,
	channelClient *clients.ChannelClient,
) *DashboardService {
	return &DashboardService{
		sellerClient:    sellerClient,
		orderClient:     orderClient,
		inventoryClient: inventoryClient,
		billingClient:   billingClient,
		channelClient:   channelClient,
	}
}

// GetDashboardSummary retrieves the complete dashboard summary
func (s *DashboardService) GetDashboardSummary(ctx context.Context, sellerID string, period domain.Period) (*domain.DashboardSummary, error) {
	summary := &domain.DashboardSummary{
		SellerID:    sellerID,
		Period:      period,
		GeneratedAt: time.Now(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	// Fetch order metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics, err := s.getOrderMetrics(ctx, sellerID, period)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, fmt.Errorf("order metrics: %w", err))
			return
		}
		summary.OrderMetrics = *metrics
	}()

	// Fetch inventory metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics, err := s.getInventoryMetrics(ctx, sellerID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, fmt.Errorf("inventory metrics: %w", err))
			return
		}
		summary.InventoryMetrics = *metrics
	}()

	// Fetch billing metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics, err := s.getBillingMetrics(ctx, sellerID, period)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, fmt.Errorf("billing metrics: %w", err))
			return
		}
		summary.BillingMetrics = *metrics
	}()

	// Fetch channel metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics, err := s.getChannelMetrics(ctx, sellerID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, fmt.Errorf("channel metrics: %w", err))
			return
		}
		summary.ChannelMetrics = metrics
	}()

	// Fetch alerts
	wg.Add(1)
	go func() {
		defer wg.Done()
		alerts := s.generateAlerts(ctx, sellerID, summary)
		mu.Lock()
		defer mu.Unlock()
		summary.Alerts = alerts
	}()

	wg.Wait()

	// Log errors but don't fail - return partial data
	for _, err := range errs {
		log.Printf("Warning: %v", err)
	}

	return summary, nil
}

func (s *DashboardService) getOrderMetrics(ctx context.Context, sellerID string, period domain.Period) (*domain.OrderMetrics, error) {
	startDate := period.Start.Format("2006-01-02")
	endDate := period.End.Format("2006-01-02")

	stats, err := s.orderClient.GetOrderStats(ctx, sellerID, startDate, endDate)
	if err != nil {
		// Return default metrics on error
		return &domain.OrderMetrics{
			OrdersByChannel: make(map[string]int64),
			OrdersByDay:     []domain.DailyCount{},
		}, nil
	}

	metrics := &domain.OrderMetrics{
		OrdersByChannel: make(map[string]int64),
		OrdersByDay:     []domain.DailyCount{},
	}

	// Parse stats response
	if v, ok := stats["totalOrders"].(float64); ok {
		metrics.TotalOrders = int64(v)
	}
	if v, ok := stats["pendingOrders"].(float64); ok {
		metrics.PendingOrders = int64(v)
	}
	if v, ok := stats["inProgressOrders"].(float64); ok {
		metrics.InProgressOrders = int64(v)
	}
	if v, ok := stats["shippedOrders"].(float64); ok {
		metrics.ShippedOrders = int64(v)
	}
	if v, ok := stats["deliveredOrders"].(float64); ok {
		metrics.DeliveredOrders = int64(v)
	}
	if v, ok := stats["cancelledOrders"].(float64); ok {
		metrics.CancelledOrders = int64(v)
	}
	if v, ok := stats["totalRevenue"].(float64); ok {
		metrics.TotalRevenue = v
	}
	if v, ok := stats["averageOrderValue"].(float64); ok {
		metrics.AverageOrderValue = v
	}
	if v, ok := stats["fulfillmentRate"].(float64); ok {
		metrics.FulfillmentRate = v
	}
	if v, ok := stats["shipOnTimeRate"].(float64); ok {
		metrics.ShipOnTimeRate = v
	}

	return metrics, nil
}

func (s *DashboardService) getInventoryMetrics(ctx context.Context, sellerID string) (*domain.InventoryMetrics, error) {
	stats, err := s.inventoryClient.GetInventoryStats(ctx, sellerID)
	if err != nil {
		return &domain.InventoryMetrics{
			ByWarehouse:        []domain.WarehouseStock{},
			TopSellingProducts: []domain.ProductMetric{},
			SlowMovingProducts: []domain.ProductMetric{},
		}, nil
	}

	metrics := &domain.InventoryMetrics{
		ByWarehouse:        []domain.WarehouseStock{},
		TopSellingProducts: []domain.ProductMetric{},
		SlowMovingProducts: []domain.ProductMetric{},
	}

	if v, ok := stats["totalSkus"].(float64); ok {
		metrics.TotalSKUs = int64(v)
	}
	if v, ok := stats["totalUnits"].(float64); ok {
		metrics.TotalUnits = int64(v)
	}
	if v, ok := stats["lowStockSkus"].(float64); ok {
		metrics.LowStockSKUs = int64(v)
	}
	if v, ok := stats["outOfStockSkus"].(float64); ok {
		metrics.OutOfStockSKUs = int64(v)
	}
	if v, ok := stats["inventoryValue"].(float64); ok {
		metrics.InventoryValue = v
	}
	if v, ok := stats["storageFees"].(float64); ok {
		metrics.StorageFees = v
	}

	return metrics, nil
}

func (s *DashboardService) getBillingMetrics(ctx context.Context, sellerID string, period domain.Period) (*domain.BillingMetrics, error) {
	startDate := period.Start.Format("2006-01-02")
	endDate := period.End.Format("2006-01-02")

	stats, err := s.billingClient.GetBillingStats(ctx, sellerID, startDate, endDate)
	if err != nil {
		return &domain.BillingMetrics{
			FeeBreakdown: domain.FeeBreakdown{},
			ChargesByDay: []domain.DailyCharge{},
		}, nil
	}

	metrics := &domain.BillingMetrics{
		FeeBreakdown: domain.FeeBreakdown{},
		ChargesByDay: []domain.DailyCharge{},
	}

	if v, ok := stats["currentBalance"].(float64); ok {
		metrics.CurrentBalance = v
	}
	if v, ok := stats["pendingCharges"].(float64); ok {
		metrics.PendingCharges = v
	}
	if v, ok := stats["lastInvoiceAmount"].(float64); ok {
		metrics.LastInvoiceAmount = v
	}
	if v, ok := stats["mtdCharges"].(float64); ok {
		metrics.MTDCharges = v
	}

	// Parse fee breakdown
	if breakdown, ok := stats["feeBreakdown"].(map[string]interface{}); ok {
		if v, ok := breakdown["storageFees"].(float64); ok {
			metrics.FeeBreakdown.StorageFees = v
		}
		if v, ok := breakdown["pickFees"].(float64); ok {
			metrics.FeeBreakdown.PickFees = v
		}
		if v, ok := breakdown["packFees"].(float64); ok {
			metrics.FeeBreakdown.PackFees = v
		}
		if v, ok := breakdown["shippingFees"].(float64); ok {
			metrics.FeeBreakdown.ShippingFees = v
		}
		if v, ok := breakdown["total"].(float64); ok {
			metrics.FeeBreakdown.Total = v
		}
	}

	return metrics, nil
}

func (s *DashboardService) getChannelMetrics(ctx context.Context, sellerID string) ([]domain.ChannelMetrics, error) {
	channelsResp, err := s.channelClient.GetChannels(ctx, sellerID)
	if err != nil {
		return []domain.ChannelMetrics{}, nil
	}

	var metrics []domain.ChannelMetrics

	channels, ok := channelsResp["channels"].([]interface{})
	if !ok {
		return metrics, nil
	}

	for _, ch := range channels {
		channel, ok := ch.(map[string]interface{})
		if !ok {
			continue
		}

		m := domain.ChannelMetrics{}
		if v, ok := channel["id"].(string); ok {
			m.ChannelID = v
		}
		if v, ok := channel["name"].(string); ok {
			m.ChannelName = v
		}
		if v, ok := channel["type"].(string); ok {
			m.ChannelType = v
		}
		if v, ok := channel["status"].(string); ok {
			m.Status = v
		}
		if stats, ok := channel["stats"].(map[string]interface{}); ok {
			if v, ok := stats["totalOrders"].(float64); ok {
				m.TotalOrders = int64(v)
			}
			if v, ok := stats["pendingOrders"].(float64); ok {
				m.PendingOrders = int64(v)
			}
		}

		metrics = append(metrics, m)
	}

	return metrics, nil
}

func (s *DashboardService) generateAlerts(ctx context.Context, sellerID string, summary *domain.DashboardSummary) []domain.Alert {
	var alerts []domain.Alert

	// Low stock alert
	if summary.InventoryMetrics.LowStockSKUs > 0 {
		alerts = append(alerts, domain.Alert{
			ID:        fmt.Sprintf("lowstock-%s-%d", sellerID, time.Now().Unix()),
			Type:      domain.AlertTypeLowStock,
			Severity:  "warning",
			Title:     "Low Stock Alert",
			Message:   fmt.Sprintf("%d SKUs are running low on stock", summary.InventoryMetrics.LowStockSKUs),
			ActionURL: "/inventory?status=low_stock",
			CreatedAt: time.Now(),
		})
	}

	// Out of stock alert
	if summary.InventoryMetrics.OutOfStockSKUs > 0 {
		alerts = append(alerts, domain.Alert{
			ID:        fmt.Sprintf("outofstock-%s-%d", sellerID, time.Now().Unix()),
			Type:      domain.AlertTypeOutOfStock,
			Severity:  "critical",
			Title:     "Out of Stock Alert",
			Message:   fmt.Sprintf("%d SKUs are out of stock", summary.InventoryMetrics.OutOfStockSKUs),
			ActionURL: "/inventory?status=out_of_stock",
			CreatedAt: time.Now(),
		})
	}

	// Pending orders alert
	if summary.OrderMetrics.PendingOrders > 10 {
		alerts = append(alerts, domain.Alert{
			ID:        fmt.Sprintf("pending-%s-%d", sellerID, time.Now().Unix()),
			Type:      domain.AlertTypeOrderIssue,
			Severity:  "warning",
			Title:     "High Pending Orders",
			Message:   fmt.Sprintf("You have %d pending orders awaiting fulfillment", summary.OrderMetrics.PendingOrders),
			ActionURL: "/orders?status=pending",
			CreatedAt: time.Now(),
		})
	}

	// Channel sync error alert
	for _, ch := range summary.ChannelMetrics {
		if ch.ErrorCount > 0 {
			alerts = append(alerts, domain.Alert{
				ID:        fmt.Sprintf("channel-%s-%d", ch.ChannelID, time.Now().Unix()),
				Type:      domain.AlertTypeChannelError,
				Severity:  "warning",
				Title:     fmt.Sprintf("%s Sync Issues", ch.ChannelName),
				Message:   fmt.Sprintf("%d errors occurred during channel sync", ch.ErrorCount),
				ActionURL: fmt.Sprintf("/integrations/%s", ch.ChannelID),
				CreatedAt: time.Now(),
			})
		}
	}

	// Performance alert
	if summary.OrderMetrics.FulfillmentRate > 0 && summary.OrderMetrics.FulfillmentRate < 90 {
		alerts = append(alerts, domain.Alert{
			ID:        fmt.Sprintf("perf-%s-%d", sellerID, time.Now().Unix()),
			Type:      domain.AlertTypePerformance,
			Severity:  "info",
			Title:     "Fulfillment Performance",
			Message:   fmt.Sprintf("Your fulfillment rate is %.1f%%. Consider reviewing your operations.", summary.OrderMetrics.FulfillmentRate),
			ActionURL: "/analytics",
			CreatedAt: time.Now(),
		})
	}

	return alerts
}

// GetOrders retrieves orders for a seller
func (s *DashboardService) GetOrders(ctx context.Context, filter domain.OrderFilter) ([]domain.SellerOrder, int64, error) {
	params := map[string]string{
		"sellerId": filter.SellerID,
		"page":     fmt.Sprintf("%d", filter.Page),
		"pageSize": fmt.Sprintf("%d", filter.PageSize),
	}

	if len(filter.Status) > 0 {
		params["status"] = filter.Status[0]
	}
	if filter.ChannelID != "" {
		params["channelId"] = filter.ChannelID
	}
	if filter.Search != "" {
		params["search"] = filter.Search
	}

	resp, err := s.orderClient.GetOrders(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	var orders []domain.SellerOrder
	var total int64

	if v, ok := resp["total"].(float64); ok {
		total = int64(v)
	}

	if ordersData, ok := resp["orders"].([]interface{}); ok {
		for _, o := range ordersData {
			order, ok := o.(map[string]interface{})
			if !ok {
				continue
			}

			sellerOrder := domain.SellerOrder{}
			if v, ok := order["orderId"].(string); ok {
				sellerOrder.OrderID = v
			}
			if v, ok := order["status"].(string); ok {
				sellerOrder.Status = v
			}
			if v, ok := order["totalAmount"].(float64); ok {
				sellerOrder.TotalAmount = v
			}
			if v, ok := order["trackingNumber"].(string); ok {
				sellerOrder.TrackingNumber = v
			}

			orders = append(orders, sellerOrder)
		}
	}

	return orders, total, nil
}

// GetInventory retrieves inventory for a seller
func (s *DashboardService) GetInventory(ctx context.Context, filter domain.InventoryFilter) ([]domain.SellerInventory, int64, error) {
	params := map[string]string{
		"sellerId": filter.SellerID,
		"page":     fmt.Sprintf("%d", filter.Page),
		"pageSize": fmt.Sprintf("%d", filter.PageSize),
	}

	if filter.WarehouseID != "" {
		params["warehouseId"] = filter.WarehouseID
	}
	if filter.Search != "" {
		params["search"] = filter.Search
	}

	resp, err := s.inventoryClient.GetInventory(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	var inventory []domain.SellerInventory
	var total int64

	if v, ok := resp["total"].(float64); ok {
		total = int64(v)
	}

	if items, ok := resp["items"].([]interface{}); ok {
		for _, i := range items {
			item, ok := i.(map[string]interface{})
			if !ok {
				continue
			}

			inv := domain.SellerInventory{
				Locations: []domain.InventoryLocation{},
			}
			if v, ok := item["sku"].(string); ok {
				inv.SKU = v
			}
			if v, ok := item["name"].(string); ok {
				inv.Name = v
			}
			if v, ok := item["available"].(float64); ok {
				inv.Available = int64(v)
			}
			if v, ok := item["reserved"].(float64); ok {
				inv.Reserved = int64(v)
			}
			if v, ok := item["status"].(string); ok {
				inv.Status = v
			}

			inventory = append(inventory, inv)
		}
	}

	return inventory, total, nil
}

// GetInvoices retrieves invoices for a seller
func (s *DashboardService) GetInvoices(ctx context.Context, filter domain.InvoiceFilter) ([]domain.SellerInvoice, int64, error) {
	params := map[string]string{
		"sellerId": filter.SellerID,
		"page":     fmt.Sprintf("%d", filter.Page),
		"pageSize": fmt.Sprintf("%d", filter.PageSize),
	}

	resp, err := s.billingClient.GetInvoices(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	var invoices []domain.SellerInvoice
	var total int64

	if v, ok := resp["total"].(float64); ok {
		total = int64(v)
	}

	if invData, ok := resp["invoices"].([]interface{}); ok {
		for _, i := range invData {
			inv, ok := i.(map[string]interface{})
			if !ok {
				continue
			}

			invoice := domain.SellerInvoice{
				LineItems: []domain.InvoiceLineItem{},
			}
			if v, ok := inv["invoiceId"].(string); ok {
				invoice.InvoiceID = v
			}
			if v, ok := inv["invoiceNumber"].(string); ok {
				invoice.InvoiceNumber = v
			}
			if v, ok := inv["status"].(string); ok {
				invoice.Status = v
			}
			if v, ok := inv["total"].(float64); ok {
				invoice.Total = v
			}

			invoices = append(invoices, invoice)
		}
	}

	return invoices, total, nil
}

// GetChannels retrieves channels for a seller
func (s *DashboardService) GetChannels(ctx context.Context, sellerID string) ([]domain.ChannelMetrics, error) {
	return s.getChannelMetrics(ctx, sellerID)
}

// ConnectChannel connects a new channel
func (s *DashboardService) ConnectChannel(ctx context.Context, sellerID string, req map[string]interface{}) (map[string]interface{}, error) {
	req["sellerId"] = sellerID
	return s.channelClient.ConnectChannel(ctx, req)
}

// DisconnectChannel disconnects a channel
func (s *DashboardService) DisconnectChannel(ctx context.Context, channelID string) error {
	return s.channelClient.DisconnectChannel(ctx, channelID)
}

// SyncChannel triggers a channel sync
func (s *DashboardService) SyncChannel(ctx context.Context, channelID string, syncType string) (map[string]interface{}, error) {
	if syncType == "inventory" {
		return s.channelClient.SyncInventory(ctx, channelID, nil)
	}
	return s.channelClient.SyncOrders(ctx, channelID)
}

// GetAPIKeys retrieves API keys for a seller
func (s *DashboardService) GetAPIKeys(ctx context.Context, sellerID string) ([]domain.APIKeyInfo, error) {
	keysResp, err := s.sellerClient.GetSellerAPIKeys(ctx, sellerID)
	if err != nil {
		return nil, err
	}

	var keys []domain.APIKeyInfo
	for _, k := range keysResp {
		key := domain.APIKeyInfo{}
		if v, ok := k["keyId"].(string); ok {
			key.KeyID = v
		}
		if v, ok := k["name"].(string); ok {
			key.Name = v
		}
		if v, ok := k["prefix"].(string); ok {
			key.Prefix = v
		}
		if v, ok := k["status"].(string); ok {
			key.Status = v
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// GenerateAPIKey creates a new API key
func (s *DashboardService) GenerateAPIKey(ctx context.Context, sellerID string, name string, permissions []string) (map[string]interface{}, error) {
	req := map[string]interface{}{
		"name":        name,
		"permissions": permissions,
	}
	return s.sellerClient.GenerateAPIKey(ctx, sellerID, req)
}

// RevokeAPIKey revokes an API key
func (s *DashboardService) RevokeAPIKey(ctx context.Context, sellerID, keyID string) error {
	return s.sellerClient.RevokeAPIKey(ctx, sellerID, keyID)
}
