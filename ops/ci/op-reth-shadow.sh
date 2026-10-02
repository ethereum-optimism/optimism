#!/usr/bin/env bash
# Compile with caches; execute fresh verdicts separately. Called at repo root.
set -euo pipefail
REPO_ROOT="$(pwd)"
REPORT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/op-reth-report.py"
[[ -f rust/Cargo.lock && -d .git || -f rust/Cargo.lock && -f .git ]] || {
  echo 'Run op-reth-shadow.sh from the repository root.' >&2; exit 1;
}
export CARGO_HOME="${REPO_ROOT}/.ci/rust-cache/cargo"
export CARGO_TARGET_DIR="${REPO_ROOT}/rust/target"
export CARGO_INCREMENTAL=0 RUSTC_WRAPPER=sccache
export SCCACHE_DIR="${REPO_ROOT}/.ci/rust-cache/sccache"
export SCCACHE_CACHE_SIZE=10G SCCACHE_IDLE_TIMEOUT=0 SCCACHE_LOG=warn
export SCCACHE_BASEDIRS="${REPO_ROOT}"
job="${1:?Pass a build or verdict job}"
report_dir="${REPO_ROOT}/.ci/op-reth/${job}"
mkdir -p "$report_dir"
started="$(date +%s)"

finish() {
  local status=$? diagnostics=0
  trap - EXIT
  if [[ "$job" == *-build ]]; then
    sccache --show-stats --stats-format json >"$report_dir/sccache.json" || diagnostics=$?
    sccache --stop-server >"$report_dir/sccache-stop.log" 2>&1 || diagnostics=$?
  fi
  python3 "$REPORT" metadata "$report_dir" "$job" "$started" "$status" || diagnostics=$?
  if [[ "$status" == 0 && "$diagnostics" != 0 ]]; then status=$diagnostics; fi
  exit "$status"
}
trap finish EXIT

if [[ "$job" == *-build ]]; then
  case "${TARGET_CACHE_MODE:-keep}" in
    keep) ;;
    sccache-only) rm -rf "${REPO_ROOT}/rust/target" ;;
    *) echo 'TARGET_CACHE_MODE must be keep or sccache-only.' >&2; exit 1 ;;
  esac
  mkdir -p "$CARGO_HOME" "$SCCACHE_DIR"
  sccache --start-server
  sccache --zero-stats
  # Keep a checksum-matching bundle without touching its mtime. Removing it on
  # every build dirties chainspec and forces downstream recompilation/relinking.
  # A stale bundle is removed so build.rs regenerates from the pinned submodule.
  python3 "$REPORT" prepare-superchain "$report_dir"
fi

