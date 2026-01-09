---
sidebar_position: 6
---

# WES Execution Workflow

The WES (Warehouse Execution System) Execution Workflow orchestrates order execution through configurable process paths. It is the central execution engine started by the Orchestrator after wave assignment.

## Overview

WES execution follows one of three process paths based on order characteristics:

| Path Type | Stages | Criteria |
|-----------|--------|----------|
| `pick_pack` | Picking → Packing | 1-3 items |
| `pick_wall_pack` | Picking → Walling → Packing | 4-20 items |
| `pick_consolidate_pack` | Picking → Consolidation → Packing | Multi-zone orders |

## Complete Flow Diagram

```mermaid
sequenceDiagram
    autonumber
    participant O as Orchestrator
    participant WES as WES Workflow
    participant TRepo as Template Repository
    participant RRepo as Route Repository
    participant Pick as Picking Service
    participant Wall as Walling Service
    participant Pack as Packing Service

    O->>WES: Start WESExecutionWorkflow(orderId, items, multiZone)

    Note over WES: Phase 1: Plan Resolution
    WES->>TRepo: ResolveExecutionPlan(itemCount, multiZone)
    TRepo-->>WES: ExecutionPlan (template, stages)

    Note over WES: Phase 2: Route Creation
    WES->>RRepo: CreateTaskRoute(orderId, templateId)
    RRepo-->>WES: TaskRoute (routeId, stages)

    Note over WES: Phase 3: Stage Execution
    loop For each stage
        alt Picking Stage
            WES->>Pick: ExecutePickingStage(routeId)
            Pick->>Pick: Create pick task
            Pick->>Pick: Wait for pick completion
            Pick-->>WES: PickingResult

        else Walling Stage (pick_wall_pack only)
            WES->>Wall: Create walling task
            WES->>WES: Wait for wallingCompleted signal
            Note over WES: 15 minute timeout
            Wall-->>WES: Signal wallingCompleted
            WES-->>WES: WallingResult

        else Consolidation Stage (pick_consolidate_pack only)
            WES->>WES: ExecuteConsolidationStage(routeId)
            WES-->>WES: ConsolidationResult

        else Packing Stage
            WES->>Pack: ExecutePackingStage(routeId)
            Pack->>Pack: Create pack task
            Pack->>Pack: Wait for pack completion
            Pack-->>WES: PackingResult
        end

        WES->>RRepo: CompleteStage(routeId)
    end

    WES-->>O: WESExecutionResult
```

## pick_pack Path (Simple Orders)

For small orders (1-3 items), the flow is streamlined:

```mermaid
sequenceDiagram
    participant WES as WES Workflow
    participant Pick as Picking
    participant Pack as Packing

    Note over WES: Template: tpl-pick-pack
    WES->>Pick: Stage 1: Picking (30 min timeout)
    Pick-->>WES: Items picked

    WES->>Pack: Stage 2: Packing (15 min timeout)
    Pack-->>WES: Order packed

    Note over WES: Route completed
```

## pick_wall_pack Path (Medium Orders)

For medium orders (4-20 items), items go through a put-wall for sorting:

```mermaid
sequenceDiagram
    participant WES as WES Workflow
    participant Pick as Picking
    participant Wall as Walling
    participant Orch as Orchestrator
    participant Pack as Packing

    Note over WES: Template: tpl-pick-wall-pack
    WES->>Pick: Stage 1: Picking (30 min timeout)
    Pick-->>WES: Items picked to totes

    Note over WES: Stage 2: Walling
    WES->>Wall: Create walling task
    WES->>WES: await signal "wallingCompleted"

    Note over Wall: Walliner sorts items
    Wall->>Orch: POST /signals/walling-completed
    Orch->>WES: Signal wallingCompleted

    WES->>Pack: Stage 3: Packing (15 min timeout)
    Pack-->>WES: Order packed

    Note over WES: Route completed
```

## pick_consolidate_pack Path (Multi-Zone Orders)

For orders spanning multiple warehouse zones:

```mermaid
sequenceDiagram
    participant WES as WES Workflow
    participant Pick as Picking
    participant Consol as Consolidation
    participant Pack as Packing

    Note over WES: Template: tpl-pick-consolidate-pack
    WES->>Pick: Stage 1: Picking (30 min timeout)
    Note over Pick: Parallel picks from multiple zones
    Pick-->>WES: All zone picks complete

    WES->>Consol: Stage 2: Consolidation (20 min timeout)
    Note over Consol: Merge totes from zones
    Consol-->>WES: Items consolidated

    WES->>Pack: Stage 3: Packing (15 min timeout)
    Pack-->>WES: Order packed

    Note over WES: Route completed
```

## Signal Handling

The walling stage uses a signal-based pattern for external completion:

```mermaid
sequenceDiagram
    participant WES as WES Workflow
    participant Selector as Workflow Selector
    participant Timer as Timer

    WES->>Selector: Add signal channel "wallingCompleted"
    WES->>Selector: Add timer (15 minutes)

    alt Signal received
        Selector->>WES: wallingCompleted signal
        WES->>WES: Continue to packing
    else Timeout
        Selector->>WES: Timer fired
        WES->>WES: Mark stage failed
        WES->>WES: Compensation flow
    end
```

## Error Handling

```mermaid
sequenceDiagram
    participant WES as WES Workflow
    participant Stage as Current Stage
    participant RRepo as Route Repository

    WES->>Stage: Execute stage
    alt Success
        Stage-->>WES: Stage completed
        WES->>RRepo: CompleteStage()
        WES->>WES: Advance to next stage
    else Failure
        Stage-->>WES: Error
        WES->>RRepo: FailStage(error)
        WES->>WES: Return failure result
        Note over WES: Parent workflow handles compensation
    end
```

## Workflow Input/Output

### Input
```json
{
  "orderId": "ORD-12345",
  "waveId": "WAVE-001",
  "items": [
    {"sku": "SKU-001", "quantity": 2, "locationId": "LOC-A1", "zone": "A"},
    {"sku": "SKU-002", "quantity": 1, "locationId": "LOC-B2", "zone": "B"}
  ],
  "multiZone": true,
  "processPathId": "PATH-001",
  "specialHandling": ["fragile"]
}
```

### Output
```json
{
  "routeId": "RT-a1b2c3d4",
  "orderId": "ORD-12345",
  "status": "completed",
  "pathType": "pick_consolidate_pack",
  "stagesCompleted": 3,
  "totalStages": 3,
  "pickResult": {
    "stageType": "picking",
    "taskId": "PT-001",
    "success": true
  },
  "packingResult": {
    "stageType": "packing",
    "taskId": "PK-001",
    "success": true
  },
  "completedAt": 1705326000000
}
```

## Related Documentation

- [WES Service](/services/wes-service) - Service documentation
- [Orchestrator](/services/orchestrator) - Parent workflow
- [TaskRoute Aggregate](/domain-driven-design/aggregates/task-route) - Domain model
- [Walling Workflow](/architecture/sequence-diagrams/walling-workflow) - Walling details
