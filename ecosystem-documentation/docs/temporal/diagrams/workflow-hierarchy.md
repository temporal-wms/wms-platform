---
sidebar_position: 1
slug: /temporal/diagrams/workflow-hierarchy
---

# Workflow Hierarchy

Visual representation of parent-child workflow relationships in the WMS Platform.

## Complete Workflow Hierarchy

```mermaid
graph TD
    subgraph "Entry Points"
        OF[OrderFulfillmentWorkflow]
        IB[InboundFulfillmentWorkflow]
        RPO[ReprocessingOrchestrationWorkflow]
        CANCEL[OrderCancellationWorkflow]
    end

    subgraph "Orchestrator Workflows"
        PL[PlanningWorkflow]
        WES[WESExecutionWorkflow]
        SORT[SortationWorkflow]
        SHIP_O[ShippingWorkflow]
        GW[GiftWrapWorkflow]
        PICK_O[OrchestratedPickingWorkflow]
        CON_O[ConsolidationWorkflow]
        PACK_O[PackingWorkflow]
        SS[StockShortageWorkflow]
        RPB[ReprocessingBatchWorkflow]
    end

    subgraph "Service Workflows"
        PICK_S[PickingWorkflow<br/>picking-queue]
        CON_S[ConsolidationWorkflow<br/>consolidation-queue]
        PACK_S[PackingWorkflow<br/>packing-queue]
        SHIP_S[ShippingWorkflow<br/>shipping-queue]
        WES_S[WESExecutionWorkflow<br/>wes-execution-queue]
    end

    %% OrderFulfillment children
    OF -->|Stage 1| PL
    OF -->|Stage 2| WES
    OF -->|Stage 3| SORT
    OF -->|Stage 4| SHIP_O

    %% Planning children
    PL -.->|Optional| GW

    %% WES execution children
    WES -->|Per Stage| PICK_O
    WES -->|Per Stage| CON_O
    WES -->|Per Stage| PACK_O

    %% Cross-queue execution
    WES -.->|Cross-Queue| PICK_S
    WES -.->|Cross-Queue| CON_S
    WES -.->|Cross-Queue| PACK_S
    WES -.->|Cross-Queue| WES_S

    %% Shipping children
    SHIP_O -.->|Cross-Queue| SHIP_S

    %% Cancellation relationships
    CANCEL -->|Reverse| OF

    %% Reprocessing relationships
    RPO -->|Batch| RPB
    RPB -->|Retry| OF

    %% Shortage handling
    PICK_O -.->|On Shortage| SS

    %% Styling
    classDef entry fill:#e1f5fe,stroke:#01579b
    classDef orchestrator fill:#fff3e0,stroke:#e65100
    classDef service fill:#e8f5e9,stroke:#1b5e20

    class OF,IB,RPO,CANCEL entry
    class PL,WES,SORT,SHIP_O,GW,PICK_O,CON_O,PACK_O,SS,RPB orchestrator
    class PICK_S,CON_S,PACK_S,SHIP_S,WES_S service
```

---

## Orchestrator Task Queue

All orchestrator workflows run on the `orchestrator` task queue:

```mermaid
graph LR
    subgraph "orchestrator queue"
        OF[OrderFulfillment]
        PL[Planning]
        SORT[Sortation]
        GW[GiftWrap]
        SS[StockShortage]
        CANCEL[Cancellation]
        RPO[Reprocessing]
        IB[Inbound]
    end

    W1[Worker 1] --> OF
    W1 --> PL
    W2[Worker 2] --> SORT
    W2 --> GW
    W3[Worker 3] --> SS
    W3 --> CANCEL
    W3 --> RPO
    W3 --> IB
```

---

## Service Task Queues

Service workflows run on dedicated task queues for isolation:

```mermaid
graph TD
    subgraph "wes-execution-queue"
        WES[WESExecutionWorkflow]
    end

    subgraph "picking-queue"
        PICK[PickingWorkflow]
    end

    subgraph "consolidation-queue"
        CON[ConsolidationWorkflow]
    end

    subgraph "packing-queue"
        PACK[PackingWorkflow]
    end

    subgraph "shipping-queue"
        SHIP[ShippingWorkflow]
    end

    WES -->|Child| PICK
    WES -->|Child| CON
    WES -->|Child| PACK
    PACK -->|Next| SHIP
```

---

## Child Workflow Execution Pattern

```mermaid
sequenceDiagram
    participant P as Parent Workflow
    participant T as Temporal Server
    participant C as Child Workflow
    participant W as Child Worker

    P->>T: ExecuteChildWorkflow(options)
    Note right of P: TaskQueue: "target-queue"
    T->>W: Schedule Child Task
    W->>C: Start Child Workflow

    loop Child Execution
        C->>C: Execute Activities
        C->>T: Heartbeat
    end

    C->>T: Complete
    T->>P: Return Result
```

---

## Workflow Depth Levels

| Level | Workflows | Task Queue |
|-------|-----------|------------|
| 0 (Entry) | OrderFulfillment, Inbound, Reprocessing | orchestrator |
| 1 | Planning, WESExecution, Sortation, Shipping | orchestrator, wes-execution-queue |
| 2 | Picking, Consolidation, Packing, GiftWrap | Various service queues |
| 3 | Service-level workflows | Service-specific queues |

---

## Cross-Queue Communication

When parent workflows execute children on different queues:

```go
childOpts := workflow.ChildWorkflowOptions{
    TaskQueue:                "picking-queue",     // Different queue
    WorkflowExecutionTimeout: 4 * time.Hour,
    // No retry policy for child workflows
}
childCtx := workflow.WithChildOptions(ctx, childOpts)

err := workflow.ExecuteChildWorkflow(childCtx, "PickingWorkflow", input).Get(ctx, &result)
```

## Related Documentation

- [Task Queues](../task-queues) - Queue configuration details
- [Order Flow](./order-flow) - Complete order processing flow
- [Signal Flow](./signal-flow) - Signal timing between workflows
