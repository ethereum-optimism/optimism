#!/bin/bash
#
# Boba Build Script
#
# This script prepares the build environment by:
# 1. Applying Boba-specific patches to the upstream optimism submodule
# 2. Overlaying Boba-specific code (contracts, configs)
# 3. Building the specified targets
#
# Usage:
#   ./scripts/boba-build.sh [target...]
#
# Examples:
#   ./scripts/boba-build.sh              # Build all
#   ./scripts/boba-build.sh op-node      # Build op-node only
#   ./scripts/boba-build.sh op-batcher op-proposer  # Build multiple targets
#
# Environment variables:
#   BOBA_BUILD_DIR    - Build directory (default: .boba-build)
#   BOBA_CLEAN_BUILD  - Set to "1" to force a clean build
#   BOBA_DRY_RUN      - Set to "1" to only prepare, don't build
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SUBMODULE_DIR="$REPO_ROOT/optimism"
OVERLAYS_DIR="$REPO_ROOT/overlays"
BUILD_DIR="${BOBA_BUILD_DIR:-$REPO_ROOT/.boba-build}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    if [ ! -d "$SUBMODULE_DIR" ]; then
        log_error "Submodule not found at $SUBMODULE_DIR"
        log_error "Run: git submodule update --init"
        exit 1
    fi

    if [ ! -f "$SUBMODULE_DIR/go.mod" ]; then
        log_error "go.mod not found in submodule. Is it initialized?"
        exit 1
    fi
}

# Prepare build directory
prepare_build_dir() {
    if [ "${BOBA_CLEAN_BUILD:-0}" = "1" ] && [ -d "$BUILD_DIR" ]; then
        log_info "Cleaning build directory..."
        rm -rf "$BUILD_DIR"
    fi

    if [ ! -d "$BUILD_DIR" ]; then
        log_info "Creating build directory at $BUILD_DIR"
        mkdir -p "$BUILD_DIR"
    fi
}

# Create symlinks to upstream content
setup_upstream_links() {
    log_info "Setting up upstream content..."

    # For Go code, we'll work directly in the submodule directory
    # but we need to patch go.mod there

    # For packages (contracts), we need to overlay
    if [ ! -d "$BUILD_DIR/packages" ]; then
        mkdir -p "$BUILD_DIR/packages"
    fi
}

# Apply go.mod patch
apply_go_mod_patch() {
    local patch_file="$OVERLAYS_DIR/go.mod.patch"
    local target_go_mod="$SUBMODULE_DIR/go.mod"
    local backup_go_mod="$BUILD_DIR/go.mod.upstream"

    if [ ! -f "$patch_file" ]; then
        log_warn "No go.mod.patch found at $patch_file"
        return 0
    fi

    # Backup original if not already done
    if [ ! -f "$backup_go_mod" ]; then
        log_info "Backing up upstream go.mod..."
        cp "$target_go_mod" "$backup_go_mod"
    fi

    log_info "Applying go.mod patch..."

    # Apply sed patterns from patch file
    while IFS= read -r line; do
        # Skip comments and empty lines
        [[ "$line" =~ ^#.*$ ]] && continue
        [[ -z "$line" ]] && continue

        # Apply sed pattern
        if [[ "$line" =~ ^s\| ]]; then
            sed -i "$line" "$target_go_mod"
        fi
    done < "$patch_file"

    log_info "go.mod patched successfully"
}

# Restore original go.mod
restore_go_mod() {
    local target_go_mod="$SUBMODULE_DIR/go.mod"
    local backup_go_mod="$BUILD_DIR/go.mod.upstream"

    if [ -f "$backup_go_mod" ]; then
        log_info "Restoring upstream go.mod..."
        cp "$backup_go_mod" "$target_go_mod"
    fi
}

# Apply contract overlays
apply_contract_overlays() {
    local contracts_overlay="$OVERLAYS_DIR/contracts-bedrock"
    local contracts_target="$SUBMODULE_DIR/packages/contracts-bedrock"

    if [ ! -d "$contracts_overlay" ]; then
        log_info "No contract overlays found"
        return 0
    fi

    log_info "Applying contract overlays..."

    # Copy overlay files to submodule (will be gitignored)
    if [ -d "$contracts_overlay/src/boba" ]; then
        mkdir -p "$contracts_target/src/boba"
        cp -r "$contracts_overlay/src/boba/"* "$contracts_target/src/boba/"
        log_info "  - Copied boba contracts to src/boba/"
    fi

    if [ -d "$contracts_overlay/deployments" ]; then
        for dir in "$contracts_overlay/deployments/"*/; do
            dirname=$(basename "$dir")
            mkdir -p "$contracts_target/deployments/$dirname"
            cp -r "$dir"* "$contracts_target/deployments/$dirname/"
            log_info "  - Copied deployment config: $dirname"
        done
    fi

    if [ -d "$contracts_overlay/deploy-config" ]; then
        mkdir -p "$contracts_target/deploy-config"
        cp "$contracts_overlay/deploy-config/"*.json "$contracts_target/deploy-config/" 2>/dev/null || true
        log_info "  - Copied deploy-config files"
    fi
}

# Run go mod tidy in the submodule
run_go_mod_tidy() {
    log_info "Running go mod tidy..."
    cd "$SUBMODULE_DIR"
    go mod tidy
    cd "$REPO_ROOT"
}

# Build targets
build_targets() {
    local targets=("$@")

    if [ ${#targets[@]} -eq 0 ]; then
        log_info "Building all targets..."
        cd "$SUBMODULE_DIR"
        make build
    else
        cd "$SUBMODULE_DIR"
        for target in "${targets[@]}"; do
            log_info "Building $target..."
            make "$target"
        done
    fi

    cd "$REPO_ROOT"
}

# Cleanup handler
cleanup() {
    if [ "${BOBA_KEEP_PATCHES:-0}" != "1" ]; then
        restore_go_mod
    fi
}

# Main
main() {
    log_info "Boba Build Script"
    log_info "================="

    check_prerequisites
    prepare_build_dir

    # Set up trap for cleanup
    trap cleanup EXIT

    setup_upstream_links
    apply_go_mod_patch
    apply_contract_overlays

    if [ "${BOBA_DRY_RUN:-0}" = "1" ]; then
        log_info "Dry run complete. Build directory prepared at $BUILD_DIR"
        log_info "Submodule patched at $SUBMODULE_DIR"
        exit 0
    fi

    run_go_mod_tidy
    build_targets "$@"

    log_info "Build complete!"
}

main "$@"
