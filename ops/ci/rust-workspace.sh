#!/usr/bin/env bash
# Shared Circle/RWX commands and evidence. Run from the monorepo root.
set -euo pipefail
HELPERS="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
job="${1:?Pass a Rust workspace job}"
case "$job" in tests|tests-build|doctest|docs|clippy|build|features|feature-plan|no-std|udeps) ;;
  *) echo "Unknown Rust workspace job: $job" >&2; exit 1;;
esac
index="${CI_RUST_PARTITION_INDEX:-${CIRCLE_NODE_INDEX:-0}}"
total="${CI_RUST_PARTITION_TOTAL:-${CIRCLE_NODE_TOTAL:-10}}"
if [[ "$job" == features ]]; then
  [[ "$index" =~ ^[0-9]$ && "$total" == 10 ]] || {
    echo 'Expected an index from 0 to 9 in ten feature partitions.' >&2; exit 1;
  }
  report="$ROOT/.ci/rust-workspace/features-$index"
else
  report="$ROOT/.ci/rust-workspace/$job"
fi
rm -rf "$report"
mkdir -p "$report"
export CARGO_INCREMENTAL=0
if [[ "$job" == clippy ]]; then export RUSTFLAGS=-Dwarnings; fi
if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx ]]; then
  export CARGO_HOME="$ROOT/.ci/rust-cache/cargo" CARGO_TARGET_DIR="$ROOT/rust/target"
  export RUSTC_WRAPPER=sccache SCCACHE_DIR="$ROOT/.ci/rust-cache/sccache"
  export SCCACHE_CACHE_SIZE=10G SCCACHE_IDLE_TIMEOUT=0 SCCACHE_BASEDIRS="$ROOT" SCCACHE_LOG=warn
  mkdir -p "$CARGO_HOME" "$SCCACHE_DIR" "$CARGO_TARGET_DIR"
  sccache --start-server
  sccache --zero-stats
  python3 "$HELPERS/op-reth-report.py" prepare-superchain "$report"
  python3 "$HELPERS/rust-target-cache.py" prepare >"$report/cache-source.json"
fi
python3 "$HELPERS/rust-workspace-report.py" begin "$report" "$job"
finish() {
  local status=$? diagnostics=0
  trap - EXIT
  if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx ]]; then
    # Original verdicts are copied into the report artifact, never compiler caches.
    rm -rf "$ROOT/rust/target/nextest/default"
    sccache --show-stats --stats-format json >"$report/sccache.json" || diagnostics=$?
    sccache --stop-server >"$report/sccache-stop.log" 2>&1 || diagnostics=$?
    if [[ "$status" == 0 ]]; then
      python3 "$HELPERS/rust-target-cache.py" commit || diagnostics=$?
    fi
  fi
  if [[ "$status" == 0 && "$diagnostics" != 0 ]]; then status=$diagnostics; fi
  python3 "$HELPERS/rust-workspace-report.py" finish "$report" "$status" || status=$?
  exit "$status"
}
trap finish EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
stage() { python3 "$HELPERS/rust-workspace-report.py" stage "$report" "$@"; }
json_stage() { python3 "$HELPERS/rust-workspace-report.py" json-stage "$report" "$@"; }
json_stage workspace cargo metadata --no-deps --locked --all-features --format-version 1
docs() {
  stage doctests-list cargo test --doc --workspace --locked --all-features -- --list
  stage doctests just test-docs
}
features() {
  local partition="$1" phase
  for phase in features feature-tests; do
    local recipe=hack
    if [[ "$phase" == feature-tests ]]; then recipe=hack-tests-default; fi
    # In the pinned cargo-hack, dry-run printing does not advance partition
    # progress. Discover the whole plan, then verify live global indices.
    stage "$phase-list" just "$recipe" "" true "$seed" true
    if [[ "$job" != feature-plan ]]; then
      stage "$phase" just "$recipe" "$partition" true "$seed"
    fi
  done
}
seed="$(git rev-parse HEAD)"
case "$job" in
  tests-build)
    stage archive mold -run cargo nextest archive --workspace --all-features --locked --archive-file "$report/tests.tar.zst"
    # Compile the extra test once; its fresh verdict remains in the runtime task.
    stage beacon-build cargo test --profile fast-build --locked -p kona-providers-alloy \
      test_filtered_beacon_blobs_deserializes_on_small_stack --no-run
    python3 "$HELPERS/rust-workspace-report.py" artifact "$report"
    ;;
  tests)
    rm -f "$ROOT/rust/target/nextest/default/junit.xml"
    args=()
    if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx ]]; then
      producer="${TEST_ARCHIVE:?TEST_ARCHIVE must identify the compiled producer artifact}"
      python3 "$HELPERS/rust-workspace-report.py" verify-artifact "$producer"
      mkdir -p "$report/unpacked"
      tar --zstd -xf "$producer/tests.tar.zst" -C "$report/unpacked"
      args=(--binaries-metadata "$report/unpacked/target/nextest/binaries-metadata.json"
        --cargo-metadata "$report/unpacked/target/nextest/cargo-metadata.json"
        --target-dir-remap "$report/unpacked/target" --workspace-remap "$ROOT/rust")
    else
      args=(--workspace --all-features)
    fi
    json_stage unit-list cargo nextest list "${args[@]}" -E '!test(test_online)' --message-format json
    status=0
    if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx ]]; then
      stage unit cargo nextest run "${args[@]}" -E '!test(test_online)' || status=$?
    else
      stage unit just test-unit || status=$?
    fi
    if [[ -f "$ROOT/rust/target/nextest/default/junit.xml" ]]; then
      cp "$ROOT/rust/target/nextest/default/junit.xml" "$report/junit.xml"
    fi
    rm -rf "$report/unpacked"
    if [[ "$status" != 0 ]]; then exit "$status"; fi
    stage beacon-list cargo test --profile fast-build --locked -p kona-providers-alloy \
      test_filtered_beacon_blobs_deserializes_on_small_stack -- --list
    stage beacon just test-beacon-blob-stack
    docs
    ;;
  doctest) docs ;;
  docs) stage docs just lint-docs ;;
  clippy) stage clippy cargo clippy --workspace --all-targets --all-features --locked ;;
  build) stage build mold -run cargo build --profile dev --workspace --features default ;;
  features) features "$((index + 1))/$total" ;;
  feature-plan) features "" ;;
  no-std) stage no-std just check-no-std ;;
  udeps) stage udeps just check-udeps ;;
esac
