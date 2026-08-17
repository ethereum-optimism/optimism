#!/usr/bin/env bash
# Dry-run test for the workflow routing policy (compute-workflow-conditions.sh).
# Seeds the params JSON, sets the trigger/branch/tag/schedule environment, runs
# the routing script, then asserts the expected c-run_* flags are (or are not)
# set in the resulting JSON.
#
# Usage:
#   bash .circleci/scripts/test-decision-tree.sh
#
# Requires: jq, yq (same version used in CI)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROUTING_SCRIPT="${SCRIPT_DIR}/compute-workflow-conditions.sh"

# Use an isolated params file so the test never writes the pipeline's real
# /tmp/pipeline-parameters.json. The routing script's finalize writes c-run_*
# flags, which the later collect-params/compute steps would otherwise inherit.
# Exported so the routing script (via workflow-helpers.sh) writes here too.
OUTPUT="$(mktemp)"
export OUTPUT
trap 'rm -f "${OUTPUT}"' EXIT

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
  local expected=()
  local unexpected=()
  local collect_expected=true
  for wf in "$@"; do
    if [[ "${wf}" == "--not" ]]; then
      collect_expected=false
      continue
    fi
    if ${collect_expected}; then
      expected+=("${wf}")
    else
      unexpected+=("${wf}")
    fi
  done

  # Seed the JSON and run the routing policy as the pipeline would.
  echo "${json_seed}" > "${OUTPUT}"
  TRIGGER_SOURCE="${trigger}" BRANCH="${branch}" TAG="${tag}" SCHEDULE_NAME="${schedule}" \
    bash "${ROUTING_SCRIPT}" >/dev/null || true

  local _json
  _json=$(cat "${OUTPUT}")

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
  for wf in "${unexpected[@]}"; do
    local val
    val=$(echo "${_json}" | jq -r ".\"c-run_${wf}\" // false")
    if [[ "${val}" != "false" ]]; then
      echo "  FAIL: expected c-run_${wf}=false, got ${val}"
      all_pass=false
    fi
  done

  # The authoritative post-merge signal must exist only for webhook pushes to
  # develop, never for PR, merge queue, tag, API, or scheduled pipelines.
  local expected_gate=false
  if [[ "${trigger}" == "webhook" && "${branch}" == "develop" && -z "${tag}" ]]; then
    expected_gate=true
  fi
  local actual_gate
  actual_gate=$(echo "${_json}" | jq -r '."c-run_post_merge_gate" // false')
  if [[ "${actual_gate}" != "${expected_gate}" ]]; then
    echo "  FAIL: expected c-run_post_merge_gate=${expected_gate}, got ${actual_gate}"
    all_pass=false
  fi

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
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-docs_changes_detected": true, "c-only_docs_changes": true}' \
  ci_gate_skip contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "PR (feature branch), docs + rust changed" \
  "webhook" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": true, "c-contracts_changed": false, "c-docs_changes_detected": true, "c-only_docs_changes": false}' \
  main release contracts_feature_tests_short rust_ci rust_e2e_ci

# Footgun guard: a docs PR that also touches code outside the detection regexes
# (e.g., op-node/, op-batcher/, any new top-level dir) MUST run main. Without
# the all-match check, this scenario previously hit the docs-only fast path.
run_scenario \
  "PR (feature branch), docs + undetected code (footgun guard)" \
  "webhook" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-docs_changes_detected": true, "c-only_docs_changes": false}' \
  main release contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "PR (feature branch), nothing changed" \
  "webhook" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-circleci_changed": false, "c-docs_changes_detected": false, "c-only_docs_changes": false}' \
  main release contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip \
  --not circleci_schedule_trigger_check

run_scenario \
  "PR (feature branch), CircleCI changed" \
  "webhook" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": true, "c-contracts_changed": true, "c-circleci_changed": true, "c-docs_changes_detected": false, "c-only_docs_changes": false}' \
  main release contracts_feature_tests rust_ci rust_e2e_ci circleci_schedule_trigger_check

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
  main release publish_contract_artifacts develop_fault_proofs develop_kontrol_tests contracts_feature_tests post_merge_gate rust_ci rust_e2e_ci kona_publish_prestates

run_scenario \
  "After merge (develop), no rust changes" \
  "webhook" "develop" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-docs_changes_detected": false}' \
  main release publish_contract_artifacts develop_fault_proofs develop_kontrol_tests contracts_feature_tests post_merge_gate rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "Scheduled: build_four_hours" \
  "scheduled_pipeline" "" "" "build_four_hours" \
  '{}' \
  scheduled_todo_issues scheduled_cannon_full_tests

run_scenario \
  "Scheduled: build_daily" \
  "scheduled_pipeline" "" "" "build_daily" \
  '{}' \
  scheduled_preimage_reproducibility scheduled_stale_check scheduled_heavy_fuzz_tests scheduled_daily_tests scheduled_sp1_elf_smoke circleci_schedule_trigger_check

run_scenario \
  "Scheduled: build_weekly" \
  "scheduled_pipeline" "" "" "build_weekly" \
  '{}' \
  scheduled_rust_nightly_bump

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
  "API: rust_nightly_bump_dispatch" \
  "api" "" "" "" \
  '{"c-main_dispatch": false, "c-rust_nightly_bump_dispatch": true, "c-github-event-type": "__not_set__"}' \
  release scheduled_rust_nightly_bump

run_scenario \
  "API: publish_contract_artifacts_dispatch" \
  "api" "" "" "" \
  '{"c-main_dispatch": false, "c-publish_contract_artifacts_dispatch": true, "c-github-event-type": "__not_set__"}' \
  release publish_contract_artifacts

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
