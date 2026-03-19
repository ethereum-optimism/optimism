# Heavily inspired by Lighthouse: https://github.com/sigp/lighthouse/blob/693886b94176faa4cb450f024696cb69cda2fe58/Makefile

GIT_SHA := `git rev-parse HEAD`
GIT_TAG := `git describe --tags --abbrev=0 2>/dev/null || echo "unknown"`
BIN_DIR := "dist/bin"
CARGO_TARGET_DIR := env("CARGO_TARGET_DIR", "target")
FEATURES := env("FEATURES", if os() == "windows" { "asm-keccak min-debug-logs" } else { "jemalloc asm-keccak min-debug-logs" })
PROFILE := env("PROFILE", "release")
CARGO_INSTALL_EXTRA_FLAGS := env("CARGO_INSTALL_EXTRA_FLAGS", "")
DOCKER_IMAGE_NAME := env("DOCKER_IMAGE_NAME", "us-docker.pkg.dev/oplabs-tools-artifacts/images/op-reth")

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

# Build and install the op-reth binary under `$CARGO_HOME/bin`
install:
  cargo install --path bin --bin op-reth --force --locked \
    --features "{{FEATURES}}" \
    --profile "{{PROFILE}}" \
    {{CARGO_INSTALL_EXTRA_FLAGS}}

# Build the op-reth binary into `target` directory
build:
  cargo build --bin op-reth --features "{{FEATURES}}" --profile "{{PROFILE}}"

# Build the op-reth binary into `target/debug` directory
build-debug:
  cargo build --bin op-reth --features "{{FEATURES}}"

# Build op-reth natively for the given target
build-native target:
  cargo build --bin op-reth --target {{target}} --features "{{FEATURES}}" --profile "{{PROFILE}}"

# Cross-compile op-reth for the given target (requires `cross` and Docker)
build-cross target:
  #!/usr/bin/env bash
  set -euo pipefail
  features="{{FEATURES}}"
  env_args=()
  # For aarch64, set the page size for jemalloc (64-KiB pages)
  if [[ "{{target}}" == "aarch64-unknown-linux-gnu" ]]; then
    env_args+=(JEMALLOC_SYS_WITH_LG_PAGE=16)
  fi
  # No jemalloc on Windows
  if [[ "{{target}}" == "x86_64-pc-windows-gnu" ]]; then
    features=$(echo "$features" | sed 's/jemalloc-prof//g; s/jemalloc//g' | xargs)
  fi
  env "${env_args[@]}" \
    RUSTFLAGS="-C link-arg=-lgcc -Clink-arg=-static-libgcc" \
    cross build --bin op-reth --target {{target}} --features "$features" --profile "{{PROFILE}}"

# Build op-reth for x86_64-apple-darwin (native)
build-x86_64-apple-darwin:
  just build-native x86_64-apple-darwin

# Build op-reth for aarch64-apple-darwin (native)
build-aarch64-apple-darwin:
  just build-native aarch64-apple-darwin

# Build op-reth for x86_64-unknown-linux-gnu (cross)
build-x86_64-unknown-linux-gnu:
  just build-cross x86_64-unknown-linux-gnu

# Build op-reth for aarch64-unknown-linux-gnu (cross)
build-aarch64-unknown-linux-gnu:
  just build-cross aarch64-unknown-linux-gnu

# Build op-reth for x86_64-pc-windows-gnu (cross)
build-x86_64-pc-windows-gnu:
  just build-cross x86_64-pc-windows-gnu

# Create release tarballs for supported targets
build-release-tarballs:
  #!/usr/bin/env bash
  set -euo pipefail
  mkdir -p {{BIN_DIR}}
  just build-x86_64-unknown-linux-gnu
  cp {{CARGO_TARGET_DIR}}/x86_64-unknown-linux-gnu/{{PROFILE}}/op-reth {{BIN_DIR}}/op-reth
  (cd {{BIN_DIR}} && tar -czf op-reth-{{GIT_TAG}}-x86_64-unknown-linux-gnu.tar.gz op-reth && rm op-reth)
  just build-aarch64-unknown-linux-gnu
  cp {{CARGO_TARGET_DIR}}/aarch64-unknown-linux-gnu/{{PROFILE}}/op-reth {{BIN_DIR}}/op-reth
  (cd {{BIN_DIR}} && tar -czf op-reth-{{GIT_TAG}}-aarch64-unknown-linux-gnu.tar.gz op-reth && rm op-reth)
  just build-x86_64-pc-windows-gnu
  cp {{CARGO_TARGET_DIR}}/x86_64-pc-windows-gnu/{{PROFILE}}/op-reth.exe {{BIN_DIR}}/op-reth.exe
  (cd {{BIN_DIR}} && tar -czf op-reth-{{GIT_TAG}}-x86_64-pc-windows-gnu.tar.gz op-reth.exe && rm op-reth.exe)

# Run unit tests
test-unit:
  cargo install cargo-nextest --locked
  cargo nextest run --locked --workspace --features 'jemalloc-prof' -E 'kind(lib)' -E 'kind(bin)' -E 'kind(proc-macro)'

# Run unit tests with coverage
cov-unit:
  rm -f lcov.info
  cargo llvm-cov nextest --lcov --output-path lcov.info --locked --workspace --features 'jemalloc-prof' -E 'kind(lib)' -E 'kind(bin)' -E 'kind(proc-macro)'

