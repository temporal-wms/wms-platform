---
sidebar_position: 17
slug: /temporal/activities/continuous-optimization-activities
---

# Continuous Optimization Activities

Activities for monitoring system health, rebalancing workloads, and optimizing warehouse operations.

## Activity Struct

```go
type ContinuousOptimizationActivities struct {
    clients *clients.ServiceClients
}
```

## Activities

### MonitorSystemHealth

Monitors overall system health and routing efficiency across stations.

**Signature:**
```go
func (a *ContinuousOptimizationActivities) MonitorSystemHealth(ctx context.Context, input MonitorSystemHealthInput) (*MonitorSystemHealthResult, error)
```

**Input:**
```go
type MonitorSystemHealthInput struct {
    FacilityID string `json:"facilityId,omitempty"`
    Zone       string `json:"zone,omitempty"`
    TimeWindow string `json:"timeWindow"` // e.g., "1h", "24h"
}
```

**Output:**
```go
type MonitorSystemHealthResult struct {
    OverallHealth           string             `json:"overallHealth"` // "healthy", "degraded", "critical"
    StationUtilization      map[string]float64 `json:"stationUtilization"`
    OverloadedStations      []string           `json:"overloadedStations"`   // >90% utilization
    UnderutilizedStations   []string           `json:"underutilizedStations"` // <30% utilization
    AverageConfidence       float64            `json:"averageConfidence"`
    CapacityConstrainedRate float64            `json:"capacityConstrainedRate"`
    RebalancingRecommended  bool               `json:"rebalancingRecommended"`
    ReroutingOpportunities  int                `json:"reroutingOpportunities"`
    Timestamp               time.Time          `json:"timestamp"`
}
```

**Health Determination Logic:**

| Health Status | Conditions |
|---------------|------------|
| `healthy` | No overloaded stations, <20% capacity constrained |
| `degraded` | 1-3 overloaded stations OR 20-40% capacity constrained |
| `critical` | >3 overloaded stations OR >40% capacity constrained |

---

### RebalanceWaves

Redistributes orders across stations to balance load.

**Signature:**
```go
func (a *ContinuousOptimizationActivities) RebalanceWaves(ctx context.Context, input RebalanceWavesInput) (*RebalanceWavesResult, error)
```

**Input:**
```go
type RebalanceWavesInput struct {
    FacilityID            string   `json:"facilityId"`
    OverloadedStations    []string `json:"overloadedStations"`
    UnderutilizedStations []string `json:"underutilizedStations"`
    MaxOrdersToRebalance  int      `json:"maxOrdersToRebalance"` // Default: 50
}
```

**Output:**
```go
type RebalanceWavesResult struct {
    OrdersRebalanced  int                       `json:"ordersRebalanced"`
    StationChanges    map[string]StationChange  `json:"stationChanges"`
    NewUtilization    map[string]float64        `json:"newUtilization"`
    RebalancedAt      time.Time                 `json:"rebalancedAt"`
    Success           bool                      `json:"success"`
}

type StationChange struct {
    StationID          string  `json:"stationId"`
    OrdersAdded        int     `json:"ordersAdded"`
    OrdersRemoved      int     `json:"ordersRemoved"`
    UtilizationBefore  float64 `json:"utilizationBefore"`
    UtilizationAfter   float64 `json:"utilizationAfter"`
}
```

**Behavior:**
- Returns success with 0 orders if no imbalance detected
- Distributes orders in round-robin fashion across underutilized stations
- Respects `MaxOrdersToRebalance` limit

---

### TriggerDynamicRerouting

Triggers rerouting of in-flight orders based on changing conditions.

**Signature:**
```go
func (a *ContinuousOptimizationActivities) TriggerDynamicRerouting(ctx context.Context, input TriggerDynamicReroutingInput) (*TriggerDynamicReroutingResult, error)
```

**Input:**
```go
type TriggerDynamicReroutingInput struct {
    FacilityID   string   `json:"facilityId"`
    OrderIDs     []string `json:"orderIds,omitempty"`     // Specific orders
    StationID    string   `json:"stationId,omitempty"`    // Or all from station
    Reason       string   `json:"reason"`
    Priority     string   `json:"priority"`               // "high", "medium", "low"
    ForceReroute bool     `json:"forceReroute"`
}
```

**Output:**
```go
type TriggerDynamicReroutingResult struct {
    OrdersRerouted     int                  `json:"ordersRerouted"`
    ReroutingDecisions []ReroutingDecision  `json:"reroutingDecisions"`
    AverageConfidence  float64              `json:"averageConfidence"`
    ReroutedAt         time.Time            `json:"reroutedAt"`
    Success            bool                 `json:"success"`
}

type ReroutingDecision struct {
    OrderID          string  `json:"orderId"`
    OldStationID     string  `json:"oldStationId"`
    NewStationID     string  `json:"newStationId"`
    Score            float64 `json:"score"`
    Confidence       float64 `json:"confidence"`
    ImprovementScore float64 `json:"improvementScore"`
}
```

---

### PredictCapacityNeeds

Forecasts capacity requirements based on historical patterns.

**Signature:**
```go
func (a *ContinuousOptimizationActivities) PredictCapacityNeeds(ctx context.Context, input PredictCapacityNeedsInput) (*PredictCapacityNeedsResult, error)
```

**Input:**
```go
type PredictCapacityNeedsInput struct {
    FacilityID       string `json:"facilityId"`
    Zone             string `json:"zone,omitempty"`
    PredictionWindow string `json:"predictionWindow"` // "1h", "4h", "24h"
    HistoricalWindow string `json:"historicalWindow"` // "7d", "30d"
}
```

**Output:**
```go
type PredictCapacityNeedsResult struct {
    PredictedOrderVolume     int                `json:"predictedOrderVolume"`
    PredictedStationLoad     map[string]float64 `json:"predictedStationLoad"`
    RecommendedStaffing      map[string]int     `json:"recommendedStaffing"`
    PredictedBottlenecks     []string           `json:"predictedBottlenecks"`
    ConfidenceScore          float64            `json:"confidenceScore"`
    RecommendedActions       []string           `json:"recommendedActions"`
    PredictionTime           time.Time          `json:"predictionTime"`
    ForecastHorizon          time.Time          `json:"forecastHorizon"`
}
```

**Prediction Logic:**
- Assumes 20% growth factor from historical data
- Identifies bottlenecks where predicted utilization >85%
- Recommends 1 worker per 20% predicted utilization

## Configuration

| Property | Value |
|----------|-------|
| Default Timeout | 2 minutes |
| Retry Policy | 3 maximum attempts |
| Heartbeat | Not required |

## Related Workflows

- [Continuous Optimization Workflow](../workflows/continuous-optimization) - Primary consumer

## Related Documentation

- [Process Path Service](/services/process-path-service) - Routing metrics source
- [Architecture - Data Flow](/architecture/system-diagrams/data-flow)
