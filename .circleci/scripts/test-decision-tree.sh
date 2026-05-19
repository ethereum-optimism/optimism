#!/usr/bin/env bash
# Dry-run test for the workflow decision tree in config.yml.
# Extracts the decision tree dynamically using yq, then runs it against
# a table of scenarios, asserting the expected c-run_* flags are set.
#
# Usage:
#   bash .circleci/scripts/test-decision-tree.sh
#
# Requires: jq, yq (same version used in CI)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CONFIG="${REPO_ROOT}/.circleci/config.yml"
OUTPUT="/tmp/pipeline-parameters.json"

# Extract the inline command from the "Compute workflow conditions..." step
STEP_NAME="Compute workflow conditions from pipeline parameters and store in JSON file"
DECISION_TREE=$(yq "
  .jobs.prepare-continuation-config.steps[]
  | select(.run.name == \"${STEP_NAME}\")
  | .run.command
" "${CONFIG}")

if [[ -z "${DECISION_TREE}" ]]; then
  echo "ERROR: Could not extract decision tree from ${CONFIG}" >&2
  exit 1
fi

# Strip the source/init_json/finalize lines — we handle those ourselves
DECISION_TREE=$(echo "${DECISION_TREE}" | grep -v '^\s*set -euo pipefail' \
  | grep -v '^\s*source ' \
  | grep -v '^\s*init_json' \
  | grep -v '^\s*finalize ')

# --- Test harness ---
PASS=0
FAIL=0

run_scenario() {
  local name="${1}"
  local trigger="${2}"
  local branch="${3}"
  local tag="${4}"
  local schedule="${5}"
  local json_seed="${6}"
  shift 6
  local expected=("$@")

  # Seed the JSON
  echo "${json_seed}" > "${OUTPUT}"

  # Set environment
  export TRIGGER_SOURCE="${trigger}"
  export BRANCH="${branch}"
  export TAG="${tag}"
  export SCHEDULE_NAME="${schedule}"

  # Source helpers and run decision tree
  # shellcheck disable=SC1091
  source "${SCRIPT_DIR}/workflow-helpers.sh"
  _json=$(cat "${OUTPUT}")

  eval "${DECISION_TREE}" 2>/dev/null

  # Check expected workflows are enabled
  local all_pass=true
  for wf in "${expected[@]}"; do
    local val
    val=$(echo "${_json}" | jq -r ".\"c-run_${wf}\" // false")
    if [[ "${val}" != "true" ]]; then
      echo "  FAIL: expected c-run_${wf}=true, got ${val}"
      all_pass=false
    fi
  done

  if ${all_pass}; then
    echo "PASS: ${name}"
    PASS=$((PASS + 1))
  else
    echo "FAIL: ${name}"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== Decision Tree Dry-Run Tests ==="
echo ""

# --- Scenarios ---

run_scenario \
  "Tag push → release only" \
  "webhook" "" "v1.0.0" "" \
  '{}' \
  release

run_scenario \
  "PR (feature branch), rust changed" \
  "webhook" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": true, "c-contracts_changed": false, "c-docs_changes_detected": false}' \
  main release contracts_feature_tests_short rust_ci rust_e2e_ci

run_scenario \
  "PR (feature branch), contracts changed" \
  "webhook" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": true, "c-docs_changes_detected": false}' \
  main release contracts_feature_tests rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "PR (feature branch), docs only" \
  "webhook" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-docs_changes_detected": true}' \
  contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "PR (feature branch), docs + rust changed" \
  "webhook" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": true, "c-contracts_changed": false, "c-docs_changes_detected": true}' \
  main release contracts_feature_tests_short rust_ci rust_e2e_ci

run_scenario \
  "PR (feature branch), nothing changed" \
  "webhook" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-docs_changes_detected": false}' \
  main release contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "Merge queue, rust changed" \
  "webhook" "gh-readonly-queue/develop/pr-123" "" "" \
  '{"c-rust_changes_detected": true, "c-contracts_changed": false, "c-docs_changes_detected": false}' \
  main release contracts_feature_tests rust_ci rust_e2e_ci

run_scenario \
  "Merge queue, no changes" \
  "webhook" "gh-readonly-queue/develop/pr-123" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-docs_changes_detected": false}' \
  main release contracts_feature_tests rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "After merge (develop), rust changed" \
  "webhook" "develop" "" "" \
  '{"c-rust_changes_detected": true, "c-contracts_changed": false, "c-docs_changes_detected": true}' \
  main release develop_fault_proofs develop_kontrol_tests contracts_feature_tests rust_ci rust_e2e_ci kona_publish_prestates

run_scenario \
  "After merge (develop), no rust changes" \
  "webhook" "develop" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-docs_changes_detected": false}' \
  main release develop_fault_proofs develop_kontrol_tests contracts_feature_tests rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "Scheduled: build_four_hours" \
  "scheduled_pipeline" "" "" "build_four_hours" \
  '{}' \
  scheduled_todo_issues scheduled_cannon_full_tests

run_scenario \
  "Scheduled: build_daily" \
  "scheduled_pipeline" "" "" "build_daily" \
  '{}' \
  scheduled_preimage_reproducibility scheduled_stale_check scheduled_heavy_fuzz_tests

run_scenario \
  "Scheduled: build_weekly" \
  "scheduled_pipeline" "" "" "build_weekly" \
  '{}' \
  scheduled_weekly_tests scheduled_kona_link_checker

run_scenario \
  "API: main_dispatch (no github event)" \
  "api" "" "" "" \
  '{"c-main_dispatch": true, "c-github-event-type": "__not_set__"}' \
  release main contracts_feature_tests

run_scenario \
  "API: rust_ci_dispatch" \
  "api" "" "" "" \
  '{"c-main_dispatch": false, "c-rust_ci_dispatch": true, "c-github-event-type": "__not_set__"}' \
  release rust_ci

run_scenario \
  "API: github event labeled PR" \
  "api" "" "" "" \
  '{"c-main_dispatch": false, "c-github-event-type": "pull_request", "c-github-event-action": "labeled"}' \
  release close_issue

# --- Summary ---
echo ""
echo "=== Results: ${PASS} passed, ${FAIL} failed ==="

if [[ ${FAIL} -gt 0 ]]; then
  exit 1
fi
