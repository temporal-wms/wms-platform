.PHONY: help build-all load-all deploy-all clean-images
.DEFAULT_GOAL := help

# Configuration
REGISTRY ?= wms
VERSION ?= latest
KIND_CLUSTER ?= kind

# List of all apps
APPS := consolidation dashboard facility inventory labor orders packing picking receiving routing shell shipping sortation stow walling waves

# Docker image names
IMAGES := $(foreach app,$(APPS),$(REGISTRY)/$(app):$(VERSION))

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "$(GREEN)WMS Frontend - Docker Build & Kind Upload$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Configuration:$(NC)"
	@echo "  REGISTRY=$(REGISTRY)"
	@echo "  VERSION=$(VERSION)"
	@echo "  KIND_CLUSTER=$(KIND_CLUSTER)"

# Build targets for individual apps
build-consolidation: ## Build consolidation app image
	@echo "$(GREEN)Building consolidation app...$(NC)"
	docker build -f apps/consolidation/Dockerfile -t $(REGISTRY)/consolidation:$(VERSION) --build-arg APP_NAME=consolidation .

build-dashboard: ## Build dashboard app image
	@echo "$(GREEN)Building dashboard app...$(NC)"
	docker build -f apps/dashboard/Dockerfile -t $(REGISTRY)/dashboard:$(VERSION) --build-arg APP_NAME=dashboard .

build-facility: ## Build facility app image
	@echo "$(GREEN)Building facility app...$(NC)"
	docker build -f apps/facility/Dockerfile -t $(REGISTRY)/facility:$(VERSION) --build-arg APP_NAME=facility .

build-inventory: ## Build inventory app image
	@echo "$(GREEN)Building inventory app...$(NC)"
	docker build -f apps/inventory/Dockerfile -t $(REGISTRY)/inventory:$(VERSION) --build-arg APP_NAME=inventory .

build-labor: ## Build labor app image
	@echo "$(GREEN)Building labor app...$(NC)"
	docker build -f apps/labor/Dockerfile -t $(REGISTRY)/labor:$(VERSION) --build-arg APP_NAME=labor .

build-orders: ## Build orders app image
	@echo "$(GREEN)Building orders app...$(NC)"
	docker build -f apps/orders/Dockerfile -t $(REGISTRY)/orders:$(VERSION) --build-arg APP_NAME=orders .

build-packing: ## Build packing app image
	@echo "$(GREEN)Building packing app...$(NC)"
	docker build -f apps/packing/Dockerfile -t $(REGISTRY)/packing:$(VERSION) --build-arg APP_NAME=packing .

build-picking: ## Build picking app image
	@echo "$(GREEN)Building picking app...$(NC)"
	docker build -f apps/picking/Dockerfile -t $(REGISTRY)/picking:$(VERSION) --build-arg APP_NAME=picking .

build-receiving: ## Build receiving app image
	@echo "$(GREEN)Building receiving app...$(NC)"
	docker build -f apps/receiving/Dockerfile -t $(REGISTRY)/receiving:$(VERSION) --build-arg APP_NAME=receiving .

build-routing: ## Build routing app image
	@echo "$(GREEN)Building routing app...$(NC)"
	docker build -f apps/routing/Dockerfile -t $(REGISTRY)/routing:$(VERSION) --build-arg APP_NAME=routing .

build-shell: ## Build shell app image
	@echo "$(GREEN)Building shell app...$(NC)"
	docker build -f apps/shell/Dockerfile -t $(REGISTRY)/shell:$(VERSION) --build-arg APP_NAME=shell .

build-shipping: ## Build shipping app image
	@echo "$(GREEN)Building shipping app...$(NC)"
	docker build -f apps/shipping/Dockerfile -t $(REGISTRY)/shipping:$(VERSION) --build-arg APP_NAME=shipping .

build-sortation: ## Build sortation app image
	@echo "$(GREEN)Building sortation app...$(NC)"
	docker build -f apps/sortation/Dockerfile -t $(REGISTRY)/sortation:$(VERSION) --build-arg APP_NAME=sortation .

build-stow: ## Build stow app image
	@echo "$(GREEN)Building stow app...$(NC)"
	docker build -f apps/stow/Dockerfile -t $(REGISTRY)/stow:$(VERSION) --build-arg APP_NAME=stow .

build-walling: ## Build walling app image
	@echo "$(GREEN)Building walling app...$(NC)"
	docker build -f apps/walling/Dockerfile -t $(REGISTRY)/walling:$(VERSION) --build-arg APP_NAME=walling .

build-waves: ## Build waves app image
	@echo "$(GREEN)Building waves app...$(NC)"
	docker build -f apps/waves/Dockerfile -t $(REGISTRY)/waves:$(VERSION) --build-arg APP_NAME=waves .

# Load targets for individual apps
load-consolidation: ## Load consolidation image to kind
	@echo "$(GREEN)Loading consolidation to kind...$(NC)"
	kind load docker-image $(REGISTRY)/consolidation:$(VERSION) --name $(KIND_CLUSTER)

load-dashboard: ## Load dashboard image to kind
	@echo "$(GREEN)Loading dashboard to kind...$(NC)"
	kind load docker-image $(REGISTRY)/dashboard:$(VERSION) --name $(KIND_CLUSTER)

load-facility: ## Load facility image to kind
	@echo "$(GREEN)Loading facility to kind...$(NC)"
	kind load docker-image $(REGISTRY)/facility:$(VERSION) --name $(KIND_CLUSTER)

load-inventory: ## Load inventory image to kind
	@echo "$(GREEN)Loading inventory to kind...$(NC)"
	kind load docker-image $(REGISTRY)/inventory:$(VERSION) --name $(KIND_CLUSTER)

