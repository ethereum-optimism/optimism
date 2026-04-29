set positional-arguments

NIGHTLY := `grep -oE 'nightly-[0-9]{4}-[0-9]{2}-[0-9]{2}' ../mise.toml | head -1`

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
  rustup toolchain install {{NIGHTLY}} --component rustfmt --component rust-src

############################### Build ###############################

# Build the workspace
build *args='':
  cargo build --workspace {{args}}

# Build the workspace in release mode
build-release *args='':
  cargo build --workspace --release {{args}}

# Build kona-node
build-kona-node:
  cargo build --release --bin kona-node

# Build kona-node in debug mode (faster compilation for local E2E test iteration)
build-kona-node-debug:
  cargo build --bin kona-node

alias build-node := build-kona-node

# Build op-reth
build-op-reth:
  cargo build --release --bin op-reth

# Build op-reth in debug mode (faster compilation for local E2E test iteration)
build-op-reth-debug:
  cargo build --bin op-reth

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
  cargo test --doc --workspace --locked --all-features

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

# Audit dependencies for licenses, advisories, and bans
deny:
  cargo deny --all-features check all

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
MIPS64_TARGET_SPEC := justfile_directory() / "kona/docker/cannon/mips64-unknown-none.json"

# Build kona-client for the MIPS64 cannon target
build-kona-client-elf VARIANT:
    #!/usr/bin/env bash
    set -euo pipefail

    # Ensure nightly toolchain with rust-src is installed
    just install-nightly

    # Cross-compilation environment
    export CC_mips64_unknown_none=mips64-linux-gnuabi64-gcc
    export CXX_mips64_unknown_none=mips64-linux-gnuabi64-g++
    export CARGO_TARGET_MIPS64_UNKNOWN_NONE_LINKER=mips64-linux-gnuabi64-gcc
    export RUSTFLAGS="-Clink-arg=-e_start -Cllvm-args=-mno-check-zero-division"
    export CARGO_BUILD_TARGET="{{MIPS64_TARGET_SPEC}}"
    export RUSTUP_TOOLCHAIN="{{NIGHTLY}}"

    # Custom configs support
    if [[ -n "${KONA_CUSTOM_CONFIGS_DIR:-}" ]]; then
      export KONA_CUSTOM_CONFIGS=true
    fi

    echo "Building kona-client ELF (variant: {{VARIANT}})..."
    cargo build \
      -Zbuild-std=core,alloc \
      -Zjson-target-spec \
      -p kona-client \
      --bin {{VARIANT}} \
      --locked \
      --profile release-client-lto

# Lint kona-std-fpvm for the MIPS64 cannon target
lint-kona-cannon:
    #!/usr/bin/env bash
    set -euo pipefail

    just install-nightly

    export CC_mips64_unknown_none=mips64-linux-gnuabi64-gcc
    export CXX_mips64_unknown_none=mips64-linux-gnuabi64-g++
    export CARGO_TARGET_MIPS64_UNKNOWN_NONE_LINKER=mips64-linux-gnuabi64-gcc
    export RUSTFLAGS="-Clink-arg=-e_start -Cllvm-args=-mno-check-zero-division"
    export CARGO_BUILD_TARGET="{{MIPS64_TARGET_SPEC}}"
    export RUSTUP_TOOLCHAIN="{{NIGHTLY}}"

    cargo clippy -p kona-std-fpvm --all-features -Zbuild-std=core,alloc -Zjson-target-spec -- -D warnings

# Build all kona prestates (runs natively — use build-kona-reproducible-prestate for Docker)
build-kona-prestates: build-kona-cannon-prestate build-kona-interop-prestate

build-kona-cannon-prestate:
    @just build-kona-prestate kona-client prestate-artifacts-cannon

build-kona-interop-prestate:
    @just build-kona-prestate kona-client-int prestate-artifacts-cannon-interop

