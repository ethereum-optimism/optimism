#!/usr/bin/env bash
# Complete rollup package tests, partitioned without CI-provider timing services.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"
PACKAGE_PREFIX="github.com/ethereum-optimism/optimism/op-node/rollup"
MANIFEST=".ci/go-rollup/manifest.json"
CI_SHARD_TOTAL="${CI_SHARD_TOTAL:-4}"

case "${1:-run}" in
  prepare)
    mkdir -p .ci/go-rollup
    # Keep the command's exit status, then reject errors hidden by go list -e.
    go list -e -tags=ci -json ./op-node/rollup/... >.ci/go-rollup/go-list.json
    python3 ops/ci/runtime/go-package-shards.py create \
      --prefix "${PACKAGE_PREFIX}" --total "${CI_SHARD_TOTAL}" \
      --timings ops/ci/runtime/go-rollup-timings.json \
      --output "${MANIFEST}" <.ci/go-rollup/go-list.json
    ;;
  build)
    export GOCACHE="${REPO_ROOT}/.ci/go-cache/rollup/build"
    export GOMODCACHE="${REPO_ROOT}/.ci/go-cache/rollup/modules"
    mkdir -p "${GOCACHE}" "${GOMODCACHE}"
    attempt=0
    until go mod download; do
      attempt=$((attempt + 1))
      if [[ "${attempt}" -ge 5 ]]; then exit 1; fi
      sleep "$((2 ** attempt))"
    done
    bash ops/ci/runtime/go-rollup-tests.sh prepare
    python3 ops/ci/runtime/go-compiled-tests.py build
    bash ops/ci/runtime/rwx-source-archive.sh .ci/go-rollup/build/source.tar.gz
    ;;
  run)
    CI_SHARD_INDEX="${CI_SHARD_INDEX:-0}"
    mkdir -p tmp/test-results tmp/testlogs/per-test
    python3 ops/ci/runtime/go-package-shards.py select \
      --prefix "${PACKAGE_PREFIX}" --total "${CI_SHARD_TOTAL}" \
      --index "${CI_SHARD_INDEX}" --manifest "${MANIFEST}" \
      >tmp/testlogs/packages.txt
    PACKAGES="$(tr '\n' ' ' <tmp/testlogs/packages.txt)"
    if [[ -z "${PACKAGES// /}" ]]; then
      echo "No packages assigned to shard ${CI_SHARD_INDEX}/${CI_SHARD_TOTAL}."
      exit 0
    fi
    echo "Shard ${CI_SHARD_INDEX}/${CI_SHARD_TOTAL} running packages: ${PACKAGES}"
    PARALLEL="$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)"
    OP_TESTLOG_FILE_LOGGER_OUTDIR="$(realpath tmp/testlogs)"
    export PARALLEL OP_TESTLOG_FILE_LOGGER_OUTDIR
    if [[ "${RWX_COMPILED_GO:-false}" == true ]]; then
      python3 ops/ci/runtime/go-compiled-tests.py verify
      ./ops/scripts/gotestsum-split.sh --format=standard-verbose \
        --junitfile="tmp/test-results/results-${CI_SHARD_INDEX}.xml" \
        --jsonfile="tmp/testlogs/log.json" \
        --rerun-fails=3 --rerun-fails-max-failures=50 --raw-command \
        -- python3 ops/ci/runtime/go-compiled-tests.py run
    else
      ./ops/scripts/gotestsum-split.sh --format=standard-verbose \
      --junitfile="tmp/test-results/results-${CI_SHARD_INDEX}.xml" \
      --jsonfile="tmp/testlogs/log.json" \
      --rerun-fails=3 --rerun-fails-max-failures=50 --packages="${PACKAGES}" \
      -- -count=1 -p=4 -parallel="${PARALLEL}" -timeout="${TEST_TIMEOUT:-40m}" -tags=ci
    fi
    ;;
  *)
    echo "Usage: $0 [prepare|build|run]" >&2
    exit 1
    ;;
esac
