# E2e integration tests for kona.
import "./tests/justfile"
# Builds docker images for kona
import "./docker/apps/justfile"
# Vocs Documentation commands
import "./docs/justfile"

set positional-arguments
alias t := tests
alias la := lint-all
alias l := lint-native
alias lint := lint-native
alias f := fmt-native-fix
alias b := build-native
alias h := hack

# default recipe to display help information
default:
  @just --list

# Build the rollup node in a single command.
build-node:
  cargo build --release --bin kona-node

# Build the supervisor
build-supervisor:
  cargo build --release --bin kona-supervisor

# Run all tests (excluding online tests)
tests: test test-docs

# Test for the native target with all features. By default, excludes online tests.
test *args="-E '!test(test_online)'":
  cargo nextest run --workspace --all-features {{args}}
  just test-custom-embeds

# Run all online tests
test-online:
  just test "-E 'test(test_online)'"

# Test custom embedded chain configuration functionality
test-custom-embeds:
  cargo test --package kona-registry custom_chain_is_loaded_when_enabled \
      --config 'env.KONA_CUSTOM_CONFIGS="true"' \
      --config "env.KONA_CUSTOM_CONFIGS_DIR=\"{{justfile_directory()}}/crates/protocol/registry/tests/fixtures/custom\"" \
      --config 'env.KONA_CUSTOM_CONFIGS_TEST="true"'

# Runs the tests with llvm-cov
llvm-cov-tests:
  # collect coverage of `just test` and `just test-custom-embeds`
  cargo llvm-cov nextest --no-report --locked --workspace \
    --all-features \
    --exclude kona-node --exclude kona-p2p --exclude kona-sources \
    --ignore-run-fail --profile ci -E '!test(test_online)'

  cargo llvm-cov nextest --no-report --locked \
    --all-features \
    --ignore-run-fail --profile ci \
    --package kona-registry \
    -E 'test(custom_chain_is_loaded_when_enabled)' \
    --config 'env.KONA_CUSTOM_CONFIGS="true"' \
    --config "env.KONA_CUSTOM_CONFIGS_DIR=\"{{justfile_directory()}}/crates/protocol/registry/tests/fixtures/custom\"" \
    --config 'env.KONA_CUSTOM_CONFIGS_TEST="true"'

  cargo llvm-cov report --lcov --output-path lcov.info

# Runs benchmarks
benches:
  cargo bench --no-run --workspace --features test-utils --exclude example-gossip --exclude example-discovery

# Lint the workspace for all available targets
lint-all: lint-native lint-cannon lint-asterisc lint-docs lint-typos

# Check spelling with typos (`cargo install typos-cli`)
lint-typos:
  typos

# Runs `cargo hack check` against the workspace
hack:
  cargo hack check --feature-powerset --no-dev-deps

# Fixes the formatting of the workspace
fmt-native-fix:
  cargo +nightly fmt --all

# Check the formatting of the workspace
fmt-native-check:
  cargo +nightly fmt --all -- --check

# Lint the workspace
lint-native: fmt-native-check lint-docs
  cargo clippy --workspace --all-features --all-targets -- -D warnings

# Lint the workspace (mips arch). Currently, only the `kona-std-fpvm` crate is linted for the `cannon` target, as it is the only crate with architecture-specific code.
lint-cannon:
  docker run \
    --rm \
    -v `pwd`/:/workdir \
    -w="/workdir" \
    ghcr.io/op-rs/kona/cannon-builder:0.3.0 cargo clippy -p kona-std-fpvm --all-features -Zbuild-std=core,alloc -- -D warnings

# Lint the workspace (risc-v arch). Currently, only the `kona-std-fpvm` crate is linted for the `asterisc` target, as it is the only crate with architecture-specific code.
lint-asterisc:
  docker run \
    --rm \
    -v `pwd`/:/workdir \
    -w="/workdir" \
    ghcr.io/op-rs/kona/asterisc-builder:0.3.0 cargo clippy -p kona-std-fpvm --all-features -Zbuild-std=core,alloc -- -D warnings

# Lint the Rust documentation
lint-docs:
  RUSTDOCFLAGS="-D warnings" cargo doc --workspace --no-deps --document-private-items

# Test the Rust documentation
test-docs:
  cargo test --doc --workspace --locked

# Build for the native target
build-native *args='':
  cargo build --workspace $@

# Build `kona-client` for the `cannon` target.
build-cannon-client:
  docker run \
    --rm \
    -v `pwd`/:/workdir \
    -w="/workdir" \
    ghcr.io/op-rs/kona/cannon-builder:0.3.0 cargo build -Zbuild-std=core,alloc -p kona-client --bin kona-client --profile release-client-lto

# Build `kona-client` for the `asterisc` target.
build-asterisc-client:
  docker run \
    --rm \
    -v `pwd`/:/workdir \
    -w="/workdir" \
    ghcr.io/op-rs/kona/asterisc-builder:0.3.0 cargo build -Zbuild-std=core,alloc -p kona-client --bin kona-client --profile release-client-lto

# Check for unused dependencies in the crate graph.
check-udeps:
  cargo +nightly udeps --workspace --all-features --all-targets


# Updates the `superchain-registry` git submodule source
source-registry:
  @just --justfile ./crates/protocol/registry/justfile source

# Generate file bindings for super-registry
bind-registry:
  @just --justfile ./crates/protocol/registry/justfile bind
