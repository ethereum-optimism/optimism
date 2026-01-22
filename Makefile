# Boba Makefile
#
# This Makefile wraps the upstream optimism build system with Boba-specific patches.
# It applies go.mod patches and overlays before building.
#
# Usage:
#   make build          - Build all Go binaries
#   make op-node        - Build specific target
#   make test           - Run tests
#   make clean          - Clean build artifacts
#   make submodule-update - Update upstream submodule
#

SHELL := /usr/bin/env bash
.PHONY: all build test clean submodule-update help patch-apply patch-restore

# Directories
REPO_ROOT := $(shell pwd)
SUBMODULE := $(REPO_ROOT)/optimism
OVERLAYS := $(REPO_ROOT)/overlays
BUILD_DIR := $(REPO_ROOT)/.boba-build

# Targets that are forwarded to the submodule Makefile
UPSTREAM_TARGETS := op-node op-batcher op-proposer op-challenger op-program \
                    op-dispute-mon op-conductor op-supervisor op-wheel \
                    op-deployer op-faucet cannon

# Default target
all: build

# Initialize submodules if needed
$(SUBMODULE)/go.mod:
	git submodule update --init --recursive

# Help
help:
	@echo "Boba Build System"
	@echo "================="
	@echo ""
	@echo "Targets:"
	@echo "  build              - Build all Go binaries (with Boba patches)"
	@echo "  test               - Run tests"
	@echo "  clean              - Clean build artifacts and restore patches"
	@echo "  submodule-update   - Update upstream submodule to latest"
	@echo "  patch-apply        - Apply Boba patches to submodule"
	@echo "  patch-restore      - Restore submodule to upstream state"
	@echo ""
	@echo "Individual component targets:"
	@echo "  $(UPSTREAM_TARGETS)"
	@echo ""
	@echo "Environment variables:"
	@echo "  BOBA_KEEP_PATCHES=1  - Don't restore patches after build"
	@echo "  BOBA_DRY_RUN=1       - Prepare but don't build"
	@echo ""

