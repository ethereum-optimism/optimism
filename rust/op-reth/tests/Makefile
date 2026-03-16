# Variables
GO_PKG_NAME ?= proofs/core
SOURCE_DIR := $(shell pwd)
OP_DEVSTACK_PROOF_SEQUENCER_EL ?= op-geth
OP_DEVSTACK_PROOF_VALIDATOR_EL ?= op-reth-with-proof

.PHONY: all build build-contracts build-bedrock-contracts test-e2e-sysgo

# Default target
all: build test-e2e-sysgo

# Build op-reth (release)
build:
	@echo "Building op-reth binary..."
	cd .. && cargo build --release --bin op-reth

# Build smart contract artifacts with Foundry
build-contracts:
	@echo "Building contracts with forge..."
	@cd "$(SOURCE_DIR)/proofs/contracts" && forge build || { echo "forge build failed"; exit 1; }

# Build contracts-bedrock forge artifacts (required by sysgo deployer)
build-bedrock-contracts:
	@echo "Building contracts-bedrock forge artifacts..."
	@cd "$(SOURCE_DIR)/../../.." && cd packages/contracts-bedrock && just install && just build-no-tests

# Run E2E tests using Sysgo
test-e2e-sysgo: build-contracts
	@echo "Running E2E tests with Sysgo"
	export DISABLE_OP_E2E_LEGACY=true; \
	export DEVSTACK_ORCHESTRATOR=sysgo; \
	export OP_RETH_ENABLE_PROOF_HISTORY=true; \
	export SKIP_P2P_CONNECTION_CHECK=true; \
	export OP_RETH_EXEC_PATH="${SOURCE_DIR}/../../target/release/op-reth"; \
	export OP_DEVSTACK_PROOF_SEQUENCER_EL=$(OP_DEVSTACK_PROOF_SEQUENCER_EL); \
	export OP_DEVSTACK_PROOF_VALIDATOR_EL=$(OP_DEVSTACK_PROOF_VALIDATOR_EL); \
	go test -count=1 -timeout 40m -v ./$(GO_PKG_NAME)
