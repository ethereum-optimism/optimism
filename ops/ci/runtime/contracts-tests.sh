#!/usr/bin/env bash
# The complete standard contract suite, with CircleCI's profiles and features.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}/packages/contracts-bedrock"
# A previous task's or local CLI environment must not silently filter tests,
# lower fuzzing, or enable another feature. Unset feature overrides so the
# checked-in Config.sol defaults remain authoritative for other features.
# shellcheck disable=SC1091  # shared helper is checked separately and resolved at runtime
source "${REPO_ROOT}/ops/ci/runtime/contracts-test-env.sh"

# Fixed paths are per-task: each matrix child has its own isolated filesystem.
# Do not reuse a stale verdict or failure cache from a prior task snapshot.
rm -rf results/reports
mkdir -p results/reports
rm -rf results/results.xml cache/test-failures cache/fuzz cache/invariant
export JUNIT_TEST_PATH=results/results.xml
forge config --json > results/reports/foundry-config.json 2> results/reports/config.stderr.log
python3 "${REPO_ROOT}/ops/ci/runtime/contracts-test-report.py" prepare results/reports/foundry-config.json
forge --version > results/reports/forge-version.txt

status=0
if [[ "${RWX_COMPILED_CONTRACTS:-false}" == true ]]; then
  # The producer used the exact Just Go build and compiled the convention
  # checker. Verify its provenance before consuming compilation state.
  python3 "${REPO_ROOT}/ops/ci/runtime/rwx-contracts-build.py" verify
  forge test --junit >"${JUNIT_TEST_PATH}" 2>results/reports/test.log || status=$?
  cat results/reports/test.log
else
  just test 2>&1 | tee results/reports/test.log || status=$?
fi
if [[ "${status}" -ne 0 ]]; then
  # Diagnostic reruns must never replace the initial failure or its JUnit XML.
  if [[ "${RWX_COMPILED_CONTRACTS:-false}" == true ]]; then
    forge test --rerun -vvv 2>&1 | tee results/reports/rerun-traces.log || true
  else
    just test-rerun 2>&1 | tee results/reports/rerun-traces.log || true
  fi
fi
report_status=0
python3 "${REPO_ROOT}/ops/ci/runtime/contracts-test-report.py" verdict results/results.xml \
  > results/reports/report-validation.log 2>&1 || report_status=$?
cat results/reports/report-validation.log
if [[ "${status}" == 0 && "${report_status}" != 0 ]]; then status="${report_status}"; fi
if [[ "${status}" == 0 ]]; then
  if [[ "${RWX_COMPILED_CONTRACTS:-false}" == true ]]; then
    "${REPO_ROOT}/.ci/contracts-prepare/test-validation" 2>&1 | tee results/reports/test-validation.log || status=$?
  else
    just lint-forge-tests-check-no-build 2>&1 | tee results/reports/test-validation.log || status=$?
  fi
fi
# Keep counterexamples and files emitted by Solidity tests available even when
# the verdict fails. These are reports, not reusable compilation state.
for path in cache/test-failures cache/fuzz cache/invariant .resource-metering.csv .testdata; do
  if [[ -e "${path}" ]]; then
    mkdir -p "results/reports/generated/$(dirname "${path}")"
    cp -a "${path}" "results/reports/generated/${path}"
  fi
done
exit "${status}"
