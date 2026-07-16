#!/usr/bin/env bash
set -euo pipefail

format="${1:-text}"
input="$(cat)"

command="$(
  jq -r '
    .tool_input.command
    // .input.command
    // .input.cmd
    // .arguments.command
    // .arguments.cmd
    // empty
  ' <<<"${input}" 2>/dev/null || true
)"

[[ -n "${command}" ]] || exit 0

# Match common ways agents invoke git commit from a shell command.
if ! grep -qE '(^|[;&|]|\|\|)\s*(env\s+[^;&|]*\s+)?git(\s+-C\s+\S+)?\s+commit(\s|$)' <<<"${command}"; then
  exit 0
fi

context="REMINDER: You just created a commit in the Optimism repo. Before pushing, run \`ops/scripts/precommit-targets.sh --run\` from the repo/worktree and report the result. If you intentionally skip it, say exactly why before pushing."

case "${format}" in
  claude)
    jq -n --arg ctx "${context}" '{
      hookSpecificOutput: {
        hookEventName: "PostToolUse",
        additionalContext: $ctx
      }
    }'
    ;;
  text|codex)
    printf '%s\n' "${context}"
    ;;
  *)
    printf 'unknown reminder format: %s\n' "${format}" >&2
    exit 2
    ;;
esac
