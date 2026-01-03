package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/wes-service/internal/domain"
)

func TestNewStageTemplate(t *testing.T) {
	stages := []domain.StageDefinition{
		{Order: 1, StageType: domain.StagePicking, TaskType: "picking", Required: true, TimeoutMins: 30},
		{Order: 2, StageType: domain.StagePacking, TaskType: "packing", Required: true, TimeoutMins: 15},
	}
	maxItems := 10
	criteria := domain.SelectionCriteria{MaxItems: &maxItems, Priority: 1}

	template := domain.NewStageTemplate("tpl-001", domain.PathPickPack, "Test Template", "Test description", stages, criteria)

	require.NotNil(t, template)
	assert.Equal(t, "tpl-001", template.TemplateID)
	assert.Equal(t, domain.PathPickPack, template.PathType)
	assert.Equal(t, "Test Template", template.Name)
	assert.Equal(t, "Test description", template.Description)
	assert.Len(t, template.Stages, 2)
	assert.True(t, template.Active)
	assert.False(t, template.IsDefault)
}

func TestStageTemplate_SetDefault(t *testing.T) {
	template := domain.DefaultPickPackTemplate()

	assert.False(t, template.IsDefault)

	template.SetDefault()

	assert.True(t, template.IsDefault)
}

func TestStageTemplate_Deactivate(t *testing.T) {
	template := domain.DefaultPickPackTemplate()

	assert.True(t, template.Active)

	template.Deactivate()

	assert.False(t, template.Active)
}

func TestStageTemplate_Matches_ItemCount(t *testing.T) {
	minItems := 4
	maxItems := 10
	stages := []domain.StageDefinition{
		{Order: 1, StageType: domain.StagePicking, Required: true},
	}
	criteria := domain.SelectionCriteria{MinItems: &minItems, MaxItems: &maxItems, Priority: 1}
	template := domain.NewStageTemplate("tpl-test", domain.PathPickPack, "Test", "Test", stages, criteria)

	// Below minimum
	assert.False(t, template.Matches(3, false, ""))

	// At minimum
	assert.True(t, template.Matches(4, false, ""))

	// In range
	assert.True(t, template.Matches(7, false, ""))

	// At maximum
	assert.True(t, template.Matches(10, false, ""))

	// Above maximum
	assert.False(t, template.Matches(11, false, ""))
}

func TestStageTemplate_Matches_MultiZone(t *testing.T) {
	stages := []domain.StageDefinition{
		{Order: 1, StageType: domain.StagePicking, Required: true},
	}
	criteria := domain.SelectionCriteria{RequiresMultiZone: true, Priority: 1}
	template := domain.NewStageTemplate("tpl-test", domain.PathPickConsolidatePack, "Test", "Test", stages, criteria)

	// RequiresMultiZone but order is not multi-zone
	assert.False(t, template.Matches(5, false, ""))

	// RequiresMultiZone and order is multi-zone
	assert.True(t, template.Matches(5, true, ""))
}

func TestStageTemplate_Matches_OrderType(t *testing.T) {
	stages := []domain.StageDefinition{
		{Order: 1, StageType: domain.StagePicking, Required: true},
	}
	criteria := domain.SelectionCriteria{OrderTypes: []string{"express", "same_day"}, Priority: 1}
	template := domain.NewStageTemplate("tpl-test", domain.PathPickPack, "Test", "Test", stages, criteria)

	// Matching order type
	assert.True(t, template.Matches(5, false, "express"))
	assert.True(t, template.Matches(5, false, "same_day"))

	// Non-matching order type
	assert.False(t, template.Matches(5, false, "standard"))

	// Empty order type matches if template has types specified
	// (template requires specific types but order has none)
	assert.True(t, template.Matches(5, false, ""))
}

func TestStageTemplate_Matches_Inactive(t *testing.T) {
	template := domain.DefaultPickPackTemplate()

	// Active template should match
	assert.True(t, template.Matches(2, false, ""))

	// Deactivate
	template.Deactivate()

	// Inactive template should not match
	assert.False(t, template.Matches(2, false, ""))
}

