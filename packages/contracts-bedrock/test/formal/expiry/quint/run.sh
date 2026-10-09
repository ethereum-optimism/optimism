#!/usr/bin/env bash
# Checks the Quint model and FAILS unless every result is the expected one:
#   - safe instances: `Safety` holds and every non-vacuity witness is violated; in `safe` the
#     defense-in-depth property `MessengerSilentAfterUpgrade` holds, and in `safeNoTargetRule` it is
#     violated (safety does not depend on the target rule);
#   - unsafe instances: `NoDoubleSpend` itself is violated (an actual double spend).
# `verify` (Apalache bounded model checking) is authoritative. `simulate` (random traces) is a quick
# sanity pass: safe instances must show no violation and must reach a refund; unsafe-instance results
# are informational, since random search can miss narrow interleavings. `test` runs the scripted
# traces (`run witness*`, `run blocked*` in `safe`) with `quint test`; it needs no Java.
#
# Usage: ./run.sh [test|simulate|verify|all]   (default: all)
# Knobs: SAMPLES, STEPS (simulation); DEPTH (Apalache bound in steps for every check except
# `Safety` in the safe instances, default 15); SAFE_DEPTH (bound for `Safety` in the safe instances,
# default 10, the depth recorded in README.md; deeper runs take hours per step); RESEND_SAFE_DEPTH
# (the same for `safeResendRestarts`, default 9); JOBS (parallel Apalache servers, one per check, on
# ports BASE_PORT..).
# Requires quint 0.33 (npm i -g @informalsystems/quint) and bash >= 4.3; `verify` needs Java 17+
# (quint fetches Apalache). On a shared host, run it under a memory cap (README.md's results used
# 8-16 GB per check).
set -uo pipefail
cd "$(dirname "$0")" || exit 1
# The scheduler below uses `wait -n`, which needs bash 4.3 or later.
if (( BASH_VERSINFO[0] < 4 || (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] < 3) )); then
  echo "needs bash >= 4.3 (this is $BASH_VERSION)" >&2; exit 2
fi

MODE="${1:-all}"
case "$MODE" in test|simulate|verify|all) ;; *) echo "unknown mode: $MODE" >&2; exit 2 ;; esac
SAMPLES="${SAMPLES:-20000}"
STEPS="${STEPS:-30}"
DEPTH="${DEPTH:-15}"
SAFE_DEPTH="${SAFE_DEPTH:-10}"
RESEND_SAFE_DEPTH="${RESEND_SAFE_DEPTH:-9}"
JOBS="${JOBS:-8}"
[[ "$JOBS" =~ ^[1-9][0-9]*$ ]] || { echo "JOBS must be a positive integer: $JOBS" >&2; exit 2; }
BASE_PORT="${BASE_PORT:-8900}"
LOGDIR="${LOGDIR:-logs}"
mkdir -p "$LOGDIR"

SAFE=(safe safeNoTargetRule safeNoMargin safeShorterWindow safeResendRestarts)
WITNESSES=(NoRefundEver NoEdgeRelay NoRefundOfM2 NoRefundOfM3)
UNSAFE=(messengerTrustedPrestaged messengerTrustedNoTargetRule periodBelowWindow expireGeNoMargin
  noRealMessengerCheck noLockboxCheck noSenderCheck nonstandardJoin resendNoRestart duplicateChainId)

# depth_for <expected> <main> <invariant>: the Apalache bound for one check.
depth_for() {
  if [[ "$1" == holds && "$3" == Safety ]]; then
    if [[ "$2" == safeResendRestarts ]]; then echo "$RESEND_SAFE_DEPTH"; else echo "$SAFE_DEPTH"; fi
  else
    echo "$DEPTH"
  fi
}

# classify <exit code> <output file>: quint exits 0 when no violation is found and 1 on a violation.
# Only an invariant counterexample counts as a violation (quint 0.33: `verify` ends with
# "error: found a counterexample", `run` with "error: Invariant violated"). A deadlock trace also
# prints "[violation]", so it is classified as an error, not as the expected violation.
classify() {
  case "$1" in
    0) echo holds ;;
    1) if grep -q -i "deadlock" "$2"; then echo "error(deadlock)"
       elif grep -q -E "^error: (found a counterexample|Invariant violated)" "$2"; then echo violated
       else echo "error(1)"; fi ;;
    *) echo "error($1)" ;;
  esac
}

