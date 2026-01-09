# WMS Platform Deployment Makefile
# Usage: make <target>

.PHONY: help all clean cluster-create cluster-delete \
        infra-deploy infra-mongodb infra-kafka infra-temporal \
        kong-deploy kong-undeploy kong-status kong-logs kong-test \
        observability-deploy observability-prometheus observability-dashboards \
        observability-red-dashboards observability-tracing-dashboards observability-events-dashboards \
        temporal-namespace wms-build wms-load wms-deploy wms-redeploy \
        data-mesh-deploy superset-deploy \
        status logs port-forward

# Default target
.DEFAULT_GOAL := help

# Variables
CLUSTER_NAME := wms-platform
NAMESPACE := wms-platform-dev
SCRIPT_DIR := ./scripts
HELM_DIR := ./helm
KIND_CONFIG := ./kind/kind-config.yaml

# Services list
SERVICES := order-service inventory-service waving-service routing-service \
            picking-service consolidation-service packing-service \
            shipping-service labor-service facility-service unit-service \
            process-path-service walling-service wes-service orchestrator \
            billing-service channel-service receiving-service seller-portal \
            seller-service sortation-service stow-service

#------------------------------------------------------------------------------
# Help
#------------------------------------------------------------------------------
help: ## Show this help message
	@echo "WMS Platform Deployment Commands"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Cluster Management:"
	@grep -E '^cluster-' $(MAKEFILE_LIST) | awk -F ':.*?## ' '{printf "  %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "Infrastructure:"
	@grep -E '^infra-' $(MAKEFILE_LIST) | awk -F ':.*?## ' '{printf "  %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "Kong API Gateway:"
	@grep -E '^kong-' $(MAKEFILE_LIST) | awk -F ':.*?## ' '{printf "  %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "Observability:"
	@grep -E '^observability-' $(MAKEFILE_LIST) | awk -F ':.*?## ' '{printf "  %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "Data Mesh:"
	@grep -E '^(data-mesh-|superset-)' $(MAKEFILE_LIST) | awk -F ':.*?## ' '{printf "  %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "Temporal:"
	@grep -E '^temporal-' $(MAKEFILE_LIST) | awk -F ':.*?## ' '{printf "  %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "WMS Platform:"
	@grep -E '^wms-' $(MAKEFILE_LIST) | awk -F ':.*?## ' '{printf "  %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "Utilities:"
	@grep -E '^(status|logs|port-forward|all|clean):' $(MAKEFILE_LIST) | awk -F ':.*?## ' '{printf "  %-25s %s\n", $$1, $$2}'

#------------------------------------------------------------------------------
# Complete Setup
#------------------------------------------------------------------------------
all: cluster-create repos infra-deploy kong-deploy observability-deploy temporal-namespace wms-build wms-load wms-deploy ## Complete setup: cluster + infra + kong + observability + WMS

clean: wms-undeploy cluster-delete ## Delete everything: WMS + cluster

#------------------------------------------------------------------------------
# Cluster Management
#------------------------------------------------------------------------------
cluster-create: ## Create Kind cluster
	@echo "Creating Kind cluster..."
	@if kind get clusters 2>/dev/null | grep -q "$(CLUSTER_NAME)"; then \
		echo "Cluster '$(CLUSTER_NAME)' already exists"; \
	else \
		kind create cluster --name $(CLUSTER_NAME) --config $(KIND_CONFIG); \
	fi

cluster-delete: ## Delete Kind cluster
	@echo "Deleting Kind cluster..."
	kind delete cluster --name $(CLUSTER_NAME)

cluster-info: ## Show cluster info
	kubectl cluster-info --context kind-$(CLUSTER_NAME)

#------------------------------------------------------------------------------
# Helm Repositories
#------------------------------------------------------------------------------
repos: ## Add and update Helm repositories
	@echo "Adding Helm repositories..."
	helm repo add bitnami https://charts.bitnami.com/bitnami 2>/dev/null || true
	helm repo add strimzi https://strimzi.io/charts/ 2>/dev/null || true
	helm repo add temporalio https://temporalio.github.io/helm-charts 2>/dev/null || true
	helm repo add grafana https://grafana.github.io/helm-charts 2>/dev/null || true
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
	helm repo update

