#!/usr/bin/env bash
# Ensures UI/API supplied setup params are accepted by the continuation config.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SETUP_CONFIG="${REPO_ROOT}/.circleci/config.yml"
MERGED_CONFIG="${1:-/tmp/merged-config.yml}"

if [[ ! -f "${MERGED_CONFIG}" ]]; then
  echo "ERROR: merged continuation config not found: ${MERGED_CONFIG}" >&2
  echo "Run .circleci/scripts/merge-configs.sh first." >&2
  exit 1
fi

missing=$(
  comm -23 \
    <(yq -r '.parameters | keys | .[]' "${SETUP_CONFIG}" | sort) \
    <(yq -r '.parameters | keys | .[]' "${MERGED_CONFIG}" | sort)
)

if [[ -n "${missing}" ]]; then
  echo "ERROR: setup pipeline params missing from continuation config:" >&2
  while IFS= read -r param; do
    echo "  - ${param}" >&2
  done <<< "${missing}"
  echo >&2
  echo "CircleCI forwards explicitly supplied UI/API parameters into continuation validation." >&2
  echo "Declare these names as unused passthrough params in the continuation config." >&2
  exit 1
fi

mismatched=$(
  jq -r --argjson continuation "$(yq -o=json '.parameters' "${MERGED_CONFIG}")" '
    to_entries[]
    | select($continuation[.key] != .value)
    | "  - \(.key)\n    setup: \(.value | tojson)\n    continuation: \($continuation[.key] | tojson)"
  ' < <(yq -o=json '.parameters' "${SETUP_CONFIG}")
)

if [[ -n "${mismatched}" ]]; then
  echo "ERROR: setup pipeline params do not match continuation passthrough declarations:" >&2
  echo "${mismatched}" >&2
  echo >&2
  echo "Keep passthrough defaults and types identical to the setup config." >&2
  exit 1
fi

echo "All setup pipeline params are declared in the continuation config."
