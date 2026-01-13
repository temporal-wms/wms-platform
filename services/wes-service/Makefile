COVERPKG ?= ./internal/...
WES_BASE_URL ?= http://localhost:8016

.PHONY: test coverage integration-up integration-test integration-down

test:
	go test ./tests/...

coverage:
	go test ./tests/... -coverpkg=$(COVERPKG) -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -n 1

integration-up:
	docker compose -f docker-compose.integration.yml up -d --build

integration-test:
	WES_BASE_URL=$(WES_BASE_URL) go test -tags=integration ./tests/integration/...

integration-down:
	docker compose -f docker-compose.integration.yml down -v
