# Testing Strategy

This document outlines the testing strategy for the WMS Platform.

## Test Pyramid

```
         /\
        /  \  E2E Tests
       /----\  (Few, Slow)
      /      \
     /--------\  Integration Tests
    /          \  (Some, Medium)
   /------------\
  /              \  Unit Tests
 /----------------\  (Many, Fast)
```

## Test Types

### 1. Unit Tests

Test individual functions and methods in isolation.

**Location**: `*_test.go` files alongside source code

**Example**:

```go
// internal/domain/aggregate_test.go
func TestOrder_Validate(t *testing.T) {
    tests := []struct {
        name    string
        order   *Order
        wantErr bool
    }{
        {
            name:    "valid order",
            order:   validOrder(),
            wantErr: false,
        },
        {
            name:    "no items",
            order:   orderWithNoItems(),
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.order.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Run**:

```bash
# All unit tests
go test ./internal/domain/...

# Specific package
go test ./internal/application/...

# With coverage
go test -cover ./...
```

### 2. Integration Tests

Test service interactions with real dependencies (MongoDB, Kafka).

**Location**: `tests/integration/`

**Example**:

```go
// tests/integration/repository_test.go
func TestOrderRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()

    // Setup test container
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "mongo:7.0",
            ExposedPorts: []string{"27017/tcp"},
            WaitingFor:   wait.ForListeningPort("27017/tcp"),
        },
        Started: true,
    })
    require.NoError(t, err)
    defer container.Terminate(ctx)

    // Get connection string
    endpoint, _ := container.Endpoint(ctx, "")
    uri := fmt.Sprintf("mongodb://%s", endpoint)

    // Create repository
    client, _ := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    repo := NewOrderRepository(client.Database("test"))

    // Test CRUD operations
    order := domain.NewOrder(...)
    err = repo.Save(ctx, order)
    require.NoError(t, err)

    found, err := repo.FindByID(ctx, order.ID)
    require.NoError(t, err)
    assert.Equal(t, order.ID, found.ID)
}
```

**Run**:

```bash
# Start test infrastructure
docker compose -f docker-compose.test.yml up -d

# Run integration tests
go test -v ./tests/integration/...

# Or using make
make test-integration
```

### 3. Contract Tests

Verify API contracts and event schemas.

**Location**: `tests/contracts/`

**Example** (using Pact):

```go
// tests/contracts/consumer_test.go
func TestOrderServiceConsumer(t *testing.T) {
    pact := &dsl.Pact{
        Consumer: "order-service",
        Provider: "inventory-service",
    }

    // Define expected interaction
    pact.
        AddInteraction().
        Given("inventory exists for SKU").
        UponReceiving("a request for inventory").
        WithRequest(dsl.Request{
            Method: "GET",
            Path:   "/api/v1/inventory/SKU-001",
        }).
        WillRespondWith(dsl.Response{
            Status: 200,
            Body: map[string]interface{}{
                "sku":      "SKU-001",
                "quantity": 100,
            },
        })

    // Run test
    err := pact.Verify(func() error {
        client := NewInventoryClient(pact.Server.Port)
        inv, err := client.GetInventory("SKU-001")
        assert.Equal(t, "SKU-001", inv.SKU)
        return err
    })

    assert.NoError(t, err)
}
```

**Run**:

```bash
# Consumer tests
go test -v ./tests/contracts/consumer/...

# Provider verification
go test -v ./tests/contracts/provider/...
```

### 4. End-to-End Tests

Test complete workflows across services.

**Location**: `tests/e2e/`

**Example**:

```go
// tests/e2e/order_fulfillment_test.go
func TestOrderFulfillmentFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test")
    }

    ctx := context.Background()

    // Create order
    orderResp, err := http.Post(orderServiceURL+"/api/v1/orders",
        "application/json",
        orderJSON)
    require.NoError(t, err)
    require.Equal(t, http.StatusCreated, orderResp.StatusCode)

    var order OrderResponse
    json.NewDecoder(orderResp.Body).Decode(&order)

    // Wait for workflow to complete
    require.Eventually(t, func() bool {
        resp, _ := http.Get(orderServiceURL+"/api/v1/orders/"+order.ID)
        var o OrderResponse
        json.NewDecoder(resp.Body).Decode(&o)
        return o.Status == "shipped"
    }, 60*time.Second, 1*time.Second)
}
```

## Test Fixtures

### Database Fixtures

```go
// tests/fixtures/orders.go
func ValidOrder() *domain.Order {
    return &domain.Order{
        ID:         "ORD-TEST-001",
        CustomerID: "CUST-001",
        Status:     domain.OrderStatusReceived,
        Items: []domain.OrderItem{
            {
                SKU:      "SKU-001",
                Quantity: 2,
            },
        },
    }
}
```

### Event Fixtures

```go
// tests/fixtures/events.go
func OrderReceivedEvent() *events.OrderReceivedEvent {
    return &events.OrderReceivedEvent{
        OrderID:    "ORD-TEST-001",
        CustomerID: "CUST-001",
        ReceivedAt: time.Now(),
    }
}
```

## Mocking

### Repository Mocks

```go
// internal/mocks/order_repository.go
type MockOrderRepository struct {
    mock.Mock
}

func (m *MockOrderRepository) Save(ctx context.Context, order *domain.Order) error {
    args := m.Called(ctx, order)
    return args.Error(0)
}

func (m *MockOrderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*domain.Order), args.Error(1)
}
```

### Using Mocks

```go
func TestOrderService_CreateOrder(t *testing.T) {
    mockRepo := new(mocks.MockOrderRepository)
    mockPublisher := new(mocks.MockEventPublisher)

    service := NewOrderService(mockRepo, mockPublisher)

    // Setup expectations
    mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil)
    mockPublisher.On("Publish", mock.Anything).Return(nil)

    // Execute
    order, err := service.CreateOrder(ctx, cmd)

    // Verify
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
    mockPublisher.AssertExpectations(t)
}
```

## Coverage

### Minimum Coverage Requirements

| Type | Minimum |
|------|---------|
| Domain | 90% |
| Application | 80% |
| Infrastructure | 70% |
| Overall | 75% |

### Running Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View HTML report
go tool cover -html=coverage.out -o coverage.html

# Check coverage threshold
go tool cover -func=coverage.out | grep total
```

## CI/CD Integration

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: make test
      - run: make lint

  integration-tests:
    runs-on: ubuntu-latest
    services:
      mongodb:
        image: mongo:7.0
        ports:
          - 27017:27017
      kafka:
        image: confluentinc/cp-kafka:7.5.0
        ports:
          - 9092:9092
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test-integration
```

## Best Practices

1. **Test behavior, not implementation**
2. **Use table-driven tests for multiple cases**
3. **Keep tests independent and isolated**
4. **Use meaningful test names**
5. **Clean up resources after tests**
6. **Mock external dependencies**
7. **Test error paths, not just happy paths**