# Each check: "<expected> <main> <invariant>".
CHECKS=()
for m in "${SAFE[@]}"; do
  CHECKS+=("holds $m Safety")
  for w in "${WITNESSES[@]}"; do CHECKS+=("violated $m $w"); done
done
CHECKS+=("holds safe MessengerSilentAfterUpgrade" "violated safeNoTargetRule MessengerSilentAfterUpgrade")
for m in "${UNSAFE[@]}"; do CHECKS+=("violated $m NoDoubleSpend"); done

FAILURES=0

if [[ "$MODE" == test || "$MODE" == all ]]; then
  out="$LOGDIR/test-safe.log"
  quint test expiry.qnt --main=safe --match='^(witness|blocked)' > "$out" 2>&1
  code=$?
  n="$(grep -c 'passed 1 test' "$out")"
  if [[ $code == 0 && $n -gt 0 ]]; then echo "ok    test safe: $n scripted traces pass"; else echo "FAIL  test safe (see $out)"; FAILURES=$((FAILURES+1)); fi
fi

if [[ "$MODE" == simulate || "$MODE" == all ]]; then
  for m in "${SAFE[@]}"; do
    for spec in "holds Safety" "violated NoRefundEver"; do
      read -r want inv <<<"$spec"
      out="$LOGDIR/sim-$m-$inv.log"
      quint run expiry.qnt --main="$m" --invariant="$inv" --max-samples="$SAMPLES" --max-steps="$STEPS" > "$out" 2>&1
      got="$(classify $? "$out")"
      if [[ "$got" == "$want" ]]; then echo "ok    sim $want: $m $inv"; else echo "FAIL  sim expected $want, got $got: $m $inv (see $out)"; FAILURES=$((FAILURES+1)); fi
    done
  done
  for m in "${UNSAFE[@]}"; do
    out="$LOGDIR/sim-$m-NoDoubleSpend.log"
    quint run expiry.qnt --main="$m" --invariant=NoDoubleSpend --max-samples="$SAMPLES" --max-steps="$STEPS" > "$out" 2>&1
    got="$(classify $? "$out")"
    case "$got" in
      violated) echo "info  sim found a double spend: $m" ;;
      holds) echo "info  sim found no double spend (Apalache decides): $m" ;;
      *) echo "FAIL  sim tool error $got: $m (see $out)"; FAILURES=$((FAILURES+1)) ;;
    esac
  done
fi

if [[ "$MODE" == verify || "$MODE" == all ]]; then
  # One Apalache server per running check, so checks run in parallel.
  i=0
  for c in "${CHECKS[@]}"; do
    read -r want m inv <<<"$c"
    port=$((BASE_PORT + i))
    out="$LOGDIR/verify-$m-$inv.log"
    d="$(depth_for "$want" "$m" "$inv")"
    ( start=$(date +%s)
      quint verify expiry.qnt --main="$m" --invariant="$inv" --max-steps="$d" \
        --server-endpoint="localhost:$port" > "$out" 2>&1
      code=$?
      echo "EXIT=$code SECONDS=$(( $(date +%s) - start ))" >> "$out" ) &
    i=$((i + 1))
    # Keep at most JOBS checks running; start the next one as soon as any finishes.
    while (( $(jobs -rp | wc -l) >= JOBS )); do wait -n; done
  done
  wait
  for c in "${CHECKS[@]}"; do
    read -r want m inv <<<"$c"
    out="$LOGDIR/verify-$m-$inv.log"
    code="$(sed -n 's/^EXIT=\([0-9]*\).*/\1/p' "$out" | tail -n1)"
    secs="$(sed -n 's/.*SECONDS=\([0-9]*\).*/\1/p' "$out" | tail -n1)"
    got="$(classify "${code:-99}" "$out")"
    d="$(depth_for "$want" "$m" "$inv")"
    if [[ "$got" == "$want" ]]; then echo "ok    verify depth=$d ${secs}s $want: $m $inv"; else echo "FAIL  verify expected $want, got $got: $m $inv (see $out)"; FAILURES=$((FAILURES+1)); fi
  done
fi

echo "failures: $FAILURES"
exit $((FAILURES > 0))
