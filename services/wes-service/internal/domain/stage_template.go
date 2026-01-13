package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ProcessPathType represents the type of process path
type ProcessPathType string

const (
	PathPickPack            ProcessPathType = "pick_pack"
	PathPickWallPack        ProcessPathType = "pick_wall_pack"
	PathPickConsolidatePack ProcessPathType = "pick_consolidate_pack"
)

// StageTemplate represents a configurable process path template
type StageTemplate struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	TemplateID        string             `bson:"templateId" json:"templateId"`
	TenantID    string `bson:"tenantId" json:"tenantId"`
	FacilityID  string `bson:"facilityId" json:"facilityId"`
	WarehouseID string `bson:"warehouseId" json:"warehouseId"`
	PathType          ProcessPathType    `bson:"pathType" json:"pathType"`
	Name              string             `bson:"name" json:"name"`
	Description       string             `bson:"description" json:"description"`
	Stages            []StageDefinition  `bson:"stages" json:"stages"`
	SelectionCriteria SelectionCriteria  `bson:"selectionCriteria" json:"selectionCriteria"`
	IsDefault         bool               `bson:"isDefault" json:"isDefault"`
	Active            bool               `bson:"active" json:"active"`
	CreatedAt         time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// SelectionCriteria defines when a template should be selected
type SelectionCriteria struct {
	MinItems          *int     `bson:"minItems,omitempty" json:"minItems,omitempty"`
	MaxItems          *int     `bson:"maxItems,omitempty" json:"maxItems,omitempty"`
	RequiresMultiZone bool     `bson:"requiresMultiZone" json:"requiresMultiZone"`
	OrderTypes        []string `bson:"orderTypes,omitempty" json:"orderTypes,omitempty"`
	Priority          int      `bson:"priority" json:"priority"`
}

// NewStageTemplate creates a new stage template
func NewStageTemplate(templateID string, pathType ProcessPathType, name, description string, stages []StageDefinition, criteria SelectionCriteria) *StageTemplate {
	now := time.Now()
	return &StageTemplate{
		TemplateID:        templateID,
		PathType:          pathType,
		Name:              name,
		Description:       description,
		Stages:            stages,
		SelectionCriteria: criteria,
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// SetDefault marks this template as the default
func (t *StageTemplate) SetDefault() {
	t.IsDefault = true
	t.UpdatedAt = time.Now()
}

// Deactivate deactivates the template
func (t *StageTemplate) Deactivate() {
	t.Active = false
	t.UpdatedAt = time.Now()
}

// Matches checks if this template matches the given criteria
func (t *StageTemplate) Matches(itemCount int, multiZone bool, orderType string) bool {
	if !t.Active {
		return false
	}

	// Check item count
	if t.SelectionCriteria.MinItems != nil && itemCount < *t.SelectionCriteria.MinItems {
		return false
	}
	if t.SelectionCriteria.MaxItems != nil && itemCount > *t.SelectionCriteria.MaxItems {
		return false
	}

	// Check multi-zone requirement
	if t.SelectionCriteria.RequiresMultiZone && !multiZone {
		return false
	}

	// Check order type if specified
	if len(t.SelectionCriteria.OrderTypes) > 0 && orderType != "" {
		found := false
		for _, ot := range t.SelectionCriteria.OrderTypes {
			if ot == orderType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// GetStageByType returns the stage definition for the given type
func (t *StageTemplate) GetStageByType(stageType StageType) *StageDefinition {
	for i := range t.Stages {
		if t.Stages[i].StageType == stageType {
			return &t.Stages[i]
		}
	}
	return nil
}

// HasStage checks if the template has a specific stage type
func (t *StageTemplate) HasStage(stageType StageType) bool {
	return t.GetStageByType(stageType) != nil
}

// DefaultPickPackTemplate returns the default pick-pack template
func DefaultPickPackTemplate() *StageTemplate {
	maxItems := 3
	return NewStageTemplate(
		"tpl-pick-pack",
		PathPickPack,
		"Direct Pick to Pack",
		"Simple flow for small orders - picker brings items directly to packer",
		[]StageDefinition{
			{Order: 1, StageType: StagePicking, TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: StagePacking, TaskType: "packing", Required: true, TimeoutMins: 15},
		},
		SelectionCriteria{MaxItems: &maxItems, Priority: 1},
	)
}

// DefaultPickWallPackTemplate returns the default pick-wall-pack template
func DefaultPickWallPackTemplate() *StageTemplate {
	minItems := 4
	maxItems := 20
	return NewStageTemplate(
		"tpl-pick-wall-pack",
		PathPickWallPack,
		"Pick Wall Pack",
		"Flow with put-wall sorting for medium orders - walliner sorts picked items before packing",
		[]StageDefinition{
			{Order: 1, StageType: StagePicking, TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: StageWalling, TaskType: "walling", Required: true, TimeoutMins: 10, Config: StageConfig{RequiresPutWall: true}},
			{Order: 3, StageType: StagePacking, TaskType: "packing", Required: true, TimeoutMins: 15},
		},
		SelectionCriteria{MinItems: &minItems, MaxItems: &maxItems, Priority: 2},
	)
}

// DefaultPickConsolidatePackTemplate returns the default pick-consolidate-pack template
func DefaultPickConsolidatePackTemplate() *StageTemplate {
	return NewStageTemplate(
		"tpl-pick-consolidate-pack",
		PathPickConsolidatePack,
		"Multi-Zone Consolidation",
		"Flow for orders spanning multiple zones - consolidation step merges picks before packing",
		[]StageDefinition{
			{Order: 1, StageType: StagePicking, TaskType: "picking", Required: true, TimeoutMins: 30},
			{Order: 2, StageType: StageConsolidation, TaskType: "consolidation", Required: true, TimeoutMins: 20},
			{Order: 3, StageType: StagePacking, TaskType: "packing", Required: true, TimeoutMins: 15},
		},
		SelectionCriteria{RequiresMultiZone: true, Priority: 3},
	)
}
