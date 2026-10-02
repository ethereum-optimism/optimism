#!/usr/bin/env bash
# Fresh aggregate Go verdicts; reusable compilation lives in a separate task.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
export PARALLEL="${PARALLEL:-8}" TEST_TIMEOUT="${TEST_TIMEOUT:-40m}"
export GOMODCACHE="$PWD/.ci/go-cache/full/modules"
export GOCACHE="$PWD/.ci/go-cache/full/${1:-run}"
mkdir -p "$GOMODCACHE" "$GOCACHE"
case "${1:-run}" in
  discover)
    case "${CI_SHARD_TOTAL:-12}" in 12|24) ;; *) echo "Full suite requires 12 or 24 shards" >&2; exit 1 ;; esac
    python3 ops/ci/go-suite.py discover --suite go-tests --total "${CI_SHARD_TOTAL:-12}" --timings ops/ci/go-tests-timings.json
    ;;
  build)
    python3 ops/ci/go-compiled-tests.py build --suite go-tests
    ;;
  run)
    mkdir -p tmp/test-results tmp/testlogs/per-test
    cp .ci/go-tests/{manifest.json,go-list.json,all-packages.txt} tmp/testlogs/
    python3 ops/ci/go-package-shards.py select --prefix github.com/ethereum-optimism/optimism \
      --total "$CI_SHARD_TOTAL" --index "$CI_SHARD_INDEX" --manifest .ci/go-tests/manifest.json >tmp/testlogs/packages.txt
    export ENABLE_KURTOSIS=true OP_E2E_CANNON_ENABLED=false OP_E2E_USE_HTTP=true ENABLE_ANVIL=true
    export NAT_INTEROP_LOADTEST_TARGET=10 NAT_INTEROP_LOADTEST_TIMEOUT=30s
    export OP_TESTLOG_FILE_LOGGER_OUTDIR="$PWD/tmp/testlogs"
    # shellcheck disable=SC1091
    source ops/scripts/source-ci-archive-rpcs.sh
    (cd cannon && just diff-hello-elf)
    python3 ops/ci/go-compiled-tests.py verify --suite go-tests
    ./ops/scripts/gotestsum-split.sh --format=standard-verbose \
      --junitfile="tmp/test-results/results-${CI_SHARD_INDEX}.xml" --jsonfile=tmp/testlogs/log.json \
      --rerun-fails=3 --rerun-fails-max-failures=50 --raw-command \
      -- python3 ops/ci/go-compiled-tests.py run --suite go-tests
    ;;
  *) echo 'Expected discover, build or run' >&2; exit 1 ;;
esac
