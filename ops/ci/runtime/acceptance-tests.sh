#!/usr/bin/env bash
# Run shard CI_SHARD_INDEX of CircleCI's op-acceptance-tests job with the
# unchanged recipe. The caller restores the build outputs that the job attaches.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../../.."
# CircleCI's memory-all jobs set acceptance_test_jobs: "16" and LOG_LEVEL=info.
export ACCEPTANCE_TEST_JOBS=16 LOG_LEVEL=info
export GOMODCACHE="$PWD/.ci/go-cache/full/modules"
export GOCACHE="$PWD/.ci/go-cache/acceptance/run"
export GOROOT
GOROOT="$(go env GOROOT)"
export PATH="$GOROOT/bin:$PATH"
mkdir -p "$GOCACHE"
export RUST_BINARY_PATH_KONA_HOST="$PWD/rust/target/release/kona-host"
export RUST_BINARY_PATH_KONA_CLIENT="$PWD/rust/target/release/kona-client"
export RUST_BINARY_PATH_KONA_NODE="$PWD/rust/target/release/kona-node"
export RUST_BINARY_PATH_OP_ZK_PROPOSER="$PWD/rust/target/release/op-zk-proposer"
export RUST_BINARY_PATH_OP_RETH="$PWD/rust/target/release/op-reth"
export RUST_BINARY_PATH_KONA_SP1_SUPER_RANGE_EXECUTOR="$PWD/.circleci-cache/rust-binaries/kona-sp1-super-range-executor"
for binary in "$RUST_BINARY_PATH_KONA_HOST" "$RUST_BINARY_PATH_KONA_CLIENT" \
  "$RUST_BINARY_PATH_KONA_NODE" "$RUST_BINARY_PATH_OP_ZK_PROPOSER" \
  "$RUST_BINARY_PATH_OP_RETH" \
  "$RUST_BINARY_PATH_KONA_SP1_SUPER_RANGE_EXECUTOR"; do
  test -x "$binary" || {
    echo "Missing executable runtime dependency: $binary" >&2
    exit 1
  }
done
# Fail before the suite starts when the pinned geth is not installed.
test -x "$(mise which geth)"
# Like CircleCI's "Disable fsync for all subprocesses" step.
if [[ -f /usr/lib/x86_64-linux-gnu/libeatmydata.so ]]; then
  export LD_PRELOAD=libeatmydata.so
fi
# The recipe shards through ops/scripts/shard-tests.sh. Its `circleci tests
# split` call reaches the shim, which prints this shard's share of the tests.
# CircleCI's memory-all jobs have parallelism: 4, and .rwx/acceptance.yml has
# one verdict task for each of the four shards.
export CIRCLE_NODE_INDEX="$CI_SHARD_INDEX" CIRCLE_NODE_TOTAL=4
export PATH="$PWD/ops/ci/runtime/acceptance-circleci:$PATH"
cd op-acceptance-tests
just acceptance-test
