# Variables
DOCKER_IMAGE_NAME := op-reth
DOCKER_TAG := local
DOCKERFILE_PATH := ../../../DockerfileOpProof
KURTOSIS_PACKAGE := github.com/ethpandaops/optimism-package@998796c0f3bb478d63d729e65f0b76e24112e00d
DEVNET ?= opgeth-seq-opreth-val
GO_PKG_NAME ?= proofs/core
SOURCE_DIR := $(shell pwd)

.PHONY: all build build-contracts run clean help

# Default target
all: build run

# Build the op-reth Docker image
build:
	@echo "Building $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) Docker image..."
	cd ../../../ && docker build -f $(notdir $(DOCKERFILE_PATH)) -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .

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

# Run E2E tests using Kurtosis
test-e2e-kurtosis: build-contracts
	@echo "Running E2E tests with Kurtosis for $(DEVNET)"
	@DEVNET_PATH="$(SOURCE_DIR)/devnets/$(DEVNET).yaml"; \
    if [ ! -z "$(DEVNET_CUSTOM_PATH)" ]; then \
        DEVNET_PATH="$(DEVNET_CUSTOM_PATH)"; \
    fi; \
	export OP_DEPLOYER_ARTIFACTS="$(SOURCE_DIR)/artifacts"; \
	export DEVNET_ENV_URL="ktnative://$(DEVNET)$$DEVNET_PATH"; \
	export DISABLE_OP_E2E_LEGACY=true; \
	export DEVSTACK_ORCHESTRATOR=sysext; \
	go test -count=1 -timeout 40m -v ./$(GO_PKG_NAME)

# Stop and clean Kurtosis services
clean:
	@echo "Cleaning up Kurtosis services..."
	kurtosis clean -a
