# Workflow Versioning & Deployment Guide

## Overview

Temporal workflows are **deterministic** - they must replay identically to maintain consistency. Changing workflow code while instances are running can cause replay failures. This guide explains how to safely deploy workflow changes using versioning.

## Why Versioning Matters

**The Problem**: Temporal replays workflow history on every decision. If you change workflow logic and deploy, running workflows will fail when they replay with the new code.

```go
// Version 1 (deployed Monday)
func MyWorkflow(ctx workflow.Context) error {
    var result string
    workflow.ExecuteActivity(ctx, "Step1").Get(ctx, &result)
    // return here
}

// Version 2 (deployed Tuesday) - BREAKS running workflows!
func MyWorkflow(ctx workflow.Context) error {
    var result string
    workflow.ExecuteActivity(ctx, "Step1").Get(ctx, &result)
    workflow.ExecuteActivity(ctx, "Step2").Get(ctx, &result) // NEW step
    // return here
}
```

**The Solution**: Use `workflow.GetVersion()` to maintain backward compatibility.

## Versioning Infrastructure

### Version Constants

All workflow versions are defined in `/orchestrator/internal/workflows/versions.go`:

```go
const (
    OrderFulfillmentWorkflowVersion = 1
    OrchestratedPickingWorkflowVersion = 1
    ConsolidationWorkflowVersion = 1
    // ... etc
)
```

### Change ID Constants

Track specific changes within workflows:

```go
const (
    OrderFulfillmentMultiRouteSupport = "multi-route-support"
    PickingPartialSuccess = "partial-success-handling"
    ConsolidationMultiRoute = "multi-route-consolidation"
    ReprocessingContinueAsNew = "continue-as-new-batching"
)
```

## How to Version Workflows

### Step 1: Add Version Check

When making breaking changes, add a version check:

```go
func OrderFulfillmentWorkflow(ctx workflow.Context, input Input) error {
    logger := workflow.GetLogger(ctx)

    // Get version - returns DefaultVersion for old workflows, new version for new ones
    version := workflow.GetVersion(ctx, "OrderFulfillmentWorkflow",
        workflow.DefaultVersion, OrderFulfillmentWorkflowVersion)

    logger.Info("Workflow version", "version", version)

    // ... rest of workflow logic
}
```

### Step 2: Add Conditional Logic for Breaking Changes

When changing workflow structure, check the version:

```go
// Example: Adding a new step
version := workflow.GetVersion(ctx, "add-consolidation",
    workflow.DefaultVersion, 2)

if version == workflow.DefaultVersion {
    // Old path - for workflows started before this change
    logger.Info("Using legacy path without consolidation")
    err = workflow.ExecuteActivity(ctx, "PackDirectly", input).Get(ctx, nil)
} else {
    // New path - for workflows started after this change
    logger.Info("Using new path with consolidation")
    err = workflow.ExecuteActivity(ctx, "Consolidate", input).Get(ctx, nil)
    if err == nil {
        err = workflow.ExecuteActivity(ctx, "Pack", input).Get(ctx, nil)
    }
}
```

### Step 3: Update Version Constant

Increment the version constant in `versions.go`:

```go
const (
    // Before
    OrderFulfillmentWorkflowVersion = 1

    // After
    OrderFulfillmentWorkflowVersion = 2  // Incremented
)
```

## Deployment Workflow

### Pre-Deployment Checklist

- [ ] Identify if change is breaking (adds/removes/reorders workflow steps)
- [ ] Add `workflow.GetVersion()` for breaking changes
- [ ] Increment version constant
- [ ] Add version branching logic
- [ ] Test both code paths (old and new versions)
- [ ] Document the change in workflow comments

### Safe Deployment Process

#### Option 1: Rolling Deployment (Recommended)

1. **Deploy new code** with versioning logic
   ```bash
   # Build and deploy new worker image
   docker build -t orchestrator:v1.2.0 .
   kubectl set image deployment/orchestrator-worker worker=orchestrator:v1.2.0
   ```

2. **Monitor running workflows**
   ```bash
   # Check for replay errors
   kubectl logs -l app=orchestrator-worker --tail=100 | grep -i "replay\|nondeterministic"
   ```

