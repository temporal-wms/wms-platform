# Workflow Replay Test Data

This directory contains workflow history JSON files used for replay testing to verify workflow determinism.

## Purpose

Replay tests help ensure that workflow code changes don't break running workflows by:
- Verifying new code paths are compatible with existing workflow executions
- Catching non-determinism errors before production deployment
- Validating that `workflow.GetVersion()` is properly used for breaking changes

## How to Add History Files

### Using Temporal CLI

```bash
# Export a completed workflow history
temporal workflow show -w order-fulfillment-ORD-123 -o json > order_fulfillment_ord123.json

# Export with namespace
temporal workflow show -w order-fulfillment-ORD-123 -n production -o json > order_fulfillment_prod_ord123.json
```

### Using Temporal UI

1. Navigate to the workflow in Temporal Web UI
2. Click the "Download" button to export history as JSON
3. Save the file to this directory

### Naming Convention

Use the following naming pattern for history files:

- `order_fulfillment_<descriptor>.json` - Order fulfillment workflow histories
- `planning_<descriptor>.json` - Planning workflow histories
- `reprocessing_<descriptor>.json` - Reprocessing workflow histories

Examples:
- `order_fulfillment_success_sameday.json` - Same-day order that completed successfully
- `order_fulfillment_failure_validation.json` - Order that failed validation
- `order_fulfillment_compensation_cancel.json` - Order that was cancelled with compensation

## Best Practices

1. **Export before major changes**: Before making significant workflow changes, export histories from production/staging
2. **Cover all paths**: Include histories for success, failure, and compensation scenarios
3. **Version coverage**: Include histories from different workflow versions
4. **Regular updates**: Periodically update histories to match current execution patterns
5. **CI/CD integration**: Run replay tests as part of deployment pipeline

## Running Replay Tests

```bash
# Run all replay tests
go test -v ./orchestrator/tests/workflows/... -run Replay

# Run specific workflow replay test
go test -v ./orchestrator/tests/workflows/... -run TestOrderFulfillmentWorkflow_Replay
```

## Troubleshooting

If replay tests fail with non-determinism errors:

1. Check for removed or reordered activities
2. Verify activity/workflow input changes are backward compatible
3. Ensure `workflow.GetVersion()` is used for breaking changes
4. Check for use of non-deterministic functions (time.Now(), rand, maps iteration)
