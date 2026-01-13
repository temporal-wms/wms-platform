# Continuous Optimization System

## Overview

The Continuous Optimization System provides real-time monitoring, dynamic rebalancing, and predictive capacity planning for your WMS platform. It implements Amazon-style adaptive optimization that continuously adjusts routing decisions based on changing conditions.

## Features

### 1. System Health Monitoring
- **Real-time metrics collection**: Monitors station utilization, routing confidence, and capacity constraints
- **Health classification**: Categorizes system as `healthy`, `degraded`, or `critical`
- **Bottleneck detection**: Identifies overloaded and underutilized stations
- **Rerouting opportunity identification**: Detects orders that could benefit from alternate routing

### 2. Automatic Wave Rebalancing
- **Load balancing**: Redistributes orders from overloaded stations to underutilized ones
- **Configurable thresholds**: Triggers rebalancing when utilization exceeds configured limits
- **Order limits**: Controls maximum orders moved per rebalancing cycle
- **Impact tracking**: Records before/after utilization for all affected stations

### 3. Dynamic Order Rerouting
- **In-flight adjustments**: Reroutes orders already assigned to waves when better paths emerge
- **Confidence-based decisions**: Only reroutes when confidence exceeds threshold
- **Improvement scoring**: Calculates benefit of each rerouting decision
- **Reason tracking**: Records why each rerouting occurred

### 4. Capacity Prediction
- **Forecasting**: Predicts order volume and station load for next 1-24 hours
- **Historical analysis**: Uses past patterns to identify trends
- **Bottleneck prediction**: Identifies stations likely to become overloaded
- **Staffing recommendations**: Suggests worker allocation based on predicted load
- **Proactive actions**: Recommends preemptive measures before capacity issues arise

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│         Continuous Optimization Workflow (Temporal)         │
│  ┌────────────────────────────────────────────────────┐    │
│  │  Every 5 minutes (configurable):                    │    │
│  │  1. Monitor System Health                           │    │
│  │  2. Trigger Rebalancing (if needed)                 │    │
│  │  3. Trigger Dynamic Rerouting (if opportunities)    │    │
│  │  4. Generate Capacity Predictions                   │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                            │
                            ├──────────────────────────────────┐
                            ▼                                  ▼
    ┌──────────────────────────────┐      ┌──────────────────────────────┐
    │   ContinuousOptimization     │      │    Routing Optimizer         │
    │        Activities            │      │  (Phase 3.1 - ATROPS-like)   │
    │                              │      │                              │
    │  - MonitorSystemHealth       │◄─────┤  - OptimizeStationSelection  │
    │  - RebalanceWaves            │      │  - GetRoutingMetrics         │
    │  - TriggerDynamicRerouting   │      │  - RerouteOrder              │
    │  - PredictCapacityNeeds      │      │                              │
    └──────────────────────────────┘      └──────────────────────────────┘
                   │                                     │
                   └──────────────┬──────────────────────┘
                                  ▼
                  ┌────────────────────────────┐
                  │   Process Path Service     │
                  │   Routing Endpoints:       │
                  │   - /optimize-routing      │
                  │   - /metrics               │
                  │   - /reroute               │
                  └────────────────────────────┘
```

### Activity Details

#### MonitorSystemHealth
**Purpose**: Collect real-time metrics and assess system health

**Inputs**:
- `facilityId`: Facility to monitor
- `zone`: Optional zone filter
- `timeWindow`: Metrics collection window (e.g., "1h", "24h")

**Outputs**:
- `overallHealth`: "healthy", "degraded", or "critical"
- `stationUtilization`: Map of station ID → utilization %
- `overloadedStations`: Stations above 90% utilization
- `underutilizedStations`: Stations below 30% utilization
- `rebalancingRecommended`: Boolean flag
- `reroutingOpportunities`: Count of orders that could be rerouted

**Health Classification**:
- **Healthy**: <3 overloaded stations, <20% capacity-constrained rate
- **Degraded**: ≥1 overloaded station OR >20% capacity-constrained rate
- **Critical**: ≥3 overloaded stations OR >40% capacity-constrained rate

#### RebalanceWaves
**Purpose**: Redistribute orders from overloaded to underutilized stations

**Inputs**:
- `facilityId`: Facility to rebalance
- `overloadedStations`: List of overloaded station IDs
- `underutilizedStations`: List of underutilized station IDs
- `maxOrdersToRebalance`: Maximum orders to move (default: 50)

**Outputs**:
- `ordersRebalanced`: Total orders moved
- `stationChanges`: Map of station changes with before/after utilization
- `newUtilization`: Updated utilization percentages

**Algorithm**:
1. Distributes `maxOrdersToRebalance` evenly across overloaded stations
2. Uses round-robin to select underutilized target stations
3. Simulates order movement and calculates new utilization
4. Records all changes for auditing

#### TriggerDynamicRerouting
**Purpose**: Reroute in-flight orders based on changing conditions

**Inputs**:
- `facilityId`: Facility context
- `orderIds`: Specific orders to reroute (optional)
- `stationId`: Reroute all orders from this station (optional)
- `reason`: Reason for rerouting
- `priority`: "high", "medium", or "low"
- `forceReroute`: Force rerouting even if not optimal

**Outputs**:
- `ordersRerouted`: Count of successfully rerouted orders
- `reroutingDecisions`: Detailed decision list with scores
- `averageConfidence`: Average confidence across all decisions

**Use Cases**:
- Station goes offline → Reroute all its orders
- Station becomes overloaded → Reroute some orders to relieve pressure
- Better path emerges → Proactively reroute for optimization

#### PredictCapacityNeeds
**Purpose**: Forecast future capacity requirements

**Inputs**:
- `facilityId`: Facility to predict
- `predictionWindow`: Forecast horizon (e.g., "1h", "4h", "24h")
- `historicalWindow`: Historical data window (e.g., "7d", "30d")

**Outputs**:
- `predictedOrderVolume`: Forecasted order count
- `predictedStationLoad`: Map of station ID → predicted utilization %
- `predictedBottlenecks`: Stations likely to become overloaded
- `recommendedStaffing`: Suggested worker count per station
- `recommendedActions`: List of proactive measures
- `confidenceScore`: Prediction confidence (0.0-1.0)

**Prediction Algorithm**:
1. Analyzes historical routing metrics from `historicalWindow`
2. Applies growth factor (default: 20% increase)
3. Projects station utilization based on current trends
4. Identifies stations predicted to exceed 85% utilization
5. Calculates staffing needs: 1 worker per 20% utilization
6. Generates actionable recommendations

## Configuration

### Starting the Optimization Workflow

```go
import (
    "go.temporal.io/sdk/client"
    "github.com/wms-platform/orchestrator/internal/workflows"
)

