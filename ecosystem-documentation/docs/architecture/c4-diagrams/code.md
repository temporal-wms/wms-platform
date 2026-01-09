---
sidebar_position: 4
---

# C4 Level 4: Code Diagrams

Code diagrams show the internal structure of components at the class/struct level, focusing on key domain aggregates and their relationships.

## Order Aggregate

```mermaid
classDiagram
    class Order {
        <<Aggregate Root>>
        +ID string
        +CustomerID string
        +Status OrderStatus
        +Priority Priority
        +Items []OrderItem
        +ShippingAddress Address
        +WaveID *string
        +CreatedAt time.Time
        +UpdatedAt time.Time
        +Validate() error
        +AssignToWave(waveID string)
        +Cancel(reason string)
        +MarkShipped(trackingNumber string)
        +Complete()
    }

    class OrderItem {
        <<Entity>>
        +ID string
        +SKU string
        +ProductName string
        +Quantity int
        +PickedQuantity int
        +Price Money
        +Weight Weight
        +Dimensions Dimensions
        +Pick(quantity int)
        +IsFullyPicked() bool
    }

    class Address {
        <<Value Object>>
        +Street string
        +City string
        +State string
        +ZipCode string
        +Country string
        +Validate() error
        +Format() string
    }

    class Money {
        <<Value Object>>
        +Amount float64
        +Currency string
        +Add(Money) Money
        +Multiply(int) Money
    }

    class OrderStatus {
        <<Enumeration>>
        RECEIVED
        VALIDATED
        WAVE_ASSIGNED
        PICKING
        CONSOLIDATING
        PACKING
        SHIPPED
        COMPLETED
        CANCELLED
    }

    Order "1" *-- "*" OrderItem : contains
    Order "1" *-- "1" Address : shippingAddress
    OrderItem "1" *-- "1" Money : price
    Order --> OrderStatus : status
```

## Wave Aggregate

```mermaid
classDiagram
    class Wave {
        <<Aggregate Root>>
        +ID string
        +WaveNumber string
        +Status WaveStatus
        +Type WaveType
        +Orders []WaveOrder
        +ScheduledAt *time.Time
        +ReleasedAt *time.Time
        +CompletedAt *time.Time
        +CreatedAt time.Time
        +AddOrder(order Order)
        +RemoveOrder(orderID string)
        +Schedule(scheduledAt time.Time)
        +Release()
        +Complete()
        +Cancel(reason string)
        +CanAddOrder() bool
        +GetOrderCount() int
    }

    class WaveOrder {
        <<Entity>>
        +OrderID string
        +Priority Priority
        +ItemCount int
        +AddedAt time.Time
    }

    class WaveStatus {
        <<Enumeration>>
        OPEN
        SCHEDULED
        RELEASED
        IN_PROGRESS
        COMPLETED
        CANCELLED
    }

    class WaveType {
        <<Enumeration>>
        STANDARD
        EXPRESS
        PRIORITY
        BULK
    }

    Wave "1" *-- "*" WaveOrder : orders
    Wave --> WaveStatus : status
    Wave --> WaveType : type
```

## PickTask Aggregate

```mermaid
classDiagram
    class PickTask {
        <<Aggregate Root>>
        +ID string
        +OrderID string
        +WaveID string
        +WorkerID *string
        +Status PickTaskStatus
        +Items []PickItem
        +Route *PickRoute
        +StartedAt *time.Time
        +CompletedAt *time.Time
        +AssignWorker(workerID string)
        +Start()
        +PickItem(itemID string, quantity int, location Location)
        +ReportException(itemID string, reason string)
        +Complete()
        +GetProgress() float64
    }

    class PickItem {
        <<Entity>>
        +ID string
        +SKU string
        +ProductName string
        +Quantity int
        +PickedQuantity int
        +Location Location
        +Status PickItemStatus
        +PickedAt *time.Time
        +ExceptionReason *string
    }

    class Location {
        <<Value Object>>
        +Zone string
        +Aisle string
        +Rack string
        +Level string
        +Position string
        +X int
        +Y int
        +DistanceFrom(Location) float64
    }

    class PickRoute {
        <<Value Object>>
        +Stops []RouteStop
        +TotalDistance float64
        +EstimatedTime time.Duration
    }

    class RouteStop {
        <<Value Object>>
        +Location Location
        +ItemID string
        +Sequence int
    }

    PickTask "1" *-- "*" PickItem : items
    PickTask "1" *-- "0..1" PickRoute : route
    PickItem "1" *-- "1" Location : location
    PickRoute "1" *-- "*" RouteStop : stops
    RouteStop "1" *-- "1" Location : location
```

## Shipment Aggregate

```mermaid
classDiagram
    class Shipment {
        <<Aggregate Root>>
        +ID string
        +OrderID string
        +Carrier Carrier
        +Service ShippingService
        +Status ShipmentStatus
        +TrackingNumber *string
        +Label *ShippingLabel
        +Packages []Package
        +ShippedAt *time.Time
        +DeliveredAt *time.Time
        +GenerateLabel()
        +Manifest()
        +Confirm()
        +RecordDelivery()
    }

    class Package {
        <<Entity>>
        +ID string
        +PackageType PackageType
        +Weight Weight
        +Dimensions Dimensions
        +Items []PackageItem
    }

    class ShippingLabel {
        <<Value Object>>
        +TrackingNumber string
        +Barcode string
        +LabelData []byte
        +LabelFormat string
        +GeneratedAt time.Time
    }

    class Carrier {
        <<Enumeration>>
        UPS
        FEDEX
        USPS
        DHL
    }

    class ShipmentStatus {
        <<Enumeration>>
        CREATED
        LABEL_GENERATED
        MANIFESTED
        SHIPPED
        IN_TRANSIT
        DELIVERED
    }

    Shipment "1" *-- "*" Package : packages
    Shipment "1" *-- "0..1" ShippingLabel : label
    Shipment --> Carrier : carrier
    Shipment --> ShipmentStatus : status
```

