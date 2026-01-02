package activities

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
)

// SLAMActivities contains activities for the SLAM process
// SLAM = Scan, Label, Apply, Manifest
type SLAMActivities struct {
	// In a real implementation, these would be service clients
}

// NewSLAMActivities creates a new SLAMActivities instance
func NewSLAMActivities() *SLAMActivities {
	return &SLAMActivities{}
}

// ScanPackageInput represents input for package scanning
type ScanPackageInput struct {
	OrderID        string  `json:"orderId"`
	PackageID      string  `json:"packageId"`
	ExpectedWeight float64 `json:"expectedWeight"`
	ExpectedSKUs   []string `json:"expectedSkus"`
}

// ScanPackageResult represents the result of package scanning
type ScanPackageResult struct {
	PackageID    string    `json:"packageId"`
	ActualWeight float64   `json:"actualWeight"`
	VerifiedSKUs []string  `json:"verifiedSkus"`
	ScanTime     time.Time `json:"scanTime"`
	Success      bool      `json:"success"`
	Errors       []string  `json:"errors,omitempty"`
}

// ScanPackage scans and verifies package contents
func (a *SLAMActivities) ScanPackage(ctx context.Context, input ScanPackageInput) (*ScanPackageResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Scanning package",
		"orderId", input.OrderID,
		"packageId", input.PackageID,
		"expectedWeight", input.ExpectedWeight,
	)

	// Simulate package scanning
	// In production, this would integrate with barcode scanners and scales
	actualWeight := input.ExpectedWeight * (0.98 + 0.04*float64(time.Now().UnixNano()%100)/100) // Simulate slight weight variance

	result := &ScanPackageResult{
		PackageID:    input.PackageID,
		ActualWeight: math.Round(actualWeight*100) / 100,
		VerifiedSKUs: input.ExpectedSKUs,
		ScanTime:     time.Now(),
		Success:      true,
	}

	return result, nil
}

// GenerateLabelInput represents input for label generation
type GenerateLabelInput struct {
	OrderID       string  `json:"orderId"`
	PackageID     string  `json:"packageId"`
	CarrierCode   string  `json:"carrierCode"`
	ServiceType   string  `json:"serviceType"`
	Weight        float64 `json:"weight"`
	RecipientName string  `json:"recipientName"`
	RecipientAddr string  `json:"recipientAddress"`
	RecipientCity string  `json:"recipientCity"`
	RecipientZip  string  `json:"recipientZip"`
}

// GenerateLabelResult represents the result of label generation
type GenerateLabelResult struct {
	PackageID      string    `json:"packageId"`
	TrackingNumber string    `json:"trackingNumber"`
	LabelFormat    string    `json:"labelFormat"`
	LabelURL       string    `json:"labelUrl"`
	GeneratedAt    time.Time `json:"generatedAt"`
	Success        bool      `json:"success"`
}

// GenerateLabel generates a shipping label
func (a *SLAMActivities) GenerateLabel(ctx context.Context, input GenerateLabelInput) (*GenerateLabelResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating shipping label",
		"orderId", input.OrderID,
		"packageId", input.PackageID,
		"carrier", input.CarrierCode,
	)

	// Simulate label generation
	// In production, this would call carrier APIs (UPS, FedEx, USPS, etc.)
	trackingNumber := fmt.Sprintf("%s%s", input.CarrierCode[:2], uuid.New().String()[:12])

	result := &GenerateLabelResult{
		PackageID:      input.PackageID,
		TrackingNumber: trackingNumber,
		LabelFormat:    "ZPL", // Zebra Printer Language
		LabelURL:       fmt.Sprintf("https://labels.wms.local/%s/%s.zpl", input.CarrierCode, trackingNumber),
		GeneratedAt:    time.Now(),
		Success:        true,
	}

	return result, nil
}

// ApplyLabelInput represents input for applying label
type ApplyLabelInput struct {
	PackageID      string `json:"packageId"`
	TrackingNumber string `json:"trackingNumber"`
	StationID      string `json:"stationId"`
	WorkerID       string `json:"workerId"`
}

// ApplyLabelResult represents the result of label application
type ApplyLabelResult struct {
	PackageID      string    `json:"packageId"`
	TrackingNumber string    `json:"trackingNumber"`
	AppliedAt      time.Time `json:"appliedAt"`
	StationID      string    `json:"stationId"`
	Success        bool      `json:"success"`
}

