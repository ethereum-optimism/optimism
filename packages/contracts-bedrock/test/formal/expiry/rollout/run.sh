#!/usr/bin/env bash
# Checks the rollout model and FAILS unless every result is the expected one:
#   - safe instances: `Safety` holds, and every non-vacuity witness is violated (rolloutSafe: all
#     witnesses; rolloutWindowAtP and downgradeHardened: the subsets listed below);
#   - unsafe instances (one activation condition dropped each): `NoDoubleSpend` itself is violated.
# `verify` (Apalache bounded model checking) is authoritative. `simulate` (random traces) is a quick
# sanity pass for the safe instances.
#
# `test` runs the scripted traces (`run cex*`, `run witness*`, `run blocked*`) in every instance:
# concrete double-spend traces for the unsafe instances and reachability witnesses for the safe ones.
#
# Usage: ./run.sh [test|simulate|verify|all]   (default: all)
# Knobs: DEPTH (Apalache bound in steps for the checks expected to be violated, default 15; they stop
# at the first counterexample); SAFE_DEPTH (bound for the checks expected to hold, default 11);
# FULL_DEPTH (bound for SafetyFull = Safety + ExpiredImpliesNeverRelayable, default 10); JOBS (parallel Apalache servers); BASE_PORT
# (ports BASE_PORT.. are used, one per check; keep them clear of other jobs on a shared host); ONLY (a regex: run only
# checks whose "<main> <invariant>" matches); SKIP (a regex: leave those checks out); SAMPLES/STEPS (simulation).
# Requires quint (npm i -g @informalsystems/quint); `verify` needs Java 17+ (quint fetches Apalache).
set -uo pipefail
cd "$(dirname "$0")" || exit 1

MODE="${1:-all}"
case "$MODE" in test|simulate|verify|all) ;; *) echo "unknown mode: $MODE" >&2; exit 2 ;; esac
SAMPLES="${SAMPLES:-20000}"
STEPS="${STEPS:-25}"
DEPTH="${DEPTH:-15}"
SAFE_DEPTH="${SAFE_DEPTH:-11}"
FULL_DEPTH="${FULL_DEPTH:-10}"
JOBS="${JOBS:-8}"
BASE_PORT="${BASE_PORT:-9300}"
ONLY="${ONLY:-.}"
SKIP="${SKIP:-^$}"
LOGDIR="${LOGDIR:-logs}"
QUINT="${QUINT:-quint}"
mkdir -p "$LOGDIR"

WITNESSES=(NoRefundEver NoRefundOfM2 NoRefundOfM3 NoEdgeRelay NoLateRelayBeforeExporter NoDeferredL1Fact
  NoRefundExporterFirst NoRefundAfterLagoon NoRefundAfterLagoonReverted NoRefundAfterL1Rollback
  NoRefundAfterBridgeRollback NoRefundAfterExporterRemoved NoRefundAfterMessengerRollback
  NoJoin NoLateInteropEnable NoRogueExporterEver NoRogueWithdrawal NoMessengerWithdrawal NoResend
  NoRefundUnderPeriodOverride)
UNSAFE=(noWindowRuleOnB windowAbovePOnC govMessengerDowngrade lagoonWithoutGuard rogueExporterMember
  rogueThenJoin duplicateChainId l1EarlierDesign periodOverrideUnguarded)

# The unsafe instances are in rollout-unsafe-*.qnt (they import the model from rollout.qnt).
file_of() { grep -l "^module $1 {" rollout.qnt rollout-unsafe-*.qnt | head -1; }

classify() {
  case "$1" in
    0) echo holds ;;
    1) if grep -q -E "\[violation\]|found a counterexample|Invariant violated" "$2"; then echo violated; else echo "error(1)"; fi ;;
    *) echo "error($1)" ;;
  esac
}

