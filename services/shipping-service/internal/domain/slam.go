package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SLAM errors
var (
	ErrSLAMAlreadyComplete      = errors.New("SLAM process already complete")
	ErrInvalidSLAMStep          = errors.New("invalid SLAM step")
	ErrWeightOutOfTolerance     = errors.New("weight out of tolerance")
	ErrScanVerificationFailed   = errors.New("scan verification failed")
	ErrLabelNotApplied          = errors.New("label has not been applied")
	ErrSLAMInvalidTransition    = errors.New("invalid SLAM state transition")
)

// Default weight tolerance (5% variance allowed)
const DefaultWeightTolerancePercent = 5.0

// SLAMStatus represents the status of the SLAM process
type SLAMStatus string

const (
	SLAMStatusPending     SLAMStatus = "pending"
	SLAMStatusScanning    SLAMStatus = "scanning"
	SLAMStatusLabeling    SLAMStatus = "labeling"
	SLAMStatusApplying    SLAMStatus = "applying"
	SLAMStatusManifesting SLAMStatus = "manifesting"
	SLAMStatusComplete    SLAMStatus = "complete"
	SLAMStatusFailed      SLAMStatus = "failed"
)

// IsValid checks if the status is valid
func (s SLAMStatus) IsValid() bool {
	switch s {
	case SLAMStatusPending, SLAMStatusScanning, SLAMStatusLabeling,
		SLAMStatusApplying, SLAMStatusManifesting, SLAMStatusComplete, SLAMStatusFailed:
		return true
	default:
		return false
	}
}

// SLAMStep represents a single step in the SLAM process
type SLAMStep struct {
	StepType    string     `bson:"stepType" json:"stepType"` // scan, label, apply, manifest
	Status      string     `bson:"status" json:"status"`     // pending, in_progress, completed, failed
	StartedAt   *time.Time `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt *time.Time `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	PerformedBy string     `bson:"performedBy,omitempty" json:"performedBy,omitempty"`
	Notes       string     `bson:"notes,omitempty" json:"notes,omitempty"`
}

// WeightVerification represents weight verification data
type WeightVerification struct {
	ExpectedWeight    float64 `bson:"expectedWeight" json:"expectedWeight"`       // in kg
	ActualWeight      float64 `bson:"actualWeight" json:"actualWeight"`           // in kg
	Variance          float64 `bson:"variance" json:"variance"`                   // absolute difference
	VariancePercent   float64 `bson:"variancePercent" json:"variancePercent"`     // percentage difference
	TolerancePercent  float64 `bson:"tolerancePercent" json:"tolerancePercent"`   // allowed tolerance
	Passed            bool    `bson:"passed" json:"passed"`
	VerifiedAt        time.Time `bson:"verifiedAt" json:"verifiedAt"`
	VerifiedBy        string  `bson:"verifiedBy,omitempty" json:"verifiedBy,omitempty"`
	ScaleID           string  `bson:"scaleId,omitempty" json:"scaleId,omitempty"`
}

// BarcodeVerification represents barcode scan verification
type BarcodeVerification struct {
	ExpectedBarcode string    `bson:"expectedBarcode" json:"expectedBarcode"`
	ScannedBarcode  string    `bson:"scannedBarcode" json:"scannedBarcode"`
	Matched         bool      `bson:"matched" json:"matched"`
	ScannerID       string    `bson:"scannerId,omitempty" json:"scannerId,omitempty"`
	ScannedAt       time.Time `bson:"scannedAt" json:"scannedAt"`
}

// LabelApplication represents label application data
type LabelApplication struct {
	LabelID        string    `bson:"labelId" json:"labelId"`
	TrackingNumber string    `bson:"trackingNumber" json:"trackingNumber"`
	LabelFormat    string    `bson:"labelFormat" json:"labelFormat"` // ZPL, PDF, PNG
	PrinterID      string    `bson:"printerId,omitempty" json:"printerId,omitempty"`
	PrintedAt      time.Time `bson:"printedAt" json:"printedAt"`
	AppliedAt      *time.Time `bson:"appliedAt,omitempty" json:"appliedAt,omitempty"`
	AppliedBy      string    `bson:"appliedBy,omitempty" json:"appliedBy,omitempty"`
	Verified       bool      `bson:"verified" json:"verified"`
}

