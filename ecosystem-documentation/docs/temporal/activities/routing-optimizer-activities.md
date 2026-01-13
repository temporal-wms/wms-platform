---
sidebar_position: 22
slug: /temporal/activities/routing-optimizer-activities
---

# Routing Optimizer Activities

Activities for dynamic routing optimization, station selection, and order rerouting.

## Activity Struct

```go
type RoutingOptimizerActivities struct {
    clients *clients.ServiceClients
}
```

## Activities

### OptimizeStationSelection

Uses ML-like optimization to select the best station for an order.

**Signature:**
```go
func (a *RoutingOptimizerActivities) OptimizeStationSelection(ctx context.Context, input OptimizeStationSelectionInput) (*OptimizeStationSelectionResult, error)
```

**Input:**
```go
type OptimizeStationSelectionInput struct {
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
```

**Output:**
```go
type OptimizeStationSelectionResult struct {
    SelectedStationID string                 `json:"selectedStationId"`
    Score             float64                `json:"score"`
    Reasoning         map[string]float64     `json:"reasoning"`     // Factor -> weight
    AlternateStations []AlternateStationInfo `json:"alternateStations"`
    Confidence        float64                `json:"confidence"`
    Success           bool                   `json:"success"`
}

type AlternateStationInfo struct {
    StationID string  `json:"stationId"`
    Score     float64 `json:"score"`
    Rank      int     `json:"rank"`
}
```

**Scoring Factors:**

| Factor | Weight | Description |
|--------|--------|-------------|
| `capacity` | 0.25 | Station current capacity |
| `skills_match` | 0.20 | Worker skill availability |
| `equipment_match` | 0.15 | Equipment availability |
| `distance` | 0.15 | Distance from current location |
| `historical_perf` | 0.15 | Historical performance metrics |
| `deadline_urgency` | 0.10 | Time to promised delivery |

---

### GetRoutingMetrics

Retrieves current routing metrics for a facility or zone.

**Signature:**
```go
func (a *RoutingOptimizerActivities) GetRoutingMetrics(ctx context.Context, input GetRoutingMetricsInput) (*GetRoutingMetricsResult, error)
```

**Input:**
```go
type GetRoutingMetricsInput struct {
    FacilityID string `json:"facilityId,omitempty"`
    Zone       string `json:"zone,omitempty"`
    TimeWindow string `json:"timeWindow,omitempty"` // "1h", "24h"
}
```

**Output:**
```go
type GetRoutingMetricsResult struct {
    TotalRoutingDecisions   int                `json:"totalRoutingDecisions"`
    AverageDecisionTime     int64              `json:"averageDecisionTimeMs"`
    AverageConfidence       float64            `json:"averageConfidence"`
    StationUtilization      map[string]float64 `json:"stationUtilization"`
    CapacityConstrainedRate float64            `json:"capacityConstrainedRate"`
    RouteChanges            int                `json:"routeChanges"`
    RebalancingRecommended  bool               `json:"rebalancingRecommended"`
    LastUpdated             time.Time          `json:"lastUpdated"`
}
```

---

### RerouteOrder

Dynamically reroutes an order to a better station.

**Signature:**
```go
func (a *RoutingOptimizerActivities) RerouteOrder(ctx context.Context, input RerouteOrderInput) (*RerouteOrderResult, error)
```

**Input:**
```go
type RerouteOrderInput struct {
    OrderID      string   `json:"orderId"`
    CurrentPath  string   `json:"currentPath"`   // Current station/path
    Reason       string   `json:"reason"`
    Requirements []string `json:"requirements"`
    Priority     string   `json:"priority"`
    ForceReroute bool     `json:"forceReroute"`  // Force even if not optimal
}
```

**Output:**
```go
type RerouteOrderResult struct {
    NewStationID      string    `json:"newStationId"`
    PreviousStationID string    `json:"previousStationId"`
    Score             float64   `json:"score"`
    Confidence        float64   `json:"confidence"`
    RerouteTime       time.Time `json:"rerouteTime"`
    Success           bool      `json:"success"`
}
```

**Reroute Reasons:**

| Reason | Description |
|--------|-------------|
| `station_overload` | Current station at capacity |
| `station_offline` | Station unavailable |
| `equipment_failure` | Required equipment down |
| `worker_unavailable` | No certified workers |
| `deadline_risk` | At risk of missing SLA |
| `optimization` | Better option available |
| `manual_override` | Supervisor request |

## Configuration

| Property | Value |
|----------|-------|
| Default Timeout | 2 minutes |
| Retry Policy | 3 maximum attempts |
| Heartbeat | Not required |

## Usage Example

```go
// Optimize station selection for new order
optimizeInput := activities.OptimizeStationSelectionInput{
    OrderID:            "ORD-12345",
    Priority:           "same_day",
    Requirements:       []string{"multi_item"},
    SpecialHandling:    []string{"fragile"},
    ItemCount:          5,
    TotalWeight:        3.5,
    PromisedDeliveryAt: time.Now().Add(8 * time.Hour),
    RequiredSkills:     []string{},
    RequiredEquipment:  []string{"pallet_jack"},
    StationType:        "pick_pack",
}

var selection activities.OptimizeStationSelectionResult
err := workflow.ExecuteActivity(ctx, routingActivities.OptimizeStationSelection, optimizeInput).Get(ctx, &selection)

logger.Info("Station selected",
    "stationId", selection.SelectedStationID,
    "score", selection.Score,
    "confidence", selection.Confidence,
    "alternates", len(selection.AlternateStations),
)

// Later, if rerouting needed
if needsReroute {
    rerouteInput := activities.RerouteOrderInput{
        OrderID:      "ORD-12345",
        CurrentPath:  selection.SelectedStationID,
        Reason:       "station_overload",
        Requirements: []string{"multi_item"},
        Priority:     "same_day",
        ForceReroute: false,
    }

    var reroute activities.RerouteOrderResult
    err = workflow.ExecuteActivity(ctx, routingActivities.RerouteOrder, rerouteInput).Get(ctx, &reroute)
}
```

## Related Workflows

- [Planning Workflow](../workflows/planning) - Uses for initial routing
- [Continuous Optimization Workflow](../workflows/continuous-optimization) - Dynamic rerouting
- [WES Execution Workflow](../workflows/wes-execution) - Station assignment

## Related Documentation

- [Process Path Service](/services/process-path-service) - Routing engine
- [Routing Service](/services/routing-service) - Pick route optimization