// ApplyLabel confirms label application to package
func (a *SLAMActivities) ApplyLabel(ctx context.Context, input ApplyLabelInput) (*ApplyLabelResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Confirming label application",
		"packageId", input.PackageID,
		"trackingNumber", input.TrackingNumber,
		"stationId", input.StationID,
	)

	// In production, this would:
	// 1. Verify label was printed successfully
	// 2. Confirm label was applied to correct package
	// 3. Scan verification barcode
	// 4. Update package status

	result := &ApplyLabelResult{
		PackageID:      input.PackageID,
		TrackingNumber: input.TrackingNumber,
		AppliedAt:      time.Now(),
		StationID:      input.StationID,
		Success:        true,
	}

	return result, nil
}

// AddToManifestInput represents input for adding to manifest
type AddToManifestInput struct {
	PackageID      string  `json:"packageId"`
	OrderID        string  `json:"orderId"`
	TrackingNumber string  `json:"trackingNumber"`
	CarrierCode    string  `json:"carrierCode"`
	Weight         float64 `json:"weight"`
	Destination    string  `json:"destination"`
}

// AddToManifestResult represents the result of manifest assignment
type AddToManifestResult struct {
	PackageID    string    `json:"packageId"`
	ManifestID   string    `json:"manifestId"`
	BatchID      string    `json:"batchId"`
	AssignedChute string   `json:"assignedChute,omitempty"`
	ManifestedAt time.Time `json:"manifestedAt"`
	Success      bool      `json:"success"`
}

// AddToManifest adds a package to a carrier manifest
func (a *SLAMActivities) AddToManifest(ctx context.Context, input AddToManifestInput) (*AddToManifestResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Adding package to manifest",
		"packageId", input.PackageID,
		"trackingNumber", input.TrackingNumber,
		"carrier", input.CarrierCode,
	)

	// In production, this would:
	// 1. Find or create appropriate manifest for carrier/destination
	// 2. Add package to manifest
	// 3. Assign sortation chute if applicable
	// 4. Update manifest totals

	manifestID := fmt.Sprintf("MAN-%s-%s", input.CarrierCode, time.Now().Format("20060102"))
	batchID := fmt.Sprintf("SRT-%s-%s", input.Destination[:3], time.Now().Format("150405"))

	result := &AddToManifestResult{
		PackageID:     input.PackageID,
		ManifestID:    manifestID,
		BatchID:       batchID,
		AssignedChute: fmt.Sprintf("CHUTE-%02d", time.Now().Unix()%20+1),
		ManifestedAt:  time.Now(),
		Success:       true,
	}

	return result, nil
}

// VerifyWeightInput represents input for weight verification
type VerifyWeightInput struct {
	PackageID      string  `json:"packageId"`
	ExpectedWeight float64 `json:"expectedWeight"`
	ActualWeight   float64 `json:"actualWeight"`
	TolerancePercent float64 `json:"tolerancePercent"`
}

// VerifyWeightResult represents the result of weight verification
type VerifyWeightResult struct {
	PackageID        string  `json:"packageId"`
	WithinTolerance  bool    `json:"withinTolerance"`
	VariancePercent  float64 `json:"variancePercent"`
	ExpectedWeight   float64 `json:"expectedWeight"`
	ActualWeight     float64 `json:"actualWeight"`
	RequiresReview   bool    `json:"requiresReview"`
}

// VerifyWeight verifies package weight is within tolerance
func (a *SLAMActivities) VerifyWeight(ctx context.Context, input VerifyWeightInput) (*VerifyWeightResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Verifying weight",
		"packageId", input.PackageID,
		"expected", input.ExpectedWeight,
		"actual", input.ActualWeight,
	)

	variance := math.Abs(input.ActualWeight-input.ExpectedWeight) / input.ExpectedWeight * 100
	withinTolerance := variance <= input.TolerancePercent

	result := &VerifyWeightResult{
		PackageID:       input.PackageID,
		WithinTolerance: withinTolerance,
		VariancePercent: math.Round(variance*100) / 100,
		ExpectedWeight:  input.ExpectedWeight,
		ActualWeight:    input.ActualWeight,
		RequiresReview:  !withinTolerance,
	}

	if !withinTolerance {
		logger.Warn("Weight variance exceeds tolerance",
			"packageId", input.PackageID,
			"variance", variance,
			"tolerance", input.TolerancePercent,
		)
	}

	return result, nil
}

// ExecuteSLAMInput represents input for the full SLAM process
type ExecuteSLAMInput struct {
	OrderID        string   `json:"orderId"`
	PackageID      string   `json:"packageId"`
	ExpectedWeight float64  `json:"expectedWeight"`
	ExpectedSKUs   []string `json:"expectedSkus"`
	CarrierCode    string   `json:"carrierCode"`
	ServiceType    string   `json:"serviceType"`
	RecipientName  string   `json:"recipientName"`
	RecipientAddr  string   `json:"recipientAddress"`
	RecipientCity  string   `json:"recipientCity"`
	RecipientZip   string   `json:"recipientZip"`
	StationID      string   `json:"stationId"`
	WorkerID       string   `json:"workerId"`
}