ALL=("holds rolloutSafe Safety")
for w in "${WITNESSES[@]}"; do ALL+=("violated rolloutSafe $w"); done
ALL+=("holds rolloutWindowAtP Safety")
for w in NoRefundEver NoEdgeRelay NoRefundOfM3 NoRefundAfterLagoon; do ALL+=("violated rolloutWindowAtP $w"); done
ALL+=("holds downgradeHardened Safety")
for w in NoRefundEver NoRefundAfterMessengerRollback NoResend; do ALL+=("violated downgradeHardened $w"); done
for m in "${UNSAFE[@]}"; do ALL+=("violated $m NoDoubleSpend"); done
ALL+=("holdsfull rolloutSafe SafetyFull")
CHECKS=()
for c in "${ALL[@]}"; do
  read -r _ m inv <<<"$c"
  if [[ "$m $inv" =~ $ONLY && ! "$m $inv" =~ $SKIP ]]; then CHECKS+=("$c"); fi
done

FAILURES=0

if [[ "$MODE" == test || "$MODE" == all ]]; then
  for m in rolloutSafe rolloutWindowAtP downgradeHardened "${UNSAFE[@]}"; do
    out="$LOGDIR/test-$m.log"
    $QUINT test "$(file_of "$m")" --main="$m" --match='^(witness|blocked|cex)' > "$out" 2>&1
    code=$?
    n="$(grep -c 'passed 1 test' "$out")"
    if [[ $code == 0 && $n -gt 0 ]]; then echo "ok    test $m: $n scripted traces pass"; else echo "FAIL  test $m (see $out)"; FAILURES=$((FAILURES+1)); fi
  done
fi

if [[ "$MODE" == simulate || "$MODE" == all ]]; then
  for m in rolloutSafe rolloutWindowAtP downgradeHardened; do
    out="$LOGDIR/sim-$m-Safety.log"
    $QUINT run "$(file_of "$m")" --main="$m" --invariant=Safety --max-samples="$SAMPLES" --max-steps="$STEPS" > "$out" 2>&1
    got="$(classify $? "$out")"
    if [[ "$got" == holds ]]; then echo "ok    sim holds: $m Safety"; else echo "FAIL  sim expected holds, got $got: $m Safety (see $out)"; FAILURES=$((FAILURES+1)); fi
  done
fi

if [[ "$MODE" == verify || "$MODE" == all ]]; then
  i=0
  for c in "${CHECKS[@]}"; do
    read -r want m inv <<<"$c"
    port=$((BASE_PORT + i))
    out="$LOGDIR/verify-$m-$inv.log"
    d="$DEPTH"; [[ "$want" == holds ]] && d="$SAFE_DEPTH"; [[ "$want" == holdsfull ]] && d="$FULL_DEPTH"
    ( start=$(date +%s)
      echo "DEPTH=$d" > "$out"
      $QUINT verify "$(file_of "$m")" --main="$m" --invariant="$inv" --max-steps="$d" \
        --server-endpoint="localhost:$port" >> "$out" 2>&1
      code=$?
      echo "EXIT=$code SECONDS=$(( $(date +%s) - start ))" >> "$out" ) &
    i=$((i + 1))
    while (( $(jobs -rp | wc -l) >= JOBS )); do wait -n; done
  done
  wait
  for c in "${CHECKS[@]}"; do
    read -r want m inv <<<"$c"
    out="$LOGDIR/verify-$m-$inv.log"
    code="$(sed -n 's/^EXIT=\([0-9]*\).*/\1/p' "$out" | tail -n1)"
    secs="$(sed -n 's/.*SECONDS=\([0-9]*\).*/\1/p' "$out" | tail -n1)"
    d="$(sed -n 's/^DEPTH=\([0-9]*\)$/\1/p' "$out" | head -n1)"
    got="$(classify "${code:-99}" "$out")"
    [[ "$want" == holdsfull ]] && want=holds
    if [[ "$got" == "$want" ]]; then echo "ok    verify depth=$d ${secs}s $want: $m $inv"; else echo "FAIL  verify expected $want, got $got: $m $inv (see $out)"; FAILURES=$((FAILURES+1)); fi
  done
fi

echo "failures: $FAILURES"
exit $((FAILURES > 0))
