---
sidebar_position: 10
---

# Reprocessing Workflow

This document describes the order reprocessing flow for handling failed workflow executions, including retry logic and dead letter queue management.

## Overview

The reprocessing workflow handles orders that have failed during fulfillment. It implements an automatic retry mechanism with exponential backoff and moves orders to a dead letter queue (DLQ) when retries are exhausted.

## Reprocessing Flow

```mermaid
sequenceDiagram
    autonumber
    participant RW as Reprocessing Workflow
    participant RA as Reprocessing Activities
    participant OS as Order Service
    participant T as Temporal
    participant DLQ as Dead Letter Queue

    rect rgb(240, 248, 255)
        Note over RW,DLQ: Phase 1: Query Failed Workflows
        RW->>RA: QueryFailedWorkflows
        RA->>OS: GET /reprocessing/eligible
        OS-->>RA: Failed orders list
        RA-->>RW: []FailedWorkflowInfo
    end

    rect rgb(255, 248, 240)
        Note over RW,DLQ: Phase 2: Process Each Failed Order
        loop For each failed workflow
            RW->>RA: ProcessFailedWorkflow
            RA->>OS: GET retry count

            alt retryCount < MaxRetries (5)
                RA->>OS: POST /reprocessing/{orderId}/retry-count
                RA->>OS: POST /reprocessing/{orderId}/reset
                RA->>T: ExecuteWorkflow (OrderFulfillment)
                T-->>RA: New workflow started
                RA-->>RW: {Restarted: true, NewWorkflowID}
            else retryCount >= MaxRetries
                RA->>OS: POST /reprocessing/{orderId}/dlq
                OS->>DLQ: Move to dead letter queue
                OS->>OS: Publish MovedToDLQEvent
                RA-->>RW: {MovedToDLQ: true}
            end
        end
    end

    rect rgb(240, 255, 240)
        Note over RW,DLQ: Phase 3: Manual Resolution (if DLQ)
        DLQ->>OS: PATCH /dead-letter-queue/{orderId}/resolve
        OS->>OS: Apply resolution (retry/cancel/escalate)
    end
```

## Retry Decision Logic

```mermaid
flowchart TD
    A[Failed Workflow] --> B{Retry Count?}
    B -->|< 5| C[Increment Retry Count]
    C --> D[Reset Order Status]
    D --> E[Start New Workflow]
    E --> F[Log Success]

    B -->|>= 5| G[Move to DLQ]
    G --> H[Publish MovedToDLQEvent]
    H --> I[Await Manual Resolution]

    I --> J{Resolution Type}
    J -->|manual_retry| K[Reset & Restart]
    J -->|cancelled| L[Cancel Order]
    J -->|escalated| M[Notify Management]
```

## Order State Transitions

```mermaid
stateDiagram-v2
    [*] --> Processing: Order in fulfillment
    Processing --> Failed: Workflow failure

    Failed --> RetryScheduled: Retry count < 5
    RetryScheduled --> Processing: Workflow restarted

    Failed --> InDLQ: Retry count >= 5
    InDLQ --> Processing: Manual retry
    InDLQ --> Cancelled: Manual cancel
    InDLQ --> Escalated: Manual escalate

    Processing --> Completed: Success
    Cancelled --> [*]
    Escalated --> [*]
    Completed --> [*]
```

## Failure Metrics

```mermaid
graph LR
    subgraph "Metrics Recorded"
        A[RecordRetrySuccess] --> M[Prometheus]
        B[RecordRetryFailure] --> M
        C[RecordMovedToDLQ] --> M
        D[RecordWorkflowRetry] --> M
    end
```

## DLQ Resolution Types

| Resolution | Action | Next State |
|------------|--------|------------|
| `manual_retry` | Reset order, start new workflow | Processing |
| `cancelled` | Cancel order, notify customer | Cancelled |
| `escalated` | Create ticket, notify management | Escalated |

## API Endpoints Used

### Reprocessing Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/reprocessing/eligible` | Query failed workflows |
| GET | `/reprocessing/orders/{id}/retry-count` | Get retry metadata |
| POST | `/reprocessing/orders/{id}/retry-count` | Increment retry count |
| POST | `/reprocessing/orders/{id}/reset` | Reset order for retry |
| POST | `/reprocessing/orders/{id}/dlq` | Move to DLQ |

### Dead Letter Queue Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/dead-letter-queue` | List DLQ entries |
| GET | `/dead-letter-queue/stats` | DLQ statistics |
| PATCH | `/dead-letter-queue/{id}/resolve` | Resolve DLQ entry |

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| MaxReprocessingRetries | 5 | Maximum retry attempts |
| ReprocessingInterval | 5 minutes | Time between reprocessing runs |
| RetryBackoff | Exponential | Backoff strategy |

## Related Documentation

- [Order Service - Reprocessing API](/services/order-service#reprocessing-api)
- [Order Service - Dead Letter Queue](/services/order-service#dead-letter-queue-api)
- [Reprocessing Activities](/temporal/activities/reprocessing-activities)
- [Domain Events](/domain-driven-design/domain-events#order-events)