func TestStageTemplate_GetStageByType(t *testing.T) {
	template := domain.DefaultPickWallPackTemplate()

	// Get existing stage
	pickingStage := template.GetStageByType(domain.StagePicking)
	require.NotNil(t, pickingStage)
	assert.Equal(t, domain.StagePicking, pickingStage.StageType)
	assert.Equal(t, 1, pickingStage.Order)

	wallingStage := template.GetStageByType(domain.StageWalling)
	require.NotNil(t, wallingStage)
	assert.Equal(t, domain.StageWalling, wallingStage.StageType)
	assert.Equal(t, 2, wallingStage.Order)

	// Non-existing stage
	consolidationStage := template.GetStageByType(domain.StageConsolidation)
	assert.Nil(t, consolidationStage)
}

func TestStageTemplate_HasStage(t *testing.T) {
	template := domain.DefaultPickWallPackTemplate()

	assert.True(t, template.HasStage(domain.StagePicking))
	assert.True(t, template.HasStage(domain.StageWalling))
	assert.True(t, template.HasStage(domain.StagePacking))
	assert.False(t, template.HasStage(domain.StageConsolidation))
}

func TestDefaultPickPackTemplate(t *testing.T) {
	template := domain.DefaultPickPackTemplate()

	assert.Equal(t, "tpl-pick-pack", template.TemplateID)
	assert.Equal(t, domain.PathPickPack, template.PathType)
	assert.Len(t, template.Stages, 2)
	assert.True(t, template.Active)

	// Check stages
	assert.True(t, template.HasStage(domain.StagePicking))
	assert.True(t, template.HasStage(domain.StagePacking))
	assert.False(t, template.HasStage(domain.StageWalling))

	// Should match small orders (max 3 items)
	assert.True(t, template.Matches(3, false, ""))
	assert.False(t, template.Matches(4, false, ""))
}

func TestDefaultPickWallPackTemplate(t *testing.T) {
	template := domain.DefaultPickWallPackTemplate()

	assert.Equal(t, "tpl-pick-wall-pack", template.TemplateID)
	assert.Equal(t, domain.PathPickWallPack, template.PathType)
	assert.Len(t, template.Stages, 3)
	assert.True(t, template.Active)

	// Check stages
	assert.True(t, template.HasStage(domain.StagePicking))
	assert.True(t, template.HasStage(domain.StageWalling))
	assert.True(t, template.HasStage(domain.StagePacking))
	assert.False(t, template.HasStage(domain.StageConsolidation))

	// Check walling stage has put wall config
	wallingStage := template.GetStageByType(domain.StageWalling)
	require.NotNil(t, wallingStage)
	assert.True(t, wallingStage.Config.RequiresPutWall)

	// Should match medium orders (4-20 items)
	assert.False(t, template.Matches(3, false, ""))
	assert.True(t, template.Matches(4, false, ""))
	assert.True(t, template.Matches(10, false, ""))
	assert.True(t, template.Matches(20, false, ""))
	assert.False(t, template.Matches(21, false, ""))
}

func TestDefaultPickConsolidatePackTemplate(t *testing.T) {
	template := domain.DefaultPickConsolidatePackTemplate()

	assert.Equal(t, "tpl-pick-consolidate-pack", template.TemplateID)
	assert.Equal(t, domain.PathPickConsolidatePack, template.PathType)
	assert.Len(t, template.Stages, 3)
	assert.True(t, template.Active)

	// Check stages
	assert.True(t, template.HasStage(domain.StagePicking))
	assert.True(t, template.HasStage(domain.StageConsolidation))
	assert.True(t, template.HasStage(domain.StagePacking))
	assert.False(t, template.HasStage(domain.StageWalling))

	// Should only match multi-zone orders
	assert.False(t, template.Matches(5, false, ""))
	assert.True(t, template.Matches(5, true, ""))
	assert.True(t, template.Matches(50, true, ""))
}
