---
sidebar_position: 18
slug: /temporal/activities/labor-activities
---

# Labor Activities

Activities for managing worker certifications, assignments, and availability.

## Activity Struct

```go
type LaborActivities struct {
    clients *clients.ServiceClients
}
```

## Activities

### ValidateWorkerCertification

Validates that certified workers are available for required skills.

**Signature:**
```go
func (a *LaborActivities) ValidateWorkerCertification(ctx context.Context, input ValidateCertificationInput) (*ValidateCertificationResult, error)
```

**Input:**
```go
type ValidateCertificationInput struct {
    RequiredSkills []string `json:"requiredSkills"` // Skills needed
    Zone           string   `json:"zone,omitempty"`
    ShiftTime      string   `json:"shiftTime,omitempty"`
    MinWorkers     int      `json:"minWorkers"`     // Minimum count needed
}
```

**Output:**
```go
type ValidateCertificationResult struct {
    CertifiedWorkersAvailable int      `json:"certifiedWorkersAvailable"`
    AvailableWorkerIDs        []string `json:"availableWorkerIds"`
    MissingSkills             []string `json:"missingSkills,omitempty"`
    SufficientLabor           bool     `json:"sufficientLabor"`
    Success                   bool     `json:"success"`
}
```

**Behavior:**
- Returns success with `SufficientLabor: true` if no special skills required
- Identifies which skills have insufficient coverage
- Used for hazmat, cold chain, and other special handling validation

---

### AssignCertifiedWorker

Assigns a certified worker to an order/station.

**Signature:**
```go
func (a *LaborActivities) AssignCertifiedWorker(ctx context.Context, input AssignCertifiedWorkerInput) (*clients.Worker, error)
```

**Input:**
```go
type AssignCertifiedWorkerInput struct {
    OrderID        string   `json:"orderId"`
    StationID      string   `json:"stationId"`
    RequiredSkills []string `json:"requiredSkills"`
    Zone           string   `json:"zone,omitempty"`
    Priority       string   `json:"priority,omitempty"`
}
```

**Output:**
```go
type Worker struct {
    WorkerID string   `json:"workerId"`
    Name     string   `json:"name"`
    Skills   []string `json:"skills"`
    Zone     string   `json:"zone"`
    Status   string   `json:"status"`
}
```

---

### GetAvailableWorkers

Retrieves available workers, optionally filtered by skills.

**Signature:**
```go
func (a *LaborActivities) GetAvailableWorkers(ctx context.Context, input GetAvailableWorkersInput) ([]clients.Worker, error)
```

**Input:**
```go
type GetAvailableWorkersInput struct {
    Zone           string   `json:"zone,omitempty"`
    RequiredSkills []string `json:"requiredSkills,omitempty"`
    ShiftTime      string   `json:"shiftTime,omitempty"`
}
```

**Output:** Array of `Worker` objects

## Common Skills

| Skill | Description | Use Case |
|-------|-------------|----------|
| `hazmat_handling` | Hazardous materials certified | Hazmat orders |
| `cold_chain` | Temperature-controlled handling | Perishable items |
| `forklift` | Forklift operation certified | Heavy items |
| `fragile_handling` | Specialized fragile item handling | High-value fragile |
| `gift_wrap` | Gift wrapping trained | Gift wrap orders |

## Configuration

| Property | Value |
|----------|-------|
| Default Timeout | 2 minutes |
| Retry Policy | 3 maximum attempts |
| Heartbeat | Not required |

## Error Handling

| Error | Description | Recovery |
|-------|-------------|----------|
| No certified workers | Required skills not covered | Escalate or queue |
| Assignment failed | Worker unavailable | Retry with different worker |
| Service unavailable | Labor service down | Standard retry policy |

## Usage Example

```go
// Validate before assignment
validateInput := activities.ValidateCertificationInput{
    RequiredSkills: []string{"hazmat_handling"},
    Zone:           "ZONE-A",
    MinWorkers:     1,
}

var validation activities.ValidateCertificationResult
err := workflow.ExecuteActivity(ctx, laborActivities.ValidateWorkerCertification, validateInput).Get(ctx, &validation)

if validation.SufficientLabor {
    // Assign worker
    assignInput := activities.AssignCertifiedWorkerInput{
        OrderID:        "ORD-12345",
        StationID:      "STATION-A01",
        RequiredSkills: []string{"hazmat_handling"},
    }

    var worker clients.Worker
    err = workflow.ExecuteActivity(ctx, laborActivities.AssignCertifiedWorker, assignInput).Get(ctx, &worker)
}
```

## Related Workflows

- [Planning Workflow](../workflows/planning) - Validates labor before planning
- [WES Execution Workflow](../workflows/wes-execution) - Assigns workers to tasks

## Related Documentation

- [Labor Service](/services/labor-service) - Workforce management