// ManifestAssignment represents manifest assignment data
type ManifestAssignment struct {
	ManifestID   string    `bson:"manifestId" json:"manifestId"`
	CarrierCode  string    `bson:"carrierCode" json:"carrierCode"`
	PickupTime   *time.Time `bson:"pickupTime,omitempty" json:"pickupTime,omitempty"`
	AssignedAt   time.Time `bson:"assignedAt" json:"assignedAt"`
	AssignedBy   string    `bson:"assignedBy,omitempty" json:"assignedBy,omitempty"`
}

// SLAMTask represents the SLAM (Scan, Label, Apply, Manifest) process
type SLAMTask struct {
	ID                  primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	TaskID              string               `bson:"taskId" json:"taskId"`
	PackageID           string               `bson:"packageId" json:"packageId"`
	OrderID             string               `bson:"orderId" json:"orderId"`
	ShipmentID          string               `bson:"shipmentId" json:"shipmentId"`
	Status              SLAMStatus           `bson:"status" json:"status"`
	Steps               []SLAMStep           `bson:"steps" json:"steps"`
	BarcodeVerification *BarcodeVerification `bson:"barcodeVerification,omitempty" json:"barcodeVerification,omitempty"`
	WeightVerification  *WeightVerification  `bson:"weightVerification,omitempty" json:"weightVerification,omitempty"`
	LabelApplication    *LabelApplication    `bson:"labelApplication,omitempty" json:"labelApplication,omitempty"`
	ManifestAssignment  *ManifestAssignment  `bson:"manifestAssignment,omitempty" json:"manifestAssignment,omitempty"`
	StationID           string               `bson:"stationId,omitempty" json:"stationId,omitempty"`
	AssignedWorkerID    string               `bson:"assignedWorkerId,omitempty" json:"assignedWorkerId,omitempty"`
	FailureReason       string               `bson:"failureReason,omitempty" json:"failureReason,omitempty"`
	StartedAt           *time.Time           `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt         *time.Time           `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	CreatedAt           time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt           time.Time            `bson:"updatedAt" json:"updatedAt"`
	DomainEvents        []DomainEvent        `bson:"-" json:"-"`
}

// NewSLAMTask creates a new SLAM task
func NewSLAMTask(taskID, packageID, orderID, shipmentID string) *SLAMTask {
	now := time.Now().UTC()
	task := &SLAMTask{
		ID:         primitive.NewObjectID(),
		TaskID:     taskID,
		PackageID:  packageID,
		OrderID:    orderID,
		ShipmentID: shipmentID,
		Status:     SLAMStatusPending,
		Steps: []SLAMStep{
			{StepType: "scan", Status: "pending"},
			{StepType: "label", Status: "pending"},
			{StepType: "apply", Status: "pending"},
			{StepType: "manifest", Status: "pending"},
		},
		CreatedAt:    now,
		UpdatedAt:    now,
		DomainEvents: make([]DomainEvent, 0),
	}

	task.addDomainEvent(&SLAMTaskCreatedEvent{
		TaskID:     taskID,
		PackageID:  packageID,
		OrderID:    orderID,
		ShipmentID: shipmentID,
		CreatedAt:  now,
	})

	return task
}

// StartScan starts the scanning step
func (t *SLAMTask) StartScan(workerID, stationID string) error {
	if t.Status != SLAMStatusPending {
		return ErrSLAMInvalidTransition
	}

	now := time.Now().UTC()
	t.Status = SLAMStatusScanning
	t.AssignedWorkerID = workerID
	t.StationID = stationID
	t.StartedAt = &now
	t.updateStep("scan", "in_progress", &now, nil, workerID)
	t.UpdatedAt = now

	return nil
}

