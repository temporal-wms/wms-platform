package workflows

// Workflow versioning constants following Temporal best practices
// When making breaking changes to workflow logic, increment the version and use workflow.GetVersion()
// to maintain backward compatibility with running workflow instances.
//
// Usage example:
//   version := workflow.GetVersion(ctx, "my-change-id", workflow.DefaultVersion, MyWorkflowVersion)
//   if version == MyWorkflowVersion {
//       // New code path
//   } else {
//       // Legacy code path for running workflows
//   }

const (
	// OrderFulfillmentWorkflowVersion tracks version of the main saga workflow
	OrderFulfillmentWorkflowVersion = 1

	// OrchestratedPickingWorkflowVersion tracks version of the picking workflow
	OrchestratedPickingWorkflowVersion = 1

	// ConsolidationWorkflowVersion tracks version of the consolidation workflow
	ConsolidationWorkflowVersion = 1

	// PackingWorkflowVersion tracks version of the packing workflow
	PackingWorkflowVersion = 1

	// ShippingWorkflowVersion tracks version of the shipping workflow
	ShippingWorkflowVersion = 1

	// PlanningWorkflowVersion tracks version of the planning workflow
	PlanningWorkflowVersion = 1

	// ReprocessingOrchestrationWorkflowVersion tracks version of reprocessing workflow
	// Version 1: Added ContinueAsNew support for handling large batches
	ReprocessingOrchestrationWorkflowVersion = 1

	// GiftWrapWorkflowVersion tracks version of gift wrap workflow
	GiftWrapWorkflowVersion = 1

	// SortationWorkflowVersion tracks version of sortation workflow
	SortationWorkflowVersion = 1

	// InboundFulfillmentWorkflowVersion tracks version of inbound workflow
	InboundFulfillmentWorkflowVersion = 1

	// StockShortageWorkflowVersion tracks version of shortage handling workflow
	StockShortageWorkflowVersion = 1
)

// VersionChangeID constants for tracking specific changes within workflows
// These are used as the changeID parameter in workflow.GetVersion()
const (
	// OrderFulfillment change IDs
	OrderFulfillmentMultiRouteSupport = "multi-route-support"
	OrderFulfillmentUnitTracking      = "unit-level-tracking"

	// Picking change IDs
	PickingPartialSuccess = "partial-success-handling"

	// Consolidation change IDs
	ConsolidationMultiRoute = "multi-route-consolidation"

	// Reprocessing change IDs
	ReprocessingContinueAsNew = "continue-as-new-batching"
)
