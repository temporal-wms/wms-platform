# Gift Wrap Workflow

This diagram shows the gift wrap child workflow that coordinates special gift wrapping for orders.

## Gift Wrap Sequence Diagram

```mermaid
sequenceDiagram
    autonumber
    participant Parent as OrderFulfillmentWorkflow
    participant GiftWrap as GiftWrapWorkflow
    participant LaborSvc as Labor Service
    participant PackingSvc as Packing Service
    participant Worker as Gift Wrap Worker
    participant Station as Gift Wrap Station

    Parent->>GiftWrap: Start GiftWrapWorkflow
    Note over GiftWrap: WorkflowID: giftwrap-{orderId}

    alt StationID Not Provided
        rect rgb(240, 248, 255)
            Note over GiftWrap,LaborSvc: Step 1: Find Capable Station
            GiftWrap->>LaborSvc: FindCapableStation Activity
            Note right of GiftWrap: requirements: ["gift_wrap"]
            LaborSvc->>LaborSvc: Find station with gift_wrap capability
            LaborSvc-->>GiftWrap: StationID
        end
    end

    rect rgb(255, 250, 240)
        Note over GiftWrap,PackingSvc: Step 2: Create Gift Wrap Task
        GiftWrap->>PackingSvc: CreateGiftWrapTask Activity
        PackingSvc->>PackingSvc: Create task record
        PackingSvc->>PackingSvc: Include wrap details & message
        PackingSvc-->>GiftWrap: TaskID
    end

    rect rgb(240, 255, 240)
        Note over GiftWrap,LaborSvc: Step 3: Assign Gift Wrap Worker
        GiftWrap->>LaborSvc: AssignGiftWrapWorker Activity
        LaborSvc->>LaborSvc: Find worker with gift wrap certification
        LaborSvc->>LaborSvc: Assign to station
        LaborSvc-->>GiftWrap: WorkerID
    end

    rect rgb(255, 240, 245)
        Note over GiftWrap,Worker: Step 4: Wait for Gift Wrap Completion
        GiftWrap->>GiftWrap: Wait for Signal (gift-wrap-completed)
        Note right of GiftWrap: Timeout: 20 minutes

        Worker->>Station: Retrieve items
        Worker->>Worker: Select wrapping paper
        Worker->>Worker: Wrap items
        Worker->>Worker: Add ribbon/bow
        Worker->>PackingSvc: Mark complete

        alt Signal Received
            PackingSvc->>GiftWrap: Signal: gift-wrap-completed
        else Timeout
            GiftWrap->>PackingSvc: CheckGiftWrapStatus Activity
            PackingSvc-->>GiftWrap: Status (complete/pending)
        end
    end

    alt Has Gift Message
        rect rgb(245, 245, 255)
            Note over GiftWrap,PackingSvc: Step 5: Apply Gift Message
            GiftWrap->>PackingSvc: ApplyGiftMessage Activity
            PackingSvc->>Worker: Print message card
            Worker->>Worker: Attach to package
            PackingSvc-->>GiftWrap: Message Applied
        end
    end

    rect rgb(255, 253, 231)
        Note over GiftWrap,PackingSvc: Step 6: Complete Gift Wrap Task
        GiftWrap->>PackingSvc: CompleteGiftWrapTask Activity
        PackingSvc->>PackingSvc: Mark task complete
        PackingSvc->>PackingSvc: Update order status
        PackingSvc-->>GiftWrap: Task Completed
    end

    GiftWrap-->>Parent: GiftWrapResult

    Note over Parent: Continue to Packing
```

## Gift Wrap Task State Machine

```mermaid
stateDiagram-v2
    [*] --> Pending: Task Created
    Pending --> Assigned: Worker Assigned
    Assigned --> InProgress: Wrapping Started
    InProgress --> Wrapped: Items Wrapped
    Wrapped --> MessageApplied: Gift Message Added
    MessageApplied --> Completed: Task Finalized
    Wrapped --> Completed: No Message Required
    Completed --> [*]

    Pending --> Cancelled: Timeout
    Assigned --> Cancelled: Worker Unavailable
    InProgress --> Cancelled: Items Missing
    Cancelled --> [*]
```

## Gift Wrap Process Flow

