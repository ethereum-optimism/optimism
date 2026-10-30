#!/usr/bin/env bash
# Moves --match-path/--mp out of the forge args and narrows TEST_FILES to the files it matches.
# A CLI --match-path overrides each pass's FOUNDRY_MATCH_PATH, which would run a matched file
# under both EVMs. Callers must run with `set -f` so glob arguments are not expanded.
# Usage: apply_match_path "$@"; reads and updates TEST_FILES and sets FORGE_ARGS.

apply_match_path() {
  local match_path=""
  FORGE_ARGS=()
  while [ $# -gt 0 ]; do
    case "$1" in
      --match-path | --mp)
        match_path="$2"
        shift 2
        ;;
      --match-path=* | --mp=*)
        match_path="${1#*=}"
        shift
        ;;
      *)
        FORGE_ARGS+=("$1")
        shift
        ;;
    esac
  done
  [ -n "$match_path" ] || return 0
  # shellcheck disable=SC2053 # The pattern is a glob on purpose.
  TEST_FILES=$(while read -r f; do [[ "$f" == ${match_path#./} ]] && echo "$f"; done <<< "$TEST_FILES" || true)
  [ -n "$TEST_FILES" ] || {
    echo "No test files match --match-path $match_path" >&2
    return 1
  }
}
