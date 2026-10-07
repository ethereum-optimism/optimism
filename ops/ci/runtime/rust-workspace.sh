#!/usr/bin/env bash
# Shared Circle/RWX commands and evidence. Run from the monorepo root.
set -euo pipefail
HELPERS="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
job="${1:?Pass a Rust workspace job}"
case "$job" in tests|tests-build|tests-unit|tests-compiler|tests-merge|doctest|docs|clippy|build|features|feature-plan|no-std|udeps|wasm-unknown|wasm-wasi|zepter|typos|registry|interop) ;;
  *) echo "Unknown Rust workspace job: $job" >&2; exit 1;;
esac
index="${CI_RUST_PARTITION_INDEX:-${CIRCLE_NODE_INDEX:-0}}"
total="${CI_RUST_PARTITION_TOTAL:-${CIRCLE_NODE_TOTAL:-10}}"
if [[ "$job" == features ]]; then
  if ! [[ "$index" =~ ^(0|[1-9][0-9]*)$ && "$total" =~ ^[1-9][0-9]*$ ]] || (( index >= total )); then
    echo 'Expected a nonnegative feature index below a positive partition total.' >&2
    exit 1
  fi
  report="$ROOT/.ci/rust-workspace/features-$index"
elif [[ "$job" == tests-merge ]]; then
  report="$ROOT/.ci/rust-workspace/tests"
else
  report="$ROOT/.ci/rust-workspace/$job"
fi
rm -rf "$report"
mkdir -p "$report"
export CARGO_INCREMENTAL=0
if [[ "$job" == clippy ]]; then export RUSTFLAGS=-Dwarnings; fi
if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx ]]; then
  # Trial incremental host checks/test compilation; release and prestate
  # producers keep their existing policy. Registry dependencies still use sccache.
  case "$job" in
    features|tests-build|tests|tests-unit|tests-compiler|tests-merge)
      case "${CI_RUST_INCREMENTAL:-0}" in
        0) ;;
        1)
          # sccache rejects the global CARGO_INCREMENTAL=1 switch. Cargo's
          # profile settings enable local incremental invocations instead;
          # sccache passes those through and caches nonincremental dependencies.
          unset CARGO_INCREMENTAL CARGO_BUILD_INCREMENTAL
          export CARGO_PROFILE_DEV_INCREMENTAL=true CARGO_PROFILE_TEST_INCREMENTAL=true
          export CARGO_PROFILE_FAST_BUILD_INCREMENTAL=true
          ;;
        *) echo 'CI_RUST_INCREMENTAL must be 0 or 1.' >&2; exit 1 ;;
      esac
      ;;
  esac
  export CARGO_HOME="$ROOT/.ci/rust-cache/cargo" CARGO_TARGET_DIR="$ROOT/rust/target"
  export RUSTC_WRAPPER=sccache SCCACHE_DIR="$ROOT/.ci/rust-cache/sccache"
  export SCCACHE_CACHE_SIZE=10G SCCACHE_IDLE_TIMEOUT=0 SCCACHE_BASEDIRS="$ROOT" SCCACHE_LOG=warn
  mkdir -p "$CARGO_HOME" "$SCCACHE_DIR" "$CARGO_TARGET_DIR"
  if [[ "$job" != tests-merge ]]; then
    sccache --start-server
    sccache --zero-stats
  fi
