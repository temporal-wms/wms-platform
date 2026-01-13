package domain

import (
	"math"
	"time"
)

// RoutingOptimizer provides intelligent routing decisions based on real-time conditions
type RoutingOptimizer struct {
	// Weights for scoring algorithm (configurable)
	CapacityWeight      float64
	DistanceWeight      float64
	UtilizationWeight   float64
	ThroughputWeight    float64
	SLAWeight           float64
	CertificationWeight float64
}

// NewRoutingOptimizer creates a new routing optimizer with default weights
func NewRoutingOptimizer() *RoutingOptimizer {
	return &RoutingOptimizer{
		CapacityWeight:      0.30, // 30% weight on available capacity
		DistanceWeight:      0.15, // 15% weight on distance/location
		UtilizationWeight:   0.20, // 20% weight on current utilization
		ThroughputWeight:    0.20, // 20% weight on historical throughput
		SLAWeight:           0.10, // 10% weight on SLA compliance
		CertificationWeight: 0.05, // 5% weight on worker certification availability
	}
}

// StationCandidate represents a candidate station for routing
type StationCandidate struct {
	StationID            string
	StationType          string
	Zone                 string
	Capabilities         []string
	MaxConcurrentTasks   int
	CurrentTasks         int
	AvailableCapacity    int
	CurrentUtilization   float64 // 0.0 to 1.0
	AverageThroughput    float64 // Tasks per hour
	DistanceScore        float64 // Normalized distance score (0.0 to 1.0)
	SLAComplianceRate    float64 // Historical SLA compliance (0.0 to 1.0)
	CertifiedWorkers     int     // Number of certified workers available
	LastTaskCompletedAt  time.Time
}

// OrderRoutingContext contains context for routing an order
type OrderRoutingContext struct {
	OrderID            string
	Priority           string              // same_day, next_day, standard
	Requirements       []ProcessRequirement
	SpecialHandling    []string
	ItemCount          int
	TotalWeight        float64
	PromisedDeliveryAt time.Time
	RequiredSkills     []string
	RequiredEquipment  []string
	Zone               string
}

// RoutingDecision represents the optimal routing decision
type RoutingDecision struct {
	SelectedStationID string
	Score             float64
	Reasoning         map[string]float64 // Factor -> contribution to score
	AlternateStations []AlternateStation
	Confidence        float64 // 0.0 to 1.0
	DecisionTime      time.Time
}

// AlternateStation represents an alternative routing option
type AlternateStation struct {
	StationID string
	Score     float64
	Rank      int
}

// OptimizeStationRouting selects the optimal station based on multiple factors
func (ro *RoutingOptimizer) OptimizeStationRouting(
	context OrderRoutingContext,
	candidates []StationCandidate,
) *RoutingDecision {
	if len(candidates) == 0 {
		return nil
	}

	// Score each candidate
	scoredCandidates := make([]scoredStation, 0, len(candidates))
	for _, candidate := range candidates {
		score, reasoning := ro.scoreStation(context, candidate)
		scoredCandidates = append(scoredCandidates, scoredStation{
			candidate: candidate,
			score:     score,
			reasoning: reasoning,
		})
	}

	// Sort by score (highest first)
	for i := 0; i < len(scoredCandidates); i++ {
		for j := i + 1; j < len(scoredCandidates); j++ {
			if scoredCandidates[j].score > scoredCandidates[i].score {
				scoredCandidates[i], scoredCandidates[j] = scoredCandidates[j], scoredCandidates[i]
			}
		}
	}

	// Select best candidate
	best := scoredCandidates[0]

	// Build alternates list
	alternates := make([]AlternateStation, 0, len(scoredCandidates)-1)
	for i := 1; i < len(scoredCandidates) && i <= 5; i++ { // Top 5 alternates
		alternates = append(alternates, AlternateStation{
			StationID: scoredCandidates[i].candidate.StationID,
			Score:     scoredCandidates[i].score,
			Rank:      i + 1,
		})
	}

	// Calculate confidence based on score distribution
	confidence := ro.calculateConfidence(scoredCandidates)

	return &RoutingDecision{
		SelectedStationID: best.candidate.StationID,
		Score:             best.score,
		Reasoning:         best.reasoning,
		AlternateStations: alternates,
		Confidence:        confidence,
		DecisionTime:      time.Now(),
	}
}

