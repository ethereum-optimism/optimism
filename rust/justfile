set positional-arguments

NIGHTLY := "nightly-2026-02-20"

# Aliases
alias t := test
alias l := lint
alias f := fmt-fix
alias b := build

# default recipe to display help information
default:
  @just --list

############################### Toolchain ############################

# Install the pinned nightly toolchain
install-nightly:
  rustup toolchain install {{NIGHTLY}} --component rustfmt

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
  cargo +{{NIGHTLY}} fmt --all -- --check

# Fix formatting (requires nightly)
fmt-fix:
  cargo +{{NIGHTLY}} fmt --all

# Run clippy
lint-clippy:
  cargo clippy --workspace --all-features --all-targets -- -D warnings

# Lint Rust documentation
lint-docs:
  RUSTDOCFLAGS="--cfg docsrs -D warnings --show-type-layout --generate-link-to-definition -Zunstable-options" \
    cargo +{{NIGHTLY}} doc --workspace --all-features --no-deps --document-private-items

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
  cargo +{{NIGHTLY}} udeps --release --workspace --all-features --all-targets

# Run cargo hack for feature powerset checking
# shuffle: "true" to shuffle package order before partitioning (spreads heavy/light crates more evenly)
# seed: deterministic seed for shuffle (all partition nodes must use the same seed)
hack partition="" shuffle="false" seed="default":
  #!/usr/bin/env bash
  set -euo pipefail
  if [ "{{partition}}" != "" ]; then
    echo "Running cargo hack with partition {{partition}}"
  else
    echo "Running cargo hack without partition"
  fi

  PKG_FLAGS=""
  if [ "{{shuffle}}" = "true" ]; then
    PKGS=$(cargo metadata --no-deps --format-version 1 \
      | jq -r '.packages[].name' \
      | shuf --random-source=<(openssl enc -aes-256-ctr -pass "pass:{{seed}}" -nosalt </dev/zero 2>/dev/null))
    PKG_FLAGS=$(echo "$PKGS" | sed 's/^/-p /' | tr '\n' ' ')
    echo "Shuffled package order (seed={{seed}}): $PKGS"
  fi

  cargo hack check --each-feature --no-dev-deps $PKG_FLAGS {{ if partition != "" { "--partition " + partition } else { "" } }}

######################### Documentation ################################

DOCS_DIR := justfile_directory() / "docs"

# Start the documentation development server
docs-dev:
    cd "{{DOCS_DIR}}" && just docs-dev

# Build the documentation for production
docs-build:
    cd "{{DOCS_DIR}}" && just docs-build

# Preview the built documentation
docs-preview:
    cd "{{DOCS_DIR}}" && just docs-preview

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
    just cannon {{VARIANT}} "{{KONA_DIR}}/{{OUTPUT_DIR}}"

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
