#!/usr/bin/env bash
# Build one half of the default-feature release workspace, split as CircleCI splits it:
#   binaries   rust-e2e-binaries: every library, and the four binaries that the E2E
#              verdict tasks run. Packs those four binaries.
#   workspace  rust-workspace-release: every library, and every other binary.
# Both halves select the whole workspace, so features unify as in one
# `cargo build --workspace`. The verdict tasks wait only for the first half.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
part="${1:?Pass binaries or workspace}"
runtime=(op-reth op-reth-test-engine kona-node kona-host)
export CI_COMMIT_SHA="${CI_COMMIT_SHA:?Missing source SHA}"
export CARGO_HOME="$PWD/.ci/rust-cache/cargo" CARGO_TARGET_DIR="$PWD/rust/target"
export CARGO_INCREMENTAL=0 RUSTC_WRAPPER=sccache SCCACHE_DIR="$PWD/.ci/rust-cache/sccache"
export SCCACHE_BASEDIRS="$PWD" SCCACHE_CACHE_SIZE=10G SCCACHE_IDLE_TIMEOUT=0
mkdir -p "$CARGO_HOME" "$CARGO_TARGET_DIR" "$SCCACHE_DIR" .ci/go-tests/dependencies/rust-e2e-release
targets=(--lib)
case "$part" in
  binaries)
    for name in "${runtime[@]}"; do targets+=(--bin "$name"); done
    ;;
  workspace)
    # Assignments, so that a failure of either command stops the build.
    metadata="$(cd rust && cargo metadata --no-deps --format-version 1 --locked)"
    names="$(python3 -c 'import json, sys
for package in json.load(sys.stdin)["packages"]:
    for target in package["targets"]:
        if "bin" in target["kind"]: print(target["name"])' <<<"$metadata")"
    for name in $names; do
      case " ${runtime[*]} " in
        *" $name "*) ;;
        *) targets+=(--bin "$name") ;;
      esac
    done
    ;;
  *)
    echo 'Expected binaries or workspace' >&2
    exit 1
    ;;
esac
sccache --start-server
sccache --zero-stats
trap 'sccache --show-stats --stats-format json >.ci/go-tests/dependencies/rust-e2e-release/sccache.json; sccache --stop-server' EXIT
python3 ops/ci/runtime/op-reth-report.py prepare-superchain
python3 ops/ci/runtime/rust-target-cache.py prepare
# Like Circle's rust-build: --locked, and no LTO for release builds that only feed CI tests.
(cd rust && CARGO_PROFILE_RELEASE_LTO=off mold -run cargo build --locked --profile release --workspace --features default "${targets[@]}")
python3 ops/ci/runtime/rust-target-cache.py commit
if [[ "$part" == binaries ]]; then
  python3 ops/ci/runtime/go-artifacts.py pack rust-e2e-release "${runtime[@]/#/rust/target/release/}"
fi
