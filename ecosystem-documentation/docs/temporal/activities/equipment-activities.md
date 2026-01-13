---
sidebar_position: 20
slug: /temporal/activities/equipment-activities
---

# Equipment Activities

Activities for managing equipment availability and reservations in the warehouse.

## Activity Struct

```go
type EquipmentActivities struct {
    clients *clients.ServiceClients
}
```

## Activities

### CheckEquipmentAvailability

Checks if required equipment is available.

**Signature:**
```go
func (a *EquipmentActivities) CheckEquipmentAvailability(ctx context.Context, input CheckEquipmentAvailabilityInput) (*CheckEquipmentAvailabilityResult, error)
```

**Input:**
```go
type CheckEquipmentAvailabilityInput struct {
    EquipmentTypes []string `json:"equipmentTypes"` // Types needed
    Zone           string   `json:"zone,omitempty"`
    Quantity       int      `json:"quantity"`       // Units needed
}
```

**Output:**
```go
type CheckEquipmentAvailabilityResult struct {
    AvailableEquipment    map[string]int `json:"availableEquipment"`    // Type -> count
    InsufficientEquipment []string       `json:"insufficientEquipment"` // Types lacking
    AllAvailable          bool           `json:"allAvailable"`
    Success               bool           `json:"success"`
}
```

**Behavior:**
- Returns `AllAvailable: true` if no equipment types required
- Checks each type against required quantity
- Reports which types have insufficient availability

---

### ReserveEquipment

Reserves specific equipment for an order.

**Signature:**
```go
func (a *EquipmentActivities) ReserveEquipment(ctx context.Context, input ReserveEquipmentInput) (*ReserveEquipmentResult, error)
```

**Input:**
```go
type ReserveEquipmentInput struct {
    EquipmentType string `json:"equipmentType"`
    OrderID       string `json:"orderId"`
    Quantity      int    `json:"quantity"`
    Zone          string `json:"zone,omitempty"`
    ReservationID string `json:"reservationId"`
}
```

**Output:**
```go
type ReserveEquipmentResult struct {
    ReservationID        string   `json:"reservationId"`
    EquipmentType        string   `json:"equipmentType"`
    ReservedEquipmentIDs []string `json:"reservedEquipmentIds"`
    Quantity             int      `json:"quantity"`
    Success              bool     `json:"success"`
}
```

---

### ReleaseEquipment

Releases previously reserved equipment.

**Signature:**
```go
func (a *EquipmentActivities) ReleaseEquipment(ctx context.Context, input ReleaseEquipmentInput) error
```

**Input:**
```go
type ReleaseEquipmentInput struct {
    ReservationID string `json:"reservationId"`
    EquipmentType string `json:"equipmentType"`
    OrderID       string `json:"orderId"`
    Reason        string `json:"reason,omitempty"`
}
```

**Used for:** Compensation when workflows fail or complete

---

### GetEquipmentByType

Retrieves equipment by type with optional filtering.

**Signature:**
```go
func (a *EquipmentActivities) GetEquipmentByType(ctx context.Context, input GetEquipmentByTypeInput) ([]clients.Equipment, error)
```

**Input:**
```go
type GetEquipmentByTypeInput struct {
    EquipmentType string `json:"equipmentType"`
    Zone          string `json:"zone,omitempty"`
    Status        string `json:"status,omitempty"` // available, in_use, maintenance
}
```

**Output:** Array of Equipment objects

## Equipment Types

| Type | Description | Use Case |
|------|-------------|----------|
| `pallet_jack` | Manual pallet jack | Standard picking |
| `forklift` | Powered forklift | Heavy pallets |
| `reach_truck` | Reach truck | High racking |
| `order_picker` | Order picker lift | Multi-level picking |
| `cold_chain_cart` | Temperature-controlled cart | Perishables |
| `hazmat_container` | Hazmat-rated container | Hazardous materials |
| `tote` | Standard tote | Item consolidation |
| `shipping_container` | Shipping box/container | Packing |

## Configuration

| Property | Value |
|----------|-------|
| Default Timeout | 2 minutes |
| Retry Policy | 3 maximum attempts |
| Heartbeat | Not required |

## Usage Example

```go
// Check availability before reserving
checkInput := activities.CheckEquipmentAvailabilityInput{
    EquipmentTypes: []string{"forklift", "pallet_jack"},
    Zone:           "ZONE-A",
    Quantity:       2,
}

var availability activities.CheckEquipmentAvailabilityResult
err := workflow.ExecuteActivity(ctx, equipmentActivities.CheckEquipmentAvailability, checkInput).Get(ctx, &availability)

if availability.AllAvailable {
    // Reserve equipment
    reserveInput := activities.ReserveEquipmentInput{
        EquipmentType: "forklift",
        OrderID:       "ORD-12345",
        Quantity:      1,
        ReservationID: fmt.Sprintf("RES-%s-forklift", orderID),
    }

    var reservation activities.ReserveEquipmentResult
    err = workflow.ExecuteActivity(ctx, equipmentActivities.ReserveEquipment, reserveInput).Get(ctx, &reservation)

    // Later, release on completion
    defer func() {
        releaseInput := activities.ReleaseEquipmentInput{
            ReservationID: reservation.ReservationID,
            EquipmentType: "forklift",
            OrderID:       "ORD-12345",
            Reason:        "order_completed",
        }
        workflow.ExecuteActivity(ctx, equipmentActivities.ReleaseEquipment, releaseInput).Get(ctx, nil)
    }()
}
```

## Related Workflows

- [Planning Workflow](../workflows/planning) - Validates equipment availability
- [WES Execution Workflow](../workflows/wes-execution) - Reserves equipment

## Related Documentation

- [Facility Service](/services/facility-service) - Equipment management
