#!/bin/bash
# gotestsum-split.sh — Drop-in gotestsum wrapper that splits JSON logs per test.
#
# Usage: gotestsum-split.sh [gotestsum args...]
#
# Drop-in replacement for gotestsum. Passes all arguments through, then splits
# the --jsonfile output into per-test log files via split-test-logs.sh.
#
# If --jsonfile is not provided, the wrapper adds one automatically using a
# default path (tmp/testlogs/log.json relative to cwd), ensuring per-test
# logs are always generated.
#
# Preserves gotestsum's exit code so the split runs even on test failure
# (when per-test logs are most useful for debugging).

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Scan args for --jsonfile.
json_file=""
next_is_jsonfile=false
for arg in "$@"; do
  if $next_is_jsonfile; then
    json_file="$arg"
    break
  fi
  case "$arg" in
    --jsonfile=*) json_file="${arg#--jsonfile=}" ; break ;;
    --jsonfile)   next_is_jsonfile=true ;;
  esac
done

# If --jsonfile wasn't provided, add one automatically.
if [ -z "$json_file" ]; then
  json_file="tmp/testlogs/log.json"
  mkdir -p "$(dirname "$json_file")"
  set -- --jsonfile="$json_file" "$@"
fi

gotestsum "$@"
GOTESTSUM_EXIT=$?

"$SCRIPT_DIR/split-test-logs.sh" "$json_file"
SPLIT_EXIT=$?

if [ $SPLIT_EXIT -ne 0 ]; then
  exit $SPLIT_EXIT
fi
exit $GOTESTSUM_EXIT
