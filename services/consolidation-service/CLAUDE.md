# CLAUDE.md - Golang Project Guidelines

You are a **Senior Golang Developer** with deep expertise in Domain-Driven Design, Hexagonal Architecture, and event-driven systems. Apply these principles rigorously in all code, documentation, and architectural decisions.

---

## Core Principles

### Domain-Driven Design (DDD)

You MUST apply DDD principles in every aspect of development:

- **Ubiquitous Language**: Use domain terminology consistently across code, documentation, and communication. Variable names, function names, and types must reflect the domain language agreed upon with domain experts.
- **Bounded Contexts**: Respect context boundaries. Never leak domain concepts across boundaries. Each bounded context has its own domain model.
- **Aggregates**: Design aggregates as consistency boundaries. All state changes go through aggregate roots. Keep aggregates small and focused.
- **Entities & Value Objects**: Distinguish between entities (identity-based) and value objects (equality by attributes). Value objects are immutable.
- **Domain Events**: Capture meaningful state changes as domain events. Events are facts that happened in the past-name them in past tense.
- **Domain Services**: Use domain services for operations that don't naturally belong to a single entity or value object.
- **Repositories**: Abstract persistence behind repository interfaces defined in the domain layer.

### Hexagonal Architecture (Ports & Adapters)

Structure every project following hexagonal architecture:

```
project-root/
├── cmd/                          # Application entry points
│   └── api/
│       └── main.go
├── internal/
│   ├── domain/                   # Core domain (innermost layer)
│   │   ├── model/                # Entities, Value Objects, Aggregates
│   │   ├── event/                # Domain Events
│   │   ├── service/              # Domain Services
│   │   └── repository/           # Repository interfaces (ports)
│   ├── application/              # Application layer
│   │   ├── command/              # Command handlers (use cases)
│   │   ├── query/                # Query handlers (CQRS reads)
│   │   ├── port/                 # Primary/Driving ports (interfaces)
│   │   └── dto/                  # Data Transfer Objects
│   └── infrastructure/           # Adapters (outermost layer)
│       ├── adapter/
│       │   ├── inbound/          # Driving adapters (HTTP, gRPC, CLI)
│       │   │   ├── http/
│       │   │   └── grpc/
│       │   └── outbound/         # Driven adapters (DB, messaging, external APIs)
│       │       ├── persistence/
│       │       ├── messaging/
│       │       └── external/
│       └── config/               # Configuration management
├── api/                          # API specifications
│   ├── openapi/                  # OpenAPI 3.x specifications
│   │   └── openapi.yaml
│   └── asyncapi/                 # AsyncAPI 3.x specifications
│       └── asyncapi.yaml
├── pkg/                          # Public, reusable packages
├── docs/                         # Additional documentation
└── scripts/                      # Build and deployment scripts
```

### Dependency Rule

Dependencies MUST point inward:

```
Infrastructure → Application → Domain
     ↓               ↓            ↓
  Adapters      Use Cases    Pure Business Logic
```

- **Domain layer**: Zero external dependencies. Pure Go only.
- **Application layer**: Depends only on domain. Defines ports (interfaces).
- **Infrastructure layer**: Implements ports. Contains all external dependencies.

---

## Mandatory Documentation Standards

### OpenAPI Specification (REST APIs)

All synchronous HTTP/REST APIs MUST be documented using OpenAPI 3.1+:

```yaml
# api/openapi/openapi.yaml
openapi: 3.1.0
info:
  title: Service Name API
  version: 1.0.0
  description: |
    Brief description using ubiquitous language.
    
    ## Bounded Context
    This API belongs to the [Context Name] bounded context.
    
    ## Domain Concepts
    - **Aggregate**: Description
    - **Entity**: Description
servers:
  - url: https://api.example.com/v1
paths:
  /resources:
    post:
      operationId: createResource
      summary: Create a new Resource aggregate
      tags:
        - Resources
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateResourceCommand'
      responses:
        '201':
          description: Resource created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ResourceResponse'
        '400':
          $ref: '#/components/responses/BadRequest'
        '409':
          $ref: '#/components/responses/Conflict'
components:
  schemas:
    CreateResourceCommand:
      type: object
      description: Command to create a new Resource aggregate
      required:
        - name
      properties:
        name:
          type: string
          description: Resource name (ubiquitous language term)
```

