#!/usr/bin/env bash
# Self-test for check-pr-title.sh. Runs a table of subjects through the
# checker and asserts the expected verdict.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
checker="${script_dir}/check-pr-title.sh"
failures=0

expect() {
  local expected="${1}"
  local subject="${2}"
  local actual
  if bash "${checker}" "${subject}" >/dev/null 2>&1; then
    actual="pass"
  else
    actual="fail"
  fi
  if [[ "${actual}" != "${expected}" ]]; then
    echo "MISMATCH: expected ${expected}, got ${actual}: '${subject}'" >&2
    failures=$((failures + 1))
  fi
}

expect pass 'op-node: handle unsafe head reorgs'
expect pass 'op-node,op-batcher: share event loop metrics'
expect pass 'contracts-bedrock: bump solidity to 0.8.30'
expect pass 'docs: add span batches how-to'
expect pass 'ci: shard cannon tests'
expect pass 'all: update license headers'
expect pass 'rust/kona: clean up derive pipeline'
expect pass 'op-e2e: cover deposit-only blocks'
expect pass 'Revert "op-node: handle unsafe head reorgs"'

expect fail ''
expect fail 'fix(op-revm): discard journal after non-deposit tx errors'
expect fail 'fix: harden op-challenger images'
expect fail 'feat: add supernode config flag'
expect fail 'feat!: breaking change'
expect fail 'chore: update dependencies'
expect fail 'upkeep: update dependencies'
expect fail 'test: add coverage'
expect fail 'op-node fix things without separator'
expect fail 'op-node:missing space after colon'
expect fail 'Op-Node: uppercase scope'
expect fail ': empty scope'
expect fail 'op-node:   '
expect fail 'op-node, op-batcher: space after comma'
expect fail 'op-node,fix: conventional type hidden in scope list'

if [[ "${failures}" -gt 0 ]]; then
  echo "${failures} test case(s) failed" >&2
  exit 1
fi
echo "all test cases passed"
