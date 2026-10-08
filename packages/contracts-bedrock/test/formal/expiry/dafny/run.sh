#!/usr/bin/env bash
# Verify the expiry bridge lemmas against the op-supernode Dafny model.
#   DAFNY=/path/to/dafny ./run.sh            # bridge + expected failures
#   DAFNY=/path/to/dafny ./run.sh --model    # also re-verify the whole existing model,
#                                            # all six files (~10 min on a loaded 32-core box)
# Exits non-zero if ExpiryBridge.dfy has any error, if ExpectFail.dfy does not fail in exactly the
# expected way, if a lemma/method of ExpiryBridge.dfy has no Nonvacuous_ witness in NonVacuity.dfy
# that calls it, if NonVacuity.dfy has any error, or if `assert false` is provable at the end of
# any witness (see below).
set -euo pipefail
cd "$(dirname "$0")"
DAFNY="${DAFNY:-dafny}"
CORES="${CORES:-8}"
# --allow-warnings: the included model has three `assume` statements without {:axiom}
# (Interop.dfy:1019, 1020, 1631); Dafny warns on them and would otherwise exit 2.
FLAGS=(--cores "$CORES" --allow-warnings)

echo "dafny: $("$DAFNY" --version)"

if [[ "${1:-}" == "--model" ]]; then
  echo "== existing model: op-supernode/dafny-models (Interop.dfy + every included file)"
  # --verify-included-files: Interop.dfy includes VerifiedDB, ChainContainer, LogsDB, Types, Utils;
  # without it Dafny verifies only Interop.dfy's own declarations.
  # A generous per-VC time limit: under heavy host load the default limit made
  # ApplyPendingTransition's VC time out (internal prover error).
  "$DAFNY" verify "${FLAGS[@]}" --verify-included-files --verification-time-limit 1200 \
    ../../../../../../op-supernode/dafny-models/Interop.dfy
fi

echo "== ExpiryBridge.dfy (must verify with 0 errors)"
"$DAFNY" verify "${FLAGS[@]}" ExpiryBridge.dfy

echo "== ExpectFail.dfy (must fail: exit 4, exactly the two EXPECT-FAIL postconditions)"
set +e
out="$("$DAFNY" verify "${FLAGS[@]}" ExpectFail.dfy 2>&1)"
status=$?
set -e
echo "$out" | grep -E "^ExpectFail.dfy.*(Error|Related location)|verifier finished" || true

fail() { echo "FAIL: ExpectFail.dfy: $*" >&2; exit 1; }
# Dafny exit code 4 = verification errors (not a parse/resolution error, not success).
[[ $status -eq 4 ]] || fail "expected exit status 4, got $status"
echo "$out" | grep -q "finished with 2 verified, 2 errors" || fail "expected '2 verified, 2 errors'"
# Every error must be a postcondition failure.
n_err=$(echo "$out" | grep -cE "^ExpectFail.dfy\([0-9]+,[0-9]+\): Error:" || true)
n_post=$(echo "$out" | grep -cE "^ExpectFail.dfy\([0-9]+,[0-9]+\): Error: a postcondition could not be proved" || true)
[[ $n_err -eq 2 && $n_post -eq 2 ]] || fail "expected 2 postcondition errors, got $n_post of $n_err"
# The failing postconditions must be exactly the lines marked EXPECT-FAIL (one per lemma).
expected=$(grep -n "// EXPECT-FAIL" ExpectFail.dfy | grep -v "^[0-9]*://" | cut -d: -f1 | sort -n | tr '\n' ' ')
actual=$(echo "$out" | sed -nE 's/^ExpectFail.dfy\(([0-9]+),[0-9]+\): Related location: this is the postcondition that could not be proved.*/\1/p' | sort -n | tr '\n' ' ')
[[ -n "$expected" && "$expected" == "$actual" ]] || fail "failing postcondition lines '$actual' != EXPECT-FAIL lines '$expected'"
echo "OK: NoPeriodAssumption and ExclusiveBoundary are rejected at their EXPECT-FAIL postconditions (lines $expected)"

echo "== NonVacuity.dfy: one witness per lemma/method of ExpiryBridge.dfy"
missing=0
while IFS= read -r name; do
  if ! grep -q "method Nonvacuous_${name}(" NonVacuity.dfy; then
    echo "missing witness: Nonvacuous_${name}" >&2; missing=1
  elif ! grep -q "B\.${name}(" NonVacuity.dfy; then
    echo "witness Nonvacuous_${name} does not call B.${name}" >&2; missing=1
  fi
done < <(sed -nE 's/^  (lemma|method) ([A-Za-z0-9_]+)\(.*/\2/p' ExpiryBridge.dfy)
[[ $missing -eq 0 ]] || { echo "FAIL: NonVacuity.dfy does not cover ExpiryBridge.dfy" >&2; exit 1; }
n_wit=$(grep -cE "^  method Nonvacuous_" NonVacuity.dfy)
echo "every lemma/method has a witness ($n_wit witnesses)"

echo "== NonVacuity.dfy (must verify with 0 errors)"
"$DAFNY" verify "${FLAGS[@]}" NonVacuity.dfy

echo "== vacuity probe: 'assert false' at the end of every witness must FAIL"
# A witness whose context (its residual requires + the constructor's postconditions) were
# contradictory would prove anything. Insert `assert false;` before each witness's closing brace
# and require exactly one failure per witness, at the probe lines. (A smoke test: Dafny failing
# to prove false is evidence, not a proof, that the context is consistent.)
probe=NonVacuity.probe.dfy
trap 'rm -f "$probe"' EXIT
awk '/^  method Nonvacuous_/ { inw = 1 } inw && /^  }$/ { print "    assert false; // VACUITY-PROBE"; inw = 0 } { print }' \
  NonVacuity.dfy > "$probe"
set +e
pout="$("$DAFNY" verify "${FLAGS[@]}" "$probe" 2>&1)"
pstatus=$?
set -e
[[ $pstatus -eq 4 ]] || { echo "FAIL: vacuity probe: expected exit 4, got $pstatus" >&2; exit 1; }
p_expected=$(grep -n "VACUITY-PROBE" "$probe" | cut -d: -f1 | sort -n | tr '\n' ' ')
pre='NonVacuity\.probe\.dfy'
p_actual=$(echo "$pout" | sed -nE "s/^${pre}\(([0-9]+),[0-9]+\): Error: assertion might not hold.*/\1/p" | sort -n | tr '\n' ' ')
p_errs=$(echo "$pout" | grep -cE "^${pre}\([0-9]+,[0-9]+\): Error" || true)
[[ "$p_expected" == "$p_actual" && $p_errs -eq $n_wit ]] || {
  echo "$pout" | grep -E "Error|finished" >&2
  echo "FAIL: vacuity probe: failing lines '$p_actual' != probe lines '$p_expected'" >&2; exit 1; }
echo "OK: assert false is unprovable in all $n_wit witness contexts"
