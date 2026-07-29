# OP Stack L2 Rollup Makefile

.PHONY: help setup up down logs clean reset

# Default target
help: ## Show this help message
	@echo "OP Stack L2 Rollup Management Commands:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

setup: ## Deploy L1 contracts and configure rollup components
	mkdir -p deployer
	@if [ ! -f op-deployer ]; then \
		echo "❌ Error: op-deployer not found. Run 'make download' first."; \
		exit 1; \
	fi
	@if [ ! -f .env ]; then \
		echo "❌ Error: .env file not found. Copy .example.env to .env and configure it."; \
		exit 1; \
	fi
	@echo "Running setup script..."
	@./scripts/setup-rollup.sh

up: ## Start all services
	docker-compose up -d --wait
	@make test-l2

down: ## Stop all services
	docker-compose down

logs: ## View logs from all services
	docker-compose logs -f

logs-%: ## View logs from a specific service (e.g., make logs-sequencer)
	docker-compose logs -f $*

status: ## Show status of all services
	docker-compose ps

restart: ## Restart all services
	docker-compose restart

restart-%: ## Restart a specific service (e.g., make restart-batcher)
	docker-compose restart $*

clean: ## Remove all containers and volumes
	@# Create minimal env files if they don't exist to avoid docker-compose errors
	@mkdir -p batcher proposer challenger sequencer
	@echo "# Minimal env for cleanup" > batcher/.env 2>/dev/null || true
	@echo "# Minimal env for cleanup" > proposer/.env 2>/dev/null || true
	@echo "# Minimal env for cleanup" > challenger/.env 2>/dev/null || true
	docker-compose down -v --remove-orphans 2>/dev/null || true

reset: ## Complete reset - removes all data and redeploys
	@echo "This will completely reset your L2 rollup deployment!"
	@read -p "Are you sure? (y/N) " confirm && [ "$$confirm" = "y" ] || exit 1
	make clean
	rm -rf deployer sequencer batcher proposer challenger
	@echo "Reset complete. Run 'make setup' to redeploy."

test-l2: ## Test L2 connectivity
	@echo "Testing L2 RPC connection..."
	@curl -s -X POST -H "Content-Type: application/json" \
		--data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
		http://localhost:8545 | jq -r '.result' 2>/dev/null && echo "✅ L2 connection successful" || echo "❌ L2 connection failed"

test-l1: ## Test L1 connectivity
	@echo "Testing L1 RPC connection..."
	@if [ -f .env ]; then \
		set -a; source .env; set +a; \
	fi; \
	curl -s -X POST -H "Content-Type: application/json" \
		--data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
		"$$L1_RPC_URL" | jq -r '.result' 2>/dev/null && echo "✅ L1 connection successful" || echo "❌ L1 connection failed"

deps: ## Install development dependencies with mise (optional)
	mise install

download: ## Download op-deployer binary
	./scripts/download-op-deployer.sh

init: ## Initialize the project
	./scripts/download-op-deployer.sh
	@if [ ! -f .env ]; then \
		cp .example.env .env; \
		echo "Please edit .env with your actual configuration values"; \
	else \
		echo ".env already exists, skipping copy"; \
	fi
	@echo "Then run: make setup"