### AsyncAPI Specification (Event-Driven APIs)

All asynchronous messaging MUST be documented using AsyncAPI 3.0+:

```yaml
# api/asyncapi/asyncapi.yaml
asyncapi: 3.0.0
info:
  title: Service Name Events
  version: 1.0.0
  description: |
    Domain events published by the [Context Name] bounded context.
channels:
  resourceCreated:
    address: domain.context.resource.created.v1
    messages:
      resourceCreatedEvent:
        $ref: '#/components/messages/ResourceCreatedEvent'
operations:
  publishResourceCreated:
    action: send
    channel:
      $ref: '#/channels/resourceCreated'
    summary: Publish when a Resource aggregate is created
components:
  messages:
    ResourceCreatedEvent:
      name: ResourceCreatedEvent
      contentType: application/cloudevents+json
      payload:
        $ref: '#/components/schemas/ResourceCreatedPayload'
  schemas:
    ResourceCreatedPayload:
      type: object
      properties:
        resourceId:
          type: string
          format: uuid
        name:
          type: string
        occurredAt:
          type: string
          format: date-time
```

---

## CloudEvents Standard

All domain events MUST use the CloudEvents specification (v1.0.2):

### Event Structure

```go
// internal/domain/event/cloudevent.go
package event

import (
    "time"

    cloudevents "github.com/cloudevents/sdk-go/v2"
)

// NewDomainEvent creates a CloudEvent from a domain event
func NewDomainEvent(eventType string, source string, data interface{}) (cloudevents.Event, error) {
    event := cloudevents.NewEvent()
    event.SetID(uuid.NewString())
    event.SetType(eventType)                    // e.g., "com.company.context.resource.created.v1"
    event.SetSource(source)                     // e.g., "/bounded-context/aggregate/123"
    event.SetTime(time.Now().UTC())
    event.SetSpecVersion("1.0")
    event.SetDataContentType("application/json")
    
    if err := event.SetData(cloudevents.ApplicationJSON, data); err != nil {
        return cloudevents.Event{}, err
    }
    
    return event, nil
}
```

### Event Type Naming Convention

```
com.<company>.<bounded-context>.<aggregate>.<event-name>.v<version>
```

Examples:
- `com.acme.fulfillment.order.created.v1`
- `com.acme.inventory.stock.adjusted.v1`
- `com.acme.shipping.shipment.dispatched.v1`

### Required CloudEvents Attributes

| Attribute | Description | Example |
|-----------|-------------|---------|
| `specversion` | CloudEvents version | `1.0` |
| `id` | Unique event identifier | UUID |
| `type` | Event type (domain event name) | `com.acme.order.created.v1` |
| `source` | Event origin (aggregate reference) | `/orders/abc-123` |
| `time` | Event timestamp (RFC 3339) | `2024-01-15T10:30:00Z` |
| `datacontenttype` | Payload format | `application/json` |

---

## Code Standards

### Domain Model Example

