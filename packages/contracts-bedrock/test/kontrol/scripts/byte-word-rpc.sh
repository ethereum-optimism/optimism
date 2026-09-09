#!/bin/bash
set -Eeuo pipefail

# Keep --solver-transcript visible to pyk so it closes the server with SIGINT.
# Each proof starts its own server; fresh PID suffixes avoid file collisions.
test "$1" = --solver-transcript
prefix="$2-$$"
shift 2
mkdir -p "$(dirname "$prefix")"
exec kore-rpc-booster \
  --solver-transcript "$prefix.smt2" \
  --log-file "$prefix.log" \
  --log-level SMT --log-level Simplify --log-level SimplifyKore \
  --log-format json --pretty-print decoded \
  "$@"