load-labor: ## Load labor image to kind
	@echo "$(GREEN)Loading labor to kind...$(NC)"
	kind load docker-image $(REGISTRY)/labor:$(VERSION) --name $(KIND_CLUSTER)

load-orders: ## Load orders image to kind
	@echo "$(GREEN)Loading orders to kind...$(NC)"
	kind load docker-image $(REGISTRY)/orders:$(VERSION) --name $(KIND_CLUSTER)

load-packing: ## Load packing image to kind
	@echo "$(GREEN)Loading packing to kind...$(NC)"
	kind load docker-image $(REGISTRY)/packing:$(VERSION) --name $(KIND_CLUSTER)

load-picking: ## Load picking image to kind
	@echo "$(GREEN)Loading picking to kind...$(NC)"
	kind load docker-image $(REGISTRY)/picking:$(VERSION) --name $(KIND_CLUSTER)

load-receiving: ## Load receiving image to kind
	@echo "$(GREEN)Loading receiving to kind...$(NC)"
	kind load docker-image $(REGISTRY)/receiving:$(VERSION) --name $(KIND_CLUSTER)

load-routing: ## Load routing image to kind
	@echo "$(GREEN)Loading routing to kind...$(NC)"
	kind load docker-image $(REGISTRY)/routing:$(VERSION) --name $(KIND_CLUSTER)

load-shell: ## Load shell image to kind
	@echo "$(GREEN)Loading shell to kind...$(NC)"
	kind load docker-image $(REGISTRY)/shell:$(VERSION) --name $(KIND_CLUSTER)

load-shipping: ## Load shipping image to kind
	@echo "$(GREEN)Loading shipping to kind...$(NC)"
	kind load docker-image $(REGISTRY)/shipping:$(VERSION) --name $(KIND_CLUSTER)

load-sortation: ## Load sortation image to kind
	@echo "$(GREEN)Loading sortation to kind...$(NC)"
	kind load docker-image $(REGISTRY)/sortation:$(VERSION) --name $(KIND_CLUSTER)

load-stow: ## Load stow image to kind
	@echo "$(GREEN)Loading stow to kind...$(NC)"
	kind load docker-image $(REGISTRY)/stow:$(VERSION) --name $(KIND_CLUSTER)

load-walling: ## Load walling image to kind
	@echo "$(GREEN)Loading walling to kind...$(NC)"
	kind load docker-image $(REGISTRY)/walling:$(VERSION) --name $(KIND_CLUSTER)

load-waves: ## Load waves image to kind
	@echo "$(GREEN)Loading waves to kind...$(NC)"
	kind load docker-image $(REGISTRY)/waves:$(VERSION) --name $(KIND_CLUSTER)

# Deploy targets (build + load) for individual apps
deploy-consolidation: build-consolidation load-consolidation ## Build and load consolidation app
deploy-dashboard: build-dashboard load-dashboard ## Build and load dashboard app
deploy-facility: build-facility load-facility ## Build and load facility app
deploy-inventory: build-inventory load-inventory ## Build and load inventory app
deploy-labor: build-labor load-labor ## Build and load labor app
deploy-orders: build-orders load-orders ## Build and load orders app
deploy-packing: build-packing load-packing ## Build and load packing app
deploy-picking: build-picking load-picking ## Build and load picking app
deploy-receiving: build-receiving load-receiving ## Build and load receiving app
deploy-routing: build-routing load-routing ## Build and load routing app
deploy-shell: build-shell load-shell ## Build and load shell app
deploy-shipping: build-shipping load-shipping ## Build and load shipping app
deploy-sortation: build-sortation load-sortation ## Build and load sortation app
deploy-stow: build-stow load-stow ## Build and load stow app
deploy-walling: build-walling load-walling ## Build and load walling app
deploy-waves: build-waves load-waves ## Build and load waves app

# Batch operations
build-all: ## Build all app images
	@echo "$(GREEN)Building all apps...$(NC)"
	@for app in $(APPS); do \
		echo "$(YELLOW)Building $$app...$(NC)"; \
		docker build -f apps/$$app/Dockerfile -t $(REGISTRY)/$$app:$(VERSION) --build-arg APP_NAME=$$app . || exit 1; \
	done
	@echo "$(GREEN)All apps built successfully!$(NC)"

load-all: ## Load all images to kind
	@echo "$(GREEN)Loading all images to kind...$(NC)"
	@for app in $(APPS); do \
		echo "$(YELLOW)Loading $$app...$(NC)"; \
		kind load docker-image $(REGISTRY)/$$app:$(VERSION) --name $(KIND_CLUSTER) || exit 1; \
	done
	@echo "$(GREEN)All images loaded to kind successfully!$(NC)"

deploy-all: build-all load-all ## Build all apps and load to kind
	@echo "$(GREEN)All apps deployed to kind successfully!$(NC)"

# Utility targets
list-images: ## List all built images
	@echo "$(GREEN)Built images:$(NC)"
	@docker images | grep "$(REGISTRY)/" | grep "$(VERSION)"

clean-images: ## Remove all built images
	@echo "$(YELLOW)Removing all built images...$(NC)"
	@for app in $(APPS); do \
		docker rmi $(REGISTRY)/$$app:$(VERSION) 2>/dev/null || true; \
	done
	@echo "$(GREEN)Images cleaned!$(NC)"

check-kind: ## Check if kind cluster exists
	@kind get clusters | grep -q "^$(KIND_CLUSTER)$$" && echo "$(GREEN)Kind cluster '$(KIND_CLUSTER)' exists$(NC)" || (echo "$(YELLOW)Kind cluster '$(KIND_CLUSTER)' not found$(NC)" && exit 1)