# Build a single kona prestate variant
build-kona-prestate VARIANT OUTPUT_DIR:
    #!/usr/bin/env bash
    set -euo pipefail

    OUTPUT="{{KONA_DIR}}/{{OUTPUT_DIR}}"
    mkdir -p "$OUTPUT"

    echo "=== Building cannon ==="
    # cannon/justfile imports ../justfiles/go.just which imports git.just.
    # These relative imports resolve from cannon/'s directory, so we cd there
    # and call just directly — NOT via rust/justfile delegation.
    cd "{{justfile_directory()}}/../cannon"
    just cannon
    CANNON_BIN="$(pwd)/bin/cannon"

    echo "=== Building kona-client ELF (variant: {{VARIANT}}) ==="
    cd "{{justfile_directory()}}"
    just build-kona-client-elf {{VARIANT}}

    # Locate the built ELF
    ELF="{{justfile_directory()}}/target/mips64-unknown-none/release-client-lto/{{VARIANT}}"

    echo "=== Generating prestate ==="
    "$CANNON_BIN" load-elf \
      --path="$ELF" \
      --out="$OUTPUT/prestate.bin.gz" \
      --meta="$OUTPUT/meta.json" \
      --type multithreaded64-5

    "$CANNON_BIN" run \
      --proof-at "=0" \
      --stop-at "=1" \
      --input "$OUTPUT/prestate.bin.gz" \
      --meta "$OUTPUT/meta.json" \
      --proof-fmt "$OUTPUT/%d.json" \
      --output ""

    mv "$OUTPUT/0.json" "$OUTPUT/prestate-proof.json"

    # Copy with hash-based name for challenger lookup
    HASH=$(jq -r .pre "$OUTPUT/prestate-proof.json")
    cp "$OUTPUT/prestate.bin.gz" "$OUTPUT/${HASH}.bin.gz"
    echo "Prestate for {{VARIANT}}: ${HASH}"

# Build a single reproducible kona prestate variant via Docker.
# Cannon is built from source as a stage within the Dockerfile.
# Build context is the monorepo root (same pattern as op-program).
# Set KONA_CUSTOM_CONFIGS_DIR to bake custom chain configs into the prestate.
build-kona-reproducible-prestate-variant VARIANT OUTPUT_DIR:
    #!/usr/bin/env bash
    set -euo pipefail

    MONOREPO_ROOT="{{justfile_directory()}}/.."
    OUTPUT="{{KONA_DIR}}/{{OUTPUT_DIR}}"

    # The Dockerfile always copies from the `kona-custom-configs` named build
    # context, so point it at an empty temp dir when no configs are requested.
    if [[ -n "${KONA_CUSTOM_CONFIGS_DIR:-}" ]]; then
      if [[ ! -d "${KONA_CUSTOM_CONFIGS_DIR}" ]]; then
        echo "KONA_CUSTOM_CONFIGS_DIR=${KONA_CUSTOM_CONFIGS_DIR} is not a directory" >&2
        exit 1
      fi
      CUSTOM_CONFIGS_CONTEXT="${KONA_CUSTOM_CONFIGS_DIR}"
      CUSTOM_CONFIGS_FLAG=true
    else
      CUSTOM_CONFIGS_CONTEXT=$(mktemp -d)
      trap 'rm -rf "${CUSTOM_CONFIGS_CONTEXT}"' EXIT
      CUSTOM_CONFIGS_FLAG=false
    fi

    docker build \
      --platform linux/amd64 \
      --build-arg VARIANT={{VARIANT}} \
      --build-arg KONA_CUSTOM_CONFIGS="${CUSTOM_CONFIGS_FLAG}" \
      --build-context kona-custom-configs="${CUSTOM_CONFIGS_CONTEXT}" \
      --output "$OUTPUT" \
      --progress plain \
      -f "{{KONA_DIR}}/docker/fpvm-prestates/cannon-repro.dockerfile" \
      "$MONOREPO_ROOT"

    # Add hash-named copy for challenger lookup
    HASH=$(jq -r .pre "$OUTPUT/prestate-proof.json")
    cp "$OUTPUT/prestate.bin.gz" "$OUTPUT/${HASH}.bin.gz"
    echo "Prestate for {{VARIANT}}: ${HASH}"

# Build all reproducible kona prestates via Docker
build-kona-reproducible-prestate:
    @just build-kona-reproducible-prestate-variant kona-client prestate-artifacts-cannon
    @just build-kona-reproducible-prestate-variant kona-client-int prestate-artifacts-cannon-interop

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