#------------------------------------------------------------------------------
# Infrastructure
#------------------------------------------------------------------------------
infra-deploy: repos infra-mongodb infra-kafka infra-temporal ## Deploy all infrastructure (MongoDB, Kafka, Temporal)

infra-mongodb: ## Deploy MongoDB with replica set
	@echo "Deploying MongoDB..."
	kubectl create namespace mongodb 2>/dev/null || true
	helm upgrade --install mongodb oci://registry-1.docker.io/bitnamicharts/mongodb \
		-n mongodb \
		-f $(HELM_DIR)/infrastructure/mongodb/values-prod.yaml \
		--wait --timeout 5m

infra-kafka: ## Deploy Strimzi Kafka with KRaft
	@echo "Deploying Strimzi Kafka Operator..."
	kubectl create namespace kafka 2>/dev/null || true
	helm upgrade --install strimzi-kafka-operator strimzi/strimzi-kafka-operator \
		-n kafka \
		-f $(HELM_DIR)/infrastructure/kafka/strimzi-operator-values.yaml \
		--wait --timeout 5m
	@echo "Waiting for operator to be ready..."
	kubectl wait --for=condition=ready pod -l name=strimzi-cluster-operator -n kafka --timeout=120s
	@echo "Deploying Kafka cluster..."
	kubectl apply -f $(HELM_DIR)/infrastructure/kafka/kafka-cluster.yaml
	@echo "Waiting for Kafka cluster (this may take a few minutes)..."
	@sleep 30
	@kubectl apply -f $(HELM_DIR)/infrastructure/kafka/kafka-topics.yaml

infra-temporal: ## Deploy Temporal with PostgreSQL
	@echo "Deploying PostgreSQL for Temporal..."
	kubectl create namespace temporal 2>/dev/null || true
	helm upgrade --install temporal-postgresql oci://registry-1.docker.io/bitnamicharts/postgresql \
		-n temporal \
		-f $(HELM_DIR)/infrastructure/temporal/postgresql-values.yaml \
		--wait --timeout 5m
	@echo "Waiting for PostgreSQL..."
	kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql -n temporal --timeout=120s
	@sleep 10
	@echo "Deploying Temporal..."
	helm upgrade --install temporal temporalio/temporal \
		-n temporal \
		-f $(HELM_DIR)/infrastructure/temporal/temporal-values.yaml \
		--wait --timeout 10m

infra-undeploy: ## Remove all infrastructure
	helm uninstall temporal -n temporal 2>/dev/null || true
	helm uninstall temporal-postgresql -n temporal 2>/dev/null || true
	kubectl delete -f $(HELM_DIR)/infrastructure/kafka/kafka-topics.yaml 2>/dev/null || true
	kubectl delete -f $(HELM_DIR)/infrastructure/kafka/kafka-cluster.yaml 2>/dev/null || true
	helm uninstall strimzi-kafka-operator -n kafka 2>/dev/null || true
	helm uninstall mongodb -n mongodb 2>/dev/null || true

#------------------------------------------------------------------------------
# Kong API Gateway
#------------------------------------------------------------------------------
kong-deploy: repos ## Deploy Kong API Gateway with Gateway API support
	$(SCRIPT_DIR)/deploy-kong.sh values-kind.yaml

kong-deploy-prod: repos ## Deploy Kong API Gateway (production)
	$(SCRIPT_DIR)/deploy-kong.sh values.yaml

kong-undeploy: ## Remove Kong API Gateway
	$(SCRIPT_DIR)/deploy-kong.sh undeploy

kong-status: ## Show Kong and Gateway API status
	@echo "=== Kong Pods ==="
	@kubectl get pods -n kong 2>/dev/null || echo "Kong namespace not found"
	@echo ""
	@echo "=== Kong Services ==="
	@kubectl get svc -n kong 2>/dev/null || echo "Kong namespace not found"
	@echo ""
	@echo "=== Gateway Classes ==="
	@kubectl get gatewayclasses 2>/dev/null || echo "No GatewayClasses found"
	@echo ""
	@echo "=== Gateways ==="
	@kubectl get gateways -A 2>/dev/null || echo "No Gateways found"
	@echo ""
	@echo "=== HTTPRoutes ==="
	@kubectl get httproutes -n $(NAMESPACE) 2>/dev/null || echo "No HTTPRoutes found in $(NAMESPACE)"

