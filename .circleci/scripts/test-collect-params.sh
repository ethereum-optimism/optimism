#!/usr/bin/env bash
# Tests change detection in collect-params.sh (detect and detect_all modes).
# Builds a throwaway git repo, points origin/<base> at its first commit, runs
# the real script against the real routing.yml, and asserts the c-* flags.
#
# Usage:
#   bash .circleci/scripts/test-collect-params.sh
#
# Requires: git, jq, yq (same version used in CI)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COLLECT_SCRIPT="${SCRIPT_DIR}/collect-params.sh"

WORK_DIR="$(mktemp -d)"
REPO_DIR="${WORK_DIR}/repo"
OUTPUT="${WORK_DIR}/params.json"
export OUTPUT
trap 'rm -rf "${WORK_DIR}"' EXIT

git_in_repo() {
  git -C "${REPO_DIR}" -c user.name=test -c user.email=test@example.com -c commit.gpgsign=false "$@"
}

commit_file() { # <path>
  mkdir -p "$(dirname "${REPO_DIR}/${1}")"
  echo "${RANDOM}" > "${REPO_DIR}/${1}"
  git_in_repo add "${1}"
  git_in_repo commit --quiet --message "change ${1}"
}

git init --quiet "${REPO_DIR}"
commit_file README.md
BASE_COMMIT=$(git_in_repo rev-parse HEAD)
git_in_repo update-ref refs/remotes/origin/base "${BASE_COMMIT}"

# --- Test harness ---
PASS=0
FAIL=0

# Runs detect and detect_all for HEAD = base + <changed paths> (space-separated,
# none if empty), then asserts each expected flag given as name=value.
run_scenario() { # <name> <base_revision> <changed_paths> <name=value>...
  local name="${1}" base_revision="${2}" changed_paths="${3}"
  shift 3

  git_in_repo checkout --quiet --detach "${BASE_COMMIT}"
  local changed_path
  for changed_path in ${changed_paths}; do
    commit_file "${changed_path}"
  done

  echo '{}' > "${OUTPUT}"
  local log="${WORK_DIR}/scenario.log" mode
  : > "${log}"
  for mode in detect detect_all; do
    if ! (cd "${REPO_DIR}" && BASE_REVISION="${base_revision}" bash "${COLLECT_SCRIPT}" "${mode}" >> "${log}" 2>&1); then
      echo "FAIL: ${name} (collect-params.sh ${mode} exited non-zero)"
      cat "${log}"
      FAIL=$((FAIL + 1))
      return
    fi
  done

  local all_pass=true expectation flag expected actual
  for expectation in "$@"; do
    flag="${expectation%%=*}"
    expected="${expectation#*=}"
    actual=$(jq -r ".\"c-${flag}\"" "${OUTPUT}")
    if [[ "${actual}" != "${expected}" ]]; then
      echo "  FAIL: expected c-${flag}=${expected}, got ${actual}"
      all_pass=false
    fi
  done

  if ${all_pass}; then
    echo "PASS: ${name}"
    PASS=$((PASS + 1))
  else
    echo "FAIL: ${name}"
    cat "${log}"
    FAIL=$((FAIL + 1))
  fi
}

# --- Scenarios ---

run_scenario \
  "Docs-only change takes the docs fast path" \
  base docs/public-docs/page.md \
  only_docs_changes=true docs_changes_detected=true rust_changes_detected=false contracts_changed=false

run_scenario \
  "Rust change is detected" \
  base rust/kona/lib.rs \
  only_docs_changes=false rust_changes_detected=true contracts_changed=false

run_scenario \
  "Code that no pattern matches skips the docs fast path" \
  base op-node/main.go \
  only_docs_changes=false docs_changes_detected=false rust_changes_detected=false contracts_changed=false

run_scenario \
  "Docs plus code skips the docs fast path" \
  base "docs/public-docs/page.md rust/kona/lib.rs" \
  only_docs_changes=false docs_changes_detected=true rust_changes_detected=true

run_scenario \
  "No change detects nothing" \
  base "" \
  only_docs_changes=false rust_changes_detected=false contracts_changed=false

run_scenario \
  "Missing base ref treats every path as changed" \
  missing-branch docs/public-docs/page.md \
  only_docs_changes=false docs_changes_detected=true rust_changes_detected=true contracts_changed=true circleci_changed=true

# --- Summary ---
echo ""
echo "=== Results: ${PASS} passed, ${FAIL} failed ==="

if [[ ${FAIL} -gt 0 ]]; then
  exit 1
fi
