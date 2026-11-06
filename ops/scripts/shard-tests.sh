#!/bin/bash
# shard-tests.sh — split `go test -list` output across CircleCI parallel nodes.
#
# Reads raw `go test -list` output on stdin, keeps the test names, and writes
# this node's share to stdout, split by historical timing data
# (--timings-type=testname, sourced from the JUnit files jobs upload via
# store_test_results). Outside CircleCI parallelism (CIRCLE_NODE_TOTAL unset
# or 1) every unique test name passes through.
#
# Exits 1 when no test names arrive: `go test -list` failed (compile error,
# or a crash in TestMain — which -list executes) or a name filter matched
# nothing. Without this guard every node would receive an empty shard, which
# could let a broken package report green.
set -euo pipefail

ALL_TESTS_FILE=$(mktemp)
trap 'rm -f "$ALL_TESTS_FILE"' EXIT

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
# `awk 1` ends the last line with a newline, since callers' `read` and `wc -l` would
# miss a test without one.
circleci tests split --split-by=timings --timings-type=testname --time-default=1m "$ALL_TESTS_FILE" | awk 1
