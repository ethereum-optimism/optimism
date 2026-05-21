# Heavily inspired by Lighthouse: https://github.com/sigp/lighthouse/blob/stable/Makefile
# and Reth: https://github.com/paradigmxyz/reth/blob/main/Makefile
.DEFAULT_GOAL := help

GIT_VER ?= $(shell git describe --tags --always --dirty="-dev")
GIT_TAG ?= $(shell git describe --tags --abbrev=0)

FEATURES ?=

##@ Help
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: v
v: ## Show the current version
	@echo "Version: ${GIT_VER}"

##@ Build

.PHONY: clean
clean: ## Clean up
	cargo clean

.PHONY: build
build: ## Build (debug version)
	cargo build --features "$(FEATURES)"

.PHONY: docker-image
docker-image: ## Build a rollup-boost Docker image
	# The Dockerfile references the parent rust/ workspace via the named
	# `monorepo-rust` BuildKit context to resolve sibling path deps
	# (../op-reth/..., ../op-alloy/...). Requires buildx.
	docker buildx build --load \
		--platform linux/amd64 \
		--build-arg FEATURES="$(FEATURES)" \
		--build-context monorepo-rust=.. \
		-t rollup-boost \
		.

##@ Dev

.PHONY: lint
lint: ## Run the linters
	# Pinned to the monorepo's mise.toml nightly so local `make lint`
	# reproduces CI exactly (plain `+nightly` drifts as new lints land).
	cargo +nightly-2026-02-20 fmt -- --check
	cargo clippy --features "$(FEATURES)" -- -D warnings

.PHONY: test
test: ## Run the tests for rollup-boost
	# `test_invalid_args` shells out to `target/debug/rollup-boost` via
	# `assert_cmd::Command::cargo_bin`, which does not build the bin — so build
	# it up front before running the test suite.
	cargo build --bin rollup-boost --features "$(FEATURES)"
	cargo test --verbose --features "$(FEATURES)"

.PHONY: lt
lt: lint test ## Run "lint" and "test"

.PHONY: fmt
fmt: ## Format the code
	cargo +nightly fmt
	cargo fix --allow-staged
	cargo clippy --features "$(FEATURES)" --fix --allow-staged
