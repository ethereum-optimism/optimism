#!/usr/bin/env bash
set -euo pipefail

# ci_flake_shake_calc_iterations.sh
#
# Purpose:
#   Compute the number of iterations each CircleCI parallel worker should run
#   and export FLAKE_SHAKE_ITERATIONS and FLAKE_SHAKE_WORKER_ID into $BASH_ENV.
#
# Usage:
#   ci_flake_shake_calc_iterations.sh <TOTAL_ITER> [WORKERS] [WORKER_ID]
#
# Arguments:
#   TOTAL_ITER (required): total iterations across all workers.
#   WORKERS    (optional): number of parallel workers (defaults to $CIRCLE_NODE_TOTAL or 1).
#   WORKER_ID  (optional): 1-based worker id (defaults to $((CIRCLE_NODE_INDEX+1)) or 1).
#
# Notes:
#   - Remainder iterations are distributed one-by-one to the first N workers.

TOTAL_ITER=${1:?TOTAL_ITER is required}
WORKERS=${2:-${CIRCLE_NODE_TOTAL:-1}}
WORKER_ID=${3:-$((${CIRCLE_NODE_INDEX:-0} + 1))}

ITER_PER_WORKER=$(( TOTAL_ITER / WORKERS ))
REMAINDER=$(( TOTAL_ITER % WORKERS ))

# Distribute the remainder fairly: the first $REMAINDER workers get one extra iteration
if [ "$WORKER_ID" -le "$REMAINDER" ] && [ "$REMAINDER" -ne 0 ]; then
  ITER_COUNT=$(( ITER_PER_WORKER + 1 ))
else
  ITER_COUNT=$ITER_PER_WORKER
fi

echo "Worker $WORKER_ID running $ITER_COUNT of $TOTAL_ITER iterations"
if [ -n "${BASH_ENV:-}" ]; then
  echo "export FLAKE_SHAKE_ITERATIONS=$ITER_COUNT" >> "$BASH_ENV"
  echo "export FLAKE_SHAKE_WORKER_ID=$WORKER_ID" >> "$BASH_ENV"
fi

exit 0


