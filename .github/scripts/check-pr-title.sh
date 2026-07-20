#!/usr/bin/env bash
# Validates a commit subject against the Scoped Commits format
# (https://scopedcommits.com):
#
#   <scope>: <description>
#
# Multiple comma-separated scopes are permitted (no spaces), and "all" is the
# scope for tree-wide changes. Conventional Commits type prefixes (feat, fix,
# chore, ...) are rejected; see CONTRIBUTING.md for the rationale.
set -euo pipefail

subject="${1-}"

fail() {
  echo "FAIL: ${1}" >&2
  echo "  subject:  '${subject}'" >&2
  echo "  expected: '<scope>: <description>' where scope names the component or area changed" >&2
  echo "  examples: 'op-node: handle unsafe head reorgs', 'op-node,op-batcher: share event loop metrics', 'all: update license headers'" >&2
  echo "  see https://scopedcommits.com and CONTRIBUTING.md" >&2
  exit 1
}

if [[ -z "${subject}" ]]; then
  fail "empty subject"
fi

# GitHub auto-generates 'Revert "<original subject>"' titles for revert PRs.
if [[ "${subject}" =~ ^Revert\ \".+\"$ ]]; then
  echo "OK: revert of '${subject}'"
  exit 0
fi

prefix="${subject%%:*}"
if [[ "${prefix}" == "${subject}" ]]; then
  fail "missing ':' separator between scope and description"
fi

rest="${subject#*:}"
if [[ "${rest}" != ' '* ]]; then
  fail "':' must be followed by a single space"
fi

description="${rest:1}"
if [[ -z "${description// /}" ]]; then
  fail "empty description"
fi

scope_list_regex='^[a-z0-9][a-z0-9._/-]*(,[a-z0-9][a-z0-9._/-]*)*$'
if [[ ! "${prefix}" =~ ${scope_list_regex} ]]; then
  fail "scope '${prefix}' must be lowercase component names ([a-z0-9._/-]), comma-separated without spaces; 'type(scope):' is not accepted"
fi

conventional_types=(build chore feat fix perf refactor revert style test upkeep)
IFS=',' read -ra scopes <<<"${prefix}"
for scope in "${scopes[@]}"; do
  for conventional_type in "${conventional_types[@]}"; do
    if [[ "${scope}" == "${conventional_type}" ]]; then
      fail "'${scope}' is a Conventional Commits type, not a scope — name the component or area the change touches instead"
    fi
  done
done

echo "OK: '${subject}'"