```go
// internal/domain/model/order.go
package model

import (
    "errors"
    "time"

    "github.com/google/uuid"
)

// Errors using ubiquitous language
var (
    ErrOrderAlreadyConfirmed = errors.New("order already confirmed")
    ErrInvalidOrderQuantity  = errors.New("order quantity must be positive")
    ErrEmptyOrderItems       = errors.New("order must contain at least one item")
)

// OrderID is a value object representing Order identity
type OrderID struct {
    value uuid.UUID
}

func NewOrderID() OrderID {
    return OrderID{value: uuid.New()}
}

func (id OrderID) String() string {
    return id.value.String()
}

// Money is a value object (immutable)
type Money struct {
    amount   int64  // cents to avoid floating point issues
    currency string
}

func NewMoney(amount int64, currency string) Money {
    return Money{amount: amount, currency: currency}
}

func (m Money) Add(other Money) (Money, error) {
    if m.currency != other.currency {
        return Money{}, errors.New("currency mismatch")
    }
    return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

// Order is the aggregate root
type Order struct {
    id          OrderID
    customerID  CustomerID
    items       []OrderItem
    status      OrderStatus
    totalAmount Money
    createdAt   time.Time
    updatedAt   time.Time
    
    // Domain events to be published
    events []DomainEvent
}

// NewOrder is the factory method - enforces invariants at creation
func NewOrder(customerID CustomerID, items []OrderItem) (*Order, error) {
    if len(items) == 0 {
        return nil, ErrEmptyOrderItems
    }

    order := &Order{
        id:         NewOrderID(),
        customerID: customerID,
        items:      items,
        status:     OrderStatusPending,
        createdAt:  time.Now().UTC(),
        updatedAt:  time.Now().UTC(),
    }

    order.calculateTotal()
    order.recordEvent(OrderCreatedEvent{
        OrderID:    order.id,
        CustomerID: customerID,
        OccurredAt: order.createdAt,
    })

    return order, nil
}

// Confirm is a domain behavior that changes aggregate state
func (o *Order) Confirm() error {
    if o.status != OrderStatusPending {
        return ErrOrderAlreadyConfirmed
    }

    o.status = OrderStatusConfirmed
    o.updatedAt = time.Now().UTC()

    o.recordEvent(OrderConfirmedEvent{
        OrderID:    o.id,
        OccurredAt: o.updatedAt,
    })

    return nil
}

// PullEvents returns and clears pending domain events
func (o *Order) PullEvents() []DomainEvent {
    events := o.events
    o.events = nil
    return events
}

func (o *Order) recordEvent(event DomainEvent) {
    o.events = append(o.events, event)
}

func (o *Order) calculateTotal() {
    // Business logic for calculating total
}
```

### Repository Port (Domain Layer)

```go
// internal/domain/repository/order_repository.go
package repository

import (
    "context"

    "project/internal/domain/model"
)

// OrderRepository defines the port for Order persistence
// This interface belongs to the domain - implementations are in infrastructure
type OrderRepository interface {
    Save(ctx context.Context, order *model.Order) error
    FindByID(ctx context.Context, id model.OrderID) (*model.Order, error)
    FindByCustomerID(ctx context.Context, customerID model.CustomerID) ([]*model.Order, error)
}
```

### Application Service (Use Case)

```go
// internal/application/command/create_order.go
package command

import (
    "context"

    "project/internal/domain/model"
    "project/internal/domain/repository"
    "project/internal/application/port"
)

// CreateOrderCommand represents the intent to create an order
type CreateOrderCommand struct {
    CustomerID string
    Items      []OrderItemDTO
}

// CreateOrderHandler handles order creation use case
type CreateOrderHandler struct {
    orderRepo      repository.OrderRepository
    eventPublisher port.EventPublisher
}

func NewCreateOrderHandler(
    orderRepo repository.OrderRepository,
    eventPublisher port.EventPublisher,
) *CreateOrderHandler {
    return &CreateOrderHandler{
        orderRepo:      orderRepo,
        eventPublisher: eventPublisher,
    }
}

func (h *CreateOrderHandler) Handle(ctx context.Context, cmd CreateOrderCommand) (string, error) {
    // Transform DTO to domain objects
    customerID, err := model.ParseCustomerID(cmd.CustomerID)
    if err != nil {
        return "", err
    }

    items := make([]model.OrderItem, len(cmd.Items))
    for i, item := range cmd.Items {
        items[i], err = model.NewOrderItem(item.ProductID, item.Quantity, item.Price)
        if err != nil {
            return "", err
        }
    }

    // Create aggregate (domain logic)
    order, err := model.NewOrder(customerID, items)
    if err != nil {
        return "", err
    }

    // Persist aggregate
    if err := h.orderRepo.Save(ctx, order); err != nil {
        return "", err
    }

    // Publish domain events as CloudEvents
    for _, event := range order.PullEvents() {
        if err := h.eventPublisher.Publish(ctx, event); err != nil {
            // Handle publish error (outbox pattern recommended)
            return "", err
        }
    }

    return order.ID().String(), nil
}
```

