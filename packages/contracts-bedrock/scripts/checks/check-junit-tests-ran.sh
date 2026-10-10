#!/usr/bin/env bash
set -euo pipefail

# Fails unless a `forge test --junit` report exists and records at least one executed test.
#
# Since Foundry v1.4.0, `forge test` exits 0 when its filters (--match-path, --match-contract,
# --match-test, FOUNDRY_MATCH_PATH, ...) match no tests: it prints a warning and writes no JUnit
# report. Foundry v1.2.3 failed in that case. Without this check, a renamed path, contract or
# test, or a broken generated glob, would turn a CI test gate green without running anything.
#
# Skipped tests count as executed: Forge ran their setUp and they appear in the report. A nonzero
# count is a minimum guard, not proof that every intended test ran.
#
# Set ALLOW_EMPTY_TEST_RUN to a reason to deliberately accept a run with zero tests.
#
# Usage: check-junit-tests-ran.sh <junit-report.xml>

REPORT="${1:?usage: check-junit-tests-ran.sh <junit-report.xml>}"

fail() {
  if [ -n "${ALLOW_EMPTY_TEST_RUN:-}" ]; then
    echo "check-junit-tests-ran: $1; allowed because ALLOW_EMPTY_TEST_RUN=${ALLOW_EMPTY_TEST_RUN}" >&2
    exit 0
  fi
  echo "error: $1" >&2
  echo "Forge exits 0 when test filters match nothing, so an empty report means no tests ran." >&2
  echo "Check the --match-path/--match-contract/--match-test filters and FOUNDRY_MATCH_PATH." >&2
  exit 1
}

if [ ! -s "$REPORT" ]; then
  fail "JUnit report '$REPORT' is missing or empty"
fi

if ! grep -q '<testsuites' "$REPORT"; then
  fail "JUnit report '$REPORT' does not contain a <testsuites> element"
fi

# Count every occurrence, including several on one line. grep exits 1 when nothing matches,
# which must not abort the script under pipefail.
count() { { grep -o "$1" "$REPORT" || true; } | wc -l | tr -d ' '; }
testcases=$(count '<testcase[[:space:]>]')
skipped=$(count '<skipped[[:space:]/>]')

if [ "$testcases" -eq 0 ]; then
  fail "JUnit report '$REPORT' records zero executed tests"
fi

echo "check-junit-tests-ran: $testcases test case(s) recorded in $REPORT ($skipped skipped)"
