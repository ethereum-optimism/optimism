#!/usr/bin/env bash
# Temporary provider-alignment extensions to the permanent routing fixtures.
set -euo pipefail
CIRCLECI_SCRIPTS="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.circleci/scripts" && pwd)"
compare_route() {
  local trigger="$1" branch="$2" tag="$3" schedule="$4" json_seed="$5"
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
    return 1
  fi
  if [[ "$(jq -S . "${OUTPUT}")" != "$(jq -S . "${TEST_DIR}/circleci.json")" ]]; then
    echo "  FAIL: CircleCI and shared routing results differ"
    return 1
  fi

  return 0
}
compare_collector() {
  local fixture="$1" mode="$2"
  if [[ "$mode" == detect ]]; then echo '{}' > "${TEST_DIR}/circleci.json"; fi
    OUTPUT="${TEST_DIR}/circleci.json" CHANGED_FILES_FILE="${SCRIPT_DIR}/fixtures/routing/${fixture}.txt" \
      bash "${CIRCLECI_SCRIPTS}/collect-params.sh" "${mode}" >/dev/null
  if [[ "$(jq -S . "${OUTPUT}")" != "$(jq -S . "${TEST_DIR}/circleci.json")" ]]; then
    echo "FAIL: ${fixture} collector compatibility"
    FAIL=$((FAIL + 1))
  fi
}
# The Circle orb checks scripts individually without following sourced files.
# shellcheck disable=SC1091
source "$(dirname "${BASH_SOURCE[0]}")/../../tests/test-decision-tree.sh"