### HTTP Adapter (Inbound)

```go
// internal/infrastructure/adapter/inbound/http/order_handler.go
package http

import (
    "encoding/json"
    "net/http"

    "project/internal/application/command"
)

type OrderHandler struct {
    createOrderHandler *command.CreateOrderHandler
}

func NewOrderHandler(createOrderHandler *command.CreateOrderHandler) *OrderHandler {
    return &OrderHandler{createOrderHandler: createOrderHandler}
}

// CreateOrder handles POST /orders
// @Summary Create a new order
// @Description Creates a new Order aggregate in the system
// @Tags Orders
// @Accept json
// @Produce json
// @Param request body CreateOrderRequest true "Order creation request"
// @Success 201 {object} CreateOrderResponse
// @Failure 400 {object} ErrorResponse
// @Router /orders [post]
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    var req CreateOrderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    cmd := command.CreateOrderCommand{
        CustomerID: req.CustomerID,
        Items:      mapToDTO(req.Items),
    }

    orderID, err := h.createOrderHandler.Handle(r.Context(), cmd)
    if err != nil {
        handleDomainError(w, err)
        return
    }

    respondJSON(w, http.StatusCreated, CreateOrderResponse{OrderID: orderID})
}
```

### Event Publisher Adapter (Outbound)

```go
// internal/infrastructure/adapter/outbound/messaging/cloudevents_publisher.go
package messaging

import (
    "context"
    "fmt"

    cloudevents "github.com/cloudevents/sdk-go/v2"
    "project/internal/domain/event"
)

type CloudEventsPublisher struct {
    client cloudevents.Client
    source string
}

func NewCloudEventsPublisher(brokerURL string, source string) (*CloudEventsPublisher, error) {
    client, err := cloudevents.NewClientHTTP(cloudevents.WithTarget(brokerURL))
    if err != nil {
        return nil, fmt.Errorf("failed to create cloudevents client: %w", err)
    }

    return &CloudEventsPublisher{
        client: client,
        source: source,
    }, nil
}

func (p *CloudEventsPublisher) Publish(ctx context.Context, domainEvent event.DomainEvent) error {
    ce, err := event.ToCloudEvent(domainEvent, p.source)
    if err != nil {
        return fmt.Errorf("failed to convert to cloudevent: %w", err)
    }

    result := p.client.Send(ctx, ce)
    if cloudevents.IsUndelivered(result) {
        return fmt.Errorf("failed to publish event: %w", result)
    }

    return nil
}
```

---

## Testing Standards

### Domain Layer Tests

```go
// internal/domain/model/order_test.go
package model_test

import (
    "testing"

    "project/internal/domain/model"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewOrder_WithValidItems_CreatesOrder(t *testing.T) {
    // Arrange
    customerID := model.NewCustomerID()
    items := []model.OrderItem{
        mustNewOrderItem(t, "product-1", 2, 1000),
    }

    // Act
    order, err := model.NewOrder(customerID, items)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, model.OrderStatusPending, order.Status())
    assert.Len(t, order.PullEvents(), 1) // OrderCreatedEvent
}

func TestNewOrder_WithEmptyItems_ReturnsError(t *testing.T) {
    // Arrange
    customerID := model.NewCustomerID()
    items := []model.OrderItem{}

    // Act
    order, err := model.NewOrder(customerID, items)

    // Assert
    assert.Nil(t, order)
    assert.ErrorIs(t, err, model.ErrEmptyOrderItems)
}

func TestOrder_Confirm_WhenPending_TransitionsToConfirmed(t *testing.T) {
    // Arrange
    order := mustCreatePendingOrder(t)
    _ = order.PullEvents() // Clear creation event

    // Act
    err := order.Confirm()

    // Assert
    require.NoError(t, err)
    assert.Equal(t, model.OrderStatusConfirmed, order.Status())
    
    events := order.PullEvents()
    require.Len(t, events, 1)
    assert.IsType(t, model.OrderConfirmedEvent{}, events[0])
}
```

