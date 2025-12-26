# Consolidation Service - DDD Aggregates

This document describes the aggregate structure for the Consolidation bounded context.

## Aggregate: ConsolidationUnit

The ConsolidationUnit aggregate manages combining items from multiple totes.

```mermaid
graph TD
    subgraph "ConsolidationUnit Aggregate"
        Unit[ConsolidationUnit<br/><<Aggregate Root>>]

        subgraph "Entities"
            Expected[ExpectedItem]
            Consolidated[ConsolidatedItem]
        end

        subgraph "Value Objects"
            UnitStatus[UnitStatus]
            Strategy[ConsolidationStrategy]
            ExpectedStatus[ExpectedStatus]
        end

        Unit -->|expects| Expected
        Unit -->|received| Consolidated
        Unit -->|status| UnitStatus
        Unit -->|strategy| Strategy
        Expected -->|status| ExpectedStatus
    end

    style Unit fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "ConsolidationUnit Aggregate Boundary"
        CU[ConsolidationUnit]
        EI[ExpectedItem]
        CI[ConsolidatedItem]
    end

    subgraph "External References"
        O[OrderID]
        W[WaveID]
        S[Station]
        T[SourceToteID]
    end

    CU -.->|for| O
    CU -.->|in| W
    CU -.->|at| S
    EI -.->|from| T

    style CU fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Station assigned | Must have station before start |
| Items match | Consolidated items must match expected |
| No duplicates | Each item scanned only once |
| Complete tracking | All items must be accounted for |

## Domain Events

```mermaid
graph LR
    Unit[ConsolidationUnit] -->|emits| E1[ConsolidationStartedEvent]
    Unit -->|emits| E2[ItemConsolidatedEvent]
    Unit -->|emits| E3[ShortItemReportedEvent]
    Unit -->|emits| E4[ConsolidationCompletedEvent]
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [Consolidation Workflow](../../../../orchestrator/docs/diagrams/consolidation-workflow.md) - Workflow details
