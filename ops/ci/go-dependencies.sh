#!/usr/bin/env bash
# Build the aggregate Circle Go workload's dependencies without executing tests.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
job="${1:?Pass a dependency name}"
export GOMODCACHE="$PWD/.ci/go-cache/full/modules" GOCACHE="$PWD/.ci/go-cache/full/dependencies"
mkdir -p "$GOMODCACHE" "$GOCACHE"
case "$job" in
  go)
    attempt=0
    until go mod download; do
      attempt=$((attempt + 1)); [[ "$attempt" -lt 5 ]] || exit 1
      sleep "$((2 ** attempt))"
    done
    just sync-superchain-go
    git diff --exit-code op-core/superchain/superchain-configs.zip.sha256
    just cannon
    (cd cannon && just diff-hello-elf)
    python3 ops/ci/go-artifacts.py pack go \
      op-core/superchain/superchain-configs.zip cannon/bin cannon/multicannon/embeds cannon/testdata/bin .ci/go-cache/full/modules
    ;;
  contracts)
    git submodule update --init --recursive -- packages/contracts-bedrock/lib
    export FOUNDRY_PROFILE=ci
    (cd packages/contracts-bedrock && just forge-build)
    (cd op-deployer && just copy-contract-artifacts)
    python3 ops/ci/go-artifacts.py pack contracts packages/contracts-bedrock/cache \
      packages/contracts-bedrock/artifacts packages/contracts-bedrock/forge-artifacts \
      op-deployer/pkg/deployer/artifacts/forge-artifacts
    ;;
  kona)
    export CARGO_HOME="$PWD/.ci/rust-cache/cargo" CARGO_TARGET_DIR="$PWD/rust/target"
    export CARGO_INCREMENTAL=0 RUSTC_WRAPPER=sccache SCCACHE_DIR="$PWD/.ci/rust-cache/sccache"
    export SCCACHE_BASEDIRS="$PWD" SCCACHE_CACHE_SIZE=10G SCCACHE_IDLE_TIMEOUT=0
    mkdir -p "$CARGO_HOME" "$SCCACHE_DIR"
    if [[ "${TARGET_CACHE_MODE:-keep}" == sccache-only ]]; then rm -rf rust/target; fi
    mkdir -p .ci/go-tests/dependencies/kona
    sccache --start-server
    sccache --zero-stats
    trap 'sccache --show-stats --stats-format json >.ci/go-tests/dependencies/kona/sccache.json; sccache --stop-server' EXIT
    (cd rust && mold -run cargo build --locked --profile release --features default \
      --package kona-host --package kona-client --package kona-node --package kona-sp1-proposer)
    python3 ops/ci/go-artifacts.py pack kona rust/target/release/kona-host rust/target/release/kona-client \
      rust/target/release/kona-node rust/target/release/kona-sp1-proposer
    ;;
  sp1-executor)
    export CARGO_HOME="$PWD/.ci/rust-cache/cargo" CARGO_TARGET_DIR="$PWD/rust/target"
    export CARGO_INCREMENTAL=0 RUSTC_WRAPPER=sccache SCCACHE_DIR="$PWD/.ci/rust-cache/sccache"
    export SCCACHE_BASEDIRS="$PWD" SCCACHE_CACHE_SIZE=10G SCCACHE_IDLE_TIMEOUT=0
    mkdir -p "$CARGO_HOME" "$SCCACHE_DIR" .ci/go-tests/dependencies/sp1-executor
    sccache --start-server
    sccache --zero-stats
    trap 'sccache --show-stats --stats-format json >.ci/go-tests/dependencies/sp1-executor/sccache.json; sccache --stop-server' EXIT
    (cd rust && mold -run cargo build --locked --profile release --all-features \
      --package kona-sp1-super-range-executor --bin kona-sp1-super-range-executor)
    mkdir -p .circleci-cache/rust-binaries
    cp rust/target/release/kona-sp1-super-range-executor .circleci-cache/rust-binaries/
    python3 ops/ci/go-artifacts.py pack sp1-executor .circleci-cache/rust-binaries/kona-sp1-super-range-executor
    ;;
  op-reth)
    bash /usr/local/lib/optimism-ci/op-reth-shadow.sh source
    bash /usr/local/lib/optimism-ci/op-reth-shadow.sh release-build
    python3 ops/ci/go-artifacts.py pack op-reth rust/target/release/op-reth rust/target/release/op-reth-sdm-fixture
    ;;
  prestate)
    just reproducible-prestate
    python3 ops/ci/go-artifacts.py pack prestate rust/kona/prestate-artifacts-*
    ;;
  *) echo 'Unknown dependency' >&2; exit 1 ;;
esac
