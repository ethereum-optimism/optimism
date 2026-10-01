#!/usr/bin/env bash
# The complete standard contract suite, with CircleCI's profiles and features.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${REPO_ROOT}/packages/contracts-bedrock"
: "${CI_BRANCH:?CI_BRANCH must name the tested branch}"
CONTRACT_FEATURE="${CONTRACT_FEATURE:-main}"
case "${CONTRACT_FEATURE}" in
  main) FEATURE_ENV="" ;;
  CUSTOM_GAS_TOKEN) FEATURE_ENV=SYS_FEATURE__CUSTOM_GAS_TOKEN ;;
  OPTIMISM_PORTAL_INTEROP) FEATURE_ENV=DEV_FEATURE__OPTIMISM_PORTAL_INTEROP ;;
  ZK_DISPUTE_GAME) FEATURE_ENV=DEV_FEATURE__ZK_DISPUTE_GAME ;;
  *) echo "Unknown standard contract feature: ${CONTRACT_FEATURE}" >&2; exit 1 ;;
esac

# A previous task's or local CLI environment must not silently filter tests,
# lower fuzzing, or enable another feature. Unset feature overrides so the
# checked-in Config.sol defaults remain authoritative for other features.
for name in $(compgen -e); do
  case "${name}" in FOUNDRY_*|DAPP_*|DEV_FEATURE__*|SYS_FEATURE__*) unset "${name}" ;; esac
done
export CONTRACT_FEATURE
if [[ "${CI_BRANCH}" == develop ]]; then export FOUNDRY_PROFILE=ci; else export FOUNDRY_PROFILE=liteci; fi
if [[ -n "${FEATURE_ENV}" ]]; then export "${FEATURE_ENV}=true"; fi
export FORK_TEST=false L2_FORK_TEST=false L2CM_ACTIVATION_TEST=false
unset ETH_RPC_URL ETH_RPC_JWT ETH_RPC_HEADERS ETHERSCAN_API_KEY MAINNET_RPC_URL \
  FORK_RPC_URL FORK_BLOCK_NUMBER L2_FORK_RPC_URL L2_FORK_BLOCK_NUMBER

# Fixed paths are per-task: each matrix child has its own isolated filesystem.
# Do not reuse a stale verdict or failure cache from a prior task snapshot.
rm -rf results/reports
mkdir -p results/reports
rm -f results/results.xml cache/test-failures
export JUNIT_TEST_PATH=results/results.xml
forge config --json >results/reports/foundry-config.json 2>results/reports/config.stderr.log
python3 "${REPO_ROOT}/ops/ci/contracts-test-report.py" prepare results/reports/foundry-config.json
forge --version >results/reports/forge-version.txt

status=0
just test 2>&1 | tee results/reports/test.log || status=$?
if [[ "${status}" -ne 0 ]]; then
  # Diagnostic reruns must never replace the initial failure or its JUnit XML.
  just test-rerun 2>&1 | tee results/reports/rerun-traces.log || true
fi
report_status=0
python3 "${REPO_ROOT}/ops/ci/contracts-test-report.py" verdict results/results.xml \
  >results/reports/report-validation.log 2>&1 || report_status=$?
cat results/reports/report-validation.log
if [[ "${status}" == 0 && "${report_status}" != 0 ]]; then status="${report_status}"; fi
if [[ "${status}" == 0 ]]; then
  just lint-forge-tests-check-no-build 2>&1 | tee results/reports/test-validation.log || status=$?
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
