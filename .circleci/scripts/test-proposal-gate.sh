#!/usr/bin/env bash
# Asserts the proposal/** release-branch gate (ci-gate-proposals) requires every
# job that required-contracts-ci requires, plus the NUT, op-deployer and Go
# test jobs it adds. Both lists are kept by hand in continue/main.yml; this
# stops the release-branch gate from silently falling behind develop's
# contracts gate when a job is added or renamed.
#
# Also asserts that every job an entry of the ci-gate-proposals workflow
# requires is in that workflow: the entries alias `main`'s, so a new dependency
# added there is otherwise missing here, and a continuation that fails to
# compile produces no checks at all.
#
# Usage:
#   .circleci/scripts/test-proposal-gate.sh
#
# Requires: yq (same version used in CI)
set -euo pipefail

MAIN="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/continue/main.yml"
WORKFLOW="ci-gate-proposals"

# Jobs the proposal gate requires beyond required-contracts-ci's list.
PROPOSAL_ONLY_JOBS="go-tests
check-nut-locks
check-nut-prefork-states
nut-provenance-verify
op-deployer-forge-version"

# requires_of <workflow> <job> <name>: the job names that gate requires, one
# per line. A requires entry is either "job" or {job: terminal}.
requires_of() {
  yq -r "
    explode(.)
    | .workflows.\"${1}\".jobs[]
    | select(has(\"${2}\"))
    | .\"${2}\"
    | select((.name // \"${2}\") == \"${3}\")
    | .requires[]
    | ((select(tag == \"!!str\")), (select(tag == \"!!map\") | keys | .[]))
  " "${MAIN}" | sort -u
}

# fail_with_list <message> <lines>
fail_with_list() {
  echo "ERROR: ${1}" >&2
  while IFS= read -r line; do
    echo "  - ${line}" >&2
  done <<< "${2}"
  exit 1
}

expected=$({
  requires_of contracts-feature-tests required-contracts-ci required-contracts-ci
  echo "${PROPOSAL_ONLY_JOBS}"
} | sort -u)
actual=$(requires_of "${WORKFLOW}" ci-gate "${WORKFLOW}")

if [[ -z "${expected}" ]]; then
  echo "ERROR: found no requires on required-contracts-ci in ${MAIN}" >&2
  exit 1
fi
if [[ -z "${actual}" ]]; then
  echo "ERROR: found no requires on ${WORKFLOW} in ${MAIN}" >&2
  exit 1
fi

missing=$(comm -23 <(echo "${expected}") <(echo "${actual}"))
if [[ -n "${missing}" ]]; then
  fail_with_list "${WORKFLOW} does not require every job it must. Missing:" "${missing}"
fi

# Job names in the workflow (matrix names stay unexpanded) and every job the
# non-gate entries require; the gate's own requires are checked above.
present=$(yq -r "
  explode(.) | .workflows.\"${WORKFLOW}\".jobs[] | to_entries[0] | .value.name // .key
" "${MAIN}" | sort -u)
needed=$(yq -r "
  explode(.) | .workflows.\"${WORKFLOW}\".jobs[] | to_entries[0]
  | select(.key != \"ci-gate\") | .value.requires[]?
  | ((select(tag == \"!!str\")), (select(tag == \"!!map\") | keys | .[]))
" "${MAIN}" | sort -u)
unmet=$(comm -23 <(echo "${needed}") <(echo "${present}"))
if [[ -n "${unmet}" ]]; then
  fail_with_list "${WORKFLOW} entries require jobs that are not in the workflow:" "${unmet}"
fi

echo "${WORKFLOW} covers required-contracts-ci and its own jobs ($(echo "${actual}" | wc -l | tr -d ' ') requires), all dependencies present."
