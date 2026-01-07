package activities

import (
	"context"
	"fmt"
	"log/slog"

	"go.temporal.io/sdk/activity"
)

// BillingActivities contains activities related to billing operations
type BillingActivities struct {
	clients *ServiceClients
	logger  *slog.Logger
}

// NewBillingActivities creates a new BillingActivities instance
func NewBillingActivities(clients *ServiceClients, logger *slog.Logger) *BillingActivities {
	return &BillingActivities{
		clients: clients,
		logger:  logger,
	}
}

// FulfillmentFeeInput represents input for recording fulfillment fees
type FulfillmentFeeInput struct {
	OrderID        string                 `json:"orderId"`
	SellerID       string                 `json:"sellerId"`
	TenantID       string                 `json:"tenantId"`
	FacilityID     string                 `json:"facilityId"`
	WarehouseID    string                 `json:"warehouseId"`
	Items          []FulfillmentFeeItem   `json:"items"`
	TotalValue     float64                `json:"totalValue"`
	TrackingNumber string                 `json:"trackingNumber"`
	Carrier        string                 `json:"carrier"`
	Weight         float64                `json:"weight"`
	GiftWrap       bool                   `json:"giftWrap"`
	HasHazmat      bool                   `json:"hasHazmat"`
	HasColdChain   bool                   `json:"hasColdChain"`
}

// FulfillmentFeeItem represents an item for fee calculation
type FulfillmentFeeItem struct {
	SKU      string  `json:"sku"`
	Quantity int     `json:"quantity"`
	Weight   float64 `json:"weight"`
}

// FulfillmentFeeResult represents the result of recording fees
type FulfillmentFeeResult struct {
	Success      bool                `json:"success"`
	ActivityIDs  []string            `json:"activityIds"`
	TotalFees    float64             `json:"totalFees"`
	FeeBreakdown map[string]float64  `json:"feeBreakdown"`
	Error        string              `json:"error,omitempty"`
}