case "$job" in
  source)
    just update-superchain-registry-submodule
    git submodule status superchain-registry >"$report_dir/submodule.txt"
    ;;
  release-build)
    (cd rust && mold -run cargo build --locked --profile release \
      --package op-reth --features default) \
      2>&1 | tee "$report_dir/build.log"
    cp rust/target/release/op-reth "$report_dir/"
    python3 "$REPORT" binaries "$report_dir" "${CI_COMMIT_SHA:?}" op-reth
    ;;
  release)
    python3 "$REPORT" verify-binaries .ci/op-reth/release-build "${CI_COMMIT_SHA:?}" op-reth
    .ci/op-reth/release-build/op-reth --version | tee "$report_dir/op-reth-version.txt"
    ;;
  integration-build)
    (cd rust && mold -run cargo nextest archive --locked -p reth-optimism-node \
      --archive-file "$report_dir/tests.tar.zst") 2>&1 | tee "$report_dir/build.log"
    python3 "$REPORT" binaries "$report_dir" "${CI_COMMIT_SHA:?}" tests.tar.zst
    ;;
  integration)
    python3 "$REPORT" verify-binaries .ci/op-reth/integration-build "${CI_COMMIT_SHA:?}" tests.tar.zst
    # Extract manually to a stable path so JUnit and metadata survive nextest's
    # temporary-directory cleanup. Runtime config and source remain from the PR.
    unpacked="$report_dir/unpacked"
    rm -rf "$unpacked"
    mkdir -p "$unpacked"
    tar --zstd -xf .ci/op-reth/integration-build/tests.tar.zst -C "$unpacked"
    nextest_args=(--binaries-metadata "$unpacked/target/nextest/binaries-metadata.json"
      --cargo-metadata "$unpacked/target/nextest/cargo-metadata.json"
      --target-dir-remap "$unpacked/target" --workspace-remap "$REPO_ROOT/rust")
    # Nextest's report store is relative to the remapped workspace, not the
    # binary target-dir remap. Keep the committed config and collect its path.
    junit="$REPO_ROOT/rust/target/nextest/default/junit.xml"
    rm -f "$junit"
    (cd rust && cargo nextest list "${nextest_args[@]}" --message-format json) >"$report_dir/discovery.json"
    status=0
    (cd rust && RUST_BACKTRACE=1 cargo nextest run "${nextest_args[@]}") \
      2>&1 | tee "$report_dir/tests.log" || status=$?
    if [[ -s "$junit" ]]; then cp "$junit" "$report_dir/junit.xml"; fi
    report_status=0
    python3 "$REPORT" integration "$report_dir" || report_status=$?
    # The reusable archive is a producer artifact; verdict artifacts only need
    # discovery, original JUnit, coverage and logs, not a second copy of binaries.
    rm -rf "$unpacked"
    # A reporting failure must never replace the original failing verdict.
    if [[ "$status" != 0 ]]; then exit "$status"; fi
    exit "$report_status"
    ;;
  codec-base-build|codec-head-build)
    (cd rust && mold -run cargo build --locked --bin op-reth --features dev \
      --manifest-path op-reth/bin/Cargo.toml) 2>&1 | tee "$report_dir/build.log"
    cp rust/target/debug/op-reth "$report_dir/op-reth"
    if [[ "$job" == codec-base-build ]]; then expected="${CODEC_BASE_SHA:?}"; else expected="${CI_COMMIT_SHA:?}"; fi
    python3 "$REPORT" binaries "$report_dir" "$expected" op-reth
    ;;
  codec-vectors)
    python3 "$REPORT" verify-binaries .ci/op-reth/codec-base-build "${CODEC_BASE_SHA:?}" op-reth
    rm -rf testdata/micro/compact
    .ci/op-reth/codec-base-build/op-reth test-vectors compact --write 2>&1 | tee "$report_dir/generate.log"
    python3 "$REPORT" vectors "$report_dir" "${CODEC_BASE_SHA:?}"
    ;;
  codec)
    python3 "$REPORT" verify-binaries .ci/op-reth/codec-head-build "${CI_COMMIT_SHA:?}" op-reth
    python3 "$REPORT" verify-vectors .ci/op-reth/codec-vectors "${CODEC_BASE_SHA:?}"
    .ci/op-reth/codec-head-build/op-reth test-vectors compact --read 2>&1 | tee "$report_dir/read.log"
    ;;
  snapshot-build)
    (cd rust && OP_RETH_SYNC_SUPERCHAIN=0 mold -run cargo build --locked \
      -p reth-optimism-chainspec --features superchain-configs) 2>&1 | tee "$report_dir/build.log"
    ;;
  snapshot)
    # The producer always leaves Cargo's recorded env at 0. Switching to 1 is
    # declared by build.rs, forcing fresh regeneration even on a content hit.
    unset RUSTC_WRAPPER
    (cd rust && OP_RETH_SYNC_SUPERCHAIN=1 mold -run cargo build --locked \
      -p reth-optimism-chainspec --features superchain-configs) 2>&1 | tee "$report_dir/regenerate.log"
    git diff --exit-code -- \
      rust/op-reth/crates/chainspec/res/superchain-configs.tar.sha256 \
      rust/op-reth/crates/chainspec/src/superchain/chain_specs.rs \
      >"$report_dir/snapshot.diff"
    ;;
  *) echo "Unknown op-reth shadow job: $job" >&2; exit 1 ;;
esac
