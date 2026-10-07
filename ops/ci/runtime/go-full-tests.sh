#!/usr/bin/env bash
# Fresh aggregate Go verdicts; reusable compilation lives in a separate task.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../../.."
export PARALLEL="${PARALLEL:-8}" TEST_TIMEOUT="${TEST_TIMEOUT:-40m}"
export ENABLE_KURTOSIS=true OP_E2E_CANNON_ENABLED=false OP_E2E_USE_HTTP=true ENABLE_ANVIL=true
export NAT_INTEROP_LOADTEST_TARGET=10 NAT_INTEROP_LOADTEST_TIMEOUT=30s
export GOMODCACHE="$PWD/.ci/go-cache/full/modules"
export GOCACHE="$PWD/.ci/go-cache/full/${1:-run}"
mkdir -p "$GOMODCACHE" "$GOCACHE"
# Unlike `go test`, invoking compiled binaries does not replace mise's GOROOT
# when GOTOOLCHAIN selects the newer version required by go.mod.
export GOROOT
GOROOT="$(go env GOROOT)"
export PATH="$GOROOT/bin:$PATH"
phase="${1:-run}"
phase_started="$(date +%s)"
finish_phase() {
  local status=$?
  trap - EXIT
  if [[ "$phase" == run && -f tmp/testlogs/log.json ]]; then
    if ! python3 ops/ci/runtime/go-report.py tmp/testlogs/log.json tmp/testlogs/native.json; then
      [[ "$status" -ne 0 ]] || status=1
    fi
  fi
  python3 - "$phase" "$phase_started" "$status" <<'PYCODE'
import json, os, sys, time
from pathlib import Path
phase, started, status = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])
output = Path('tmp/testlogs') if phase == 'run' else Path('.ci/go-tests') / phase
output.mkdir(parents=True, exist_ok=True)
metadata = {'phase': phase, 'started_at_unix': started, 'finished_at_unix': time.time(),
            'exit_code': status, 'commit_sha': os.environ['CI_COMMIT_SHA'],
            'shard_index': os.environ.get('CI_SHARD_INDEX'), 'shard_total': os.environ.get('CI_SHARD_TOTAL')}
(output / 'phase.json').write_text(json.dumps(metadata, indent=2) + '\n')
PYCODE
  exit "$status"
}
trap finish_phase EXIT
case "${1:-run}" in
  discover)
    case "${CI_SHARD_TOTAL:-12}" in 12) ;; *) echo "Full suite requires 12 shards" >&2; exit 1 ;; esac
    python3 ops/ci/runtime/go-suite.py discover --suite go-tests --total "${CI_SHARD_TOTAL:-12}" --timings ops/ci/runtime/go-tests-timings.json
    ;;
  build)
    python3 ops/ci/runtime/go-compiled-tests.py build --suite go-tests
    ;;
  run)
    mkdir -p tmp/test-results tmp/testlogs/per-test
    cp .ci/go-tests/{manifest.json,go-list.json,all-packages.txt} tmp/testlogs/
    python3 ops/ci/runtime/go-package-shards.py select --prefix github.com/ethereum-optimism/optimism \
      --total "$CI_SHARD_TOTAL" --index "$CI_SHARD_INDEX" --manifest .ci/go-tests/manifest.json >tmp/testlogs/packages.txt
    export ENABLE_KURTOSIS=true OP_E2E_CANNON_ENABLED=false OP_E2E_USE_HTTP=true ENABLE_ANVIL=true
    export NAT_INTEROP_LOADTEST_TARGET=10 NAT_INTEROP_LOADTEST_TIMEOUT=30s
    export OP_TESTLOG_FILE_LOGGER_OUTDIR="$PWD/tmp/testlogs"
    # shellcheck disable=SC1091
    source ops/scripts/source-ci-archive-rpcs.sh
    (cd cannon && just diff-hello-elf)
    python3 ops/ci/runtime/go-compiled-tests.py verify --suite go-tests
    # Keep full output in JSON/per-test artifacts without exhausting live-log quotas.
    ./ops/scripts/gotestsum-split.sh --format=pkgname \
      --junitfile="tmp/test-results/results-${CI_SHARD_INDEX}.xml" --jsonfile=tmp/testlogs/log.json \
      --rerun-fails=3 --rerun-fails-max-failures=50 --raw-command \
      -- python3 ops/ci/runtime/go-compiled-tests.py run --suite go-tests
    ;;
  *) echo 'Expected discover, build or run' >&2; exit 1 ;;
esac
