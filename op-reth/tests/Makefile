# Variables
DOCKER_IMAGE_NAME := op-reth
DOCKER_TAG := local
DOCKERFILE_PATH := ../../../DockerfileOpProof
KURTOSIS_PACKAGE := github.com/ethpandaops/optimism-package@998796c0f3bb478d63d729e65f0b76e24112e00d
DEVNET ?= opgeth-seq-opreth-val
GO_PKG_NAME ?= proofs/core
SOURCE_DIR := $(shell pwd)
OP_DEVSTACK_PROOF_SEQUENCER_EL ?= op-geth
OP_DEVSTACK_PROOF_VALIDATOR_EL ?= op-reth

.PHONY: all build-docker build-contracts unzip-contract-artifacts update-packages run clean help

# Default target
all: build-docker run

# Build op-reth
build:
	@echo "Building op-reth binary..."
	cd ../../../ && cargo build --bin op-reth --manifest-path crates/optimism/bin/Cargo.toml

# Build the op-reth Docker image
build-docker:
	@echo "Building $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) Docker image..."
	cd ../../../ && docker build -f $(notdir $(DOCKERFILE_PATH)) -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .

# Build coverage-enabled op-reth Docker image
build-docker-with-cov:
	@echo "Building coverage-enabled $(DOCKER_IMAGE_NAME):cov Docker image..."
	cd ../../../ && docker build \
	  --build-arg RUSTFLAGS="-Cinstrument-coverage" \
	  --build-arg CARGO_INCREMENTAL=0 \
	  --build-arg LLVM_PROFILE_FILE="/coverage/%m-%p.profraw" \
	  -f $(notdir $(DOCKERFILE_PATH)) \
	  -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .

# Run Kurtosis with the optimism devnet
run:
	@echo "Starting Optimism devnet with historical proof configuration..."
	@DEVNET_PATH="./devnets/$(DEVNET).yaml"; \
    if [ ! -z "$(DEVNET_CUSTOM_PATH)" ]; then \
        DEVNET_PATH="$(DEVNET_CUSTOM_PATH)"; \
    fi; \
    kurtosis run $(KURTOSIS_PACKAGE) --args-file $$DEVNET_PATH --enclave $(DEVNET)

# Build smart contract artifacts with Foundry
build-contracts:
	@echo "Building contracts with forge..."
	@cd "$(SOURCE_DIR)/proofs/contracts" && forge build || { echo "forge build failed"; exit 1; }

# Unzip contract artifacts
unzip-contract-artifacts:
	@echo "Unzipping contract artifacts..."
	mkdir -p "$(SOURCE_DIR)/artifacts/src"; \
	tar --zstd -xf "$(SOURCE_DIR)/artifacts/compressed/artifacts.tzst" -C "$(SOURCE_DIR)/artifacts/src"

# Update contract artifacts from the optimism submodule
update-packages:
	@echo "Updating contract artifacts from optimism submodule..."
	cd "$(SOURCE_DIR)/optimism/op-deployer" && just build-contracts copy-contract-artifacts
	mkdir -p "$(SOURCE_DIR)/artifacts/compressed"
	cp "$(SOURCE_DIR)/optimism/op-deployer/pkg/deployer/artifacts/forge-artifacts/artifacts.tzst" "$(SOURCE_DIR)/artifacts/compressed/artifacts.tzst"

# Run E2E tests using Kurtosis
test-e2e-kurtosis: build-contracts
	@echo "Running E2E tests with Kurtosis for $(DEVNET)"
	@DEVNET_PATH="$(SOURCE_DIR)/devnets/$(DEVNET).yaml"; \
    if [ ! -z "$(DEVNET_CUSTOM_PATH)" ]; then \
        DEVNET_PATH="$(DEVNET_CUSTOM_PATH)"; \
    fi; \
	export OP_DEPLOYER_ARTIFACTS="$(SOURCE_DIR)/artifacts/src/forge-artifacts"; \
	export DEVNET_ENV_URL="ktnative://$(DEVNET)$$DEVNET_PATH"; \
	export DISABLE_OP_E2E_LEGACY=true; \
	export DEVSTACK_ORCHESTRATOR=sysext; \
	go test -count=1 -timeout 40m -v ./$(GO_PKG_NAME)

# Run E2E tests using Sysgo
test-e2e-sysgo: unzip-contract-artifacts build-contracts
	@echo "Running E2E tests with Sysgo"
	export OP_DEPLOYER_ARTIFACTS="$(SOURCE_DIR)/artifacts/src/forge-artifacts"; \
	export DISABLE_OP_E2E_LEGACY=true; \
	export DEVSTACK_ORCHESTRATOR=sysgo; \
	export OP_RETH_ENABLE_PROOF_HISTORY=true; \
	export OP_RETH_EXEC_PATH="${SOURCE_DIR}/../../../target/debug/op-reth"; \
	export OP_DEVSTACK_PROOF_SEQUENCER_EL=$(OP_DEVSTACK_PROOF_SEQUENCER_EL); \
	export OP_DEVSTACK_PROOF_VALIDATOR_EL=$(OP_DEVSTACK_PROOF_VALIDATOR_EL); \
	go test -count=1 -timeout 40m -v ./$(GO_PKG_NAME)

# Stop and clean Kurtosis services
clean:
	@echo "Cleaning up Kurtosis services..."
	kurtosis clean -a