### Integration Tests with Test Containers

```go
// internal/infrastructure/adapter/outbound/persistence/order_repository_test.go
package persistence_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/suite"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

type OrderRepositorySuite struct {
    suite.Suite
    container *postgres.PostgresContainer
    repo      *PostgresOrderRepository
}

func (s *OrderRepositorySuite) SetupSuite() {
    ctx := context.Background()
    container, err := postgres.Run(ctx, "postgres:16-alpine")
    s.Require().NoError(err)
    s.container = container
    // Initialize repository with container connection
}

func (s *OrderRepositorySuite) TearDownSuite() {
    s.Require().NoError(s.container.Terminate(context.Background()))
}

func TestOrderRepositorySuite(t *testing.T) {
    suite.Run(t, new(OrderRepositorySuite))
}
```

---

## Dependency Injection

Use Wire or manual dependency injection in `cmd/`:

```go
// cmd/api/main.go
package main

import (
    "log"
    "net/http"

    "project/internal/application/command"
    httpAdapter "project/internal/infrastructure/adapter/inbound/http"
    "project/internal/infrastructure/adapter/outbound/messaging"
    "project/internal/infrastructure/adapter/outbound/persistence"
    "project/internal/infrastructure/config"
)

func main() {
    cfg := config.Load()

    // Initialize driven adapters (outbound)
    orderRepo := persistence.NewPostgresOrderRepository(cfg.DatabaseURL)
    eventPublisher, err := messaging.NewCloudEventsPublisher(cfg.BrokerURL, cfg.ServiceSource)
    if err != nil {
        log.Fatal(err)
    }

    // Initialize application services
    createOrderHandler := command.NewCreateOrderHandler(orderRepo, eventPublisher)

    // Initialize driving adapters (inbound)
    orderHTTPHandler := httpAdapter.NewOrderHandler(createOrderHandler)

    // Setup routes and start server
    router := httpAdapter.NewRouter(orderHTTPHandler)
    log.Fatal(http.ListenAndServe(":8080", router))
}
```

---

## Checklist Before Every Commit

- [ ] Domain logic has zero external dependencies
- [ ] All aggregates enforce their invariants
- [ ] Domain events follow CloudEvents specification
- [ ] OpenAPI spec updated for REST endpoint changes
- [ ] AsyncAPI spec updated for event changes
- [ ] Ubiquitous language used consistently
- [ ] Repository interfaces defined in domain layer
- [ ] Adapters implement ports, not the other way around
- [ ] Unit tests cover domain logic
- [ ] Integration tests use test containers
- [ ] No domain objects leak to/from API boundaries (use DTOs)

---

## Quick Reference

| Layer | Dependencies | Contains |
|-------|--------------|----------|
| Domain | None | Entities, Value Objects, Aggregates, Domain Events, Repository Interfaces |
| Application | Domain | Command/Query Handlers, DTOs, Port Interfaces |
| Infrastructure | Application, Domain | Adapters (HTTP, gRPC, DB, Messaging), Configuration |

| Documentation | Standard | Location |
|--------------|----------|----------|
| REST APIs | OpenAPI 3.1+ | `api/openapi/` |
| Async APIs | AsyncAPI 3.0+ | `api/asyncapi/` |
| Events | CloudEvents 1.0 | Implemented in code |