// scoreStation calculates a weighted score for a station candidate
func (ro *RoutingOptimizer) scoreStation(
	context OrderRoutingContext,
	candidate StationCandidate,
) (float64, map[string]float64) {
	reasoning := make(map[string]float64)

	// 1. Capacity Score (higher available capacity = better)
	capacityScore := float64(candidate.AvailableCapacity) / float64(candidate.MaxConcurrentTasks)
	if capacityScore > 1.0 {
		capacityScore = 1.0
	}
	reasoning["capacity"] = capacityScore * ro.CapacityWeight

	// 2. Distance/Zone Score (same zone = better)
	distanceScore := 1.0 // Default: same facility
	if context.Zone != "" && candidate.Zone != context.Zone {
		distanceScore = 0.5 // Cross-zone penalty
	}
	reasoning["distance"] = distanceScore * ro.DistanceWeight

	// 3. Utilization Score (lower utilization = better, but not idle)
	utilizationScore := 1.0 - candidate.CurrentUtilization
	if candidate.CurrentUtilization < 0.3 {
		// Penalty for idle stations (want to keep stations active)
		utilizationScore = candidate.CurrentUtilization / 0.3
	}
	reasoning["utilization"] = utilizationScore * ro.UtilizationWeight

	// 4. Throughput Score (higher throughput = better)
	// Normalize throughput (assuming typical range 10-100 tasks/hour)
	throughputScore := math.Min(candidate.AverageThroughput/100.0, 1.0)
	reasoning["throughput"] = throughputScore * ro.ThroughputWeight

	// 5. SLA Compliance Score
	slaScore := candidate.SLAComplianceRate
	// Boost for priority orders
	if context.Priority == "same_day" {
		slaScore *= 1.2
		if slaScore > 1.0 {
			slaScore = 1.0
		}
	}
	reasoning["sla"] = slaScore * ro.SLAWeight

	// 6. Certification Score (availability of certified workers)
	certificationScore := 0.5 // Default: assume some availability
	if len(context.RequiredSkills) > 0 {
		// Score based on certified worker availability
		certificationScore = math.Min(float64(candidate.CertifiedWorkers)/float64(len(context.RequiredSkills)), 1.0)
	}
	reasoning["certification"] = certificationScore * ro.CertificationWeight

	// Calculate total weighted score
	totalScore := reasoning["capacity"] +
		reasoning["distance"] +
		reasoning["utilization"] +
		reasoning["throughput"] +
		reasoning["sla"] +
		reasoning["certification"]

	return totalScore, reasoning
}

// calculateConfidence calculates confidence in the routing decision
func (ro *RoutingOptimizer) calculateConfidence(scored []scoredStation) float64 {
	if len(scored) == 0 {
		return 0.0
	}
	if len(scored) == 1 {
		return 1.0 // Only one option, fully confident
	}

	// Confidence based on score gap between best and second-best
	best := scored[0].score
	secondBest := scored[1].score

	if best == 0 {
		return 0.5 // Low confidence if best score is zero
	}

	scoreGap := best - secondBest
	confidence := math.Min(scoreGap/best, 1.0)

	// Ensure minimum confidence of 0.5 if we have a clear winner
	if confidence < 0.5 && scoreGap > 0.1 {
		confidence = 0.5
	}

	return confidence
}

// scoredStation is a helper struct for sorting
type scoredStation struct {
	candidate StationCandidate
	score     float64
	reasoning map[string]float64
}

// DynamicRoutingMetrics captures real-time metrics for routing decisions
type DynamicRoutingMetrics struct {
	TotalRoutingDecisions   int
	AverageDecisionTime     time.Duration
	AverageConfidence       float64
	StationUtilization      map[string]float64 // Station ID -> utilization
	CapacityConstrainedRate float64            // % of times constrained by capacity
	RouteChanges            int                // Number of rerouting events
	LastUpdated             time.Time
}

// RecommendRebalancing suggests whether wave rebalancing is needed
func (ro *RoutingOptimizer) RecommendRebalancing(metrics DynamicRoutingMetrics) bool {
	// Rebalance if capacity constraints are frequent
	if metrics.CapacityConstrainedRate > 0.7 {
		return true
	}

	// Rebalance if confidence is consistently low
	if metrics.AverageConfidence < 0.6 {
		return true
	}

	// Check for uneven station utilization
	if ro.hasUnevenUtilization(metrics.StationUtilization) {
		return true
	}

	return false
}

// hasUnevenUtilization checks if station utilization is unbalanced
func (ro *RoutingOptimizer) hasUnevenUtilization(utilization map[string]float64) bool {
	if len(utilization) < 2 {
		return false
	}

	var sum, count float64
	for _, util := range utilization {
		sum += util
		count++
	}

	avgUtilization := sum / count

	// Check variance
	var variance float64
	for _, util := range utilization {
		diff := util - avgUtilization
		variance += diff * diff
	}
	variance /= count

	// If standard deviation > 0.3, utilization is uneven
	stdDev := math.Sqrt(variance)
	return stdDev > 0.3
}
