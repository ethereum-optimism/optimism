# default recipe to display help information
default:
  @just --list

# Check for unused dependencies in the crate graph.
check-udeps:
  cargo +nightly udeps --workspace --lib --examples --tests --benches --all-features --locked

# Run unit tests with optional edge storage feature
test edge='':
  #!/usr/bin/env bash
  set -euo pipefail
  RUST_BACKTRACE=1 cargo nextest run \
    --features "asm-keccak {{edge}}" --locked \
    --workspace \
    --no-tests=warn \
    -E "!kind(test) and not binary(e2e_testsuite) and not test(test_online)"

# Run integration tests for reth-optimism-node
test-integration:
  RUST_BACKTRACE=1 cargo nextest run --locked -p reth-optimism-node

# Check Windows cross-compilation
check-windows:
  rustup target add x86_64-pc-windows-gnu
  cargo check -p op-reth --target x86_64-pc-windows-gnu

# Build all examples
examples:
  cargo build --examples --locked

# Install documentation dependencies
docs-install:
    cd docs/vocs && bun install

# Start the documentation development server
docs-dev: docs-install
    cd docs/vocs && bun run dev

# Build the documentation for production
docs-build: docs-install
    cd docs/vocs && bun run build

# Preview the built documentation
docs-preview: docs-build
    cd docs/vocs && bun run preview
