# Contributing to WMS Platform

Thank you for contributing to the WMS Platform. This guide will help you get started.

## Development Setup

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Make
- MongoDB (via Docker)
- Kafka (via Docker)
- Temporal (via Docker)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/your-org/wms-platform.git
cd wms-platform

# Start infrastructure
make infra-up

# Build all services
make build

# Run tests
make test
```

### Project Structure

```
wms-platform/
├── services/           # Microservices
│   ├── order-service/
│   ├── receiving-service/
│   ├── stow-service/
│   ├── sortation-service/
│   ├── facility-service/
│   └── ...
├── orchestrator/       # Temporal workflows
├── shared/             # Shared packages
├── docs/               # Documentation
└── ecosystem-documentation/  # Docusaurus docs
```

## Code Standards

### Go Style Guide

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` and `goimports` for formatting
- Run `golangci-lint` before committing

```bash
# Format code
make fmt

# Run linter
make lint
```

### Project Conventions

1. **Domain-Driven Design**: Each service follows DDD patterns
   - Aggregates in `internal/domain/`
   - Application services in `internal/application/`
   - Infrastructure in `internal/infrastructure/`

2. **Event-Driven Architecture**: Use CloudEvents format for all events

3. **API Design**: Follow REST conventions, use OpenAPI specs

### Commit Messages

Use conventional commits:

```
feat: add new putaway strategy
fix: resolve race condition in task assignment
docs: update receiving-service README
test: add integration tests for sortation
refactor: extract storage strategy interface
```

## Pull Request Process

1. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/your-feature
   ```

2. **Make your changes** following code standards

3. **Run tests**:
   ```bash
   make test
   make test-integration
   ```

4. **Update documentation** if needed

5. **Submit PR** with:
   - Clear description of changes
   - Link to related issues
   - Screenshots for UI changes

### PR Checklist

- [ ] Tests pass locally
- [ ] Linter passes
- [ ] Documentation updated
- [ ] Commit messages follow convention
- [ ] No breaking changes (or clearly documented)

## Testing Guidelines

See [Testing Strategy](docs/testing/README.md) for details.

### Running Tests

```bash
# Unit tests
make test

# Integration tests (requires Docker)
make test-integration

# Contract tests
make test-contracts

# All tests
make test-all
```

### Writing Tests

- Unit tests alongside code (`*_test.go`)
- Integration tests in `tests/integration/`
- Contract tests in `tests/contracts/`

## Service Development

### Creating a New Service

1. Copy service template:
   ```bash
   cp -r services/template services/new-service
   ```

2. Update `go.mod` and imports

3. Implement domain model in `internal/domain/`

4. Add API handlers in `internal/api/` or `cmd/api/`

5. Add documentation:
   - `README.md`
   - `docs/class-diagram.md`
   - `docs/ddd/aggregates.md`
   - `docs/openapi.yaml`
   - `docs/asyncapi.yaml`

### Adding Domain Events

1. Define event in `internal/domain/events.go`
2. Update AsyncAPI spec
3. Register event type in producer

## Infrastructure

### Running Locally

```bash
# Start all infrastructure
make infra-up

# Start specific services
docker compose up mongodb kafka temporal -d

# View logs
make logs
```

### Environment Variables

Each service uses environment variables for configuration. See service READMEs for details.

Common variables:
- `MONGODB_URI`
- `KAFKA_BROKERS`
- `OTEL_EXPORTER_OTLP_ENDPOINT`
- `LOG_LEVEL`

## Getting Help

- Open an issue for bugs or feature requests
- Join discussions for questions
- Review existing PRs for examples

## License

By contributing, you agree that your contributions will be licensed under the project's license.