# Apply patches to submodule
patch-apply: $(SUBMODULE)/go.mod
	@echo "Applying Boba patches..."
	@mkdir -p $(BUILD_DIR)
	@# Backup go.mod and go.sum if not already done
	@if [ ! -f "$(BUILD_DIR)/go.mod.upstream" ]; then \
		cp "$(SUBMODULE)/go.mod" "$(BUILD_DIR)/go.mod.upstream"; \
	fi
	@if [ ! -f "$(BUILD_DIR)/go.sum.upstream" ]; then \
		cp "$(SUBMODULE)/go.sum" "$(BUILD_DIR)/go.sum.upstream"; \
	fi
	@# Apply go.mod patch
	@if [ -f "$(OVERLAYS)/go.mod.patch" ]; then \
		while IFS= read -r line; do \
			[[ "$$line" =~ ^#.*$$ ]] && continue; \
			[[ -z "$$line" ]] && continue; \
			if [[ "$$line" =~ ^s\| ]]; then \
				sed -i "$$line" "$(SUBMODULE)/go.mod"; \
			fi; \
		done < "$(OVERLAYS)/go.mod.patch"; \
	fi
	@# Apply contract overlays
	@if [ -d "$(OVERLAYS)/contracts-bedrock/src/boba" ]; then \
		mkdir -p "$(SUBMODULE)/packages/contracts-bedrock/src/boba"; \
		cp -r "$(OVERLAYS)/contracts-bedrock/src/boba/"* "$(SUBMODULE)/packages/contracts-bedrock/src/boba/"; \
	fi
	@if [ -d "$(OVERLAYS)/contracts-bedrock/deployments" ]; then \
		for dir in $(OVERLAYS)/contracts-bedrock/deployments/*/; do \
			dirname=$$(basename "$$dir"); \
			mkdir -p "$(SUBMODULE)/packages/contracts-bedrock/deployments/$$dirname"; \
			cp -r "$$dir"* "$(SUBMODULE)/packages/contracts-bedrock/deployments/$$dirname/"; \
		done; \
	fi
	@if [ -d "$(OVERLAYS)/contracts-bedrock/deploy-config" ]; then \
		mkdir -p "$(SUBMODULE)/packages/contracts-bedrock/deploy-config"; \
		cp "$(OVERLAYS)/contracts-bedrock/deploy-config/"*.json "$(SUBMODULE)/packages/contracts-bedrock/deploy-config/" 2>/dev/null || true; \
	fi
	@# Copy patched OpenZeppelin contracts (for regenesis support)
	@if [ -d "$(OVERLAYS)/contracts-bedrock/lib/openzeppelin-contracts-patched" ]; then \
		cp -r "$(OVERLAYS)/contracts-bedrock/lib/openzeppelin-contracts-patched" "$(SUBMODULE)/packages/contracts-bedrock/lib/"; \
	fi
	@# Add Boba remappings to foundry.toml
	@if [ -f "$(OVERLAYS)/foundry-remappings.txt" ]; then \
		if [ ! -f "$(BUILD_DIR)/foundry.toml.upstream" ]; then \
			cp "$(SUBMODULE)/packages/contracts-bedrock/foundry.toml" "$(BUILD_DIR)/foundry.toml.upstream"; \
		fi; \
		while IFS= read -r line; do \
			[[ "$$line" =~ ^#.*$$ ]] && continue; \
			[[ -z "$$line" ]] && continue; \
			if ! grep -q "$$line" "$(SUBMODULE)/packages/contracts-bedrock/foundry.toml"; then \
				sed -i "/^remappings = \[/a\  '$$line'," "$(SUBMODULE)/packages/contracts-bedrock/foundry.toml"; \
			fi; \
		done < "$(OVERLAYS)/foundry-remappings.txt"; \
	fi
	@# Copy kurtosis-devnet overlays (e.g., boba-local.yaml)
	@if [ -d "$(OVERLAYS)/kurtosis-devnet" ]; then \
		cp -r "$(OVERLAYS)/kurtosis-devnet/"* "$(SUBMODULE)/kurtosis-devnet/"; \
	fi
	@# Apply git patch files (exclude go.mod.patch which uses sed format)
	@for patchfile in $$(find "$(OVERLAYS)" -name "*.patch" -type f ! -name "go.mod.patch" 2>/dev/null); do \
		echo "Applying patch: $$patchfile"; \
		(cd "$(SUBMODULE)" && patch -p1 < "$$patchfile") || true; \
	done
	@echo "Patches applied."

# Restore submodule to upstream state
patch-restore:
	@echo "Restoring upstream state..."
	@if [ -f "$(BUILD_DIR)/go.mod.upstream" ]; then \
		cp "$(BUILD_DIR)/go.mod.upstream" "$(SUBMODULE)/go.mod"; \
	fi
	@if [ -f "$(BUILD_DIR)/go.sum.upstream" ]; then \
		cp "$(BUILD_DIR)/go.sum.upstream" "$(SUBMODULE)/go.sum"; \
	fi
	@if [ -f "$(BUILD_DIR)/foundry.toml.upstream" ]; then \
		cp "$(BUILD_DIR)/foundry.toml.upstream" "$(SUBMODULE)/packages/contracts-bedrock/foundry.toml"; \
	fi
	@# Remove overlay files from submodule
	@rm -rf "$(SUBMODULE)/packages/contracts-bedrock/src/boba" 2>/dev/null || true
	@rm -rf "$(SUBMODULE)/packages/contracts-bedrock/deployments/boba-"* 2>/dev/null || true
	@rm -f "$(SUBMODULE)/packages/contracts-bedrock/deploy-config/boba-"*.json 2>/dev/null || true
	@rm -rf "$(SUBMODULE)/packages/contracts-bedrock/lib/openzeppelin-contracts-patched" 2>/dev/null || true
	@rm -f "$(SUBMODULE)/kurtosis-devnet/boba-"*.yaml 2>/dev/null || true
	@# Revert any applied patches
	@(cd "$(SUBMODULE)" && git checkout -- . 2>/dev/null) || true
	@echo "Restored."

# Build all targets
build: patch-apply
	@echo "Building with Boba patches..."
	cd $(SUBMODULE) && go mod tidy && $(MAKE) build
	@if [ "$(BOBA_KEEP_PATCHES)" != "1" ]; then \
		$(MAKE) patch-restore; \
	fi

# Test
test: patch-apply
	cd $(SUBMODULE) && go mod tidy && $(MAKE) test
	@if [ "$(BOBA_KEEP_PATCHES)" != "1" ]; then \
		$(MAKE) patch-restore; \
	fi

# Individual component targets
$(UPSTREAM_TARGETS): patch-apply
	cd $(SUBMODULE) && go mod tidy && $(MAKE) $@
	@if [ "$(BOBA_KEEP_PATCHES)" != "1" ]; then \
		$(MAKE) patch-restore; \
	fi

# Clean
clean: patch-restore
	rm -rf $(BUILD_DIR)
	cd $(SUBMODULE) && $(MAKE) clean 2>/dev/null || true

# Update submodule
submodule-update:
	@echo "Updating upstream submodule..."
	cd $(SUBMODULE) && git fetch origin && git checkout origin/develop
	git add $(SUBMODULE)
	@echo "Submodule updated. Review changes and commit when ready."

# Pin submodule to specific tag/commit
# Usage: make submodule-pin REF=op-node/v1.16.0
submodule-pin:
ifndef REF
	$(error REF is required. Usage: make submodule-pin REF=op-node/v1.16.0)
endif
	@echo "Pinning upstream submodule to $(REF)..."
	cd $(SUBMODULE) && git fetch origin && git checkout $(REF)
	git add $(SUBMODULE)
	@echo "Submodule pinned to $(REF). Review changes and commit when ready."