// VerifyBarcode verifies the package barcode
func (t *SLAMTask) VerifyBarcode(expectedBarcode, scannedBarcode, scannerID string) error {
	if t.Status != SLAMStatusScanning {
		return ErrSLAMInvalidTransition
	}

	now := time.Now().UTC()
	matched := expectedBarcode == scannedBarcode

	t.BarcodeVerification = &BarcodeVerification{
		ExpectedBarcode: expectedBarcode,
		ScannedBarcode:  scannedBarcode,
		Matched:         matched,
		ScannerID:       scannerID,
		ScannedAt:       now,
	}

	if !matched {
		t.Status = SLAMStatusFailed
		t.FailureReason = "barcode verification failed"
		t.updateStep("scan", "failed", nil, &now, "")
		t.addDomainEvent(&SLAMScanFailedEvent{
			TaskID:          t.TaskID,
			ExpectedBarcode: expectedBarcode,
			ScannedBarcode:  scannedBarcode,
			FailedAt:        now,
		})
		return ErrScanVerificationFailed
	}

	t.updateStep("scan", "completed", nil, &now, "")
	t.UpdatedAt = now

	t.addDomainEvent(&SLAMScanCompletedEvent{
		TaskID:    t.TaskID,
		PackageID: t.PackageID,
		ScannedAt: now,
	})

	return nil
}

// VerifyWeight verifies the package weight
func (t *SLAMTask) VerifyWeight(expectedWeight, actualWeight float64, tolerancePercent float64, scaleID, workerID string) error {
	if t.Status != SLAMStatusScanning {
		return ErrSLAMInvalidTransition
	}

	if tolerancePercent <= 0 {
		tolerancePercent = DefaultWeightTolerancePercent
	}

	now := time.Now().UTC()
	variance := actualWeight - expectedWeight
	if variance < 0 {
		variance = -variance
	}

	variancePercent := 0.0
	if expectedWeight > 0 {
		variancePercent = (variance / expectedWeight) * 100
	}

	passed := variancePercent <= tolerancePercent

	t.WeightVerification = &WeightVerification{
		ExpectedWeight:   expectedWeight,
		ActualWeight:     actualWeight,
		Variance:         variance,
		VariancePercent:  variancePercent,
		TolerancePercent: tolerancePercent,
		Passed:           passed,
		VerifiedAt:       now,
		VerifiedBy:       workerID,
		ScaleID:          scaleID,
	}

	t.UpdatedAt = now

	t.addDomainEvent(&SLAMWeightVerifiedEvent{
		TaskID:          t.TaskID,
		PackageID:       t.PackageID,
		ExpectedWeight:  expectedWeight,
		ActualWeight:    actualWeight,
		VariancePercent: variancePercent,
		Passed:          passed,
		VerifiedAt:      now,
	})

	if !passed {
		// Weight out of tolerance - may require investigation but don't fail the whole process
		// Return error for workflow to handle
		return ErrWeightOutOfTolerance
	}

	return nil
}

// StartLabeling starts the labeling step
func (t *SLAMTask) StartLabeling() error {
	if t.Status != SLAMStatusScanning {
		return ErrSLAMInvalidTransition
	}

	now := time.Now().UTC()
	t.Status = SLAMStatusLabeling
	t.updateStep("label", "in_progress", &now, nil, t.AssignedWorkerID)
	t.UpdatedAt = now

	return nil
}

// ApplyLabel prints and applies the label
func (t *SLAMTask) ApplyLabel(labelID, trackingNumber, labelFormat, printerID string) error {
	if t.Status != SLAMStatusLabeling {
		return ErrSLAMInvalidTransition
	}

	now := time.Now().UTC()
	t.LabelApplication = &LabelApplication{
		LabelID:        labelID,
		TrackingNumber: trackingNumber,
		LabelFormat:    labelFormat,
		PrinterID:      printerID,
		PrintedAt:      now,
	}

	t.Status = SLAMStatusApplying
	t.updateStep("label", "completed", nil, &now, "")
	t.updateStep("apply", "in_progress", &now, nil, t.AssignedWorkerID)
	t.UpdatedAt = now

	t.addDomainEvent(&SLAMLabelPrintedEvent{
		TaskID:         t.TaskID,
		PackageID:      t.PackageID,
		TrackingNumber: trackingNumber,
		PrinterID:      printerID,
		PrintedAt:      now,
	})

	return nil
}

