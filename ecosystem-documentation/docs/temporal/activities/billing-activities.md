---
sidebar_position: 16
slug: /temporal/activities/billing-activities
---

# Billing Activities

Activities for recording fulfillment fees and billable events for seller accounts.

## Activity Struct

```go
type BillingActivities struct {
    clients *ServiceClients
    logger  *slog.Logger
}
```

## Activities

### RecordFulfillmentFees

Records all billable activities for an order fulfillment including pick, pack, shipping, and special handling fees.

**Signature:**
```go
func (a *BillingActivities) RecordFulfillmentFees(ctx context.Context, input map[string]interface{}) (*FulfillmentFeeResult, error)
```

**Purpose:** Calculates and records all fulfillment-related fees for seller billing.

**Input Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `orderId` | string | Order identifier |
| `sellerId` | string | Seller account ID |
| `tenantId` | string | Tenant identifier |
| `facilityId` | string | Fulfillment facility ID |
| `warehouseId` | string | Warehouse identifier |
| `items` | []object | Order items with quantity and weight |
| `carrier` | string | Shipping carrier name |
| `weight` | float64 | Total package weight |
| `giftWrap` | bool | Gift wrapping requested |
| `hasHazmat` | bool | Contains hazardous materials |
| `hasColdChain` | bool | Requires cold chain handling |

**Output:**

```go
type FulfillmentFeeResult struct {
    Success      bool                `json:"success"`
    ActivityIDs  []string            `json:"activityIds"`   // IDs of recorded activities
    TotalFees    float64             `json:"totalFees"`     // Sum of all fees
    FeeBreakdown map[string]float64  `json:"feeBreakdown"`  // Fee by type
    Error        string              `json:"error,omitempty"`
}
```

**Fee Types Recorded:**

| Fee Type | Trigger | Calculation |
|----------|---------|-------------|
| `pick` | Always | Per unit picked |
| `pack` | Always | Per order |
| `shipping` | Carrier provided | Based on carrier and weight |
| `gift_wrap` | `giftWrap: true` | Per order |
| `hazmat` | `hasHazmat: true` | Per unit |
| `cold_chain` | `hasColdChain: true` | Per unit |

**Error Handling:**
- Individual fee recording failures are logged as warnings but don't fail the activity
- Returns `Success: true` even if some fees couldn't be recorded
- No seller ID means no fees recorded (silently succeeds)

**Example:**
```go
input := map[string]interface{}{
    "orderId":    "ORD-12345",
    "sellerId":   "SELLER-001",
    "tenantId":   "TENANT-001",
    "facilityId": "FAC-001",
    "items": []map[string]interface{}{
        {"sku": "SKU-001", "quantity": 2, "weight": 1.5},
        {"sku": "SKU-002", "quantity": 1, "weight": 0.5},
    },
    "carrier":     "UPS",
    "weight":      2.0,
    "giftWrap":    true,
    "hasHazmat":   false,
    "hasColdChain": false,
}

var result activities.FulfillmentFeeResult
err := workflow.ExecuteActivity(ctx, billingActivities.RecordFulfillmentFees, input).Get(ctx, &result)

// result.FeeBreakdown: {"pick": 3.00, "pack": 1.50, "shipping": 8.99, "gift_wrap": 2.99}
// result.TotalFees: 16.48
```

## Configuration

| Property | Value |
|----------|-------|
| Default Timeout | 2 minutes |
| Retry Policy | Standard (3 attempts) |
| Heartbeat | Not required |

## Integration Points

### Billing Service API

The activity calls the billing service to record each fee:

```http
POST /api/v1/activities
```

Request body:
```json
{
  "sellerId": "SELLER-001",
  "tenantId": "TENANT-001",
  "facilityId": "FAC-001",
  "warehouseId": "WH-001",
  "type": "pick",
  "orderId": "ORD-12345",
  "quantity": 3,
  "description": "Pick fee for order ORD-12345 (3 units)"
}
```

## Related Workflows

- [Order Fulfillment Workflow](../workflows/order-fulfillment) - Records fees after shipping
- [Shipping Workflow](../workflows/shipping) - May trigger fee recording

## Related Documentation

- [Billing Service](/services/billing-service) - Fee calculation and invoice generation
