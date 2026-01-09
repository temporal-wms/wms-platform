package domain

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExceptionType represents the type of unit exception
type ExceptionType string

const (
	ExceptionTypeDamaged   ExceptionType = "damaged"
	ExceptionTypeMissing   ExceptionType = "missing"
	ExceptionTypeMisrouted ExceptionType = "misrouted"
	ExceptionTypeShortage  ExceptionType = "shortage"
	ExceptionTypeQuality   ExceptionType = "quality"
	ExceptionTypeOther     ExceptionType = "other"
)

// ExceptionStage represents where in the process the exception occurred
type ExceptionStage string

const (
	ExceptionStageReceiving     ExceptionStage = "receiving"
	ExceptionStagePicking       ExceptionStage = "picking"
	ExceptionStageConsolidation ExceptionStage = "consolidation"
	ExceptionStagePacking       ExceptionStage = "packing"
	ExceptionStageShipping      ExceptionStage = "shipping"
)

// UnitException represents an exception for a unit that failed processing
type UnitException struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ExceptionID   string             `bson:"exceptionId" json:"exceptionId"`
	UnitID        string             `bson:"unitId" json:"unitId"`
	OrderID       string             `bson:"orderId" json:"orderId"`
	SKU           string             `bson:"sku" json:"sku"`
	ExceptionType ExceptionType      `bson:"exceptionType" json:"exceptionType"`
	Stage         ExceptionStage     `bson:"stage" json:"stage"`
	Description   string             `bson:"description" json:"description"`
	StationID     string             `bson:"stationId,omitempty" json:"stationId,omitempty"`
	ReportedBy    string             `bson:"reportedBy" json:"reportedBy"`
	Resolution    string             `bson:"resolution,omitempty" json:"resolution,omitempty"`
	ResolvedBy    string             `bson:"resolvedBy,omitempty" json:"resolvedBy,omitempty"`
	ResolvedAt    *time.Time         `bson:"resolvedAt,omitempty" json:"resolvedAt,omitempty"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// NewUnitException creates a new unit exception
func NewUnitException(unitID, orderID, sku string, exceptionType ExceptionType, stage ExceptionStage, description, stationID, reportedBy string) *UnitException {
	now := time.Now()
	return &UnitException{
		ExceptionID:   uuid.New().String(),
		UnitID:        unitID,
		OrderID:       orderID,
		SKU:           sku,
		ExceptionType: exceptionType,
		Stage:         stage,
		Description:   description,
		StationID:     stationID,
		ReportedBy:    reportedBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// Resolve marks the exception as resolved
func (e *UnitException) Resolve(resolution, resolvedBy string) {
	now := time.Now()
	e.Resolution = resolution
	e.ResolvedBy = resolvedBy
	e.ResolvedAt = &now
	e.UpdatedAt = now
}

// IsResolved returns true if the exception has been resolved
func (e *UnitException) IsResolved() bool {
	return e.ResolvedAt != nil
}
