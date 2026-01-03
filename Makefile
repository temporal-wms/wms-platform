.PHONY: all build test clean docker-build docker-up docker-down help

# Variables
SERVICES := order-service waving-service routing-service picking-service consolidation-service packing-service shipping-service inventory-service labor-service facility-service unit-service process-path-service
GO := go
DOCKER_COMPOSE := docker compose -f deployments/docker-compose.yml

# Colors
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

help: ## Display this help
	@echo "$(BLUE)WMS Platform - Warehouse Management System$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'

# =========================
# Development
# =========================

setup: ## Initial setup - install dependencies and tools
	@echo "$(GREEN)Setting up development environment...$(NC)"
	$(GO) work sync
	$(GO) mod download

build: ## Build all services
	@echo "$(GREEN)Building all services...$(NC)"
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		cd services/$$service && $(GO) build -o bin/api ./cmd/api && cd ../..; \
	done
	@echo "Building orchestrator..."
	cd orchestrator && $(GO) build -o bin/worker ./cmd/worker

build-service: ## Build a specific service (usage: make build-service SERVICE=order-service)
	@echo "$(GREEN)Building $(SERVICE)...$(NC)"
	cd services/$(SERVICE) && $(GO) build -o bin/api ./cmd/api

test: ## Run tests for all services
	@echo "$(GREEN)Running tests...$(NC)"
	$(GO) test ./...

test-service: ## Run tests for a specific service (usage: make test-service SERVICE=order-service)
	@echo "$(GREEN)Running tests for $(SERVICE)...$(NC)"
	cd services/$(SERVICE) && $(GO) test ./...

lint: ## Run linters
	@echo "$(GREEN)Running linters...$(NC)"
	golangci-lint run ./...

clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(NC)"
	@for service in $(SERVICES); do \
		rm -rf services/$$service/bin; \
	done
	rm -rf orchestrator/bin

# =========================
# Docker
# =========================

docker-infra: ## Start infrastructure only (MongoDB, Kafka, Temporal)
	@echo "$(GREEN)Starting infrastructure...$(NC)"
	$(DOCKER_COMPOSE) up -d mongodb kafka temporal temporal-ui kafka-ui

docker-infra-down: ## Stop infrastructure
	@echo "$(GREEN)Stopping infrastructure...$(NC)"
	$(DOCKER_COMPOSE) down

docker-build: ## Build all Docker images
	@echo "$(GREEN)Building Docker images...$(NC)"
	$(DOCKER_COMPOSE) build

docker-up: ## Start all services
	@echo "$(GREEN)Starting all services...$(NC)"
	$(DOCKER_COMPOSE) --profile full up -d

docker-up-core: ## Start core services only (order-service + infrastructure)
	@echo "$(GREEN)Starting core services...$(NC)"
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop all services
	@echo "$(GREEN)Stopping all services...$(NC)"
	$(DOCKER_COMPOSE) --profile full down

docker-logs: ## View logs for all services
	$(DOCKER_COMPOSE) logs -f

docker-logs-service: ## View logs for a specific service (usage: make docker-logs-service SERVICE=order-service)
	$(DOCKER_COMPOSE) logs -f $(SERVICE)

docker-clean: ## Remove all containers and volumes
	@echo "$(GREEN)Cleaning Docker resources...$(NC)"
	$(DOCKER_COMPOSE) --profile full down -v
	docker system prune -f

# =========================
# Kafka
# =========================

kafka-create-topics: ## Create Kafka topics
	@echo "$(GREEN)Creating Kafka topics...$(NC)"
	docker exec wms-kafka kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic wms.orders.inbound --partitions 12 --replication-factor 1
	docker exec wms-kafka kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic wms.orders.events --partitions 6 --replication-factor 1
	docker exec wms-kafka kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic wms.waves.events --partitions 6 --replication-factor 1
	docker exec wms-kafka kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic wms.picking.events --partitions 12 --replication-factor 1
	docker exec wms-kafka kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic wms.packing.events --partitions 6 --replication-factor 1
	docker exec wms-kafka kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic wms.shipping.events --partitions 6 --replication-factor 1
	docker exec wms-kafka kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic wms.inventory.events --partitions 6 --replication-factor 1
	docker exec wms-kafka kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic wms.labor.events --partitions 6 --replication-factor 1
	docker exec wms-kafka kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic wms.shipments.outbound --partitions 6 --replication-factor 1

