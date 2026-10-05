#!/usr/bin/env bash
# Dry-run test for the workflow routing policy (compute-workflow-conditions.sh).
# Seeds the params JSON, sets the trigger/branch/tag/schedule environment, runs
# the routing script, then asserts the expected c-run_* flags are (or are not)
# set in the resulting JSON.
#
# Usage:
#   mise exec yq jq -- bash ops/ci/test-decision-tree.sh
#
# Requires: Bash 4+, jq, yq (same versions used in CI)
set -euo pipefail

if (( BASH_VERSINFO[0] < 4 )); then
  echo "Routing fixtures require Bash 4+ (CI uses Linux); macOS /bin/bash 3 is unsupported." >&2
  exit 2
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROUTING_SCRIPT="${SCRIPT_DIR}/compute-workflow-conditions.sh"
CIRCLECI_SCRIPTS="${SCRIPT_DIR}/../../.circleci/scripts"

# Use an isolated params file so the test never writes the pipeline's real
# /tmp/pipeline-parameters.json. The routing script's finalize writes c-run_*
# flags, which the later collect-params/compute steps would otherwise inherit.
# Exported so the routing script (via workflow-helpers.sh) writes here too.
TEST_DIR="$(mktemp -d)"
OUTPUT="${TEST_DIR}/params.json"
export OUTPUT
trap 'rm -rf "${TEST_DIR}"' EXIT

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
  local all_pass=true
  if ! CI_EVENT="${trigger}" CI_BRANCH="${branch}" CI_TAG="${tag}" CI_SCHEDULE_NAME="${schedule}" \
    bash "${ROUTING_SCRIPT}" >"${TEST_DIR}/route.log" 2>&1; then
    cat "${TEST_DIR}/route.log"
    echo "  FAIL: shared routing policy exited unsuccessfully"
    all_pass=false
  fi

  # CircleCI must translate its legacy trigger metadata into exactly the same
  # routing output, including passthrough parameters and skip gates.
  local legacy_trigger
  case "${trigger}" in
    push) legacy_trigger=webhook ;;
    schedule) legacy_trigger=scheduled_pipeline ;;
    dispatch) legacy_trigger=api ;;
  esac
  echo "${json_seed}" > "${TEST_DIR}/circleci.json"
  if ! OUTPUT="${TEST_DIR}/circleci.json" TRIGGER_SOURCE="${legacy_trigger}" \
    BRANCH="${branch}" TAG="${tag}" SCHEDULE_NAME="${schedule}" \
    bash "${CIRCLECI_SCRIPTS}/compute-workflow-conditions.sh" >"${TEST_DIR}/circleci.log" 2>&1; then
    cat "${TEST_DIR}/circleci.log"
    echo "  FAIL: CircleCI routing adapter exited unsuccessfully"
    all_pass=false
  fi
  if [[ "$(jq -S . "${OUTPUT}")" != "$(jq -S . "${TEST_DIR}/circleci.json")" ]]; then
    echo "  FAIL: CircleCI and shared routing results differ"
    all_pass=false
  fi

  local _json
  _json=$(cat "${OUTPUT}")

  # Pin the complete enabled set as well, so an unexpected expensive or
  # side-effecting workflow cannot sneak into a trigger path.
  local expected_set actual_set
  expected_set=$(printf '%s\n' "${expected[@]}" | sort)
  actual_set=$(jq -r 'to_entries[] | select(.key | startswith("c-run_")) | select(.value == true) | .key | ltrimstr("c-run_")' "${OUTPUT}" | sort)
  if [[ "${expected_set}" != "${actual_set}" ]]; then
    echo "  FAIL: complete enabled workflow set differs"
    all_pass=false
  fi

  # Check expected workflows are enabled
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
  "push" "" "v1.0.0" "" \
  '{}' \
  release

run_scenario \
  "PR (feature branch), rust changed" \
  "push" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": true, "c-contracts_changed": false, "c-docs_changes_detected": false}' \
  main release contracts_feature_tests_short rust_ci rust_e2e_ci

run_scenario \
  "PR (feature branch), contracts changed" \
  "push" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": true, "c-docs_changes_detected": false}' \
  main release contracts_feature_tests rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "PR (feature branch), docs only" \
  "push" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-docs_changes_detected": true, "c-only_docs_changes": true}' \
  ci_gate_skip contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "PR (feature branch), docs + rust changed" \
  "push" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": true, "c-contracts_changed": false, "c-docs_changes_detected": true, "c-only_docs_changes": false}' \
  main release contracts_feature_tests_short rust_ci rust_e2e_ci

# Footgun guard: a docs PR that also touches code outside the detection regexes
# (e.g., op-node/, op-batcher/, any new top-level dir) MUST run main. Without
# the all-match check, this scenario previously hit the docs-only fast path.
run_scenario \
  "PR (feature branch), docs + undetected code (footgun guard)" \
  "push" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-docs_changes_detected": true, "c-only_docs_changes": false}' \
  main release contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "PR (feature branch), nothing changed" \
  "push" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-circleci_changed": false, "c-docs_changes_detected": false, "c-only_docs_changes": false}' \
  main release contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip \
  --not circleci_schedule_trigger_check

