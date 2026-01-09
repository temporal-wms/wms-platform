# Operational Runbooks - WMS Temporal Workflows

## Table of Contents
1. [Stuck Workflow Recovery](#stuck-workflow-recovery)
2. [Failed Workflow Investigation](#failed-workflow-investigation)
3. [High Workflow Latency](#high-workflow-latency)
4. [Worker Pod Issues](#worker-pod-issues)
5. [Database Connection Problems](#database-connection-problems)
6. [Deployment Rollback](#deployment-rollback)
7. [Workflow History Size Limit](#workflow-history-size-limit)
8. [Signal Loss Investigation](#signal-loss-investigation)
9. [Activity Timeout Investigation](#activity-timeout-investigation)
10. [Emergency Workflow Termination](#emergency-workflow-termination)

---

## 1. Stuck Workflow Recovery

### Symptoms
- Workflow appears "hung" in Temporal UI
- No recent activity or state changes
- Customer orders not progressing

### Diagnosis

```bash
# 1. Check workflow status
temporal workflow show --workflow-id order-fulfillment-ORD-12345

# 2. Check if workflow is waiting for signal
temporal workflow show --workflow-id order-fulfillment-ORD-12345 \
    | grep -A5 "PendingActivities\|SignalRequested"

# 3. Query workflow status (if query handler exists)
temporal workflow query --workflow-id order-fulfillment-ORD-12345 \
    --query-type getStatus

# 4. Check worker logs for this workflow
kubectl logs -l app=orchestrator-worker --tail=500 \
    | grep "ORD-12345"
```

### Resolution Options

#### Option A: Send Missing Signal

If workflow is waiting for a signal that never arrived:

```bash
# Example: Send waveAssigned signal
temporal workflow signal --workflow-id planning-ORD-12345 \
    --name waveAssigned \
    --input '{"waveId":"WAVE-001","scheduledStart":"2026-01-04T10:00:00Z"}'
```

#### Option B: Terminate and Restart

If workflow is truly stuck:

```bash
# 1. Terminate stuck workflow
temporal workflow terminate --workflow-id order-fulfillment-ORD-12345 \
    --reason "Stuck workflow - manual restart required"

# 2. Restart workflow with same inputs
# (Retrieve original input from workflow history first)
temporal workflow show --workflow-id order-fulfillment-ORD-12345 \
    | grep -A20 "WorkflowExecutionStarted" \
    | grep "Input"

# 3. Start new workflow with original input
temporal workflow start --task-queue orchestrator-queue \
    --type OrderFulfillmentWorkflow \
    --workflow-id order-fulfillment-ORD-12345-retry \
    --input '{...original input...}'
```

#### Option C: Cancel Gracefully

If workflow supports cancellation:

```bash
temporal workflow cancel --workflow-id order-fulfillment-ORD-12345 \
    --reason "Customer requested cancellation"
```

---

## 2. Failed Workflow Investigation

### Symptoms
- Workflow status shows "Failed"
- Customer orders not completing
- Error notifications

### Investigation Steps

```bash
# 1. Get failure details
temporal workflow show --workflow-id order-fulfillment-ORD-12345 \
    | grep -A10 "Failure"

# 2. Check recent activity failures
temporal workflow show --workflow-id order-fulfillment-ORD-12345 \
    | grep -B5 -A10 "ActivityTaskFailed"

# 3. Check worker logs around failure time
kubectl logs -l app=orchestrator-worker --since=1h \
    | grep -A10 "ORD-12345.*error\|ORD-12345.*failed"

# 4. Get full workflow history
temporal workflow show --workflow-id order-fulfillment-ORD-12345 \
    --output json > /tmp/failed-workflow-ORD-12345.json
```

### Common Failure Patterns

#### Pattern 1: Activity Timeout

**Symptoms**: `ActivityTaskTimedOut` in history

```bash
# Check which activity timed out
temporal workflow show --workflow-id order-fulfillment-ORD-12345 \
    | grep "ActivityTaskTimedOut" -B3

# Check if worker was overloaded
kubectl top pods -l app=orchestrator-worker
```

**Resolution**:
- Increase activity timeout in workflow code
- Scale worker pods if CPU/memory high
- Investigate slow external service

#### Pattern 2: Business Logic Failure

**Symptoms**: `ApplicationError` in history

```bash
# Get business error details
temporal workflow show --workflow-id order-fulfillment-ORD-12345 \
    | grep "ApplicationError" -A5
```

**Resolution**:
- Review business error type (e.g., `OrderValidationFailed`, `InsufficientInventory`)
- Fix root cause (invalid data, missing inventory, etc.)
- Restart workflow if issue resolved

#### Pattern 3: Panic/Crash

**Symptoms**: Worker logs show panic

```bash
# Find panic in logs
kubectl logs -l app=orchestrator-worker --tail=1000 \
    | grep -A20 "panic\|runtime error"
```

**Resolution**:
- Fix nil pointer or runtime error in code
- Deploy hotfix
- Workflow will automatically retry after worker restart

---

## 3. High Workflow Latency

### Symptoms
- Orders taking longer than usual to complete
- SLA breaches
- Customer complaints

### Investigation

```bash
# 1. Check workflow completion times
temporal workflow list --query 'WorkflowType="OrderFulfillmentWorkflow"' \
    --limit 50 \
    | grep -E "Completed|Failed" \
    | head -20

# 2. Check worker resource usage
kubectl top pods -l app=orchestrator-worker

# 3. Check activity latencies
kubectl logs -l app=orchestrator-worker --tail=500 \
    | grep "Activity completed" \
    | awk '{print $(NF-1), $NF}' \
    | sort -k2 -n \
    | tail -20

# 4. Check database performance
kubectl logs -l app=mongodb --tail=100 | grep "slow query"
```

### Resolution Options

#### Option A: Scale Workers

```bash
# Increase worker replicas
kubectl scale deployment/orchestrator-worker --replicas=5

# Monitor rollout
kubectl rollout status deployment/orchestrator-worker
```

#### Option B: Increase Worker Resources

```yaml
# Edit deployment
kubectl edit deployment orchestrator-worker

# Increase CPU/memory:
resources:
  requests:
    cpu: "2000m"
    memory: "2Gi"
  limits:
    cpu: "4000m"
    memory: "4Gi"
```

#### Option C: Optimize Slow Activities

```bash
# Find slowest activities
kubectl logs -l app=orchestrator-worker --since=1h \
    | grep "Activity.*duration" \
    | awk '{print $X, $Y}' \  # Extract activity name and duration
    | sort -k2 -n \
    | tail -10
```

---

## 4. Worker Pod Issues

### Symptoms
- Worker pods crashing
- CrashLoopBackOff status
- Workflows not progressing

### Diagnosis

```bash
# 1. Check pod status
kubectl get pods -l app=orchestrator-worker

# 2. Check recent events
kubectl describe pod <pod-name> | grep -A20 Events

# 3. Check logs for errors
kubectl logs <pod-name> --tail=100

# 4. Check previous container logs (if crashed)
kubectl logs <pod-name> --previous
```

### Common Issues

#### Issue 1: Out of Memory

**Symptoms**: Pod status `OOMKilled`

```bash
# Check memory usage
kubectl top pod <pod-name>

# Resolution
kubectl edit deployment orchestrator-worker
# Increase memory limits
```

#### Issue 2: Cannot Connect to Temporal

**Symptoms**: Logs show "connection refused" or "dial tcp" errors

```bash
# Check Temporal server pods
kubectl get pods -n temporal

# Check network connectivity
kubectl exec -it <worker-pod> -- wget -O- temporal-frontend:7233/health

# Resolution: Check service endpoints
kubectl get svc -n temporal
```

#### Issue 3: MongoDB Connection Failure

**Symptoms**: Logs show "failed to connect to MongoDB"

```bash
# Check MongoDB pods
kubectl get pods -l app=mongodb

# Test connection from worker
kubectl exec -it <worker-pod> -- nc -zv mongodb 27017

# Check credentials
kubectl get secret mongodb-credentials -o yaml
```

---

## 5. Database Connection Problems

### Symptoms
- Activities failing with database errors
- "connection pool exhausted" errors
- Slow query performance

### Investigation

```bash
# 1. Check MongoDB pod health
kubectl get pods -l app=mongodb
kubectl logs -l app=mongodb --tail=100

# 2. Check connection pool metrics (if exposed)
kubectl logs -l app=orchestrator-worker \
    | grep "connection pool"

# 3. Check for long-running queries
kubectl exec -it <mongodb-pod> -- mongo \
    --eval "db.currentOp({'secs_running': {\$gte: 10}})"
```

### Resolution

```bash
# 1. Restart MongoDB if needed
kubectl delete pod <mongodb-pod-name>
kubectl wait --for=condition=ready pod -l app=mongodb

# 2. Increase connection pool size (in code)
# Edit mongo client config, rebuild, redeploy

# 3. Add indexes for slow queries
kubectl exec -it <mongodb-pod> -- mongo wes_db \
    --eval "db.task_routes.createIndex({status: 1, createdAt: -1})"
```

---

## 6. Deployment Rollback

### When to Rollback
- Increased error rate after deployment
- Worker pods crashingnew code
- Workflow replay failures detected

### Rollback Procedure

```bash
# 1. Check current rollout status
kubectl rollout status deployment/orchestrator-worker

# 2. Rollback to previous version
kubectl rollout undo deployment/orchestrator-worker

# 3. Verify rollback succeeded
kubectl rollout status deployment/orchestrator-worker
kubectl get pods -l app=orchestrator-worker

# 4. Check workflows resume normally
temporal workflow list --query 'ExecutionStatus="Running"' --limit 10

# 5. Monitor error rate
kubectl logs -l app=orchestrator-worker --since=5m \
    | grep -i "error\|fail" \
    | wc -l
```

### Post-Rollback

```bash
# 1. Investigate root cause
kubectl logs <failed-pod-name> --previous > /tmp/failed-deployment-logs.txt

# 2. Test fix in staging environment

# 3. Redeploy with fix when ready
```

---

## 7. Workflow History Size Limit

### Symptoms
- Workflows failing with "history size limit exceeded"
- Long-running workflows suddenly failing
- Error: "exceeded 50000 events"

### Diagnosis

```bash
# Check history size for specific workflow
temporal workflow show --workflow-id reprocessing-run-1234567 \
    | grep "Total Events"

# Find workflows approaching limit
temporal workflow list --query 'WorkflowType="ReprocessingOrchestrationWorkflow"' \
    | while read wfid; do
        events=$(temporal workflow show --workflow-id $wfid | grep "Total Events" | awk '{print $3}')
        if [ "$events" -gt 40000 ]; then
            echo "$wfid: $events events (WARNING)"
        fi
    done
```

### Resolution

#### Immediate Fix: Terminate and Restart

```bash
# 1. Terminate workflow approaching limit
temporal workflow terminate --workflow-id reprocessing-run-1234567 \
    --reason "History size limit - manual restart with ContinueAsNew"

# 2. Verify ContinueAsNew is implemented (check code)
# See: /orchestrator/internal/workflows/reprocessing.go

# 3. Restart workflow - it will use ContinueAsNew automatically
```

#### Long-term Fix: Implement ContinueAsNew

Already implemented for ReprocessingOrchestrationWorkflow (batch limit: 1000 workflows).

For other workflows, add ContinueAsNew:
```go
if workflowCompletionPercent > 80 {
    return workflow.NewContinueAsNewError(ctx, MyWorkflow, input)
}
```

---

## 8. Signal Loss Investigation

### Symptoms
- Workflow stuck waiting for signal
- Signal sent but workflow didn't receive it
- PlanningWorkflow waiting for waveAssigned

### Investigation

```bash
# 1. Check if signal was sent
temporal workflow show --workflow-id planning-ORD-12345 \
    | grep "SignalReceived\|WorkflowExecutionSignaled"

# 2. Check workflow is listening for signal
temporal workflow show --workflow-id planning-ORD-12345 \
    | grep -A5 "MarkerRecorded" \
    | grep "signal"

# 3. Check sender logs
kubectl logs -l app=wes-service --tail=500 \
    | grep "Sending signal.*waveAssigned"
```

### Common Causes

1. **Wrong Workflow ID**: Signal sent to incorrect workflow
2. **Signal Name Mismatch**: Sent "waveAssign" instead of "waveAssigned"
3. **Workflow Completed Before Signal**: Race condition

### Resolution

```bash
# Resend signal with correct parameters
temporal workflow signal --workflow-id planning-ORD-12345 \
    --name waveAssigned \
    --input '{"waveId":"WAVE-123","scheduledStart":"2026-01-04T10:00:00Z"}'
```

---

## 9. Activity Timeout Investigation

### Symptoms
- Activities timing out frequently
- "ActivityTaskTimedOut" in workflow history
- External service calls failing

### Investigation

```bash
# 1. Identify which activity is timing out
temporal workflow show --workflow-id order-fulfillment-ORD-12345 \
    | grep -B5 "ActivityTaskTimedOut"

# 2. Check activity execution time
kubectl logs -l app=orchestrator-worker \
    | grep "CreatePickTask.*duration"

# 3. Check if external service is slow
kubectl exec -it <worker-pod> -- time curl http://picking-service/health
```

### Resolution Options

#### Option A: Increase Timeout

```go
// In workflow code
ao := workflow.ActivityOptions{
    StartToCloseTimeout: 5 * time.Minute,  // Increase from 2 min
    HeartbeatTimeout:    30 * time.Second,
}
```

#### Option B: Add Activity Heartbeats

Already implemented for:
- ConsolidateItems (consolidation_activities.go:76)
- ConfirmInventoryPick (inventory_activities.go:35)

For other long activities, add:
```go
activity.RecordHeartbeat(ctx, fmt.Sprintf("Processing %d/%d", i, total))
```

#### Option C: Optimize External Service Call

- Add caching
- Use batch APIs
- Increase external service resources

---

## 10. Emergency Workflow Termination

### When to Use
- Workflow causing system issues
- Infinite loop detected
- Need to stop all workflows for maintenance

### Single Workflow Termination

```bash
temporal workflow terminate --workflow-id order-fulfillment-ORD-12345 \
    --reason "Emergency stop: [ticket-123] - system overload"
```

### Bulk Termination

```bash
# Terminate all workflows of a specific type
temporal workflow list --query 'WorkflowType="ReprocessingOrchestrationWorkflow"' \
    | awk '{print $1}' \
    | while read wfid; do
        temporal workflow terminate --workflow-id $wfid \
            --reason "Emergency stop: maintenance window"
    done

# Terminate all running workflows (DANGEROUS - use with caution)
temporal workflow list --query 'ExecutionStatus="Running"' \
    | awk '{print $1}' \
    | head -100 \  # Limit to avoid accidents
    | while read wfid; do
        temporal workflow terminate --workflow-id $wfid \
            --reason "Emergency stop: [incident-456]"
    done
```

### Post-Termination

```bash
# 1. Verify terminations completed
temporal workflow list --query 'ExecutionStatus="Terminated"' --limit 20

# 2. Document which workflows were terminated
temporal workflow list --query 'ExecutionStatus="Terminated"' \
    | grep "Emergency stop" > /tmp/terminated-workflows.txt

# 3. Plan restart strategy for affected orders
```

---

## Quick Reference

| Situation | Command |
|-----------|---------|
| Check workflow status | `temporal workflow show --workflow-id <ID>` |
| List running workflows | `temporal workflow list --query 'ExecutionStatus="Running"'` |
| Send signal | `temporal workflow signal --workflow-id <ID> --name <signal> --input '{...}'` |
| Query workflow | `temporal workflow query --workflow-id <ID> --query-type getStatus` |
| Terminate workflow | `temporal workflow terminate --workflow-id <ID> --reason "..."` |
| Check worker logs | `kubectl logs -l app=orchestrator-worker --tail=100` |
| Scale workers | `kubectl scale deployment/orchestrator-worker --replicas=N` |
| Rollback deployment | `kubectl rollout undo deployment/orchestrator-worker` |

## Escalation

| Severity | Response Time | Escalation Path |
|----------|---------------|-----------------|
| P0 - System Down | Immediate | On-call engineer → Engineering Lead → VP Eng |
| P1 - Major Degradation | 15 minutes | On-call engineer → Team Lead |
| P2 - Partial Outage | 1 hour | Engineer → Team Lead (next business day) |
| P3 - Minor Issue | 4 hours | Engineer (handled during business hours) |

---

**Last Updated**: 2026-01-04
**Version**: 1.0