## InventoryItem Aggregate

```mermaid
classDiagram
    class InventoryItem {
        <<Aggregate Root>>
        +ID string
        +SKU string
        +ProductName string
        +Location Location
        +Quantity int
        +ReservedQuantity int
        +MinStock int
        +MaxStock int
        +Reservations []Reservation
        +Reserve(orderID string, quantity int) error
        +ReleaseReservation(orderID string)
        +Pick(quantity int)
        +Receive(quantity int)
        +Adjust(quantity int, reason string)
        +GetAvailableQuantity() int
        +IsLowStock() bool
    }

    class Reservation {
        <<Entity>>
        +ID string
        +OrderID string
        +Quantity int
        +ReservedAt time.Time
        +ExpiresAt time.Time
    }

    class Location {
        <<Value Object>>
        +Zone string
        +Aisle string
        +Rack string
        +Level string
        +Position string
    }

    InventoryItem "1" *-- "*" Reservation : reservations
    InventoryItem "1" *-- "1" Location : location
```

## Worker Aggregate

```mermaid
classDiagram
    class Worker {
        <<Aggregate Root>>
        +ID string
        +EmployeeID string
        +Name string
        +Role WorkerRole
        +Status WorkerStatus
        +CurrentTask *TaskAssignment
        +Certifications []Certification
        +Performance PerformanceMetrics
        +StartShift()
        +EndShift()
        +AssignTask(taskType string, taskID string)
        +CompleteTask()
        +RecordPerformance(metric string, value float64)
        +IsAvailable() bool
        +CanPerform(taskType string) bool
    }

    class TaskAssignment {
        <<Entity>>
        +TaskType string
        +TaskID string
        +AssignedAt time.Time
        +Zone *string
    }

    class PerformanceMetrics {
        <<Value Object>>
        +TasksCompleted int
        +AverageTaskTime time.Duration
        +Accuracy float64
        +LastUpdated time.Time
    }

    class Certification {
        <<Value Object>>
        +Type string
        +IssuedAt time.Time
        +ExpiresAt time.Time
        +IsValid() bool
    }

    class WorkerRole {
        <<Enumeration>>
        PICKER
        PACKER
        RECEIVER
        FORKLIFT_OPERATOR
        SUPERVISOR
    }

    Worker "1" *-- "0..1" TaskAssignment : currentTask
    Worker "1" *-- "*" Certification : certifications
    Worker "1" *-- "1" PerformanceMetrics : performance
    Worker --> WorkerRole : role
```

## Repository Interfaces

```mermaid
classDiagram
    class OrderRepository {
        <<Interface>>
        +Save(order Order) error
        +FindByID(id string) (*Order, error)
        +FindByCustomerID(customerID string) ([]Order, error)
        +FindByStatus(status OrderStatus) ([]Order, error)
        +FindPendingForWaving() ([]Order, error)
        +Update(order Order) error
    }

    class WaveRepository {
        <<Interface>>
        +Save(wave Wave) error
        +FindByID(id string) (*Wave, error)
        +FindOpen() ([]Wave, error)
        +FindByStatus(status WaveStatus) ([]Wave, error)
        +Update(wave Wave) error
    }

    class PickTaskRepository {
        <<Interface>>
        +Save(task PickTask) error
        +FindByID(id string) (*PickTask, error)
        +FindByWaveID(waveID string) ([]PickTask, error)
        +FindByWorkerID(workerID string) ([]PickTask, error)
        +FindPending() ([]PickTask, error)
        +Update(task PickTask) error
    }

    class MongoOrderRepository {
        +collection *mongo.Collection
        +Save(order Order) error
        +FindByID(id string) (*Order, error)
    }

    OrderRepository <|.. MongoOrderRepository
```

## Event Types

```mermaid
classDiagram
    class DomainEvent {
        <<Interface>>
        +GetEventType() string
        +GetAggregateID() string
        +GetTimestamp() time.Time
    }

    class OrderReceivedEvent {
        +OrderID string
        +CustomerID string
        +Items []OrderItemData
        +TotalAmount Money
        +Priority Priority
    }

    class WaveReleasedEvent {
        +WaveID string
        +WaveNumber string
        +OrderIDs []string
        +ItemCount int
    }

    class PickTaskCompletedEvent {
        +TaskID string
        +OrderID string
        +WaveID string
        +WorkerID string
        +Duration time.Duration
        +ItemsPicked int
    }

    class ShipConfirmedEvent {
        +ShipmentID string
        +OrderID string
        +TrackingNumber string
        +Carrier string
    }

    DomainEvent <|.. OrderReceivedEvent
    DomainEvent <|.. WaveReleasedEvent
    DomainEvent <|.. PickTaskCompletedEvent
    DomainEvent <|.. ShipConfirmedEvent
```

## Related Diagrams

- [Component Diagram](./components) - Component-level view
- [Aggregates](/domain-driven-design/aggregates/order) - Detailed aggregate documentation
- [Domain Events](/domain-driven-design/domain-events) - Complete event catalog
