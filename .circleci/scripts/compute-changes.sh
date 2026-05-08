#!/usr/bin/env bash
# Generic change-detection engine for CircleCI dynamic configuration.
#
# Path-detection rules are defined as DETECT_* environment variables in
# config.yml. This script loops over them, checks git diff output against
# each regex, and emits the results as pipeline parameters JSON.
#
# Testable locally:
#   BASE_REVISION=develop \
#   DEFAULT_DOCKER_IMAGE=cimg/base:2026.03 \
#   DETECT_rust_files_changed='^rust/' \
#   ... \
#   bash .circleci/scripts/compute-changes.sh
set -euo pipefail

# ---------------------------------------------------------------------------
# Detect changed files
# ---------------------------------------------------------------------------
CHANGED=$(git diff --name-only "origin/${BASE_REVISION}...HEAD" 2>/dev/null \
  || git diff --name-only HEAD~1 HEAD || true)

echo "=== Changed files ==="
echo "${CHANGED:-<none>}"
echo "====================="

# Normalizes CircleCI boolean env vars (rendered as 0/1) to JSON true/false.
to_bool() {
  case "${1}" in
    1|true|True|TRUE) echo "true" ;;
    *) echo "false" ;;
  esac
}

# ---------------------------------------------------------------------------
# Build base JSON with passthrough parameters
# ---------------------------------------------------------------------------
json=$(jq -n \
  --arg  c_default_docker_image                "${DEFAULT_DOCKER_IMAGE}" \
  --arg  c_rust_base_image                     "${RUST_BASE_IMAGE}" \
  --arg  c_base_image                          "${BASE_IMAGE}" \
  --argjson c_main_dispatch                    "$(to_bool "${MAIN_DISPATCH}")" \
  --argjson c_fault_proofs_dispatch            "$(to_bool "${FAULT_PROOFS_DISPATCH}")" \
  --argjson c_reproducibility_dispatch         "$(to_bool "${REPRODUCIBILITY_DISPATCH}")" \
  --argjson c_kontrol_dispatch                 "$(to_bool "${KONTROL_DISPATCH}")" \
  --argjson c_cannon_full_test_dispatch        "$(to_bool "${CANNON_FULL_TEST_DISPATCH}")" \
  --argjson c_sdk_dispatch                     "$(to_bool "${SDK_DISPATCH}")" \
  --argjson c_publish_contract_artifacts_dispatch "$(to_bool "${PUBLISH_CONTRACT_ARTIFACTS_DISPATCH}")" \
  --argjson c_stale_check_dispatch             "$(to_bool "${STALE_CHECK_DISPATCH}")" \
  --argjson c_contracts_coverage_dispatch      "$(to_bool "${CONTRACTS_COVERAGE_DISPATCH}")" \
  --argjson c_heavy_fuzz_dispatch              "$(to_bool "${HEAVY_FUZZ_DISPATCH}")" \
  --argjson c_ai_contracts_test_dispatch       "$(to_bool "${AI_CONTRACTS_TEST_DISPATCH}")" \
  --argjson c_rust_ci_dispatch                 "$(to_bool "${RUST_CI_DISPATCH}")" \
  --argjson c_rust_e2e_dispatch                "$(to_bool "${RUST_E2E_DISPATCH}")" \
  --argjson c_l2_fork_test_dispatch            "$(to_bool "${L2_FORK_TEST_DISPATCH}")" \
  --arg  c_github_event_type                   "${GITHUB_EVENT_TYPE}" \
  --arg  c_github_event_action                 "${GITHUB_EVENT_ACTION}" \
  --arg  c_github_event_base64                 "${GITHUB_EVENT_BASE64}" \
  --arg  c_go_cache_version                    "${GO_CACHE_VERSION}" \
  '{
    "c-default_docker_image":                  $c_default_docker_image,
    "c-rust_base_image":                       $c_rust_base_image,
    "c-base_image":                            $c_base_image,
    "c-main_dispatch":                         $c_main_dispatch,
    "c-fault_proofs_dispatch":                 $c_fault_proofs_dispatch,
    "c-reproducibility_dispatch":              $c_reproducibility_dispatch,
    "c-kontrol_dispatch":                      $c_kontrol_dispatch,
    "c-cannon_full_test_dispatch":             $c_cannon_full_test_dispatch,
    "c-sdk_dispatch":                          $c_sdk_dispatch,
    "c-publish_contract_artifacts_dispatch":   $c_publish_contract_artifacts_dispatch,
    "c-stale_check_dispatch":                  $c_stale_check_dispatch,
    "c-contracts_coverage_dispatch":           $c_contracts_coverage_dispatch,
    "c-heavy_fuzz_dispatch":                   $c_heavy_fuzz_dispatch,
    "c-ai_contracts_test_dispatch":            $c_ai_contracts_test_dispatch,
    "c-rust_ci_dispatch":                      $c_rust_ci_dispatch,
    "c-rust_e2e_dispatch":                     $c_rust_e2e_dispatch,
    "c-l2_fork_test_dispatch":                 $c_l2_fork_test_dispatch,
    "c-github-event-type":                     $c_github_event_type,
    "c-github-event-action":                   $c_github_event_action,
    "c-github-event-base64":                   $c_github_event_base64,
    "c-go-cache-version":                      $c_go_cache_version
  }')

# ---------------------------------------------------------------------------
# Auto-detect: iterate over all DETECT_* env vars defined in config.yml
# ---------------------------------------------------------------------------
for var in $(compgen -A variable DETECT_); do
  param_name="${var#DETECT_}"
  pattern="${!var}"

  if [ -n "${CHANGED}" ] && echo "${CHANGED}" | grep -qE "${pattern}"; then
    value=true
  else
    value=false
  fi

  echo "  ${param_name} = ${value}  (pattern: ${pattern})"
  json=$(echo "${json}" | jq --argjson v "${value}" '. + {"c-'"${param_name}"'": $v}')
done

echo "${json}" > /tmp/pipeline-parameters.json

echo "=== Pipeline parameters ==="
cat /tmp/pipeline-parameters.json
echo "==========================="
