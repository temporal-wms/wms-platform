# Consolidation Workflow

This diagram shows the consolidation child workflow for multi-item orders, where items from different pick totes are combined into a single unit.

## Consolidation Sequence Diagram

```mermaid
sequenceDiagram
    autonumber
    participant Parent as OrderFulfillmentWorkflow
    participant Consolidation as ConsolidationWorkflow
    participant ConsolidationSvc as Consolidation Service
    participant Worker as Consolidation Worker
    participant Scanner as Barcode Scanner

    Parent->>Consolidation: Start ConsolidationWorkflow
    Note over Consolidation: WorkflowID: consolidation-{orderId}<br/>Only for multi-item orders

    rect rgb(240, 248, 255)
        Note over Consolidation,ConsolidationSvc: Step 1: Create Consolidation Unit
        Consolidation->>ConsolidationSvc: CreateConsolidationUnit Activity
        ConsolidationSvc->>ConsolidationSvc: Create unit with expected items
        ConsolidationSvc->>ConsolidationSvc: Assign to station
        ConsolidationSvc-->>Consolidation: ConsolidationID
    end

    rect rgb(255, 250, 240)
        Note over Consolidation,Worker: Step 2: Consolidate Items
        Consolidation->>ConsolidationSvc: ConsolidateItems Activity

        loop For Each Source Tote
            Worker->>Scanner: Scan Tote Barcode
            Scanner->>ConsolidationSvc: Verify Tote
            loop For Each Item in Tote
                Worker->>Scanner: Scan Item Barcode
                Scanner->>ConsolidationSvc: Record Item
                ConsolidationSvc->>ConsolidationSvc: Mark Item Received
                Worker->>Worker: Place in Consolidation Bin
            end
        end

        ConsolidationSvc-->>Consolidation: Items Consolidated
    end

    rect rgb(240, 255, 240)
        Note over Consolidation,ConsolidationSvc: Step 3: Verify Consolidation
        Consolidation->>ConsolidationSvc: VerifyConsolidation Activity
        ConsolidationSvc->>ConsolidationSvc: Check all items present
        ConsolidationSvc->>ConsolidationSvc: Validate quantities
        ConsolidationSvc-->>Consolidation: Verification Complete
    end

    rect rgb(255, 240, 245)
        Note over Consolidation,ConsolidationSvc: Step 4: Complete Consolidation
        Consolidation->>ConsolidationSvc: CompleteConsolidation Activity
        ConsolidationSvc->>ConsolidationSvc: Mark unit ready for packing
        ConsolidationSvc-->>Consolidation: Consolidation Complete
    end

    Consolidation-->>Parent: ConsolidationResult

    Note over Parent: Continue to Packing
```

## Consolidation State Machine

```mermaid
stateDiagram-v2
    [*] --> Pending: Unit Created
    Pending --> InProgress: Worker Started
    InProgress --> InProgress: Item Scanned
    InProgress --> Verified: All Items Received
    InProgress --> Short: Items Missing
    Short --> InProgress: Short Items Received
    Verified --> Completed: Verification Passed
    Completed --> [*]

    Pending --> Cancelled: Order Cancelled
    Short --> Cancelled: Timeout
    Cancelled --> [*]
```

## Consolidation Station Layout

```mermaid
graph TD
    subgraph "Consolidation Station"
        subgraph "Input"
            T1[Tote 1<br/>from Zone A]
            T2[Tote 2<br/>from Zone B]
            T3[Tote 3<br/>from Zone C]
        end

        subgraph "Workstation"
            Scanner[Barcode Scanner]
            Display[Order Display]
            Worker[Worker]
        end

        subgraph "Output"
            Bin[Consolidation Bin<br/>Ready for Packing]
        end

        T1 --> Worker
        T2 --> Worker
        T3 --> Worker
        Worker --> Scanner
        Scanner --> Display
        Worker --> Bin
    end
```

## Data Structures

### ConsolidationUnit
| Field | Type | Description |
|-------|------|-------------|
| ConsolidationID | string | Unique identifier |
| OrderID | string | Associated order |
| WaveID | string | Wave reference |
| Status | string | Current status |
| Station | string | Assigned station |
| ExpectedItems | []ExpectedItem | Items to receive |
| ConsolidatedItems | []ConsolidatedItem | Items received |
| DestinationBin | string | Output location |

### ExpectedItem
| Field | Type | Description |
|-------|------|-------------|
| SKU | string | Item SKU |
| Quantity | int | Expected quantity |
| SourceToteID | string | Source tote |
| Status | string | pending/partial/received/short |

### ConsolidatedItem
| Field | Type | Description |
|-------|------|-------------|
| SKU | string | Item SKU |
| Quantity | int | Received quantity |
| ScannedAt | time | Scan timestamp |
| VerifiedBy | string | Worker ID |

## Consolidation Strategies

| Strategy | Description | Use Case |
|----------|-------------|----------|
| order | Consolidate by order | Standard fulfillment |
| carrier | Group by carrier | Batch shipping |
| route | Group by delivery route | Regional optimization |
| time | Group by time window | SLA management |

## Related Diagrams

- [Picking Workflow](picking-workflow.md) - Previous step
- [Packing Workflow](packing-workflow.md) - Next step
- [Order Fulfillment Flow](../../../docs/diagrams/order-fulfillment-flow.md) - Parent workflow
