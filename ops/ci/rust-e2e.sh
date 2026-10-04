#!/usr/bin/env bash
# Shared full Rust E2E workloads. Compiler outputs are reusable; verdicts are fresh.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
mode="${1:?Pass compile, run or circle}"
job="${2:?Pass the E2E job name}"
export CI_COMMIT_SHA="${CI_COMMIT_SHA:-${CIRCLE_SHA1:?Missing source revision}}"
export CI_E2E_PROVIDER="${CI_E2E_PROVIDER:-circleci}"
case "$job" in proof|restart|simple-kona|simple-kona-sequencer|op-reth) ;; *) echo 'Unknown E2E workload' >&2; exit 1;; esac
if [[ "$CI_E2E_PROVIDER" == rwx ]]; then
  export GOMODCACHE="$PWD/.ci/go-cache/full/modules"
  export GOCACHE="$PWD/.ci/go-cache/rust-e2e/$mode"
  mkdir -p "$GOMODCACHE" "$GOCACHE"
fi
if [[ "$mode" == compile || "$mode" == circle ]]; then
  python3 ops/ci/rust-e2e.py compile "$job"
  if [[ "$mode" == compile ]]; then exit; fi
fi
if [[ "$mode" != run && "$mode" != circle ]]; then echo 'Unknown E2E mode' >&2; exit 1; fi
export CI_SHARD_INDEX="${CI_SHARD_INDEX:-${CIRCLE_NODE_INDEX:-0}}"
if [[ "$CI_E2E_PROVIDER" == circleci ]]; then
  # Accept real Circle metadata only in its own adapter; native RWX uses its
  # internal manifest and never invents CIRCLE_NODE_* timing-split variables.
  case "$job" in proof) expected=8 ;; restart) expected=3 ;; *) expected=1 ;; esac
  [[ "${CIRCLE_NODE_TOTAL:-1}" == "$expected" ]] || { echo 'Circle E2E parallelism differs from workload' >&2; exit 1; }
fi
export PARALLEL="${PARALLEL:-$(nproc 2>/dev/null || sysctl -n hw.ncpu)}"
report="$PWD/.ci/rust-e2e/reports/$job"
rm -rf "$report"
mkdir -p "$report/events" "$report/junit"
finish() {
  local status=$?
  trap - EXIT
  python3 ops/ci/rust-e2e.py report "$job" "$status" || status=$?
  exit "$status"
}
trap finish EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
python3 ops/ci/rust-e2e.py env "$job" >"$report/environment.nul"
while IFS= read -r -d '' key && IFS= read -r -d '' value; do export "$key=$value"; done <"$report/environment.nul"
python3 ops/ci/rust-e2e.py select "$job"
if [[ "$job" == op-reth ]]; then
  python3 ops/ci/rust-workspace-report.py stage-at "$report" rust/op-reth/tests proof-contracts just build-contracts
fi
if [[ -d tmp/testlogs/dependencies ]]; then cp -R tmp/testlogs/dependencies "$report/"; fi
if [[ ! -s "$report/assigned.txt" ]]; then
  echo 'Validated empty E2E shard; no tests assigned.'
  exit 0
fi
run_tests() {
  local name="$1"; shift
  local status=0 split_status=0
  gotestsum --format=testname --junitfile="$report/junit/$name.xml" --jsonfile="$report/events/$name.json" \
    --raw-command -- python3 ops/ci/rust-e2e.py execute "$job" "$@" || status=$?
  bash ops/scripts/split-test-logs.sh "$report/events/$name.json" || split_status=$?
  if [[ "$status" == 0 ]]; then status=$split_status; fi
  return "$status"
}
if [[ "$job" == proof ]]; then
  # Circle xargs executes each top-level test in a new process, even after
  # another test failed. Keep the first failure and collect every invocation.
  first_status=0
  while IFS= read -r name; do
    status=0
    run_tests "$name" "$name" || status=$?
    if [[ "$first_status" == 0 && "$status" != 0 ]]; then first_status=$status; fi
  done <"$report/assigned.txt"
  exit "$first_status"
else
  names=()
  while IFS= read -r name; do names+=("$name"); done <"$report/assigned.txt"
  run_tests package "${names[@]}"
fi