func startOptimization() error {
    c, err := client.Dial(client.Options{})
    if err != nil {
        return err
    }
    defer c.Close()

    input := workflows.ContinuousOptimizationWorkflowInput{
        FacilityID:                 "facility-123",
        MonitoringInterval:         "5m",     // Check every 5 minutes
        CapacityThreshold:          0.85,     // Rebalance at 85% utilization
        UnderutilizationThreshold:  0.30,     // Underutilized below 30%
        MaxOrdersPerRebalance:      50,       // Move max 50 orders per cycle
        EnableAutoRebalancing:      true,     // Automatic rebalancing
        EnableAutoRerouting:        true,     // Automatic rerouting
        EnableCapacityPrediction:   true,     // Generate predictions
    }

    workflowOptions := client.StartWorkflowOptions{
        ID:        "continuous-optimization-facility-123",
        TaskQueue: "orchestrator",
    }

    we, err := c.ExecuteWorkflow(context.Background(), workflowOptions,
        workflows.ContinuousOptimizationWorkflow, input)
    if err != nil {
        return err
    }

    log.Printf("Started continuous optimization workflow: %s, RunID: %s",
        we.GetID(), we.GetRunID())
    return nil
}
```

### Stopping the Optimization Workflow

```go
func stopOptimization(workflowID string) error {
    c, err := client.Dial(client.Options{})
    if err != nil {
        return err
    }
    defer c.Close()

    // Send stop signal
    err = c.SignalWorkflow(context.Background(), workflowID, "", "stop-optimization", "stop")
    if err != nil {
        return err
    }

    log.Printf("Sent stop signal to workflow: %s", workflowID)
    return nil
}
```

### Configuration Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `facilityId` | string | required | Facility to optimize |
| `zone` | string | optional | Specific zone within facility |
| `monitoringInterval` | string | "5m" | Interval between optimization cycles |
| `capacityThreshold` | float64 | 0.85 | Utilization % that triggers rebalancing |
| `underutilizationThreshold` | float64 | 0.30 | Utilization % considered underutilized |
| `maxOrdersPerRebalance` | int | 50 | Maximum orders moved per cycle |
| `enableAutoRebalancing` | bool | true | Automatically trigger rebalancing |
| `enableAutoRerouting` | bool | true | Automatically trigger rerouting |
| `enableCapacityPrediction` | bool | true | Generate capacity forecasts |

## Integration with Other Systems

### Process Path Escalation (Phase 3.2)
When continuous optimization detects constraints, it works with the escalation system:
1. **Capacity exceeded** → Triggers path escalation to `standard` or `degraded` tier
2. **Rebalancing fails** → Orders may escalate to `manual` tier
3. **Fallback stations** → Uses escalation's fallback station list

### Real-Time Routing (Phase 3.1)
Optimization uses the ML routing optimizer for decisions:
1. **Station selection** → Calls `OptimizeStationSelection` for rebalanced orders
2. **Metrics collection** → Uses `GetRoutingMetrics` for health monitoring
3. **Rerouting** → Calls `RerouteOrder` for dynamic adjustments

### Wave Planning (Phases 1 & 2)
Rebalancing affects wave composition:
1. **Wave modification** → Can reassign orders between waves
2. **Station reservations** → Respects capacity reservations
3. **Worker certifications** → Validates certified labor availability

## Metrics and Monitoring

### Key Performance Indicators

**System Health Metrics**:
- Average station utilization across facility
- Percentage of time in healthy/degraded/critical state
- Average confidence score for routing decisions
- Capacity-constrained rate (% of orders delayed by capacity)

**Optimization Impact Metrics**:
- Total orders rebalanced per day/week
- Total orders rerouted per day/week
- Average improvement score from rerouting
- Rebalancing cycle success rate

**Capacity Planning Metrics**:
- Prediction accuracy (actual vs. predicted volume)
- Bottleneck prediction accuracy
- Lead time for capacity adjustments
- Staffing recommendation effectiveness

### Logging

All optimization activities log structured data:

```
INFO  [MonitorSystemHealth] System health checked
  overallHealth=degraded
  overloadedStations=3
  underutilizedStations=2
  rebalancingRecommended=true

