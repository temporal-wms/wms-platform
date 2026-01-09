package application

import (
	"context"
	"sort"

	"github.com/wms-platform/wes-service/internal/domain"
)

// TemplateSelector selects the appropriate stage template based on order characteristics
type TemplateSelector struct {
	templateRepo domain.StageTemplateRepository
}

// NewTemplateSelector creates a new template selector
func NewTemplateSelector(templateRepo domain.StageTemplateRepository) *TemplateSelector {
	return &TemplateSelector{
		templateRepo: templateRepo,
	}
}

// SelectTemplate selects the best matching template for the given criteria
func (s *TemplateSelector) SelectTemplate(ctx context.Context, itemCount int, multiZone bool, consolidationRequired bool, orderType string) (*domain.StageTemplate, error) {
	// Get all active templates
	templates, err := s.templateRepo.FindActive(ctx)
	if err != nil {
		return nil, err
	}

	if len(templates) == 0 {
		// Return default pick-pack template if no templates configured
		return domain.DefaultPickPackTemplate(), nil
	}

	// Filter matching templates
	var matchingTemplates []*domain.StageTemplate
	for _, t := range templates {
		if s.matches(t, itemCount, multiZone, consolidationRequired, orderType) {
			matchingTemplates = append(matchingTemplates, t)
		}
	}

	if len(matchingTemplates) == 0 {
		// Try to find the default template
		defaultTemplate, err := s.templateRepo.FindDefault(ctx)
		if err == nil && defaultTemplate != nil {
			return defaultTemplate, nil
		}
		// Fall back to built-in default
		return domain.DefaultPickPackTemplate(), nil
	}

	// Sort by priority (higher priority = preferred)
	sort.Slice(matchingTemplates, func(i, j int) bool {
		return matchingTemplates[i].SelectionCriteria.Priority > matchingTemplates[j].SelectionCriteria.Priority
	})

	return matchingTemplates[0], nil
}

// matches checks if a template matches the given criteria
func (s *TemplateSelector) matches(template *domain.StageTemplate, itemCount int, multiZone bool, consolidationRequired bool, orderType string) bool {
	criteria := template.SelectionCriteria

	// Check item count
	if criteria.MinItems != nil && itemCount < *criteria.MinItems {
		return false
	}
	if criteria.MaxItems != nil && itemCount > *criteria.MaxItems {
		return false
	}

	// If consolidation is required, the template must have a consolidation stage
	if consolidationRequired && !template.HasStage(domain.StageConsolidation) {
		return false
	}

	// If multi-zone is required by criteria, the order must be multi-zone
	if criteria.RequiresMultiZone && !multiZone {
		return false
	}

	// Check order type if specified
	if len(criteria.OrderTypes) > 0 && orderType != "" {
		found := false
		for _, ot := range criteria.OrderTypes {
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

// SelectTemplateForProcessPath selects a template based on process path result
func (s *TemplateSelector) SelectTemplateForProcessPath(ctx context.Context, processPath *ProcessPathResultDTO, itemCount int, multiZone bool) (*domain.StageTemplate, error) {
	if processPath == nil {
		return s.SelectTemplate(ctx, itemCount, multiZone, false, "")
	}

	// Determine if consolidation is required
	consolidationRequired := processPath.ConsolidationRequired

	// If multi-zone is required based on process path
	if containsRequirement(processPath.Requirements, "multi_item") {
		consolidationRequired = true
	}

	return s.SelectTemplate(ctx, itemCount, multiZone, consolidationRequired, "")
}

// containsRequirement checks if a requirement list contains a specific requirement
func containsRequirement(requirements []string, req string) bool {
	for _, r := range requirements {
		if r == req {
			return true
		}
	}
	return false
}