// ExecuteSLAMResult represents the full SLAM process result
type ExecuteSLAMResult struct {
	TaskID                string  `json:"taskId"`
	PackageID             string  `json:"packageId"`
	TrackingNumber        string  `json:"trackingNumber"`
	ManifestID            string  `json:"manifestId"`
	ActualWeight          float64 `json:"actualWeight"`
	WeightVariancePercent float64 `json:"weightVariancePercent"`
	WeightVerified        bool    `json:"weightVerified"`
	LabelApplied          bool    `json:"labelApplied"`
	Manifested            bool    `json:"manifested"`
	Success               bool    `json:"success"`
	Errors                []string `json:"errors,omitempty"`
}

// ExecuteSLAM executes the full SLAM process
func (a *SLAMActivities) ExecuteSLAM(ctx context.Context, input ExecuteSLAMInput) (*ExecuteSLAMResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing SLAM process",
		"orderId", input.OrderID,
		"packageId", input.PackageID,
	)

	taskID := fmt.Sprintf("SLAM-%s", uuid.New().String()[:8])
	result := &ExecuteSLAMResult{
		TaskID:    taskID,
		PackageID: input.PackageID,
		Success:   true,
	}

	// Step 1: Scan and weigh
	scanResult, err := a.ScanPackage(ctx, ScanPackageInput{
		OrderID:        input.OrderID,
		PackageID:      input.PackageID,
		ExpectedWeight: input.ExpectedWeight,
		ExpectedSKUs:   input.ExpectedSKUs,
	})
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("scan failed: %v", err))
		return result, nil
	}
	result.ActualWeight = scanResult.ActualWeight

	// Step 2: Verify weight
	weightResult, _ := a.VerifyWeight(ctx, VerifyWeightInput{
		PackageID:        input.PackageID,
		ExpectedWeight:   input.ExpectedWeight,
		ActualWeight:     scanResult.ActualWeight,
		TolerancePercent: 5.0, // 5% tolerance
	})
	result.WeightVariancePercent = weightResult.VariancePercent
	result.WeightVerified = weightResult.WithinTolerance

	// Step 3: Generate label
	labelResult, err := a.GenerateLabel(ctx, GenerateLabelInput{
		OrderID:       input.OrderID,
		PackageID:     input.PackageID,
		CarrierCode:   input.CarrierCode,
		ServiceType:   input.ServiceType,
		Weight:        scanResult.ActualWeight,
		RecipientName: input.RecipientName,
		RecipientAddr: input.RecipientAddr,
		RecipientCity: input.RecipientCity,
		RecipientZip:  input.RecipientZip,
	})
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("label generation failed: %v", err))
		return result, nil
	}
	result.TrackingNumber = labelResult.TrackingNumber

	// Step 4: Apply label
	applyResult, err := a.ApplyLabel(ctx, ApplyLabelInput{
		PackageID:      input.PackageID,
		TrackingNumber: labelResult.TrackingNumber,
		StationID:      input.StationID,
		WorkerID:       input.WorkerID,
	})
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("label apply failed: %v", err))
		return result, nil
	}
	result.LabelApplied = applyResult.Success

	// Step 5: Add to manifest
	manifestResult, err := a.AddToManifest(ctx, AddToManifestInput{
		PackageID:      input.PackageID,
		OrderID:        input.OrderID,
		TrackingNumber: labelResult.TrackingNumber,
		CarrierCode:    input.CarrierCode,
		Weight:         scanResult.ActualWeight,
		Destination:    input.RecipientZip,
	})
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("manifest failed: %v", err))
		return result, nil
	}
	result.ManifestID = manifestResult.ManifestID
	result.Manifested = manifestResult.Success

	logger.Info("SLAM process completed",
		"taskId", taskID,
		"trackingNumber", result.TrackingNumber,
		"success", result.Success,
	)

	return result, nil
}

// RegisterSLAMActivities registers all SLAM activities with the worker
func RegisterSLAMActivities(activities *SLAMActivities) map[string]interface{} {
	return map[string]interface{}{
		"ScanPackage":    activities.ScanPackage,
		"GenerateLabel":  activities.GenerateLabel,
		"ApplyLabel":     activities.ApplyLabel,
		"AddToManifest":  activities.AddToManifest,
		"VerifyWeight":   activities.VerifyWeight,
		"ExecuteSLAM":    activities.ExecuteSLAM,
	}
}
