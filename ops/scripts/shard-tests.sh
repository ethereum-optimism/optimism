#!/bin/bash
# shard-tests.sh — split `go test -list` output across CircleCI parallel nodes.
#
# Reads raw `go test -list` output on stdin, keeps the test names, and writes
# this node's share to stdout, split by historical timing data
# (--timings-type=testname, sourced from the JUnit files jobs upload via
# store_test_results, develop's when the shard-timings job provides them).
# Outside CircleCI parallelism (CIRCLE_NODE_TOTAL unset or 1) every unique test
# name passes through.
#
# Exits 1 when no test names arrive: `go test -list` failed (compile error,
# or a crash in TestMain — which -list executes) or a name filter matched
# nothing. Without this guard every node would receive an empty shard, which
# could let a broken package report green.
set -euo pipefail

ALL_TESTS_FILE=$(mktemp)
SHARD_FILE=$(mktemp)
SPLIT_LOG=$(mktemp)
trap 'rm -f "$ALL_TESTS_FILE" "$SHARD_FILE" "$SPLIT_LOG"' EXIT

# grep exits 1 for no matches, handled below, and above 1 for an error.
grep -E '^Test' > "$ALL_TESTS_FILE" || [ "$?" -eq 1 ]
# A name is listed once per package that has it. Keep one, so it goes to one node,
# and that node's caller runs every package's copy.
LC_ALL=C sort -u -o "$ALL_TESTS_FILE" "$ALL_TESTS_FILE"

TOTAL_TESTS=$(wc -l < "$ALL_TESTS_FILE" | tr -d ' ')
if [ "$TOTAL_TESTS" -eq 0 ]; then
  echo "ERROR: no tests to shard — go test -list failed (compile error or TestMain crash) or the filter matched nothing" >&2
  exit 1
fi
echo "Found $TOTAL_TESTS tests to split across ${CIRCLE_NODE_TOTAL:-1} node(s)" >&2

if [ -z "${CIRCLE_NODE_TOTAL:-}" ] || [ "${CIRCLE_NODE_TOTAL:-1}" -le 1 ]; then
  cat "$ALL_TESTS_FILE"
  exit 0
fi

# Canceled runs can leave partial history. CircleCI's tiny default weight for
# missing timings can put every unknown test on the same otherwise-light shard.
# Give unknown acceptance tests a realistic setup/runtime budget instead.
split_tests() {
  circleci tests split --split-by=timings --timings-type=testname --time-default=1m "$@" "$ALL_TESTS_FILE"
}

# CircleCI's own timings come from this branch's last green run of the job, which
# can predate tests added or fixed on develop since. The shard-timings CI job puts
# develop's timings here, the same file on every node.
TIMINGS_FILE="$(cd "$(dirname "$0")/../.." && pwd)/shard-timings/${CIRCLE_JOB:-}.json"
if [ -s "$TIMINGS_FILE" ]; then
  echo "Splitting by develop's timings from $TIMINGS_FILE" >&2
  split_tests --timings-file "$TIMINGS_FILE" > "$SHARD_FILE" 2> "$SPLIT_LOG" || {
    cat "$SPLIT_LOG" >&2
    exit 1
  }
  cat "$SPLIT_LOG" >&2
  # An unreadable file makes CircleCI split by name, so use its own timings instead.
  if grep -q '^Error reading historical timing data' "$SPLIT_LOG"; then
    echo "warning: can't read $TIMINGS_FILE, splitting by this branch's timings" >&2
    split_tests > "$SHARD_FILE"
  fi
else
  split_tests > "$SHARD_FILE"
fi
# `awk 1` ends the last line with a newline, since callers' `read` and `wc -l` would
# miss a test without one.
awk 1 "$SHARD_FILE"
