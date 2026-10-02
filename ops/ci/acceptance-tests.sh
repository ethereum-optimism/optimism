#!/usr/bin/env bash
# Provider-independent discovery and fresh verdicts for the shared Just runner.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
phase="${1:?Pass discover or run}"
[[ "$CI_SHARD_TOTAL" == 8 ]] || { echo "Acceptance workflow requires exactly eight shards" >&2; exit 1; }
export ACCEPTANCE_TEST_JOBS=8 ACCEPTANCE_TEST_PARALLEL=1 ACCEPTANCE_TEST_TIMEOUT=30m LOG_LEVEL=info
export GOMODCACHE="$PWD/.ci/go-cache/full/modules"
export GOCACHE="$PWD/.ci/go-cache/acceptance/$phase"
export GOROOT
GOROOT="$(go env GOROOT)"
export PATH="$GOROOT/bin:$PATH"
mkdir -p "$GOCACHE"
case "$phase" in
  discover)
    mkdir -p .ci/acceptance/discovery
    go list -e -json ./op-acceptance-tests/tests/... > .ci/acceptance/discovery/packages.json
    go test -list '^Test' -json ./op-acceptance-tests/tests/... > .ci/acceptance/discovery/listing.json
    python3 ops/ci/acceptance-manifest.py create --directory .ci/acceptance/discovery --total "$CI_SHARD_TOTAL"
    ;;
  run)
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
      test -x "$binary" || { echo "Missing executable runtime dependency: $binary" >&2; exit 1; }
    done
    # Match Circle's mock verifier setup; real SP1 ELFs are a separate gate.
    unset KONA_SP1_ELF_DIR RUST_JIT_BUILD
    geth_binary="$(mise which geth)"
    test -x "$geth_binary"
    mkdir -p tmp/testlogs/dependencies
    python3 - "$geth_binary" <<'PYCODE'
import hashlib, json, subprocess, sys
from pathlib import Path
binary = Path(sys.argv[1])
metadata = {'geth_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
            'geth_version': subprocess.check_output([str(binary), 'version'], text=True).strip()}
Path('tmp/testlogs/dependencies/runtime-tools.json').write_text(json.dumps(metadata, indent=2) + '\n')
PYCODE
    export ACCEPTANCE_TEST_HIDE_FAILURE_OUTPUT=true
    export LD_PRELOAD=libeatmydata.so
    mkdir -p .ci/acceptance/reports
    finish() {
      local status=$?
      trap - EXIT
      if ! python3 ops/ci/acceptance-report.py .ci/acceptance/reports; then
        [[ "$status" -ne 0 ]] || status=1
      fi
      exit "$status"
    }
    trap finish EXIT
    (cd op-acceptance-tests && just acceptance-test)
    ;;
  *) echo 'Expected discover or run' >&2; exit 1 ;;
esac
