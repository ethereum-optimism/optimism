#!/usr/bin/env bash
# Detects which paths changed relative to the base revision and emits a
# pipeline-parameters.json file consumed by the continuation step.
#
# All pipeline parameter values are injected as environment variables by the
# calling CircleCI job so this script is testable locally:
#
#   BASE_REVISION=develop \
#   DEFAULT_DOCKER_IMAGE=cimg/base:2026.03 \
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

# Returns "true" if any changed file matches the ERE pattern, "false" otherwise.
changed() {
  [ -n "${CHANGED}" ] && echo "${CHANGED}" | grep -qE "$1" && echo "true" || echo "false"
}

# ---------------------------------------------------------------------------
# Compute path-based boolean flags
# ---------------------------------------------------------------------------

# rust/ directory changed (used to gate rust-ci workflows)
RUST_FILES_CHANGED=$(changed '^rust/')

# Contracts, circleci config, GitHub Actions, or root tooling files changed
CONTRACTS_CHANGED=$(changed \
  '^(packages/contracts-bedrock|\.circleci|\.github|ops/check-changed)/|^(package\.json|mise\.toml)$')

# Public docs changed (gates docs-ci workflow)
DOCS_CHANGES_DETECTED=$(changed '^docs/public-docs/')

# Rust CI: triggered by rust/ or .circleci/ changes only
RUST_CHANGES_DETECTED=$(changed '^(rust|\.circleci)/')

# Rust E2E: broader scope — also triggered by op-e2e/ changes.
# Previously this shared c-rust_changes_detected with rust-ci, which caused
# op-e2e/ changes to also trigger the full rust-ci suite. Now explicit.
RUST_E2E_CHANGES_DETECTED=$(changed '^(rust|op-e2e|\.circleci)/')

# ---------------------------------------------------------------------------
# Emit parameters JSON — use jq for correct escaping of string values
# ---------------------------------------------------------------------------
jq -n \
  --arg  c_default_docker_image                "${DEFAULT_DOCKER_IMAGE}" \
  --arg  c_rust_base_image                     "${RUST_BASE_IMAGE}" \
  --arg  c_base_image                          "${BASE_IMAGE}" \
  --argjson c_main_dispatch                    "${MAIN_DISPATCH}" \
  --argjson c_fault_proofs_dispatch            "${FAULT_PROOFS_DISPATCH}" \
  --argjson c_reproducibility_dispatch         "${REPRODUCIBILITY_DISPATCH}" \
  --argjson c_kontrol_dispatch                 "${KONTROL_DISPATCH}" \
  --argjson c_cannon_full_test_dispatch        "${CANNON_FULL_TEST_DISPATCH}" \
  --argjson c_sdk_dispatch                     "${SDK_DISPATCH}" \
  --argjson c_publish_contract_artifacts_dispatch "${PUBLISH_CONTRACT_ARTIFACTS_DISPATCH}" \
  --argjson c_stale_check_dispatch             "${STALE_CHECK_DISPATCH}" \
  --argjson c_contracts_coverage_dispatch      "${CONTRACTS_COVERAGE_DISPATCH}" \
  --argjson c_heavy_fuzz_dispatch              "${HEAVY_FUZZ_DISPATCH}" \
  --argjson c_ai_contracts_test_dispatch       "${AI_CONTRACTS_TEST_DISPATCH}" \
  --argjson c_rust_ci_dispatch                 "${RUST_CI_DISPATCH}" \
  --argjson c_rust_e2e_dispatch                "${RUST_E2E_DISPATCH}" \
  --argjson c_l2_fork_test_dispatch            "${L2_FORK_TEST_DISPATCH}" \
  --arg  c_github_event_type                   "${GITHUB_EVENT_TYPE}" \
  --arg  c_github_event_action                 "${GITHUB_EVENT_ACTION}" \
  --arg  c_github_event_base64                 "${GITHUB_EVENT_BASE64}" \
  --arg  c_go_cache_version                    "${GO_CACHE_VERSION}" \
  --argjson c_rust_files_changed               "${RUST_FILES_CHANGED}" \
  --argjson c_contracts_changed                "${CONTRACTS_CHANGED}" \
  --argjson c_docs_changes_detected            "${DOCS_CHANGES_DETECTED}" \
  --argjson c_rust_changes_detected            "${RUST_CHANGES_DETECTED}" \
  --argjson c_rust_e2e_changes_detected        "${RUST_E2E_CHANGES_DETECTED}" \
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
    "c-go-cache-version":                      $c_go_cache_version,
    "c-rust_files_changed":                    $c_rust_files_changed,
    "c-contracts_changed":                     $c_contracts_changed,
    "c-docs_changes_detected":                 $c_docs_changes_detected,
    "c-rust_changes_detected":                 $c_rust_changes_detected,
    "c-rust_e2e_changes_detected":             $c_rust_e2e_changes_detected
  }' \
  > /tmp/pipeline-parameters.json

echo "=== Pipeline parameters ==="
cat /tmp/pipeline-parameters.json
echo "==========================="