kong-logs: ## Show Kong logs
	kubectl logs -f -l app.kubernetes.io/name=kong -n kong --tail=100

kong-test: ## Test Kong routing to WMS services
	@echo "Testing Kong API Gateway routing..."
	@echo ""
	@echo "Order Service:"
	@curl -s http://localhost:8888/api/order-service/v1/health 2>/dev/null || echo "  Failed - service may not be running"
	@echo ""
	@echo "Inventory Service:"
	@curl -s http://localhost:8888/api/inventory-service/v1/health 2>/dev/null || echo "  Failed - service may not be running"
	@echo ""
	@echo "Seller Service:"
	@curl -s http://localhost:8888/api/seller-service/v1/health 2>/dev/null || echo "  Failed - service may not be running"
	@echo ""
	@echo "Billing Service:"
	@curl -s http://localhost:8888/api/billing-service/v1/health 2>/dev/null || echo "  Failed - service may not be running"

#------------------------------------------------------------------------------
# Observability
#------------------------------------------------------------------------------
observability-deploy: repos ## Deploy observability stack (Loki, Tempo, Promtail, Grafana)
	$(SCRIPT_DIR)/deploy-observability.sh

observability-loki: ## Deploy Loki only
	$(SCRIPT_DIR)/deploy-observability.sh loki

observability-tempo: ## Deploy Tempo only
	$(SCRIPT_DIR)/deploy-observability.sh tempo

observability-grafana: ## Deploy Grafana only
	$(SCRIPT_DIR)/deploy-observability.sh grafana

observability-prometheus: ## Deploy Prometheus only
	$(SCRIPT_DIR)/deploy-observability.sh prometheus

observability-dashboards: ## Create Grafana log dashboards for all WMS services
	@echo "Creating Grafana log dashboards..."
	python3 ./o11y/logs/create-dashboards.py --clean

observability-red-dashboards: ## Create Grafana RED metrics dashboards for all WMS services
	@echo "Creating Grafana RED metrics dashboards..."
	python3 ./o11y/red-metrics/create-dashboards.py --clean

observability-tracing-dashboards: ## Create Grafana tracing dashboards for all WMS services
	@echo "Creating Grafana tracing dashboards..."
	python3 ./o11y/tracing/create-dashboards.py --clean

observability-events-dashboards: ## Create Grafana Kafka/event dashboards for all WMS services
	@echo "Creating Grafana event-driven dashboards..."
	python3 ./o11y/events/create-dashboards.py --clean

observability-failure-dashboards: ## Create Grafana failure metrics dashboards (retries, DLQ, workflow failures)
	@echo "Creating Grafana failure metrics dashboards..."
	python3 ./o11y/failure-metrics/create-dashboards.py --clean

observability-api-platform-dashboard: ## Deploy API Platform Overview dashboard (RED metrics for all services)
	@echo "Deploying API Platform Overview dashboard..."
	python3 ./o11y/api-platform/deploy-dashboard.py --clean

observability-events-platform-dashboard: ## Deploy Events Platform Overview dashboard (Kafka events for all services)
	@echo "Deploying Events Platform Overview dashboard..."
	python3 ./o11y/events-platform/deploy-dashboard.py --clean

observability-undeploy: ## Remove observability stack
	helm uninstall prometheus -n observability 2>/dev/null || true
	helm uninstall grafana -n observability 2>/dev/null || true
	helm uninstall promtail -n observability 2>/dev/null || true
	helm uninstall tempo -n observability 2>/dev/null || true
	helm uninstall loki -n observability 2>/dev/null || true

#------------------------------------------------------------------------------
# Data Mesh
#------------------------------------------------------------------------------
data-mesh-deploy: repos ## Deploy all data mesh components (MinIO, Trino, OpenMetadata, Superset)
	./data-mesh/scripts/deploy-data-mesh.sh all

superset-deploy: repos ## Deploy Apache Superset for order search dashboard
	./data-mesh/scripts/deploy-data-mesh.sh superset

superset-undeploy: ## Remove Apache Superset
	helm uninstall superset -n data-mesh 2>/dev/null || true

#------------------------------------------------------------------------------
# Temporal Namespace
#------------------------------------------------------------------------------
temporal-namespace: ## Create WMS namespace in Temporal
	$(SCRIPT_DIR)/create-temporal-namespace.sh

