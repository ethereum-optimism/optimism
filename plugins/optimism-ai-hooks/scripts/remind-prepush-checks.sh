#!/usr/bin/env bash
set -euo pipefail

input="$(cat)"

workdir="$(
  jq -r '
    .tool_input.workdir
    // .input.workdir
    // .workdir
    // .cwd
    // empty
  ' <<<"${input}" 2>/dev/null || true
)"

repo_root=""
if [[ -n "${workdir}" ]]; then
  repo_root="$(git -C "${workdir}" rev-parse --show-toplevel 2>/dev/null || true)"
fi
if [[ -z "${repo_root}" ]]; then
  repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
fi

shared_hook="${repo_root}/.agents/hooks/remind-prepush-checks.sh"
[[ -x "${shared_hook}" ]] || exit 0

printf '%s' "${input}" | bash "${shared_hook}" codex