3. **Verify new workflows** use new version
   ```bash
   # Start a test workflow
   temporal workflow start --task-queue orchestrator-queue \
       --type OrderFulfillmentWorkflow \
       --input '{"orderId": "TEST-001"}'

   # Query version
   temporal workflow show --workflow-id order-fulfillment-TEST-001 | grep "version"
   ```

4. **Wait for old workflows to complete** (optional)
   - Old workflows will continue running with DefaultVersion
   - New workflows will use new version
   - Both paths coexist safely

#### Option 2: Gradual Migration

For critical changes, migrate gradually:

1. Deploy version with both code paths
2. Monitor for 24-48 hours
3. After all old workflows complete, remove legacy path
4. Deploy cleaned-up code

### Post-Deployment Verification

```bash
# 1. Check worker health
kubectl get pods -l app=orchestrator-worker
kubectl logs -l app=orchestrator-worker --tail=50

# 2. Verify no replay failures
temporal workflow list --query 'ExecutionStatus="Failed"' | head -20

# 3. Check version distribution
temporal workflow list --query 'WorkflowType="OrderFulfillmentWorkflow"' \
    | xargs -I {} temporal workflow show --workflow-id {}  \
    | grep "Workflow version"
```

## Common Scenarios

### Scenario 1: Adding a New Activity

**Change**: Add email notification after packing

**Solution**:

```go
// Get version for this specific change
notificationVersion := workflow.GetVersion(ctx, "add-email-notification",
    workflow.DefaultVersion, 2)

// Packing (exists in both versions)
err = workflow.ExecuteActivity(ctx, "Pack", input).Get(ctx, &packResult)

// Only send email in version 2+
if notificationVersion >= 2 {
    workflow.ExecuteActivity(ctx, "SendPackedEmail", packResult).Get(ctx, nil)
}
```

### Scenario 2: Changing Activity Parameters

**Change**: Add `priority` field to PickingActivity

**Solution**:

```go
pickVersion := workflow.GetVersion(ctx, "picking-with-priority",
    workflow.DefaultVersion, 2)

if pickVersion == workflow.DefaultVersion {
    // Old signature
    input := PickingInput{OrderID: orderID, Items: items}
    err = workflow.ExecuteActivity(ctx, "CreatePickTask", input).Get(ctx, nil)
} else {
    // New signature with priority
    input := PickingInputV2{OrderID: orderID, Items: items, Priority: priority}
    err = workflow.ExecuteActivity(ctx, "CreatePickTask", input).Get(ctx, nil)
}
```

### Scenario 3: Removing a Step

**Change**: Remove gift wrap check (no longer needed)

**Solution**:

```go
giftWrapVersion := workflow.GetVersion(ctx, "remove-giftwrap-check",
    workflow.DefaultVersion, 2)

if giftWrapVersion == workflow.DefaultVersion {
    // Old workflows still need this check
    var needsGiftWrap bool
    workflow.ExecuteActivity(ctx, "CheckGiftWrap", orderID).Get(ctx, &needsGiftWrap)
    if needsGiftWrap {
        workflow.ExecuteActivity(ctx, "ApplyGiftWrap", orderID).Get(ctx, nil)
    }
}
// else: New workflows skip gift wrap entirely
```

## Non-Breaking Changes (No Versioning Required)

These changes are safe without versioning:

✅ **Activity implementation changes** - As long as input/output types don't change
✅ **Logging changes** - Adding/removing logs
✅ **Adding workflow queries** - Queries don't affect determinism
✅ **Retry policy tweaks** - Activity retry policies can change
✅ **Timeout adjustments** - Can be changed
✅ **Variable renaming** - Internal refactoring
✅ **Adding signals** - New signal handlers are safe

## Breaking Changes (Require Versioning)

These changes MUST use versioning:

❌ **Adding/removing activities**
❌ **Reordering activities**
❌ **Changing activity input/output types**
❌ **Adding/removing child workflows**
❌ **Changing control flow** (if/else, loops)
❌ **Changing signals consumed** (expecting different signals)