INFO  [RebalanceWaves] Rebalancing completed
  ordersRebalanced=45
  totalRebalancingEvents=12
  stationsAffected=5

INFO  [TriggerDynamicRerouting] Dynamic rerouting completed
  ordersRerouted=23
  averageConfidence=0.87
  totalReroutingEvents=8

INFO  [PredictCapacityNeeds] Capacity prediction generated
  predictedVolume=1200
  predictedBottlenecks=2
  confidenceScore=0.75
```

## Best Practices

### Monitoring Interval Selection
- **High volume facilities**: 3-5 minutes for rapid response
- **Medium volume facilities**: 5-10 minutes for balance
- **Low volume facilities**: 10-15 minutes to reduce overhead

### Threshold Tuning
- **Aggressive optimization**: Lower capacity threshold (0.75-0.80)
- **Balanced optimization**: Standard threshold (0.85)
- **Conservative optimization**: Higher threshold (0.90-0.95)

### Rebalancing Limits
- **Peak hours**: Lower limits (25-50 orders) to minimize disruption
- **Off-peak hours**: Higher limits (50-100 orders) for aggressive optimization
- **Special events**: Disable auto-rebalancing and use manual control

### Prediction Windows
- **Short-term tactical**: 1-4 hour predictions for immediate actions
- **Medium-term operational**: 4-12 hour predictions for shift planning
- **Long-term strategic**: 24+ hour predictions for capacity planning

## Troubleshooting

### High Rebalancing Frequency
**Symptom**: Rebalancing triggers every cycle
**Causes**:
- Capacity threshold too low
- Insufficient station capacity overall
- Uneven order distribution in wave planning

**Solutions**:
1. Increase `capacityThreshold` to 0.90
2. Increase `maxOrdersPerRebalance` for more aggressive rebalancing
3. Review wave planning configuration for better initial distribution

### Low Rerouting Confidence
**Symptom**: `averageConfidence` below 0.60
**Causes**:
- Insufficient station alternatives
- All stations similarly loaded
- Poor routing optimizer training data

**Solutions**:
1. Add more stations with overlapping capabilities
2. Adjust routing optimizer weights in Phase 3.1 configuration
3. Collect more historical routing data for ML training

### Prediction Inaccuracy
**Symptom**: Actual volume differs significantly from predictions
**Causes**:
- Insufficient historical data
- Seasonal patterns not captured
- Recent changes in business operations

**Solutions**:
1. Increase `historicalWindow` to capture more patterns
2. Adjust growth factor in prediction algorithm
3. Integrate external demand forecasting data

## Future Enhancements

### Planned Improvements
1. **ML-based prediction**: Replace rule-based forecasting with trained models
2. **Multi-facility optimization**: Coordinate optimization across multiple facilities
3. **Predictive escalation**: Escalate paths preemptively based on predictions
4. **Dynamic threshold adjustment**: Auto-tune thresholds based on historical performance
5. **Cost optimization**: Factor in labor costs, equipment costs in rebalancing decisions
6. **A/B testing framework**: Test optimization strategies and measure impact

### Integration Opportunities
1. **Demand planning integration**: Use sales forecasts for better predictions
2. **Labor management integration**: Coordinate with shift scheduling
3. **Inventory management**: Optimize based on inventory positions
4. **Transportation management**: Coordinate with outbound shipping schedules

## Summary

The Continuous Optimization System completes the Amazon-level fulfillment optimization stack:

| Phase | Capability | Status |
|-------|-----------|--------|
| 1 | Path-Aware Wave Planning | ✅ Complete |
| 2 | Resource Integration (Capacity/Labor/Equipment) | ✅ Complete |
| 3.1 | Real-Time Routing (ATROPS-like) | ✅ Complete |
| 3.2 | Conditional Path Escalation | ✅ Complete |
| 3.3 | Continuous Optimization | ✅ Complete |

Your WMS platform now features:
- **Adaptive routing** that adjusts to real-time conditions
- **4-tier escalation** matching Amazon's receive workflow
- **Automatic rebalancing** to maintain optimal load distribution
- **Predictive capacity planning** for proactive resource allocation
- **Dynamic rerouting** for in-flight optimization

This provides the foundation for Amazon-level warehouse efficiency and throughput optimization.