```mermaid
flowchart TD
    Start[Receive Order] --> Check{Station Assigned?}
    Check -->|No| Find[Find Capable Station]
    Check -->|Yes| Task[Create Task]
    Find --> Task

    Task --> Assign[Assign Worker]
    Assign --> Wait[Wait for Completion]

    Wait --> Select[Select Wrapping Paper]
    Select --> Wrap[Wrap Items]
    Wrap --> Ribbon[Add Ribbon/Bow]

    Ribbon --> Message{Has Gift Message?}
    Message -->|Yes| Print[Print Message Card]
    Print --> Attach[Attach to Package]
    Attach --> Complete[Complete Task]
    Message -->|No| Complete

    Complete --> Return[Return to Workflow]
```

## Wrap Types

```mermaid
graph LR
    subgraph "Standard Options"
        Classic[Classic Paper]
        Floral[Floral Pattern]
        Holiday[Holiday Theme]
    end

    subgraph "Premium Options"
        Satin[Satin Ribbon]
        Velvet[Velvet Box]
        Custom[Custom Design]
    end

    subgraph "Accessories"
        Bow[Gift Bow]
        Tag[Gift Tag]
        Card[Message Card]
    end

    Classic --> Bow
    Floral --> Bow
    Holiday --> Tag
    Satin --> Tag
    Velvet --> Card
    Custom --> Card
```

## Data Structures

### GiftWrapInput
| Field | Type | Description |
|-------|------|-------------|
| OrderID | string | Order to gift wrap |
| WaveID | string | Processing wave |
| Items | []GiftWrapItem | Items to wrap |
| WrapDetails | GiftWrapDetails | Wrap configuration |
| StationID | string | Pre-assigned station (optional) |

### GiftWrapDetails
| Field | Type | Description |
|-------|------|-------------|
| WrapType | string | Paper type (classic/floral/holiday/premium) |
| GiftMessage | string | Message for gift card |
| HidePrice | bool | Whether to exclude price tags |

### GiftWrapResult
| Field | Type | Description |
|-------|------|-------------|
| TaskID | string | Completed task ID |
| OrderID | string | Order ID |
| StationID | string | Station used |
| WorkerID | string | Worker who wrapped |
| CompletedAt | time.Time | Completion timestamp |
| Success | bool | Completion status |

### GiftWrapTask
| Field | Type | Description |
|-------|------|-------------|
| TaskID | string | Unique identifier |
| OrderID | string | Associated order |
| StationID | string | Assigned station |
| WorkerID | string | Assigned worker |
| Status | string | Current status |
| WrapType | string | Wrap configuration |
| GiftMessage | string | Gift message text |
| HidePrice | bool | Exclude pricing |

## Worker Requirements

| Certification | Description |
|---------------|-------------|
| gift_wrap_basic | Can perform standard wrapping |
| gift_wrap_premium | Can perform premium/custom wrapping |
| gift_message | Can create handwritten cards |

## Station Capabilities

| Capability | Equipment |
|------------|-----------|
| gift_wrap | Wrapping paper, scissors, tape |
| ribbon_station | Ribbon, bows, accessories |
| message_printer | Card printer, pens |
| premium_materials | Velvet boxes, satin |

## Error Handling

```mermaid
flowchart TD
    Start[Start Wrapping] --> Station{Station Found?}
    Station -->|No| Error1[Failed: No capable station]
    Station -->|Yes| Worker{Worker Available?}

    Worker -->|No| Error2[Failed: No certified worker]
    Worker -->|Yes| Wrap{Wrapping OK?}

    Wrap -->|Materials Missing| Error3[Alert: Restock needed]
    Error3 --> Retry[Wait & Retry]
    Retry --> Wrap

    Wrap -->|Timeout| Check[Check Status]
    Check -->|Complete| Success
    Check -->|Pending| Error4[Failed: Gift wrap timeout]

    Wrap -->|Success| Message{Apply Message?}
    Message -->|Error| Warn[Warn: Continue without message]
    Warn --> Complete
    Message -->|OK| Complete
    Complete --> Success[Return Result]
```

## Related Diagrams

- [Order Fulfillment Flow](order-fulfillment.md) - Parent workflow
- [Packing Workflow](packing-workflow.md) - Next step
- [Consolidation Workflow](consolidation-workflow.md) - Previous step (if multi-item)
