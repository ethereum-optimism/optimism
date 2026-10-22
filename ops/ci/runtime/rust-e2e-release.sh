#!/usr/bin/env bash
# Preserve Circle's entire default-feature workspace release build.
set -euo pipefail
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
export CI_COMMIT_SHA="${CI_COMMIT_SHA:-${CIRCLE_SHA1:?Missing source SHA}}"
report="$ROOT/.ci/rust-workspace/e2e-release"
rm -rf "$report"
mkdir -p "$report"
mkdir -p .ci/go-tests/dependencies/rust-e2e-release
export CARGO_INCREMENTAL=0
if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx ]]; then
  export CARGO_HOME="$ROOT/.ci/rust-cache/cargo" CARGO_TARGET_DIR="$ROOT/rust/target"
  export RUSTC_WRAPPER=sccache SCCACHE_DIR="$ROOT/.ci/rust-cache/sccache"
  export SCCACHE_BASEDIRS="$ROOT" SCCACHE_CACHE_SIZE=10G SCCACHE_IDLE_TIMEOUT=0
  mkdir -p "$CARGO_HOME" "$CARGO_TARGET_DIR" "$SCCACHE_DIR"
  sccache --start-server
  sccache --zero-stats
  python3 ops/ci/runtime/op-reth-report.py prepare-superchain "$report"
  python3 ops/ci/runtime/rust-target-cache.py prepare >"$report/cache-source.json"
fi
python3 ops/ci/runtime/rust-e2e-release-report.py begin "$report"
finish() {
  local status=$? diagnostics=0
  trap - EXIT
  if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx ]]; then
    sccache --show-stats --stats-format json >"$report/sccache.json" || diagnostics=$?
    sccache --stop-server >"$report/sccache-stop.log" 2>&1 || diagnostics=$?
    if [[ "$status" == 0 ]]; then python3 ops/ci/runtime/rust-target-cache.py commit || diagnostics=$?; fi
  fi
  if [[ "$status" == 0 && "$diagnostics" != 0 ]]; then status=$diagnostics; fi
  python3 ops/ci/runtime/rust-e2e-release-report.py finish "$report" "$status" || status=$?
  exit "$status"
}
trap finish EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
python3 ops/ci/runtime/rust-workspace-report.py json-stage "$report" workspace \
  cargo metadata --no-deps --locked --format-version 1
python3 ops/ci/runtime/rust-workspace-report.py json-stage "$report" build \
  mold -run cargo build --profile release --workspace --features default --message-format=json
python3 ops/ci/runtime/rust-e2e-release-report.py artifacts "$report"
paths=()
while IFS= read -r path; do paths+=("$path"); done < <(python3 ops/ci/runtime/rust-e2e-release-report.py binary-paths "$report")
[[ "${#paths[@]}" -gt 0 ]] || { echo 'Missing workspace release binaries' >&2; exit 1; }
python3 ops/ci/runtime/go-artifacts.py pack rust-e2e-release "${paths[@]}"
cp .ci/go-tests/dependencies/rust-e2e-release/metadata.json "$report/dependency.json"
