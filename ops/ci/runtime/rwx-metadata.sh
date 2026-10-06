#!/usr/bin/env bash
# Adapt RWX event metadata to the shared workflow-routing contract.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

: "${CI_COMMIT_SHA:?CI_COMMIT_SHA must identify the checked-out commit}"
: "${RWX_VALUES:?RWX_VALUES must name the task output-values directory}"
export CI_EVENT="${CI_EVENT:-push}"
export CI_BRANCH="${CI_BRANCH:-}"
export CI_TAG="${CI_TAG:-}"
export CI_SCHEDULE_NAME="${CI_SCHEDULE_NAME:-}"
export CI_BASE_REVISION="${CI_BASE_REVISION:-develop}"

if [[ ! "${CI_COMMIT_SHA}" =~ ^[0-9a-f]{40}$ ]]; then
  echo "ERROR: CI_COMMIT_SHA must be a full commit SHA." >&2
  exit 1
fi
if [[ "$(git rev-parse HEAD)" != "${CI_COMMIT_SHA}" ]]; then
  echo "ERROR: checked-out HEAD differs from CI_COMMIT_SHA." >&2
  exit 1
fi
if [[ "${CI_CACHE_WARM:-false}" == true ]]; then
  # A referenced task must succeed even when an `if` expression could bypass
  # its value. Keep a real routing dependency for warm runs, but select no
  # verdicts. Compiler tasks explicitly opt in through their warm flag.
  case "${CI_BRANCH}" in
    develop | codex/rwx-ci-pilot) ;;
    *) echo 'ERROR: cache warming is restricted to develop and the pilot rehearsal.' >&2; exit 1 ;;
  esac
  mkdir -p .ci "${RWX_VALUES}"
  printf '%s\n' '{"cache-rebuild":true,"c-run_main":false,"c-run_rust_ci":false,"c-run_rust_e2e_ci":false,"c-run_contracts_feature_tests":false}' \
    >.ci/pipeline-parameters.json
  printf 'false\n' >"${RWX_VALUES}/run-main"
  printf 'false\n' >"${RWX_VALUES}/run-rust-ci"
  printf 'false\n' >"${RWX_VALUES}/run-rust-e2e-ci"
  echo "Prepared compiler-only cache warming for ${CI_COMMIT_SHA}."
  exit 0
fi
if [[ "${CI_EVENT}" == push && -z "${CI_BRANCH}" && -z "${CI_TAG}" ]]; then
  echo "ERROR: a push must identify its branch or tag." >&2
  exit 1
fi
git check-ref-format --branch "${CI_BASE_REVISION}" >/dev/null

# Fetch the protected comparison branch explicitly. A shallow checkout must
# include enough history to find the merge base; an unavailable base fails the
# task rather than treating an incomplete diff as a docs-only change.
base_refspec="+refs/heads/${CI_BASE_REVISION}:refs/remotes/origin/${CI_BASE_REVISION}"
if [[ "$(git rev-parse --is-shallow-repository)" == true ]]; then
  git fetch --no-tags --unshallow origin "${CI_COMMIT_SHA}" "${base_refspec}"
else
  git fetch --no-tags origin "${base_refspec}"
fi
git merge-base "origin/${CI_BASE_REVISION}" HEAD >/dev/null

mkdir -p .ci "${RWX_VALUES}"
export OUTPUT="${REPO_ROOT}/.ci/pipeline-parameters.json"
export CHANGED_FILES_FILE="${REPO_ROOT}/.ci/changed-files.txt"
git diff --name-only "origin/${CI_BASE_REVISION}...HEAD" >"${CHANGED_FILES_FILE}"
# The CLI can apply an uncommitted patch, including newly added files. Include
# it in routing so local configuration experiments exercise the intended jobs.
git diff --name-only HEAD >>"${CHANGED_FILES_FILE}"
git ls-files --others --exclude-standard >>"${CHANGED_FILES_FILE}"
sort -u "${CHANGED_FILES_FILE}" -o "${CHANGED_FILES_FILE}"

printf '{}\n' >"${OUTPUT}"
bash ops/ci/runtime/collect-params.sh detect
bash ops/ci/runtime/collect-params.sh detect_all
bash ops/ci/runtime/compute-workflow-conditions.sh

jq -r '."c-run_main" // false' "${OUTPUT}" >"${RWX_VALUES}/run-main"
jq -r '."c-run_rust_ci" // false' "${OUTPUT}" >"${RWX_VALUES}/run-rust-ci"
jq -r '."c-run_rust_e2e_ci" // false' "${OUTPUT}" >"${RWX_VALUES}/run-rust-e2e-ci"
