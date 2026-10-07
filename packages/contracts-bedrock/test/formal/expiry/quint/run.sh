#!/usr/bin/env bash
# Checks the Quint model and FAILS unless every result is the expected one:
#   - safe instances: `Safety` holds and every non-vacuity witness is violated; in `safe` the
#     defense-in-depth property `MessengerSilentAfterUpgrade` holds, and in `safeNoTargetRule` it is
#     violated (safety does not depend on the target rule);
#   - unsafe instances: `NoDoubleSpend` itself is violated (an actual double spend).
# `verify` (Apalache bounded model checking) is authoritative. `simulate` (random traces) is a quick
# sanity pass: safe instances must show no violation and must reach a refund; unsafe-instance results
# are informational, since random search can miss narrow interleavings.
#
# Usage: ./run.sh [simulate|verify|all]   (default: all)
# Knobs: SAMPLES, STEPS (simulation); DEPTH (Apalache bound in steps); JOBS (parallel Apalache
# servers, one per check, on ports BASE_PORT..).
# Requires quint (npm i -g @informalsystems/quint); `verify` needs Java 17+ (quint fetches Apalache).
set -uo pipefail
cd "$(dirname "$0")"

MODE="${1:-all}"
case "$MODE" in simulate|verify|all) ;; *) echo "unknown mode: $MODE" >&2; exit 2 ;; esac
SAMPLES="${SAMPLES:-20000}"
STEPS="${STEPS:-30}"
DEPTH="${DEPTH:-15}"
JOBS="${JOBS:-8}"
BASE_PORT="${BASE_PORT:-8900}"
LOGDIR="${LOGDIR:-logs}"
mkdir -p "$LOGDIR"

SAFE=(safe safeNoTargetRule safeNoMargin safeShorterWindow safeResendRestarts)
WITNESSES=(NoRefundEver NoEdgeRelay NoRefundOfM2 NoRefundOfM3)
UNSAFE=(messengerTrustedPrestaged messengerTrustedNoTargetRule periodBelowWindow expireGeNoMargin
  noRealMessengerCheck noLockboxCheck noSenderCheck nonstandardJoin resendNoRestart duplicateChainId)

# classify <exit code> <output file>: quint exits 0 when no violation is found and 1 on a violation.
classify() {
  case "$1" in
    0) echo holds ;;
    1) if grep -q -E "\[violation\]|found a counterexample|Invariant violated" "$2"; then echo violated; else echo "error(1)"; fi ;;
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
  pids=()
  for c in "${CHECKS[@]}"; do
    read -r want m inv <<<"$c"
    port=$((BASE_PORT + (i % JOBS)))
    out="$LOGDIR/verify-$m-$inv.log"
    ( start=$(date +%s)
      quint verify expiry.qnt --main="$m" --invariant="$inv" --max-steps="$DEPTH" \
        --server-endpoint="localhost:$port" > "$out" 2>&1
      code=$?
      echo "EXIT=$code SECONDS=$(( $(date +%s) - start ))" >> "$out" ) &
    pids+=($!)
    i=$((i + 1))
    if (( ${#pids[@]} >= JOBS )); then wait "${pids[0]}"; pids=("${pids[@]:1}"); fi
  done
  wait
  for c in "${CHECKS[@]}"; do
    read -r want m inv <<<"$c"
    out="$LOGDIR/verify-$m-$inv.log"
    code="$(sed -n 's/^EXIT=\([0-9]*\).*/\1/p' "$out" | tail -n1)"
    secs="$(sed -n 's/.*SECONDS=\([0-9]*\).*/\1/p' "$out" | tail -n1)"
    got="$(classify "${code:-99}" "$out")"
    if [[ "$got" == "$want" ]]; then echo "ok    verify depth=$DEPTH ${secs}s $want: $m $inv"; else echo "FAIL  verify expected $want, got $got: $m $inv (see $out)"; FAILURES=$((FAILURES+1)); fi
  done
fi

echo "failures: $FAILURES"
exit $((FAILURES > 0))