## Version Cleanup

After all old workflows complete (check with `temporal workflow list`):

1. Remove legacy code paths
2. Set `DefaultVersion` as the minimum version
3. Deploy cleaned-up code

```go
// Before cleanup
version := workflow.GetVersion(ctx, "my-change", workflow.DefaultVersion, 2)
if version == workflow.DefaultVersion {
    // Old path
} else {
    // New path
}

// After cleanup (all old workflows completed)
version := workflow.GetVersion(ctx, "my-change", 2, 2)  // Min version is now 2
// Only new path remains - legacy code removed
```

## Rollback Procedures

### Emergency Rollback

If new version causes issues:

```bash
# 1. Rollback to previous image
kubectl set image deployment/orchestrator-worker worker=orchestrator:v1.1.0

# 2. Verify rollback
kubectl rollout status deployment/orchestrator-worker

# 3. Check workflows resume normally
temporal workflow list --query 'ExecutionStatus="Running"' | head -10
```

**Important**: Rollback is safe because versioning allows both old and new code to run.

### Partial Rollback

If only specific workflows are affected:

1. Keep new deployment running
2. Manually terminate affected workflows
3. Restart with fixed code
4. Investigate root cause

## Testing Versioning

### Unit Tests

Test both version paths:

```go
func TestOrderFulfillment_OldVersion(t *testing.T) {
    env := testsuite.WorkflowTestSuite{}.NewTestWorkflowEnvironment()

    // Mock GetVersion to return DefaultVersion
    env.OnWorkflow(OrderFulfillmentWorkflow, mock.Anything).Return(nil)

    // Test old path...
}

func TestOrderFulfillment_NewVersion(t *testing.T) {
    env := testsuite.WorkflowTestSuite{}.NewTestWorkflowEnvironment()

    // Mock GetVersion to return new version
    env.OnWorkflow(OrderFulfillmentWorkflow, mock.Anything).Return(nil)

    // Test new path...
}
```

### Replay Tests

Verify old workflow histories replay with new code:

```go
func TestOrderFulfillment_ReplayOldHistory(t *testing.T) {
    // Load old workflow history from production
    history := loadHistoryFromFile("old_workflow_history.json")

    env := testsuite.WorkflowTestSuite{}.NewTestWorkflowEnvironment()
    env.RegisterWorkflow(OrderFulfillmentWorkflow)

    // Replay should succeed with new code
    err := env.ReplayWorkflowHistory(history)
    require.NoError(t, err, "Replay failed - versioning broken!")
}
```

## Monitoring Version Distribution

Track which versions are running:

```bash
# Custom metric in workflow
queryStatus.WorkflowVersion = version
queryStatus.VersionChangeID = "add-consolidation"

# Query all running workflows for version info
temporal workflow list --query 'WorkflowType="OrderFulfillmentWorkflow"' \
    | while read wfid; do
        temporal workflow query --workflow-id $wfid --query-type getStatus \
        | jq '.workflowVersion'
    done \
    | sort | uniq -c
```

## Quick Reference

| Situation | Action | Versioning Required? |
|-----------|--------|---------------------|
| Add new activity | Add `GetVersion()` check | ✅ YES |
| Change activity params | Add `GetVersion()` check | ✅ YES |
| Add signal handler | Just add it | ❌ NO |
| Add query handler | Just add it | ❌ NO |
| Change activity implementation | Deploy directly | ❌ NO |
| Adjust timeouts | Deploy directly | ❌ NO |
| Add logging | Deploy directly | ❌ NO |
| Reorder activities | Add `GetVersion()` check | ✅ YES |
| Remove activity | Add `GetVersion()` check | ✅ YES |

## Support

- **Temporal Docs**: https://docs.temporal.io/workflows#versions
- **Version Constants**: `/orchestrator/internal/workflows/versions.go`
- **Retry Policies**: `/orchestrator/internal/workflows/retry_policies.go`
- **Error Classification**: `/orchestrator/docs/ERROR_CLASSIFICATION_GUIDE.md`

---

**Last Updated**: 2026-01-04
**Versioning Infrastructure Version**: 1.0
