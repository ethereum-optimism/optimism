#!/usr/bin/env bash
# Rust workspace commands and evidence for RWX tasks. Run from the monorepo root.
set -euo pipefail
HELPERS="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
job="${1:?Pass a Rust workspace job}"
case "$job" in tests-build | tests-unit | tests-compiler | docs | clippy | build | features | no-std | udeps | wasm-unknown | wasm-wasi | zepter | typos | deny | registry | interop) ;;
*)
  echo "Unknown Rust workspace job: $job" >&2
  exit 1
  ;;
esac
index="${CI_RUST_PARTITION_INDEX:-0}"
total="${CI_RUST_PARTITION_TOTAL:-10}"
if [[ "$job" == features ]]; then
  if ! [[ "$index" =~ ^(0|[1-9][0-9]*)$ && "$total" =~ ^[1-9][0-9]*$ ]] || ((index >= total)); then
    echo 'Expected a nonnegative feature index below a positive partition total.' >&2
    exit 1
  fi
  report="$ROOT/.ci/rust-workspace/features-$index"
else
  report="$ROOT/.ci/rust-workspace/$job"
fi
rm -rf "$report"
mkdir -p "$report"
export CARGO_INCREMENTAL=0
if [[ "$job" == clippy ]]; then export RUSTFLAGS=-Dwarnings; fi
export CARGO_HOME="$ROOT/.ci/rust-cache/cargo" CARGO_TARGET_DIR="$ROOT/rust/target"
export RUSTC_WRAPPER=sccache SCCACHE_DIR="$ROOT/.ci/rust-cache/sccache"
export SCCACHE_CACHE_SIZE=10G SCCACHE_IDLE_TIMEOUT=0 SCCACHE_BASEDIRS="$ROOT" SCCACHE_LOG=warn
mkdir -p "$CARGO_HOME" "$SCCACHE_DIR" "$CARGO_TARGET_DIR"
sccache --start-server
sccache --zero-stats
python3 "$HELPERS/rust-workspace-report.py" begin "$report" "$job"
finish() {
  local status=$? diagnostics=0
  trap - EXIT
  # Original verdicts are copied into the report artifact, never compiler caches.
  rm -rf "$ROOT/rust/target/nextest/default"
  sccache --show-stats --stats-format json > "$report/sccache.json" || diagnostics=$?
  sccache --stop-server > "$report/sccache-stop.log" 2>&1 || diagnostics=$?
  if [[ "$status" == 0 ]]; then
    python3 "$HELPERS/rust-target-cache.py" commit > "$report/cache-publication.json" || diagnostics=$?
  fi
  if [[ "$status" == 0 && "$diagnostics" != 0 ]]; then status=$diagnostics; fi
  python3 "$HELPERS/rust-workspace-report.py" finish "$report" "$status" || status=$?
  exit "$status"
}
trap finish EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
stage() { python3 "$HELPERS/rust-workspace-report.py" stage "$report" "$@"; }
stage_at() { python3 "$HELPERS/rust-workspace-report.py" stage-at "$report" "$@"; }
json_stage() { python3 "$HELPERS/rust-workspace-report.py" json-stage "$report" "$@"; }
python3 "$HELPERS/op-reth-report.py" prepare-superchain
python3 "$HELPERS/rust-target-cache.py" prepare > "$report/cache-source.json"
case "$job" in
  tests-build)
    stage archive mold -run cargo nextest archive --workspace --all-features --locked --archive-file "$report/tests.tar.zst"
    if [[ "${CI_RUST_BASELINE:-false}" == true ]]; then
      # A cold chainspec build generates a watched archive after Cargo stamps
      # its build-script fingerprint. Settle that input before publishing a
      # reusable compiler seed. This is preparation; HEAD creates its own archive.
      json_stage compiler-settle cargo nextest list --workspace --all-features --locked --message-format json
    fi
    # Compile the extra test once; its fresh verdict remains in the runtime task.
    stage beacon-build cargo test --profile fast-build --locked -p kona-providers-alloy \
      test_filtered_beacon_blobs_deserializes_on_small_stack --no-run
    ;;
  tests-unit)
    rm -f "$ROOT/rust/target/nextest/default/junit.xml"
    producer="${TEST_ARCHIVE:?TEST_ARCHIVE must identify the compiled producer artifact}"
    # Extract at the producer's target directory. Every RWX task in this run
    # has the same workspace path, so nextest's recorded paths and compile-time
    # paths such as env!("CARGO_BIN_EXE_lokahi") resolve without a remap.
    tar --zstd -xf "$producer/tests.tar.zst" -C "$ROOT/rust"
    status=0
    stage unit cargo nextest run --binaries-metadata "$ROOT/rust/target/nextest/binaries-metadata.json" \
      --cargo-metadata "$ROOT/rust/target/nextest/cargo-metadata.json" -E '!test(test_online)' || status=$?
    # RWX reads nextest's own JUnit report from the report directory, also after a test failure.
    if [[ -f "$ROOT/rust/target/nextest/default/junit.xml" ]]; then
      cp "$ROOT/rust/target/nextest/default/junit.xml" "$report/junit.xml"
    fi
    exit "$status"
    ;;
  tests-compiler)
    # CircleCI's rust-tests job runs `just test`. The tests-unit job runs its `test-unit`
    # recipe from a nextest archive; these are the other recipes that `just test` runs.
    stage beacon just test-beacon-blob-stack
    stage doctests just test-docs
    stage custom-embeds just test-custom-embeds
    ;;
  docs) stage docs just lint-docs ;;
  clippy) stage clippy cargo clippy --workspace --all-targets --all-features --locked ;;
  build) stage build mold -run cargo build --locked --profile dev --workspace --features default ;;
  features)
    # Circle's recipes and partition argument. One seed shuffles every partition.
    # The ancestor compiler producer has the same manifests and Cargo.lock, so
    # its partition holds the same crates.
    partition=""
    if [[ "$total" != 1 ]]; then partition="$((index + 1))/$total"; fi
    seed="$(sha256sum rust/Cargo.lock | cut -d' ' -f1)"
    stage features just hack "$partition" true "$seed"
    stage feature-tests just hack-tests-default "$partition" true "$seed"
    ;;
  no-std) stage no-std just check-no-std ;;
  udeps) stage udeps just check-udeps ;;
  wasm-unknown | wasm-wasi)
    # Native build scripts must not lazily reopen restored GCC/glibc inputs
    # while Cargo links them concurrently. Preserve byte/inode/read evidence.
    stage_at . host-linker sudo python3 "$HELPERS/rwx-materialize-linker.py"
    target=wasm32-wasip1
    packages=(-p op-alloy-consensus -p op-alloy-rpc-types-engine -p alloy-op-evm)
    if [[ "$job" == wasm-unknown ]]; then
      target=wasm32-unknown-unknown
      packages=(-p op-alloy-consensus -p op-alloy-rpc-types -p op-alloy-rpc-types-engine -p alloy-op-evm --no-default-features)
    fi
    stage wasm-target rustup target add "$target"
    stage wasm cargo hack build --target "$target" "${packages[@]}"
    ;;
  zepter) stage zepter zepter run check ;;
  typos) stage typos typos ;;
  deny) stage deny just deny ;;
  registry)
    # A restored target may already have KONA_SYNC_SUPERCHAIN=true. Cleaning
    # just this crate forces its build script to regenerate all three snapshots.
    stage registry-clean cargo clean -p kona-registry
    stage_at rust/kona registry env KONA_SYNC_SUPERCHAIN=true cargo build -p kona-registry
    stage_at . registry-diff git diff --exit-code -- rust/kona/crates/protocol/registry/etc/
    ;;
  interop)
    export GOPATH="$ROOT/.ci/interop-go/gopath" GOCACHE="$ROOT/.ci/interop-go/build"
    mkdir -p "$GOPATH" "$GOCACHE"
    stage_at . superchain-go just build-superchain-go
    stage_at . interop bash ops/scripts/test-interop-deposits-diff.sh
    ;;
esac