// RecordFulfillmentFees records all billable activities for an order fulfillment
func (a *BillingActivities) RecordFulfillmentFees(ctx context.Context, input map[string]interface{}) (*FulfillmentFeeResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Recording fulfillment fees", "orderId", input["orderId"], "sellerId", input["sellerId"])

	result := &FulfillmentFeeResult{
		ActivityIDs:  []string{},
		FeeBreakdown: make(map[string]float64),
	}

	// Extract input values
	orderID, _ := input["orderId"].(string)
	sellerID, _ := input["sellerId"].(string)
	tenantID, _ := input["tenantId"].(string)
	facilityID, _ := input["facilityId"].(string)
	warehouseID, _ := input["warehouseId"].(string)

	if sellerID == "" {
		// No seller, no fees to record
		result.Success = true
		return result, nil
	}

	// Calculate total units for pick fees
	totalUnits := 0
	if items, ok := input["items"].([]interface{}); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if qty, ok := itemMap["Quantity"].(float64); ok {
					totalUnits += int(qty)
				} else if qty, ok := itemMap["quantity"].(float64); ok {
					totalUnits += int(qty)
				}
			}
		}
	}

	// Record pick fee
	pickFeeInput := map[string]interface{}{
		"sellerId":    sellerID,
		"tenantId":    tenantID,
		"facilityId":  facilityID,
		"warehouseId": warehouseID,
		"type":        "pick",
		"orderId":     orderID,
		"quantity":    totalUnits,
		"description": fmt.Sprintf("Pick fee for order %s (%d units)", orderID, totalUnits),
	}

	pickResult, err := a.recordActivity(ctx, pickFeeInput)
	if err != nil {
		logger.Warn("Failed to record pick fee", "error", err)
	} else if pickResult != nil {
		result.ActivityIDs = append(result.ActivityIDs, pickResult["activityId"].(string))
		if amount, ok := pickResult["amount"].(float64); ok {
			result.FeeBreakdown["pick"] = amount
			result.TotalFees += amount
		}
	}

	// Record pack fee (per order)
	packFeeInput := map[string]interface{}{
		"sellerId":    sellerID,
		"tenantId":    tenantID,
		"facilityId":  facilityID,
		"warehouseId": warehouseID,
		"type":        "pack",
		"orderId":     orderID,
		"quantity":    1,
		"description": fmt.Sprintf("Pack fee for order %s", orderID),
	}

	packResult, err := a.recordActivity(ctx, packFeeInput)
	if err != nil {
		logger.Warn("Failed to record pack fee", "error", err)
	} else if packResult != nil {
		result.ActivityIDs = append(result.ActivityIDs, packResult["activityId"].(string))
		if amount, ok := packResult["amount"].(float64); ok {
			result.FeeBreakdown["pack"] = amount
			result.TotalFees += amount
		}
	}

	// Record shipping fee if carrier provided
	if carrier, ok := input["carrier"].(string); ok && carrier != "" {
		weight, _ := input["weight"].(float64)
		shippingFeeInput := map[string]interface{}{
			"sellerId":    sellerID,
			"tenantId":    tenantID,
			"facilityId":  facilityID,
			"warehouseId": warehouseID,
			"type":        "shipping",
			"orderId":     orderID,
			"quantity":    1,
			"carrier":     carrier,
			"weight":      weight,
			"description": fmt.Sprintf("Shipping fee for order %s via %s", orderID, carrier),
		}

		shippingResult, err := a.recordActivity(ctx, shippingFeeInput)
		if err != nil {
			logger.Warn("Failed to record shipping fee", "error", err)
		} else if shippingResult != nil {
			result.ActivityIDs = append(result.ActivityIDs, shippingResult["activityId"].(string))
			if amount, ok := shippingResult["amount"].(float64); ok {
				result.FeeBreakdown["shipping"] = amount
				result.TotalFees += amount
			}
		}
	}

	// Record gift wrap fee if applicable
	if giftWrap, ok := input["giftWrap"].(bool); ok && giftWrap {
		giftWrapInput := map[string]interface{}{
			"sellerId":    sellerID,
			"tenantId":    tenantID,
			"facilityId":  facilityID,
			"warehouseId": warehouseID,
			"type":        "gift_wrap",
			"orderId":     orderID,
			"quantity":    1,
			"description": fmt.Sprintf("Gift wrap fee for order %s", orderID),
		}

		giftResult, err := a.recordActivity(ctx, giftWrapInput)
		if err != nil {
			logger.Warn("Failed to record gift wrap fee", "error", err)
		} else if giftResult != nil {
			result.ActivityIDs = append(result.ActivityIDs, giftResult["activityId"].(string))
			if amount, ok := giftResult["amount"].(float64); ok {
				result.FeeBreakdown["gift_wrap"] = amount
				result.TotalFees += amount
			}
		}
	}

	// Record hazmat fee if applicable
	if hasHazmat, ok := input["hasHazmat"].(bool); ok && hasHazmat {
		hazmatInput := map[string]interface{}{
			"sellerId":    sellerID,
			"tenantId":    tenantID,
			"facilityId":  facilityID,
			"warehouseId": warehouseID,
			"type":        "hazmat",
			"orderId":     orderID,
			"quantity":    totalUnits,
			"description": fmt.Sprintf("Hazmat handling fee for order %s", orderID),
		}

		hazmatResult, err := a.recordActivity(ctx, hazmatInput)
		if err != nil {
			logger.Warn("Failed to record hazmat fee", "error", err)
		} else if hazmatResult != nil {
			result.ActivityIDs = append(result.ActivityIDs, hazmatResult["activityId"].(string))
			if amount, ok := hazmatResult["amount"].(float64); ok {
				result.FeeBreakdown["hazmat"] = amount
				result.TotalFees += amount
			}
		}
	}

	// Record cold chain fee if applicable
	if hasColdChain, ok := input["hasColdChain"].(bool); ok && hasColdChain {
		coldChainInput := map[string]interface{}{
			"sellerId":    sellerID,
			"tenantId":    tenantID,
			"facilityId":  facilityID,
			"warehouseId": warehouseID,
			"type":        "cold_chain",
			"orderId":     orderID,
			"quantity":    totalUnits,
			"description": fmt.Sprintf("Cold chain handling fee for order %s", orderID),
		}

		coldResult, err := a.recordActivity(ctx, coldChainInput)
		if err != nil {
			logger.Warn("Failed to record cold chain fee", "error", err)
		} else if coldResult != nil {
			result.ActivityIDs = append(result.ActivityIDs, coldResult["activityId"].(string))
			if amount, ok := coldResult["amount"].(float64); ok {
				result.FeeBreakdown["cold_chain"] = amount
				result.TotalFees += amount
			}
		}
	}

	result.Success = true
	logger.Info("Fulfillment fees recorded",
		"orderId", orderID,
		"sellerId", sellerID,
		"totalFees", result.TotalFees,
		"activityCount", len(result.ActivityIDs),
	)

	return result, nil
}

// recordActivity calls the billing service to record a billable activity
func (a *BillingActivities) recordActivity(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// Call billing service API
	resp, err := a.clients.PostJSON(ctx, "billing", "/api/v1/activities", input)
	if err != nil {
		return nil, fmt.Errorf("failed to record activity: %w", err)
	}

	result, ok := resp.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response type")
	}

	return result, nil
}
