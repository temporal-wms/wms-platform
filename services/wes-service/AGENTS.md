# Repository Guidelines

## Project Structure & Module Organization
Entry points live in `cmd/api` (HTTP server) and `cmd/worker` (Temporal workers). Shared business logic and orchestration code sits under `internal/` via slices such as `internal/domain`, `internal/application`, and `internal/workflows`. HTTP handlers live in `api/`, while specs (`openapi.yaml`, `asyncapi.yaml`) stay in `docs/`. Tests mirror production packages through `tests/domain`, `tests/activities`, `tests/workflows`, and `tests/integration`. Build artifacts live in `bin/`; `Dockerfile` defines the deployable container.

## Build, Test, and Development Commands
Requires Go 1.24+ plus local MongoDB, Kafka, and Temporal services.

```bash
go build -o bin/wes-service ./cmd/api         # compile API server
go build -o bin/wes-worker ./cmd/worker       # compile worker process
go run ./cmd/api                              # run API locally on :8016
go run ./cmd/worker                           # start Temporal workers
docker build -t wms/wes-service .             # build production image
```

## Coding Style & Naming Conventions
Stick to idiomatic Go with tabs, `camelCase` locals, `PascalCase` exported types, and short singular package names (`workflow`, `application`). Run `go fmt ./...` and `go vet ./...` before committing; both enforce formatting, import order, and obvious bug checks expected by CI. Temporal definitions should end with `_workflow.go` or `_activity.go` so they stay easy to trace.

## Testing Guidelines
Unit tests live beside the code they verify, follow the `*_test.go` pattern, and typically use `testify` suites for setup and assertions. Broader scenarios belong in `tests/`, where suites should start only the dependencies they require (Mongo for repositories, Kafka for outbox, Temporal for workflow replay). Common commands: `go test ./internal/domain/...` for pure domain logic, `go test ./tests/workflows/...` for orchestration, and `go test ./...` before pushing. Name tests after the aggregate or workflow plus the behavior (`TestTaskRoute_AdvancesStageOnComplete`).

## Commit & Pull Request Guidelines
Keep commit subjects short and imperative (`Trigger GitHub Pages deployment`) and add an optional scope prefix when touching multiple subsystems (`feature/routes: emit events`). Every PR should describe the behavior change, reference the tracking issue, attach proof of testing (`go test ./...`, worker logs, screenshots for docs), and note operational impact (new env vars or schema updates). Split restructures into preparatory commits so reviewers can focus on the functional change.

## Security & Configuration Tips
Keep secrets and `.env` files out of git; rely on the environment variables listed in `README.md` (`MONGODB_URI`, `PROCESS_PATH_SERVICE_URL`, `KAFKA_BROKERS`, etc.). Prefer shell exports or compose overrides for local tweaks instead of editing tracked manifests. Keep Mongo, Kafka, and Temporal on unique ports so workers stay healthy.
