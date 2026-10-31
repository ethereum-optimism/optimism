#!/usr/bin/env bash
# Adapt RWX event metadata to CircleCI's workflow routing, read-only.
#
# RWX selects workloads with the CircleCI routing data and policy
# (.circleci/routing.yml, .circleci/scripts/collect-params.sh and
# compute-workflow-conditions.sh), so the two providers cannot drift. This
# adapter supplies the inputs those scripts document and adds one RWX rule:
# a change under .rwx/ or ops/ci/ selects every workload, like a .circleci/
# change does for CircleCI.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

: "${CI_COMMIT_SHA:?CI_COMMIT_SHA must identify the checked-out commit}"
: "${RWX_VALUES:?RWX_VALUES must name the task output-values directory}"
CI_EVENT="${CI_EVENT:-push}"
CI_BRANCH="${CI_BRANCH:-}"
CI_TAG="${CI_TAG:-}"
CI_BASE_REVISION="${CI_BASE_REVISION:-develop}"

if [[ ! "${CI_COMMIT_SHA}" =~ ^[0-9a-f]{40}$ ]]; then
  echo "ERROR: CI_COMMIT_SHA must be a full commit SHA." >&2
  exit 1
fi
if [[ "$(git rev-parse HEAD)" != "${CI_COMMIT_SHA}" ]]; then
  echo "ERROR: checked-out HEAD differs from CI_COMMIT_SHA." >&2
  exit 1
fi
if [[ "${CI_EVENT}" != push ]]; then
  echo "ERROR: RWX routes push events only; got CI_EVENT=${CI_EVENT}." >&2
  exit 1
fi
if [[ "${CI_CACHE_WARM:-false}" == true ]]; then
  # A referenced task must succeed even when an `if` expression could bypass
  # its value. Keep a real routing dependency for warm runs, but select no
  # verdicts. Compiler tasks explicitly opt in through their warm flag.
  if [[ "${CI_BRANCH}" != develop ]]; then
    echo 'ERROR: cache warming is restricted to develop.' >&2
    exit 1
  fi
  mkdir -p .ci "${RWX_VALUES}"
  printf '%s\n' '{"cache-rebuild":true,"c-run_main":false,"c-run_rust_ci":false,"c-run_rust_e2e_ci":false,"c-run_contracts_feature_tests":false}' \
    > .ci/pipeline-parameters.json
  printf 'false\n' > "${RWX_VALUES}/run-main"
  printf 'false\n' > "${RWX_VALUES}/run-rust-ci"
  printf 'false\n' > "${RWX_VALUES}/run-rust-e2e-ci"
  echo "Prepared compiler-only cache warming for ${CI_COMMIT_SHA}."
  exit 0
fi
if [[ -z "${CI_BRANCH}" && -z "${CI_TAG}" ]]; then
  echo "ERROR: a push must identify its branch or tag." >&2
  exit 1
fi
git check-ref-format --branch "${CI_BASE_REVISION}" > /dev/null

# Fetch the protected comparison branch explicitly. A shallow checkout must
# include enough history to find the merge base; an unavailable base fails the
# task rather than treating an incomplete diff as a docs-only change.
base_refspec="+refs/heads/${CI_BASE_REVISION}:refs/remotes/origin/${CI_BASE_REVISION}"
if [[ "$(git rev-parse --is-shallow-repository)" == true ]]; then
  git fetch --no-tags --unshallow origin "${CI_COMMIT_SHA}" "${base_refspec}"
else
  git fetch --no-tags origin "${base_refspec}"
fi
git merge-base "origin/${CI_BASE_REVISION}" HEAD > /dev/null

mkdir -p .ci "${RWX_VALUES}"
# collect-params.sh always appends to this fixed file; start from a clean one.
PARAMS=/tmp/pipeline-parameters.json
printf '{}\n' > "${PARAMS}"
BASE_REVISION="${CI_BASE_REVISION}" bash .circleci/scripts/collect-params.sh detect
BASE_REVISION="${CI_BASE_REVISION}" bash .circleci/scripts/collect-params.sh detect_all

# The RWX rule. `rwx run` can also upload an uncommitted patch, which the
# committed diff above cannot see: treat any such change like a CI change, so
# a local experiment runs every workload instead of a docs-only shortcut.
committed="$(git diff --name-only --no-renames "origin/${CI_BASE_REVISION}...HEAD")"
uncommitted="$(git status --porcelain --untracked-files=all -- . ':(exclude).ci')"
if grep -qE '^(\.rwx|ops/ci)/' <<< "${committed}" || [[ -n "${uncommitted}" ]]; then
  echo "RWX or uncommitted CI change: selecting every workload."
  jq '. + {"c-circleci_changed": true, "c-contracts_changed": true, "c-rust_changes_detected": true, "c-only_docs_changes": false}' \
    "${PARAMS}" > "${PARAMS}.next"
  mv "${PARAMS}.next" "${PARAMS}"
fi

TRIGGER_SOURCE=webhook BRANCH="${CI_BRANCH}" TAG="${CI_TAG}" SCHEDULE_NAME="" \
  CIRCLE_PROJECT_USERNAME=ethereum-optimism CIRCLE_PROJECT_REPONAME=optimism OUTPUT="${PARAMS}" \
  bash .circleci/scripts/compute-workflow-conditions.sh
# The policy writes only the workflows it enables. RWX route tasks and gate
# aggregates require these four flags as booleans, so write false explicitly.
jq '{"c-run_main": false, "c-run_rust_ci": false, "c-run_rust_e2e_ci": false, "c-run_contracts_feature_tests": false} + .' \
  "${PARAMS}" > .ci/pipeline-parameters.json

jq -r '."c-run_main" // false' .ci/pipeline-parameters.json > "${RWX_VALUES}/run-main"
jq -r '."c-run_rust_ci" // false' .ci/pipeline-parameters.json > "${RWX_VALUES}/run-rust-ci"
jq -r '."c-run_rust_e2e_ci" // false' .ci/pipeline-parameters.json > "${RWX_VALUES}/run-rust-e2e-ci"