// ConfirmLabelApplied confirms the label has been applied
func (t *SLAMTask) ConfirmLabelApplied(workerID string) error {
	if t.Status != SLAMStatusApplying || t.LabelApplication == nil {
		return ErrSLAMInvalidTransition
	}

	now := time.Now().UTC()
	t.LabelApplication.AppliedAt = &now
	t.LabelApplication.AppliedBy = workerID
	t.LabelApplication.Verified = true

	t.updateStep("apply", "completed", nil, &now, workerID)
	t.Status = SLAMStatusManifesting
	t.updateStep("manifest", "in_progress", &now, nil, workerID)
	t.UpdatedAt = now

	t.addDomainEvent(&SLAMLabelAppliedEvent{
		TaskID:    t.TaskID,
		PackageID: t.PackageID,
		AppliedBy: workerID,
		AppliedAt: now,
	})

	return nil
}

// AssignToManifest assigns the package to a manifest
func (t *SLAMTask) AssignToManifest(manifestID, carrierCode string, pickupTime *time.Time, workerID string) error {
	if t.Status != SLAMStatusManifesting {
		return ErrSLAMInvalidTransition
	}

	if t.LabelApplication == nil || t.LabelApplication.AppliedAt == nil {
		return ErrLabelNotApplied
	}

	now := time.Now().UTC()
	t.ManifestAssignment = &ManifestAssignment{
		ManifestID:  manifestID,
		CarrierCode: carrierCode,
		PickupTime:  pickupTime,
		AssignedAt:  now,
		AssignedBy:  workerID,
	}

	t.Status = SLAMStatusComplete
	t.CompletedAt = &now
	t.updateStep("manifest", "completed", nil, &now, workerID)
	t.UpdatedAt = now

	t.addDomainEvent(&SLAMManifestAssignedEvent{
		TaskID:         t.TaskID,
		PackageID:      t.PackageID,
		ManifestID:     manifestID,
		CarrierCode:    carrierCode,
		TrackingNumber: t.LabelApplication.TrackingNumber,
		AssignedAt:     now,
	})

	t.addDomainEvent(&SLAMCompletedEvent{
		TaskID:         t.TaskID,
		PackageID:      t.PackageID,
		OrderID:        t.OrderID,
		ShipmentID:     t.ShipmentID,
		TrackingNumber: t.LabelApplication.TrackingNumber,
		ManifestID:     manifestID,
		CompletedAt:    now,
	})

	return nil
}

// Fail marks the SLAM task as failed
func (t *SLAMTask) Fail(reason string) {
	now := time.Now().UTC()
	t.Status = SLAMStatusFailed
	t.FailureReason = reason
	t.UpdatedAt = now

	// Mark current step as failed
	for i := range t.Steps {
		if t.Steps[i].Status == "in_progress" {
			t.Steps[i].Status = "failed"
			t.Steps[i].CompletedAt = &now
			break
		}
	}
}

// IsComplete returns true if SLAM is complete
func (t *SLAMTask) IsComplete() bool {
	return t.Status == SLAMStatusComplete
}

// IsFailed returns true if SLAM has failed
func (t *SLAMTask) IsFailed() bool {
	return t.Status == SLAMStatusFailed
}

// GetTrackingNumber returns the tracking number if available
func (t *SLAMTask) GetTrackingNumber() string {
	if t.LabelApplication != nil {
		return t.LabelApplication.TrackingNumber
	}
	return ""
}

// updateStep updates a specific step
func (t *SLAMTask) updateStep(stepType, status string, startedAt, completedAt *time.Time, performedBy string) {
	for i := range t.Steps {
		if t.Steps[i].StepType == stepType {
			t.Steps[i].Status = status
			if startedAt != nil {
				t.Steps[i].StartedAt = startedAt
			}
			if completedAt != nil {
				t.Steps[i].CompletedAt = completedAt
			}
			if performedBy != "" {
				t.Steps[i].PerformedBy = performedBy
			}
			break
		}
	}
}

// addDomainEvent adds a domain event
func (t *SLAMTask) addDomainEvent(event DomainEvent) {
	t.DomainEvents = append(t.DomainEvents, event)
}

// GetDomainEvents returns all domain events
func (t *SLAMTask) GetDomainEvents() []DomainEvent {
	return t.DomainEvents
}

// ClearDomainEvents clears all domain events
func (t *SLAMTask) ClearDomainEvents() {
	t.DomainEvents = make([]DomainEvent, 0)
}

// SLAM Domain Events

