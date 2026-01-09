# Labor Service - Class Diagram

This diagram shows the domain model for the Labor Service bounded context.

## Domain Model

```mermaid
classDiagram
    class Worker {
        <<Aggregate Root>>
        +WorkerID string
        +EmployeeID string
        +Name string
        +Email string
        +Status WorkerStatus
        +CurrentZone string
        +Skills []Skill
        +CurrentShift Shift
        +Breaks []Break
        +CurrentTask TaskAssignment
        +Performance PerformanceMetrics
        +StartShift(shiftType string)
        +EndShift()
        +StartBreak(breakType string)
        +EndBreak()
        +AssignTask(task TaskAssignment)
        +StartTask()
        +CompleteTask()
        +UpdateZone(zone string)
        +AddSkill(skill Skill)
        +HasSkill(skillType string) bool
    }

    class Skill {
        <<Entity>>
        +SkillID string
        +Type SkillType
        +Level int
        +Certified bool
        +CertifiedAt time.Time
        +ExpiresAt time.Time
    }

    class Shift {
        <<Entity>>
        +ShiftID string
        +Type ShiftType
        +StartTime time.Time
        +EndTime time.Time
        +Zone string
        +TasksCompleted int
        +ItemsProcessed int
        +TotalBreakMinutes int
    }

    class Break {
        <<Value Object>>
        +StartTime time.Time
        +EndTime time.Time
        +Type BreakType
        +Duration() time.Duration
    }

    class TaskAssignment {
        <<Entity>>
        +TaskID string
        +TaskType TaskType
        +Priority int
        +AssignedAt time.Time
        +StartedAt time.Time
        +CompletedAt time.Time
        +Zone string
    }

    class PerformanceMetrics {
        <<Value Object>>
        +TotalTasksCompleted int
        +AverageTaskTime time.Duration
        +AverageItemsPerHour float64
        +AccuracyRate float64
        +LastUpdated time.Time
    }

    class WorkerStatus {
        <<Enumeration>>
        available
        on_task
        on_break
        offline
    }

    class SkillType {
        <<Enumeration>>
        picking
        packing
        receiving
        consolidation
        replenishment
    }

    class ShiftType {
        <<Enumeration>>
        morning
        afternoon
        night
    }

    class BreakType {
        <<Enumeration>>
        break
        lunch
    }

    class TaskType {
        <<Enumeration>>
        picking
        packing
        receiving
        consolidation
        replenishment
    }

    Worker "1" *-- "*" Skill : has
    Worker "1" *-- "0..1" Shift : working
    Worker "1" *-- "*" Break : takes
    Worker "1" *-- "0..1" TaskAssignment : assigned
    Worker "1" *-- "1" PerformanceMetrics : tracked
    Worker --> WorkerStatus : has
    Skill --> SkillType : type of
    Shift --> ShiftType : type of
    Break --> BreakType : type of
    TaskAssignment --> TaskType : type of
```

## Worker Lifecycle

```mermaid
stateDiagram-v2
    [*] --> offline: Worker Registered
    offline --> available: StartShift()
    available --> on_task: AssignTask()
    on_task --> available: CompleteTask()
    available --> on_break: StartBreak()
    on_break --> available: EndBreak()
    available --> offline: EndShift()
    on_task --> on_break: StartBreak()
    on_break --> on_task: EndBreak() [has task]
```

## Performance Tracking

```mermaid
flowchart TD
    Task[Task Completed] --> Record[Record Metrics]
    Record --> Update[Update Performance]
    Update --> Calc[Calculate Averages]
    Calc --> Store[Store Metrics]

    subgraph "Metrics"
        TaskCount[Tasks Completed]
        AvgTime[Average Task Time]
        ItemsHour[Items per Hour]
        Accuracy[Accuracy Rate]
    end

    Store --> TaskCount
    Store --> AvgTime
    Store --> ItemsHour
    Store --> Accuracy
```

## Related Diagrams

- [Aggregate Diagram](ddd/aggregates.md) - DDD aggregate structure
- [Picking Workflow](../../../orchestrator/docs/diagrams/picking-workflow.md) - Worker assignment
