#!/usr/bin/env bash
# Build the whole default-feature release workspace, which CircleCI splits across
# rust-e2e-binaries and rust-workspace-release. Pack the four binaries that the
# E2E verdict tasks run.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
export CI_COMMIT_SHA="${CI_COMMIT_SHA:?Missing source SHA}"
export CARGO_HOME="$PWD/.ci/rust-cache/cargo" CARGO_TARGET_DIR="$PWD/rust/target"
export CARGO_INCREMENTAL=0 RUSTC_WRAPPER=sccache SCCACHE_DIR="$PWD/.ci/rust-cache/sccache"
export SCCACHE_BASEDIRS="$PWD" SCCACHE_CACHE_SIZE=10G SCCACHE_IDLE_TIMEOUT=0
mkdir -p "$CARGO_HOME" "$CARGO_TARGET_DIR" "$SCCACHE_DIR" .ci/go-tests/dependencies/rust-e2e-release
sccache --start-server
sccache --zero-stats
trap 'sccache --show-stats --stats-format json >.ci/go-tests/dependencies/rust-e2e-release/sccache.json; sccache --stop-server' EXIT
python3 ops/ci/runtime/op-reth-report.py prepare-superchain
python3 ops/ci/runtime/rust-target-cache.py prepare
# Like Circle's rust-build: --locked, and no LTO for release builds that only feed CI tests.
(cd rust && CARGO_PROFILE_RELEASE_LTO=off mold -run cargo build --locked --profile release --workspace --features default)
python3 ops/ci/runtime/rust-target-cache.py commit
python3 ops/ci/runtime/go-artifacts.py pack rust-e2e-release rust/target/release/kona-host \
  rust/target/release/kona-node rust/target/release/op-reth rust/target/release/op-reth-test-engine