run_scenario \
  "PR (feature branch), CircleCI changed" \
  "push" "feat/my-thing" "" "" \
  '{"c-rust_changes_detected": true, "c-contracts_changed": true, "c-circleci_changed": true, "c-docs_changes_detected": false, "c-only_docs_changes": false}' \
  main release contracts_feature_tests rust_ci rust_e2e_ci circleci_schedule_trigger_check

run_scenario \
  "Merge queue, rust changed" \
  "push" "gh-readonly-queue/develop/pr-123" "" "" \
  '{"c-rust_changes_detected": true, "c-contracts_changed": false, "c-docs_changes_detected": false, "c-only_docs_changes": false}' \
  main release contracts_feature_tests rust_ci rust_e2e_ci

run_scenario \
  "Merge queue, no changes" \
  "push" "gh-readonly-queue/develop/pr-123" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-docs_changes_detected": false, "c-only_docs_changes": false}' \
  main release contracts_feature_tests rust_ci_gate_short rust_e2e_gate_skip

run_scenario \
  "Merge queue, docs only" \
  "push" "gh-readonly-queue/develop/pr-123" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-docs_changes_detected": true, "c-only_docs_changes": true}' \
  ci_gate_skip contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip \
  --not main release contracts_feature_tests rust_ci rust_e2e_ci

# Develop runs the full post-merge set unconditionally. The two scenarios below
# seed opposite change-detection results and assert an identical routing, which
# is what pins that behaviour: on a develop push the changed-file list is always
# empty (BASE_REVISION is develop, so HEAD is the base), so any path gating here
# would be dead code that never fires in production.
run_scenario \
  "After merge (develop), empty change set (production reality)" \
  "push" "develop" "" "" \
  '{"c-rust_changes_detected": false, "c-contracts_changed": false, "c-circleci_changed": false, "c-docs_changes_detected": false, "c-only_docs_changes": false}' \
  main release publish_contract_artifacts develop_fault_proofs develop_kontrol_tests contracts_feature_tests rust_ci rust_e2e_ci kona_publish_prestates circleci_schedule_trigger_check \
  --not rust_ci_gate_short rust_e2e_gate_skip ci_gate_skip contracts_feature_tests_short

run_scenario \
  "After merge (develop), change detection must not alter routing" \
  "push" "develop" "" "" \
  '{"c-rust_changes_detected": true, "c-contracts_changed": true, "c-circleci_changed": true, "c-docs_changes_detected": true, "c-only_docs_changes": true}' \
  main release publish_contract_artifacts develop_fault_proofs develop_kontrol_tests contracts_feature_tests rust_ci rust_e2e_ci kona_publish_prestates circleci_schedule_trigger_check \
  --not rust_ci_gate_short rust_e2e_gate_skip ci_gate_skip contracts_feature_tests_short

run_scenario \
  "Scheduled: build_four_hours" \
  "schedule" "" "" "build_four_hours" \
  '{}' \
  scheduled_todo_issues scheduled_cannon_full_tests

run_scenario \
  "Scheduled: build_daily" \
  "schedule" "" "" "build_daily" \
  '{}' \
  scheduled_preimage_reproducibility scheduled_stale_check scheduled_heavy_fuzz_tests scheduled_daily_tests scheduled_sp1_elf_smoke circleci_schedule_trigger_check

run_scenario \
  "Scheduled: build_weekly" \
  "schedule" "" "" "build_weekly" \
  '{}' \
  scheduled_rust_nightly_bump

run_scenario \
  "API: main_dispatch (no github event)" \
  "dispatch" "" "" "" \
  '{"c-main_dispatch": true, "c-github-event-type": "__not_set__"}' \
  release main contracts_feature_tests

run_scenario \
  "API: rust_ci_dispatch" \
  "dispatch" "" "" "" \
  '{"c-main_dispatch": false, "c-rust_ci_dispatch": true, "c-github-event-type": "__not_set__"}' \
  release rust_ci

run_scenario \
  "API: rust_nightly_bump_dispatch" \
  "dispatch" "" "" "" \
  '{"c-main_dispatch": false, "c-rust_nightly_bump_dispatch": true, "c-github-event-type": "__not_set__"}' \
  release scheduled_rust_nightly_bump

run_scenario \
  "API: publish_contract_artifacts_dispatch" \
  "dispatch" "" "" "" \
  '{"c-main_dispatch": false, "c-publish_contract_artifacts_dispatch": true, "c-github-event-type": "__not_set__"}' \
  release publish_contract_artifacts

run_scenario \
  "API: github event labeled PR" \
  "dispatch" "" "" "" \
  '{"c-main_dispatch": false, "c-github-event-type": "pull_request", "c-github-event-action": "labeled"}' \
  release close_issue

