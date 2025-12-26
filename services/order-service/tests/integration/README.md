# Order Service Integration Tests

This directory contains integration tests for the order-service that use real MongoDB instances via testcontainers.

## Prerequisites

- Docker must be running on your machine
- Go 1.21 or later

## Running Tests

The integration tests use testcontainers-go to spin up real MongoDB instances. Due to Docker credential helper issues on macOS, you need to set specific environment variables:

```bash
# Run all integration tests
DOCKER_CONFIG=/dev/null TESTCONTAINERS_RYUK_DISABLED=true go test -v ./tests/integration/... -timeout 10m

# Run a specific test
DOCKER_CONFIG=/dev/null TESTCONTAINERS_RYUK_DISABLED=true go test -v ./tests/integration/... -run TestOrderRepository_Save -timeout 5m

# Run tests with coverage
DOCKER_CONFIG=/dev/null TESTCONTAINERS_RYUK_DISABLED=true go test -v ./tests/integration/... -cover -timeout 10m
```

## Test Structure

Each test function:
1. Spins up a fresh MongoDB container using testcontainers
2. Creates a repository instance connected to the test database
3. Runs test scenarios
4. Automatically cleans up the container on completion

## Test Coverage

The integration tests cover all repository methods:

- **Save**: Upsert operations for orders
- **FindByID**: Retrieve order by ID
- **FindByCustomerID**: Query orders by customer with pagination
- **FindByStatus**: Query orders by status with pagination
- **FindByWaveID**: Retrieve all orders in a wave
- **FindValidatedOrders**: Find orders ready for wave assignment
- **UpdateStatus**: Update order status
- **AssignToWave**: Assign validated orders to waves
- **Delete**: Soft delete (mark as cancelled)
- **Count**: Count orders with various filters

## Environment Variables

- `DOCKER_CONFIG=/dev/null`: Bypasses Docker credential helper issues on macOS
- `TESTCONTAINERS_RYUK_DISABLED=true`: Disables Ryuk container for cleanup (helps with credential issues)

## Troubleshooting

**Test timeout errors**: Ensure Docker Desktop is running and has sufficient resources allocated

**Container pull errors**: Check your internet connection and Docker Hub access

**Port conflicts**: The tests use random ports assigned by Docker, so conflicts are unlikely