// SLAMTaskCreatedEvent is emitted when a SLAM task is created
type SLAMTaskCreatedEvent struct {
	TaskID     string    `json:"taskId"`
	PackageID  string    `json:"packageId"`
	OrderID    string    `json:"orderId"`
	ShipmentID string    `json:"shipmentId"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (e *SLAMTaskCreatedEvent) EventType() string     { return "slam.task.created" }
func (e *SLAMTaskCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// SLAMScanCompletedEvent is emitted when scanning is complete
type SLAMScanCompletedEvent struct {
	TaskID    string    `json:"taskId"`
	PackageID string    `json:"packageId"`
	ScannedAt time.Time `json:"scannedAt"`
}

func (e *SLAMScanCompletedEvent) EventType() string     { return "slam.scan.completed" }
func (e *SLAMScanCompletedEvent) OccurredAt() time.Time { return e.ScannedAt }

// SLAMScanFailedEvent is emitted when scanning fails
type SLAMScanFailedEvent struct {
	TaskID          string    `json:"taskId"`
	ExpectedBarcode string    `json:"expectedBarcode"`
	ScannedBarcode  string    `json:"scannedBarcode"`
	FailedAt        time.Time `json:"failedAt"`
}

func (e *SLAMScanFailedEvent) EventType() string     { return "slam.scan.failed" }
func (e *SLAMScanFailedEvent) OccurredAt() time.Time { return e.FailedAt }

// SLAMWeightVerifiedEvent is emitted when weight is verified
type SLAMWeightVerifiedEvent struct {
	TaskID          string    `json:"taskId"`
	PackageID       string    `json:"packageId"`
	ExpectedWeight  float64   `json:"expectedWeight"`
	ActualWeight    float64   `json:"actualWeight"`
	VariancePercent float64   `json:"variancePercent"`
	Passed          bool      `json:"passed"`
	VerifiedAt      time.Time `json:"verifiedAt"`
}

func (e *SLAMWeightVerifiedEvent) EventType() string     { return "slam.weight.verified" }
func (e *SLAMWeightVerifiedEvent) OccurredAt() time.Time { return e.VerifiedAt }

// SLAMLabelPrintedEvent is emitted when a label is printed
type SLAMLabelPrintedEvent struct {
	TaskID         string    `json:"taskId"`
	PackageID      string    `json:"packageId"`
	TrackingNumber string    `json:"trackingNumber"`
	PrinterID      string    `json:"printerId"`
	PrintedAt      time.Time `json:"printedAt"`
}

func (e *SLAMLabelPrintedEvent) EventType() string     { return "slam.label.printed" }
func (e *SLAMLabelPrintedEvent) OccurredAt() time.Time { return e.PrintedAt }

// SLAMLabelAppliedEvent is emitted when a label is applied
type SLAMLabelAppliedEvent struct {
	TaskID    string    `json:"taskId"`
	PackageID string    `json:"packageId"`
	AppliedBy string    `json:"appliedBy"`
	AppliedAt time.Time `json:"appliedAt"`
}

func (e *SLAMLabelAppliedEvent) EventType() string     { return "slam.label.applied" }
func (e *SLAMLabelAppliedEvent) OccurredAt() time.Time { return e.AppliedAt }

// SLAMManifestAssignedEvent is emitted when assigned to manifest
type SLAMManifestAssignedEvent struct {
	TaskID         string    `json:"taskId"`
	PackageID      string    `json:"packageId"`
	ManifestID     string    `json:"manifestId"`
	CarrierCode    string    `json:"carrierCode"`
	TrackingNumber string    `json:"trackingNumber"`
	AssignedAt     time.Time `json:"assignedAt"`
}

func (e *SLAMManifestAssignedEvent) EventType() string     { return "slam.manifest.assigned" }
func (e *SLAMManifestAssignedEvent) OccurredAt() time.Time { return e.AssignedAt }

// SLAMCompletedEvent is emitted when SLAM is complete
type SLAMCompletedEvent struct {
	TaskID         string    `json:"taskId"`
	PackageID      string    `json:"packageId"`
	OrderID        string    `json:"orderId"`
	ShipmentID     string    `json:"shipmentId"`
	TrackingNumber string    `json:"trackingNumber"`
	ManifestID     string    `json:"manifestId"`
	CompletedAt    time.Time `json:"completedAt"`
}

func (e *SLAMCompletedEvent) EventType() string     { return "slam.completed" }
func (e *SLAMCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }
