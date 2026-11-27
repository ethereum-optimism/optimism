#!/usr/bin/env bash
# The Go test workload of .rwx/go-tests.yml: discover the packages, compile one
# shard's test binaries, or run them. Compilation and the run are separate tasks.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../../.."
phase="${1:?Expected discover, build or run}"
# .rwx/go-tests.yml has one compile task and one verdict task per shard.
shards=12
# Each phase has its own writable build cache. Test binaries inherit the run
# cache, module cache and toolchain, e.g. for the deployer CLI's TestMain build.
export GOMODCACHE="$PWD/.ci/go-cache/full/modules"
export GOCACHE="$PWD/.ci/go-cache/full/$phase"
mkdir -p "$GOMODCACHE" "$GOCACHE"
# Unlike `go test`, invoking compiled binaries does not replace mise's GOROOT
# when GOTOOLCHAIN selects the newer version required by go.mod.
unset GOROOT
export GOROOT
GOROOT="$(go env GOROOT)"
export PATH="$GOROOT/bin:$PATH"
case "$phase" in
  discover)
    python3 ops/ci/runtime/go-suite.py --total "$shards" --timings ops/ci/runtime/go-tests-timings.json
    ;;
  build)
    python3 ops/ci/runtime/go-compiled-tests.py build
    ;;
  run)
    mkdir -p tmp/test-results tmp/testlogs
    # Compiler-seed layers can carry an older commit's manifest. Use the one
    # that this shard's binaries were compiled from.
    python3 ops/ci/runtime/go-compiled-tests.py restore
    # The environment and gotestsum options of the `_go-tests-ci-internal`
    # recipe in the root justfile. The restored `go` dependency already holds
    # cannon/testdata/bin/hello.64.elf, which the recipe builds at this point.
    export ENABLE_KURTOSIS=true OP_E2E_CANNON_ENABLED=false OP_E2E_USE_HTTP=true ENABLE_ANVIL=true
    PARALLEL="$(nproc 2> /dev/null || sysctl -n hw.ncpu 2> /dev/null || echo 4)"
    export PARALLEL TEST_TIMEOUT="${TEST_TIMEOUT:-40m}"
    export OP_TESTLOG_FILE_LOGGER_OUTDIR="$PWD/tmp/testlogs"
    # shellcheck disable=SC1091
    source ops/scripts/source-ci-archive-rpcs.sh
    export NAT_INTEROP_LOADTEST_TARGET=10 NAT_INTEROP_LOADTEST_TIMEOUT=30s
    status=0
    # --format=pkgname keeps full output in the JSON and per-test artifacts
    # without exhausting live-log quotas.
    ./ops/scripts/gotestsum-split.sh --format=pkgname \
      --junitfile="tmp/test-results/results-${CI_SHARD_INDEX}.xml" --jsonfile=tmp/testlogs/log.json \
      --rerun-fails=3 --rerun-fails-max-failures=50 --raw-command \
      -- python3 ops/ci/runtime/go-compiled-tests.py run || status=$?
    # RWX reads the shortened copy. The go-json artifact keeps the complete log.
    if [[ -f tmp/testlogs/log.json ]]; then
      python3 ops/ci/runtime/go-report.py tmp/testlogs/log.json tmp/testlogs/native.json || [[ "$status" -ne 0 ]] || status=1
    fi
    exit "$status"
    ;;
  *)
    echo 'Expected discover, build or run' >&2
    exit 1
    ;;
esac
