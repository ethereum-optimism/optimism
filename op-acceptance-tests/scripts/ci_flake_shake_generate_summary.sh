#!/usr/bin/env bash
set -euo pipefail

# ci_flake_shake_generate_summary.sh
#
# Purpose:
#   Used in CI by the flake-shake report job to transform the aggregated
#   flake-shake report (JSON) into two derivative artifacts:
#     1) daily-summary.json   – compact daily snapshot for downstream tooling
#     2) promotion-ready.json – list of tests with 100% pass rate (promotion candidates)
#
# Usage: ci_flake_shake_generate_summary.sh [REPORT_JSON] [OUT_DIR]
#   $1 REPORT_JSON (optional) – path to flake-shake aggregated report JSON
#       Default: final-report/flake-shake-report.json
#   $2 OUT_DIR (optional) – directory where outputs will be written
#       Default: final-report
#
# Side-effects (env):
#   Exports UNSTABLE_COUNT into $BASH_ENV for later CI steps (if available).
#
# Requirements:
#   - jq must be available in PATH
#
# Notes:
#   - This script is intentionally simple and idempotent; it does not mutate
#     the input report and only writes to OUT_DIR.

REPORT_JSON=${1:-final-report/flake-shake-report.json}
OUT_DIR=${2:-final-report}

if [ ! -f "$REPORT_JSON" ]; then
  echo "ERROR: Report not found at $REPORT_JSON" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

# Print a short human-readable summary to the job logs
echo "=== Flake-Shake Results ==="
STABLE=$(jq '[.tests[] | select(.recommendation == "STABLE")] | length' "$REPORT_JSON")
UNSTABLE=$(jq '[.tests[] | select(.recommendation == "UNSTABLE")] | length' "$REPORT_JSON")
echo "✅ STABLE: $STABLE tests"
echo "⚠️ UNSTABLE: $UNSTABLE tests"
if [ "$UNSTABLE" -gt 0 ]; then
  echo "Unstable tests:"
  jq -r '.tests[] | select(.recommendation == "UNSTABLE") | "  - \(.test_name) (\(.pass_rate)%)"' "$REPORT_JSON"
fi

# Write daily summary JSON (compact per-day snapshot)
jq '{date, gate, total_runs, iterations,
     totals: {
       stable: ([.tests[] | select(.recommendation=="STABLE")] | length),
       unstable: ([.tests[] | select(.recommendation=="UNSTABLE")] | length)
     },
     stable_tests: [
       .tests[] | select(.recommendation=="STABLE") |
       {test_name, package, total_runs, pass_rate}
     ],
     unstable_tests: [
       .tests[] | select(.recommendation=="UNSTABLE") |
       {test_name, package, total_runs, passes, failures, pass_rate}
     ]
   }' "$REPORT_JSON" > "$OUT_DIR/daily-summary.json"

# Write promotion readiness (100% pass) JSON
jq '{ready: [.tests[] | select(.recommendation=="STABLE") | {test_name, package, total_runs, pass_rate, avg_duration, min_duration, max_duration}]}' "$REPORT_JSON" > "$OUT_DIR/promotion-ready.json"

# Export UNSTABLE_COUNT for later CI steps (if BASH_ENV is present)
if [ -n "${BASH_ENV:-}" ]; then
  echo "export UNSTABLE_COUNT=$UNSTABLE" >> "$BASH_ENV"
fi

echo "Wrote: $OUT_DIR/daily-summary.json, $OUT_DIR/promotion-ready.json"