fi
report_job="$job"
if [[ "$job" == tests-merge ]]; then report_job=tests; fi
python3 "$HELPERS/rust-workspace-report.py" begin "$report" "$report_job"
finish() {
  local status=$? diagnostics=0
  trap - EXIT
  if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx && "$job" != tests-merge ]]; then
    # Original verdicts are copied into the report artifact, never compiler caches.
    rm -rf "$ROOT/rust/target/nextest/default"
    sccache --show-stats --stats-format json >"$report/sccache.json" || diagnostics=$?
    sccache --stop-server >"$report/sccache-stop.log" 2>&1 || diagnostics=$?
    if [[ "$status" == 0 ]]; then
      python3 "$HELPERS/rust-target-cache.py" commit >"$report/cache-publication.json" || diagnostics=$?
      python3 "$HELPERS/rust-target-cache.py" size | tee "$report/cache-size.json" || diagnostics=$?
      if [[ "$diagnostics" == 0 && -n "${packed_target:-}" ]]; then
        stage_at . cache-pack python3 "$HELPERS/rust-target-cache.py" pack "$packed_target" || diagnostics=$?
      fi
    fi
  fi
  if [[ "$status" == 0 && "$diagnostics" != 0 ]]; then status=$diagnostics; fi
  local finish_mode=finish
  if [[ "$job" == tests-merge ]]; then finish_mode=finish-merged-tests; fi
  python3 "$HELPERS/rust-workspace-report.py" "$finish_mode" "$report" "$status" || status=$?
  exit "$status"
}
trap finish EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
stage() { python3 "$HELPERS/rust-workspace-report.py" stage "$report" "$@"; }
stage_at() { python3 "$HELPERS/rust-workspace-report.py" stage-at "$report" "$@"; }
json_stage() { python3 "$HELPERS/rust-workspace-report.py" json-stage "$report" "$@"; }
if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx && "$job" != tests-merge ]]; then
  case "$job" in
    tests-build)
      packed_target="$ROOT/.ci/rust-cache/target-cache.tar.zst"
      # This immutable artifact seeds a first verdict. RWX retains the individual
      # target files; the archive is excluded from filesystem cache outputs.
      ;;
    tests|tests-compiler)
      # Prefer the runtime's protected baseline. Seed an empty target from the
      # current producer, without inheriting the producer's cache layer history.
      if [[ ! -f "$CARGO_TARGET_DIR/.rwx-source-fingerprint.json" &&
            ! -f "$CARGO_TARGET_DIR/.rwx-source-pending.json" ]]; then
        stage_at . cache-restore python3 "$HELPERS/rust-target-cache.py" restore "${COMPILED_TARGET:?COMPILED_TARGET must identify the producer artifact}"
      fi
      ;;
  esac
  python3 "$HELPERS/op-reth-report.py" prepare-superchain "$report"
  python3 "$HELPERS/rust-target-cache.py" prepare >"$report/cache-source.json"
fi
json_stage workspace cargo metadata --no-deps --locked --all-features --format-version 1
compiler_tests() {
  stage beacon-list cargo test --profile fast-build --locked -p kona-providers-alloy \
    test_filtered_beacon_blobs_deserializes_on_small_stack -- --list
  stage beacon just test-beacon-blob-stack
  docs
}
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
  tests|tests-unit)
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
    if [[ "$job" == tests-unit ]]; then exit 0; fi
    compiler_tests
    ;;
  tests-compiler) compiler_tests ;;
  tests-merge)
    python3 "$HELPERS/rust-workspace-report.py" merge-tests "$report" \
      "${UNIT_REPORT:?UNIT_REPORT must identify the nextest report}" \
      "${COMPILER_REPORT:?COMPILER_REPORT must identify the compiler-dependent report}"
    ;;
  doctest) docs ;;
  docs) stage docs just lint-docs ;;
  clippy) stage clippy cargo clippy --workspace --all-targets --all-features --locked ;;
  build) stage build mold -run cargo build --profile dev --workspace --features default ;;
  features)
    if [[ "$total" == 1 ]]; then features ""; else features "$((index + 1))/$total"; fi
    ;;
  feature-plan) features "" ;;
  no-std) stage no-std just check-no-std ;;
  udeps) stage udeps just check-udeps ;;
  wasm-unknown|wasm-wasi)
    target=wasm32-wasip1
    packages=(-p op-alloy-consensus -p op-alloy-rpc-types-engine -p alloy-op-evm)
    if [[ "$job" == wasm-unknown ]]; then
      target=wasm32-unknown-unknown
      packages=(-p op-alloy-consensus -p op-alloy-rpc-types -p op-alloy-rpc-types-engine -p alloy-op-evm --no-default-features)
    fi
    stage wasm-target rustup target add "$target"
    stage wasm-list cargo hack build --target "$target" "${packages[@]}" --print-command-list
    stage wasm cargo hack build --target "$target" "${packages[@]}"
    ;;
  zepter) stage zepter zepter run check ;;
  typos) stage typos typos ;;
  registry)
    python3 "$HELPERS/rust-workspace-report.py" registry-snapshot "$report" before
    # A restored target may already have KONA_SYNC_SUPERCHAIN=true. Cleaning
    # just this crate forces its build script to regenerate all three snapshots.
    stage registry-clean cargo clean -p kona-registry
    status=0
    stage_at rust/kona registry env KONA_SYNC_SUPERCHAIN=true cargo build -p kona-registry || status=$?
    python3 "$HELPERS/rust-workspace-report.py" registry-snapshot "$report" after
    if [[ "$status" != 0 ]]; then exit "$status"; fi
    stage_at . registry-diff git diff --exit-code -- rust/kona/crates/protocol/registry/etc/
    ;;
  interop)
    if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx ]]; then
      export GOPATH="$ROOT/.ci/interop-go/gopath" GOCACHE="$ROOT/.ci/interop-go/build"
      mkdir -p "$GOPATH" "$GOCACHE"
    fi
    export CI_INTEROP_REPORT_DIR="$report"
    stage_at . superchain-go just build-superchain-go
    stage_at . interop bash ops/scripts/test-interop-deposits-diff.sh
    ;;
esac