temporal-ui: ## Open Temporal UI in browser
	@echo "Opening Temporal UI..."
	@open http://localhost:8080 2>/dev/null || xdg-open http://localhost:8080 2>/dev/null || echo "Open http://localhost:8080 in your browser"

#------------------------------------------------------------------------------
# WMS Platform
#------------------------------------------------------------------------------
wms-build: ## Build all WMS Docker images
	@echo "Building WMS Docker images..."
	@for svc in $(SERVICES); do \
		if [ -d "../services/$$svc" ]; then \
			echo "Building $$svc..."; \
			docker build -t wms-platform/$$svc:latest -f ../services/$$svc/Dockerfile ..; \
		fi \
	done
	@if [ -d "../orchestrator" ]; then \
		echo "Building orchestrator..."; \
		docker build -t wms-platform/orchestrator:latest -f ../orchestrator/Dockerfile ..; \
	fi

wms-load: ## Load WMS Docker images into Kind
	@echo "Loading images into Kind..."
	@for svc in $(SERVICES); do \
		if docker image inspect wms-platform/$$svc:latest >/dev/null 2>&1; then \
			echo "Loading $$svc..."; \
			kind load docker-image wms-platform/$$svc:latest --name $(CLUSTER_NAME); \
		fi \
	done

wms-deploy: ## Deploy WMS platform via Helm
	@echo "Deploying WMS platform..."
	kubectl create namespace $(NAMESPACE) 2>/dev/null || true
	helm upgrade --install wms-platform $(HELM_DIR)/wms-platform \
		-n $(NAMESPACE) \
		-f $(HELM_DIR)/wms-platform/values-kind.yaml \
		--wait --timeout 5m

wms-redeploy: wms-build wms-load ## Rebuild and redeploy WMS services
	@echo "Redeploying WMS services..."
	kubectl rollout restart deployment -n $(NAMESPACE)

wms-undeploy: ## Remove WMS platform
	helm uninstall wms-platform -n $(NAMESPACE) 2>/dev/null || true

wms-restart: ## Restart all WMS pods
	kubectl rollout restart deployment -n $(NAMESPACE)

#------------------------------------------------------------------------------
# Utilities
#------------------------------------------------------------------------------
status: ## Show status of all deployments
	@echo "=== Kind Cluster ==="
	@kind get clusters 2>/dev/null || echo "No clusters found"
	@echo ""
	@echo "=== Namespaces ==="
	@kubectl get namespaces 2>/dev/null || echo "Cannot connect to cluster"
	@echo ""
	@echo "=== MongoDB ==="
	@kubectl get pods -n mongodb 2>/dev/null || echo "MongoDB namespace not found"
	@echo ""
	@echo "=== Kafka ==="
	@kubectl get pods -n kafka 2>/dev/null || echo "Kafka namespace not found"
	@echo ""
	@echo "=== Temporal ==="
	@kubectl get pods -n temporal 2>/dev/null || echo "Temporal namespace not found"
	@echo ""
	@echo "=== Observability ==="
	@kubectl get pods -n observability 2>/dev/null || echo "Observability namespace not found"
	@echo ""
	@echo "=== WMS Platform ==="
	@kubectl get pods -n $(NAMESPACE) 2>/dev/null || echo "WMS namespace not found"

logs: ## Show logs for a service (usage: make logs SVC=order-service)
	@if [ -z "$(SVC)" ]; then \
		echo "Usage: make logs SVC=<service-name>"; \
		echo "Available services: $(SERVICES)"; \
	else \
		kubectl logs -f -l app=$(SVC) -n $(NAMESPACE); \
	fi

port-forward: ## Set up port forwarding for all services
	@echo "Setting up port forwards..."
	@echo "Grafana: http://localhost:3000"
	@echo "Temporal UI: http://localhost:8080"
	@echo "MongoDB: localhost:27017"
	@echo ""
	@echo "Press Ctrl+C to stop"
	@kubectl port-forward svc/grafana 3000:3000 -n observability &
	@kubectl port-forward svc/temporal-web 8080:8080 -n temporal &
	@kubectl port-forward svc/mongodb 27017:27017 -n mongodb &
	@wait

grafana: ## Open Grafana in browser
	@echo "Opening Grafana..."
	@open http://localhost:3000 2>/dev/null || xdg-open http://localhost:3000 2>/dev/null || echo "Open http://localhost:3000 in your browser"
