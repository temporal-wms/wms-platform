package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ShortItem represents an item with a stock shortage
type ShortItem struct {
	SKU          string `json:"sku"`
	LocationID   string `json:"locationId"`
	RequestedQty int    `json:"requestedQty"`
	AvailableQty int    `json:"availableQty"`
	ShortageQty  int    `json:"shortageQty"`
	Reason       string `json:"reason"` // not_found, damaged, quantity_mismatch
}

// StockShortageWorkflowInput represents input for the stock shortage handling workflow
type StockShortageWorkflowInput struct {
	OrderID        string       `json:"orderId"`
	CustomerID     string       `json:"customerId"`
	ShortItems     []ShortItem  `json:"shortItems"`
	CompletedItems []PickedItem `json:"completedItems"`
	ReportedBy     string       `json:"reportedBy"`
}

// StockShortageWorkflowResult represents the result of shortage handling
type StockShortageWorkflowResult struct {
	OrderID              string   `json:"orderId"`
	Strategy             string   `json:"strategy"` // partial_ship, full_backorder, cancelled
	ShippedItemCount     int      `json:"shippedItemCount"`
	BackorderedItemCount int      `json:"backorderedItemCount"`
	BackorderID          string   `json:"backorderId,omitempty"`
	CustomerNotified     bool     `json:"customerNotified"`
}

// PartialShipmentThreshold is the minimum fulfillment ratio (0.0-1.0) to auto-ship
// Below this threshold, the order is held for supervisor review
const PartialShipmentThreshold = 0.50

// StockShortageWorkflow handles compensation for confirmed stock shortages during picking
// This workflow:
// 1. Records inventory shortages and adjusts inventory
// 2. Decides fulfillment strategy based on threshold
// 3. Ships what's available (if threshold met) or creates backorder
// 4. Notifies customer of partial shipment or delay
func StockShortageWorkflow(ctx workflow.Context, input StockShortageWorkflowInput) (*StockShortageWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting stock shortage workflow",
		"orderId", input.OrderID,
		"shortItemCount", len(input.ShortItems),
		"completedItemCount", len(input.CompletedItems),
	)

	result := &StockShortageWorkflowResult{
		OrderID: input.OrderID,
	}

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Record inventory shortages for each short item
	logger.Info("Recording inventory shortages", "orderId", input.OrderID)
	for _, item := range input.ShortItems {
		err := workflow.ExecuteActivity(ctx, "RecordStockShortage", map[string]interface{}{
			"sku":         item.SKU,
			"locationId":  item.LocationID,
			"orderId":     input.OrderID,
			"expectedQty": item.RequestedQty,
			"actualQty":   item.AvailableQty,
			"reason":      item.Reason,
			"reportedBy":  input.ReportedBy,
		}).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to record shortage for item, continuing",
				"sku", item.SKU,
				"error", err,
			)
		}
	}

	// Step 2: Calculate fulfillment ratio
	totalRequested := 0
	totalAvailable := 0
	for _, item := range input.CompletedItems {
		totalRequested += item.Quantity
		totalAvailable += item.Quantity
	}
	for _, item := range input.ShortItems {
		totalRequested += item.RequestedQty
		totalAvailable += item.AvailableQty
	}

	fulfillmentRatio := 0.0
	if totalRequested > 0 {
		fulfillmentRatio = float64(totalAvailable) / float64(totalRequested)
	}

	hasCompletedItems := len(input.CompletedItems) > 0

	logger.Info("Calculated fulfillment ratio",
		"orderId", input.OrderID,
		"fulfillmentRatio", fulfillmentRatio,
		"threshold", PartialShipmentThreshold,
		"hasCompletedItems", hasCompletedItems,
	)

	// Step 3: Decide fulfillment strategy
	if hasCompletedItems && fulfillmentRatio >= PartialShipmentThreshold {
		// Partial shipment - ship what we have, backorder the rest
		result.Strategy = "partial_ship"
		result.ShippedItemCount = len(input.CompletedItems)
		result.BackorderedItemCount = len(input.ShortItems)

		logger.Info("Using partial shipment strategy", "orderId", input.OrderID)

		// Step 3a: Mark order as partially fulfilled
		err := workflow.ExecuteActivity(ctx, "MarkOrderPartiallyFulfilled", map[string]interface{}{
			"orderId":          input.OrderID,
			"fulfillmentRatio": fulfillmentRatio,
		}).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to mark order as partially fulfilled", "orderId", input.OrderID, "error", err)
		}

		// Step 3b: Create backorder for missing items
		backorderItems := make([]map[string]interface{}, len(input.ShortItems))
		for i, item := range input.ShortItems {
			backorderItems[i] = map[string]interface{}{
				"sku":         item.SKU,
				"quantity":    item.ShortageQty,
				"reason":      item.Reason,
			}
		}

		var backorderID string
		err = workflow.ExecuteActivity(ctx, "CreateBackorder", map[string]interface{}{
			"originalOrderId": input.OrderID,
			"customerId":      input.CustomerID,
			"items":           backorderItems,
		}).Get(ctx, &backorderID)
		if err != nil {
			logger.Error("Failed to create backorder", "orderId", input.OrderID, "error", err)
		} else {
			result.BackorderID = backorderID
			logger.Info("Backorder created", "orderId", input.OrderID, "backorderId", backorderID)
		}

		// Step 3c: Notify customer about partial shipment
		err = workflow.ExecuteActivity(ctx, "NotifyCustomerPartialShipment", map[string]interface{}{
			"orderId":            input.OrderID,
			"customerId":         input.CustomerID,
			"shippedItemCount":   len(input.CompletedItems),
			"backorderedItemCount": len(input.ShortItems),
			"backorderId":        backorderID,
		}).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to notify customer about partial shipment", "orderId", input.OrderID, "error", err)
		} else {
			result.CustomerNotified = true
		}

	} else if hasCompletedItems {
		// Below threshold - hold for supervisor review
		result.Strategy = "hold_for_review"
		result.ShippedItemCount = 0
		result.BackorderedItemCount = len(input.ShortItems) + len(input.CompletedItems)

		logger.Info("Order below threshold, holding for supervisor review",
			"orderId", input.OrderID,
			"fulfillmentRatio", fulfillmentRatio,
		)

		// Notify supervisor for manual decision
		err := workflow.ExecuteActivity(ctx, "NotifySupervisorShortageReview", map[string]interface{}{
			"orderId":          input.OrderID,
			"fulfillmentRatio": fulfillmentRatio,
			"shortItems":       input.ShortItems,
			"completedItems":   input.CompletedItems,
		}).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to notify supervisor", "orderId", input.OrderID, "error", err)
		}

	} else {
		// Complete shortage - no items available
		result.Strategy = "full_backorder"
		result.ShippedItemCount = 0
		result.BackorderedItemCount = len(input.ShortItems)

		logger.Info("Complete shortage, creating full backorder", "orderId", input.OrderID)

		// Create backorder for all items
		backorderItems := make([]map[string]interface{}, len(input.ShortItems))
		for i, item := range input.ShortItems {
			backorderItems[i] = map[string]interface{}{
				"sku":      item.SKU,
				"quantity": item.RequestedQty,
				"reason":   item.Reason,
			}
		}

		var backorderID string
		err := workflow.ExecuteActivity(ctx, "CreateBackorder", map[string]interface{}{
			"originalOrderId": input.OrderID,
			"customerId":      input.CustomerID,
			"items":           backorderItems,
		}).Get(ctx, &backorderID)
		if err != nil {
			logger.Error("Failed to create backorder", "orderId", input.OrderID, "error", err)
		} else {
			result.BackorderID = backorderID
		}

		// Notify customer about full shortage
		err = workflow.ExecuteActivity(ctx, "NotifyCustomerShortage", map[string]interface{}{
			"orderId":     input.OrderID,
			"customerId":  input.CustomerID,
			"backorderId": backorderID,
			"items":       input.ShortItems,
		}).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to notify customer about shortage", "orderId", input.OrderID, "error", err)
		} else {
			result.CustomerNotified = true
		}
	}

	logger.Info("Stock shortage workflow completed",
		"orderId", input.OrderID,
		"strategy", result.Strategy,
		"shippedItems", result.ShippedItemCount,
		"backorderedItems", result.BackorderedItemCount,
	)

	return result, nil
}