# Generate an HTML coverage report and open it in the browser
cov-report-html: cov-unit
  cargo llvm-cov report --html
  open target/llvm-cov/html/index.html

# Build and push a cross-arch Docker image with the given build and push tags
docker-build-push-tags build_tag push_tag features=FEATURES:
  #!/usr/bin/env bash
  set -euo pipefail
  FEATURES="{{features}}" just build-x86_64-unknown-linux-gnu
  mkdir -p {{BIN_DIR}}/amd64
  cp {{CARGO_TARGET_DIR}}/x86_64-unknown-linux-gnu/{{PROFILE}}/op-reth {{BIN_DIR}}/amd64/op-reth
  FEATURES="{{features}}" just build-aarch64-unknown-linux-gnu
  mkdir -p {{BIN_DIR}}/arm64
  cp {{CARGO_TARGET_DIR}}/aarch64-unknown-linux-gnu/{{PROFILE}}/op-reth {{BIN_DIR}}/arm64/op-reth
  docker buildx build --file ./DockerfileOp.cross . \
    --platform linux/amd64,linux/arm64 \
    --tag {{DOCKER_IMAGE_NAME}}:{{build_tag}} \
    --tag {{DOCKER_IMAGE_NAME}}:{{push_tag}} \
    --provenance=false \
    --push

# Build and push a cross-arch Docker image tagged with the latest git tag
docker-build-push:
  just docker-build-push-tags "{{GIT_TAG}}" "{{GIT_TAG}}"

# Build and push a cross-arch Docker image tagged with the latest git sha
docker-build-push-git-sha:
  just docker-build-push-tags "{{GIT_SHA}}" "{{GIT_SHA}}"

# Build and push a cross-arch Docker image tagged with the latest git tag and `latest`
docker-build-push-latest:
  just docker-build-push-tags "{{GIT_TAG}}" "latest"

# Build and push a cross-arch Docker image tagged nightly
docker-build-push-nightly:
  just docker-build-push-tags "nightly" "nightly"

# Build and push a cross-arch Docker image with edge features tagged nightly-edge-profiling
docker-build-push-nightly-edge-profiling:
  just docker-build-push-tags "nightly-edge-profiling" "nightly-edge-profiling" "{{FEATURES}} edge"

# Build and push a cross-arch Docker image with profiling profile tagged nightly-profiling
docker-build-push-nightly-profiling:
  just docker-build-push-tags "nightly-profiling" "nightly-profiling"

# Perform a `cargo` clean and remove the binary directory
clean:
  cargo clean
  rm -rf {{BIN_DIR}}

# Build op-reth with optimisations and symbols for profiling
profiling:
  RUSTFLAGS="-C target-cpu=native" cargo build --profile profiling --features jemalloc,asm-keccak --bin op-reth

# Build op-reth with the most aggressive optimisations
maxperf:
  RUSTFLAGS="-C target-cpu=native" cargo build --profile maxperf --features jemalloc,asm-keccak --bin op-reth

# Build op-reth with max optimisations minus asm-keccak
maxperf-no-asm:
  RUSTFLAGS="-C target-cpu=native" cargo build --profile maxperf --features jemalloc --bin op-reth

# Format Rust code with nightly
fmt:
  cargo +nightly fmt

# Run clippy lints
clippy:
  cargo +nightly clippy \
    --workspace \
    --lib \
    --examples \
    --tests \
    --benches \
    --all-features \
    -- -D warnings

# Check for typos in the codebase
lint-typos:
  #!/usr/bin/env bash
  set -euo pipefail
  if ! command -v typos &> /dev/null; then
    echo "typos not found. Please install it by running: cargo install typos-cli"
    echo "See: https://github.com/crate-ci/typos"
    exit 1
  fi
  typos

# Lint and format TOML files with dprint
lint-toml:
  #!/usr/bin/env bash
  set -euo pipefail
  if ! command -v dprint &> /dev/null; then
    echo "dprint not found. Please install it by running: cargo install --locked dprint"
    echo "See: https://github.com/dprint/dprint"
    exit 1
  fi
  dprint fmt

# Run all linters (fmt, clippy, typos, toml)
lint: fmt clippy lint-typos lint-toml

# Run clippy with auto-fix
clippy-fix:
  cargo +nightly clippy \
    --workspace \
    --lib \
    --examples \
    --tests \
    --benches \
    --all-features \
    --fix \
    --allow-staged \
    --allow-dirty \
    -- -D warnings

# Run clippy fix and format
fix-lint: clippy-fix fmt

# Generate Rust documentation
rustdocs:
  #!/usr/bin/env bash
  set -euo pipefail
  RUSTDOCFLAGS="\
    --cfg docsrs \
    --show-type-layout \
    --generate-link-to-definition \
    --enable-index-page -Zunstable-options -D warnings" \
    cargo +nightly doc \
    --document-private-items

# Run cargo tests across the workspace
cargo-test:
  cargo test \
    --workspace \
    --bin "op-reth" \
    --lib --examples \
    --tests \
    --benches \
    --all-features

# Run documentation tests
test-doc:
  cargo test --doc --workspace --all-features

# Run all tests (cargo + doc)
test-all: cargo-test test-doc

# Run lints, docs, and all tests
pr: lint rustdocs test-all
