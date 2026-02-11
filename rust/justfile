set positional-arguments

# Aliases
alias t := test
alias l := lint
alias f := fmt-fix
alias b := build

# default recipe to display help information
default:
  @just --list

############################### Build ###############################

# Build the workspace
build *args='':
  cargo build --workspace {{args}}

# Build the workspace in release mode
build-release *args='':
  cargo build --workspace --release {{args}}

# Build the rollup node
build-node:
  cargo build --release --bin kona-node

# Build the supervisor
build-supervisor:
  cargo build --release --bin kona-supervisor

# Build op-reth
build-op-reth:
  cargo build --release --bin op-reth

############################### Test ################################

# Run all tests (unit + doc tests)
test: test-unit test-docs

# Run unit tests (excluding online tests)
test-unit *args="-E '!test(test_online)'":
  cargo nextest run --workspace --all-features {{args}}

# Run online tests only
test-online:
  cargo nextest run --workspace --all-features -E 'test(test_online)'

# Run doc tests
test-docs:
  cargo test --doc --workspace --locked

############################### Lint ################################

# Run all lints
lint: fmt-check lint-clippy lint-docs

# Check formatting (requires nightly)
fmt-check:
  cargo +nightly fmt --all -- --check

# Fix formatting (requires nightly)
fmt-fix:
  cargo +nightly fmt --all

# Run clippy
lint-clippy:
  cargo clippy --workspace --all-features --all-targets -- -D warnings

# Lint Rust documentation
lint-docs:
  RUSTDOCFLAGS="-D warnings" cargo doc --workspace --no-deps --document-private-items

############################ no_std #################################

# Check no_std compatibility for proof, protocol, alloy, and op-alloy crates
check-no-std:
  #!/usr/bin/env bash
  set -euo pipefail

  no_std_packages=(
    # proof crates
    kona-executor
    kona-mpt
    kona-preimage
    kona-proof
    kona-proof-interop

    # protocol crates
    kona-genesis
    kona-hardforks
    kona-registry
    kona-protocol
    kona-derive
    kona-driver
    kona-interop

    # utilities
    kona-serde

    # alloy
    alloy-op-evm
    alloy-op-hardforks

    # op-alloy
    op-alloy
    op-alloy-consensus
    op-alloy-rpc-types
    op-alloy-rpc-types-engine
  )

  # We need to install the riscv32imac-unknown-none-elf target before starting to build the no-std crates.
  rustup target add riscv32imac-unknown-none-elf

  for package in "${no_std_packages[@]}"; do
    echo "Checking no_std build for: $package"
    cargo build -p "$package" --target riscv32imac-unknown-none-elf --no-default-features
    echo "Successfully checked no_std build for: $package"
  done

########################### Benchmarks ##############################

# Run benchmarks (compile only)
bench:
  cargo bench --no-run --workspace --features test-utils --exclude example-gossip --exclude example-discovery

########################## Misc tools ###############################

# Check for unused dependencies (requires nightly + cargo-udeps)
check-udeps:
  cargo +nightly udeps --release --workspace --all-features --all-targets

# Run cargo hack for feature powerset checking
hack partition="":
  #!/usr/bin/env bash
  set -euo pipefail
  cargo hack check --feature-powerset --depth 2 --no-dev-deps {{ if partition != "" { "--partition " + partition } else { "" } }}

######################### Kona Prestates ##############################

KONA_DIR := justfile_directory() / "kona"

# Build all kona prestates
build-kona-prestates: build-kona-cannon-prestate build-kona-interop-prestate

build-kona-cannon-prestate:
    @just build-kona-prestate kona-client prestate-artifacts-cannon

build-kona-interop-prestate:
    @just build-kona-prestate kona-client-int prestate-artifacts-cannon-interop

build-kona-prestate VARIANT OUTPUT_DIR:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "Building prestate for {{VARIANT}}..."
    cd "{{KONA_DIR}}/docker/fpvm-prestates"
    CANNON_TAG=$(cat "{{KONA_DIR}}/.config/cannon_tag")
    just cannon {{VARIANT}} "${CANNON_TAG}" "{{KONA_DIR}}/{{OUTPUT_DIR}}"

    cd "{{KONA_DIR}}"

    # Copy with hash-based name for challenger lookup
    HASH=$(jq -r .pre "{{OUTPUT_DIR}}/prestate-proof.json")
    cp "{{OUTPUT_DIR}}/prestate.bin.gz" "{{OUTPUT_DIR}}/${HASH}.bin.gz"
    echo "Prestate for {{VARIANT}}: ${HASH}"

build-kona-reproducible-prestate:
    @just build-kona-prestates

output-kona-prestate-hash:
    @echo "-------------------- Kona Prestates --------------------"
    @echo ""
    @echo "Cannon Absolute prestate hash:"
    @jq -r .pre {{KONA_DIR}}/prestate-artifacts-cannon/prestate-proof.json
    @echo ""
    @echo "Cannon Interop Absolute prestate hash:"
    @jq -r .pre {{KONA_DIR}}/prestate-artifacts-cannon-interop/prestate-proof.json
    @echo ""

reproducible-kona-prestate: build-kona-reproducible-prestate output-kona-prestate-hash

clean-kona-prestates:
  #!/usr/bin/env bash
  set -euo pipefail
  rm -rf "{{KONA_DIR}}/build"
  rm -rf "{{KONA_DIR}}/prestate-artifacts-cannon" "{{KONA_DIR}}/prestate-artifacts-cannon-interop"