// BackorderFulfillmentWorkflow handles auto-fulfillment of backorders when stock arrives
// This workflow is triggered by InventoryReceivedEvent for backordered SKUs
func BackorderFulfillmentWorkflow(ctx workflow.Context, input map[string]interface{}) error {
	logger := workflow.GetLogger(ctx)

	backorderID, _ := input["backorderId"].(string)
	originalOrderID, _ := input["originalOrderId"].(string)
	customerID, _ := input["customerId"].(string)

	logger.Info("Starting backorder fulfillment workflow",
		"backorderId", backorderID,
		"originalOrderId", originalOrderID,
	)

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Reserve stock for backorder items
	logger.Info("Reserving stock for backorder", "backorderId", backorderID)
	err := workflow.ExecuteActivity(ctx, "ReserveStockForBackorder", map[string]interface{}{
		"backorderId": backorderID,
	}).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to reserve stock for backorder: %w", err)
	}

	// Step 2: Create new pick task linked to original order
	logger.Info("Creating pick task for backorder", "backorderId", backorderID)
	var taskID string
	err = workflow.ExecuteActivity(ctx, "CreateBackorderPickTask", map[string]interface{}{
		"backorderId":     backorderID,
		"originalOrderId": originalOrderID,
	}).Get(ctx, &taskID)
	if err != nil {
		return fmt.Errorf("failed to create pick task for backorder: %w", err)
	}

	// Step 3: Mark backorder as in progress
	err = workflow.ExecuteActivity(ctx, "MarkBackorderInProgress", map[string]interface{}{
		"backorderId": backorderID,
		"pickTaskId":  taskID,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to mark backorder in progress", "backorderId", backorderID, "error", err)
	}

	// Step 4: Notify customer that backorder is being fulfilled
	err = workflow.ExecuteActivity(ctx, "NotifyCustomerBackorderShipping", map[string]interface{}{
		"backorderId":     backorderID,
		"originalOrderId": originalOrderID,
		"customerId":      customerID,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to notify customer", "backorderId", backorderID, "error", err)
	}

	logger.Info("Backorder fulfillment workflow initiated",
		"backorderId", backorderID,
		"pickTaskId", taskID,
	)

	return nil
}