# Exercise actual changed path lists through the shared collector, rather than
# hand-seeding change booleans. The CircleCI collector must produce the same JSON.
run_changed_scenario() {
  local fixture="${1}" branch="${2}"
  shift 2
  echo '{}' > "${OUTPUT}"
  echo '{}' > "${TEST_DIR}/circleci.json"
  local mode
  for mode in detect detect_all; do
    CHANGED_FILES_FILE="${SCRIPT_DIR}/fixtures/routing/${fixture}.txt" \
      bash "${SCRIPT_DIR}/collect-params.sh" "${mode}" >/dev/null
    OUTPUT="${TEST_DIR}/circleci.json" CHANGED_FILES_FILE="${SCRIPT_DIR}/fixtures/routing/${fixture}.txt" \
      bash "${CIRCLECI_SCRIPTS}/collect-params.sh" "${mode}" >/dev/null
  done
  if [[ "$(jq -S . "${OUTPUT}")" != "$(jq -S . "${TEST_DIR}/circleci.json")" ]]; then
    echo "FAIL: ${fixture} collector compatibility"
    FAIL=$((FAIL + 1))
  fi
  run_scenario "Changed paths: ${fixture} (${branch})" push "${branch}" "" "" "$(cat "${OUTPUT}")" "$@"
}

run_changed_scenario docs-only feat/routing \
  ci_gate_skip contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip \
  --not main release contracts_feature_tests rust_ci rust_e2e_ci
run_changed_scenario docs-and-unknown feat/routing \
  main release contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip \
  --not ci_gate_skip contracts_feature_tests rust_ci rust_e2e_ci
run_changed_scenario rust feat/routing \
  main release contracts_feature_tests_short rust_ci rust_e2e_ci --not ci_gate_skip contracts_feature_tests
run_changed_scenario contracts feat/routing \
  main release contracts_feature_tests rust_ci_gate_short rust_e2e_gate_skip --not ci_gate_skip rust_ci rust_e2e_ci
for fixture in circleci rwx shared-ci; do
  run_changed_scenario "${fixture}" feat/routing \
    main release contracts_feature_tests rust_ci rust_e2e_ci circleci_schedule_trigger_check \
    --not ci_gate_skip contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip
done
run_changed_scenario empty develop \
  main release publish_contract_artifacts develop_fault_proofs develop_kontrol_tests \
  contracts_feature_tests rust_ci rust_e2e_ci kona_publish_prestates circleci_schedule_trigger_check
run_changed_scenario docs-only gh-readonly-queue/develop/pr-123 \
  ci_gate_skip contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip --not main release
run_changed_scenario docs-and-unknown gh-readonly-queue/develop/pr-123 \
  main release contracts_feature_tests rust_ci_gate_short rust_e2e_gate_skip --not ci_gate_skip

run_scenario "Passthrough parameters survive, detection and dispatch parameters do not" \
  push feat/routing "" "" \
  '{"c-default_docker_image":"example/image:1","c-go-cache-version":"test","c-go_fresh_tests_effective":true,"c-contract_coverage_replay_effective":true,"c-main_dispatch":true,"c-only_docs_changes":true}' \
  ci_gate_skip contracts_feature_tests_short rust_ci_gate_short rust_e2e_gate_skip
if jq -e '."c-default_docker_image" == "example/image:1" and ."c-go-cache-version" == "test" and ."c-go_fresh_tests_effective" == true and ."c-contract_coverage_replay_effective" == true and (has("c-main_dispatch") | not) and (has("c-only_docs_changes") | not)' "${OUTPUT}" >/dev/null; then
  echo "PASS: parameter JSON contract"
  PASS=$((PASS + 1))
else
  echo "FAIL: parameter JSON contract"
  FAIL=$((FAIL + 1))
fi

if CHANGED_FILES_FILE="${TEST_DIR}/missing.txt" bash "${SCRIPT_DIR}/collect-params.sh" detect >/dev/null 2>&1; then
  echo "FAIL: missing changed-file input must fail"
  FAIL=$((FAIL + 1))
else
  echo "PASS: missing changed-file input fails"
  PASS=$((PASS + 1))
fi

# An early match followed by more than a pipe buffer must not turn into a
# false detection under pipefail. Cover both any-match and all-match modes.
large_changes="${TEST_DIR}/large-changes.txt"
{
  echo 'ops/ci/probe.sh'
  for ((i = 0; i < 10000; i++)); do echo "docs/public-docs/page-${i}.md"; done
} >"${large_changes}"
echo '{}' >"${OUTPUT}"
for mode in detect detect_all; do
  CHANGED_FILES_FILE="${large_changes}" bash "${SCRIPT_DIR}/collect-params.sh" "${mode}" >/dev/null
done
if jq -e '."c-rust_changes_detected" == true and ."c-only_docs_changes" == false' "${OUTPUT}" >/dev/null; then
  echo 'PASS: large changed-file list retains early matches and nonmatches'
  PASS=$((PASS + 1))
else
  echo 'FAIL: large changed-file list routing'
  FAIL=$((FAIL + 1))
fi

# --- Summary ---
echo ""
echo "=== Results: ${PASS} passed, ${FAIL} failed ==="

if [[ ${FAIL} -gt 0 ]]; then
  exit 1
fi
