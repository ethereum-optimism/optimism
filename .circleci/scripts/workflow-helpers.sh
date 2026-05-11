#!/usr/bin/env bash
# workflow-helpers.sh — predicates and helpers for the workflow decision tree.
# Source this file from the "Compute workflow conditions" step in config.yml.
# The decision tree itself lives in config.yml for full visibility.
set -euo pipefail

OUTPUT="/tmp/pipeline-parameters.json"
_json=$(cat "${OUTPUT}")

# --- run: enable one or more workflows ---
run() {
  for wf in "$@"; do
    _json=$(echo "${_json}" | jq '. + {"c-run_'"${wf}"'": true}')
    echo "  [enable] c-run_${wf}"
  done
}

# --- flush: write accumulated changes and print summary ---
flush() {
  echo "${_json}" > "${OUTPUT}"
  echo "=== Enabled workflows ==="
  echo "${_json}" | jq -r 'to_entries[] | select(.key | startswith("c-run_")) | select(.value == true) | "  \(.key)"'
  echo "========================="
}

# --- Trigger type predicates ---
is_scheduled()   { [[ "${TRIGGER_SOURCE}" == "scheduled_pipeline" ]]; }
is_webhook()     { [[ "${TRIGGER_SOURCE}" == "webhook" ]]; }
is_api()         { [[ "${TRIGGER_SOURCE}" == "api" ]]; }
has_tag()        { [[ -n "${TAG}" ]]; }

# --- Branch predicates ---
is_merge_queue() { [[ "${BRANCH}" =~ ^gh-readonly-queue/ ]]; }
is_develop()     { [[ "${BRANCH}" == "develop" ]]; }

# --- Path change predicates (read from JSON) ---
rust_changed()      { [[ $(echo "${_json}" | jq '.["c-rust_changes_detected"]') == "true" ]]; }
contracts_changed() { [[ $(echo "${_json}" | jq '.["c-contracts_changed"]') == "true" ]]; }
docs_changed()      { [[ $(echo "${_json}" | jq '.["c-docs_changes_detected"]') == "true" ]]; }

# --- Dispatch predicates (read from JSON) ---
main_dispatch()              { [[ $(echo "${_json}" | jq '.["c-main_dispatch"]') == "true" ]]; }
fault_proofs_dispatch()      { [[ $(echo "${_json}" | jq '.["c-fault_proofs_dispatch"]') == "true" ]]; }
kontrol_dispatch()           { [[ $(echo "${_json}" | jq '.["c-kontrol_dispatch"]') == "true" ]]; }
cannon_full_test_dispatch()  { [[ $(echo "${_json}" | jq '.["c-cannon_full_test_dispatch"]') == "true" ]]; }
reproducibility_dispatch()   { [[ $(echo "${_json}" | jq '.["c-reproducibility_dispatch"]') == "true" ]]; }
stale_check_dispatch()       { [[ $(echo "${_json}" | jq '.["c-stale_check_dispatch"]') == "true" ]]; }
heavy_fuzz_dispatch()        { [[ $(echo "${_json}" | jq '.["c-heavy_fuzz_dispatch"]') == "true" ]]; }
ai_contracts_test_dispatch() { [[ $(echo "${_json}" | jq '.["c-ai_contracts_test_dispatch"]') == "true" ]]; }
l2_fork_test_dispatch()      { [[ $(echo "${_json}" | jq '.["c-l2_fork_test_dispatch"]') == "true" ]]; }
rust_ci_dispatch()           { [[ $(echo "${_json}" | jq '.["c-rust_ci_dispatch"]') == "true" ]]; }
rust_e2e_dispatch()          { [[ $(echo "${_json}" | jq '.["c-rust_e2e_dispatch"]') == "true" ]]; }

# --- Event predicates ---
event_not_set() { [[ $(echo "${_json}" | jq -r '.["c-github-event-type"]') == "__not_set__" ]]; }
is_close_issue_event() {
  [[ $(echo "${_json}" | jq -r '.["c-github-event-type"]') == "pull_request" ]] &&
  [[ $(echo "${_json}" | jq -r '.["c-github-event-action"]') == "labeled" ]];
}