kafka-list-topics: ## List all Kafka topics
	docker exec wms-kafka kafka-topics.sh --bootstrap-server localhost:9092 --list

# =========================
# MongoDB
# =========================

mongo-shell: ## Open MongoDB shell
	docker exec -it wms-mongodb mongosh

# =========================
# Temporal
# =========================

temporal-namespace: ## Create Temporal namespace
	@echo "$(GREEN)Creating Temporal namespace...$(NC)"
	docker exec wms-temporal temporal operator namespace create wms

# =========================
# Generators
# =========================

proto-gen: ## Generate protobuf code
	@echo "$(GREEN)Generating protobuf code...$(NC)"
	cd shared/pkg/proto && protoc --go_out=. --go-grpc_out=. *.proto

asyncapi-gen: ## Generate AsyncAPI documentation
	@echo "$(GREEN)Generating AsyncAPI documentation...$(NC)"
	npx @asyncapi/cli generate fromTemplate shared/api/asyncapi/wms-events.asyncapi.yaml @asyncapi/html-template -o docs/asyncapi

# =========================
# Run Locally
# =========================

run-order-service: ## Run order-service locally
	@echo "$(GREEN)Running order-service...$(NC)"
	cd services/order-service && $(GO) run ./cmd/api

run-orchestrator: ## Run orchestrator locally
	@echo "$(GREEN)Running orchestrator...$(NC)"
	cd orchestrator && $(GO) run ./cmd/worker

run-unit-service: ## Run unit-service locally
	@echo "$(GREEN)Running unit-service...$(NC)"
	cd services/unit-service && $(GO) run ./cmd/api

run-process-path-service: ## Run process-path-service locally
	@echo "$(GREEN)Running process-path-service...$(NC)"
	cd services/process-path-service && $(GO) run ./cmd/api

# =========================
# Contract Testing
# =========================

.PHONY: test-contracts test-pact test-openapi test-events test-provider pact-publish

test-contracts: test-pact test-openapi test-events ## Run all contract tests

test-pact: ## Run Pact consumer tests
	@echo "$(GREEN)Running Pact consumer tests...$(NC)"
	cd orchestrator && $(GO) test -v ./tests/contracts/consumer/...

test-openapi: ## Run OpenAPI validation tests
	@echo "$(GREEN)Running OpenAPI validation tests...$(NC)"
	cd orchestrator && $(GO) test -v ./tests/contracts/openapi/...

test-events: ## Run event schema validation tests
	@echo "$(GREEN)Running event schema validation tests...$(NC)"
	cd orchestrator && $(GO) test -v ./tests/contracts/events/...

test-provider-%: ## Run provider verification for a specific service (usage: make test-provider-order-service)
	@echo "$(GREEN)Running provider verification for $*...$(NC)"
	cd services/$* && $(GO) test -v ./tests/contracts/provider/...

test-providers: ## Run provider verification for all services
	@echo "$(GREEN)Running provider verification for all services...$(NC)"
	@for service in $(SERVICES); do \
		echo "Verifying $$service..."; \
		cd services/$$service && $(GO) test -v ./tests/contracts/provider/... || true && cd ../..; \
	done

pact-publish: ## Publish pacts to broker (if configured)
	@echo "$(GREEN)Publishing pacts...$(NC)"
	@if [ -z "$(PACT_BROKER_URL)" ]; then \
		echo "$(YELLOW)PACT_BROKER_URL not set, skipping publish$(NC)"; \
	else \
		pact-broker publish contracts/pacts \
			--consumer-app-version=$(shell git rev-parse --short HEAD) \
			--branch=$(shell git rev-parse --abbrev-ref HEAD) \
			--broker-base-url=$(PACT_BROKER_URL); \
	fi

# =========================
# Default
# =========================

all: setup build test ## Setup, build, and test all
